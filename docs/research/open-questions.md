# Open questions

Phase 0, 2026-11-15. Rule: **do not guess.** Each question is resolved in one of two ways:
- a black-box experiment against the OSS v6.10.3 reference binary with official SDKs (behavioral questions);
- a user decision (scope, licensing, architecture).

Resolved items move to the source register (evidence) or to known-differences (deviations).

## A. User decisions

### A.1 Decided 2026-11-15

| ID | Decision | ADR |
|---|---|---|
| DEC-01 | Build on Centrifugo OSS v6.10.3 (Apache-2.0) as the code base; keep license notices and attribution; no PRO code; PRO features from public docs; record the provenance of every borrowed component | ADR-0002, ADR-0029, docs/legal/provenance.md |
| DEC-02 | Deployment-per-tenant for v1; the architecture must not block full multi-tenancy later | ADR-0014 |
| DEC-03 | No bidirectional gRPC client transport | ADR-0021 |
| DEC-04 | Own names by default; separate compatibility mode for the needed config and API names; legal limits documented | ADR-0022, docs/legal/naming-and-trademarks.md |
| DEC-05 | No mixed clusters with real Centrifugo nodes | ADR-0023 |
| DEC-06 | PRO features that are not publicly documented are not in mandatory scope; unknown behavior becomes an open question | ADR-0024 |
| DEC-07 | Admin API only from confirmed public contracts; unconfirmed capabilities are not presented as compatible | ADR-0025 |
| DEC-08 | No GitHub MCP or Context7 for now; all tools and settings project-local | ADR-0026 |
| DEC-09 | Helm deferred until the core is stable | ADR-0027 |

### A.2 Follow-up decisions (decided 2026-11-15)

| ID | Decision | Where |
|---|---|---|
| DEC-10 | Project license: **Apache-2.0 for the whole repo**. Free for anyone; "AS IS", no warranty, no liability for the authors (§7–8) | ADR-0030 |
| DEC-11 | CLAUDE.md "Mission" updated: OSS is a fork, clean-room applies to PRO only | CLAUDE.md |
| DEC-12 | Product name **Cedrazeka**; binary `cedrazeka`, env `CEDRAZEKA_`, metrics `cedrazeka`, key prefix `cedrazeka`, gRPC packages `cedrazeka.*.v1` | ADR-0022 |

## B. OSS core (protocol, transports, API, config): CORE-OQ

Source annex: [oss-core-research.md](oss-core-research.md) §5.

