# ADR 0002: External identity bound by subject

## Status

Accepted.

## Context

An external identity provider supplies a stable subject (`sub`) and an email address that may change. Local users may also have been created before the backend stored the provider subject.

## Decision

`users.auth_sub` is the primary external identifier. The backend looks up users by `sub` first. On first login only, it may use email to find a legacy local account without an `auth_sub`, then permanently bind the subject to that account. Access is denied if the subject is already bound to another account or the matched local account has a different subject.

## Consequences

- Email changes at the provider do not change the local identity after the first binding.
- Email alone cannot reassociate an account that is already bound.
- Administrators must provision local users before their first login.
