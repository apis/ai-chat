package chatSession

import (
	"github.com/ollama/ollama/api"
)

type ChatBlockResponse struct {
	ChatBlock ChatBlock
	New       bool
}

type ChatBlock struct {
	SystemMessage    string
	UserMessage      string
	AssistantMessage string
	ToolResponses    []api.Message // Added to store tool call responses
	Completed        bool
	Failed           bool
}

type ChatSession interface {
	EnqueueMessage(message string) error
	Shutdown()
	ChatBlocks() []ChatBlock
}

type ChatBlockResponseFunc func(response ChatBlockResponse)
