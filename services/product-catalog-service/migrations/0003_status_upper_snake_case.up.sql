-- Migrate product status values and constraint to UPPER_SNAKE_CASE.
-- Also adds INACTIVE which was missing from the original constraint.
ALTER TABLE products DROP CONSTRAINT products_status_check;

UPDATE products SET status = 'DRAFT'    WHERE status = 'draft';
UPDATE products SET status = 'ACTIVE'   WHERE status = 'active';
UPDATE products SET status = 'INACTIVE' WHERE status = 'inactive';
UPDATE products SET status = 'ARCHIVED' WHERE status = 'archived';

ALTER TABLE products
    ALTER COLUMN status SET DEFAULT 'DRAFT',
    ADD CONSTRAINT products_status_check
        CHECK (status IN ('DRAFT', 'ACTIVE', 'INACTIVE', 'ARCHIVED'));
