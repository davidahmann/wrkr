package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDoctorProductionReadinessFailsWithoutStrictConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WRKR_PROFILE", "")
	t.Setenv("WRKR_SIGNING_MODE", "")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "")
	t.Setenv("WRKR_RETENTION_DAYS", "")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "")

	now := time.Date(2026, 2, 14, 4, 30, 0, 0, time.UTC)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"doctor", "--production-readiness", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code == 0 {
		t.Fatalf("expected non-zero for failed production readiness, stdout=%s stderr=%s", out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), `"profile": "production-readiness"`) {
		t.Fatalf("expected production profile in output: %s", out.String())
	}
	if !strings.Contains(out.String(), `"strict_config_profile"`) {
		t.Fatalf("expected strict config check in output: %s", out.String())
	}
}

func TestDoctorProductionReadinessPassesWithStrictConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WRKR_PROFILE", "strict")
	t.Setenv("WRKR_SIGNING_MODE", "ed25519")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "file")
	t.Setenv("WRKR_RETENTION_DAYS", "14")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "14")
	t.Setenv("WRKR_SERVE_LISTEN", "127.0.0.1:9488")
	t.Setenv("WRKR_ALLOW_UNSAFE", "0")

	keyPath := filepath.Join(t.TempDir(), "wrkr-ed25519.key")
	if err := os.WriteFile(keyPath, []byte("fake-ed25519-private-key"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	t.Setenv("WRKR_SIGNING_KEY_PATH", keyPath)

	now := time.Date(2026, 2, 14, 4, 30, 0, 0, time.UTC)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"doctor", "--production-readiness", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 0 {
		t.Fatalf("expected readiness pass, got code=%d stdout=%s stderr=%s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("expected ok=true output: %s", out.String())
	}
}

func TestDoctorProductionReadinessServeFlagsRequireHardening(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WRKR_PROFILE", "strict")
	t.Setenv("WRKR_SIGNING_MODE", "ed25519")
	t.Setenv("WRKR_SIGNING_KEY_SOURCE", "env")
	t.Setenv("WRKR_RETENTION_DAYS", "14")
	t.Setenv("WRKR_OUTPUT_RETENTION_DAYS", "14")

	now := time.Date(2026, 2, 14, 4, 30, 0, 0, time.UTC)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run(
		[]string{"doctor", "--production-readiness", "--serve-listen", ":9488", "--json"},
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code == 0 {
		t.Fatalf("expected readiness failure for wildcard non-hardened serve listen, stdout=%s stderr=%s", out.String(), errBuf.String())
	}

	out.Reset()
	errBuf.Reset()
	code = run(
		[]string{"doctor", "--production-readiness", "--serve-listen", ":9488", "--serve-allow-non-loopback", "--serve-auth-token", "token", "--serve-max-body-bytes", "1024", "--json"},
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code != 0 {
		t.Fatalf("expected readiness pass for hardened wildcard serve config, stdout=%s stderr=%s", out.String(), errBuf.String())
	}
}

func TestStorePruneDryRunJSON(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), ".wrkr")
	outRoot := filepath.Join(t.TempDir(), "wrkr-out")

	if err := os.MkdirAll(filepath.Join(storeRoot, "jobs", "job_old"), 0o750); err != nil {
		t.Fatalf("mkdir store fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeRoot, "jobs", "job_old", "events.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write store fixture: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(outRoot, "jobpacks"), 0o750); err != nil {
		t.Fatalf("mkdir out fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outRoot, "jobpacks", "jobpack_old.zip"), []byte("zip"), 0o600); err != nil {
		t.Fatalf("write out fixture: %v", err)
	}

	old := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(storeRoot, "jobs", "job_old", "events.jsonl"), old, old); err != nil {
		t.Fatalf("chtimes store fixture: %v", err)
	}
	if err := os.Chtimes(filepath.Join(outRoot, "jobpacks", "jobpack_old.zip"), old, old); err != nil {
		t.Fatalf("chtimes out fixture: %v", err)
	}

	now := time.Date(2026, 2, 14, 4, 30, 0, 0, time.UTC)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run(
		[]string{"store", "prune", "--store-root", storeRoot, "--out-dir", outRoot, "--job-max-age", "24h", "--jobpack-max-age", "24h", "--dry-run", "--json"},
		&out,
		&errBuf,
		func() time.Time { return now },
	)
	if code != 0 {
		t.Fatalf("store prune dry-run failed: code=%d stdout=%s stderr=%s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), `"dry_run": true`) {
		t.Fatalf("expected dry_run flag in output: %s", out.String())
	}
	if !strings.Contains(out.String(), `"matched":`) {
		t.Fatalf("expected matched count in output: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "jobs", "job_old")); err != nil {
		t.Fatalf("dry-run should not remove job dir: %v", err)
	}
}

func TestStorePruneRequiresCriteria(t *testing.T) {
	now := time.Date(2026, 2, 14, 4, 30, 0, 0, time.UTC)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"store", "prune", "--json"}, &out, &errBuf, func() time.Time { return now })
	if code != 6 {
		t.Fatalf("expected E_INVALID_INPUT_SCHEMA exit code 6, got %d (stderr=%s)", code, errBuf.String())
	}
}
