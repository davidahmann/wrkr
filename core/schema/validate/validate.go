package validate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

func Compile(rel string) (*jsonschema.Schema, error) {
	schemaPath, err := SchemaPath(rel)
	if err != nil {
		return nil, err
	}

	compiler := jsonschema.NewCompiler()
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(schemaPath)}).String()
	schema, err := compiler.Compile(uri)
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
