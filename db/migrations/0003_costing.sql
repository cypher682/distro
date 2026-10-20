ALTER TABLE stock_movements
    ADD COLUMN unit_cost_cents bigint NOT NULL DEFAULT 0
        CHECK (unit_cost_cents >= 0),
    ADD COLUMN total_cost_cents bigint NOT NULL DEFAULT 0;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_cost_sign_check CHECK (
        (quantity_delta > 0 AND total_cost_cents >= 0)
        OR (quantity_delta < 0 AND total_cost_cents <= 0)
    );
