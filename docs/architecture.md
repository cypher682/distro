# Architecture

Distro is a Go modular monolith backed by PostgreSQL. The first slice proves the
order-to-cash and inventory-accounting flow for a wholesale distributor.

## Runtime

```text
HTTP client
    |
    v
cmd/api -> internal/httpapi -> internal/core -> Store
                                               |-- MemoryStore (tests)
                                               `-- PostgresStore
                                                        |
                                                        v
                                                   PostgreSQL
```

- `cmd/api` loads configuration, opens the PostgreSQL pool, and serves HTTP.
- `internal/httpapi` owns routing, JSON encoding, tenant header parsing, and
  HTTP status mapping.
- `internal/core` owns domain rules, validation, and the `Store` contract.
- `PostgresStore` implements the contract in transactions. `MemoryStore` is the
  fast unit-test implementation.

The service layer is deliberately thin. Complex invariants that need database
locks live in `PostgresStore`; the memory implementation mirrors the same
behavior for tests.

## Data model

| Table | Purpose |
|---|---|
| `tenants` | Tenant boundary |
| `retailers` | Buyers and credit limits |
| `warehouses` | Stock locations |
| `products` | Tenant catalog and selling price |
| `stock_movements` | Append-only quantity and cost movements |
| `reservations` | Active, released, or fulfilled order allocations |
| `orders`, `order_lines` | Draft-to-invoiced sales orders |
| `invoices`, `payments` | Receivables and cash application |
| `ledger_entries` | Double-entry accounting entries |

Money and costs are integer cents. Stock is derived from movements; there is no
mutable on-hand column.

## Invariants

- On-hand = sum of `stock_movements.quantity_delta` for a product and warehouse.
- Available = on-hand minus active reservations. It cannot go negative.
- Receipts and invoices lock product rows before reading stock balances, so
  concurrent reservations and issues serialize per product.
- An order can be reserved only when stock and retailer credit are sufficient.
- Credit exposure = unpaid open invoices plus reserved order value.
- Invoice postings balance accounts receivable against revenue.
- COGS postings balance cost of goods sold against inventory.
- Stock valuation = sum of signed movement costs and must equal the inventory
  ledger balance.
- Stock receipts post inventory against accounts payable; negative adjustments
  post inventory shrinkage against inventory.

See [ADR-0002](adr/0002-core-money-and-stock.md) and
[ADR-0003](adr/0003-inventory-costing.md) for the accounting decisions.

## Order flow

1. `POST /api/orders` snapshots product prices into a draft order.
2. `POST /api/orders/{id}/reserve` checks credit and availability, then creates
   active reservations. Product rows are locked to prevent oversell.
3. `POST /api/orders/{id}/invoice` computes weighted-average issue cost, writes
   negative stock movements, marks reservations fulfilled, and posts revenue and
   COGS.
4. `POST /api/invoices/{id}/payments` applies cash and reduces accounts
   receivable.
5. `GET /api/ledger/trial-balance` and `GET /api/inventory/valuation` expose
   reconciliation checks.

## Errors

Domain errors are sentinel values: `ErrNotFound`, `ErrConflict`, and
`ErrInvalid`. `httpapi` maps them to `404`, `409`, and `400`. Unknown errors are
logged and returned as `500` without internal details.

## Security boundaries

There is no authentication or database row-level security in this MVP. The
`X-Tenant-ID` header is trusted and is only suitable for local development or a
trusted network. Do not expose the API to untrusted clients. See
[SECURITY.md](../SECURITY.md).

## Deferred

Procurement, picking and dispatch, multi-warehouse allocation, lots/FIFO, tax,
period close, aging and dunning, public API credentials, webhooks, and a web UI
are outside this slice. The modular package boundaries leave room to add them
without splitting the service prematurely.
