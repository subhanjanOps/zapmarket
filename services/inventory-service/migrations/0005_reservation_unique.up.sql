ALTER TABLE inventory_reservations
    ADD CONSTRAINT uq_reservation_order_sku UNIQUE (order_id, sku_id);
