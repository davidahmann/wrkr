package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockHelperCoveragePaths(t *testing.T) {
	t.Parallel()

	if pid, ok := parseOwnerPID("pid=123;ts=1"); !ok || pid != 123 {
		t.Fatalf("expected parseOwnerPID success, got pid=%d ok=%t", pid, ok)
	}
	if _, ok := parseOwnerPID("pid=bad"); ok {
		t.Fatal("expected parseOwnerPID invalid value to fail")
	}
	if _, ok := parseOwnerPID("ts=1"); ok {
		t.Fatal("expected parseOwnerPID missing pid to fail")
	}

	if dead := lockOwnerIsDead("unknown-owner-format"); dead {
		t.Fatal("expected unknown owner format to fail closed")
	}

	if _, err := AcquireLockWithOptions("", "owner", LockOptions{}); err == nil {
		t.Fatal("expected invalid lock path to fail")
	}

	var lock *FileLock
	if err := lock.Release(); err != nil {
		t.Fatalf("nil lock release should be no-op: %v", err)
	}
	lock = &FileLock{path: "bad\x00path", owner: "owner"}
	if err := lock.Release(); err == nil {
		t.Fatal("expected invalid release path error")
	}
}

func TestReadLockOwnerCoveragePaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "append.lock")
	if err := os.WriteFile(lockPath, []byte("pid=1;ts=1\n"), 0o600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	owner, err := readLockOwner(lockPath)
	if err != nil {
		t.Fatalf("readLockOwner: %v", err)
	}
	if owner != "pid=1;ts=1" {
		t.Fatalf("unexpected lock owner %q", owner)
	}

	if _, err := readLockOwner(filepath.Join(dir, "missing.lock")); err == nil {
		t.Fatal("expected missing lock file error")
	}
}

