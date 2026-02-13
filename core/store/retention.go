package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultOutRoot = "./wrkr-out"

type PruneOptions struct {
	StoreRoot         string
	OutRoot           string
	Now               func() time.Time
	DryRun            bool
	JobMaxAge         time.Duration
	JobpackMaxAge     time.Duration
	ReportMaxAge      time.Duration
	IntegrationMaxAge time.Duration
	MaxJobpacks       int
	MaxReports        int
}

type PruneEntry struct {
	Kind      string    `json:"kind"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
	Reason    string    `json:"reason"`
}

type PruneReport struct {
	GeneratedAt time.Time    `json:"generated_at"`
	DryRun      bool         `json:"dry_run"`
	Checked     int          `json:"checked"`
	Matched     int          `json:"matched"`
	Removed     int          `json:"removed"`
	FreedBytes  int64        `json:"freed_bytes"`
	Entries     []PruneEntry `json:"entries"`
}

type candidate struct {
	kind      string
	path      string
	sizeBytes int64
	modTime   time.Time
	reason    string
}

type fileInfo struct {
	path    string
	modTime time.Time
	size    int64
}

func Prune(opts PruneOptions) (PruneReport, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	storeRoot := strings.TrimSpace(opts.StoreRoot)
	if storeRoot == "" {
		var err error
		storeRoot, err = DefaultRoot()
		if err != nil {
			return PruneReport{}, err
		}
	}
	outRoot := strings.TrimSpace(opts.OutRoot)
	if outRoot == "" {
		outRoot = defaultOutRoot
	}

	report := PruneReport{
		GeneratedAt: now().UTC(),
		DryRun:      opts.DryRun,
		Entries:     make([]PruneEntry, 0, 16),
	}

	candidates := map[string]candidate{}
	addCandidate := func(c candidate) {
		if existing, ok := candidates[c.path]; ok {
			if existing.reason == "age" {
				return
			}
			if c.reason == "age" {
				candidates[c.path] = c
			}
			return
		}
		candidates[c.path] = c
	}

	checked, err := collectJobCandidates(storeRoot, opts.JobMaxAge, now(), addCandidate)
	if err != nil {
		return PruneReport{}, err
	}
	report.Checked += checked

	jobpacks, checked, err := collectFileInfos(filepath.Join(outRoot, "jobpacks"), "*.zip")
	if err != nil {
		return PruneReport{}, err
	}
	report.Checked += checked
	applyAgeCandidates(jobpacks, opts.JobpackMaxAge, now(), "jobpack", addCandidate)
	applyCountCandidates(jobpacks, opts.MaxJobpacks, "jobpack", addCandidate)

	reports, checked, err := collectFileInfos(filepath.Join(outRoot, "reports"), "*")
	if err != nil {
		return PruneReport{}, err
	}
	report.Checked += checked
	applyAgeCandidates(reports, opts.ReportMaxAge, now(), "report", addCandidate)
	applyCountCandidates(reports, opts.MaxReports, "report", addCandidate)

	integrations, checked, err := collectFileInfosRecursive(filepath.Join(outRoot, "integrations"))
	if err != nil {
		return PruneReport{}, err
	}
	report.Checked += checked
	applyAgeCandidates(integrations, opts.IntegrationMaxAge, now(), "integration", addCandidate)

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	report.Matched = len(paths)

	for _, path := range paths {
		c := candidates[path]
		entry := PruneEntry{
			Kind:      c.kind,
			Path:      c.path,
			SizeBytes: c.sizeBytes,
			ModTime:   c.modTime.UTC(),
			Reason:    c.reason,
		}
		report.Entries = append(report.Entries, entry)

		if opts.DryRun {
			continue
		}
		if err := removePath(c.path); err != nil {
			return PruneReport{}, err
		}
		report.Removed++
		report.FreedBytes += c.sizeBytes
	}

	return report, nil
}

func collectJobCandidates(storeRoot string, maxAge time.Duration, now time.Time, add func(candidate)) (int, error) {
	jobsRoot := filepath.Join(filepath.Clean(storeRoot), "jobs")
	entries, err := os.ReadDir(jobsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read jobs root: %w", err)
	}

	checked := 0
	if maxAge <= 0 {
		for _, entry := range entries {
			if entry.IsDir() {
				checked++
			}
		}
		return checked, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		checked++
		jobPath := filepath.Join(jobsRoot, entry.Name())
		latest, sizeBytes, err := latestModAndSize(jobPath)
		if err != nil {
			return checked, err
		}
		if now.Sub(latest) >= maxAge {
			add(candidate{
				kind:      "job",
				path:      jobPath,
				sizeBytes: sizeBytes,
				modTime:   latest,
				reason:    "age",
			})
		}
	}
	return checked, nil
}

func collectFileInfos(dir, glob string) ([]fileInfo, int, error) {
	d := filepath.Clean(dir)
	entries, err := os.ReadDir(d)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("read dir %s: %w", d, err)
	}
	items := make([]fileInfo, 0, len(entries))
	checked := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		checked++
		matched, err := filepath.Match(glob, entry.Name())
		if err != nil {
			return nil, checked, err
		}
		if !matched {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, checked, err
		}
		items = append(items, fileInfo{
			path:    filepath.Join(d, entry.Name()),
			modTime: info.ModTime().UTC(),
			size:    info.Size(),
		})
	}
	sortFilesNewestFirst(items)
	return items, checked, nil
}

func collectFileInfosRecursive(root string) ([]fileInfo, int, error) {
	r := filepath.Clean(root)
	items := make([]fileInfo, 0, 16)
	checked := 0
	err := filepath.WalkDir(r, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		checked++
		info, err := d.Info()
		if err != nil {
			return err
		}
		items = append(items, fileInfo{
			path:    path,
			modTime: info.ModTime().UTC(),
			size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, nil
		}
		return nil, checked, fmt.Errorf("walk dir %s: %w", r, err)
	}
	sortFilesNewestFirst(items)
	return items, checked, nil
}

func applyAgeCandidates(items []fileInfo, maxAge time.Duration, now time.Time, kind string, add func(candidate)) {
	if maxAge <= 0 {
		return
	}
	for _, item := range items {
		if now.Sub(item.modTime) >= maxAge {
			add(candidate{
				kind:      kind,
				path:      item.path,
				sizeBytes: item.size,
				modTime:   item.modTime,
				reason:    "age",
			})
		}
	}
}

func applyCountCandidates(items []fileInfo, maxCount int, kind string, add func(candidate)) {
	if maxCount < 0 {
		return
	}
	if len(items) <= maxCount {
		return
	}
	for i := maxCount; i < len(items); i++ {
		item := items[i]
		add(candidate{
			kind:      kind,
			path:      item.path,
			sizeBytes: item.size,
			modTime:   item.modTime,
			reason:    "count",
		})
	}
}

func sortFilesNewestFirst(items []fileInfo) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].modTime.Equal(items[j].modTime) {
			return items[i].path < items[j].path
		}
		return items[i].modTime.After(items[j].modTime)
	})
}

func latestModAndSize(path string) (time.Time, int64, error) {
	var latest time.Time
	var size int64

	err := filepath.Walk(path, func(entryPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("walk %s: %w", path, err)
	}
	if latest.IsZero() {
		latest = time.Unix(0, 0).UTC()
	}
	return latest.UTC(), size, nil
}

func removePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat path %s: %w", path, err)
	}
	if info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove dir %s: %w", path, err)
		}
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file %s: %w", path, err)
	}
	return nil
}
