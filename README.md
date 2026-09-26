# Ford Retain: backend

Go API for Ford Retain: new features for Ford's existing customer platform,
aimed at keeping owners with Ford. The platform already sends owners
maintenance alerts on WhatsApp; Ford Retain adds a place where they can see
their vehicles' health, alerts and telemetry, and book and cancel maintenance
at nearby dealerships.

This repository is the **back-end**: a Go 1.27 modular monolith using the
standard library (plus the `pgx` Postgres driver). The Next.js app lives in
**ford-retain-frontend** and talks to this API from its server.

> **Status:** complete prototype. Without Ford's platform the API serves demo
> data: two owners, three pickups (2022 F-150 Lariat PowerBoost, 2023 F-150
> Lightning, 2024 Ranger Raptor) with telemetry, alerts and appointments, plus
> 24 dealerships. See `internal/seed/seed.go`.

### Demo accounts

| Email               | Password           | Vehicles                            |
|---------------------|--------------------|-------------------------------------|
| `ana@example.com`   | `ford-retain-demo` | F-150 Lariat PowerBoost, F-150 Lightning |
| `bruno@example.com` | `ford-retain-demo` | Ranger Raptor                       |

## Quick start

Only Docker is required.

```sh
make dev          # or: docker compose up
```

- API: http://localhost:8080 (health check: `/healthz`)
- Postgres on host port 5433; the mock Ford API runs inside the compose network.

`make dev` creates the `ford-retain` Docker network. To run the whole app,
start this repo first, then `make dev` in **ford-retain-frontend**, which joins
the same network, reaches the API as `http://backend:8080`, and serves the app on
http://localhost:3000.

### Production build

```sh
make prod         # or: docker compose -f compose.prod.yaml up --build
```

`Dockerfile`: static binary on distroless, running as non-root (about 16 MB).

## Architecture

```
ford-retain-frontend (Next.js server) ──► this API ──┬──► Ford platform APIs (vehicles, telemetry, alerts, dealers)
                                                     └──► Postgres: this service's own data (appointments)
```

## Ford platform integration

Vehicles, telemetry, the WhatsApp alert history and the dealer directory
belong to Ford's existing platform, so the API reads them from there instead of
owning them:

- Each module depends on a small `Source` interface (`vehicles.Source`,
  `telemetry.Source`, `notifications.Source`). Two implementations exist: the
  in-memory stores (seed data) and `internal/fordapi.Client`.
- `FORD_API_URL` picks the mode (`cmd/api/sources.go`): set, the API calls
  Ford; unset, it serves seed data from memory. `make dev` runs the Ford path
  against a local mock.
- **Contract:** Ford's real endpoints aren't available to this project, so the
  package doc of `internal/fordapi` spells out the contract we assume. The
  client and the mock server (`fordapi.NewMock`, run by `cmd/fordmock`)
  implement that same contract, and the client tests run against the mock.
  When Ford's real contract is known, change both together.
- **Failures:** if Ford is unreachable or errors, the API answers `502` ("a Ford
  service is unavailable") and logs the cause. The front-end shows its retry screen,
  and it recovers on its own when Ford is back.
- **Dealer directory:** reference data. It's loaded at startup and refreshed
  every 10 minutes (retried every 15 s until the first load succeeds), so
  booking and availability never wait on Ford and keep working from the last
  copy if Ford goes down.
- Vehicles, telemetry and alerts are fetched per request. The ownership check
  runs on every vehicle route, so a short-lived cache of each owner's garage is
  the obvious next optimisation.

## Appointments storage

Appointments are this service's own data, so it stores them itself.
`DATABASE_URL` selects Postgres; without it they live in memory.

- `appointments.Repository` has two implementations: `MemoryRepo` and
  `PostgresRepo`. One shared contract test runs against both, including 10
  concurrent bookings racing for a 2-bay slot.
- **No overbooking, even with several API instances:** `PostgresRepo.Book`
  counts and inserts in one transaction holding an advisory lock for that
  dealership and hour.
- **Migrations:** each module embeds its own `migrations/*.sql`, and
  `platform/postgres.Migrate` applies them at startup, once each, recorded per
  module in `schema_migrations`.
- **Demo data:** `SEED_APPOINTMENTS=true` loads the demo bookings, only into an
  empty table. `make dev` sets it; never set it in production.
- **Tests:** Postgres tests run when `TEST_DATABASE_URL` is set, each in a
  throwaway schema. `make test` starts the compose database and runs the
  tests against it, and CI uses a Postgres service.

## Authentication

> Accounts belong to Ford's existing platform. The sign-in here (users,
> passwords, sessions) is a **development stand-in** so the features can be
> built and tested end to end. In production, the `auth` module's
> `RequireUser` should validate Ford's identity tokens instead. Sign-up and
> password reset are deliberately out of scope: Ford's identity system owns
> them.

