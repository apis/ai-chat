package main

import (
	"ai-chat/internal/pkg/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const applicationName = "calculator-mcp"
const serverShutdownTimeout = 5 * time.Second

// CalculationRequest represents the request structure for calculator operations
type CalculationRequest struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

// CalculationResponse represents the response structure from calculator operations
type CalculationResponse struct {
	Result float64 `json:"result"`
	Error  string  `json:"error,omitempty"`
}

// MCPRequest represents an MCP request
type MCPRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	setupZerolog()

	log.Info().Msg("Parsing configuration")
	appConfig := &applicationConfig{}
	config.Parse(appConfig, applicationName)

	log.Info().Msg("Starting up calculator MCP service")

	listener := createNetListener(appConfig)
	server := startHttpServer(listener, appConfig)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done

	log.Info().Msg("Application stopping")

	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server.Shutdown failed")
	}

	time.Sleep(time.Second * 1) // Give some time for graceful shutdown
	log.Info().Msg("Application stopped")
}

func startHttpServer(listener net.Listener, appConfig *applicationConfig) *http.Server {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	httpLogger := httplog.NewLogger("calculator-mcp", httplog.Options{
		LogLevel: slog.LevelDebug,
		JSON:     true,
		Concise:  true,
	})

	router := chi.NewRouter()
	router.Use(httplog.RequestLogger(httpLogger))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://*",
			"http://*",
		},
		AllowedMethods: []string{"POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	// MCP endpoint
	router.Post("/", handleMCPRequest(appConfig))

	// Root endpoint for service info (GET)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "Calculator MCP API",
			"version": "1.0.0",
			"methods": "calculator.add, calculator.subtract, calculator.multiply, calculator.divide",
		})
	})

	server := &http.Server{
		Handler: router,
	}

	go func() {
		log.Info().Msg("Server is about to start")

		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server.Serve failed")
		}

		log.Info().Msg("Server stopped")
	}()
	return server
}

func createNetListener(appConfig *applicationConfig) net.Listener {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", appConfig.Host, appConfig.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("net.Listen failed")
	}

	log.Info().Str("address", listener.Addr().String()).Msg("Server listening")
	return listener
}

func handleMCPRequest(appConfig *applicationConfig) http.HandlerFunc {
	calculatorBaseURL := fmt.Sprintf("http://%s:%d/api/calculator",
		appConfig.CalculatorHost, appConfig.CalculatorPort)

	return func(w http.ResponseWriter, r *http.Request) {
		var req MCPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendMCPErrorResponse(w, &req, -32700, "Parse error", err.Error())
			return
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			sendMCPErrorResponse(w, &req, -32600, "Invalid Request", "Invalid JSON-RPC version")
			return
		}

		// Handle method
		var result interface{}
		var err error

		switch req.Method {
		case "calculator.add":
			result, err = handleCalculatorOperation(calculatorBaseURL+"/add", req.Params)
		case "calculator.subtract":
			result, err = handleCalculatorOperation(calculatorBaseURL+"/subtract", req.Params)
		case "calculator.multiply":
			result, err = handleCalculatorOperation(calculatorBaseURL+"/multiply", req.Params)
		case "calculator.divide":
			// Check for division by zero
			if len(req.Params) == 2 {
				if b, ok := req.Params[1].(float64); ok && b == 0 {
					sendMCPErrorResponse(w, &req, -32000, "Division by zero", "Division by zero is not allowed")
					return
				}
			}
			result, err = handleCalculatorOperation(calculatorBaseURL+"/divide", req.Params)
		default:
			sendMCPErrorResponse(w, &req, -32601, "Method not found", fmt.Sprintf("Method '%s' not found", req.Method))
			return
		}

		if err != nil {
			sendMCPErrorResponse(w, &req, -32000, "Server error", err.Error())
			return
		}

		// Send success response
		sendMCPSuccessResponse(w, &req, result)
	}
}

func handleCalculatorOperation(url string, params []interface{}) (float64, error) {
	if len(params) != 2 {
		return 0, fmt.Errorf("expected 2 parameters, got %d", len(params))
	}

	a, ok := params[0].(float64)
	if !ok {
		return 0, fmt.Errorf("first parameter must be a number")
	}

	b, ok := params[1].(float64)
	if !ok {
		return 0, fmt.Errorf("second parameter must be a number")
	}

	return callCalculatorAPI(url, a, b)
}

func callCalculatorAPI(url string, a, b float64) (float64, error) {
	// Create request payload
	reqBody, err := json.Marshal(CalculationRequest{
		A: a,
		B: b,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP client and send request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var calcResp CalculationResponse
	if err := json.NewDecoder(resp.Body).Decode(&calcResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for error in response
	if calcResp.Error != "" {
		return 0, fmt.Errorf("calculator service error: %s", calcResp.Error)
	}

	return calcResp.Result, nil
}

func sendMCPSuccessResponse(w http.ResponseWriter, req *MCPRequest, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

func sendMCPErrorResponse(w http.ResponseWriter, req *MCPRequest, code int, message, data string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Always return 200 OK for JSON-RPC

	var id interface{}
	if req != nil {
		id = req.ID
	}

	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	})
}

func setupZerolog() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger()
}
