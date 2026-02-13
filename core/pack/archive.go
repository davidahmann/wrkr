package pack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/davidahmann/wrkr/core/jcs"
	v1 "github.com/davidahmann/wrkr/core/schema/v1"
	"github.com/davidahmann/wrkr/core/sign"
)

type Archive struct {
	Path     string
	Manifest v1.JobpackManifest
	Files    map[string][]byte
}

func LoadArchive(path string) (*Archive, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open jobpack zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	files := make(map[string][]byte, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read zip entry %s: %w", f.Name, err)
		}
		files[f.Name] = data
	}

	manifestRaw, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("jobpack missing manifest.json")
	}
	var manifest v1.JobpackManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest.json: %w", err)
	}

	return &Archive{
		Path:     path,
		Manifest: manifest,
		Files:    files,
	}, nil
}

func EncodeJSONCanonical(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	canonical, err := jcs.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func MarshalJSONLCanonical[T any](records []T) ([]byte, error) {
	var b strings.Builder
	for _, record := range records {
		line, err := EncodeJSONCanonical(record)
		if err != nil {
			return nil, err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func ComputeManifestSHA256(manifest v1.JobpackManifest) (string, error) {
	tmp := manifest
	tmp.ManifestSHA256 = strings.Repeat("0", 64)
	data, err := EncodeJSONCanonical(tmp)
	if err != nil {
		return "", err
	}
	return sign.SHA256Hex(data), nil
}

func SortedFileList(files map[string][]byte) []v1.ManifestFile {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	out := make([]v1.ManifestFile, 0, len(paths))
	for _, path := range paths {
		out = append(out, v1.ManifestFile{
			Path:   path,
			SHA256: sign.SHA256Hex(files[path]),
		})
	}
	return out
}
