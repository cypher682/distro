# ADR 0002: Core money and stock invariants

Status: accepted; costing and returns extended by ADR-0003 and ADR-0004
Date: 2026-10-20

## Context

The Distro MVP must prove that stock cannot oversell, credit exposure is
visible before invoicing, and every financial posting balances. The platform
will later grow into multi-warehouse inventory, costing, tax, and collections;
the first slice needs explicit invariants that survive that growth.

## Decision

- Store money as integer cents (`bigint` in PostgreSQL, `int64` in Go). Do not
  use floating point for prices, invoices, payments, or ledger amounts.
- Keep stock as an append-only `stock_movements` ledger. On-hand is the sum of
  movements for a product and warehouse. Active reservations are counted
  separately; available is on-hand minus active reservations.
- Reserve stock inside one database transaction that locks the order, the
  retailer, and each product row. Lock products in a stable order to avoid
  deadlocks. Reject the whole reservation if any line is short.
- Represent reservations as `active`, `released`, or `fulfilled`. Cancelling a
  reserved order releases its active reservations.
- Treat invoicing as the MVP dispatch proxy: invoicing writes one negative stock
  movement per fulfilled reservation and marks the reservation fulfilled. This
  prevents invoiced stock from becoming available again.
- Post the ledger with balanced entry groups. Invoicing debits
  `accounts_receivable` and credits `revenue`. Payments debit `cash` and credit
  `accounts_receivable`. Entries are append-only; corrections will be reversals.
- Define credit exposure as open invoice balances plus reserved order totals.
  A reservation is rejected when exposure plus the new order would exceed the
  retailer credit limit.
- Use one default warehouse per tenant in this slice. Multi-warehouse
  allocation is deferred until order lines carry a warehouse or allocation
  policy.

## Consequences

- The trial balance must always net to zero; tests assert this after invoicing
  and payments.
- Stock and reservations can be reconciled from their ledgers at any time.
- Costing, tax, inventory valuation, and dispatch remain deferred. The invoice
  stock movement is an explicit MVP shortcut, not the final fulfillment model.
- Authentication, tenant RLS policies, and per-tenant authorization are not in
  this slice; the API currently trusts the `X-Tenant-ID` header and defaults to
  the seeded demo tenant.
