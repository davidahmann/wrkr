package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeAbsolutePath returns an absolute, cleaned path and rejects empty or
// NUL-containing inputs.
func NormalizeAbsolutePath(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return "", fmt.Errorf("path contains NUL byte")
	}

	abs, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return abs, nil
}

// ResolveWithinBase resolves a path and ensures it remains within baseDir.
func ResolveWithinBase(baseDir, input string) (string, error) {
	baseAbs, err := NormalizeAbsolutePath(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}
	baseCanon := canonicalizeExisting(baseAbs)
	target := strings.TrimSpace(input)
	if target == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsRune(target, '\x00') {
		return "", fmt.Errorf("path contains NUL byte")
	}

	candidate := target
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(baseAbs, candidate)
	}
	absTarget, err := NormalizeAbsolutePath(candidate)
	if err != nil {
		return "", err
	}
	targetCanon := canonicalizePath(absTarget)
	lexWithin := isWithinBase(baseAbs, absTarget)
	canonWithin := isWithinBase(baseCanon, targetCanon)

	if !canonWithin {
		return "", fmt.Errorf("path escapes base dir: %s", input)
	}
	if !lexWithin && !isWithinBase(baseCanon, absTarget) {
		return "", fmt.Errorf("path escapes base dir: %s", input)
	}
	return absTarget, nil
}

// ResolveWithinWorkingDir resolves a path within the current working directory.
func ResolveWithinWorkingDir(input string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working dir: %w", err)
	}
	return ResolveWithinBase(wd, input)
}

func canonicalizeExisting(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func canonicalizePath(path string) string {
	cleaned := filepath.Clean(path)
	probe := cleaned

	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			rel, err := filepath.Rel(probe, cleaned)
			if err != nil {
				return cleaned
			}
			if rel == "." {
				return resolved
			}
			return filepath.Clean(filepath.Join(resolved, rel))
		}
		if !os.IsNotExist(err) {
			return cleaned
		}

		next := filepath.Dir(probe)
		if next == probe {
			return cleaned
		}
		probe = next
	}
}

func isWithinBase(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
