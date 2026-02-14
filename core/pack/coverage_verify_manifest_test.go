package pack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidahmann/wrkr/core/zipx"
)

func TestVerifyManifestCoveragePaths(t *testing.T) {
	t.Parallel()

	zipBytes, err := zipx.BuildDeterministic([]zipx.Entry{
		{Name: "manifest.json", Data: []byte(`{"schema_id":"wrkr.jobpack_manifest"}`)},
	})
	if err != nil {
		t.Fatalf("BuildDeterministic: %v", err)
	}
	path := filepath.Join(t.TempDir(), "bad-manifest-schema.zip")
	if err := os.WriteFile(path, zipBytes, 0o600); err != nil {
		t.Fatalf("write bad manifest zip: %v", err)
	}

	if _, err := VerifyJobpack(path); err == nil {
		t.Fatal("expected verify failure for invalid manifest schema")
	}
}

