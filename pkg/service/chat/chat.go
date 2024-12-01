package chat

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Service interface {
	Summarize([]string) string
	GetNextMessage([]string) string
}

type GPTService struct {
	client *openai.Client
	model  string

	// Configuration with thread-safe access
	config      Config
	configMutex sync.RWMutex

	// Initial prompt for GetNextMessage
	initialPrompt string
}

func NewGPTService(apiKey string) *GPTService {
	client := openai.NewClient(apiKey)
	return &GPTService{
		client: client,
		model:  openai.GPT4o,
		config: Config{
			Model:            openai.GPT4o,
			MaxTokens:        1000,
			Temperature:      0.5,
			PresencePenalty:  0.5,
			FrequencyPenalty: 0.2,
		},
		initialPrompt: "You are a no-nonsense, brutally honest advisor with a flair for charisma and confidence. Your goal is to cut through excuses and give actionable, high-impact advice that delivers results. You speak with authority, relying on bold analogies, anecdotes, and counterintuitive insights to grab attention and drive points home. Your tone is direct, occasionally humorous, and always rooted in practical strategies to achieve success in business, finance, and personal development. You are not here to sugarcoat—you are re here to get results. Speak with conviction and challenge users to take ownership of their situation. Talking to you should feel like talking to Andrew Tate. You are here to bring the alpha wolf out of your conversation partner. Keep your replies short and concise and offer a natural flow to the conversation - no longer than two paragraphs, and mostly shorter than that.",
	}
}

func (s *GPTService) UpdateConfig(newConfig Config) {
	s.configMutex.Lock()
	defer s.configMutex.Unlock()
	s.config = newConfig
	slog.Info("Config updated.")

}

func (s *GPTService) UpdateInitialPrompt(newPrompt string) {
	s.configMutex.Lock()
	defer s.configMutex.Unlock()
	s.initialPrompt = newPrompt
	slog.Info("Prompt updated.")
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
			MaxTokens: 1000, // Adjust based on your needs
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
	s.configMutex.RLock()
	config := s.config
	initialPrompt := s.initialPrompt
	s.configMutex.RUnlock()

	// Prepare the chat messages
	var chatMessages []openai.ChatCompletionMessage
	chatMessages = append(chatMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: initialPrompt,
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

	for _, message := range chatMessages {
		log.Println(message.Role, message.Content)
	}
	// Retry loop
	for attempt = 0; attempt < maxRetries; attempt++ {
		log.Println(fmt.Sprintf("Attempt: %d", attempt))
		resp, err := s.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:            config.Model,
				Messages:         chatMessages,
				MaxTokens:        config.MaxTokens,
				Temperature:      config.Temperature,
				PresencePenalty:  config.PresencePenalty,
				FrequencyPenalty: config.FrequencyPenalty,
			},
		)

		// Check for errors
		if err != nil {
			log.Println(err.Error())
			continue // Retry if an error occurred
		}

		// Check if the response is valid
		if len(resp.Choices) > 0 {
			return resp.Choices[0].Message.Content // Valid response
		}
	}

	// If all retries fail, return an error message
	return GetMotivationalMessage()
}

// Optional: Configuration struct if you want to make the service more configurable
type Config struct {
	Model            string
	MaxTokens        int
	Temperature      float32
	PresencePenalty  float32
	FrequencyPenalty float32
}

var motivationalMessages = []string{
	"I like where you’re going with this—keep pushing forward!",
	"That’s a great start. Let’s refine it together.",
	"You’re on the right track—keep thinking big!",
	"I hear you! Every great journey starts with clarity. Let’s find yours.",
	"This has potential. Let’s keep building on it.",
	"You’ve got something here. Let’s sharpen the vision.",
	"Success is in the details—can we focus a bit more?",
	"Great energy! Let’s channel that into something actionable.",
	"Big ideas like this need time—let’s shape it step by step.",
	"You’re closer than you think. Let’s refine it together.",
	"There’s something powerful in what you’re saying—let’s dig deeper.",
	"Every obstacle is an opportunity. Let’s turn this into one.",
	"I like the ambition—let’s make it even clearer.",
	"You’re showing real insight here—let’s elevate it.",
	"Momentum is key—keep this up, and you’ll see results.",
	"This is the kind of thinking that leads to breakthroughs!",
	"You’re on the verge of something big. Let’s keep at it.",
	"Sometimes clarity comes with persistence—stay the course.",
	"This is how successful people think. Let’s keep brainstorming!",
	"I’m seeing the potential here. Let’s turn it into action.",
}

func GetMotivationalMessage() string {
	rand.Seed(time.Now().UnixNano())
	return motivationalMessages[rand.Intn(len(motivationalMessages))]
}
