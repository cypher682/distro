INSERT INTO tenants (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Distributor');

INSERT INTO warehouses (id, tenant_id, name)
VALUES ('00000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000001',
        'Main Warehouse');

INSERT INTO retailers (id, tenant_id, name, credit_limit_cents)
VALUES ('00000000-0000-0000-0000-000000000020',
        '00000000-0000-0000-0000-000000000001',
        'Demo Retailer',
        10000000);

INSERT INTO products (id, tenant_id, sku, name, unit, price_cents)
VALUES
    ('00000000-0000-0000-0000-000000000101',
     '00000000-0000-0000-0000-000000000001',
     'RICE-25',
     'Rice 25kg',
     'bag',
     22000),
    ('00000000-0000-0000-0000-000000000102',
     '00000000-0000-0000-0000-000000000001',
     'OIL-5',
     'Cooking Oil 5L',
     'carton',
     18000);

INSERT INTO stock_movements
    (tenant_id, product_id, warehouse_id, quantity_delta, unit_cost_cents, total_cost_cents, reason)
VALUES
    ('00000000-0000-0000-0000-000000000001',
     '00000000-0000-0000-0000-000000000101',
     '00000000-0000-0000-0000-000000000010',
     100,
     15000,
     1500000,
     'seed opening balance'),
    ('00000000-0000-0000-0000-000000000001',
     '00000000-0000-0000-0000-000000000102',
     '00000000-0000-0000-0000-000000000010',
     50,
     12000,
     600000,
     'seed opening balance');

WITH opening AS (
    SELECT tenant_id,
           product_id,
           total_cost_cents,
           gen_random_uuid() AS entry_group
    FROM stock_movements
    WHERE reason = 'seed opening balance'
)
INSERT INTO ledger_entries
    (tenant_id, entry_group, account, debit_cents, credit_cents, reference_type, reference_id)
SELECT tenant_id, entry_group, 'inventory', total_cost_cents, 0, 'opening_balance', product_id
FROM opening
UNION ALL
SELECT tenant_id, entry_group, 'opening_equity', 0, total_cost_cents, 'opening_balance', product_id
FROM opening;
