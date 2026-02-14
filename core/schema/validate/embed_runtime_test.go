package validate

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidahmann/wrkr/schemas"
)

func TestCompileUsesEmbeddedSchemas(t *testing.T) {
	t.Setenv("WRKR_SCHEMA_ROOT", "")

	if _, err := Compile(JobSchemaRel); err != nil {
		t.Fatalf("Compile embedded schema: %v", err)
	}
}

func TestCompileRejectsTraversalSchemaPath(t *testing.T) {
	t.Setenv("WRKR_SCHEMA_ROOT", "")

	if _, err := Compile("../jobpack/job.schema.json"); err == nil {
		t.Fatal("expected traversal schema path to fail")
	}
}

func TestCompileWithExplicitSchemaRoot(t *testing.T) {
	t.Setenv("WRKR_SCHEMA_ROOT", "")
	root := t.TempDir()

	for _, rel := range SchemaList() {
		payload, err := schemas.V1FS.ReadFile(filepath.ToSlash(filepath.Join("v1", rel)))
		if err != nil {
			t.Fatalf("read embedded schema %s: %v", rel, err)
		}

		dst := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir schema dir: %v", err)
		}
		if err := os.WriteFile(dst, payload, 0o644); err != nil {
			t.Fatalf("write schema %s: %v", rel, err)
		}
	}

	t.Setenv("WRKR_SCHEMA_ROOT", root)
	if _, err := Compile(JobSchemaRel); err != nil {
		t.Fatalf("Compile explicit root schema: %v", err)
	}
}

func TestCompileExplicitRootMissingSchema(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WRKR_SCHEMA_ROOT", root)

	_, err := Compile(JobSchemaRel)
	if err == nil {
		t.Fatal("expected missing schema error")
	}
	if !strings.Contains(err.Error(), "schema not found:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileRejectsAdditionalInvalidSchemaPaths(t *testing.T) {
	t.Setenv("WRKR_SCHEMA_ROOT", "")
	cases := []string{
		".",
		"..",
		"/jobpack/job.schema.json",
		"jobpack/../job.schema.json",
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			if _, err := Compile(tc); err == nil {
				t.Fatalf("expected invalid schema path %q to fail", tc)
			}
		})
	}
}

func TestSchemaURL(t *testing.T) {
	got := schemaURL("jobpack/job.schema.json")
	want := "https://wrkr.dev/schemas/v1/jobpack/job.schema.json"
	if got != want {
		t.Fatalf("schemaURL mismatch: got=%q want=%q", got, want)
	}
}

func TestReadSchemaBytesEmbeddedMissing(t *testing.T) {
	t.Setenv("WRKR_SCHEMA_ROOT", "")
	if _, err := readSchemaBytes("missing.schema.json"); err == nil {
		t.Fatal("expected embedded schema read to fail for missing file")
	}
}

func TestReadSchemaBytesExplicitRootReadError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WRKR_SCHEMA_ROOT", root)
	if _, err := readSchemaBytes("jobpack/job.schema.json"); err == nil {
		t.Fatal("expected explicit root read error for missing file")
	}
}

func TestReadSchemaBytesExplicitRootOpenError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	t.Setenv("WRKR_SCHEMA_ROOT", filePath)

	if _, err := readSchemaBytes("jobpack/job.schema.json"); err == nil {
		t.Fatal("expected open root failure when WRKR_SCHEMA_ROOT is a file")
	}
}

func TestSchemaPathWithExplicitRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WRKR_SCHEMA_ROOT", root)
	rel := path.Join("jobpack", "manifest.schema.json")

	dst := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir schema dir: %v", err)
	}
	if err := os.WriteFile(dst, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write schema file: %v", err)
	}

	got, err := SchemaPath(rel)
	if err != nil {
		t.Fatalf("SchemaPath explicit root: %v", err)
	}
	if got != dst {
		t.Fatalf("SchemaPath mismatch: got=%q want=%q", got, dst)
	}
}
