ALTER TABLE inventory_reservations
    DROP CONSTRAINT IF EXISTS uq_reservation_order_sku;
