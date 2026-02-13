package envfp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/davidahmann/wrkr/core/jcs"
)

type Fingerprint struct {
	Rules      []string          `json:"rules"`
	Values     map[string]string `json:"values"`
	Hash       string            `json:"hash"`
	CapturedAt time.Time         `json:"captured_at"`
}

func DefaultRules() []string {
	return []string{"os", "arch", "go_version"}
}

func Capture(rules []string, now time.Time) (Fingerprint, error) {
	if len(rules) == 0 {
		rules = DefaultRules()
	}

	norm := make([]string, 0, len(rules))
	seen := map[string]struct{}{}
	for _, rule := range rules {
		r := strings.TrimSpace(rule)
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		norm = append(norm, r)
	}
	if len(norm) == 0 {
		norm = DefaultRules()
	}
	slices.Sort(norm)

	values := make(map[string]string, len(norm))
	for _, rule := range norm {
		switch {
		case rule == "os":
			values[rule] = runtime.GOOS
		case rule == "arch":
			values[rule] = runtime.GOARCH
		case rule == "go_version":
			values[rule] = runtime.Version()
		case rule == "cwd":
			cwd, err := os.Getwd()
			if err != nil {
				return Fingerprint{}, fmt.Errorf("resolve cwd: %w", err)
			}
			values[rule] = cwd
		case rule == "hostname":
			host, err := os.Hostname()
			if err != nil {
				return Fingerprint{}, fmt.Errorf("resolve hostname: %w", err)
			}
			values[rule] = host
		case strings.HasPrefix(rule, "env:"):
			key := strings.TrimPrefix(rule, "env:")
			values[rule] = os.Getenv(key)
		default:
			return Fingerprint{}, fmt.Errorf("unsupported environment fingerprint rule %q", rule)
		}
	}

	payload, err := json.Marshal(map[string]any{
		"rules":  norm,
		"values": values,
	})
	if err != nil {
		return Fingerprint{}, fmt.Errorf("marshal fingerprint payload: %w", err)
	}
	canon, err := jcs.Canonicalize(payload)
	if err != nil {
		return Fingerprint{}, err
	}
	sum := sha256.Sum256(canon)

	return Fingerprint{
		Rules:      norm,
		Values:     values,
		Hash:       hex.EncodeToString(sum[:]),
		CapturedAt: now.UTC(),
	}, nil
}
