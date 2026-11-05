# Secretaria Backend

Go API for the School Secretariat platform: institutions, students, enrollments, groups and users, with scoped authorization and a real-time domain-event stream. The micro-frontends that consume it live in [`mfe-communication`](https://github.com/fidelis27/mfe-communication).

A Portuguese version with the full configuration reference is in [`README.pt.md`](README.pt.md).

> **Status:** deployed prototype on Render. Hardened in several areas (see below), with the remaining gaps listed under [Known limitations](#known-limitations).

## Stack

Go 1.26, MariaDB, Supabase Auth (OIDC/JWKS), WebSocket, GitHub Actions.

## Architecture

Hexagonal layering with dependencies pointing inward:

```text
cmd/                 server, migrate, seed (composition roots)
internal/
  domain/            entities and rules: authorization, enrollment, event,
                     group, institution, student, user
  application/       use cases and the in-process event bus
  adapters/
    httpserver/      handlers, middleware, WebSocket
    mariadb/         repositories (behind domain interfaces)
    oidc/            token verification
  platform/          config, error catalog
migrations/          10 versioned SQL migrations
seeds/dev/           development-only seed data
docs/adr/            architecture decision records
```

Repositories sit behind domain interfaces, `main` only wires dependencies, and business rules (including authorization) live in the domain layer rather than in SQL or handlers.

## Authentication

Every route except `GET /health` requires a Supabase access token as a Bearer token.

- Signature, issuer, audience and expiry are verified through OIDC discovery and JWKS.
- Only `RS256` and `ES256` are accepted, with a 30 second clock leeway. Calls to the issuer and JWKS endpoint are paced to one per second.
- In production the top-level `email_verified` claim must be `true`. Editable `user_metadata` is never trusted.
- Tokens are bound to a local active user by the stable `sub` (`users.auth_sub`). Unknown or inactive users get `403`, invalid tokens get `401`.

## Authorization

Roles are superadmin, admin and member, scoped by institution and group. The rules are in `internal/domain/authorization`, and handlers apply them before running an operation.

- `super_admin` needs **both** the Supabase `app_metadata.roles` entry and the local superadmin flag.
- Group admins cannot remove or demote the last admin of a group; the API answers `409`.
- Listings (institutions, groups, events, event history) are filtered to the institutions the user can see.
- Authorization is tested at **handler level**, not only at policy level, including privilege-escalation attempts.

## Domain events

Writes publish events with `eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` and `payload`.

- Each event is saved to `audit_events` (with its institution scope, in one transaction) **before** it is broadcast. That save happens **after** the entity write has committed, in a separate transaction, so a failed save can lose the event (ADR 0001).
- `GET /events` is a WebSocket. The token travels in the `bearer.<token>` subprotocol (echoed in the handshake), only allowed origins connect, frames are capped at 1 KiB, ping/pong detects dead connections, and connections close on graceful shutdown.
- `GET /events/history` returns persisted events, scoped by institution.

## Security hardening

| Area | What is in place |
| --- | --- |
| Abuse control | Token bucket per user/IP (default 120/min) with `Retry-After` |
| Responses | `Cache-Control: no-store`, `nosniff`, `Referrer-Policy` |
| Logging | Method, path without query, status, duration and correlation ID only; never the Authorization header or a full e-mail |
| CORS | Explicit allowlist in production; loopback only outside production |
| Migrations | A test fails if a new migration contains a literal e-mail address |

[`docs/security-and-quality-review.md`](docs/security-and-quality-review.md) tracks each finding as problem, risk, fix, commit and covering test.

## Data

- Migrations are applied in order by `go run ./cmd/migrate` and recorded in `schema_migrations`.
- Enrollment transfers close the origin enrollment and open the destination one inside a single database transaction, and keep the history.
- Development seeds are separate from migrations and refuse to run when `APP_ENV=production`.

## Running locally with Docker

The Docker image and Compose stack are for local development only. Production
continues to use Render's native Go build; `render.yaml` is unchanged.

1. Copy `.env.example` to `.env` and set `OIDC_ISSUER` to the issuer URL of a
   real Supabase project (`https://<project-ref>.supabase.co/auth/v1`). The
   example database passwords are fake; replace them for your local setup.
2. Start MariaDB, apply migrations, and run the API:

   ```bash
   docker compose up --build --detach api
   ```

3. Confirm the API and database are healthy:

   ```bash
   curl --fail http://localhost:3333/health
   ```

Optional development seed data can be loaded with
`docker compose --profile seed run --rm seed`. The database is persisted in the
named `mariadb_data` volume. Local CORS already permits loopback origins outside
production, so `CORS_ORIGINS` need not be set for local frontends.

To run the documented end-to-end smoke check on a machine with Docker and curl:

```bash
sh scripts/smoke-compose.sh
```

## Running it

```bash
go run ./cmd/migrate     # apply migrations
go run ./cmd/seed        # dev seeds (non-production only)
go run ./cmd/server      # start the API (default port 3333)

go test ./...
go vet ./...
go test -race ./...
```

Configuration is by environment variables (`DB_*`, `OIDC_ISSUER`, `OIDC_AUDIENCE`, `AUTH_REQUIRE_VERIFIED_EMAIL`, `RATE_LIMIT_PER_MINUTE`, `CORS_ORIGINS`, `BOOTSTRAP_SUPER_ADMIN_EMAIL`). The full table is in [`README.pt.md`](README.pt.md).

## Quality

- Tests cover domain, application, HTTP adapters, OIDC, config and migrations.
- CI runs `go build`, `go vet`, `go test -race` and a Docker image build on every PR.

## Documentation

| Doc | What it covers |
| --- | --- |
| [`docs/architecture.md`](docs/architecture.md) | Diagrams: layers, request pipeline, identity, authorization, event flow, data model, deployment |
| [`docs/openapi.yaml`](docs/openapi.yaml) | API contract (OpenAPI 3.0) |
| [`docs/adr/`](docs/adr) | Architecture decision records |
| [`docs/security-and-quality-review.md`](docs/security-and-quality-review.md) | Security and quality findings with fixes |
| [`README.pt.md`](README.pt.md) | Portuguese version with the configuration table |

## Architecture decisions

| ADR | Decision |
| --- | --- |
| [0001](docs/adr/0001-eventos.md) | Event publication failures are logged and do not fail a write that already succeeded; a transactional outbox is the long-term fix |
| [0002](docs/adr/0002-identidade.md) | Identity model |
| [0003](docs/adr/0003-autorizacao.md) | Authorization by scope through domain policies |
| [0004](docs/adr/0004-local-docker-image.md) | Local multi-stage Docker image using Alpine |

## Known limitations

- Event delivery is best effort until a transactional outbox exists (ADR 0001).
- Route authorization is applied per handler; a declarative route table with a test that walks it is still pending (ADR 0003).
- The local multi-stage Docker image now runs as a non-root user and has a `/health` check. Production image hardening, automated image vulnerability scanning, and a Docker Compose smoke run in CI remain pending.
- Still to do: keep `docs/openapi.yaml` in sync automatically (it is hand-written today), uniform error catalog, pagination/search/idempotency, coverage thresholds, MariaDB integration tests in CI, and security linters.
