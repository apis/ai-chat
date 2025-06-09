package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Calculator 🧮",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add add tool
	addTool := mcp.NewTool("calculator.add",
		mcp.WithDescription("Add two numbers"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add subtract tool
	subtractTool := mcp.NewTool("calculator.subtract",
		mcp.WithDescription("Subtract second number from first number"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add multiply tool
	multiplyTool := mcp.NewTool("calculator.multiply",
		mcp.WithDescription("Multiply two numbers"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add divide tool
	divideTool := mcp.NewTool("calculator.divide",
		mcp.WithDescription("Divide first number by second number"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number (dividend)"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number (divisor)"),
		),
	)

	// Add tool handlers
	s.AddTool(addTool, addHandler)
	s.AddTool(subtractTool, subtractHandler)
	s.AddTool(multiplyTool, multiplyHandler)
	s.AddTool(divideTool, divideHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func addHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, err := request.RequireFloat("a")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, err := request.RequireFloat("b")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := a + b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}

func subtractHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, err := request.RequireFloat("a")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, err := request.RequireFloat("b")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := a - b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}

func multiplyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, err := request.RequireFloat("a")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, err := request.RequireFloat("b")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := a * b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}

func divideHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, err := request.RequireFloat("a")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, err := request.RequireFloat("b")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if b == 0 {
		return mcp.NewToolResultError("Division by zero is not allowed"), nil
	}

	result := a / b
	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}
