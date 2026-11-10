# Secretaria Backend

Go API for the School Secretariat platform: institutions, students, enrollments, groups and users, with scoped authorization and a real-time domain-event stream. The micro-frontends that consume it live in [`mfe-communication`](https://github.com/fidelis27/mfe-communication).

A Portuguese translation is available in [`README.pt.md`](README.pt.md).

## Stack

Go, MariaDB, Supabase Auth (OIDC/JWKS), WebSocket, and GitHub Actions.

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
migrations/          versioned SQL migrations
seeds/dev/           development-only seed data
docs/adr/            architecture decision records
```

Repositories sit behind domain interfaces, `main` wires dependencies, and business rules (including authorization) live in the domain layer rather than in SQL or handlers. See [the architecture guide](docs/architecture.md).

## Configuration

The server reads environment variables. `.env.example` contains local development examples; replace the clearly fake database passwords for your own environment.

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `3333` | HTTP server port. |
| `APP_ENV` | `development` | Controls production defaults and local CORS behavior. |
| `DB_HOST`, `DB_NAME`, `DB_USER` | none | MariaDB connection settings. |
| `DB_PORT` | `3306` | MariaDB port. |
| `DB_PASSWORD` | empty | MariaDB password. |
| `DB_TLS` | `false` | Enables TLS for MariaDB. |
| `DB_TLS_SKIP_VERIFY` | `false` | Disables database certificate verification; do not enable in production. |
| `OIDC_ISSUER` | none | Supabase Auth issuer URL, including `/auth/v1`. |
| `OIDC_AUDIENCE` | `authenticated` | Accepted token audience. |
| `AUTH_REQUIRE_VERIFIED_EMAIL` | `true` in production, otherwise `false` | Requires the top-level `email_verified: true` claim. |
| `RATE_LIMIT_PER_MINUTE` | `120` | Token-bucket capacity and refill rate per user or IP. |
| `CORS_ORIGINS` | empty | Comma-separated allowed origins; configure explicitly in production. |
| `BOOTSTRAP_SUPER_ADMIN_EMAIL` | empty | Account promoted or created only when no local superadmin exists. |

## Authentication

Every registered API endpoint except `GET /health` requires a Supabase access token. Send it in the `Authorization: Bearer <token>` header for HTTP requests; the `GET /events` WebSocket accepts it in the `bearer.<access-token>` subprotocol. The verifier uses OIDC discovery and JWKS, accepts only `RS256` and `ES256`, validates issuer, audience and token times with a 30-second clock leeway, and paces issuer/JWKS requests to one per second. In production, the top-level `email_verified` claim must be `true`; editable `user_metadata` is not trusted. Tokens are bound to an active local user by the stable `sub` (`users.auth_sub`). Invalid tokens receive `401`; unknown or inactive users receive `403`.

## Authorization

Roles are scoped by institution and group. [The domain policy](internal/domain/authorization/policy.go) defines the rules, and HTTP handlers apply them before operations. A `super_admin` requires both the Supabase `app_metadata.roles` entry and the local superadmin flag. Group admins cannot remove or demote the last admin of a group; the API responds with `409`. Institution-scoped listings are filtered to the caller's visible institutions. See [ADR 0003](docs/adr/0003-scope-authorization-and-domain-policy.md).

## Public routes

The only registered API endpoint that bypasses identity authentication is `GET /health` (also matched for `HEAD` by Go's HTTP mux); it returns `200` when the health check succeeds and `503` otherwise. This is the only OpenAPI operation with `security: []`.

The CORS middleware also answers `OPTIONS` preflight requests with `204` before authentication. These are preflight responses, not data endpoints; browser CORS headers are sent only for allowed origins. All other registered API endpoints require authentication.

## Domain events

Writes publish events with `eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` and `payload`.

- Each event is saved to `audit_events` with its institution scope before it is broadcast. Persistence happens after the entity write commits, in a separate transaction, so a failed event save can lose the event (see [ADR 0001](docs/adr/0001-event-publication-after-writes.md)).
- `GET /events` is a WebSocket. The token travels in the `bearer.<access-token>` subprotocol, which is echoed in the handshake. The server checks the WebSocket Origin, limits frames to 1 KiB, uses ping/pong to detect dead connections, and closes connections during graceful shutdown.
- `GET /events/history` returns persisted events scoped by institution.

## Security

- **Token verification:** OIDC issuer, audience, signature and token times are validated; only `RS256` and `ES256` are accepted. Verified email is required by default in production.
- **Institution isolation:** authorization is enforced through the domain policy. Cross-institution event filtering is covered by [`event_handler_test.go`](internal/adapters/httpserver/event_handler_test.go) (`TestEventVisibleToInstitutionsFiltersOtherScopes`) and [`event_history_handler_test.go`](internal/adapters/httpserver/event_history_handler_test.go) (`TestEventHistoryHandlerDoesNotLeakCrossInstitutionEvents`).
- **Rate limiting:** a per-user/IP token bucket applies to `POST`, `PATCH`, `DELETE`, and `GET /events` (default capacity and refill of 120 requests per minute); rejected requests receive `429` and `Retry-After`.
- **Response headers:** all responses get `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`; authenticated responses get `Cache-Control: no-store`.
- **CORS:** production origins must be explicitly configured in `CORS_ORIGINS`; outside production, localhost and 127.0.0.1 loopback origins are also allowed. CORS is a browser access policy, not authentication. WebSocket connections are rejected when their Origin is not allowed.

## Data

- Migrations are applied in order by `go run ./cmd/migrate` and recorded in `schema_migrations`.
- Enrollment transfers close the origin enrollment and open the destination one in a single database transaction while preserving history.
- Development seeds are separate from migrations and refuse to run when `APP_ENV=production`.

## Running locally with Docker

The Docker image and Compose stack are for local development. Production uses Render's native Go build.

1. Copy `.env.example` to `.env` and set `OIDC_ISSUER` to the issuer URL of a Supabase project (`https://<project-ref>.supabase.co/auth/v1`). The example database passwords are fake; replace them for your local setup.
2. Start MariaDB and the API:

   ```bash
   docker compose up --build --detach api
   ```

3. Check API and database health:

   ```bash
   curl --fail http://localhost:3333/health
   ```

Optional development seed data can be loaded with `docker compose --profile seed run --rm seed`. The database is persisted in the named `mariadb_data` volume. Local loopback origins are allowed outside production, so `CORS_ORIGINS` need not be set for local frontends.

The end-to-end Docker smoke check requires Docker and `curl`:

```bash
sh scripts/smoke-compose.sh
```

## Running it

```bash
go run ./cmd/migrate     # apply migrations
go run ./cmd/seed        # development seeds (non-production only)
go run ./cmd/server      # start the API (default port 3333)

go test ./...
go vet ./...
go test -race ./...
```

## Quality

Tests cover domain, application, HTTP adapters, OIDC, configuration and migrations. CI runs `go build ./...`, `go vet ./...`, `go test -race ./...`, and a Docker image build on pull requests.

## Documentation

| Document | What it covers |
| --- | --- |
| [`docs/architecture.md`](docs/architecture.md) | Layers, request pipeline, identity, authorization, event flow, data model and deployment |
| [`docs/openapi.yaml`](docs/openapi.yaml) | HTTP and WebSocket API contract |
| [`docs/adr/`](docs/adr) | Architecture decision records |
| [`docs/security-and-quality-review.md`](docs/security-and-quality-review.md) | Security and quality findings with fixes |
| [`README.pt.md`](README.pt.md) | Portuguese translation of this README |

## Architecture decisions

| ADR | Decision |
| --- | --- |
| [0001](docs/adr/0001-event-publication-after-writes.md) | Event publication failures do not fail a write that has already succeeded; a transactional outbox remains the long-term solution |
| [0002](docs/adr/0002-external-identity-bound-by-subject.md) | Identity model |
| [0003](docs/adr/0003-scope-authorization-and-domain-policy.md) | Authorization by scope through domain policies |
| [0004](docs/adr/0004-local-docker-image.md) | Local multi-stage Docker image |

## Known limitations

- Event delivery is best effort until a transactional outbox exists (ADR 0001).
- Route authorization is applied per handler; a declarative route table with a test that walks it remains pending (ADR 0003).
- Production image hardening, automated image vulnerability scanning, and a Docker Compose smoke run in CI remain pending.
- OpenAPI is hand-written; pagination, search, idempotency, coverage thresholds, MariaDB integration tests in CI, and security linters remain pending.
