package main

import (
	"ai-chat/internal/pkg/config"
	"context"
	"encoding/json"
	"errors"
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

const applicationName = "calculator"
const serverShutdownTimeout = 5 * time.Second

func main() {
	setupZerolog()

	log.Info().Msg("Parsing configuration")
	appConfig := &applicationConfig{}
	config.Parse(appConfig, applicationName)

	log.Info().Msg("Starting up calculator service")

	listener := createNetListener(appConfig)
	server := startHttpServer(listener)

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

func startHttpServer(listener net.Listener) *http.Server {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	httpLogger := httplog.NewLogger("calculator-api", httplog.Options{
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
	}))

	// Calculator API endpoints
	router.Route("/api/calculator", func(r chi.Router) {
		r.Post("/add", handleAdd)
		r.Post("/subtract", handleSubtract)
		r.Post("/multiply", handleMultiply)
		r.Post("/divide", handleDivide)
	})

	// Root endpoint for service info
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "Calculator API",
			"version": "1.0.0",
			"endpoints": "/api/calculator/add, /api/calculator/subtract, " +
				"/api/calculator/multiply, /api/calculator/divide",
		})
	})

	server := &http.Server{
		Handler: router,
	}

	go func() {
		log.Info().Msg("Server is about to start")

		err := server.Serve(listener)
		if err != nil {
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal().Err(err).Msg("server.ListenAndServe failed")
			}
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

func setupZerolog() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger()
}

// Request and response structures
type CalculationRequest struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type CalculationResponse struct {
	Result float64 `json:"result"`
	Error  string  `json:"error,omitempty"`
}

// Handler functions
func handleAdd(w http.ResponseWriter, r *http.Request) {
	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	result := req.A + req.B
	sendSuccessResponse(w, result)
}

func handleSubtract(w http.ResponseWriter, r *http.Request) {
	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	result := req.A - req.B
	sendSuccessResponse(w, result)
}

func handleMultiply(w http.ResponseWriter, r *http.Request) {
	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	result := req.A * req.B
	sendSuccessResponse(w, result)
}

func handleDivide(w http.ResponseWriter, r *http.Request) {
	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.B == 0 {
		sendErrorResponse(w, "Division by zero is not allowed", http.StatusBadRequest)
		return
	}

	result := req.A / req.B
	sendSuccessResponse(w, result)
}

func sendSuccessResponse(w http.ResponseWriter, result float64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CalculationResponse{
		Result: result,
	})
}

func sendErrorResponse(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(CalculationResponse{
		Error: errMsg,
	})
}