```
browser ──cookie──► Next.js ──Authorization: Bearer──► Go API
```

- **Sign-in:** `POST /api/auth/login` checks the password (PBKDF2-SHA256,
  600,000 iterations, per-user salt; standard library `crypto/pbkdf2`) and
  returns an opaque random token. The API keeps only the token's SHA-256, with
  a 7-day expiry. Failed logins take the same time whether or not the email
  exists.
- **Lockout:** 5 failed logins for an email within 15 minutes lock it until the
  window ends (`429` with `Retry-After`; the password isn't even checked while
  locked). It's per email, not per IP, because the API only sees the Next.js
  server's address. Unknown emails lock the same way, so lockouts don't reveal
  which accounts exist.
- **Clients:** the front-end (ford-retain-frontend) keeps the token in an
  HTTP-only cookie and sends it as a bearer token from its server; the browser
  never talks to this API directly.
- **Who can see what** is decided only by the Go API, in three router layers
  (`cmd/api/routes.go`):
  - public: `/healthz`, `POST /api/auth/login`
  - signed in (`auth.RequireUser`): everything else under `/api/`
  - owner only (`vehicles.RequireOwner`): every `/api/vehicles/{vin}/…` route.
    Another owner's VIN gets `404`, so the API doesn't reveal that it exists.
- Sessions live in memory, so restarting the API signs everyone out.

## Code layout: modular monolith

```
├── cmd/api/
│   ├── main.go          # config, HTTP server, graceful shutdown
│   ├── routes.go        # wires modules into public / signed-in / owner-only layers
│   └── sources.go       # data from seed data or Ford's platform (FORD_API_URL)
├── cmd/fordmock/        # local stand-in for Ford's platform APIs
└── internal/
    ├── fordapi/         # Ford platform client + mock server (one contract)
    ├── seed/            # demo data (users, vehicles, telemetry, alerts, dealers, bookings)
    ├── platform/httpx/  # shared JSON / error / CORS / request-log helpers
    ├── platform/postgres/ # connect + per-module SQL migrations
    ├── platform/identity/ # authenticated user ID in the request context
    ├── auth/            # users, password hashing, sessions, RequireUser
    ├── vehicles/        # each owner's garage: VIN, model, paint; RequireOwner
    ├── notifications/   # maintenance alerts (WhatsApp history)
    ├── telemetry/       # component health by body zone, battery life + history
    ├── dealerships/     # directory with rating/hours/services, nearest-first search
    └── appointments/    # scheduling: slots, booking, cancel; memory or Postgres repo
```

Rules each module follows:

- A module is one package under `internal/`. It owns its types, its storage and
  its routes (`Handler.Register(mux)`). `http.go` holds the HTTP layer; the other
  files hold the domain logic.
- Modules do not reach into each other's data. When one needs another, it
  declares a small interface or function type on its side. Example:
  `appointments` defines `HoursLookup`, and `routes.go` adapts the dealerships
  store to it.
- `routes.go` is the only place that knows about every module.
- Modules that scope data by user read the ID from `platform/identity`, not
  from the `auth` package.
- Data Ford owns is read through `Source` interfaces (seed data or
  `internal/fordapi`); this service's own data goes through a `Repository`
  (in memory or Postgres). `cmd/api` picks the implementations.
- Domain errors are sentinel values (`ErrInvalid`, `ErrNotFound`) that handlers
  map to status codes with `errors.Is`.

Routing uses the standard library `http.ServeMux` with method + path patterns
(`GET /api/vehicles/{vin}/telemetry`). No framework. Every request is logged
with `log/slog` (method, path, status, duration).

### Adding a module

1. Create `internal/<name>/` with the domain file and an `http.go` exposing
   `NewHandler(...)` and `Register(*http.ServeMux)`.
2. Register it in `cmd/api/routes.go`, on the `owned` mux if its routes live
   under `/api/vehicles/{vin}/`, otherwise on `signed`.
3. Add tests next to the code: `<name>_test.go` for the domain,
   `http_test.go` for routes (see any existing module).

## API

