# ADR 0004: Local Docker image on Alpine

## Status

Accepted.

## Decision date

2026-11-05.

## Context

The backend needs a reproducible local environment with MariaDB, migrations, and the API. Production deployment continues to use Render's native Go build and must not be changed by this decision.

## Decision

Use a multi-stage image: compile Go binaries with `CGO_ENABLED=0` in `golang:<version>-alpine` and run them in `alpine:3.22`. The runtime image installs CA certificates for OIDC/JWKS requests and runs as a non-root user. BusyBox `wget` checks `GET /health`, which also checks the database connection. The image includes the `server`, `migrate`, and `seed` commands; Compose uses `seed` only under its optional profile.

## Alternatives considered

- **Distroless:** rejected because it has no shell or `wget`; implementing the health check inside a binary would add unnecessary application code.
- **A single Go image with build tools at runtime:** rejected because it would increase the image attack surface.

## Consequences

- The Docker stack is for local development only; it does not change `render.yaml` or define the production image.
- The image is smaller than the build stage, but Alpine still contains runtime utilities. Updating the base image and checking for vulnerabilities remain outstanding responsibilities.
- The Compose smoke test must run in an environment with Docker installed before compatibility of the stack is claimed.
