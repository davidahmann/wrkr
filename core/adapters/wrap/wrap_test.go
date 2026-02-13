package wrap

import (
	"testing"
	"time"

	"github.com/davidahmann/wrkr/core/store"
)

func TestRunSuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)

	result, err := Run("job_wrap_success", []string{"sh", "-lc", "printf ok"}, RunOptions{
		Now:            func() time.Time { return now },
		ExpectedOutput: []string{"reports/out.md"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 || result.Status != "completed" {
		t.Fatalf("unexpected result: %+v", result)
	}

	s, err := store.New("")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	events, err := s.LoadEvents("job_wrap_success")
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events to be written")
	}
}

func TestRunFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC)

	result, err := Run("job_wrap_fail", []string{"sh", "-lc", "exit 7"}, RunOptions{
		Now: func() time.Time { return now },
	})
	if err == nil {
		t.Fatal("expected adapter failure")
	}
	if result.ExitCode != 7 {
		t.Fatalf("expected exit 7, got %d", result.ExitCode)
	}
}
