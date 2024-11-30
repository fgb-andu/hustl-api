package main

import (
	"fmt"
	"github.com/fgb-andu/hustl-api/pkg/api"
	"github.com/fgb-andu/hustl-api/pkg/domain"
	"github.com/fgb-andu/hustl-api/pkg/repository/userprovider"
	"github.com/fgb-andu/hustl-api/pkg/service/chat"
	"log"
	"net/http"
	"os"
)

func main() {
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "5565"
	}

	// Initialize GPT service
	service := chat.NewGPTService(apiKey)
	userProvider := userprovider.NewUserProvider()
	// Create some test users
	user1, _ := userProvider.CreateUser(domain.AuthProviderGoogle, "test@example.com")
	user2, _ := userProvider.CreateUser(domain.AuthProviderGuest, "device123")

	log.Printf("Created test users: %s, %s", user1.ID, user2.ID)
	// Initialize handler with service
	handler := api.NewHandler(service, userProvider)

	// Get router
	router := handler.Router()

	// Start server
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on port %s", port)

	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatal(err)
	}
}
