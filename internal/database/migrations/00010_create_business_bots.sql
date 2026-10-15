-- +goose Up
CREATE TABLE business_bots (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    telegram_bot_id BIGINT UNIQUE,
    username VARCHAR(255) NOT NULL DEFAULT '',
    token_key VARCHAR(128) UNIQUE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id)
);
CREATE INDEX idx_business_bots_active ON business_bots(is_active) WHERE is_active = TRUE;
-- +goose Down
DROP TABLE business_bots;
