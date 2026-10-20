# Contributing

## Setup

Requirements: Go 1.26, Docker with Compose v2, and Make. The Make targets run Go
inside the Compose tool container, so a host Go installation is optional.

```bash
docker compose up -d --build
make test
```

The API and browser console are available at http://localhost:8081. Reset local
data with `make down` before recreating the stack.

## Workflow

1. Create a focused branch from `main`.
2. Keep behavior changes with their tests and documentation.
3. Run `make fmt`, `make test`, `make vet`, and `make build`.
4. Open a pull request with the problem, the change, and the verification steps.

Do not commit `.env` files, credentials, tokens, private keys, customer data, or
production connection strings. The credentials in `compose.yaml` are local
development defaults only.

## Code conventions

- Keep domain rules in `internal/core`; keep JSON API concerns in
  `internal/httpapi` and console rendering in `internal/web`.
- Use integer cents for money and costs. Never use floating point for money.
- Return the `ErrInvalid`, `ErrConflict`, and `ErrNotFound` sentinels for domain
  errors so HTTP status mapping stays consistent.
- Keep the `MemoryStore` and `PostgresStore` behavior aligned. A rule added to one
  needs a test and the equivalent implementation in the other.
- Add comments only where the reason for a decision is not obvious from the code.

## Database changes

The project currently uses numbered SQL files applied on first database start.
When changing the schema:

1. Add a new migration under `db/migrations/`.
2. Mount it in `compose.yaml` in numeric order before the seed file.
3. Update `db/seed.sql` if the seed depends on the new column or table.
4. Reset the local volume and verify the full flow, not only unit tests.
5. Record a decision in `docs/adr/` when the change affects accounting,
   inventory, tenancy, or concurrency.

## Documentation

- `README.md`: setup, demo flow, API surface, and current limitations.
- `docs/architecture.md`: components, data model, and invariants.
- `docs/operations.md`: local operation, reset, verification, and troubleshooting.
- `docs/adr/`: durable architecture decisions.
- `CHANGELOG.md`: user-visible changes.

## Commit style

Use imperative, scoped subjects, for example:

```text
Add weighted-average inventory valuation
Fix reserved stock validation on adjustments
```

Keep formatting-only changes separate from behavior changes.
