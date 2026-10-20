# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-20

### Added

- Multi-tenant catalog, stock movements, reservations, and warehouses.
- Draft, reserved, invoiced, and cancelled order lifecycle.
- Credit limits, exposure checks, invoices, and payments.
- Double-entry ledger with trial balance reconciliation.
- Moving weighted-average inventory costing, COGS postings, shrinkage, and
  inventory valuation reconciliation.
- Sales returns with invoice cost snapshots, restock or damage handling,
  receivable credits, and exact COGS reversal.
- Server-rendered browser console for stock, orders, invoices, payments,
  returns, and ledger reconciliation.
- Console catalog setup for products, retailers, and warehouses, backed by
  create/update API endpoints.
- Read endpoints for warehouses, orders, invoice lists, and invoice detail.
- PostgreSQL and in-memory stores, HTTP API, Docker Compose stack, and CI.
- Architecture, operations, security, contribution, and ADR documentation.
