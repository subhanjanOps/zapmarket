ALTER TABLE seller_profiles
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS onboarding_steps_completed;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_seller_status_check;
ALTER TABLE users ADD CONSTRAINT users_seller_status_check
    CHECK (seller_status IN ('PENDING', 'APPROVED', 'SUSPENDED'));
