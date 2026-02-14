package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/davidahmann/wrkr/schemas"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

func Compile(rel string) (*jsonschema.Schema, error) {
	normalizedRel, err := normalizeSchemaRel(rel)
	if err != nil {
		return nil, err
	}

	compiler := jsonschema.NewCompiler()
	seen := make(map[string]struct{}, len(SchemaList())+1)
	for _, resourceRel := range append(SchemaList(), normalizedRel) {
		resourceRel, err := normalizeSchemaRel(resourceRel)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[resourceRel]; ok {
			continue
		}

		payload, err := readSchemaBytes(resourceRel)
		if err != nil {
			return nil, err
		}

		resourceURL := schemaURL(resourceRel)
		if err := compiler.AddResource(resourceURL, bytes.NewReader(payload)); err != nil {
			return nil, fmt.Errorf("add schema resource %s: %w", resourceRel, err)
		}
		seen[resourceRel] = struct{}{}
	}

	schema, err := compiler.Compile(schemaURL(normalizedRel))
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", rel, err)
	}

	return schema, nil
}

func ValidateBytes(rel string, data []byte) error {
	schema, err := Compile(rel)
	if err != nil {
		return err
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode json for %s: %w", rel, err)
	}

	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate %s: %w", rel, err)
	}

	return nil
}

func schemaURL(rel string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   "wrkr.dev",
		Path:   "/schemas/v1/" + rel,
	}).String()
}

func normalizeSchemaRel(rel string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(rel))
	switch {
	case clean == ".":
		return "", fmt.Errorf("invalid schema path: %q", rel)
	case strings.HasPrefix(clean, "/"):
		return "", fmt.Errorf("invalid schema path: %q", rel)
	case clean == "..":
		return "", fmt.Errorf("invalid schema path: %q", rel)
	case strings.HasPrefix(clean, "../"):
		return "", fmt.Errorf("invalid schema path: %q", rel)
	case strings.Contains(clean, "/../"):
		return "", fmt.Errorf("invalid schema path: %q", rel)
	}
	return clean, nil
}

func readSchemaBytes(rel string) ([]byte, error) {
	explicitRoot := strings.TrimSpace(os.Getenv("WRKR_SCHEMA_ROOT"))
	if explicitRoot != "" {
		root, err := os.OpenRoot(explicitRoot)
		if err != nil {
			return nil, fmt.Errorf("open schema root: %w", err)
		}
		defer func() { _ = root.Close() }()

		f, err := root.Open(filepath.FromSlash(rel))
		if err != nil {
			p := filepath.Join(explicitRoot, filepath.FromSlash(rel))
			return nil, fmt.Errorf("schema not found: %s: %w", p, err)
		}
		defer func() { _ = f.Close() }()

		payload, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", rel, err)
		}
		return payload, nil
	}

	embeddedPath := path.Join("v1", rel)
	payload, err := schemas.V1FS.ReadFile(embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s: %w", embeddedPath, err)
	}
	return payload, nil
}

func ValidateValue(rel string, value any) error {
	schema, err := Compile(rel)
	if err != nil {
		return err
	}

	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate %s: %w", rel, err)
	}

	return nil
}
