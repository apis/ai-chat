package sessions

import (
	"ai-chat/internal/agent"
	"ai-chat/internal/pkg/chatSession"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

type SessionManager struct {
	chatSessions map[uuid.UUID]chatSession.ChatSession
}

func New() *SessionManager {
	sessionManager := &SessionManager{
		chatSessions: make(map[uuid.UUID]chatSession.ChatSession),
	}
	return sessionManager
}

func (instance *SessionManager) AddSession(id uuid.UUID, responseFunc chatSession.ChatBlockResponseFunc) error {
	_, ok := instance.chatSessions[id]
	if ok {
		return errors.New("session with such id already exists")
	}

	chat, err := chatSession.New(responseFunc, "")
	if err != nil {
		return fmt.Errorf("chatSession.New() failed: %w", err)
	}

	instance.chatSessions[id] = chat

	return nil
}

func (instance *SessionManager) AddAgentSession(id uuid.UUID, agent *agent.Agent, responseFunc chatSession.ChatBlockResponseFunc) error {
	_, ok := instance.chatSessions[id]
	if ok {
		return errors.New("session with such id already exists")
	}

	chat, err := chatSession.NewAgentChatSession(agent, responseFunc)
	if err != nil {
		return fmt.Errorf("chatSession.NewAgentChatSession() failed: %w", err)
	}

	instance.chatSessions[id] = chat

	return nil
}

func (instance *SessionManager) GetSession(id uuid.UUID) chatSession.ChatSession {

	return instance.chatSessions[id]
}

func (instance *SessionManager) Shutdown() {
	for _, session := range instance.chatSessions {
		session.Shutdown()
	}
}
