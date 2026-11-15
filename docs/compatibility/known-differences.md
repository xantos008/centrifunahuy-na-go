# Known differences from the baseline

These are deliberate deviations from OSS/PRO v6.10.3 behavior, and places where we cannot verify parity. A compatibility-suite diff (ADR-0028) is allowed only for scenarios mapped to an `accepted` entry here. Extensions are also listed as EXT-* rows in the feature matrix. Every entry needs a reason and a status. Accepted entries need an ADR.

Status values:
- `proposed`: Phase 0 recommendation.
- `pending DEC-nn`: blocked on a user decision (see [open-questions.md](../research/open-questions.md) §A).
- `accepted`.

## Not implemented

| # | Item | Reason | Status |
|---|---|---|---|
| KD-01 | SockJS transport, Tarantool engine | Removed upstream in v6 | accepted (fact) |
| KD-02 | Shared poll relay (PRO-27.05) | Upstream says "in progress, not officially supported" | proposed |
| KD-03 | Dictionary compression (PRO-30); connect `flag`/`dict`/`profile` fields are decoded and ignored | Upstream page is `draft: true`, unreleased | accepted (DEC-06, ADR-0024) |
| KD-04 | Ephemeral client publications / schemas (PRO-29) | Page unlisted; no release note | accepted (DEC-06, ADR-0024) |
| KD-05 | PRO licensing: license keys, sandbox limits, license-expiry metric | Not a product feature for us; must not be replicated | accepted |
| KD-06 | Anonymous usage stats reporting (`usage_stats`) | Our product sends no telemetry. The inherited reporter (`internal/usage`) is disabled in phase 1; the config key is accepted and ignored | proposed |
| KD-07 | Redis `history_use_lists` (deprecated upstream) | Deprecated; limitations differ between docs and code (INFRA-CONFLICT-04) | proposed: accept the key, fail validation with a clear message |
| KD-08 | Bidirectional gRPC client transport | Does not exist upstream; the user declined it | accepted (DEC-03, ADR-0021) |

## Same feature, different internals (not observable through public APIs)

| # | Item | Difference | Status |
|---|---|---|---|
| KD-10 | Redis / PostgreSQL / NATS data layout | Inherited from upstream at import (DEC-01), but **not a contract**: default prefixes differ (ADR-0022), and PRO modules add keys and control messages. **Mixed clusters with Centrifugo nodes are not supported** | accepted (DEC-05, ADR-0023) |
| KD-11 | PostgreSQL transactional publishing | The exception to KD-10: the SQL surface apps call (`cf_stream_*` functions etc.) is a public contract and **will** match the docs | proposed |
| KD-12 | ClickHouse schema | Our own versioned DDL. Upstream DDL is not public (PRO-OQ-10). Column semantics follow the documented export fields | proposed |
| KD-13 | Push device ID format | Upstream format is not public (PRO-OQ-16). Ours is opaque and sortable | proposed |
| KD-14 | Singleflight, shared position sync, rate-limit internals | We match the documented observable result, not the internal algorithm | accepted (CLAUDE.md) |
| KD-15 | Admin UI look and text | Our own design and copy. The inherited embedded upstream UI (`internal/webui`) is removed from the build | accepted (CLAUDE.md, ADR-0016, ADR-0022) |
| KD-16 | Admin HTTP API shape | OSS admin endpoints inherited unchanged; documented PRO endpoints implemented to the docs; anything else only under `/admin/x/v1/*` and labeled EXTENSION | accepted (DEC-07, ADR-0025) |
| KD-17 | Product names | Default mode uses own names (`CEDRAZEKA_*` env, `cedrazeka_*` metrics, own gRPC packages and Kafka headers). `compat: centrifugo` restores `CENTRIFUGO_*`, `centrifugo_*`, and the upstream gRPC service names and Kafka headers | accepted (DEC-04, ADR-0022); names per DEC-12 |

## Stricter than upstream (security hardening beyond the docs)

| # | Item | Upstream (documented) | Ours | Status |
|---|---|---|---|---|
| KD-20 | JWKS fetch (client and server API) | Template/regex safety check for dynamic JWKS (OSS 6.7.0); other protections undocumented (OQ-AUTH-02) | Also: no cross-host redirects, response size cap, timeouts, optional private-IP block | proposed |
| KD-21 | Admin OIDC | PKCE, discovery, per-request token check | Also: state, nonce, CSRF tokens on mutations, explicit session expiry (PRO-OQ-11) | proposed |
| KD-22 | Audit log | Process log line `component=admin_audit`; server API calls are not audited | Same log line, plus an optional durable append-only sink and request IDs | proposed (extension, off by default) |
| KD-23 | CEL | Resource limits undocumented | Cost limit, deadline, expression length cap; fail closed | proposed |
| KD-24 | Labels | Size limits undocumented (PRO-OQ-15) | Documented caps on count and key/value length | proposed |
| KD-25 | gRPC server API | No auth by default | Same default (compatibility), plus a startup WARN when exposed without a key | proposed |

## Extensions (additive, off by default or invisible to compatible clients)

| # | Item | Why |
|---|---|---|
| KD-30 | Readiness endpoint with backend checks | Upstream `/health` always returns 200 `{}` (INFRA-CONFLICT-18). We keep `/health` identical and add a separate readiness path |
| KD-31 | Snapshot coverage fields (`nodes_responded`, `nodes_known`, partial flag) | SPEC requires them; not documented upstream |
| KD-32 | Analytics field suppression / PII controls beyond `export_users`/`export_channels` | SPEC requirement; not documented upstream |
| KD-33 | Multi-tenancy | Upstream has no first-class tenant, only patterns. v1 is deployment-per-tenant. New PRO storage carries a constant `scope` (EXT-05) so full multi-tenancy stays possible (DEC-02, ADR-0014) |
