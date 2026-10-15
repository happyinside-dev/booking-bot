-- +goose Up
-- Compatibility migration for databases where the former, non-embedded
-- stage-3 migration was applied before booking settings were introduced.
ALTER TABLE businesses
    ADD COLUMN IF NOT EXISTS slot_interval_min INTEGER NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS min_booking_notice_min INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS max_booking_days INTEGER NOT NULL DEFAULT 30;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'businesses_slot_interval_positive' AND conrelid = 'businesses'::regclass) THEN
        ALTER TABLE businesses ADD CONSTRAINT businesses_slot_interval_positive CHECK (slot_interval_min > 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'businesses_min_notice_nonnegative' AND conrelid = 'businesses'::regclass) THEN
        ALTER TABLE businesses ADD CONSTRAINT businesses_min_notice_nonnegative CHECK (min_booking_notice_min >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'businesses_max_days_nonnegative' AND conrelid = 'businesses'::regclass) THEN
        ALTER TABLE businesses ADD CONSTRAINT businesses_max_days_nonnegative CHECK (max_booking_days >= 0);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE businesses
    DROP CONSTRAINT IF EXISTS businesses_slot_interval_positive,
    DROP CONSTRAINT IF EXISTS businesses_min_notice_nonnegative,
    DROP CONSTRAINT IF EXISTS businesses_max_days_nonnegative;
-- Columns intentionally remain: older 00003 installations may depend on them.
