# Architecture

Modular monolith, Clean Architecture:

`cmd/server` -> `internal/httpapi` -> services (`billing`, `provisioning`,
`automation`, `monitoring`) -> `connectors/sdk.NetworkConnector` -> vendor
packages (`mikrotik`, `generic`, `cloud`, `routers`, `olt`).

Cross-cutting: `config`, `logger` (slog JSON), `events` bus (NATS-ready seam),
`jobs` queue (Redis-Streams-ready seam), `security` (AES-GCM secrets),
`rbac`, `auth` (HS256 JWT, OIDC-ready), `health` (/health /ready /live /metrics),
`workers`, `scheduler`, `notify` (email/SMS/WA abstractions).

Tenancy: `org_id` enforced in service/repository layer. Secrets never leave
the API (masked, encrypted at rest in `device_credentials`).
