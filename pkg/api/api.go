package api

import (
	"crypto/rsa"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	jwtdecode "github.com/fgb-andu/hustl-api/internal"
	"github.com/fgb-andu/hustl-api/pkg/domain"
	"github.com/fgb-andu/hustl-api/pkg/repository/userprovider"
	"github.com/fgb-andu/hustl-api/pkg/service/chat"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"strings"
	"sync"
)

// Updated request structure to include user ID
type ChatRequest struct {
	UserID   string   `json:"user_id"`
	Messages []string `json:"messages"`
}

type ChatResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type AuthResponse struct {
	User  *domain.User `json:"user"`
	Error *string      `json:"error,omitempty"`
}

type Handler struct {
	service  chat.Service
	userProv *userprovider.UserProvider
}

func NewHandler(service chat.Service, userProv *userprovider.UserProvider) *Handler {
	return &Handler{
		service:  service,
		userProv: userProv,
	}
}

// Update the Router to include the auth endpoint
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth endpoint
		r.Post("/guest", h.HandleGuestAuth)
		r.Post("/auth", h.HandleAuth)

		// Existing endpoints
		r.Post("/summarize", h.HandleSummarize)
		r.Post("/next-message", h.HandleNextMessage)
		r.Post("/set-entitlements", h.HandleSetEntitlements)
		r.Post("/update-config", h.UpdateConfig)
		r.Post("/update-prompt", h.UpdatePrompt)

	})

	return r
}

func (h *Handler) HandleSummarize(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check message limits
	if _, err := h.userProv.GetUser(req.UserID); err != nil {
		switch err {
		case userprovider.ErrUserNotFound:
			respondWithError(w, http.StatusNotFound, "User not found")
		case userprovider.ErrDailyLimitReached:
			respondWithError(w, http.StatusForbidden, "Daily message limit reached")
		default:
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	result := h.service.Summarize(req.Messages)
	respondWithJSON(w, http.StatusOK, ChatResponse{
		Result: result,
	})
}

func (h *Handler) HandleNextMessage(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check message limits
	if err := h.userProv.CheckAndIncrementMessageCount(req.UserID); err != nil {
		switch err {
		case userprovider.ErrUserNotFound:
			respondWithError(w, http.StatusNotFound, "User not found")
		case userprovider.ErrDailyLimitReached:
			respondWithError(w, http.StatusForbidden, "Daily message limit reached")
		default:
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	result := h.service.GetNextMessage(req.Messages)
	respondWithJSON(w, http.StatusOK, ChatResponse{
		Result: result,
	})
}

// Helper functions for JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ChatResponse{
		Error: message,
	})
}

type AuthRequest struct {
	Provider *domain.AuthProvider `json:"provider,omitempty"` // Optional: google/apple
	Username *string              `json:"username,omitempty"` // Required for google/apple
	Email    *string              `json:"email,omitempty"`    // Required for google/apple
	DeviceID *string              `json:"device_id"`          // Required for all requests
}

func (h *Handler) HandleGuestAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.DeviceID == nil || *req.DeviceID == "" {
		respondWithError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Handle anonymous session
	user, err := h.userProv.GetUserByUsername(*req.DeviceID)
	if err == nil {
		// Anonymous user exists, return it
		respondWithJSON(w, http.StatusOK, AuthResponse{User: user})
		return
	}

	// Create new anonymous user
	user, err = h.userProv.CreateUser(domain.AuthProviderGuest, *req.DeviceID, "")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}
	respondWithJSON(w, http.StatusCreated, user)
}

