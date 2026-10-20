# Operations

## Prerequisites

- Docker with Compose v2
- Make
- `curl` and `jq` for the demo flow

## Local stack

```bash
docker compose up --build
```

| Service | Address |
|---|---|
| API | http://localhost:8081 |
| PostgreSQL | localhost:5434 |

The Compose credentials are development-only values. They are not suitable for a
shared or public environment.

Health checks:

```bash
curl http://localhost:8081/health
curl http://localhost:8081/health/db
```

Logs:

```bash
make logs
```

Stop without deleting data:

```bash
docker compose down
```

Reset the database, schema, and seed data:

```bash
make down
docker compose up -d --build
```

`make down` removes the PostgreSQL volume. There is no incremental migration
runner yet, so a schema change requires a reset in local development.

## Seed data

The Compose stack applies migrations and seed data only when the database volume
is first created. The seed defines one tenant, one warehouse, two products, and
two retailers.

To reload seed data into an existing reset volume:

```bash
make seed
```

The seed uses fixed IDs and is not idempotent; a second run conflicts on
existing rows. Reset first when a clean state is required.

## Verification

```bash
make test
make vet
make build
docker compose run --rm dev go test -race ./internal/core
./scripts/concurrency-demo.sh
```

The concurrency script expects a clean seed. It sends 150 parallel reservation
attempts for 100 units and should report 100 `200` responses and 50 `409`
conflicts.

Reconciliation checks:

```bash
curl -s http://localhost:8081/api/ledger/trial-balance | jq
curl -s http://localhost:8081/api/inventory/valuation | jq
```

Both responses must contain `"balanced": true`. On a clean seed, inventory value
is `2100000` cents.

## Deployment notes

- Run the API as a single process behind a TLS-terminating proxy.
- Use a managed PostgreSQL instance with backups and point-in-time recovery.
- Replace all development credentials through environment variables.
- Do not expose the API without authentication, RLS, and rate limiting.
- The schema currently depends on first-start initialization. Add a migration
  tool before operating multiple environments.

## Troubleshooting

| Symptom | Check |
|---|---|
| Port already in use | Stop the process using `8081` or `5434` |
| `db` exits during startup | Run `make down` to clear a partial volume, then recreate |
| `column ... does not exist` | Reset the volume; the schema changed |
| Valuation is unbalanced | Check for manual database writes or failed transactions |
| `409` during concurrency check | Expected once stock is exhausted; restart from clean seed for the documented counts |
