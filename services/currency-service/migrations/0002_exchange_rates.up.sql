CREATE TABLE IF NOT EXISTS exchange_rates (
  base       CHAR(3)      NOT NULL DEFAULT 'USD',
  quote      CHAR(3)      NOT NULL REFERENCES currencies(code),
  rate       NUMERIC(18,8) NOT NULL,
  as_of      TIMESTAMPTZ  NOT NULL,
  fetched_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
  PRIMARY KEY (base, quote)
);
