ALTER TABLE users
    DROP COLUMN subscription_type;

ALTER TABLE users
    DROP COLUMN subscription_platform;

ALTER TABLE users
    DROP COLUMN original_transaction_id;

ALTER TABLE users
    DROP COLUMN subscription_expires_at;

ALTER TABLE users
    DROP COLUMN subscription_last_verified;
