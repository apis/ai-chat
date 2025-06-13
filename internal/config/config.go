package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// MCPServerConfig represents configuration for an MCP server
type MCPServerConfig struct {
	Command       string   `json:"command,omitempty"`
	Args          []string `json:"args,omitempty"`
	URL           string   `json:"url,omitempty"`
	Headers       []string `json:"headers,omitempty"`
	AllowedTools  []string `json:"allowedTools,omitempty"`
	ExcludedTools []string `json:"excludedTools,omitempty"`
}

// Config represents the application configuration
type Config struct {
	MCPServers      map[string]MCPServerConfig `json:"mcpServers" yaml:"mcpServers"`
	Model           string                     `json:"model,omitempty" yaml:"model,omitempty"`
	MaxSteps        int                        `json:"max-steps,omitempty" yaml:"max-steps,omitempty"`
	MessageWindow   int                        `json:"message-window,omitempty" yaml:"message-window,omitempty"`
	Debug           bool                       `json:"debug,omitempty" yaml:"debug,omitempty"`
	SystemPrompt    string                     `json:"system-prompt,omitempty" yaml:"system-prompt,omitempty"`
	OpenAIAPIKey    string                     `json:"openai-api-key,omitempty" yaml:"openai-api-key,omitempty"`
	AnthropicAPIKey string                     `json:"anthropic-api-key,omitempty" yaml:"anthropic-api-key,omitempty"`
	GoogleAPIKey    string                     `json:"google-api-key,omitempty" yaml:"google-api-key,omitempty"`
	OpenAIURL       string                     `json:"openai-url,omitempty" yaml:"openai-url,omitempty"`
	AnthropicURL    string                     `json:"anthropic-url,omitempty" yaml:"anthropic-url,omitempty"`
	Prompt          string                     `json:"prompt,omitempty" yaml:"prompt,omitempty"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	for serverName, serverConfig := range c.MCPServers {
		if len(serverConfig.AllowedTools) > 0 && len(serverConfig.ExcludedTools) > 0 {
			return fmt.Errorf("server %s: allowedTools and excludedTools are mutually exclusive", serverName)
		}
	}
	return nil
}

// SystemPromptConfig represents system prompt configuration
type SystemPromptConfig struct {
	SystemPrompt string `json:"systemPrompt"`
}

// LoadMCPConfig loads MCP configuration from file
func LoadMCPConfig(configFile string) (*Config, error) {
	v := viper.New()

	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error getting home directory: %v", err)
		}

		// Try .mcphost files first (new format), then .mcp files (backwards compatibility)
		configNames := []string{".mcphost", ".mcp"}
		configTypes := []string{"yaml", "json"}

		var configFound bool
		for _, configName := range configNames {
			for _, configType := range configTypes {
				v.SetConfigName(configName)
				v.SetConfigType(configType)
				v.AddConfigPath(homeDir)

				if err := v.ReadInConfig(); err == nil {
					configFound = true
					break
				}
			}
			if configFound {
				break
			}
		}

		if !configFound {
			// Create default config file
			if err := createDefaultConfig(homeDir); err != nil {
				// If we can't create the file, just return default config
				return &Config{
					MCPServers: make(map[string]MCPServerConfig),
				}, nil
			}

			// Try to load the newly created config
			v.SetConfigName(".mcphost")
			v.SetConfigType("yaml")
			v.AddConfigPath(homeDir)
			if err := v.ReadInConfig(); err != nil {
				// If we still can't read it, return default config
				return &Config{
					MCPServers: make(map[string]MCPServerConfig),
				}, nil
			}
		}
	} else {
		// Use specified config file
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Validate that allowedTools and excludedTools are mutually exclusive
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadSystemPrompt loads system prompt from file or returns the string directly
func LoadSystemPrompt(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// Check if input is a file that exists
	if _, err := os.Stat(input); err == nil {
		// Treat as file path
		v := viper.New()
		v.SetConfigFile(input)

		if err := v.ReadInConfig(); err != nil {
			return "", fmt.Errorf("error reading system prompt file: %v", err)
		}

		systemPrompt := v.GetString("systemPrompt")
		if systemPrompt == "" {
			return "", fmt.Errorf("systemPrompt field not found in config file")
		}

		return systemPrompt, nil
	}

	// Treat as direct string
	return input, nil
}

// createDefaultConfig creates a default .mcphost.yml file in the user's home directory
func createDefaultConfig(homeDir string) error {
	configPath := filepath.Join(homeDir, ".mcphost.yml")

	// Create the file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer file.Close()

	// Write a clean YAML template
	content := `# MCPHost Configuration File
# All command-line flags can be configured here

# MCP Servers configuration
# Add your MCP servers here
# Example:
# mcpServers:
#   filesystem:
#     command: npx
#     args: ["@modelcontextprotocol/server-filesystem", "/path/to/allowed/files"]
#   sqlite:
#     command: uvx
#     args: ["mcp-server-sqlite", "--db-path", "/tmp/example.db"]

mcpServers:

# Application settings (all optional)
# model: "anthropic:claude-sonnet-4-20250514"  # Default model to use
# max-steps: 20                                # Maximum agent steps (0 for unlimited)
# message-window: 40                           # Number of messages to keep in context
# debug: false                                 # Enable debug logging
# system-prompt: "/path/to/system-prompt.json" # System prompt file

# API Configuration (can also use environment variables)
# openai-api-key: "your-openai-key"
# anthropic-api-key: "your-anthropic-key"  
# google-api-key: "your-google-key"
# openai-url: "https://api.openai.com/v1"
# anthropic-url: "https://api.anthropic.com"
`

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing config content: %v", err)
	}

	return nil
}
