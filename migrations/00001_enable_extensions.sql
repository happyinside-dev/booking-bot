-- +goose Up
-- btree_gist is required for the exclusion constraint that prevents
-- overlapping bookings for the same employee (see Stage 6).
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- +goose Down
DROP EXTENSION IF EXISTS btree_gist;
