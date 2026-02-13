package doctor

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	result, err := Run(func() time.Time {
		return time.Date(2026, 2, 14, 2, 30, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Profile != "default" {
		t.Fatalf("expected default profile, got %s", result.Profile)
	}
	if len(result.Checks) == 0 {
		t.Fatal("expected checks")
	}
}

func TestRunWithProductionReadiness(t *testing.T) {
	t.Setenv("WRKR_PROFILE", "strict")
	t.Setenv("WRKR_SIGNING_MODE", "ed25519")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "env")
	t.Setenv("WRKR_RETENTION_DAYS", "7")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "7")

	result, err := RunWithOptions(Options{
		Now:                 func() time.Time { return time.Date(2026, 2, 14, 2, 30, 0, 0, time.UTC) },
		ProductionReadiness: true,
	})
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if result.Profile != "production-readiness" {
		t.Fatalf("expected production-readiness profile, got %s", result.Profile)
	}
	if len(result.Checks) == 0 {
		t.Fatal("expected checks")
	}
}

func TestRunWithProductionReadinessRejectsWildcardListenFromEnv(t *testing.T) {
	t.Setenv("WRKR_PROFILE", "strict")
	t.Setenv("WRKR_SIGNING_MODE", "ed25519")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "env")
	t.Setenv("WRKR_RETENTION_DAYS", "7")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "7")
	t.Setenv("WRKR_SERVE_LISTEN", ":9488")
	t.Setenv("WRKR_SERVE_ALLOW_NON_LOOPBACK", "false")
	t.Setenv("WRKR_SERVE_AUTH_TOKEN", "")
	t.Setenv("WRKR_SERVE_MAX_BODY_BYTES", "")

	result, err := RunWithOptions(Options{
		Now:                 func() time.Time { return time.Date(2026, 2, 14, 2, 30, 0, 0, time.UTC) },
		ProductionReadiness: true,
	})
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if result.OK {
		t.Fatalf("expected readiness failure for wildcard listen: %+v", result)
	}
}

func TestRunWithProductionReadinessUsesFlagInputsOverEnv(t *testing.T) {
	t.Setenv("WRKR_PROFILE", "strict")
	t.Setenv("WRKR_SIGNING_MODE", "ed25519")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "env")
	t.Setenv("WRKR_RETENTION_DAYS", "7")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "7")
	t.Setenv("WRKR_SERVE_LISTEN", ":9488")
	t.Setenv("WRKR_SERVE_ALLOW_NON_LOOPBACK", "false")

	result, err := RunWithOptions(Options{
		Now:                      func() time.Time { return time.Date(2026, 2, 14, 2, 30, 0, 0, time.UTC) },
		ProductionReadiness:      true,
		ServeListenAddr:          "0.0.0.0:9488",
		ServeListenAddrSet:       true,
		ServeAllowNonLoopback:    true,
		ServeAllowNonLoopbackSet: true,
		ServeAuthToken:           "token",
		ServeAuthTokenSet:        true,
		ServeMaxBodyBytes:        1024,
		ServeMaxBodyBytesSet:     true,
	})
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected readiness pass with explicit hardened serve flags: %+v", result)
	}
}
