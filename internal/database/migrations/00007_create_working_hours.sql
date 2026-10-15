-- +goose Up
CREATE TABLE working_hours (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_day_off BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (employee_id, weekday),
    CHECK (is_day_off OR start_time < end_time)
);
CREATE INDEX idx_working_hours_employee ON working_hours(employee_id);
-- +goose Down
DROP TABLE working_hours;
