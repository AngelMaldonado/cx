package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Credentials stores API keys for MCP server integrations.
type Credentials struct {
	Context7APIKey string `json:"context7_api_key,omitempty"`
	LinearAPIKey   string `json:"linear_api_key,omitempty"`
}

// LoadCredentials reads stored credentials from ~/.cx/credentials.json.
// Returns an empty Credentials struct if the file doesn't exist or can't be parsed.
func LoadCredentials() *Credentials {
	dir, err := GlobalCXDir()
	if err != nil {
		return &Credentials{}
	}
	data, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		return &Credentials{}
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return &Credentials{}
	}
	return &creds
}

// SaveCredentials writes credentials to ~/.cx/credentials.json with restricted permissions.
func SaveCredentials(creds *Credentials) error {
	dir, err := GlobalCXDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "credentials.json"), append(data, '\n'), 0o600)
}

// WriteEnvFile writes a shell-sourceable env file to ~/.cx/env.
func WriteEnvFile(creds *Credentials) error {
	dir, err := GlobalCXDir()
	if err != nil {
		return err
	}
	var lines []string
	lines = append(lines, "# CX Framework — API keys")
	lines = append(lines, "# Add to your shell profile: source ~/.cx/env")
	if creds.Context7APIKey != "" {
		lines = append(lines, fmt.Sprintf("export CONTEXT7_API_KEY=%q", creds.Context7APIKey))
	}
	if creds.LinearAPIKey != "" {
		lines = append(lines, fmt.Sprintf("export LINEAR_API_KEY=%q", creds.LinearAPIKey))
	}
	lines = append(lines, "")
	return os.WriteFile(filepath.Join(dir, "env"), []byte(strings.Join(lines, "\n")), 0o600)
}

// ResolveKey checks the environment variable first, then falls back to the stored value.
func ResolveKey(envName, stored string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return stored
}
