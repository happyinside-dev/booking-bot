-- +goose Up
CREATE TABLE business_members (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL CHECK (role IN ('OWNER', 'ADMIN', 'EMPLOYEE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, user_id)
);
CREATE INDEX idx_business_members_user ON business_members(user_id);
-- +goose Down
DROP TABLE business_members;
