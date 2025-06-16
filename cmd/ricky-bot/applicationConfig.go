package main

type applicationConfig struct {
	Host           string `config_default:"localhost" config_description:"Server host interface"`
	Port           int    `config_default:"8080" config_description:"Server port"`
	SimulatedDelay int    `config_default:"0" config_description:"Simulated delay for HTMX interactions in milliseconds"`
	McpConfigFile  string `config_default:"./configs/mcp.config.json" config_description:"Path to MCP configuration file"`
	ModelName      string `config_default:"ollama:qwen3:8b" config_description:"Model to use for chat"`
	SystemPrompt   string `config_default:"You are a helpful AI assistant." config_description:"System prompt for the model"`
	MaxSteps       int    `config_default:"20" config_description:"Maximum number of steps for the agent"`
	MessageWindow  int    `config_default:"10" config_description:"Maximum number of messages to keep in history"`
}
