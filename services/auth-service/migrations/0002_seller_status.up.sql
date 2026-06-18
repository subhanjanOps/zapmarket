ALTER TABLE users ADD COLUMN IF NOT EXISTS
    seller_status VARCHAR(20)
    CHECK (seller_status IN ('PENDING', 'APPROVED', 'SUSPENDED'))
    DEFAULT NULL;

-- Existing sellers start APPROVED (they were auto-approved before this change).
UPDATE users SET seller_status = 'APPROVED' WHERE role = 'seller' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_seller_status
    ON users (seller_status) WHERE seller_status IS NOT NULL AND deleted_at IS NULL;
