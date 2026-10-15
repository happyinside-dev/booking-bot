-- +goose Up
CREATE TABLE bookings (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id),
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    service_id BIGINT NOT NULL REFERENCES services(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('PENDING','CONFIRMED','CANCELLED','COMPLETED','NO_SHOW')),
    price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (start_at < end_at)
);
ALTER TABLE bookings ADD CONSTRAINT bookings_no_overlap
EXCLUDE USING gist (
    employee_id WITH =,
    tstzrange(start_at, end_at, '[)') WITH &&
) WHERE (status IN ('PENDING', 'CONFIRMED'));
CREATE INDEX idx_bookings_user_start ON bookings(user_id, start_at DESC);
CREATE INDEX idx_bookings_business_start ON bookings(business_id, start_at);
CREATE INDEX idx_bookings_employee_start ON bookings(employee_id, start_at);
-- +goose Down
DROP TABLE bookings;
