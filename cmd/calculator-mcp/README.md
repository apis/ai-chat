# Calculator MCP Server

This is an implementation of a Model Context Protocol (MCP) server for calculator operations. It provides a set of tools for performing basic arithmetic operations.

## Overview

The Calculator MCP Server is built using the `github.com/mark3labs/mcp-go` library. It implements a stdio server that reads from stdin and writes to stdout, making it suitable for integration with LLM-powered applications that support the MCP protocol.

## Tools

The following tools are available:

- `calculator.add(a, b)` - Adds two numbers
- `calculator.subtract(a, b)` - Subtracts b from a
- `calculator.multiply(a, b)` - Multiplies two numbers
- `calculator.divide(a, b)` - Divides a by b (returns error if b is zero)

## Usage

### Building the server

```bash
# Build the server
go build -o calculator-mcp cmd/calculator-mcp/main.go

# Run the server
./calculator-mcp
```

### Integration

This MCP server is designed to be used with LLM-powered applications that support the MCP protocol. The server communicates through stdin and stdout, so it should be launched as a child process by the application.

## Implementation Details

The server is implemented using the `github.com/mark3labs/mcp-go` library. It defines four tools for basic arithmetic operations and handles the requests directly within the server.

Each tool accepts two numeric parameters (`a` and `b`) and returns the result of the operation as a formatted string. The divide tool includes a check for division by zero and returns an appropriate error message if the divisor is zero.

The server uses the `ServeStdio` function from the `github.com/mark3labs/mcp-go/server` package to handle communication through stdin and stdout.
