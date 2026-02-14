package approve

import "testing"

func TestResolveApprovedByCoveragePaths(t *testing.T) {
	t.Run("user fallback", func(t *testing.T) {
		t.Setenv("WRKR_APPROVED_BY", "")
		t.Setenv("USER", "local-user")
		if got := ResolveApprovedBy(""); got != "local-user" {
			t.Fatalf("expected USER fallback, got %q", got)
		}
	})

	t.Run("unknown fallback", func(t *testing.T) {
		t.Setenv("WRKR_APPROVED_BY", "")
		t.Setenv("USER", "")
		if got := ResolveApprovedBy(""); got != "unknown" {
			t.Fatalf("expected unknown fallback, got %q", got)
		}
	})
}

