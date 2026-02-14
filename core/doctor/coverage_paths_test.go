package doctor

import (
	"strings"
	"testing"
	"time"
)

func TestItoaCoveragePaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value int
		want  string
	}{
		{value: 0, want: "0"},
		{value: 7, want: "7"},
		{value: -42, want: "42"},
	}
	for _, tc := range cases {
		if got := itoa(tc.value); got != tc.want {
			t.Fatalf("itoa(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestParsePositiveIntFromEnvCoveragePaths(t *testing.T) {
	key := "WRKR_TEST_PARSE_INT"

	t.Run("unset", func(t *testing.T) {
		t.Setenv(key, "")
		if _, err := parsePositiveIntFromEnv(key); err == nil {
			t.Fatal("expected unset value to fail")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv(key, "abc")
		if _, err := parsePositiveIntFromEnv(key); err == nil {
			t.Fatal("expected invalid value to fail")
		}
	})

	t.Run("zero", func(t *testing.T) {
		t.Setenv(key, "0")
		if _, err := parsePositiveIntFromEnv(key); err == nil {
			t.Fatal("expected non-positive value to fail")
		}
	})

	t.Run("ok", func(t *testing.T) {
		t.Setenv(key, "12")
		got, err := parsePositiveIntFromEnv(key)
		if err != nil {
			t.Fatalf("parsePositiveIntFromEnv: %v", err)
		}
		if got != 12 {
			t.Fatalf("expected 12, got %d", got)
		}
	})
}

func TestResolveServeConfigCoveragePaths(t *testing.T) {
	t.Run("no inputs", func(t *testing.T) {
		cfg, source, provided, err := resolveServeConfig(Options{})
		if err != nil {
			t.Fatalf("resolveServeConfig: %v", err)
		}
		if provided {
			t.Fatalf("expected provided=false, got true with cfg=%+v source=%s", cfg, source)
		}
		if source != "default" {
			t.Fatalf("expected source=default, got %s", source)
		}
	})

	t.Run("invalid max body env", func(t *testing.T) {
		t.Setenv("WRKR_SERVE_MAX_BODY_BYTES", "nope")
		if _, _, _, err := resolveServeConfig(Options{}); err == nil {
			t.Fatal("expected invalid max body env to fail")
		}
	})

	t.Run("invalid allow env", func(t *testing.T) {
		t.Setenv("WRKR_SERVE_MAX_BODY_BYTES", "")
		t.Setenv("WRKR_SERVE_ALLOW_NON_LOOPBACK", "not-bool")
		if _, _, _, err := resolveServeConfig(Options{}); err == nil {
			t.Fatal("expected invalid allow env to fail")
		}
	})

	t.Run("env source", func(t *testing.T) {
		t.Setenv("WRKR_SERVE_LISTEN", "127.0.0.1:9488")
		t.Setenv("WRKR_SERVE_AUTH_TOKEN", "env-token")
		t.Setenv("WRKR_SERVE_MAX_BODY_BYTES", "1024")
		t.Setenv("WRKR_SERVE_ALLOW_NON_LOOPBACK", "false")

		cfg, source, provided, err := resolveServeConfig(Options{})
		if err != nil {
			t.Fatalf("resolveServeConfig: %v", err)
		}
		if !provided || source != "env" {
			t.Fatalf("expected provided env config, got provided=%t source=%s", provided, source)
		}
		if cfg.ListenAddr != "127.0.0.1:9488" || cfg.AuthToken != "env-token" || cfg.MaxBodyBytes != 1024 {
			t.Fatalf("unexpected env cfg: %+v", cfg)
		}
	})

	t.Run("flags override env", func(t *testing.T) {
		t.Setenv("WRKR_SERVE_LISTEN", "127.0.0.1:9488")
		t.Setenv("WRKR_SERVE_AUTH_TOKEN", "env-token")
		t.Setenv("WRKR_SERVE_MAX_BODY_BYTES", "1024")
		t.Setenv("WRKR_SERVE_ALLOW_NON_LOOPBACK", "false")

		cfg, source, provided, err := resolveServeConfig(Options{
			ServeListenAddr:          "0.0.0.0:9488",
			ServeAuthToken:           "flag-token",
			ServeMaxBodyBytes:        4096,
			ServeAllowNonLoopback:    true,
			ServeListenAddrSet:       true,
			ServeAuthTokenSet:        true,
			ServeMaxBodyBytesSet:     true,
			ServeAllowNonLoopbackSet: true,
		})
		if err != nil {
			t.Fatalf("resolveServeConfig: %v", err)
		}
		if !provided || source != "flags" {
			t.Fatalf("expected provided flags config, got provided=%t source=%s", provided, source)
		}
		if cfg.ListenAddr != "0.0.0.0:9488" || cfg.AuthToken != "flag-token" || cfg.MaxBodyBytes != 4096 || !cfg.AllowNonLoopback {
			t.Fatalf("unexpected flag cfg: %+v", cfg)
		}
	})
}

func TestValidateServeHardeningCoveragePaths(t *testing.T) {
	t.Run("no explicit input", func(t *testing.T) {
		details, err := validateServeHardening(Options{})
		if err != nil {
			t.Fatalf("validateServeHardening: %v", err)
		}
		if !strings.Contains(details, "default loopback assumptions") {
			t.Fatalf("unexpected details: %q", details)
		}
	})

	t.Run("loopback explicit", func(t *testing.T) {
		details, err := validateServeHardening(Options{
			ServeListenAddr:    "127.0.0.1:9488",
			ServeListenAddrSet: true,
		})
		if err != nil {
			t.Fatalf("validateServeHardening: %v", err)
		}
		if !strings.Contains(details, "serve listen=127.0.0.1:9488") {
			t.Fatalf("unexpected details: %q", details)
		}
	})

	t.Run("non-loopback missing hardening", func(t *testing.T) {
		_, err := validateServeHardening(Options{
			ServeListenAddr:    "0.0.0.0:9488",
			ServeListenAddrSet: true,
		})
		if err == nil {
			t.Fatal("expected hardening validation error")
		}
	})

	t.Run("non-loopback hardened", func(t *testing.T) {
		details, err := validateServeHardening(Options{
			ServeListenAddr:          "0.0.0.0:9488",
			ServeListenAddrSet:       true,
			ServeAllowNonLoopback:    true,
			ServeAllowNonLoopbackSet: true,
			ServeAuthToken:           "token",
			ServeAuthTokenSet:        true,
			ServeMaxBodyBytes:        8192,
			ServeMaxBodyBytesSet:     true,
		})
		if err != nil {
			t.Fatalf("validateServeHardening: %v", err)
		}
		if !strings.Contains(details, "non-loopback hardened") {
			t.Fatalf("unexpected details: %q", details)
		}
	})
}

func TestProductionReadinessChecksCoveragePaths(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC) }

	t.Run("failure defaults", func(t *testing.T) {
		checks := productionReadinessChecks(Options{Now: now})
		if len(checks) == 0 {
			t.Fatal("expected checks")
		}
		foundCritical := false
		for _, check := range checks {
			if !check.OK && check.Severity == "critical" {
				foundCritical = true
				break
			}
		}
		if !foundCritical {
			t.Fatalf("expected at least one critical failure: %+v", checks)
		}
	})

	t.Run("unsafe flag fails", func(t *testing.T) {
		t.Setenv("WRKR_PROFILE", "strict")
		t.Setenv("WRKR_SIGNING_MODE", "ed25519")
		t.Setenv("WRKR_SIGNING_KEY_SOURCE", "env")
		t.Setenv("WRKR_RETENTION_DAYS", "7")
		t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "7")
		t.Setenv("WRKR_ALLOW_UNSAFE", "true")

		checks := productionReadinessChecks(Options{Now: now})
		unsafeFailed := false
		for _, check := range checks {
			if check.Name == "unsafe_defaults" {
				unsafeFailed = !check.OK
				break
			}
		}
		if !unsafeFailed {
			t.Fatalf("expected unsafe_defaults to fail when WRKR_ALLOW_UNSAFE=true: %+v", checks)
		}
	})

	t.Run("file key source requires existing file", func(t *testing.T) {
		t.Setenv("WRKR_PROFILE", "strict")
		t.Setenv("WRKR_SIGNING_MODE", "ed25519")
		t.Setenv("WRKR_SIGNING_KEY_SOURCE", "file")
		t.Setenv("WRKR_SIGNING_KEY_PATH", "missing.key")
		t.Setenv("WRKR_RETENTION_DAYS", "7")
		t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "7")

		checks := productionReadinessChecks(Options{Now: now})
		keySourceFailed := false
		for _, check := range checks {
			if check.Name == "signing_key_source" {
				keySourceFailed = !check.OK
				break
			}
		}
		if !keySourceFailed {
			t.Fatalf("expected signing_key_source to fail for missing key path: %+v", checks)
		}
	})
}

func TestIsLoopbackListenCoveragePaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:9488", want: true},
		{addr: "[::1]:9488", want: true},
		{addr: "localhost:9488", want: true},
		{addr: ":9488", want: false},
		{addr: "0.0.0.0:9488", want: false},
	}
	for _, tc := range cases {
		if got := isLoopbackListen(tc.addr); got != tc.want {
			t.Fatalf("isLoopbackListen(%q) = %t, want %t", tc.addr, got, tc.want)
		}
	}
}
