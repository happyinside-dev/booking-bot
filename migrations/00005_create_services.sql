-- +goose Up

CREATE TABLE services (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    duration_min INTEGER NOT NULL CHECK(duration_min > 0),
    price NUMERIC(10,2) NOT NULL CHECK(price >= 0),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_services_business
ON services(business_id);

-- +goose Down

DROP TABLE services;
