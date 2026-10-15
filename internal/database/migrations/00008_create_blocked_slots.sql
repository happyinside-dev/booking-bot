-- +goose Up
CREATE TABLE blocked_slots (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (start_at < end_at)
);
CREATE INDEX idx_blocked_slots_employee_time ON blocked_slots(employee_id, start_at, end_at);
-- +goose Down
DROP TABLE blocked_slots;
