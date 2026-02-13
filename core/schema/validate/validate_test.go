package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fixture struct {
	Schema string `json:"schema"`
	Data   any    `json:"data"`
}

func loadFixtures(t *testing.T, dir string) []fixture {
	t.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	out := make([]fixture, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			t.Fatalf("read fixture %s: %v", f.Name(), err)
		}
		var fx fixture
		if err := json.Unmarshal(content, &fx); err != nil {
			t.Fatalf("unmarshal fixture %s: %v", f.Name(), err)
		}
		out = append(out, fx)
	}

	return out
}

func TestCompileSchemaList(t *testing.T) {
	for _, schema := range SchemaList() {
		schema := schema
		t.Run(schema, func(t *testing.T) {
			t.Parallel()
			if _, err := Compile(schema); err != nil {
				t.Fatalf("compile schema %s: %v", schema, err)
			}
		})
	}
}

func TestValidFixtures(t *testing.T) {
	fixtures := loadFixtures(t, filepath.Join("..", "testdata", "valid"))
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.Schema, func(t *testing.T) {
			t.Parallel()
			if err := ValidateValue(fx.Schema, fx.Data); err != nil {
				t.Fatalf("expected valid fixture, got error: %v", err)
			}
		})
	}
}

func TestInvalidFixtures(t *testing.T) {
	fixtures := loadFixtures(t, filepath.Join("..", "testdata", "invalid"))
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.Schema, func(t *testing.T) {
			t.Parallel()
			if err := ValidateValue(fx.Schema, fx.Data); err == nil {
				t.Fatalf("expected invalid fixture to fail validation")
			}
		})
	}
}

func TestServeOpenAPIIsValidJSON(t *testing.T) {
	path, err := SchemaPath(filepath.Join("serve", "api.openapi.json"))
	if err != nil {
		t.Fatalf("resolve openapi path: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		t.Fatalf("openapi is not valid json: %v", err)
	}

	if raw["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", raw["openapi"])
	}
}
