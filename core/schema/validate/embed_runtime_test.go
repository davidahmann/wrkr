package validate

import (
	"os"
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
