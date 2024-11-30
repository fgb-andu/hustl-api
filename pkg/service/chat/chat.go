package chat

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"strings"
)

type Service interface {
	Summarize([]string) string
	GetNextMessage([]string) string
}

type GPTService struct {
	client *openai.Client
	model  string
}

func NewGPTService(apiKey string) *GPTService {
	client := openai.NewClient(apiKey)
	return &GPTService{
		client: client,
		model:  openai.GPT4TurboPreview, // or openai.GPT3Dot5Turbo based on your needs
	}
}

func (s *GPTService) Summarize(messages []string) string {
	prompt := fmt.Sprintf(
		"Please provide a concise summary of the following conversation:\n\n%s",
		strings.Join(messages, "\n"),
	)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: s.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that provides concise summaries.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 150, // Adjust based on your needs
		},
	)

	if err != nil {
		// In a real application, you'd want to handle this error appropriately
		return fmt.Sprintf("Error generating summary: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "No summary generated"
	}

	return resp.Choices[0].Message.Content
}

func (s *GPTService) GetNextMessage(messages []string) string {
	// Convert the string array into ChatCompletionMessages
	var chatMessages []openai.ChatCompletionMessage

	// First, add a system message to set the context
	chatMessages = append(chatMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "You are a helpful assistant engaging in a conversation. Provide natural and contextually appropriate responses.",
	})

	// Add the conversation history
	for i, msg := range messages {
		role := openai.ChatMessageRoleUser
		if i%2 == 1 { // Alternate between user and assistant messages
			role = openai.ChatMessageRoleAssistant
		}
		chatMessages = append(chatMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg,
		})
	}

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    s.model,
			Messages: chatMessages,
		},
	)

	if err != nil {
		// In a real application, you'd want to handle this error appropriately
		return fmt.Sprintf("Error generating response: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "No response generated"
	}

	return resp.Choices[0].Message.Content
}

// Optional: Configuration struct if you want to make the service more configurable
type Config struct {
	Model            string
	MaxTokens        int
	Temperature      float32
	PresencePenalty  float32
	FrequencyPenalty float32
}
