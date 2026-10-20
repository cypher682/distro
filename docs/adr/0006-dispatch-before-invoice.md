# ADR-0006: Dispatch before invoice

- Status: Accepted
- Date: 2026-10-20

## Context

Invoicing previously acted as the dispatch proxy: it issued stock, fulfilled
reservations, recognized COGS, and posted revenue in one transaction. That made
it impossible to separate warehouse work from billing, and it recognized cost
when paperwork was issued rather than when goods left the warehouse.

A delivery note is the operational record of what was picked and dispatched.
Invoicing should bill a dispatched order and reuse the costs captured at
dispatch, not recompute them from a later weighted average.

## Decision

- Add `delivery_notes` and `delivery_note_lines`. A note belongs to one order and
  starts in `picking`.
- Create notes only from a reserved order. Copy active reservations and order
  line prices into the note. Keep the order `reserved`; do not move stock or post
  to the ledger while picking.
- Allow at most one non-cancelled note per order with a partial unique index.
- Dispatch is atomic: lock the note, order, and product rows; compute
  weighted-average issue cost; write negative stock movements; mark reservations
  fulfilled; post `Dr cost_of_goods_sold / Cr inventory`; and mark the note and
  order `dispatched`.
- Cancelling a note is allowed only from `picking`. It preserves reservations and
  leaves the order reserved. Cancelling an order also cancels a picking note.
- Require a dispatched order for invoicing. Copy delivery-note quantities,
  prices, and issue costs into immutable invoice lines. Post only
  `Dr accounts_receivable / Cr revenue`.
- Keep `reserved` and `dispatched` orders in credit exposure so credit is not
  released between dispatch and invoice.
- Deliveries are all-or-nothing; partial dispatch and split deliveries are
  deferred.

## Consequences

- Warehouse work and billing become separate operator steps with separate audit
  records and accounting events.
- COGS is recognized at physical dispatch, matching the inventory movement.
- Invoice cost snapshots remain exact for returns even if stock is received at a
  different cost after dispatch.
- An order can be stuck between dispatch and invoice; credit exposure continues
  to reflect that obligation.
- Existing orders and invoices need a schema reset in local development. There is
  no incremental migration runner.
- The API and console gain delivery-note lifecycle endpoints and screens.

## Alternatives considered

- **Keep invoicing as the dispatch proxy:** rejected because it couples billing
  timing to warehouse operations and misstates the event being recorded.
- **Dispatch without a document:** rejected because picking lines, operator
  visibility, and audit history would be lost.
- **Recalculate invoice cost from current weighted average:** rejected because
  later receipts could change the cost after goods have left, breaking exact
  return reversal.
- **Free credit when the order is dispatched:** rejected because the receivable
  does not exist yet, so exposure would temporarily disappear.
- **Partial delivery notes:** deferred until split shipments, backorders, and
  invoice allocation are designed together.
