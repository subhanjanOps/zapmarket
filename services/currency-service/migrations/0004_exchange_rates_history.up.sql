CREATE TABLE IF NOT EXISTS exchange_rates_history (
    base       CHAR(3)        NOT NULL REFERENCES currencies(code),
    quote      CHAR(3)        NOT NULL REFERENCES currencies(code),
    rate       NUMERIC(18, 8) NOT NULL,
    as_of      DATE           NOT NULL,
    fetched_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (base, quote, as_of)
);

CREATE INDEX IF NOT EXISTS idx_erh_base_asof ON exchange_rates_history (base, as_of DESC);
