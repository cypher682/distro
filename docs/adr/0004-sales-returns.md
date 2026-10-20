# ADR-0004: Sales returns and cost reversal

- Status: Accepted
- Date: 2026-10-20

## Context

Invoices can be issued but not reversed. A retailer may return part of an
invoice because goods are damaged, short, or unwanted. Quantity-only returns are
not enough: the ledger must reduce the receivable, inventory may need to come
back, and COGS must only reverse when the goods are actually restocked.

The current invoice response exposes only totals. Reconstructing original issue
cost from the current weighted average would be wrong after later receipts.

## Decision

- Snapshot invoice lines at invoicing time: product, warehouse, quantity,
  selling price, line total, and total issue cost. Invoice lines are immutable.
- Model returns as `sales_returns` and `sales_return_lines`. Each line points to
  the original invoice line and stores quantity, refund, cost, and `restock`.
- Return quantity is cumulative and cannot exceed the invoiced quantity. The
  invoice row is locked for the return transaction, serializing competing
  returns.
- Refunds use the original invoice unit price. Return cost is allocated from the
  invoice line's remaining cost using the same round-half-up helper as inventory
  issues. A full return consumes the remaining cost exactly.
- Post `Dr sales_returns / Cr accounts_receivable` for the refund.
- If `restock` is true, write a positive stock movement at the original cost and
  post `Dr inventory / Cr cost_of_goods_sold`.
- If `restock` is false, do not write stock or reverse COGS. The goods remain
  consumed from an accounting perspective, which is appropriate for damage or
  expiry.
- Payments are limited to the remaining balance after payments and returns.
  An invoice becomes `paid` when payments plus returns cover its total.

## Consequences

- Partial and full returns now reconcile inventory, COGS, receivables, and the
  trial balance.
- Returns after a paid invoice create a customer credit balance. Cash refunds,
  credit notes, and exchange orders remain deferred.
- Invoice lines increase storage and require backfill for any pre-existing
  invoice data. The local environment is reset rather than migrated in place.
- Returned goods re-enter inventory at original cost, not current replacement
  cost. Revaluation remains out of scope.
- A damaged return cannot later be restocked through this API; a manual stock
  adjustment is required.

## Alternatives considered

- **Use current weighted-average cost for returns:** rejected because it can
  create artificial gains or losses and does not reverse the original issue.
- **Return only quantities, no cost snapshot:** rejected because COGS and
  inventory cannot be reconciled exactly.
- **Cash refunds in the same transaction:** rejected for now because payment
  provider integration and refund allocation are not part of this slice.
- **Separate credit-note documents:** deferred until tax invoicing and document
  numbering exist.
