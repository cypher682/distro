# ADR-0001: MVP scope and deferred infrastructure

- Status: Accepted
- Date: 2026-10-20

## Context

The full product specification describes a distributor platform with multiple
services, route planning, offline field sales, channel intake, and a full
operations model. That is a multi-month product. The immediate goal is a
functional MVP that proves the core order-to-cash flow.

## Decision

Build a Go modular monolith with a single API process and PostgreSQL. The first
slice covers tenancy, catalog, stock movements and reservations, the order
lifecycle, invoices, a double-entry ledger, and credit exposure.

Use Docker Compose for local development and a single-host deployment when
needed. Defer Kubernetes, Terraform, multi-region deployment, event streaming,
service extraction, and a full observability stack.

## Consequences

- The core invariants (no oversell, balanced ledger, credit limit) can be
  proven with focused integration tests.
- The MVP has no high availability, load-testing program, or production
  operating history.
- Service boundaries remain modular so extraction is possible later.
- The schema is created from raw SQL on first start. Move to a migration tool
  before multiple environments exist.

## Alternatives considered

- Start with the full service map: rejected because it delays a working
  end-to-end flow.
- Use an existing ERP or commerce engine: rejected because reservation,
  ledger, and credit behavior are the core signals.
