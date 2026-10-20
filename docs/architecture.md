# Architecture

Distro is a Go modular monolith backed by PostgreSQL. The first slice proves the
order-to-cash and inventory-accounting flow for a wholesale distributor.

## Runtime

```text
Browser / HTTP client
    |
    v
cmd/api
  |-- /api/ and /health* -> internal/httpapi --+
  `-- /                  -> internal/web -----+-> internal/core -> Store
                                                                     |-- MemoryStore (tests)
                                                                     `-- PostgresStore
                                                                              |
                                                                              v
                                                                         PostgreSQL
```

- `cmd/api` loads configuration, opens the PostgreSQL pool, and mounts both
  handlers on one HTTP server. `/api/` and `/health*` go to the JSON API;
  everything else goes to the console.
- `internal/httpapi` owns API routing, JSON encoding, tenant header parsing,
  and HTTP status mapping.
- `internal/web` owns the server-rendered browser console. Templates and static
  assets are embedded in the binary; there is no Node build.
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
| `suppliers` | Vendors for inbound stock |
| `purchase_orders`, `purchase_order_lines` | Ordered and received supplier purchases |
| `stock_movements` | Append-only quantity and cost movements |
| `reservations` | Active, released, or fulfilled order allocations |
| `orders`, `order_lines` | Draft-to-invoiced sales orders |
| `invoices`, `invoice_lines` | Receivables and immutable invoice cost snapshots |
| `payments` | Cash application |
| `sales_returns`, `sales_return_lines` | Restocked or damaged returned goods |
| `ledger_entries` | Double-entry accounting entries |

Money and costs are integer cents. Stock is derived from movements; there is no
mutable on-hand column.

## Invariants

- On-hand = sum of `stock_movements.quantity_delta` for a product and warehouse.
- Available = on-hand minus active reservations. It cannot go negative.
- Receipts and invoices lock product rows before reading stock balances, so
  concurrent reservations and issues serialize per product.
- An order can be reserved only when stock and retailer credit are sufficient.
- Credit exposure = unpaid invoice balances after returns plus reserved order
  value. A fully returned paid invoice becomes a customer credit.
- Invoice postings balance accounts receivable against revenue.
- Invoice lines snapshot quantity, selling price, and total issue cost. This is
  the source for exact partial-return refunds and cost reversals.
- COGS postings balance cost of goods sold against inventory.
- Stock valuation = sum of signed movement costs and must equal the inventory
  ledger balance.
- Stock receipts post inventory against accounts payable; negative adjustments
  post inventory shrinkage against inventory.
- Receiving a purchase order posts each line at its captured unit cost, locks
  the purchase order and products, and can happen only once. Cancelling is only
  allowed before receipt.
- Returns cannot exceed the remaining invoiced quantity. Restocked returns debit
  inventory and credit COGS at the original invoice cost; damaged returns reduce
  the receivable but leave the cost in COGS.

See [ADR-0002](adr/0002-core-money-and-stock.md),
[ADR-0003](adr/0003-inventory-costing.md), and
[ADR-0004](adr/0004-sales-returns.md) for the accounting decisions. See
[ADR-0005](adr/0005-server-rendered-console.md) for the console architecture.

## Browser console

`internal/web` is a server-rendered HTML console mounted at `/`:

| Route | Purpose |
|---|---|
| `GET /` | Stock, receivables, ledger, and recent invoice overview |
| `GET /catalog` | Product, retailer, warehouse, and supplier setup with create/edit forms |
| `POST /catalog/{products,retailers,warehouses,suppliers}` | Create a catalog record |
| `POST /catalog/{products,retailers,warehouses,suppliers}/{id}` | Update a catalog record |
| `GET /purchasing`, `POST /purchasing` | Purchase order register and creation |
| `GET /purchasing/{id}` | Purchase order detail with receive and cancel actions |
| `POST /purchasing/{id}/receive` | Receive all lines into stock |
| `POST /purchasing/{id}/cancel` | Cancel an unreceived purchase order |
| `GET /inventory` | Stock levels, valuation, and receive/adjust form |
| `POST /inventory/receive` | Post a receipt or negative adjustment |
| `GET /orders`, `POST /orders` | Order register and draft creation |
| `GET /orders/{id}` | Order detail with reserve, cancel, and invoice actions |
| `GET /invoices` | Receivables list |
| `GET /invoices/{id}` | Invoice lines, payment form, return form, and histories |
| `POST /invoices/{id}/payments` | Record a payment |
| `POST /invoices/{id}/returns` | Post a return |
| `GET /ledger` | Trial balance, inventory reconciliation, and credit watch |

The console uses the default tenant, calls the same service layer as the API,
and renders with `html/template`. Mutations use POST and redirect back to a GET
page with a flash or error message. A small static script adds or removes order
lines; the core flows work without JavaScript.

## Procurement flow

1. `POST /api/purchase-orders` captures supplier, warehouse, products,
   quantities, and unit costs as an `ordered` purchase order.
2. `POST /api/purchase-orders/{id}/receive` locks the purchase order and product
   rows, writes positive stock movements, posts inventory against accounts
   payable, and marks the order `received`.
3. `POST /api/purchase-orders/{id}/cancel` cancels an unreceived order without
   touching stock or the ledger.

## Order flow

1. `POST /api/orders` snapshots product prices into a draft order.
2. `POST /api/orders/{id}/reserve` checks credit and availability, then creates
   active reservations. Product rows are locked to prevent oversell.
3. `POST /api/orders/{id}/invoice` computes weighted-average issue cost, writes
   negative stock movements, marks reservations fulfilled, and posts revenue and
   COGS.
4. `POST /api/invoices/{id}/returns` validates remaining quantities, credits
   revenue against accounts receivable, and reverses COGS for restocked goods.
5. `POST /api/invoices/{id}/payments` applies cash and reduces accounts
   receivable.
6. `GET /api/ledger/trial-balance` and `GET /api/inventory/valuation` expose
   reconciliation checks.

## Errors

Domain errors are sentinel values: `ErrNotFound`, `ErrConflict`, and
`ErrInvalid`. `httpapi` maps them to `404`, `409`, and `400`. Unknown errors are
logged and returned as `500` without internal details.

## Security boundaries

There is no authentication or database row-level security in this MVP. The
`X-Tenant-ID` header is trusted and the console is pinned to the default tenant.
This is only suitable for local development or a trusted network. Do not expose
the API or console to untrusted clients. See [SECURITY.md](../SECURITY.md).

## Deferred

Partial purchase receipts, picking and dispatch, multi-warehouse allocation,
lots/FIFO, tax, period close, cash refunds and exchanges, aging and dunning,
public API credentials, webhooks, authentication, tenant switching, and desktop
or mobile clients are outside this slice. The modular package boundaries leave room to add
them without splitting the service prematurely.
