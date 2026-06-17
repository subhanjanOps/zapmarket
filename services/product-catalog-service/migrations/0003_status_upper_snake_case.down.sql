ALTER TABLE products DROP CONSTRAINT products_status_check;

UPDATE products SET status = 'draft'    WHERE status = 'DRAFT';
UPDATE products SET status = 'active'   WHERE status = 'ACTIVE';
UPDATE products SET status = 'inactive' WHERE status = 'INACTIVE';
UPDATE products SET status = 'archived' WHERE status = 'ARCHIVED';

ALTER TABLE products
    ALTER COLUMN status SET DEFAULT 'draft',
    ADD CONSTRAINT products_status_check
        CHECK (status IN ('draft', 'active', 'archived'));
