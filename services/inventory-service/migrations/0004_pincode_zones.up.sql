CREATE TABLE IF NOT EXISTS pincode_zones (
    pincode      VARCHAR(10) NOT NULL,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    PRIMARY KEY (pincode)
);

-- Seed: route everything to the default warehouse until zones are configured
INSERT INTO pincode_zones (pincode, warehouse_id)
SELECT '000000', id FROM warehouses LIMIT 1
ON CONFLICT DO NOTHING;
