package main

type applicationConfig struct {
	// MCP Server configuration
	Host string `config_default:"localhost" config_description:"MCP server host interface"`
	Port int    `config_default:"8082" config_description:"MCP server port"`

	// Calculator service client configuration
	CalculatorHost string `config_default:"localhost" config_description:"Calculator service host"`
	CalculatorPort int    `config_default:"8081" config_description:"Calculator service port"`
}
