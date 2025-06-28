package config

import (
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// Config captures user-level settings stored in .vyb/config.yaml.
//
// The initial schema is intentionally minimal; new fields can be added
// without breaking forwards-compatibility as callers should always
// access configuration via the exported struct rather than raw maps.
//
// Example YAML:
//
//	provider: openai
//
// Zero-value Config is invalid – use Default() when no config file is
// found.
//
// NOTE: keep field tags in sync with YAML when extending this struct.
//
//	Use explicit field names so unknown keys are rejected when the
//	file *is* present (surfacing typo errors early).
//
//nolint:revive // field name is intentionally simple
type Config struct {
	Provider string `yaml:"provider"`
	Logging  `yaml:"logging"`
}

// Logging captures logging-specific settings.
type Logging struct {
	Level                string `yaml:"level"`
	RequestResponseDebug bool   `yaml:"request-response-debug"`
}

// defaultProvider is used when no configuration file exists or it cannot
// be parsed.  The value must always map to a known provider in the llm
// dispatcher.
const defaultProvider = "openai"

// Default returns a Config populated with hard-coded defaults. It should
// be used whenever .vyb/config.yaml is missing.
func Default() *Config {
	return &Config{
		Provider: defaultProvider,
		Logging: Logging{
			Level:                "info",
			RequestResponseDebug: false,
		},
	}
}

// Load reads .vyb/config.yaml located under projectRoot. When the file
// does not exist the function returns Default() with a nil error so the
// caller can proceed transparently. Any other I/O or unmarshalling error
// is propagated.
func Load(projectRoot string) (*Config, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("projectRoot must not be empty")
	}
	return LoadFS(os.DirFS(projectRoot))
}

// LoadFS performs the same operation as Load but works directly on an
// fs.FS. This facilitates unit-testing with fstest.MapFS.
func LoadFS(fsys fs.FS) (*Config, error) {
	const relPath = ".vyb/config.yaml"

	data, err := fs.ReadFile(fsys, relPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file – fall back to defaults.
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", relPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", relPath, err)
	}

	// Basic sanity check – default when Provider is empty.
	if cfg.Provider == "" {
		cfg.Provider = defaultProvider
	}
	return &cfg, nil
}