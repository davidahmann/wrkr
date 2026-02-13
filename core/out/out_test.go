package out

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutPaths(t *testing.T) {
	l, err := NewLayout("")
	if err != nil {
		t.Fatalf("new layout: %v", err)
	}
	if got := l.JobpackPath("job_1"); !strings.Contains(filepath.ToSlash(got), "wrkr-out/jobpacks/jobpack_job_1.zip") {
		t.Fatalf("unexpected jobpack path: %s", got)
	}
	if got := l.IntegrationPath("ci", "work_item.json"); !strings.Contains(filepath.ToSlash(got), "wrkr-out/integrations/ci/work_item.json") {
		t.Fatalf("unexpected integration path: %s", got)
	}
	if got := l.ReportPath("summary.md"); !strings.Contains(filepath.ToSlash(got), "wrkr-out/reports/summary.md") {
		t.Fatalf("unexpected report path: %s", got)
	}
}

func TestEnsureCreatesDirs(t *testing.T) {
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	l, err := NewLayout("wrkr-out")
	if err != nil {
		t.Fatalf("new layout: %v", err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatalf("ensure layout dirs: %v", err)
	}
}
