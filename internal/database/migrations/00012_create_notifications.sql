-- +goose Up
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL CHECK (type IN ('REMINDER_24H', 'REMINDER_1H')),
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'SENT', 'FAILED', 'CANCELLED')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    locked_until TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (booking_id, type)
);
CREATE INDEX idx_notifications_due
    ON notifications(status, scheduled_at)
    WHERE status IN ('PENDING', 'PROCESSING');
CREATE INDEX idx_notifications_lease
    ON notifications(locked_until)
    WHERE status = 'PROCESSING';
-- +goose Down
DROP TABLE notifications;
