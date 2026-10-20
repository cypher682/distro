# Distro

MVP scaffold for the distributor and wholesale commerce platform.

## MVP scope

- multi-tenant catalog and pricing, with console setup for products,
  retailers, and warehouses
- append-only stock movements and reservations
- order lifecycle and credit exposure
- weighted-average inventory costing and cost of goods sold
- sales returns with restock or damage handling
- invoice and double-entry ledger posting
- server-rendered browser console for stock, orders, invoices, returns, and ledger
- JSON API for ordering and operations stock flows

Deferred: procurement, picking and dispatch, routing, driver app, WhatsApp
intake, offline rep sync, aging and dunning, analytics, public API, webhooks,
and native desktop or mobile clients.

## Run

```bash
docker compose up --build
```

API and console: http://localhost:8081

The browser console is a single-tenant local operations view. It has no
authentication and trusts the seeded demo tenant; do not expose it to an
untrusted network.

Health checks:

```bash
curl http://localhost:8081/health
curl http://localhost:8081/health/db
```

The database is available on port 5434. The schema and synthetic seed data are
applied on first start. Reset with `make down`. There is no incremental
migration runner yet, so existing local volumes must be reset after a schema
change.

## Demo flow

Open http://localhost:8081/ to set up products, retailers, and warehouses, then
receive stock, create and reserve orders, issue invoices, record payments, and
post returns in the console. Catalog records can also be created from the
Catalog page without SQL. The same flow is available over the API:

```bash
# Catalog and stock
curl -s http://localhost:8081/api/products | jq
curl -s http://localhost:8081/api/stock | jq

# Create, reserve, and invoice an order
ORDER=$(curl -s -X POST http://localhost:8081/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"retailer_id":"00000000-0000-0000-0000-000000000020","lines":[{"product_id":"00000000-0000-0000-0000-000000000101","quantity":2}]}')
ORDER_ID=$(echo "$ORDER" | jq -r .id)

curl -s -X POST http://localhost:8081/api/orders/$ORDER_ID/reserve | jq
curl -s http://localhost:8081/api/retailers/00000000-0000-0000-0000-000000000020/credit | jq

INVOICE=$(curl -s -X POST http://localhost:8081/api/orders/$ORDER_ID/invoice)
INVOICE_ID=$(echo "$INVOICE" | jq -r .id)

# Return one unit to stock
curl -s -X POST http://localhost:8081/api/invoices/$INVOICE_ID/returns \
  -H 'Content-Type: application/json' \
  -d '{"lines":[{"product_id":"00000000-0000-0000-0000-000000000101","quantity":1}]}' | jq

# Pay part of the invoice, then check the ledger
curl -s -X POST http://localhost:8081/api/invoices/$INVOICE_ID/payments \
  -H 'Content-Type: application/json' \
  -d '{"amount_cents":500}' | jq

curl -s http://localhost:8081/api/ledger/trial-balance | jq

# Inventory value must equal the inventory ledger balance
curl -s http://localhost:8081/api/inventory/valuation | jq
```

The trial balance response includes a `balanced` flag. It must stay `true` after
every stock receipt, adjustment, invoice, return, and payment. The valuation
response reconciles on-hand stock value, the inventory account, and posted COGS.

## API

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/products` | List catalog products |
| POST | `/api/products` | Create a product |
| PUT | `/api/products/{id}` | Update a product |
| GET | `/api/warehouses` | List stock locations |
| POST | `/api/warehouses` | Create a warehouse |
| PUT | `/api/warehouses/{id}` | Rename a warehouse |
| GET | `/api/retailers` | List retailers and credit limits |
| POST | `/api/retailers` | Create a retailer |
| PUT | `/api/retailers/{id}` | Update a retailer |
| GET | `/api/stock` | On-hand, reserved, available, and valued stock |
| GET | `/api/inventory/valuation` | Inventory value, COGS, and ledger reconciliation |
| POST | `/api/stock/receive` | Receive stock with unit cost or apply a stock adjustment |
| POST | `/api/orders` | Create a draft order |
| GET | `/api/orders` | List the latest 100 orders with lines |
| GET | `/api/orders/{id}` | Read an order |
| POST | `/api/orders/{id}/reserve` | Reserve stock and check credit |
| POST | `/api/orders/{id}/cancel` | Cancel a draft or reserved order |
| POST | `/api/orders/{id}/invoice` | Invoice a reserved order and post the ledger |
| GET | `/api/retailers/{id}/credit` | Credit limit, exposure, and availability |
| GET | `/api/invoices` | List the latest 100 invoices with returned totals |
| GET | `/api/invoices/{id}` | Read an invoice with lines, returns, and payments |
| POST | `/api/invoices/{id}/payments` | Record a payment |
| POST | `/api/invoices/{id}/returns` | Return invoiced goods and post the credit |
| GET | `/api/ledger/trial-balance` | Account balances and balance check |

Requests use `X-Tenant-ID`; it defaults to the seeded demo tenant.

Catalog write bodies use `sku`, `name`, `unit`, and `price_cents` for products,
`name` and `credit_limit_cents` for retailers, and `name` for warehouses. SKUs
are unique per tenant.

Stock receipts use a positive `quantity` and `unit_cost_cents`. Negative
quantities are adjustments: they are valued at the current weighted average,
posted to inventory shrinkage, and cannot consume reserved stock.

Returns default to `"restock": true`: goods come back at the original invoice
cost, inventory is restored, and COGS is reversed. Set `"restock": false` for
damaged goods; the receivable is still reduced but the cost stays in COGS. Cash
refunds are handled outside the API.

## Concurrency check

With the stack running:

```bash
./scripts/concurrency-demo.sh
```

The seeded RICE-25 product has 100 units and the demo retailer has enough
credit. The default run fires 150 parallel order-and-reserve attempts and
prints the HTTP status counts. A healthy run is 100 `200` responses and 50
`409` conflicts, with zero oversell.

## Test

```bash
make test
make vet
```

Or, without Make:

```bash
docker compose run --rm dev go test ./...
```

## Reset local data

```bash
make down
```

## Documentation

- [Architecture](docs/architecture.md): components, data model, and invariants
- [Operations](docs/operations.md): reset, verification, and troubleshooting
- [Contributing](CONTRIBUTING.md): development workflow and database changes
- [Security](SECURITY.md): threat model and reporting
- [ADRs](docs/adr/): durable design decisions
- [Changelog](CHANGELOG.md): released changes

## Layout

```text
cmd/api/                HTTP entry point
internal/config/        environment configuration
internal/core/          domain types, service, memory and PostgreSQL stores
internal/httpapi/       JSON API router, handlers, health checks
internal/web/           server-rendered console, templates, and static assets
db/migrations/          schema migrations
db/seed.sql             synthetic demo data
db/demo-order.json      sample order request for concurrency checks
scripts/                local verification scripts
docs/adr/               decisions
```

## Known MVP limitations

- One default warehouse per tenant; multi-warehouse allocation is deferred.
- Invoicing is the dispatch proxy and writes stock and COGS movements. Picking
  and dispatch will replace this.
- Costing is moving weighted average only. Landed costs, FIFO/lots, and period
  close are deferred.
- Returns create a receivable credit but do not issue cash refunds or exchange
  orders. A paid invoice can therefore carry a customer credit balance.
- No tax yet.
- No authentication or database RLS yet; tenant isolation is enforced by query
  filters and the `X-Tenant-ID` header. The console is fixed to the default
  tenant and has no tenant switcher.
- Order and invoice lists are capped at the latest 100 records. Pagination is
  deferred.
