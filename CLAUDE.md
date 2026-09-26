# Ford Retain: backend

## Project brief

Original brief (verbatim):

> The goal is to retain the customer with Ford; to achieve this, the team decided to evolve the existing platform. Users receive WhatsApp alerts regarding their vehicle's maintenance needs. Within the platform, users can view a summary of notifications concerning required repairs, as well as vehicle telemetry data—such as component health and estimated remaining battery life. The platform also allows users to schedule future maintenance and locate nearby Ford dealerships on a map.
>
> you need to create:
> 1. A front-end using React with Next.js
> 2. A back-end in Go (Golang)—structured as a modular monolith following Go best practices
> 3. A README.md file documenting everything for other developers
>
>
> Don't overengineer.

It was split into two repositories: **ford-retain-backend** (this one, the Go
API) and **ford-retain-frontend** (Next.js). The brief's README requirement
applies to each.

## Working in this repo

- Read `README.md` for layout, API and commands; keep it up to date for other
  developers.
- Keep it simple: standard library in Go; the only extra dependency is the
  `pgx` Postgres driver. No new dependencies without a clear need.
- This project adds features to Ford's existing platform: integrate with what
  Ford already owns (identity, vehicles, telemetry, WhatsApp alerts, dealers)
  through `internal/fordapi` rather than rebuilding it.
- Each module under `internal/<module>/` owns its types, store and routes;
  cross-module needs go through a small interface wired in `cmd/api/routes.go`.
- API changes affect ford-retain-frontend (`src/lib/api.ts` there); mark
  breaking changes with `!` and a `BREAKING CHANGE:` footer.
- Everything runs in Docker (`make dev`, `make test`); `make test` runs what
  CI runs, so run it before committing.
