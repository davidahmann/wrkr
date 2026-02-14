package envfp

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultRulesCoverage(t *testing.T) {
	t.Parallel()

	rules := DefaultRules()
	if len(rules) != 3 {
		t.Fatalf("expected 3 default rules, got %d (%v)", len(rules), rules)
	}
}

func TestCaptureCoveragePaths(t *testing.T) {
	t.Run("fallback to defaults", func(t *testing.T) {
		fp, err := Capture(nil, time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("Capture: %v", err)
		}
		if len(fp.Rules) == 0 || fp.Hash == "" {
			t.Fatalf("expected rules/hash, got %+v", fp)
		}
	})

	t.Run("trim and dedupe", func(t *testing.T) {
		fp, err := Capture([]string{" env:WRKR_ENVFP_A ", "env:WRKR_ENVFP_A", "", "cwd"}, time.Now())
		if err != nil {
			t.Fatalf("Capture: %v", err)
		}
		if len(fp.Rules) != 2 {
			t.Fatalf("expected deduped rules, got %v", fp.Rules)
		}
	})

	t.Run("unsupported rule", func(t *testing.T) {
		if _, err := Capture([]string{"unknown_rule"}, time.Now()); err == nil {
			t.Fatal("expected unsupported rule error")
		}
	})

	t.Run("hostname and cwd", func(t *testing.T) {
		fp, err := Capture([]string{"hostname", "cwd"}, time.Now())
		if err != nil {
			t.Fatalf("Capture: %v", err)
		}
		if strings.TrimSpace(fp.Values["hostname"]) == "" || strings.TrimSpace(fp.Values["cwd"]) == "" {
			t.Fatalf("expected hostname and cwd values, got %+v", fp.Values)
		}
	})
}

