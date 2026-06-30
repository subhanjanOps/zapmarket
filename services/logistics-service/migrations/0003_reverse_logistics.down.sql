ALTER TABLE shipments
    DROP COLUMN IF EXISTS parent_shipment_id,
    DROP COLUMN IF EXISTS shipment_type;
