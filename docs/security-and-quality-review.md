# Security and quality review

| Finding | Risk | Fix | Commit | Covering test |
|---|---|---|---|---|
| A committed write followed by an event persistence/publication failure returned `500`; enrollment creation also emitted no event. | A client could retry an operation that had already been stored; without idempotency this could duplicate data, and enrollment creation events were missing. | Event failures are logged and do not change the successful write response; enrollment creation publishes `ENROLLMENT_CREATED`. This is a temporary alternative, not an outbox. | `39af1cc`, `feat(events): publish enrollment creation` | `TestPublishDomainEventBestEffortLogsFailure`, `TestPublishDomainEventPersistsEnrollmentCreation` |
| The WebSocket upgrade did not negotiate the requested subprotocol and allowed connections without an Origin header. | Browsers could reject the handshake, and clients with disallowed origins could attempt to open the stream. | The bearer subprotocol is echoed during negotiation; allowed Origin is required; frames are limited, ping/pong detects dead connections, and connections close during shutdown. | `e931405`, `a2fc873` | `TestEventHandlerNegotiatesBearerSubprotocol`, `TestEventHandlerRejectsDisallowedOrigin`, `TestEventConnectionsCloseRegisteredConnectionsOnShutdown` |
| Authentication had no configurable email-verification policy. | An unverified token could authenticate in production, or editable metadata could be mistaken for a trusted claim. | Only the top-level claim is read; verification is required by default in production, and missing-claim behavior is tested explicitly. | `e931405` | `TestIdentityMiddlewareEmailVerificationPolicy`, `TestLoadAuthRequireVerifiedEmailDefaults` |
| OIDC tokens used algorithms advertised by discovery without a local allowlist, a defined short clock leeway, or a request limit for the issuer/JWKS endpoints. | Unexpected algorithms, small clock differences, and excessive key refreshes could weaken or disrupt authentication. | Only RS256/ES256 are accepted, a 30-second leeway is applied, and OIDC/JWKS requests are paced to one per second per process. | `e931405` | `TestVerifierRejectsWrongAudience`, `TestVerifierRejectsExpiredAndUnsupportedAlgorithmTokens`, `TestPacedRoundTripperLimitsUnknownKeyRefreshes` |
| Writes and streams had no basic request limit, and responses lacked protective headers. | Resource abuse and accidental caching of authenticated responses. | Configurable per-user/IP token bucket, `Retry-After`, `no-store`, `nosniff`, and `Referrer-Policy`. | `a2fc873` | `TestRateLimitMiddlewareReturnsRetryAfter`, `TestRateLimiterIsSafeForConcurrentRequests`, `TestSecurityHeadersMiddlewareAddsBrowserHeaders` |
| HTTP logging could evolve to include sensitive headers or query strings. | Exposure of bearer tokens or full email addresses in logs. | Observability records only method, path without query, status, duration, and correlation ID. | `a2fc873` | `TestObservabilityLogsDoNotContainAuthorizationOrFullEmail` |
| Migration `007` contains a personal email address and is part of public history. | Rewriting an applied migration breaks consistency across environments; adding more literal addresses would increase exposure. | Migration `007` remains historical and unchanged; current bootstrap uses `BOOTSTRAP_SUPER_ADMIN_EMAIL`; a test blocks email literals in migrations outside the historical allowlist. | `test(migrations): block email literals` | `TestNewMigrationsDoNotContainEmailLiterals` |

## Event bug lessons

The first write/event bug passed because tests covered validation and database failures before the write, but not an event publication failure after the entity commit. The new test injects an event repository failure and verifies that it is reported as non-fatal.

The WebSocket handshake had no test that inspected the negotiated subprotocol. The previous test connected without requesting one, which was accepted by non-browser clients but rejected by browsers when `Sec-WebSocket-Protocol` was sent. The handshake test now requests the bearer subprotocol and checks the negotiated value.

## Legacy migration decision

Migration `007_promote_owner_super_admin.sql` is historical and contains an email address used in that environment. It must not be changed; new installations use `BOOTSTRAP_SUPER_ADMIN_EMAIL`, and migration tests block new literal addresses.

## Remaining work from this review

The transactional outbox remains pending. Other remaining items include declarative route authorization, keeping OpenAPI in sync, fully standardizing errors, pagination/search/idempotency, coverage thresholds, MariaDB integration tests in CI, security linters, and production Docker image hardening.
