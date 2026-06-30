ALTER TABLE users
    ADD COLUMN IF NOT EXISTS totp_secret       TEXT,
    ADD COLUMN IF NOT EXISTS totp_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS totp_enrolled_at  TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS user_backup_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backup_codes_user ON user_backup_codes(user_id) WHERE used_at IS NULL;
