# Security

## Reporting

Report vulnerabilities through GitHub's private vulnerability reporting for this
repository. Do not open a public issue for an unpatched vulnerability.

Include the affected endpoint or package, reproduction steps, impact, and any
suggested fix. Do not include real customer data, credentials, or production
connection strings.

## Supported versions

This repository is an MVP. Security fixes are applied to the `main` branch; no
long-term support releases are maintained.

## Security model

The current slice is designed for local development and trusted environments.

- There is no authentication or authorization.
- `X-Tenant-ID` is trusted and can be supplied by any client. It is not a
  security boundary.
- PostgreSQL has no row-level security. Tenant isolation relies on query filters.
- The API has no rate limiting, CSRF protection, or public TLS termination.
- Compose credentials are development defaults and must not be reused.
- Financial behavior is covered by tests but has not been independently audited.

Do not expose this API to untrusted clients. Before a shared deployment, add
authenticated tenant context, database RLS, TLS, rate limiting, secret
management, audit logging, and a security review of the ledger and stock paths.

## Secret handling

- Keep real credentials in environment variables or a secret manager, never in
  the repository.
- `.env` files are ignored; `.env.example` contains placeholders and local
  development defaults only.
- Rotate a credential immediately if it is committed. Removing it from the
  latest commit is not sufficient because Git history retains it.
