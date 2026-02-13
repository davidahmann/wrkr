package doctor

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/fsx"
	"github.com/davidahmann/wrkr/core/out"
	"github.com/davidahmann/wrkr/core/schema/validate"
	"github.com/davidahmann/wrkr/core/serve"
	"github.com/davidahmann/wrkr/core/store"
)

type CheckResult struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Severity    string `json:"severity"`
	Details     string `json:"details,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type Result struct {
	CheckedAt time.Time     `json:"checked_at"`
	Profile   string        `json:"profile"`
	OK        bool          `json:"ok"`
	Checks    []CheckResult `json:"checks"`
}

type Options struct {
	Now                      func() time.Time
	ProductionReadiness      bool
	ServeListenAddr          string
	ServeAuthToken           string
	ServeMaxBodyBytes        int64
	ServeAllowNonLoopback    bool
	ServeListenAddrSet       bool
	ServeAuthTokenSet        bool
	ServeMaxBodyBytesSet     bool
	ServeAllowNonLoopbackSet bool
}

func Run(now func() time.Time) (Result, error) {
	return RunWithOptions(Options{Now: now})
}

func RunWithOptions(opts Options) (Result, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	opts.Now = now

	profile := "default"
	if opts.ProductionReadiness {
		profile = "production-readiness"
	}
	results := make([]CheckResult, 0, 10)

	s, err := store.New("")
	if err != nil {
		results = append(results, failCritical("store_root", err.Error(), "Ensure HOME is writable and ~/.wrkr can be created."))
	} else {
		results = append(results, pass("store_root", s.Root()))
	}

	layout, err := out.NewLayout("")
	if err != nil {
		results = append(results, failCritical("output_layout", err.Error(), "Ensure output root is within the current workspace."))
	} else if err := layout.Ensure(); err != nil {
		results = append(results, failCritical("output_layout", err.Error(), "Ensure current workspace is writable or pass --out-dir for command-specific output."))
	} else {
		results = append(results, pass("output_layout", layout.Root()))
	}

	missing := 0
	for _, rel := range validate.SchemaList() {
		if _, err := validate.SchemaPath(rel); err != nil {
			missing++
		}
	}
	if missing == 0 {
		results = append(results, pass("schemas", "missing=0"))
	} else {
		results = append(results, failCritical("schemas", "missing="+itoa(missing), "Restore missing files under ./schemas/v1 and rerun."))
	}

	hookPath := filepath.Clean(".githooks/pre-push")
	if info, err := os.Stat(hookPath); err == nil && !info.IsDir() {
		results = append(results, pass("git_hook_pre_push", hookPath))
	} else {
		results = append(results, failWarning("git_hook_pre_push", hookPath, "Run `make hooks` to install repository hooks."))
	}

	if opts.ProductionReadiness {
		results = append(results, productionReadinessChecks(opts)...)
	}

	ok := true
	for _, check := range results {
		if !check.OK && check.Severity == "critical" {
			ok = false
			break
		}
	}
	return Result{
		CheckedAt: now().UTC(),
		Profile:   profile,
		OK:        ok,
		Checks:    results,
	}, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		v = -v
	}
	digits := []byte{}
	for v > 0 {
		digits = append([]byte{byte('0' + (v % 10))}, digits...)
		v /= 10
	}
	return string(digits)
}

func pass(name, details string) CheckResult {
	return CheckResult{Name: name, OK: true, Severity: "critical", Details: details}
}

func failCritical(name, details, remediation string) CheckResult {
	return CheckResult{
		Name:        name,
		OK:          false,
		Severity:    "critical",
		Details:     details,
		Remediation: remediation,
	}
}

func failWarning(name, details, remediation string) CheckResult {
	return CheckResult{
		Name:        name,
		OK:          false,
		Severity:    "warning",
		Details:     details,
		Remediation: remediation,
	}
}

func productionReadinessChecks(opts Options) []CheckResult {
	results := make([]CheckResult, 0, 6)

	profile := strings.ToLower(strings.TrimSpace(os.Getenv("WRKR_PROFILE")))
	if profile == "strict" {
		results = append(results, pass("strict_config_profile", "WRKR_PROFILE=strict"))
	} else {
		results = append(results, failCritical(
			"strict_config_profile",
			fmt.Sprintf("WRKR_PROFILE=%q", profile),
			"Set WRKR_PROFILE=strict for production workloads.",
		))
	}

	signingMode := strings.ToLower(strings.TrimSpace(os.Getenv("WRKR_SIGNING_MODE")))
	if signingMode == "ed25519" {
		results = append(results, pass("signing_mode", "WRKR_SIGNING_MODE=ed25519"))
	} else {
		results = append(results, failCritical(
			"signing_mode",
			fmt.Sprintf("WRKR_SIGNING_MODE=%q", signingMode),
			"Set WRKR_SIGNING_MODE=ed25519.",
		))
	}

	keySource := strings.ToLower(strings.TrimSpace(os.Getenv("WRKR_SIGNING_KEY_SOURCE")))
	switch keySource {
	case "env", "kms":
		results = append(results, pass("signing_key_source", "WRKR_SIGNING_KEY_SOURCE="+keySource))
	case "file":
		keyPath := strings.TrimSpace(os.Getenv("WRKR_SIGNING_KEY_PATH"))
		info, err := os.Stat(keyPath)
		if keyPath == "" || err != nil || info.IsDir() {
			results = append(results, failCritical(
				"signing_key_source",
				fmt.Sprintf("WRKR_SIGNING_KEY_SOURCE=file WRKR_SIGNING_KEY_PATH=%q", keyPath),
				"Set WRKR_SIGNING_KEY_PATH to an existing private key file.",
			))
		} else {
			results = append(results, pass("signing_key_source", "WRKR_SIGNING_KEY_SOURCE=file"))
		}
	default:
		results = append(results, failCritical(
			"signing_key_source",
			fmt.Sprintf("WRKR_SIGNING_KEY_SOURCE=%q", keySource),
			"Set WRKR_SIGNING_KEY_SOURCE to one of: file, env, kms.",
		))
	}

	if err := checkStoreLockHealth(opts.Now); err != nil {
		results = append(results, failCritical("store_lock_health", err.Error(), "Ensure ~/.wrkr/jobs is writable and no stale append lock blocks claims."))
	} else {
		results = append(results, pass("store_lock_health", "lock acquire/release successful"))
	}

	retentionDays, retentionErr := parsePositiveIntFromEnv("WRKR_RETENTION_DAYS")
	outputRetentionDays, outputRetentionErr := parsePositiveIntFromEnv("WRKR_OUTPUT_RETENTION_DAYS")
	if retentionErr != nil || outputRetentionErr != nil {
		details := "WRKR_RETENTION_DAYS and WRKR_OUTPUT_RETENTION_DAYS must be positive integers"
		if retentionErr != nil {
			details = retentionErr.Error()
		} else if outputRetentionErr != nil {
			details = outputRetentionErr.Error()
		}
		results = append(results, failCritical(
			"retention_settings",
			details,
			"Set retention env vars and automate `wrkr store prune --dry-run` in CI/nightly before enabling deletion.",
		))
	} else {
		results = append(results, pass(
			"retention_settings",
			fmt.Sprintf("WRKR_RETENTION_DAYS=%d WRKR_OUTPUT_RETENTION_DAYS=%d", retentionDays, outputRetentionDays),
		))
	}

	if unsafe := strings.TrimSpace(os.Getenv("WRKR_ALLOW_UNSAFE")); unsafe == "1" || strings.EqualFold(unsafe, "true") {
		results = append(results, failCritical(
			"unsafe_defaults",
			"WRKR_ALLOW_UNSAFE is enabled",
			"Unset WRKR_ALLOW_UNSAFE for production environments.",
		))
	} else if details, err := validateServeHardening(opts); err != nil {
		results = append(results, failCritical("unsafe_defaults", err.Error(), "Provide actual serve inputs via doctor flags or WRKR_SERVE_* env and require non-loopback hardening."))
	} else {
		if details == "" {
			details = "production-safe defaults confirmed"
		}
		results = append(results, pass("unsafe_defaults", details))
	}

	return results
}

func parsePositiveIntFromEnv(key string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is unset", key)
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s=%q must be a positive integer", key, raw)
	}
	return n, nil
}

func checkStoreLockHealth(now func() time.Time) error {
	s, err := store.New("")
	if err != nil {
		return err
	}

	probeJob := "_doctor_probe"
	if err := s.EnsureJob(probeJob); err != nil {
		return err
	}
	lockPath := filepath.Join(s.Root(), "jobs", probeJob, "append.lock")
	lock, err := fsx.AcquireLockWithOptions(
		lockPath,
		fmt.Sprintf("pid=%d;ts=%d", os.Getpid(), now().UTC().UnixNano()),
		fsx.LockOptions{StaleAfter: 2 * time.Minute, Now: now},
	)
	if err != nil {
		return err
	}
	return lock.Release()
}

func validateServeHardening(opts Options) (string, error) {
	cfg, source, provided, err := resolveServeConfig(opts)
	if err != nil {
		return "", err
	}
	if !provided {
		return "no explicit serve inputs provided; default loopback assumptions", nil
	}

	if _, err := serve.ValidateConfig(cfg); err != nil {
		return "", fmt.Errorf("serve config validation failed (%s): %w", source, err)
	}

	if isLoopbackListen(cfg.ListenAddr) {
		return fmt.Sprintf("serve listen=%s (%s)", cfg.ListenAddr, source), nil
	}
	return fmt.Sprintf(
		"serve non-loopback hardened (%s): listen=%s allow_non_loopback=%t max_body_bytes=%d",
		source,
		cfg.ListenAddr,
		cfg.AllowNonLoopback,
		cfg.MaxBodyBytes,
	), nil
}

func resolveServeConfig(opts Options) (serve.Config, string, bool, error) {
	hasFlagInput := opts.ServeListenAddrSet || opts.ServeAuthTokenSet || opts.ServeMaxBodyBytesSet || opts.ServeAllowNonLoopbackSet
	listen := strings.TrimSpace(os.Getenv("WRKR_SERVE_LISTEN"))
	auth := strings.TrimSpace(os.Getenv("WRKR_SERVE_AUTH_TOKEN"))
	maxBody := int64(0)
	if raw := strings.TrimSpace(os.Getenv("WRKR_SERVE_MAX_BODY_BYTES")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return serve.Config{}, "", false, fmt.Errorf("WRKR_SERVE_MAX_BODY_BYTES=%q must be a positive integer", raw)
		}
		maxBody = parsed
	}
	allow := false
	if raw := strings.TrimSpace(os.Getenv("WRKR_SERVE_ALLOW_NON_LOOPBACK")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return serve.Config{}, "", false, fmt.Errorf("WRKR_SERVE_ALLOW_NON_LOOPBACK=%q must be a boolean", raw)
		}
		allow = parsed
	}
	source := "env"

	if opts.ServeListenAddrSet {
		listen = strings.TrimSpace(opts.ServeListenAddr)
		source = "flags"
	}
	if opts.ServeAuthTokenSet {
		auth = strings.TrimSpace(opts.ServeAuthToken)
		source = "flags"
	}
	if opts.ServeMaxBodyBytesSet {
		maxBody = opts.ServeMaxBodyBytes
		source = "flags"
	}
	if opts.ServeAllowNonLoopbackSet {
		allow = opts.ServeAllowNonLoopback
		source = "flags"
	}

	hasEnvInput := strings.TrimSpace(os.Getenv("WRKR_SERVE_LISTEN")) != "" ||
		strings.TrimSpace(os.Getenv("WRKR_SERVE_AUTH_TOKEN")) != "" ||
		strings.TrimSpace(os.Getenv("WRKR_SERVE_MAX_BODY_BYTES")) != "" ||
		strings.TrimSpace(os.Getenv("WRKR_SERVE_ALLOW_NON_LOOPBACK")) != ""
	if !hasFlagInput && !hasEnvInput {
		return serve.Config{}, "default", false, nil
	}
	if !hasFlagInput {
		source = "env"
	}

	return serve.Config{
		ListenAddr:       listen,
		AllowNonLoopback: allow,
		AuthToken:        auth,
		MaxBodyBytes:     maxBody,
	}, source, true, nil
}

func isLoopbackListen(addr string) bool {
	host := strings.TrimSpace(addr)
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback() && !ip.IsUnspecified()
}
