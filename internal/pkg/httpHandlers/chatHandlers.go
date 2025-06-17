package httpHandlers

import (
	"ai-chat/internal/pkg/agent"
	"ai-chat/internal/pkg/chatSession"
	"ai-chat/internal/pkg/cookies"
	"ai-chat/internal/pkg/sessions"
	"ai-chat/internal/pkg/web"
	"ai-chat/internal/pkg/websocketServer"
	"bytes"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"html/template"
	"net/http"
	"sync"
	"time"
)

type ChatHandlers struct {
	templates          *template.Template
	notificationServer websocketServer.WebsocketServer
	sessionManager     *sessions.SessionManager
	mcpAgent           *agent.Agent
	sessionMessages    map[uuid.UUID][]*schema.Message
	sessionMutex       sync.RWMutex
}

func New(templates *template.Template, sessionManager *sessions.SessionManager,
	notificationServer websocketServer.WebsocketServer, mcpAgent *agent.Agent) *ChatHandlers {
	return &ChatHandlers{
		templates:          templates,
		sessionManager:     sessionManager,
		notificationServer: notificationServer,
		mcpAgent:           mcpAgent,
		sessionMessages:    make(map[uuid.UUID][]*schema.Message),
		sessionMutex:       sync.RWMutex{},
	}
}

func (instance *ChatHandlers) Main(request *http.Request, simulatedDelay int) *web.Response {
	time.Sleep(time.Duration(simulatedDelay) * time.Millisecond)

	var cookie *http.Cookie
	var id uuid.UUID
	id = cookies.GetIdFromCookie(request)
	if id == uuid.Nil || instance.sessionManager.GetSession(id) == nil {
		var err error
		id, err = uuid.NewUUID()
		if err != nil {
			return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
		}

		cookie = cookies.SetIdToCookie(id)
		if cookie == nil {
			return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
		}

		err = instance.sessionManager.AddAgentSession(id, instance.mcpAgent, instance.chatBlockResponseHandler(id))
		if err != nil {
			log.Error().Err(err).Msg("sessionManager.AddAgentSession() failed")
			return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
		}
	}

	session := instance.sessionManager.GetSession(id)
	if session == nil {
		log.Error().Msg("sessionManager.GetSession() failed")
		return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
	}

	headers := map[string]string{"HX-Trigger-After-Swap": "{\"parseAllRawMessages\":\"\"}"}
	return web.RenderResponse(http.StatusOK, instance.templates, "main.gohtml", ToUiSessions(session.ChatBlocks()), headers, cookie)
}

func (instance *ChatHandlers) Ask(request *http.Request, simulatedDelay int) *web.Response {
	id := cookies.GetIdFromCookie(request)
	if id == uuid.Nil {
		return web.GetEmptyResponse(http.StatusBadRequest, nil, nil)
	}

	session := instance.sessionManager.GetSession(id)
	if session == nil {
		log.Error().Msg("sessionManager.GetSession() failed")
		return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
	}

	err := request.ParseForm()
	if err != nil {
		log.Error().Err(err).Msg("http.Request.ParseForm() failed")
		return web.GetEmptyResponse(http.StatusBadRequest, nil, nil)
	}

	userInput := request.Form.Get("user-input")
	if userInput == "" {
		return web.GetEmptyResponse(http.StatusBadRequest, nil, nil)
	}

	// Enqueue the message to the session
	err = session.EnqueueMessage(userInput)
	if err != nil {
		log.Error().Err(err).Msg("enqueue question failed")
		return web.GetEmptyResponse(http.StatusInternalServerError, nil, nil)
	}

	time.Sleep(time.Duration(simulatedDelay) * time.Millisecond)

	headers := map[string]string{"HX-Trigger-After-Swap": "{\"clearUserInput\":\"\"}"}
	return web.GetEmptyResponse(http.StatusOK, headers, nil)
}

func (instance *ChatHandlers) chatBlockResponseHandler(id uuid.UUID) func(response chatSession.ChatBlockResponse) {
	return func(response chatSession.ChatBlockResponse) {
		uiResponse := ToUiSessionResponse(response)

		templateName := "chat-response.gohtml"
		var buffer bytes.Buffer
		if err := instance.templates.ExecuteTemplate(&buffer, templateName, uiResponse); err != nil {
			log.Error().Err(err).Str("template_name", templateName).Msg("templates.ExecuteTemplate() failed")
		}
		log.Debug().Str("system_message", response.ChatBlock.SystemMessage).
			Str("user_message", response.ChatBlock.UserMessage).
			Str("assistant_message", response.ChatBlock.AssistantMessage).
			Bool("completed", response.ChatBlock.Completed).
			Bool("failed", response.ChatBlock.Failed).
			Msg("ChatBlockResponse")
		instance.notificationServer.Publish(id, buffer.Bytes())
	}
}
