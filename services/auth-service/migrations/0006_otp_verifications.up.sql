CREATE TABLE otp_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT        NOT NULL,
    purpose     TEXT        NOT NULL CHECK (purpose IN ('email_verify','phone_verify','phone_password_reset')),
    recipient   TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_user_purpose ON otp_verifications(user_id, purpose, used_at);
