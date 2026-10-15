-- +goose Up
CREATE INDEX idx_bookings_business_status_start
    ON bookings(business_id, status, start_at);
CREATE INDEX idx_bookings_business_service_start
    ON bookings(business_id, service_id, start_at);
CREATE INDEX idx_bookings_business_employee_start
    ON bookings(business_id, employee_id, start_at);
-- +goose Down
DROP INDEX idx_bookings_business_employee_start;
DROP INDEX idx_bookings_business_service_start;
DROP INDEX idx_bookings_business_status_start;
