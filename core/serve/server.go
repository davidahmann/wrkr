package serve

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/accept"
	"github.com/davidahmann/wrkr/core/approve"
	"github.com/davidahmann/wrkr/core/dispatch"
	wrkrerrors "github.com/davidahmann/wrkr/core/errors"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/pack"
	ghreport "github.com/davidahmann/wrkr/core/report"
	"github.com/davidahmann/wrkr/core/runner"
	statusview "github.com/davidahmann/wrkr/core/status"
	"github.com/davidahmann/wrkr/core/store"
)

const (
	DefaultListenAddr  = "127.0.0.1:9488"
	defaultMaxBodySize = int64(1 << 20)
)

type Config struct {
	ListenAddr       string
	AllowNonLoopback bool
	AuthToken        string
	MaxBodyBytes     int64
	Now              func() time.Time
	ProducerVersion  string
}

type Server struct {
	cfg Config
}

func NormalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.ListenAddr) == "" {
		cfg.ListenAddr = DefaultListenAddr
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodySize
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if strings.TrimSpace(cfg.ProducerVersion) == "" {
		cfg.ProducerVersion = "dev"
	}
	return cfg
}

func ValidateConfig(raw Config) (Config, error) {
	cfg := NormalizeConfig(raw)
	host := hostPart(cfg.ListenAddr)
	loopback := isLoopbackHost(host)

	if !loopback {
		if !cfg.AllowNonLoopback {
			return Config{}, wrkrerrors.New(
				wrkrerrors.EUnsafeOperation,
				"non-loopback listen requires --allow-non-loopback",
				map[string]any{"listen": cfg.ListenAddr},
			)
		}
		if strings.TrimSpace(cfg.AuthToken) == "" {
			return Config{}, wrkrerrors.New(
				wrkrerrors.EUnsafeOperation,
				"non-loopback listen requires --auth-token",
				map[string]any{"listen": cfg.ListenAddr},
			)
		}
		if raw.MaxBodyBytes <= 0 {
			return Config{}, wrkrerrors.New(
				wrkrerrors.EUnsafeOperation,
				"non-loopback listen requires explicit --max-body-bytes",
				map[string]any{"listen": cfg.ListenAddr},
			)
		}
	}
	return cfg, nil
}

func New(cfg Config) *Server {
	return &Server{cfg: NormalizeConfig(cfg)}
}

func (s *Server) Handler() http.Handler {
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AuthToken != "" {
		token := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
		if token != s.cfg.AuthToken {
			s.writeError(w, r, wrkrerrors.New(wrkrerrors.EUnsafeOperation, "invalid auth token", nil), http.StatusUnauthorized)
			return
		}
	}

	if s.cfg.MaxBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v1/jobs:submit":
		s.handleSubmit(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":status"):
		s.handleStatus(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.Contains(r.URL.Path, "/checkpoints"):
		s.handleCheckpoints(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":approve"):
		s.handleApprove(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":export"):
		s.handleExport(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":verify"):
		s.handleVerify(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":accept"):
		s.handleAccept(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/jobs/") && strings.HasSuffix(r.URL.Path, ":report-github"):
		s.handleReportGitHub(w, r)
	default:
		s.writeError(w, r, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "unknown endpoint", map[string]any{"path": r.URL.Path}), http.StatusNotFound)
	}
}

