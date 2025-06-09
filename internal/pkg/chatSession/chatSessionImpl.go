package chatSession

import (
	"context"
	"errors"
	"github.com/ollama/ollama/api"
	"github.com/rs/zerolog/log"
)

const questionQueueBufferSize = 16

type chatSessionImpl struct {
	sessions            []*ChatBlock
	tools               api.Tools
	model               string
	stream              bool
	client              *api.Client
	questions           chan string
	exitRequested       chan any
	sessionResponseFunc ChatBlockResponseFunc
}

func (instance *chatSessionImpl) ChatBlocks() []ChatBlock {
	sessions := make([]ChatBlock, len(instance.sessions))
	for index := range instance.sessions {
		sessions[index] = *instance.sessions[index]
	}
	return sessions
}

func (instance *chatSessionImpl) EnqueueMessage(message string) error {
	select {
	case instance.questions <- message:
		return nil
	default:
		return errors.New("question queue is full")
	}
}

func toApiMessages(chatBlocks []*ChatBlock) []api.Message {
	messages := make([]api.Message, 0)
	for _, chatBlock := range chatBlocks {
		if chatBlock.Failed {
			continue
		}

		if chatBlock.SystemMessage != "" {
			systemApiMessage := api.Message{
				Role:    "system",
				Content: chatBlock.SystemMessage,
			}
			messages = append(messages, systemApiMessage)
		}

		if chatBlock.UserMessage != "" {
			userApiMessage := api.Message{
				Role:    "user",
				Content: chatBlock.UserMessage,
			}
			messages = append(messages, userApiMessage)
		}

		if chatBlock.AssistantMessage != "" {
			assistantApiMessage := api.Message{
				Role:    "assistant",
				Content: chatBlock.AssistantMessage,
			}
			messages = append(messages, assistantApiMessage)
		}
	}
	return messages
}

func New(responseFunc ChatBlockResponseFunc, toolsJson string) (ChatSession, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Error().Err(err).Msg("ollama api.ClientFromEnvironment() failed")
		return nil, err
	}

	//var apiTools api.Tools
	//if toolsJson != "" {
	//	apiTools = api.Tools{}
	//	err = json.Unmarshal([]byte(toolsJson), &apiTools)
	//	if err != nil {
	//		log.Err(err).Msg("failed to unmarshal tools json")
	//		return nil, err
	//	}
	//}

	chat := &chatSessionImpl{sessions: []*ChatBlock{},
		//model: "mistral:7b",
		model: "granite3.3:8b",
		//model:               "llama3.2:3b",
		stream:              true,
		client:              client,
		questions:           make(chan string, questionQueueBufferSize),
		exitRequested:       make(chan any, 1),
		sessionResponseFunc: responseFunc,
		tools:               nil}

	go chat.questionsProcessingHandler()

	return chat, nil
}

func (instance *chatSessionImpl) processToolsResponse(toolCalls []api.ToolCall) error {
	for _, call := range toolCalls {
		log.Info().Int("call_index", call.Function.Index).Str("call_function", call.Function.Name).Msg("==== TOOL CALL ===")
	}

	return nil
}

func (instance *chatSessionImpl) askQuestion(ctx context.Context, question string) error {
	systemMessage := "You are a helpful AI assistant. Use the provided tools when necessary."

	if len(instance.sessions) == 0 {
		systemMessage = ""
	}

	session := ChatBlock{
		SystemMessage:    systemMessage,
		UserMessage:      question,
		AssistantMessage: "",
		Completed:        false,
		Failed:           false,
	}

	instance.sessions = append(instance.sessions, &session)

	instance.sessionResponseFunc(ChatBlockResponse{
		ChatBlock: session,
		New:       true,
	})

	request := &api.ChatRequest{
		Model:    instance.model,
		Messages: toApiMessages(instance.sessions),
		Stream:   &instance.stream,
		Tools:    instance.tools,
	}

	var messageContent string

	respFunc := func(response api.ChatResponse) error {
		messageContent = messageContent + response.Message.Content
		session.AssistantMessage = messageContent

		if response.Done {
			session.Completed = true
		}

		if response.Message.ToolCalls != nil {
			instance.processToolsResponse(response.Message.ToolCalls)
		}

		instance.sessionResponseFunc(ChatBlockResponse{
			ChatBlock: session,
			New:       false,
		})

		return nil
	}

	err := instance.client.Chat(ctx, request, respFunc)
	if err != nil {
		session.Failed = true
		log.Error().Err(err).Msg("ollama api.Client.chatSessionImpl() failed")
		return err
	}

	return nil
}

func (instance *chatSessionImpl) questionsProcessingHandler() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case question := <-instance.questions:
			go func() {
				log.Info().Str("question_content", question).Msg("processing question")
				err := instance.askQuestion(ctx, question)
				if err != nil {
					// TODO: Send notification
				}
			}()
			break
		case <-instance.exitRequested:
			log.Info().Msg("questions processing cancelled")
			return
		}
	}
}

func (instance *chatSessionImpl) Shutdown() {
	select {
	case instance.exitRequested <- struct{}{}:
		log.Info().Msg("chatSessionImpl shutdown requested")
		return
	default:
		log.Info().Msg("chatSessionImpl shutdown requested again")
		return
	}
}
