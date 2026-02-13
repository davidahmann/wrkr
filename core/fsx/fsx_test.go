package fsx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("create dead pid: %v", err)
	}
	if cmd.Process == nil {
		t.Fatal("expected process info")
	}

	owner := "pid=" + strconv.Itoa(cmd.Process.Pid) + ";ts=1\n"
	if err := os.WriteFile(path, []byte(owner), 0o600); err != nil {
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

func TestAcquireLockDoesNotReclaimActiveOwner(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "append.lock")
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	owner := "pid=" + strconv.Itoa(os.Getpid()) + ";ts=1\n"
	if err := os.WriteFile(path, []byte(owner), 0o600); err != nil {
		t.Fatalf("write active lock: %v", err)
	}
	old := now.Add(-2 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("chtimes active lock: %v", err)
	}

	_, err := AcquireLockWithOptions(path, "owner2", LockOptions{
		StaleAfter: 30 * time.Second,
		Now:        func() time.Time { return now },
	})
	if !errors.Is(err, ErrLockBusy) {
		t.Fatalf("expected ErrLockBusy for active owner, got %v", err)
	}
}

func TestResolveWithinBase(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	got, err := ResolveWithinBase(base, "nested/file.txt")
	if err != nil {
		t.Fatalf("ResolveWithinBase: %v", err)
	}
	want := filepath.Join(base, "nested", "file.txt")
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		wantResolved = want
	}
	gotResolved, err := filepath.EvalSymlinks(filepath.Dir(got))
	if err == nil {
		got = filepath.Join(gotResolved, filepath.Base(got))
	}
	if got != wantResolved {
		t.Fatalf("expected %s, got %s", wantResolved, got)
	}
}

func TestResolveWithinBaseRejectsEscape(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := ResolveWithinBase(base, "../escape.txt"); err == nil {
		t.Fatal("expected ResolveWithinBase to reject escaping path")
	}
}
