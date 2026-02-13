package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWriteFile writes data to a temp file in the same directory and renames
// it over the destination path.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("invalid atomic write path: path is required")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid atomic write path: path traversal is not allowed")
	}
	resolvedPath, err := NormalizeAbsolutePath(path)
	if err != nil {
		return fmt.Errorf("invalid atomic write path: %w", err)
	}
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir for atomic write: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, resolvedPath); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
