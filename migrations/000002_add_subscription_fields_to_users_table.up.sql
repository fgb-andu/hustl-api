ALTER TABLE users
    ADD COLUMN subscription_type TEXT DEFAULT 'free';

ALTER TABLE users
    ADD COLUMN subscription_platform TEXT DEFAULT 'none';

ALTER TABLE users
    ADD COLUMN original_transaction_id TEXT;

ALTER TABLE users
    ADD COLUMN subscription_expires_at DATETIME;

ALTER TABLE users
    ADD COLUMN subscription_last_verified DATETIME;
