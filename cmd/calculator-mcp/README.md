# Calculator MCP Server

This is an implementation of an MCP (Message Call Protocol) server for the Calculator service. It provides a JSON-RPC 2.0 compatible interface to the Calculator API.

## Overview

The Calculator MCP Server acts as a bridge between clients using the MCP protocol and the Calculator service's REST API. It translates MCP method calls into HTTP requests to the Calculator service and returns the results in the MCP format.

## Methods

The following methods are available:

- `calculator.add(a, b)` - Adds two numbers
- `calculator.subtract(a, b)` - Subtracts b from a
- `calculator.multiply(a, b)` - Multiplies two numbers
- `calculator.divide(a, b)` - Divides a by b (returns error if b is zero)

## Configuration

The server can be configured using environment variables or a configuration file:

- `CALCULATOR_MCP_HOST` - Host interface to bind to (default: "localhost")
- `CALCULATOR_MCP_PORT` - Port to listen on (default: 8082)
- `CALCULATOR_MCP_CALCULATORHOST` - Calculator service host (default: "localhost")
- `CALCULATOR_MCP_CALCULATORPORT` - Calculator service port (default: 8081)

## Usage

### Running the server

```bash
# Build the server
go build -o calculator-mcp cmd/calculator-mcp/main.go cmd/calculator-mcp/applicationConfig.go

# Run the server
./calculator-mcp
```

### Example requests

Here's an example of how to call the add method:

```bash
curl -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "calculator.add",
  "params": [5, 3],
  "id": 1
}' http://localhost:8082/
```

Response:

```json
{
  "jsonrpc": "2.0",
  "result": 8,
  "id": 1
}
```

### Testing

A test script is provided to test all the methods:

```bash
# Make the script executable
chmod +x cmd/calculator-mcp/test.sh

# Run the tests
./cmd/calculator-mcp/test.sh
```

## Implementation Details

The server is implemented using Go's standard HTTP server and the Chi router. It follows the JSON-RPC 2.0 specification for request and response formats, error handling, and method invocation.

The server communicates with the Calculator service using HTTP requests to its REST API endpoints.