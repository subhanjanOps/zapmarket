CREATE TABLE IF NOT EXISTS currencies (
  code       CHAR(3)      PRIMARY KEY,
  name       TEXT         NOT NULL,
  flag       TEXT         NOT NULL DEFAULT '',
  decimals   SMALLINT     NOT NULL DEFAULT 2,
  enabled    BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
