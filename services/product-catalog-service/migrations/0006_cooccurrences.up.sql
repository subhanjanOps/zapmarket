CREATE TABLE IF NOT EXISTS product_cooccurrences (
    product_a UUID NOT NULL,
    product_b UUID NOT NULL,
    count     INT  NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_a, product_b)
);

CREATE INDEX IF NOT EXISTS idx_cooccurrences_a ON product_cooccurrences (product_a, count DESC);
