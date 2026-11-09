# ADR 0003: Scope authorization and domain policy

## Status

Accepted for the current model; declarative route authorization remains pending.

## Context

Authorization depends on a user's role and institution or group membership. Keeping these rules in handlers or SQL would duplicate business logic and make it harder to test consistently.

## Decision

Authentication middleware validates the user and places the identity in the request context. Domain policies determine superadmin privileges, institution scope, and group management permissions. Handlers apply those policies before executing operations. Registered API endpoints require authentication except `GET /health`.

## Consequences

- Institution- and group-dependent authorization continues to use domain rules instead of duplicating policy in SQL or handlers.
- Declarative per-route authorization and a test that walks the route table remain future work to reduce the current distribution of checks.
