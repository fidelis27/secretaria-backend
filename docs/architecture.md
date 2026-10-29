# Architecture

Diagrams for the Go API. The micro-frontends that consume it are documented in [`mfe-communication`](https://github.com/fidelis27/mfe-communication/blob/main/docs/architecture.md). The API contract is in [`openapi.yaml`](openapi.yaml).

Contents: [1. System context](#1-system-context) · [2. Layers](#2-layers-hexagonal) · [3. Request pipeline](#3-request-pipeline) · [4. Identity resolution](#4-identity-resolution) · [5. Authorization](#5-authorization) · [6. Write and publish flow](#6-write-and-publish-flow) · [7. WebSocket stream](#7-websocket-event-stream) · [8. Data model](#8-data-model) · [9. Deployment](#9-deployment) · [10. Known gaps](#10-known-gaps)

---

## 1. System context

```mermaid
flowchart LR
    fe(["Frontend<br/>(host + MFEs)"])
    subgraph api["secretaria-backend (Render)"]
        http["HTTP API<br/>+ WebSocket /events"]
    end
    db[("MariaDB<br/>TLS")]
    supa["Supabase Auth<br/>OIDC discovery + JWKS"]

    fe -- "HTTPS, Bearer JWT" --> http
    fe -- "WSS, subprotocol bearer.JWT" --> http
    http -- "verify signature,<br/>issuer, audience, expiry" --> supa
    http -- "SQL" --> db
```

The API never issues tokens. Supabase authenticates people; the API decides what a verified person may do, using its own users and groups tables.

---

## 2. Layers (hexagonal)

Dependencies point inward. The domain knows nothing about HTTP or SQL.

```mermaid
flowchart TB
    subgraph cmd["cmd/ (composition roots)"]
        server["server"]
        migrate["migrate"]
        seed["seed (refuses APP_ENV=production)"]
    end

    subgraph adapters["internal/adapters"]
        httpserver["httpserver<br/>handlers, middleware, WebSocket"]
        mariadb["mariadb<br/>repositories"]
        oidc["oidc<br/>token verifier"]
    end

    subgraph app["internal/application"]
        uc["use cases<br/>institution, student, enrollment,<br/>group, user, bootstrap, health"]
        ev["event service + in-process Bus"]
    end

    subgraph domain["internal/domain"]
        ent["entities and rules"]
        pol["authorization policy"]
        ports["repository interfaces (ports)"]
    end

    server --> httpserver
    server --> mariadb
    server --> oidc
    httpserver --> uc
    httpserver --> pol
    httpserver --> ev
    uc --> ent
    uc --> ports
    ev --> ports
    pol --> ports
    mariadb -. "implements" .-> ports
```

`main` only wires dependencies. Business rules live in the domain and use cases, not in SQL.

---

## 3. Request pipeline

Every request passes the same chain, outermost first. `GET /health` skips identity.

```mermaid
flowchart LR
    obs["Observability<br/>log, counters"] --> sec["Security headers<br/>nosniff, no-referrer"]
    sec --> cors["CORS<br/>allowlist; OPTIONS ends here"]
    cors --> id["Identity<br/>JWT to local user"]
    id --> rl["Rate limit<br/>POST, PATCH, DELETE, /events"]
    rl --> mux["Router"]
    mux --> h["Handler"]
    h --> p["Authorization policy"]
    p --> s["Use case"]
    s --> r["Repository"]
    r --> db[("MariaDB")]
```

Notes that matter when debugging:

- Rate limiting sits **after** identity, so authenticated requests are limited per user and anything else per IP. Plain `GET` list routes are not limited; `GET /events` is.
- Logs carry method, path without query, status, duration and correlation ID. Never the Authorization header or a full e-mail.
- Responses for authenticated calls carry `Cache-Control: no-store`.

---

## 4. Identity resolution

How a verified token becomes a local user (ADR 0002).

```mermaid
flowchart TD
    start(["Request with token"]) --> tok{"Token present<br/>and valid?"}
    tok -- "no" --> r401["401"]
    tok -- "yes" --> mail{"email_verified ok?<br/>(required in production)"}
    mail -- "no" --> r403a["403"]
    mail -- "yes" --> bysub{"User found<br/>by sub?"}
    bysub -- "yes" --> active
    bysub -- "no" --> byemail{"User found<br/>by e-mail?"}
    byemail -- "no" --> r403b["403"]
    byemail -- "yes" --> linked{"Already linked<br/>to another sub?"}
    linked -- "yes" --> r403c["403"]
    linked -- "no" --> link["Link sub to user<br/>(permanent)"]
    link --> active{"User active?"}
    active -- "no" --> r403d["403"]
    active -- "yes" --> sa["superAdmin = local flag<br/>AND token role super_admin"]
    sa --> ok(["Identity in context"])
```

Superadmin needs **both** the local flag and the Supabase role claim. E-mail is only used on first login to claim a pre-provisioned account.

---

## 5. Authorization

Rules live in `internal/domain/authorization` and handlers call them before acting. Groups belong to an institution, and a user's role is per group.

| Action | Superadmin | Group admin | Group member |
| --- | --- | --- | --- |
| Create institution, group, user; list users; read `/metrics` | yes | no | no |
| Update a user | yes | no | no |
| Create student or enrollment, transfer, suspend, reopen | yes | only for institutions where they are admin (a transfer needs both) | no |
| Manage members of a group | yes | only that group | no |
| Read institutions, students, enrollments, groups, events | all | only their institutions | only their institutions |

Guards: the last admin of a group cannot be demoted or removed (`409`); a user cannot deactivate themselves; the last active superadmin cannot be deactivated.

---

## 6. Write and publish flow

What happens on `POST /students`. Other writes follow the same shape.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as Handler
    participant P as Policy
    participant S as Use case
    participant DB as MariaDB
    participant E as Event service
    participant B as Bus

    C->>H: POST /students (Bearer JWT, x-correlation-id)
    H->>P: CanEditInstitution(user, institutionId)
    alt not allowed
        P-->>H: false
        H-->>C: 403
    else allowed
        H->>S: Create(name, institutionId)
        S->>DB: INSERT student
        S-->>H: student
        H->>E: Publish(STUDENT_CREATED, scope = institution)
        E->>DB: BEGIN, INSERT audit_events + audit_event_institutions, COMMIT
        alt event saved
            E->>B: Publish(event)
        else save failed
            E-->>H: error (logged only)
        end
        H-->>C: 201 Created
    end
```

The student write and the event save are **two separate transactions** (ADR 0001). The event row and its institution scope are saved together, and the broadcast happens only after that commit. If the event save fails, the client still gets `201` and the event is lost. A transactional outbox is the planned fix.

Transfers are different: closing the origin enrollment and opening the destination happen inside one transaction, and only the event is published afterwards.

---

## 7. WebSocket event stream

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as /events handler
    participant B as Bus

    C->>H: GET /events, Upgrade, subprotocol bearer.JWT
    H->>H: identity middleware verifies token, rate limit applies
    H->>H: resolve visible institutions for the user
    H->>B: Subscribe (buffer of 16)
    H-->>C: 101, subprotocol echoed (origin must be allowed)
    loop until close
        B-->>H: event
        alt superadmin or event.institutionIds intersects user scope
            H-->>C: JSON event
        else out of scope
            H->>H: skip
        end
        H-->>C: ping periodically
        C-->>H: pong (otherwise connection is dropped)
    end
```

Limits: client frames are capped at 1 KiB, and open connections receive a close frame on graceful shutdown. The bus publish is non-blocking: if a subscriber's buffer of 16 is full, that event is skipped for that connection. Clients should use `GET /events/history` to catch up.

---

## 8. Data model

Taken from the migrations (`migrations/001` to `010`).

```mermaid
erDiagram
    INSTITUTIONS ||--o{ STUDENTS : "has"
    INSTITUTIONS ||--o{ ENROLLMENTS : "receives"
    STUDENTS ||--o{ ENROLLMENTS : "has"
    INSTITUTIONS ||--o{ GROUPS : "scopes"
    USERS ||--o{ MEMBER_GROUPS : "joins"
    GROUPS ||--o{ MEMBER_GROUPS : "has"
    AUDIT_EVENTS ||--o{ AUDIT_EVENT_INSTITUTIONS : "visible to"
    INSTITUTIONS ||--o{ AUDIT_EVENT_INSTITUTIONS : "sees"

    INSTITUTIONS {
        varchar id PK
        varchar name
        varchar cnpj
        enum status "active, inactive"
    }
    USERS {
        varchar id PK
        varchar name
        varchar email UK
        varchar auth_sub UK "Supabase subject"
        enum status "active, inactive"
        boolean super_admin
    }
    STUDENTS {
        varchar id PK
        varchar name
        varchar institution_id FK
        enum status "active, inactive"
    }
    ENROLLMENTS {
        varchar id PK
        varchar student_id FK
        varchar institution_id FK
        enum status "active, inactive, transferred, suspended, closed"
        varchar suspension_reason
        timestamp suspended_at
    }
    GROUPS {
        varchar id PK
        varchar institution_id FK
        varchar name "unique per institution"
    }
    MEMBER_GROUPS {
        varchar user_id PK
        varchar group_id PK
        enum role "admin, member"
    }
    AUDIT_EVENTS {
        varchar event_id PK
        varchar event_type
        int version
        varchar source
        varchar correlation_id
        timestamp occurred_at
        json payload
    }
    AUDIT_EVENT_INSTITUTIONS {
        varchar event_id PK
        varchar institution_id PK
    }
```

A student has one record and may have many enrollments over time. Event visibility is not derived from the payload: it is stored explicitly in `audit_event_institutions`.

---

## 9. Deployment

```mermaid
flowchart LR
    gh["GitHub Actions<br/>go build, vet, test -race"] --> render
    subgraph render["Render web service"]
        app["./app (cmd/server)<br/>health check: /health"]
    end
    app --> db[("MariaDB, TLS")]
    app --> supa["Supabase Auth"]
    mig["go run ./cmd/migrate<br/>(schema_migrations)"] -. "manual step, not in render.yaml or CI" .-> db
```

Configuration is by environment variables (`DB_*`, `OIDC_ISSUER`, `OIDC_AUDIENCE`, `CORS_ORIGINS`, `RATE_LIMIT_PER_MINUTE`, and others); see `README.pt.md` for the full table.

---

## 10. Known gaps

- **Event delivery is best effort** (section 6, ADR 0001). Fix: transactional outbox.
- **The bus is in-process.** It cannot fan out across several instances, and slow WebSocket clients can miss events (section 7).
- **Authorization is applied per handler**, not from a declarative route table (ADR 0003).
- **Some writes publish no event:** creating institutions, users and groups, and adding group members. There is also no route to update or inactivate an institution.
- **Contract drift with the frontend.** The frontend declares `INSTITUTION_CREATED`, `INSTITUTION_UPDATED`, `INSTITUTION_INACTIVATED` and `STUDENT_UPDATED`, which this API never emits. The API emits `ENROLLMENT_CREATED`, `USER_UPDATED` and `GROUP_MEMBERSHIP_*`, which the frontend contract does not list. Align the two or remove the unused types.
- **No pagination, search or idempotency keys** on list and write routes.