func (s *Server) ListenAndServe() error {
	cfg, err := ValidateConfig(s.cfg)
	if err != nil {
		return err
	}
	server := http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobSpecPath string `json:"jobspec_path"`
		JobID       string `json:"job_id"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	result, err := dispatch.Submit(req.JobSpecPath, dispatch.SubmitOptions{
		Now:       s.cfg.Now,
		JobID:     req.JobID,
		FromServe: true,
	})
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":status")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}

	rn, st, err := openRunner(s.cfg.Now)
	if err != nil {
		s.writeError(w, r, err, http.StatusInternalServerError)
		return
	}
	if err := ensureJobExists(st, jobID); err != nil {
		s.writeError(w, r, err, http.StatusNotFound)
		return
	}
	state, err := rn.Recover(jobID)
	if err != nil {
		s.writeError(w, r, err, http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, http.StatusOK, statusview.FromRunnerState(state, s.cfg.ProducerVersion, s.cfg.Now()))
}

func (s *Server) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/jobs/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.writeError(w, r, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid checkpoints path", nil), http.StatusBadRequest)
		return
	}
	jobID := parts[0]
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}

	rn, st, err := openRunner(s.cfg.Now)
	if err != nil {
		s.writeError(w, r, err, http.StatusInternalServerError)
		return
	}
	if err := ensureJobExists(st, jobID); err != nil {
		s.writeError(w, r, err, http.StatusNotFound)
		return
	}

	if len(parts) == 2 && parts[1] == "checkpoints" {
		items, err := rn.ListCheckpoints(jobID)
		if err != nil {
			s.writeError(w, r, err, http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, http.StatusOK, items)
		return
	}
	if len(parts) == 3 && parts[1] == "checkpoints" {
		if err := rejectTraversal(parts[2]); err != nil {
			s.writeError(w, r, err, http.StatusBadRequest)
			return
		}
		item, err := rn.GetCheckpoint(jobID, parts[2])
		if err != nil {
			s.writeError(w, r, err, http.StatusNotFound)
			return
		}
		s.writeJSON(w, http.StatusOK, item)
		return
	}
	s.writeError(w, r, wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid checkpoints path", nil), http.StatusBadRequest)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":approve")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	var req struct {
		CheckpointID string `json:"checkpoint_id"`
		Reason       string `json:"reason"`
		ApprovedBy   string `json:"approved_by"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := approve.ValidateReason(req.Reason); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	rn, st, err := openRunner(s.cfg.Now)
	if err != nil {
		s.writeError(w, r, err, http.StatusInternalServerError)
		return
	}
	if err := ensureJobExists(st, jobID); err != nil {
		s.writeError(w, r, err, http.StatusNotFound)
		return
	}
	rec, err := rn.ApproveCheckpoint(jobID, req.CheckpointID, req.Reason, approve.ResolveApprovedBy(req.ApprovedBy))
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":export")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	var req struct {
		OutDir string `json:"out_dir"`
	}
	if err := decodeJSONOptional(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := rejectTraversal(req.OutDir); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	result, err := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          req.OutDir,
		Now:             s.cfg.Now,
		ProducerVersion: s.cfg.ProducerVersion,
	})
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":verify")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	var req struct {
		OutDir string `json:"out_dir"`
	}
	if err := decodeJSONOptional(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := rejectTraversal(req.OutDir); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	layout := out.NewLayout(req.OutDir)
	result, err := pack.VerifyJobpack(layout.JobpackPath(jobID))
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAccept(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":accept")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	var req struct {
		ConfigPath string `json:"config_path"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := rejectTraversal(req.ConfigPath); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	result, err := accept.Run(jobID, accept.RunOptions{
		Now:             s.cfg.Now,
		ProducerVersion: s.cfg.ProducerVersion,
		ConfigPath:      req.ConfigPath,
		WorkDir:         ".",
	})
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleReportGitHub(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), ":report-github")
	if err := rejectTraversal(jobID); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	var req struct {
		OutDir string `json:"out_dir"`
	}
	if err := decodeJSONOptional(r.Body, &req); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := rejectTraversal(req.OutDir); err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	exported, err := pack.ExportJobpack(jobID, pack.ExportOptions{
		OutDir:          req.OutDir,
		Now:             s.cfg.Now,
		ProducerVersion: s.cfg.ProducerVersion,
	})
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	summary, err := ghreport.BuildGitHubSummaryFromJobpack(exported.Path, ghreport.SummaryOptions{
		Now:             s.cfg.Now,
		ProducerVersion: s.cfg.ProducerVersion,
	})
	if err != nil {
		s.writeError(w, r, err, http.StatusBadRequest)
		return
	}
	written, err := ghreport.WriteGitHubSummary(summary, req.OutDir)
	if err != nil {
		s.writeError(w, r, err, http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"summary":               summary,
		"summary_json_path":     written.JSONPath,
		"summary_markdown_path": written.MarkdownPath,
		"step_summary_path":     written.StepSummaryPath,
	})
}

func openRunner(now func() time.Time) (*runner.Runner, *store.LocalStore, error) {
	s, err := store.New("")
	if err != nil {
		return nil, nil, err
	}
	r, err := runner.New(s, runner.Options{Now: now})
	if err != nil {
		return nil, nil, err
	}
	return r, s, nil
}

func ensureJobExists(s *store.LocalStore, jobID string) error {
	exists, err := s.JobExists(jobID)
	if err != nil {
		return err
	}
	if !exists {
		return wrkrerrors.New(
			wrkrerrors.EInvalidInputSchema,
			"job not found",
			map[string]any{"job_id": jobID},
		)
	}
	return nil
}

func decodeJSON(body io.ReadCloser, out any) error {
	defer func() { _ = body.Close() }()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return wrkrerrors.New(wrkrerrors.EUnsafeOperation, "request body too large", map[string]any{"limit": tooLarge.Limit})
		}
		return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid request json", map[string]any{"error": err.Error()})
	}
	return nil
}

func decodeJSONOptional(body io.ReadCloser, out any) error {
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return wrkrerrors.New(wrkrerrors.EUnsafeOperation, "request body too large", map[string]any{"limit": tooLarge.Limit})
		}
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return wrkrerrors.New(wrkrerrors.EInvalidInputSchema, "invalid request json", map[string]any{"error": err.Error()})
	}
	return nil
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error, status int) {
	env := wrkrerrors.ToEnvelope(err, s.cfg.ProducerVersion, s.cfg.Now())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func hostPart(listenAddr string) string {
	host, _, err := net.SplitHostPort(listenAddr)
	if err == nil {
		return host
	}
	return listenAddr
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "localhost" {
		return true
	}
	if host == "" {
		// ":<port>" and similar wildcard forms bind on all interfaces.
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback() && !ip.IsUnspecified()
}

func rejectTraversal(value string) error {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "..") {
		return wrkrerrors.New(wrkrerrors.EUnsafeOperation, "path traversal not allowed", map[string]any{"value": value})
	}
	return nil
}