| ID | Question | Evidence |
|---|---|---|
| CORE-OQ-01 | Server API `/api/rpc` (gRPC `RPC`) exists in OSS code but is undocumented. Is it part of the public compatibility surface? Which extensions are registered in OSS? `rpc.ping` is registered on the *client* handler (OSS:internal/app/app.go:555), and it is unclear whether the API executor has any extensions in OSS | OSS:internal/api/handler.go:54, OSS:internal/api/api.go:57,854-868 |
| CORE-OQ-02 | Exact request/response fields of `shared_poll_publish`, `map_clear` result, and per-field required flags for the `map_*` methods (docs list "options" informally) | DOCS:docs/server/shared_poll.md §Direct publish/API; DOCS:docs/server/map_subscriptions.md §Server API. Resolve from OSS:internal/apiproto/api.proto |
| CORE-OQ-03 | Disconnect code ranges conflict. The proto says >=5000 is reserved by Centrifuge with reconnect; client_api.md says custom codes must be in [3000, 4999] and anything other than 3000-3499 / 4000-4499 goes to disconnected | PROTO:definitions/client.proto (Disconnect comment) vs DOCS:docs/transports/client_api.md §Disconnect codes |
| CORE-OQ-04 | What does OSS return for PRO-only gRPC methods in the shared `CentrifugoApi` service (Unimplemented? 104?), and over HTTP (404)? | OSS:internal/apiproto/api.proto:25-43; OSS:internal/api/handler.go |
| CORE-OQ-05 | Publication tags filter SDK support conflict. client_sdk.md matrix: all SDKs. publication_filtering.md: "only centrifuge-js". Same kind of conflict for bidirectional emulation (http_stream.md says "only JS", the matrix also has rust and c#) | DOCS:docs/transports/client_sdk.md vs DOCS:docs/server/publication_filtering.md, DOCS:docs/transports/http_stream.md |
| CORE-OQ-06 | Code 104/113 messages differ between the client protocol ("method not found" / "concurrent pagination") and the server API ("not found" / "conflict"). Confirm this is intentional and keep two code tables | DOCS:docs/server/codes.md vs DOCS:docs/server/server_api.md; OSS:internal/apiproto/errors.go |
| CORE-OQ-07 | OSS behaviour of `SubscribeRequest.flag` bit0 (numeric channel id, PRO compaction?) and bit1 (reject unrecovered) | PROTO:definitions/client.proto only |
| CORE-OQ-08 | Does v6.10.3 still accept or require `cf_protocol_version` URL param, or the `format` and `cf_protocol` query params? Only source code and v4 migration docs mention them | DOCS:docs/getting-started/migration-v4.md; [src] LIB:handler_websocket.go:191 |
| CORE-OQ-09 | `http_server.external_http2_extended_connect`: purpose and relation to `GODEBUG=http2xconnect=1` (undocumented) | [src] OSS:internal/configtypes/types.go |
| CORE-OQ-10 | `.env` file: lookup path, precedence relative to real env vars | DOCS:blog/2025-01-16-centrifugo-v6-released.md (one bullet only) |
| CORE-OQ-11 | `enable_unreleased_features` top-level key: semantics | [src] OSS:internal/config/config.go |
| CORE-OQ-12 | Undocumented `client.*` keys: `expired_close_delay`, `expired_sub_close_delay`, `presence_update_interval`, `channel_position_check_delay`, `channel_position_max_time_lag`. Defaults and semantics (use `defaultconfig`/`configdoc` output from a reference binary) | [src] OSS:internal/configtypes/types.go |
| CORE-OQ-13 | Legacy `{method, params}` API types in `handler_legacy.go`: are they reachable in v6.10.3? | OSS:internal/api/handler_legacy.go |
| CORE-OQ-14 | Do uni transports deliver `message` pushes (from the client-message server API)? The uni doc list omits `message` | DOCS:docs/transports/uni_client_protocol.md vs PROTO:definitions/client.proto |
| CORE-OQ-15 | Numeric WS close code for "message too big" (presumably the RFC 6455 value; not stated) | DOCS:docs/transports/websocket.md |
| CORE-OQ-16 | `channel.namespace_boundary`, `user_boundary`, `user_separator`: are they configurable in OSS v6 or internal? configuration.md mentions `channel.namespace_boundary` only by analogy in `rpc.namespace_boundary` | DOCS:docs/server/configuration.md §rpc.namespace_boundary; [src] OSS:internal/configtypes/types.go |
| CORE-OQ-17 | `uni_grpc` "Supported data formats: JSON and binary". How does a uni gRPC client select JSON? | DOCS:docs/transports/uni_grpc.md; OSS:internal/unigrpc/unistream/unistream.proto (not read in depth) |
| CORE-OQ-18 | The `c-lib` clone HEAD (`8888c6e`) is newer than the centrifuge version pinned by v6.10.3 (`ae446d43faf0`). Re-verify the [src] claims taken from LIB against the pinned commit | OSS:go.mod:18 |
| CORE-OQ-19 | Delta compression minimum version is not stated in the current docs (introduced in v5.x per the blog). Confirm for the matrix | DOCS:docs/server/delta_compression.md; DOCS:blog/2024-05-30-real-time-data-compression-experiments.md |
| CORE-OQ-20 | The SPEC item "gRPC transport": does it mean only OSS `uni_grpc`, or is a bidirectional gRPC client transport expected (none is documented in OSS)? Needs a scope decision | DOCS:docs/transports/overview.md |
| CORE-OQ-21 | Unsubscribe code 2502 reason string differs: codes.md says "server tags filter changed", proto comment says "state invalidated". Which string goes on the wire? | DOCS:docs/server/codes.md vs PROTO:definitions/client.proto |
| CORE-OQ-22 | The admin UI is an embedded SPA from the separate `centrifugal/web` repo. A clean-room build needs its own UI. Confirm the admin API contract (`/admin/auth`, `/admin/api`) is in compatibility scope | DOCS:docs/server/admin_web.md; DOCS:docs/server/configuration.md §Custom internal port |

## C. OSS infrastructure (engines, auth, proxy, consumers, observability): INFRA

Source annex: [oss-infra-research.md](oss-infra-research.md) §7.

### Doc/source conflicts (record; ask for a decision where it matters)

| ID | Conflict | Evidence | Suggested default |
|---|---|---|---|
| INFRA-CONFLICT-01 | PG controller `use_notify` default: the docs table says `true`. The code struct has no `default:"true"` tag, so the effective Go default is `false`, yet the code doc string says "Default <<true>>". | D:server/engines.md §Configuration options; O:internal/configtypes/engine.go `PostgresController.UseNotify` | Black-box test the reference binary. Until then, follow the docs (`true`) and flag it. |
| INFRA-CONFLICT-02 | PG controller `batch_size` (1000) and the `tls` key on the PG broker/controller exist in code but not in the docs tables. | O:configtypes/engine.go | Accept them as compatible extras. Mark them undocumented. |
| INFRA-CONFLICT-03 | engines.md says Sentinel **or Cluster** "must use a special scheme". The same page (and R:v6.1.0) says Cluster is auto-detected from a plain address. Code still accepts `redis+cluster://`. | D:server/engines.md §engine.redis.address info box vs §Redis Cluster support | Support both auto-detection and `redis+cluster://`. |
| INFRA-CONFLICT-04 | `history_use_lists` limitations: the docs list reverse order and delta. The code doc adds no pagination and no idempotent publishing. | D:server/engines.md; O:configtypes/engine.go | Deprecated: consider not implementing it. Record a known difference. |
| INFRA-CONFLICT-05 | Kafka `publication_data_mode` header defaults: docs say `""`. Code defaults to `centrifugo-channels`, `centrifugo-idempotency-key`, `centrifugo-delta`, and adds `version_header`, `version_epoch_header`, `tags_header_prefix`, `method_header` (undocumented for Kafka; documented for the other consumers). | D:server/consumers.md §consumers[].kafka; O:configtypes/types.go `KafkaConsumerConfig` | Follow the code defaults, matching the other consumers. Needs a black-box check. |
| INFRA-CONFLICT-06 | observability.md §Logs uses `log_level`/`log_file` (v5 names). configuration.md and the code use `log.level`/`log.file`. | D:server/observability.md; D:server/configuration.md §log; O:configtypes/types.go | `log.level`/`log.file`. |
| INFRA-CONFLICT-07 | The subscribe proxy "does not proxy subscriptions with token", yet `SubscribeRequest.token` is documented as "subscription token … when subscription token authorization is used". | D:server/proxy.md §Subscribe proxy info box vs §SubscribeRequest | Open question: when is `token` populated (stream proxy? PRO?). OQ-PROXY-01. |
| INFRA-CONFLICT-08 | proxy.proto `RefreshResponse`/`SubRefreshResponse` carry `error`/`disconnect`, and `RefreshRequest` has `labels`. The docs say error/disconnect are ignored for refresh, and the docs table omits `labels`. | O:proxyproto/proxy.proto; D:server/proxy.md §Return custom error note | Ignore error/disconnect, as the docs say. Accept the fields on the wire. |
| INFRA-CONFLICT-09 | proxy.proto has fields the docs omit: `ConnectResult.profile`, `SubscribeOptions.expire_at`, `SubscribeOptions.server_tags_filter`, `Error.temporary`. | O:proxyproto/proxy.proto | Accept on the wire. `profile` semantics are unknown (OQ-PROXY-02). |
| INFRA-CONFLICT-10 | `nbf`: the SPEC lists it and the docs never mention it. The code enforces it and maps failure to "token expired" (109 / refresh flow), not "invalid". | O:internal/jwtverify/token_verifier_jwt.go (~610, ~784) | Enforce `nbf`. Map to the expired path. Document it as observed behavior. |
| INFRA-CONFLICT-11 | Subscription-token `client` claim: implemented and referenced indirectly in channel_permissions.md (3500 for a token "issued for another … client"), but absent from channel_token_auth.md. | O:internal/client/handler.go; D:server/channel_permissions.md | Implement it. |
| INFRA-CONFLICT-12 | `insecure_skip_jwks_endpoint_safety_check`: the code doc string says it disables a check about JWKS endpoints "accessible from the internet". The docs say it bypasses the regex alternation check. | O:configtypes/types.go; D:server/authentication.md | Follow the docs. |
| INFRA-CONFLICT-13 | Valkey: engines.md says "no tests were performed by Centrifugal Labs". R:v6.0.3 says Valkey was added to the integration test suite. | D:server/engines.md; R:v6.0.3 | Cosmetic. Treat Valkey as supported and test it. |
| INFRA-CONFLICT-14 | `broker.type` allowed values: docs say `redis`, `nats`, `postgres`. Code default and accepted values also include `memory` (and hidden `redisnats`). | O:internal/app/engine.go | Accept `memory`. |
| INFRA-CONFLICT-15 | `client.presence_update_interval` (27s) exists in code but is undocumented. Presence liveness depends on it together with `presence_ttl` 60s. | O:configtypes/types.go | Mirror the default. Document it as observed. |
| INFRA-CONFLICT-16 | Metric `centrifugo_client_subscriptions_accepted` exists in code but is not in the docs metrics list. | L:metrics.go; R:v6.9.5 | Expose it. |
| INFRA-CONFLICT-17 | The consumer metrics description lists the consumers but omits Azure Service Bus. | D:server/observability.md §centrifugo_consumers_messages_processed_total | Cosmetic. |
| INFRA-CONFLICT-18 | The health endpoint is advertised for K8s liveness and readiness, but the code always returns 200 `{}` without checking the broker. | D:server/configuration.md §health; O:internal/health/health.go | Match it: unconditional 200 `{}`. Readiness checks are a possible extension and must be flagged as a known difference. |

### Open questions (undocumented behavior: do not guess)

- **OQ-ENG-01** Redis key layout, Lua script semantics, PUB/SUB channel naming and the cluster hash-tag scheme. Implementation detail, not documented. Interop with a reference node on the same Redis is out of scope unless the user wants mixed clusters. **Decision needed: must our server share a Redis with real Centrifugo nodes?**
- **OQ-ENG-02** How Redis Cluster is detected automatically. The mechanism is not documented.
- **OQ-ENG-03** A multi-node PG stream broker **without** `controller` enabled: the PG broker does not implement Controller, and the docs do not say what happens (no node discovery, or a startup error?). Needs a black-box test.
- **OQ-ENG-04** Memory engine: history eviction timing (TTL sweep cadence) and presence TTL for the memory presence manager. Not documented.
  - *Observed on ref v6.10.3 (black box, CS-HIST-02, 2026-11-16), history part:* memory and Redis behave the same at the API: after `history_ttl` the publications are gone and the position stays; after `history_meta_ttl` the position resets (offset 0, new epoch). Every history read extends the meta TTL on both engines. The memory engine counts TTLs in whole seconds and sweeps once a second (centrifuge `broker_memory.go`, `[src]`), so expiry happens up to about one second early or late. Presence TTL of the memory presence manager is still open.
- **OQ-ENG-05** NATS broker and the PG broker with join/leave and presence: is presence required from a separate manager? Docs: NATS + Redis presence is an example. The behavior with the default memory presence across nodes is undocumented.
- **OQ-HIST-01** Epoch format and generation (docs: "arbitrary string"). The memory epoch changes on restart. Nothing is specified for Redis or PG.
  - *Observed on ref v6.10.3 (black box, CS-REC-03, 2026-11-16):* a Redis restart without persistence changes the epoch, and recovery from the old position then gives no `recovered` flag, the new epoch and the new offset, as a memory node restart does. PG: step 2.6.
- **OQ-HIST-02** Client "reject unrecovered" subscription flag -> 112 disconnect (L:client.go). Belongs to the client protocol scope. Confirm with the protocol researcher.
- **OQ-HIST-03** Exact interaction of `recovery_max_publication_limit` with cache mode, and with `history_size` larger than the limit.
  - *Observed on ref v6.10.3 (black box, CS-HIST-04 / CS-REC-04, 2026-11-16), memory and Redis:* with `history_size` 10 and a limit of 8, a gap of exactly 8 is recovered and a gap of 9 is not (`recovered` unset, no publications). The per-namespace limit (2) takes precedence over the global one in the same way. Client history is capped at `history_max_publication_limit` for any `limit` above it, including `-1`. Cache mode (`history_size` 1) always returns the single latest publication, whatever position the client sends; with the limit at 8 the cache case cannot hit it.
- **OQ-AUTH-01** Clock-skew leeway for `exp`/`nbf`. The docs are silent and no leeway was observed in the verifier call.
- **OQ-AUTH-02** JWKS SSRF protections beyond the template allowlist (redirect following, response size limit, private-IP blocking). Not documented. Per CLAUDE.md we must add them regardless and record that as a known difference.
- **OQ-AUTH-03** Behavior when both `exp` and `expire_at` are absent but `sub_refresh`/refresh proxies are enabled.
- **OQ-PROXY-01** When is `SubscribeRequest.token` filled? (INFRA-CONFLICT-07)
- **OQ-PROXY-02** Semantics of `ConnectResult.profile` (proto field 12). Likely PRO. Not documented.
- **OQ-PROXY-03** Refresh-proxy retry interval after a backend error ("retries after some time"). Not quantified.
- **OQ-PROXY-04** Sub-refresh scheduling granularity ("up to one minute").
- **OQ-CONS-01** Exact at-least-once guarantees per consumer after a crash, as opposed to a graceful shutdown. Docs: "in many cases reliable at least once". Must not be claimed as exactly-once.
- **OQ-CONS-02** PostgreSQL outbox: are rows deleted per batch even when some commands fail with non-temporary errors? (Docs: such messages are marked processed, which implies deletion.)
- **OQ-OBS-01** `opentelemetry.consuming` semantics (code-only key).
- **OQ-OBS-02** Full label sets for `client_recovered_publications` and the exact histogram buckets. Not fully documented.
- **OQ-OPS-01** Helm chart values and contract live in a separate repo (`centrifugal/helm-charts`) that is not cloned. A separate research pass is needed if Helm parity is in scope.

## D. PRO: PRO-OQ

Source annex: [pro-feature-inventory.md](pro-feature-inventory.md) §5 (IDs there are `OQ-nn` = `PRO-OQ-nn` here).

- **PRO-OQ-01:** Is "Ephemeral client publications" (`/docs/pro/client_publications`) released in v6.10.3?
  - The page is unlisted and has no draft flag. No release note mentions `schemas` or `client_publication`.
  - Decide whether it is in scope.
- **PRO-OQ-02:** Server API JWKS minimum version.
  - server_api_enhancements.md says PRO v6.3.2, but no v6.3.2 tag exists.
  - RN v6.4.0 announces it, with a dead link `/docs/pro/server_api_auth`.
  - Proposed: use 6.4.0.
- **PRO-OQ-03:** PostgreSQL controller read replica (RN v6.8.0 links `scalability#read-replica-support`).
  - The current docs have no such section.
  - Are the config key names and status still valid?
- **PRO-OQ-04:** When did release signing start? install_and_run.md says v6.10.3; RN v6.10.0 already lists signed releases and SBOMs. This affects only our own release process.
- **PRO-OQ-05:** Snapshot creation and listing endpoints are not documented: paths, request/response shapes, permissions. Only the behavior and `label_filter` are described.
- **PRO-OQ-06:** Exact client-protocol fields for:
  - the debounce interval in the subscribe result
  - channel compaction negotiation (the id field in push/publication)
  - the distribution of `keep_latest_publication` deltas
  - These need the public protocol `.proto` (centrifugal/protocol) and black-box tests.
- **PRO-OQ-07:** Wire details of channel compaction for the JSON protocol.
- **PRO-OQ-08:** The idle-connections blog calls `websocket.use_write_buffer_pool` and `websocket.read_buffer_size` PRO options, but the OSS docs document them. Treat them as OSS?
- **PRO-OQ-09:** The full `distributed_rate_limit` configuration block. Only `.enabled` and `.storage_type` (=redis) appear. Redis address and reuse keys are undocumented.
- **PRO-OQ-10:** ClickHouse table DDL.
  - analytics.md shows only `SHOW CREATE TABLE` commands, not the column lists. Exact columns are known only for the v6.8.2 additions.
  - Need column lists, engines and partition keys.
- **PRO-OQ-11:** Admin OIDC details are not documented:
  - state/nonce handling, logout, `cookie_domain` (appears only in RN v6.10.0), session expiry
  - the CSRF model for server-side flow
  - viewer access to tracing and config
- **PRO-OQ-12:** The `info` API fields for PRO CPU/RSS process stats.
- **PRO-OQ-13:** Status of `update_push_status`. It is marked experimental with no promotion note. Treat it as experimental.
- **PRO-OQ-14:** Client JWKS provider cache behavior: refresh interval, TTL, behavior when the endpoint is unavailable, allowed algorithms. Only the server API JWKS behavior is documented.
- **PRO-OQ-15:** Label key/value validation and size limits, and the max number of labels.
- **PRO-OQ-16:** Push device ID format. It "embeds timestamp and provider" but the format is not public. We need our own format; compatibility is impossible to verify without black-box tests.
- **PRO-OQ-17:** Push `rate_limit.key` variable syntax. The docs say `device.id` and `device.user`; is it template syntax?
- **PRO-OQ-18:** The `connections` method: does it have limits or a max result size? Is it exposed in the GRPC API proto?
- **PRO-OQ-19:** Channel state events: payload batching size, retry caps, queue bound sizes, the GRPC message schema (proxy.proto is OSS public; verify `NotifyChannelState` is present at OSS v6.10.3).
- **PRO-OQ-20:** Release notes mention other keys with no docs:
  - `client.queue_initial_cap` (RN v6.10.0)
  - removed undocumented top-level push `cache` (RN v6.10.0)
  - `emulated_headers` (OSS, RN v6.9.0)
  - Should we support these?
- **PRO-OQ-21:** Rate-limit metric naming. `centrifugo_rate_limit_hits_over_limit` (distributed limiter) and `centrifugo_rate_limit_client_over_limit_count` both exist. Confirm the labels of the former (none are documented).
- **PRO-OQ-22:** Conflict for channel state events prerequisites. The old page (channel_state_events) says the redis engine is required; event_hooks.md also allows a separate redis presence manager. Proposed: use event_hooks.md (newer).
- **PRO-OQ-23:** Dictionary compression (PRO-30) and the `profile` JWT/proxy field are unreleased. Confirm they are excluded from scope.
- **PRO-OQ-24:** Minimum versions marked "≤6.0.0" predate the scanned range. Do we need the v5.x release notes for history, or is "≤6.0.0" enough?
