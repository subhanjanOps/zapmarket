CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS categories (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id  UUID         REFERENCES categories (id),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS products (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID         NOT NULL REFERENCES categories (id),
    seller_id   UUID         NOT NULL,
    name        VARCHAR(500) NOT NULL,
    slug        VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    attributes  JSONB        NOT NULL DEFAULT '{}',
    status      VARCHAR(50)  NOT NULL DEFAULT 'DRAFT'
                CHECK (status IN ('DRAFT', 'ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_products_category ON products (category_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_products_seller   ON products (seller_id)   WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_products_status   ON products (status)      WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_products_attrs    ON products USING GIN (attributes);

CREATE TABLE IF NOT EXISTS skus (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id    UUID         NOT NULL REFERENCES products (id),
    sku_code      VARCHAR(100) NOT NULL UNIQUE,
    variant_attrs JSONB        NOT NULL DEFAULT '{}',
    price_amount  BIGINT       NOT NULL,
    currency      CHAR(3)      NOT NULL DEFAULT 'INR',
    compare_price BIGINT,
    weight_grams  INT,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_skus_product ON skus (product_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS product_images (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID        NOT NULL REFERENCES products (id),
    sku_id     UUID        REFERENCES skus (id),
    url        TEXT        NOT NULL,
    position   SMALLINT    NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
