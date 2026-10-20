CREATE TABLE suppliers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name text NOT NULL,
    contact_name text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE purchase_orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    supplier_id uuid NOT NULL REFERENCES suppliers (id),
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    status text NOT NULL DEFAULT 'ordered'
        CHECK (status IN ('ordered', 'received', 'cancelled')),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    received_at timestamptz
);

CREATE INDEX purchase_orders_tenant_idx
    ON purchase_orders (tenant_id, created_at DESC);

CREATE TABLE purchase_order_lines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    purchase_order_id uuid NOT NULL REFERENCES purchase_orders (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_cost_cents bigint NOT NULL CHECK (unit_cost_cents >= 0),
    line_total_cents bigint NOT NULL CHECK (line_total_cents >= 0),
    UNIQUE (purchase_order_id, product_id)
);

CREATE INDEX purchase_order_lines_order_idx
    ON purchase_order_lines (tenant_id, purchase_order_id);
