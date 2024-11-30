package domain

import (
	"time"
)

type AuthProvider string

const (
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderApple  AuthProvider = "apple"
	AuthProviderGuest  AuthProvider = "guest"
)

type SubscriptionType string

const (
	SubscriptionTypeFree    SubscriptionType = "free"
	SubscriptionTypePremium SubscriptionType = "premium"
)

type SubscriptionPlatform string

const (
	SubscriptionPlatformNone   SubscriptionPlatform = "none"
	SubscriptionPlatformApple  SubscriptionPlatform = "apple"
	SubscriptionPlatformGoogle SubscriptionPlatform = "google"
)

type Subscription struct {
	Type                  SubscriptionType     `json:"type"`
	Platform              SubscriptionPlatform `json:"platform"`
	OriginalTransactionID string               `json:"original_transaction_id,omitempty"`
	ExpiresAt             *time.Time           `json:"expires_at,omitempty"`
	LastVerified          *time.Time           `json:"last_verified,omitempty"`
}

type Entitlements struct {
	DailyMessageLimit int          `json:"daily_message_limit"`
	MessagesUsed      int          `json:"messages_used"`
	LastReset         time.Time    `json:"last_reset"`
	Subscription      Subscription `json:"subscription"`
}

type User struct {
	ID           string       `json:"id" db:"id"`
	AuthProvider AuthProvider `json:"auth_provider" db:"auth_provider"`
	Username     string       `json:"username" db:"username"`
	Email        string       `json:"email" db:"email"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	LastActive   time.Time    `json:"last_active" db:"last_active"`
	Entitlements Entitlements `json:"entitlements" db:"entitlements"`
}
