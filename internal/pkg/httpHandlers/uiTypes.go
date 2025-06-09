package httpHandlers

import (
	"ai-chat/internal/pkg/chatSession"
	"encoding/base64"
)

type UiSessionResponse struct {
	UiSession
	New bool
}

type UiSession struct {
	SystemMessageContent    string
	UserMessageContent      string
	AssistantMessageContent string
	Completed               bool
	Failed                  bool
}

func toUiSession(session chatSession.ChatBlock) UiSession {
	uiSession := UiSession{
		SystemMessageContent:    "",
		UserMessageContent:      "",
		AssistantMessageContent: "",
		Completed:               session.Completed,
		Failed:                  session.Failed}

	if session.SystemMessage != "" {
		systemMessageContent := base64.StdEncoding.EncodeToString([]byte(session.SystemMessage))
		uiSession.SystemMessageContent = systemMessageContent
	}

	if session.UserMessage != "" {
		userMessageContent := base64.StdEncoding.EncodeToString([]byte(session.UserMessage))
		uiSession.UserMessageContent = userMessageContent
	}

	if session.AssistantMessage != "" {
		assistantMessageContent := base64.StdEncoding.EncodeToString([]byte(session.AssistantMessage))
		uiSession.AssistantMessageContent = assistantMessageContent
	}

	return uiSession
}

func ToUiSessionResponse(response chatSession.ChatBlockResponse) UiSessionResponse {
	uiResponse := UiSessionResponse{
		UiSession: toUiSession(response.ChatBlock),
		New:       response.New}

	return uiResponse
}

func ToUiSessions(sessions []chatSession.ChatBlock) []UiSession {
	uiSessions := make([]UiSession, len(sessions))
	for i, session := range sessions {
		uiSessions[i] = toUiSession(session)
	}
	return uiSessions
}
