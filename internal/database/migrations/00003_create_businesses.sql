-- +goose Up
CREATE TABLE businesses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    timezone VARCHAR(64) NOT NULL,
    slot_interval_min INTEGER NOT NULL DEFAULT 30 CONSTRAINT businesses_slot_interval_positive CHECK (slot_interval_min > 0),
    min_booking_notice_min INTEGER NOT NULL DEFAULT 60 CONSTRAINT businesses_min_notice_nonnegative CHECK (min_booking_notice_min >= 0),
    max_booking_days INTEGER NOT NULL DEFAULT 30 CONSTRAINT businesses_max_days_nonnegative CHECK (max_booking_days >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE businesses;
