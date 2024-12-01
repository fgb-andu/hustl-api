package userprovider

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/fgb-andu/hustl-api/pkg/domain"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

const MESSAGES_RESET_TIME = 1 * time.Minute
const FREE_USER_MESSAGE_LIMIT = 5

type UserProvider struct {
	db *sql.DB
}

type Config struct {
	DatabasePath   string
	MigrationsPath string
}

func NewUserProvider(config Config) (*UserProvider, error) {
	db, err := sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db, config.MigrationsPath); err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	return &UserProvider{db: db}, nil
}

func runMigrations(db *sql.DB, migrationsPath string) error {
	log.Println("Running migrations.")
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create database driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}

// Rest of the UserProvider implementation remains the same...
func (p *UserProvider) CreateUser(authProvider domain.AuthProvider, username string, email string) (*domain.User, error) {
	log.Println("Creating user with username: " + username)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	_, err = p.db.Exec(`
    INSERT INTO users (
        id, auth_provider, username, email, 
        daily_message_limit, messages_used, last_reset, last_active,
        subscription_type, subscription_platform
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), authProvider, username, email,
		FREE_USER_MESSAGE_LIMIT, 0, time.Now(), time.Now(),
		domain.SubscriptionTypeFree, domain.SubscriptionPlatformNone,
	)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:           id.String(),
		AuthProvider: authProvider,
		Username:     username,
		Email:        email,
		Entitlements: domain.Entitlements{
			DailyMessageLimit: FREE_USER_MESSAGE_LIMIT,
			MessagesUsed:      0,
			LastReset:         time.Now(),
			Subscription:      domain.Subscription{},
		},
	}, nil
}

func (p *UserProvider) GetUser(id string) (*domain.User, error) {
	log.Println("Getting user by ID: " + id)

	var user domain.User
	var lastReset, lastActive time.Time
	var originalTransactionID sql.NullString
	var subscriptionExpiresAt sql.NullTime

	err := p.db.QueryRow(`
        SELECT id, auth_provider, username, email, 
               daily_message_limit, messages_used, last_reset, last_active, 
               subscription_type, subscription_platform, original_transaction_id, subscription_expires_at
        FROM users WHERE id = ?`, id).Scan(
		&user.ID, &user.AuthProvider, &user.Username, &user.Email,
		&user.Entitlements.DailyMessageLimit, &user.Entitlements.MessagesUsed,
		&lastReset, &lastActive,
		&user.Entitlements.Subscription.Type, &user.Entitlements.Subscription.Platform,
		&originalTransactionID, &subscriptionExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Entitlements.LastReset = lastReset
	user.LastActive = lastActive

	// Handle nullable fields
	if originalTransactionID.Valid {
		user.Entitlements.Subscription.OriginalTransactionID = originalTransactionID.String
	}
	if subscriptionExpiresAt.Valid {
		user.Entitlements.Subscription.ExpiresAt = &subscriptionExpiresAt.Time
	}

	// Check if we need to reset the message count
	if time.Since(lastReset) > MESSAGES_RESET_TIME {
		log.Println("Resetting!")
		_, err = p.db.Exec(`
            UPDATE users 
            SET messages_used = 0, last_reset = ? 
            WHERE id = ?`,
			time.Now(), id,
		)
		if err != nil {
			return nil, err
		}
		user.Entitlements.MessagesUsed = 0
		user.Entitlements.LastReset = time.Now()
	}

	return &user, nil
}

func (p *UserProvider) GetUserByUsername(username string) (*domain.User, error) {
	log.Println("Getting User by username: " + username)

	var user domain.User
	var lastReset, lastActive time.Time
	var originalTransactionID sql.NullString
	var subscriptionExpiresAt sql.NullTime

	err := p.db.QueryRow(`
        SELECT id, auth_provider, username, email, 
               daily_message_limit, messages_used, last_reset, last_active, 
               subscription_type, subscription_platform, original_transaction_id, subscription_expires_at
        FROM users WHERE username = ?`, username).Scan(
		&user.ID, &user.AuthProvider, &user.Username, &user.Email,
		&user.Entitlements.DailyMessageLimit, &user.Entitlements.MessagesUsed,
		&lastReset, &lastActive,
		&user.Entitlements.Subscription.Type, &user.Entitlements.Subscription.Platform,
		&originalTransactionID, &subscriptionExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Entitlements.LastReset = lastReset
	user.LastActive = lastActive

	// Handle nullable fields
	if originalTransactionID.Valid {
		user.Entitlements.Subscription.OriginalTransactionID = originalTransactionID.String
	}
	if subscriptionExpiresAt.Valid {
		user.Entitlements.Subscription.ExpiresAt = &subscriptionExpiresAt.Time
	}

	// Check if we need to reset the message count
	if time.Since(lastReset) > MESSAGES_RESET_TIME {
		log.Println("Resetting!")
		_, err = p.db.Exec(`
            UPDATE users 
            SET messages_used = 0, last_reset = ? 
            WHERE username = ?`,
			time.Now(), username,
		)
		if err != nil {
			return nil, err
		}
		user.Entitlements.MessagesUsed = 0
		user.Entitlements.LastReset = time.Now()
	}

	return &user, nil
}

func (p *UserProvider) CheckAndIncrementMessageCount(userID string) error {
	log.Println("Checking user and incrementing counter if needed.")

	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var messagesUsed int
	var dailyMessageLimit int
	var lastReset time.Time

	err = tx.QueryRow(`
        SELECT messages_used, daily_message_limit, last_reset 
        FROM users WHERE username = ?`, userID).Scan(
		&messagesUsed, &dailyMessageLimit, &lastReset,
	)

	if err == sql.ErrNoRows {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	// Check if we need to reset daily count
	if time.Since(lastReset) > MESSAGES_RESET_TIME {
		log.Println("Resetting!")
		messagesUsed = 0
		lastReset = time.Now()
	}

	// Check if user has reached their limit
	if messagesUsed >= dailyMessageLimit {
		return ErrDailyLimitReached
	}

	// Increment message count
	_, err = tx.Exec(`
        UPDATE users 
        SET messages_used = ?, last_reset = ?, last_active = ? 
        WHERE username = ?`,
		messagesUsed+1, lastReset, time.Now(), userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *UserProvider) Close() error {
	return p.db.Close()
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrDailyLimitReached = errors.New("daily message limit reached")
)

func (p *UserProvider) SetEntitlements(username string, entitlements domain.Entitlements) error {
	log.Println("Updating entitlements for user:", username)

	query := `
        UPDATE users 
        SET daily_message_limit = COALESCE(?, daily_message_limit),
            messages_used = COALESCE(?, messages_used),
            subscription_type = COALESCE(?, subscription_type),
            subscription_platform = COALESCE(?, subscription_platform),
            original_transaction_id = COALESCE(?, original_transaction_id),
            subscription_expires_at = COALESCE(?, subscription_expires_at)
        WHERE username = ?`

	_, err := p.db.Exec(query,
		entitlements.DailyMessageLimit,
		entitlements.MessagesUsed,
		entitlements.Subscription.Type,
		entitlements.Subscription.Platform,
		entitlements.Subscription.OriginalTransactionID,
		entitlements.Subscription.ExpiresAt,
		username,
	)

	if err != nil {
		return fmt.Errorf("failed to update entitlements: %w", err)
	}

	return nil
}
