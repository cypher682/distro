CREATE TABLE invoice_lines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    invoice_id uuid NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    order_line_id uuid NOT NULL REFERENCES order_lines (id),
    product_id uuid NOT NULL REFERENCES products (id),
    warehouse_id uuid NOT NULL REFERENCES warehouses (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint NOT NULL CHECK (unit_price_cents >= 0),
    line_total_cents bigint NOT NULL CHECK (line_total_cents >= 0),
    total_cost_cents bigint NOT NULL CHECK (total_cost_cents >= 0),
    UNIQUE (invoice_id, order_line_id)
);

CREATE INDEX invoice_lines_invoice_idx ON invoice_lines (tenant_id, invoice_id);

CREATE TABLE sales_returns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    invoice_id uuid NOT NULL REFERENCES invoices (id),
    retailer_id uuid NOT NULL REFERENCES retailers (id),
    total_refund_cents bigint NOT NULL CHECK (total_refund_cents >= 0),
    total_cost_cents bigint NOT NULL CHECK (total_cost_cents >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sales_returns_invoice_idx
    ON sales_returns (tenant_id, invoice_id);

CREATE TABLE sales_return_lines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    sales_return_id uuid NOT NULL REFERENCES sales_returns (id) ON DELETE CASCADE,
    invoice_line_id uuid NOT NULL REFERENCES invoice_lines (id),
    product_id uuid NOT NULL REFERENCES products (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    refund_cents bigint NOT NULL CHECK (refund_cents >= 0),
    cost_cents bigint NOT NULL CHECK (cost_cents >= 0),
    restock boolean NOT NULL,
    UNIQUE (sales_return_id, invoice_line_id)
);

CREATE INDEX sales_return_lines_return_idx
    ON sales_return_lines (tenant_id, sales_return_id);
