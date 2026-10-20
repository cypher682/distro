CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE retailers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name text NOT NULL,
    credit_limit_cents bigint NOT NULL DEFAULT 0 CHECK (credit_limit_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE warehouses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    sku text NOT NULL,
    name text NOT NULL,
    unit text NOT NULL DEFAULT 'unit',
    price_cents bigint NOT NULL CHECK (price_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, sku)
);

CREATE TABLE stock_movements (
    id bigserial PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    quantity_delta integer NOT NULL CHECK (quantity_delta <> 0),
    reason text NOT NULL,
    reference_type text,
    reference_id uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX stock_movements_product_idx
    ON stock_movements (tenant_id, product_id, warehouse_id, created_at);

CREATE TABLE orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    retailer_id uuid NOT NULL REFERENCES retailers (id),
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'reserved', 'invoiced', 'cancelled')),
    total_cents bigint NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE order_lines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint NOT NULL CHECK (unit_price_cents >= 0)
);

CREATE TABLE reservations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    order_id uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    status text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'released', 'fulfilled')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX reservations_active_idx
    ON reservations (tenant_id, product_id, warehouse_id)
    WHERE status = 'active';

CREATE TABLE ledger_entries (
    id bigserial PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    entry_group uuid NOT NULL,
    account text NOT NULL,
    debit_cents bigint NOT NULL DEFAULT 0 CHECK (debit_cents >= 0),
    credit_cents bigint NOT NULL DEFAULT 0 CHECK (credit_cents >= 0),
    reference_type text NOT NULL,
    reference_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (debit_cents > 0 AND credit_cents = 0)
        OR (credit_cents > 0 AND debit_cents = 0)
    )
);

CREATE INDEX ledger_entries_group_idx ON ledger_entries (tenant_id, entry_group);
