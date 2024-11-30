package main

import (
	"fmt"
	"github.com/fgb-andu/hustl-api/pkg/api"
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
		port = "5801"
	}

	// Initialize GPT service
	service := chat.NewGPTService(apiKey)

	provider, err := userprovider.NewUserProvider(userprovider.Config{
		DatabasePath:   "./users.db",
		MigrationsPath: "./migrations",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()
	// Initialize handler with service
	handler := api.NewHandler(service, provider)

	// Get router
	router := handler.Router()

	// Start server
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on port %s", port)

	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatal(err)
	}
}
