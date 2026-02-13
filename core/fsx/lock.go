package fsx

import (
	"errors"
	"fmt"
	"os"
)

var ErrLockBusy = errors.New("lock busy")

type FileLock struct {
	path  string
	owner string
}

// AcquireLock creates a lock file using O_EXCL so concurrent claimers fail fast.
func AcquireLock(path, owner string) (*FileLock, error) {
	// #nosec G304 -- caller passes a store-scoped lock path after job_id validation.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrLockBusy
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	if _, err := file.WriteString(owner + "\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write lock owner: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("sync lock file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close lock file: %w", err)
	}

	return &FileLock{path: path, owner: owner}, nil
}

func (l *FileLock) Release() error {
	if l == nil {
		return nil
	}
	if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}
