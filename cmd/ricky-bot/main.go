package main

import (
	"ai-chat/internal/agent"
	internalConfig "ai-chat/internal/config"
	"ai-chat/internal/models"
	"ai-chat/internal/pkg/config"
	"ai-chat/internal/pkg/httpHandlers"
	"ai-chat/internal/pkg/sessions"
	"ai-chat/internal/pkg/staticAssets"
	"ai-chat/internal/pkg/web"
	"ai-chat/internal/pkg/websocketServer"
	webAssets "ai-chat/web"
	"context"
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

// Windows configuration examples
// cmd /V /C "set HTMX_APP_PORT=321&& ricky-bot.exe"
// ricky-bot.exe --Port 123

// Linux configuration examples
// HTMX_APP_PORT=321 ./ricky-bot
// ./ricky-bot --Port 123

const applicationName = "ricky-bot"
const serverShutdownTimeout = 5 * time.Second
const embedFsRoot = "frontend/dist"
const templatesDir = "templates"
const uiUrlPrefix = "/chat"
const defaultUiUrl = "index.html"

func main() {
	setupZerolog()

	log.Info().Msg("Parsing configuration")
	appConfig := &applicationConfig{}
	config.Parse(appConfig, applicationName)

	log.Info().Msg("Starting up")

	templates, err := web.TemplateParseFSRecursive(webAssets.TemplateFS, templatesDir, ".gohtml", nil)
	if err != nil {
		log.Panic().Err(err).Msg("template parsing failed")
	}

	// Load MCP configuration
	log.Info().Msg("Loading MCP configuration")
	mcpConfig, err := internalConfig.LoadMCPConfig(appConfig.McpConfigFile)
	if err != nil {
		log.Panic().Err(err).Msg("failed to load MCP configuration")
	}

	// Create model configuration
	modelConfig := &models.ProviderConfig{
		ModelString:  appConfig.ModelName,
		SystemPrompt: appConfig.SystemPrompt,
	}

	// Create agent configuration
	agentConfig := &agent.AgentConfig{
		ModelConfig:   modelConfig,
		MCPConfig:     mcpConfig,
		SystemPrompt:  appConfig.SystemPrompt,
		MaxSteps:      appConfig.MaxSteps,
		MessageWindow: appConfig.MessageWindow,
	}

	// Create the agent
	ctx := context.Background()
	mcpAgent, err := agent.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Panic().Err(err).Msg("failed to create agent")
	}
	defer mcpAgent.Close()

	sessionManager := sessions.New()
	notificationServer := websocketServer.New()
	handlers := httpHandlers.New(templates, sessionManager, notificationServer, mcpAgent)

	listener := createNetListener(appConfig)
	server := startHttpServer(listener, handlers, notificationServer, appConfig.SimulatedDelay)

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

	sessionManager.Shutdown()

	time.Sleep(time.Second * 1) // Give some time for graceful shutdown
	log.Info().Msg("Application stopped")
}

func startHttpServer(listener net.Listener, handlers *httpHandlers.ChatHandlers,
	notificationServer websocketServer.WebsocketServer,
	simulatedDelay int) *http.Server {

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	httpLogger := httplog.NewLogger("backend-api", httplog.Options{
		LogLevel: slog.LevelDebug,
		JSON:     true,
		Concise:  true,
		//RequestHeaders:   true,
		//ResponseHeaders:  true,
	})

	router := chi.NewRouter()
	router.Use(httplog.RequestLogger(httpLogger))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://*",
			"http://*",
		},
	}))

	router.Handle(uiUrlPrefix+"*", staticAssets.Handler(webAssets.EmbedFs, embedFsRoot, uiUrlPrefix, defaultUiUrl))

	router.HandleFunc("/api/notifications", notificationServer.Handler)

	router.Handle("POST /api/ask", web.Handler{Request: handlers.Ask,
		SimulatedDelay: simulatedDelay})

	router.Handle("GET /api/main", web.Handler{Request: handlers.Main,
		SimulatedDelay: simulatedDelay})

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, uiUrlPrefix, http.StatusPermanentRedirect)
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

	return listener
}

func setupZerolog() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stderr).
		With().
		Timestamp().
		Logger()
}
