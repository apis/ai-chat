package chatSession

import (
	"ai-chat/internal/pkg/agent"
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
func (instance *AgentChatSession) EnqueueMessage(message string) error {
	// Create a new chat block for this message
	chatBlock := ChatBlock{
		UserMessage: message,
		Completed:   false,
		Failed:      false,
	}

	// Add the chat block to the list
	instance.messagesMutex.Lock()
	instance.chatBlocks = append(instance.chatBlocks, chatBlock)
	instance.currentChatBlock = &instance.chatBlocks[len(instance.chatBlocks)-1]
	instance.messagesMutex.Unlock()

	// Send initial UI update with user message
	instance.responseFunc(ChatBlockResponse{
		ChatBlock: chatBlock,
		New:       true,
	})

	// Add user message to the messages list
	userMessage := schema.UserMessage(message)
	instance.messagesMutex.Lock()
	instance.messages = append(instance.messages, userMessage)
	instance.messagesMutex.Unlock()

	// Process the message with the agent in instance goroutine
	go instance.processMessage()

	return nil
}

// processMessage processes the current message with the agent
func (instance *AgentChatSession) processMessage() {
	// Ensure only one message is processed at a time
	instance.processingMutex.Lock()
	defer instance.processingMutex.Unlock()

	// Create a copy of messages to avoid race conditions
	instance.messagesMutex.RLock()
	messagesCopy := make([]*schema.Message, len(instance.messages))
	copy(messagesCopy, instance.messages)
	currentChatBlock := instance.currentChatBlock
	instance.messagesMutex.RUnlock()

	// Call the agent
	ctx := context.Background()
	response, err := instance.agent.GenerateWithLoop(ctx, messagesCopy,
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
			instance.messagesMutex.Lock()
			currentChatBlock.AssistantMessage = content
			currentChatBlock.Completed = true
			instance.messagesMutex.Unlock()

			// Send UI update
			instance.responseFunc(ChatBlockResponse{
				ChatBlock: *currentChatBlock,
				New:       false,
			})
		},
		// Tool call content handler
		func(content string) {
			// Update the chat block with intermediate content
			instance.messagesMutex.Lock()
			currentChatBlock.AssistantMessage = content
			instance.messagesMutex.Unlock()

			// Send UI update
			instance.responseFunc(ChatBlockResponse{
				ChatBlock: *currentChatBlock,
				New:       false,
			})
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("Agent.GenerateWithLoop failed")
		instance.messagesMutex.Lock()
		currentChatBlock.Failed = true
		currentChatBlock.AssistantMessage = "Error: " + err.Error()
		instance.messagesMutex.Unlock()

		// Send UI update with error
		instance.responseFunc(ChatBlockResponse{
			ChatBlock: *currentChatBlock,
			New:       false,
		})
		return
	}

	// Add assistant response to messages
	instance.messagesMutex.Lock()
	instance.messages = append(instance.messages, response)
	instance.messagesMutex.Unlock()
}

// Shutdown stops the chat session
func (instance *AgentChatSession) Shutdown() {
	select {
	case instance.exitRequested <- struct{}{}:
		log.Info().Msg("AgentChatSession shutdown requested")
	default:
		log.Info().Msg("AgentChatSession shutdown already requested")
	}
}

// ChatBlocks returns the chat blocks
func (instance *AgentChatSession) ChatBlocks() []ChatBlock {
	instance.messagesMutex.RLock()
	defer instance.messagesMutex.RUnlock()

	// Create a copy of the chat blocks to avoid race conditions
	chatBlocks := make([]ChatBlock, len(instance.chatBlocks))
	copy(chatBlocks, instance.chatBlocks)

	return chatBlocks
}