All responses are JSON. Errors look like `{"error": "message"}`. Every route
except `/healthz` and login needs `Authorization: Bearer <token>` (`401`
otherwise), and `/api/vehicles/{vin}/…` routes answer `404` for vehicles the
user doesn't own.

| Method | Path                                       | Description |
|--------|--------------------------------------------|-------------|
| GET    | `/healthz`                                 | Liveness check (public) |
| POST   | `/api/auth/login`                          | `{"email","password"}` → `{"token","user"}`; `401` if wrong, `429` + `Retry-After` when locked (public) |
| GET    | `/api/auth/me`                             | The signed-in user |
| POST   | `/api/auth/logout`                         | Ends the session: `204` |
| GET    | `/api/vehicles`                            | The signed-in user's vehicles (VIN, model, paint); the first is the default |
| GET    | `/api/vehicles/{vin}`                      | One vehicle; `404` if unknown |
| GET    | `/api/vehicles/{vin}/notifications`        | Notifications for a vehicle |
| PATCH  | `/api/vehicles/{vin}/notifications/{id}/read` | Mark a notification as read |
| GET    | `/api/vehicles/{vin}/telemetry`            | Latest snapshot: model, mileage, battery (with 12-month history), components with body zone |
| GET    | `/api/dealerships?lat=&lng=`               | Dealerships with rating, hours and services; nearest first (with `distanceKm`) when lat/lng given |
| GET    | `/api/vehicles/{vin}/appointments?when=`   | Appointments; `when=upcoming` (soonest first) or `past` (most recent first) |
| GET    | `/api/dealerships/{id}/availability?from=&days=` | Bookable hourly slots per day (`from` YYYY-MM-DD, default today; `days` 1–31, default 14) |
| POST   | `/api/vehicles/{vin}/appointments`         | Create an appointment (see below) |
| DELETE | `/api/vehicles/{vin}/appointments/{id}`    | Cancel an upcoming appointment: `204`; `404` if unknown; `422` if already in the past |

Create appointment body:

```json
{ "dealershipId": "d1", "scheduledAt": "2027-01-10T10:00:00Z", "service": "Brake pads", "notes": "" }
```

`scheduledAt` must be the start of a free slot from the availability route.
Returns `201` on success, `400` for malformed JSON, and `422` when validation
fails: missing service, a time in the past, unknown dealership, a time that
isn't a slot (off the hour, outside opening hours, Sunday), or a full slot.

**Slots:** each dealership has opening hours in its own time zone (all seeded
ones are in São Paulo), split into one-hour slots, with a number of service
bays per slot (2 or 3 in the seed). Opening hours live in the `dealerships`
module. `appointments` reads them through a lookup function wired in
`routes.go`, and checks capacity and inserts under one lock, so two bookings
can't both take the last bay. The API binary embeds `time/tzdata`, because the
distroless image has no time-zone database.

## Configuration

| Variable            | Default                 | Purpose |
|---------------------|-------------------------|---------|
| `ADDR`              | `:8080`                 | Listen address |
| `ALLOWED_ORIGIN`    | `http://localhost:3000` | CORS origin |
| `DATABASE_URL`      | unset (in memory)       | Postgres for appointments; `make dev` uses the compose database |
| `SEED_APPOINTMENTS` | unset                   | `true` loads demo bookings into an empty table (dev only) |
| `FORD_API_URL`      | unset (seed data)       | Ford platform API base URL; `make dev` points it at the mock |
| `FORD_API_KEY`      | —                       | Service credential sent to Ford as a bearer token |

## Development

```sh
make test   # gofmt, go vet, go test (with Postgres), in Docker, the same checks as CI
make fmt    # gofmt -w
make build  # production image
```

`make test` runs the Go toolchain in a `golang:1.27-alpine` container on the
`ford-retain` network, sharing the module cache with `make dev`. CI
(`.github/workflows/ci.yml`) runs the same checks, with a Postgres service, on
pushes to `master` and `develop` and on pull requests.

Branches follow git flow: `master` (releases), `develop` (integration),
`feat/*` and `bugfix/*` for work. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Next steps

These are integrations with Ford's existing platform rather than things to
rebuild here:

- **Identity:** replace the stand-in login with Ford's identity provider
  (validate its tokens in `auth.RequireUser`; vehicle ownership comes from Ford).
- **Ford's real API contract:** the adapters exist (`internal/fordapi`). Swap
  the assumed contract for Ford's real one in the client and mock, and cache
  garage lookups.
- **Dealer scheduling systems:** if Ford's dealers need bookings in their own
  systems, sync them from the Postgres table (or make `Repository` an adapter).
