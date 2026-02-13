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
	if len(result.Checks) == 0 {
		t.Fatal("expected checks")
	}
}
