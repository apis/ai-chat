package chatSession

import (
	"ai-chat/internal/agent"
	"context"
	"github.com/cloudwego/eino/schema"
	"github.com/rs/zerolog/log"
	"sync"
)

// AgentChatSession is an implementation of the ChatSession interface that uses agent.Agent
type AgentChatSession struct {
	agent            *agent.Agent
	messages         []*schema.Message
	chatBlocks       []ChatBlock
	responseFunc     ChatBlockResponseFunc
	messagesMutex    sync.RWMutex
	processingMutex  sync.Mutex
	currentChatBlock *ChatBlock
	exitRequested    chan struct{}
}

// NewAgentChatSession creates a new AgentChatSession
func NewAgentChatSession(agent *agent.Agent, responseFunc ChatBlockResponseFunc) (ChatSession, error) {
	return &AgentChatSession{
		agent:         agent,
		messages:      []*schema.Message{},
		chatBlocks:    []ChatBlock{},
		responseFunc:  responseFunc,
		messagesMutex: sync.RWMutex{},
		exitRequested: make(chan struct{}, 1),
	}, nil
}

// EnqueueMessage adds a user message to the chat session and processes it
func (a *AgentChatSession) EnqueueMessage(message string) error {
	// Create a new chat block for this message
	chatBlock := ChatBlock{
		UserMessage: message,
		Completed:   false,
		Failed:      false,
	}

	// Add the chat block to the list
	a.messagesMutex.Lock()
	a.chatBlocks = append(a.chatBlocks, chatBlock)
	a.currentChatBlock = &a.chatBlocks[len(a.chatBlocks)-1]
	a.messagesMutex.Unlock()

	// Send initial UI update with user message
	a.responseFunc(ChatBlockResponse{
		ChatBlock: chatBlock,
		New:       true,
	})

	// Add user message to the messages list
	userMessage := schema.UserMessage(message)
	a.messagesMutex.Lock()
	a.messages = append(a.messages, userMessage)
	a.messagesMutex.Unlock()

	// Process the message with the agent in a goroutine
	go a.processMessage()

	return nil
}

// processMessage processes the current message with the agent
func (a *AgentChatSession) processMessage() {
	// Ensure only one message is processed at a time
	a.processingMutex.Lock()
	defer a.processingMutex.Unlock()

	// Create a copy of messages to avoid race conditions
	a.messagesMutex.RLock()
	messagesCopy := make([]*schema.Message, len(a.messages))
	copy(messagesCopy, a.messages)
	currentChatBlock := a.currentChatBlock
	a.messagesMutex.RUnlock()

	// Call the agent
	ctx := context.Background()
	response, err := a.agent.GenerateWithLoop(ctx, messagesCopy,
		// Tool call handler
		func(toolName, toolArgs string) {
			log.Info().Str("tool", toolName).Str("args", toolArgs).Msg("Tool call")
		},
		// Tool execution handler
		func(toolName string, isStarting bool) {
			if isStarting {
				log.Info().Str("tool", toolName).Msg("Tool execution started")
			} else {
				log.Info().Str("tool", toolName).Msg("Tool execution completed")
			}
		},
		// Tool result handler
		func(toolName, toolArgs, result string, isError bool) {
			if isError {
				log.Error().Str("tool", toolName).Str("result", result).Msg("Tool error")
			} else {
				log.Info().Str("tool", toolName).Str("result", result).Msg("Tool result")
			}
		},
		// Response handler
		func(content string) {
			// Update the chat block with the assistant's response
			a.messagesMutex.Lock()
			currentChatBlock.AssistantMessage = content
			currentChatBlock.Completed = true
			a.messagesMutex.Unlock()

			// Send UI update
			a.responseFunc(ChatBlockResponse{
				ChatBlock: *currentChatBlock,
				New:       false,
			})
		},
		// Tool call content handler
		func(content string) {
			// Update the chat block with intermediate content
			a.messagesMutex.Lock()
			currentChatBlock.AssistantMessage = content
			a.messagesMutex.Unlock()

			// Send UI update
			a.responseFunc(ChatBlockResponse{
				ChatBlock: *currentChatBlock,
				New:       false,
			})
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("Agent.GenerateWithLoop failed")
		a.messagesMutex.Lock()
		currentChatBlock.Failed = true
		currentChatBlock.AssistantMessage = "Error: " + err.Error()
		a.messagesMutex.Unlock()

		// Send UI update with error
		a.responseFunc(ChatBlockResponse{
			ChatBlock: *currentChatBlock,
			New:       false,
		})
		return
	}

	// Add assistant response to messages
	a.messagesMutex.Lock()
	a.messages = append(a.messages, response)
	a.messagesMutex.Unlock()
}

// Shutdown stops the chat session
func (a *AgentChatSession) Shutdown() {
	select {
	case a.exitRequested <- struct{}{}:
		log.Info().Msg("AgentChatSession shutdown requested")
	default:
		log.Info().Msg("AgentChatSession shutdown already requested")
	}
}

// ChatBlocks returns the chat blocks
func (a *AgentChatSession) ChatBlocks() []ChatBlock {
	a.messagesMutex.RLock()
	defer a.messagesMutex.RUnlock()

	// Create a copy of the chat blocks to avoid race conditions
	chatBlocks := make([]ChatBlock, len(a.chatBlocks))
	copy(chatBlocks, a.chatBlocks)

	return chatBlocks
}