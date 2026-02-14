package fsx

import (
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

