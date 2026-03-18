package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Schema  string              `yaml:"schema"`
	Context string              `yaml:"context"`
	Rules   map[string][]string `yaml:"rules"`
}

// Load reads .cx/cx.yaml from rootDir. Returns zero-value Config if absent.
func Load(rootDir string) (*Config, error) {
	path := filepath.Join(rootDir, ".cx", "cx.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading cx.yaml: %w", err)
	}

	// First unmarshal into a map to check for unknown keys
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing cx.yaml: %w", err)
	}

	known := map[string]bool{"schema": true, "context": true, "rules": true}
	for key := range raw {
		if !known[key] {
			return nil, fmt.Errorf("cx.yaml: unrecognized key %q", key)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing cx.yaml: %w", err)
	}

	return &cfg, nil
}
