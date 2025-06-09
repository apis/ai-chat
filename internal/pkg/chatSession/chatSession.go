package chatSession

type ChatBlockResponse struct {
	ChatBlock ChatBlock
	New       bool
}

type ChatBlock struct {
	SystemMessage    string
	UserMessage      string
	AssistantMessage string
	Completed        bool
	Failed           bool
}

type ChatSession interface {
	EnqueueMessage(message string) error
	Shutdown()
	ChatBlocks() []ChatBlock
}

type ChatBlockResponseFunc func(response ChatBlockResponse)
