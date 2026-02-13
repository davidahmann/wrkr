package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAtomicWriteFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := AtomicWriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("atomic write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("unexpected data: %q", string(data))
	}
}

func TestAcquireLockBusy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "append.lock")

	l1, err := AcquireLock(path, "owner1")
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	t.Cleanup(func() {
		_ = l1.Release()
	})

	_, err = AcquireLock(path, "owner2")
	if !errors.Is(err, ErrLockBusy) {
		t.Fatalf("expected ErrLockBusy, got %v", err)
	}
}

func TestAcquireLockReclaimsStaleLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "append.lock")
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	if err := os.WriteFile(path, []byte("stale-owner\n"), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}
	old := now.Add(-2 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("chtimes stale lock: %v", err)
	}

	lock, err := AcquireLockWithOptions(path, "owner2", LockOptions{
		StaleAfter: 30 * time.Second,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("expected stale lock reclaim, got %v", err)
	}
	t.Cleanup(func() { _ = lock.Release() })
}
