package pack

import (
	"sort"
)

type DiffResult struct {
	JobIDA  string   `json:"job_id_a"`
	JobIDB  string   `json:"job_id_b"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

func DiffJobpacks(pathA, pathB string) (DiffResult, error) {
	a, err := LoadArchive(pathA)
	if err != nil {
		return DiffResult{}, err
	}
	b, err := LoadArchive(pathB)
	if err != nil {
		return DiffResult{}, err
	}

	hashA := fileHashes(a.Manifest)
	hashB := fileHashes(b.Manifest)

	added := make([]string, 0)
	removed := make([]string, 0)
	changed := make([]string, 0)

	for path, shaA := range hashA {
		shaB, ok := hashB[path]
		if !ok {
			removed = append(removed, path)
			continue
		}
		if shaA != shaB {
			changed = append(changed, path)
		}
	}
	for path := range hashB {
		if _, ok := hashA[path]; !ok {
			added = append(added, path)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)

	return DiffResult{
		JobIDA:  a.Manifest.JobID,
		JobIDB:  b.Manifest.JobID,
		Added:   added,
		Removed: removed,
		Changed: changed,
	}, nil
}
