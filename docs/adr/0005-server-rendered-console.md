# ADR-0005: Server-rendered operations console

- Status: Accepted
- Date: 2026-10-20

## Context

The API covers the order-to-cash flow, but operators need a browser interface
for stock, orders, invoices, payments, returns, and ledger checks. A desktop
toolkit would add a separate runtime and packaging path. A single-page app would
add a Node build, a second deployment, and duplicate routing before the product
shape is settled.

The current deployment is one Go binary behind Compose, and the service layer
already owns every mutation and invariant.

## Decision

- Serve the console from the same Go process, with `/api/` and `/health*`
  routed to the JSON API and every other path routed to `internal/web`.
- Render HTML with `html/template`. Embed templates and static assets in the
  binary and serve them directly; do not add a Node or SPA build.
- Reuse `core.Service` for reads and mutations. The console owns presentation,
  form parsing, redirects, and operator messages, not business rules.
- Use POST/Redirect/GET for mutations. Flash and error messages travel in the
  query string and are rendered once.
- Pin the console to `httpapi.DefaultTenantID`. Add tenant selection only with
  authentication and authorization.
- Keep the main flows usable without JavaScript. The only script adds and
  removes order-line rows.
- Use the warehouse-manifest visual system: dense registers, a dark navigation
  rail, tabular numbers, and orange for primary actions, active navigation, and
  focus.

## Consequences

- The console ships with the existing binary and Docker image. There is no
  separate frontend dependency, build, or host.
- Templates render on the server, so page loads reflect committed state and
  tests can exercise full flows through the HTTP handler.
- Every new operator screen must be added to the Go router and template set.
  Rich client-side interactions will need a deliberate upgrade.
- The console is single-tenant and unauthenticated, consistent with the current
  API boundary. It must not be exposed to untrusted networks.
- Lists are capped at the latest 100 orders and invoices. Pagination and search
  remain deferred.

## Alternatives considered

- **React or Next.js SPA:** rejected for this slice because it adds a build and
  deployment surface before the interaction model is stable.
- **Wails, Fyne, or Gio desktop app:** rejected because operators need a shared
  multi-user browser surface and the API already supports it.
- **Admin panel generator:** rejected because the domain actions and accounting
  language are specific, and generated screens would obscure the workflow.
- **Server-side console with a templating library other than `html/template`:**
  rejected to avoid a dependency for a small template set.
