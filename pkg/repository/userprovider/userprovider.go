package userprovider

import (
	"errors"
	"github.com/fgb-andu/hustl-api/pkg/domain"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

const MESSAGES_RESET_TIME = 1 * time.Minute
const FREE_USER_MESSAGE_LIMIT = 5

type UserProvider struct {
	users           map[string]*domain.User
	usersByUsername map[string]*domain.User
	mu              sync.RWMutex
}

func NewUserProvider() *UserProvider {
	return &UserProvider{
		users:           make(map[string]*domain.User),
		usersByUsername: make(map[string]*domain.User),
	}
}
func (p *UserProvider) CreateUser(authProvider domain.AuthProvider, username string) (*domain.User, error) {
	log.Println("Creating User in user provider.")
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	user := domain.User{ID: id.String(), AuthProvider: authProvider, Username: username, Entitlements: domain.Entitlements{
		DailyMessageLimit: FREE_USER_MESSAGE_LIMIT,
		MessagesUsed:      0,
		LastReset:         time.Now(),
		Subscription:      domain.Subscription{},
	}}

	p.mu.Lock()
	p.users[id.String()] = &user
	p.usersByUsername[user.Username] = &user
	p.mu.Unlock()

	return &user, nil
}

func (p *UserProvider) GetUser(id string) (*domain.User, error) {
	log.Println("Getting User from user provider.")

	p.mu.RLock()
	user, exists := p.users[id]
	p.mu.RUnlock()

	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (p *UserProvider) GetUserByUsername(username string) (*domain.User, error) {
	log.Println("Getting User by Username from user provider.")

	p.mu.RLock()
	user, exists := p.users[username]
	p.mu.RUnlock()

	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CheckAndIncrementMessageCount checks if user can send a message and increments count if possible
func (p *UserProvider) CheckAndIncrementMessageCount(userID string) error {
	log.Println("Checking user and incrementing counter if needed.")

	p.mu.Lock()
	defer p.mu.Unlock()

	user, exists := p.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	// Check if we need to reset daily count
	if time.Since(user.Entitlements.LastReset) > MESSAGES_RESET_TIME {
		log.Println("Resetting!")
		user.Entitlements.MessagesUsed = 0
		user.Entitlements.LastReset = time.Now()
	}

	// Check if user has reached their limit
	if user.Entitlements.MessagesUsed >= user.Entitlements.DailyMessageLimit {
		return ErrDailyLimitReached
	}

	// Increment message count
	user.Entitlements.MessagesUsed++
	log.Println("Messaged used:", user.Entitlements.MessagesUsed)
	user.LastActive = time.Now()

	return nil
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrDailyLimitReached = errors.New("daily message limit reached")
)
