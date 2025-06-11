package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// McphostConfig holds the server configuration
type McphostConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// CalculationRequest represents the expected JSON input for calculations
type CalculationRequest struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

// CalculationResponse represents the JSON output for successful calculations
type CalculationResponse struct {
	Result float64 `json:"result"`
}

// ErrorResponse represents the JSON output for errors
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Load configuration
	configFile, err := os.Open("cmd/calculator-mcp/mcphost.config.json")
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer configFile.Close()

	var config McphostConfig
	decoder := json.NewDecoder(configFile)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to decode config: %v", err)
	}

	// Create a new HTTP ServeMux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/tool/calculator.add", addHandler)
	mux.HandleFunc("/tool/calculator.subtract", subtractHandler)
	mux.HandleFunc("/tool/calculator.multiply", multiplyHandler)
	mux.HandleFunc("/tool/calculator.divide", divideHandler)

	// Start the HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON request body"})
		return
	}

	result := req.A + req.B
	writeJSONResponse(w, http.StatusOK, CalculationResponse{Result: result})
}

func subtractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON request body"})
		return
	}

	result := req.A - req.B
	writeJSONResponse(w, http.StatusOK, CalculationResponse{Result: result})
}

func multiplyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON request body"})
		return
	}

	result := req.A * req.B
	writeJSONResponse(w, http.StatusOK, CalculationResponse{Result: result})
}

func divideHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON request body"})
		return
	}

	if req.B == 0 {
		writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{Error: "Division by zero is not allowed"})
		return
	}

	result := req.A / req.B
	writeJSONResponse(w, http.StatusOK, CalculationResponse{Result: result})
}
