# ADR-0003: Weighted-average inventory costing and COGS

- Status: Accepted
- Date: 2026-10-20

## Context

Distro records quantities but invoices do not yet relieve inventory or recognize
cost of goods sold. The ledger therefore balances without representing the
economic cost of goods shipped, and there is no way to reconcile stock value to
the accounting records.

The MVP has one warehouse, no purchase orders, and no lots or serial numbers.
Receipts arrive with a unit cost; invoices consume stock without tracking which
receipt supplied it. Finance needs a simple, auditable valuation that remains
correct under concurrent reservations and invoices.

## Decision

Use a moving weighted-average cost per tenant, product, and warehouse.

- Store `unit_cost_cents` and signed `total_cost_cents` on every stock movement.
  Receipts carry positive cost; issues carry negative cost.
- A receipt's total cost is `quantity * unit_cost_cents`. It posts
  `Dr inventory / Cr accounts_payable`.
- An issue is valued from the current observed balance:
  `round_half_up(value_cents * quantity / on_hand)`. A full liquidation consumes
  the remaining value exactly so rounding cannot strand inventory cents.
- Invoicing posts one COGS entry pair per invoice:
  `Dr cost_of_goods_sold / Cr inventory`.
- Negative manual adjustments use the same weighted-average value and post
  `Dr inventory_shrinkage / Cr inventory`. They cannot consume reserved stock.
- Receipts and invoices lock product rows in stable ID order before reading the
  observed balance. This serializes competing issues for the same product while
  preserving reservation oversell protection.
- `GET /api/inventory/valuation` reports stock value, the inventory ledger
  balance, posted COGS, and a `balanced` flag. Stock value is the sum of signed
  movement costs; it must equal the inventory account balance.

## Consequences

Inventory value and COGS now reconcile to the ledger after receipts,
adjustments, invoices, and payments. The implementation stays small and works
with the existing append-only movement model.

Moving weighted average is not suitable for lot-specific traceability, FIFO tax
reporting, or landed-cost capitalization. Backdated receipts would change
historical averages, so receipt dates are not modeled yet. A full liquidation is
the only path that guarantees exact zero value; partial issues may leave a
one-cent rounding residue by design.

The valuation endpoint performs a full stock scan per tenant. That is acceptable
at MVP scale but will need aggregation or snapshots as volume grows.

## Alternatives considered

- **FIFO/lots:** more accurate for physical traceability but requires lot
  capture, consumption ordering, and materially more schema and UI.
- **Specific identification:** rejected because the MVP does not track serial
  numbers or receipt-to-order links.
- **Standard cost with variances:** rejected because there is no costing policy,
  revaluation process, or period-close workflow yet.
- **Quantity-only ledger:** rejected because it cannot support gross margin,
  inventory reconciliation, or financial reporting.
