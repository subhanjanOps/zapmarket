-- Extend seller_status to include REJECTED
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_seller_status_check;
ALTER TABLE users ADD CONSTRAINT users_seller_status_check
    CHECK (seller_status IN ('PENDING', 'APPROVED', 'REJECTED', 'SUSPENDED'));

-- KYC rejection metadata + onboarding tracking on seller_profiles
ALTER TABLE seller_profiles
    ADD COLUMN IF NOT EXISTS rejection_reason          TEXT,
    ADD COLUMN IF NOT EXISTS reviewed_by               UUID,
    ADD COLUMN IF NOT EXISTS approved_at               TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejected_at               TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS onboarding_steps_completed JSONB NOT NULL DEFAULT '{}';
