package envfp

import (
	"testing"
	"time"
)

func TestCaptureStableHashForSameRules(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	a, err := Capture([]string{"arch", "os", "go_version"}, now)
	if err != nil {
		t.Fatalf("capture a: %v", err)
	}
	b, err := Capture([]string{"go_version", "os", "arch"}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("capture b: %v", err)
	}

	if a.Hash != b.Hash {
		t.Fatalf("expected stable hash for same rule set, got %s vs %s", a.Hash, b.Hash)
	}
}

func TestCaptureEnvRuleReflectsChanges(t *testing.T) {
	t.Setenv("WRKR_TEST_ENVFP", "one")
	a, err := Capture([]string{"env:WRKR_TEST_ENVFP"}, time.Now())
	if err != nil {
		t.Fatalf("capture a: %v", err)
	}

	t.Setenv("WRKR_TEST_ENVFP", "two")
	b, err := Capture([]string{"env:WRKR_TEST_ENVFP"}, time.Now())
	if err != nil {
		t.Fatalf("capture b: %v", err)
	}

	if a.Hash == b.Hash {
		t.Fatalf("expected hash change after env change, got %s", a.Hash)
	}
}
