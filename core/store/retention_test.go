package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneDryRunAgeAndCount(t *testing.T) {
	now := time.Date(2026, 2, 14, 4, 0, 0, 0, time.UTC)
	storeRoot := filepath.Join(t.TempDir(), ".wrkr")
	outRoot := filepath.Join(t.TempDir(), "wrkr-out")

	seedRetentionFixture(t, storeRoot, outRoot, now)

	report, err := Prune(PruneOptions{
		StoreRoot:         storeRoot,
		OutRoot:           outRoot,
		Now:               func() time.Time { return now },
		DryRun:            true,
		JobMaxAge:         24 * time.Hour,
		JobpackMaxAge:     24 * time.Hour,
		ReportMaxAge:      24 * time.Hour,
		IntegrationMaxAge: 24 * time.Hour,
		MaxJobpacks:       1,
		MaxReports:        1,
	})
	if err != nil {
		t.Fatalf("Prune dry-run: %v", err)
	}
	if !report.DryRun {
		t.Fatal("expected dry-run report")
	}
	if report.Matched == 0 {
		t.Fatal("expected prune candidates")
	}
	if report.Removed != 0 {
		t.Fatalf("expected no removals in dry-run, got %d", report.Removed)
	}

	if _, err := os.Stat(filepath.Join(storeRoot, "jobs", "job_old")); err != nil {
		t.Fatalf("dry-run should not delete old job dir: %v", err)
	}
}

func TestPruneDeletesExpiredAndRotatesCount(t *testing.T) {
	now := time.Date(2026, 2, 14, 4, 0, 0, 0, time.UTC)
	storeRoot := filepath.Join(t.TempDir(), ".wrkr")
	outRoot := filepath.Join(t.TempDir(), "wrkr-out")

	seedRetentionFixture(t, storeRoot, outRoot, now)

	report, err := Prune(PruneOptions{
		StoreRoot:         storeRoot,
		OutRoot:           outRoot,
		Now:               func() time.Time { return now },
		DryRun:            false,
		JobMaxAge:         24 * time.Hour,
		JobpackMaxAge:     24 * time.Hour,
		ReportMaxAge:      24 * time.Hour,
		IntegrationMaxAge: 24 * time.Hour,
		MaxJobpacks:       1,
		MaxReports:        1,
	})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if report.Removed == 0 {
		t.Fatal("expected removals")
	}
	if report.FreedBytes <= 0 {
		t.Fatalf("expected freed bytes > 0, got %d", report.FreedBytes)
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "jobs", "job_old")); !os.IsNotExist(err) {
		t.Fatalf("expected old job dir removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "jobs", "job_new")); err != nil {
		t.Fatalf("expected new job dir kept: %v", err)
	}

	jobpacks, err := filepath.Glob(filepath.Join(outRoot, "jobpacks", "*.zip"))
	if err != nil {
		t.Fatalf("glob jobpacks: %v", err)
	}
	if len(jobpacks) != 1 {
		t.Fatalf("expected one jobpack after rotation, got %d (%v)", len(jobpacks), jobpacks)
	}
}

func seedRetentionFixture(t *testing.T, storeRoot, outRoot string, now time.Time) {
	t.Helper()

	for _, p := range []string{
		filepath.Join(storeRoot, "jobs", "job_old"),
		filepath.Join(storeRoot, "jobs", "job_new"),
		filepath.Join(outRoot, "jobpacks"),
		filepath.Join(outRoot, "reports"),
		filepath.Join(outRoot, "integrations", "lane"),
	} {
		if err := os.MkdirAll(p, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	write := func(path string, age time.Duration) {
		t.Helper()
		if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		mod := now.Add(-age)
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}

	write(filepath.Join(storeRoot, "jobs", "job_old", "events.jsonl"), 72*time.Hour)
	write(filepath.Join(storeRoot, "jobs", "job_new", "events.jsonl"), 2*time.Hour)
	write(filepath.Join(outRoot, "jobpacks", "jobpack_old.zip"), 48*time.Hour)
	write(filepath.Join(outRoot, "jobpacks", "jobpack_new.zip"), 1*time.Hour)
	write(filepath.Join(outRoot, "reports", "accept_old.json"), 30*time.Hour)
	write(filepath.Join(outRoot, "reports", "accept_new.json"), 2*time.Hour)
	write(filepath.Join(outRoot, "integrations", "lane", "old.log"), 30*time.Hour)
}
