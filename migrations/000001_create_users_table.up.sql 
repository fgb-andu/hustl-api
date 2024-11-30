-- 000001_create_users_table.up.sql
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    auth_provider TEXT NOT NULL,
    username TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    daily_message_limit INTEGER NOT NULL,
    messages_used INTEGER NOT NULL DEFAULT 0,
    last_reset DATETIME NOT NULL,
    last_active DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);