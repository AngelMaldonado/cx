package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func CheckMCP(rootDir string) (hasMCPFile bool, missingServers []string, err error) {
	mcpPath := filepath.Join(rootDir, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, []string{"linear"}, nil
		}
		return false, nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return true, nil, nil // file exists but can't parse — don't report missing servers
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return true, []string{"linear"}, nil
	}

	if _, hasLinear := mcpServers["linear"]; !hasLinear {
		missingServers = append(missingServers, "linear")
	}

	return true, missingServers, nil
}

func LinearMCPSnippet() string {
	return `{
  "mcpServers": {
    "linear": {
      "command": "npx",
      "args": ["-y", "@anthropic/linear-mcp-server"],
      "env": {
        "LINEAR_API_KEY": "<your-api-key>"
      }
    }
  }
}`
}
