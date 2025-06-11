package chatSession

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ollama/ollama/api"
	"github.com/rs/zerolog/log"
	"net/http"
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
	calculatorMcpUrl    string
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

		if len(chatBlock.ToolResponses) > 0 {
			for _, toolResponse := range chatBlock.ToolResponses {
				messages = append(messages, api.Message{
					Role:    "tool",
					Content: toolResponse.Content,
					// ToolCallID: toolResponse.ToolCallID, // Compiler says api.Message has no ToolCallID
				})
			}
		}
	}
	return messages
}

func New(responseFunc ChatBlockResponseFunc, calculatorMcpUrl string) (ChatSession, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Error().Err(err).Msg("ollama api.ClientFromEnvironment() failed")
		return nil, err
	}

	calculatorToolsJson := `[
      {
        "type": "function",
        "function": {
          "name": "calculator.add",
          "description": "Add two numbers",
          "parameters": {
            "type": "object",
            "properties": {
              "a": {"type": "number", "description": "First number"},
              "b": {"type": "number", "description": "Second number"}
            },
            "required": ["a", "b"]
          }
        }
      },
      {
        "type": "function",
        "function": {
          "name": "calculator.subtract",
          "description": "Subtract second number from first number",
          "parameters": {
            "type": "object",
            "properties": {
              "a": {"type": "number", "description": "First number"},
              "b": {"type": "number", "description": "Second number"}
            },
            "required": ["a", "b"]
          }
        }
      },
      {
        "type": "function",
        "function": {
          "name": "calculator.multiply",
          "description": "Multiply two numbers",
          "parameters": {
            "type": "object",
            "properties": {
              "a": {"type": "number", "description": "First number"},
              "b": {"type": "number", "description": "Second number"}
            },
            "required": ["a", "b"]
          }
        }
      },
      {
        "type": "function",
        "function": {
          "name": "calculator.divide",
          "description": "Divide first number by second number",
          "parameters": {
            "type": "object",
            "properties": {
              "a": {"type": "number", "description": "First number (dividend)"},
              "b": {"type": "number", "description": "Second number (divisor)"}
            },
            "required": ["a", "b"]
          }
        }
      }
    ]`

	var apiTools api.Tools
	if err := json.Unmarshal([]byte(calculatorToolsJson), &apiTools); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal calculator tools json")
		return nil, fmt.Errorf("failed to unmarshal calculator tools: %w", err)
	}

	chat := &chatSessionImpl{sessions: []*ChatBlock{},
		model:               "llama3", // Using a model known to work well with tools
		stream:              true,
		client:              client,
		questions:           make(chan string, questionQueueBufferSize),
		exitRequested:       make(chan any, 1),
		sessionResponseFunc: responseFunc,
		tools:               apiTools,
		calculatorMcpUrl:    calculatorMcpUrl,
	}

	go chat.questionsProcessingHandler()

	return chat, nil
}

// processToolsResponse handles calls to external tools and appends their responses.
func (instance *chatSessionImpl) processToolsResponse(toolCalls []api.ToolCall) error {
	if len(toolCalls) == 0 {
		return nil
	}

	currentChatBlock := instance.sessions[len(instance.sessions)-1]

	for _, call := range toolCalls {
		log.Info().Str("tool_name", call.Function.Name).Msg("Processing tool call")

		toolURL := instance.calculatorMcpUrl + call.Function.Name

		argsBytes, err := json.Marshal(call.Function.Arguments)
		if err != nil {
			log.Error().Err(err).Str("tool_name", call.Function.Name).Msg("Failed to marshal tool arguments")
			currentChatBlock.ToolResponses = append(currentChatBlock.ToolResponses, api.Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error marshalling arguments for tool %s: %v", call.Function.Name, err),
				// ToolCallID: call.ID, // Compiler says call.ID is undefined
			})
			continue
		}
		requestBody := bytes.NewBuffer(argsBytes)

		resp, err := http.Post(toolURL, "application/json", requestBody)
		if err != nil {
			log.Error().Err(err).Str("tool_url", toolURL).Msg("Failed to call tool")
			currentChatBlock.ToolResponses = append(currentChatBlock.ToolResponses, api.Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error calling tool %s: %v", call.Function.Name, err),
				// ToolCallID: call.ID, // Compiler says call.ID is undefined
			})
			continue
		}
		defer resp.Body.Close()

		var toolCallResult struct {
			Result float64 `json:"result"`
			Error  string  `json:"error"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&toolCallResult); err != nil {
			log.Error().Err(err).Str("tool_url", toolURL).Msg("Failed to decode tool response")
			currentChatBlock.ToolResponses = append(currentChatBlock.ToolResponses, api.Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error decoding response from tool %s: %v", call.Function.Name, err),
				// ToolCallID: call.ID, // Compiler says call.ID is undefined
			})
			continue
		}

		var content string
		if toolCallResult.Error != "" {
			content = toolCallResult.Error
		} else {
			content = fmt.Sprintf("%f", toolCallResult.Result)
		}

		currentChatBlock.ToolResponses = append(currentChatBlock.ToolResponses, api.Message{
			Role:    "tool",
			Content: content,
			// ToolCallID: call.ID, // Compiler says call.ID is undefined
		})
		log.Info().Str("tool_name", call.Function.Name).Str("result", content).Msg("Tool call processed")
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

		if response.Message.ToolCalls != nil && len(response.Message.ToolCalls) > 0 {
			// Process tool calls. The responses will be added to currentChatBlock.ToolResponses
			if err := instance.processToolsResponse(response.Message.ToolCalls); err != nil {
				// Log the error, session.Failed might be set by processToolsResponse or here
				log.Error().Err(err).Msg("Error processing tool responses")
				session.Failed = true // Mark session as failed if tool processing has critical errors
			}
			// Important: After processing tools, the assistant's turn might not be "done" yet.
			// We need to send the tool responses back to Ollama.
			// The next iteration of Chat will include these tool responses.
			// So, we update the session, send the current assistant message (if any),
			// and then the main loop will re-evaluate.
			// If response.Done is true here, it means the assistant decided it's done *after* issuing tool calls.
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
