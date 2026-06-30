ALTER TABLE products
    DROP COLUMN IF EXISTS sales_rank,
    DROP COLUMN IF EXISTS avg_rating;
