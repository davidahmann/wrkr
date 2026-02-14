package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeAbsolutePath(""); err == nil {
		t.Fatal("expected empty path error")
	}
	if _, err := NormalizeAbsolutePath("bad\x00path"); err == nil {
		t.Fatal("expected NUL path error")
	}

	base := t.TempDir()
	if _, err := ResolveWithinBase(base, ""); err == nil {
		t.Fatal("expected empty relative path error")
	}
	if _, err := ResolveWithinBase(base, "bad\x00path"); err == nil {
		t.Fatal("expected NUL relative path error")
	}
}

func TestResolveWithinWorkingDirCoverage(t *testing.T) {
	wd := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	got, err := ResolveWithinWorkingDir("sub/dir/file.txt")
	if err != nil {
		t.Fatalf("ResolveWithinWorkingDir: %v", err)
	}
	resolvedWD, err := filepath.EvalSymlinks(wd)
	if err != nil {
		resolvedWD = wd
	}
	want := filepath.Join(resolvedWD, "sub", "dir", "file.txt")
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
