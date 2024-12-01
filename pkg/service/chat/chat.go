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
		model:  openai.GPT4o, // or openai.GPT3Dot5Turbo based on your needs

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
	const maxRetries = 3 // Limit the number of retries
	var attempt int
	var resp openai.ChatCompletionResponse
	var err error

	// Prepare the chat messages
	var chatMessages []openai.ChatCompletionMessage
	chatMessages = append(chatMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "You are a confident and practical advisor. Provide concise, professional, and actionable advice.",
	})
	for i, msg := range messages {
		role := openai.ChatMessageRoleUser
		if i%2 == 1 {
			role = openai.ChatMessageRoleAssistant
		}
		chatMessages = append(chatMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg,
		})
	}

	// Retry loop
	for attempt = 0; attempt < maxRetries; attempt++ {
		resp, err = s.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:            s.model,
				Messages:         chatMessages,
				MaxTokens:        150,
				Temperature:      0.7,
				PresencePenalty:  0.5,
				FrequencyPenalty: 0.2,
			},
		)

		// Check for errors
		if err != nil {
			continue // Retry if an error occurred
		}

		// Check if the response is valid
		if len(resp.Choices) > 0 {
			return resp.Choices[0].Message.Content // Valid response
		}
	}

	// If all retries fail, return an error message
	return "Unable to generate a valid response after several attempts."
}

// Optional: Configuration struct if you want to make the service more configurable
type Config struct {
	Model            string
	MaxTokens        int
	Temperature      float32
	PresencePenalty  float32
	FrequencyPenalty float32
}
