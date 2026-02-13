package accept

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davidahmann/wrkr/core/accept/checks"
	"github.com/davidahmann/wrkr/core/fsx"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "accept.yaml"
	configSchemaID    = "wrkr.accept_config"
	configSchemaVer   = "v1"
)

type Config struct {
	SchemaID          string    `yaml:"schema_id"`
	SchemaVersion     string    `yaml:"schema_version"`
	RequiredArtifacts []string  `yaml:"required_artifacts"`
	TestCommand       string    `yaml:"test_command"`
	LintCommand       string    `yaml:"lint_command"`
	PathRules         PathRules `yaml:"path_rules"`
}

type PathRules struct {
	MaxArtifactPaths  int      `yaml:"max_artifact_paths"`
	ForbiddenPrefixes []string `yaml:"forbidden_prefixes"`
	AllowedPrefixes   []string `yaml:"allowed_prefixes"`
}

func DefaultConfig() Config {
	cfg := Config{
		SchemaID:          configSchemaID,
		SchemaVersion:     configSchemaVer,
		RequiredArtifacts: []string{},
		TestCommand:       "go test ./...",
		LintCommand:       "go vet ./...",
		PathRules: PathRules{
			MaxArtifactPaths:  0,
			ForbiddenPrefixes: []string{},
			AllowedPrefixes:   []string{},
		},
	}
	cfg.normalize()
	return cfg
}

func InitConfig(path string, force bool) (string, error) {
	resolved, err := resolveConfigPath(path)
	if err != nil {
		return "", err
	}
	if !force {
		if _, err := os.Stat(resolved); err == nil {
			return "", fmt.Errorf("config already exists: %s", resolved)
		}
	}

	cfg := DefaultConfig()
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config yaml: %w", err)
	}
	if err := fsx.AtomicWriteFile(resolved, raw, 0o600); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return resolved, nil
}

func LoadConfig(path string) (Config, error) {
	resolved, err := resolveConfigPath(path)
	if err != nil {
		return Config{}, err
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return Config{}, err
	}
	defer func() { _ = root.Close() }()
	raw, err := root.ReadFile(filepath.Base(resolved))
	if err != nil {
		return Config{}, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config yaml: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

func LoadConfigOrDefault(path string) (Config, string, error) {
	resolved, err := resolveConfigPath(path)
	if err != nil {
		return Config{}, "", err
	}
	cfg, err := LoadConfig(resolved)
	if err == nil {
		return cfg, resolved, nil
	}
	if os.IsNotExist(err) {
		return DefaultConfig(), resolved, nil
	}
	return Config{}, "", err
}

func (c Config) ToChecksConfig() checks.Config {
	return checks.Config{
		RequiredArtifacts: append([]string(nil), c.RequiredArtifacts...),
		TestCommand:       c.TestCommand,
		LintCommand:       c.LintCommand,
		PathRules: checks.PathRules{
			MaxArtifactPaths:  c.PathRules.MaxArtifactPaths,
			ForbiddenPrefixes: append([]string(nil), c.PathRules.ForbiddenPrefixes...),
			AllowedPrefixes:   append([]string(nil), c.PathRules.AllowedPrefixes...),
		},
	}
}

func resolveConfigPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = DefaultConfigPath
	}
	return fsx.ResolveWithinWorkingDir(filepath.Clean(trimmed))
}

func (c *Config) normalize() {
	if strings.TrimSpace(c.SchemaID) == "" {
		c.SchemaID = configSchemaID
	}
	if strings.TrimSpace(c.SchemaVersion) == "" {
		c.SchemaVersion = configSchemaVer
	}
	c.RequiredArtifacts = normalizedList(c.RequiredArtifacts)
	c.TestCommand = strings.TrimSpace(c.TestCommand)
	c.LintCommand = strings.TrimSpace(c.LintCommand)
	if c.PathRules.MaxArtifactPaths < 0 {
		c.PathRules.MaxArtifactPaths = 0
	}
	c.PathRules.ForbiddenPrefixes = normalizedList(c.PathRules.ForbiddenPrefixes)
	c.PathRules.AllowedPrefixes = normalizedList(c.PathRules.AllowedPrefixes)
}

func normalizedList(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
