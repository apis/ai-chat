package main

type applicationConfig struct {
	Host               string `config_default:"localhost" config_description:"Server host interface"`
	Port               int    `config_default:"8080" config_description:"Server port"`
	SimulatedDelay     int    `config_default:"0" config_description:"Simulated delay for HTMX interactions in milliseconds"`
	CalculatorMcpUrl string `config_default:"http://localhost:8081/tool/" config_description:"URL for the Calculator MCP service"`
}
