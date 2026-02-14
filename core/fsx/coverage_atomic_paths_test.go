package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFileCoveragePaths(t *testing.T) {
	t.Parallel()

	if err := AtomicWriteFile("", []byte("x"), 0o600); err == nil {
		t.Fatal("expected empty path atomic write error")
	}
	if err := AtomicWriteFile("../bad.txt", []byte("x"), 0o600); err == nil {
		t.Fatal("expected traversal path atomic write error")
	}
	if err := AtomicWriteFile("bad\x00path", []byte("x"), 0o600); err == nil {
		t.Fatal("expected invalid path atomic write error")
	}

	path := filepath.Join(t.TempDir(), "dir", "file.txt")
	if err := AtomicWriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatalf("AtomicWriteFile success path: %v", err)
	}
}

func TestAtomicWriteFileMkdirFailure(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	blockingFile := filepath.Join(base, "as-file")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	target := filepath.Join(blockingFile, "nested", "file.txt")
	if err := AtomicWriteFile(target, []byte("x"), 0o600); err == nil {
		t.Fatal("expected mkdir failure when parent path is a file")
	}
}

func TestAtomicWriteFileCreateTempFailure(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	lockedDir := filepath.Join(base, "locked")
	if err := os.MkdirAll(lockedDir, 0o500); err != nil {
		t.Fatalf("mkdir locked dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedDir, 0o700)
	})

	target := filepath.Join(lockedDir, "file.txt")
	if err := AtomicWriteFile(target, []byte("x"), 0o600); err == nil {
		t.Fatal("expected create temp failure in non-writable dir")
	}
}
