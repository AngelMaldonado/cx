package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RequiredMCPServers lists the MCP servers that CX configures.
var RequiredMCPServers = []string{"context7", "linear"}

// WriteMCPConfigs generates MCP server configuration for each selected agent tool.
func WriteMCPConfigs(rootDir string, agentSlugs []string) error {
	for _, slug := range agentSlugs {
		switch slug {
		case "claude":
			if err := writeClaudeMCP(rootDir); err != nil {
				return fmt.Errorf("claude MCP: %w", err)
			}
		case "gemini":
			if err := writeGeminiMCP(rootDir); err != nil {
				return fmt.Errorf("gemini MCP: %w", err)
			}
		case "codex":
			if err := writeCodexMCP(rootDir); err != nil {
				return fmt.Errorf("codex MCP: %w", err)
			}
		}
	}
	return nil
}

// writeClaudeMCP writes .mcp.json with context7 + linear servers.
// Merges into existing file if present.
func writeClaudeMCP(rootDir string) error {
	mcpPath := filepath.Join(rootDir, ".mcp.json")

	config := make(map[string]interface{})
	if data, err := os.ReadFile(mcpPath); err == nil {
		json.Unmarshal(data, &config)
	}

	servers, _ := config["mcpServers"].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}

	// Context7: stdio — subprocess inherits CONTEXT7_API_KEY from shell if set
	servers["context7"] = map[string]interface{}{
		"command": "npx",
		"args":    []string{"-y", "@upstash/context7-mcp"},
	}

	// Linear: HTTP with OAuth — authenticate via /mcp in Claude Code
	servers["linear"] = map[string]interface{}{
		"type": "http",
		"url":  "https://mcp.linear.app/mcp",
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteMCP(mcpPath, append(data, '\n'))
}

// writeGeminiMCP writes mcpServers into .gemini/settings.json.
// Merges into existing file if present.
func writeGeminiMCP(rootDir string) error {
	settingsPath := filepath.Join(rootDir, ".gemini", "settings.json")

	config := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &config)
	}

	servers, _ := config["mcpServers"].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}

	// Context7: stdio — subprocess inherits CONTEXT7_API_KEY from shell if set
	servers["context7"] = map[string]interface{}{
		"command": "npx",
		"args":    []string{"-y", "@upstash/context7-mcp"},
	}

	// Linear: HTTP with OAuth — authenticate via /mcp in Gemini CLI
	servers["linear"] = map[string]interface{}{
		"url":  "https://mcp.linear.app/mcp",
		"type": "http",
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteMCP(settingsPath, append(data, '\n'))
}

// writeCodexMCP appends MCP server sections to .codex/config.toml.
// Skips if MCP sections already exist.
func writeCodexMCP(rootDir string) error {
	configPath := filepath.Join(rootDir, ".codex", "config.toml")

	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), "[mcp_servers") {
			return nil
		}
	}

	mcpSection := `
# MCP Servers
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]

[mcp_servers.linear]
url = "https://mcp.linear.app/mcp"
# Authenticate via: codex mcp login linear
`

	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(mcpSection)
	return err
}

// CheckMCP checks .mcp.json for required MCP servers.
func CheckMCP(rootDir string) (hasMCPFile bool, missingServers []string, err error) {
	mcpPath := filepath.Join(rootDir, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, RequiredMCPServers, nil
		}
		return false, nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return true, nil, nil
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return true, RequiredMCPServers, nil
	}

	for _, name := range RequiredMCPServers {
		if _, has := servers[name]; !has {
			missingServers = append(missingServers, name)
		}
	}

	return true, missingServers, nil
}

func atomicWriteMCP(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
