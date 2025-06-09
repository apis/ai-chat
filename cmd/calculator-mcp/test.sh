#!/bin/bash

# Test script for the Calculator MCP server
# This script sends MCP requests to the server and displays the responses

HOST="localhost"
PORT="8082"
URL="http://$HOST:$PORT/"

echo "Testing Calculator MCP Server at $URL"
echo "-------------------------------------"

# Function to send an MCP request and display the response
send_request() {
  local method=$1
  local a=$2
  local b=$3
  local description=$4

  echo "Test: $description"
  echo "Request: $method($a, $b)"
  
  response=$(curl -s -X POST -H "Content-Type: application/json" -d "{
    \"jsonrpc\": \"2.0\",
    \"method\": \"$method\",
    \"params\": [$a, $b],
    \"id\": 1
  }" $URL)
  
  echo "Response: $response"
  echo "-------------------------------------"
}

# Test addition
send_request "calculator.add" 5 3 "Addition: 5 + 3"

# Test subtraction
send_request "calculator.subtract" 10 4 "Subtraction: 10 - 4"

# Test multiplication
send_request "calculator.multiply" 6 7 "Multiplication: 6 * 7"

# Test division
send_request "calculator.divide" 20 5 "Division: 20 / 5"

# Test division by zero (should return an error)
send_request "calculator.divide" 10 0 "Division by zero: 10 / 0"

# Test invalid method
send_request "calculator.unknown" 1 2 "Invalid method: calculator.unknown"

# Test invalid parameters (string instead of number)
curl -s -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "calculator.add",
  "params": ["not a number", 3],
  "id": 1
}' $URL | jq .

echo "All tests completed."