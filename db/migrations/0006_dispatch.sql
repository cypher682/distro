ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('draft', 'reserved', 'dispatched', 'invoiced', 'cancelled'));

CREATE TABLE delivery_notes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders (id),
    retailer_id uuid NOT NULL REFERENCES retailers (id),
    status text NOT NULL DEFAULT 'picking'
        CHECK (status IN ('picking', 'dispatched', 'cancelled')),
    total_cost_cents bigint NOT NULL DEFAULT 0 CHECK (total_cost_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    dispatched_at timestamptz
);

CREATE INDEX delivery_notes_tenant_idx
    ON delivery_notes (tenant_id, created_at DESC);

CREATE UNIQUE INDEX delivery_notes_open_order_idx
    ON delivery_notes (tenant_id, order_id)
    WHERE status <> 'cancelled';

CREATE TABLE delivery_note_lines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    delivery_note_id uuid NOT NULL REFERENCES delivery_notes (id) ON DELETE CASCADE,
    reservation_id uuid NOT NULL REFERENCES reservations (id),
    order_line_id uuid NOT NULL REFERENCES order_lines (id),
    product_id uuid NOT NULL REFERENCES products (id),
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint NOT NULL CHECK (unit_price_cents >= 0),
    line_total_cents bigint NOT NULL CHECK (line_total_cents >= 0),
    total_cost_cents bigint NOT NULL DEFAULT 0 CHECK (total_cost_cents >= 0),
    UNIQUE (delivery_note_id, order_line_id)
);

CREATE INDEX delivery_note_lines_note_idx
    ON delivery_note_lines (tenant_id, delivery_note_id);
