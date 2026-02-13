package approve

import "testing"

func TestResolveApprovedBy(t *testing.T) {
	t.Setenv("WRKR_APPROVED_BY", "from-env")
	if got := ResolveApprovedBy(""); got != "from-env" {
		t.Fatalf("expected from-env, got %q", got)
	}
	if got := ResolveApprovedBy("explicit"); got != "explicit" {
		t.Fatalf("expected explicit, got %q", got)
	}
}

func TestValidateReason(t *testing.T) {
	t.Parallel()

	if err := ValidateReason(" "); err == nil {
		t.Fatal("expected empty reason to fail")
	}
	if err := ValidateReason("needed to proceed"); err != nil {
		t.Fatalf("expected non-empty reason to pass, got %v", err)
	}
}
