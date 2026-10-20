ALTER TABLE order_lines
    ADD CONSTRAINT order_lines_order_product_key UNIQUE (order_id, product_id);

CREATE TABLE invoices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    order_id uuid NOT NULL UNIQUE REFERENCES orders (id),
    retailer_id uuid NOT NULL REFERENCES retailers (id),
    status text NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'paid', 'void')),
    total_cents bigint NOT NULL CHECK (total_cents >= 0),
    paid_cents bigint NOT NULL DEFAULT 0
        CHECK (paid_cents >= 0 AND paid_cents <= total_cents),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX invoices_open_idx
    ON invoices (tenant_id, retailer_id)
    WHERE status = 'open';

CREATE TABLE payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    invoice_id uuid NOT NULL REFERENCES invoices (id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX payments_invoice_idx ON payments (tenant_id, invoice_id);
