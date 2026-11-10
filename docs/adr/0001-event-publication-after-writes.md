# ADR 0001: Event publication after writes

## Status

Accepted as a temporary alternative to a transactional outbox.

## Context

Entities and events are currently persisted in separate operations. An outbox requires transactional repositories to coordinate entities, events, and scopes, as well as a dispatcher whose lifecycle is managed by the server; this change crosses the current domain and persistence interfaces.

## Decision

For student creation, enrollment creation and transfer, enrollment suspension, and reopening, a failure to persist or publish the event is logged but does not change the successful response for the completed write.

## Consequences

- A publication failure does not prompt the client to retry a write that the backend has already confirmed.
- This is better than returning `5xx` after persisting a write, but it does not guarantee eventual delivery.
- If event persistence fails, the event may be lost.
- If the client does not receive the response even though the server sent it, a retry can still duplicate the write; this alternative does not provide idempotency.
- A transactional outbox remains necessary to guarantee at-least-once delivery.