func (h *Handler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		respondWithError(w, http.StatusUnauthorized, "Authorization header missing or invalid")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		// Fetch the public key based on provider
		switch *req.Provider {
		// case domain.AuthProviderGoogle:
		// Get Google's public key
		//return GetGooglePublicKey(token)
		case domain.AuthProviderApple:
			// Get Apple's public key
			return GetApplePublicKey(token, false)
		default:
			return nil, jwt.NewValidationError("unknown provider", jwt.ValidationErrorUnverifiable)
		}
	})
	if err != nil || !token.Valid {
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify the signing method
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
			}
			// Fetch the public key based on provider
			switch *req.Provider {
			// case domain.AuthProviderGoogle:
			// Get Google's public key
			//return GetGooglePublicKey(token)
			case domain.AuthProviderApple:
				// Get Apple's public key
				return GetApplePublicKey(token, true)
			default:
				return nil, jwt.NewValidationError("unknown provider", jwt.ValidationErrorUnverifiable)
			}
		})
		if err != nil || !token.Valid {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
	}

	// Extract claims
	_, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		respondWithError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}

	// Validate request
	if req.DeviceID == nil || *req.DeviceID == "" {
		respondWithError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Handle authenticated session (Apple/Google)
	if req.Email == nil || *req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "email is required for authenticated sessions")
		return
	}

	// Look up user by email
	user, err := h.userProv.GetUserByUsername(*req.Username)
	if err == nil {
		// User exists, return it
		respondWithJSON(w, http.StatusOK, AuthResponse{User: user})
		return
	}

	// Create new authenticated user
	user, err = h.userProv.CreateUser(*req.Provider, *req.Username, *req.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondWithJSON(w, http.StatusCreated, AuthResponse{User: user})
	return
}

var (
	publicKeyCache     = make(map[string]*rsa.PublicKey)
	publicKeyCacheLock = sync.RWMutex{}
)

// Fetch Google's public key
func GetGooglePublicKey(token *jwt.Token, forceRefresh bool) (interface{}, error) {
	return jwtdecode.FetchPublicKeyFromURL("https://www.googleapis.com/oauth2/v3/certs", token, forceRefresh)
}

// Fetch Apple's public key
func GetApplePublicKey(token *jwt.Token, forceRefresh bool) (interface{}, error) {
	return jwtdecode.FetchPublicKeyFromURL("https://appleid.apple.com/auth/keys", token, forceRefresh)
}

type SetEntitlementsRequest struct {
	Username     string              `json:"username"`
	Subscription domain.Subscription `json:"subscription,omitempty"`
}

func (h *Handler) HandleSetEntitlements(w http.ResponseWriter, r *http.Request) {
	var req SetEntitlementsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate User ID
	if req.Username == "" {
		respondWithError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Fetch existing user
	user, err := h.userProv.GetUserByUsername(req.Username)
	if err != nil {
		switch err {
		case userprovider.ErrUserNotFound:
			respondWithError(w, http.StatusNotFound, "User not found")
			return
		default:
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch user")
			return
		}
	}

	// Update entitlements
	newEntitlements := domain.Entitlements{
		DailyMessageLimit: user.Entitlements.DailyMessageLimit,
		MessagesUsed:      user.Entitlements.MessagesUsed,
		LastReset:         user.Entitlements.LastReset,
		Subscription:      user.Entitlements.Subscription,
	}
	newEntitlements.Subscription = req.Subscription

	if user.Entitlements.Subscription.Type == domain.SubscriptionTypePremium {
		newEntitlements.DailyMessageLimit = 10000
		newEntitlements.MessagesUsed = 0
	} else if user.Entitlements.Subscription.Type == domain.SubscriptionTypeFree {
		newEntitlements.DailyMessageLimit = userprovider.FREE_USER_MESSAGE_LIMIT
	}

	if err := h.userProv.SetEntitlements(req.Username, newEntitlements); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update entitlements")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Entitlements updated successfully"})
}

type UpdateConfigRequest struct {
	Model            string  `json:"model"`
	MaxTokens        int     `json:"max_tokens"`
	Temperature      float32 `json:"temperature"`
	PresencePenalty  float32 `json:"presence_penalty"`
	FrequencyPenalty float32 `json:"frequency_penalty"`
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newConfig := chat.Config{
		Model:            req.Model,
		MaxTokens:        req.MaxTokens,
		Temperature:      req.Temperature,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
	}

	h.service.(*chat.GPTService).UpdateConfig(newConfig)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Config updated successfully"})
}

type UpdatePromptRequest struct {
	Prompt string `json:"prompt"`
}

func (h *Handler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	var req UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.service.(*chat.GPTService).UpdateInitialPrompt(req.Prompt)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Prompt updated successfully"})
}
