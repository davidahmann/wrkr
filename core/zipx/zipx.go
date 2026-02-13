package zipx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sort"
	"time"
)

type Entry struct {
	Name string
	Data []byte
}

var deterministicZipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func BuildDeterministic(entries []Entry) ([]byte, error) {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, entry := range sorted {
		header := &zip.FileHeader{
			Name:     entry.Name,
			Method:   zip.Store,
			Modified: deterministicZipTime,
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create zip entry %s: %w", entry.Name, err)
		}
		if _, err := w.Write(entry.Data); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write zip entry %s: %w", entry.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	return buf.Bytes(), nil
}
