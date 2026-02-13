package fsx

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var ErrLockBusy = errors.New("lock busy")

type LockOptions struct {
	StaleAfter time.Duration
	Now        func() time.Time
}

type FileLock struct {
	path  string
	owner string
}

// AcquireLock creates a lock file using O_EXCL so concurrent claimers fail fast.
func AcquireLock(path, owner string) (*FileLock, error) {
	return AcquireLockWithOptions(path, owner, LockOptions{})
}

// AcquireLockWithOptions creates a lock file and can reclaim stale lock files.
func AcquireLockWithOptions(path, owner string, opts LockOptions) (*FileLock, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	claim := func() (*FileLock, error) {
		// #nosec G304 -- caller passes a store-scoped lock path after job_id validation.
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
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

	lock, err := claim()
	if err == nil {
		return lock, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	if opts.StaleAfter > 0 {
		info, statErr := os.Stat(path)
		if statErr == nil && now().UTC().Sub(info.ModTime().UTC()) >= opts.StaleAfter {
			if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
				return nil, fmt.Errorf("remove stale lock: %w", rmErr)
			}
			lock, err = claim()
			if err == nil {
				return lock, nil
			}
			if !errors.Is(err, os.ErrExist) {
				return nil, fmt.Errorf("acquire lock: %w", err)
			}
		}
	}

	return nil, ErrLockBusy
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
