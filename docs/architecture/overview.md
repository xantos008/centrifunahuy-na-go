# Architecture overview

Status: **proposed**, revised 2026-11-15 after user decisions DEC-01…DEC-09. Nothing here is implemented yet. Decisions are in [adr/](adr/), and feature IDs refer to [feature-matrix.md](../compatibility/feature-matrix.md).

## 0. Decisions that shape this document

| DEC | Decision | ADR |
|---|---|---|
| 01 | Fork Centrifugo OSS v6.10.3 (Apache-2.0); `centrifuge` stays an unmodified dependency; PRO equivalents are written by us from public docs | [0002](adr/0002-reuse-vs-independent-core.md), [0029](adr/0029-upstream-sync-and-pro-integration.md) |
| 02 | Deployment-per-tenant in v1; nothing may block a first-class tenant later | [0014](adr/0014-multi-tenancy.md) |
| 03 | No bidirectional gRPC client transport | [0021](adr/0021-no-bidirectional-grpc-transport.md) |
| 04 | Own names by default plus an opt-in `compat: centrifugo` mode | [0022](adr/0022-naming-and-compatibility-mode.md) |
| 05 | No mixed clusters with Centrifugo nodes | [0023](adr/0023-no-mixed-clusters.md) |
| 06 | Undocumented PRO behavior is out of mandatory scope | [0024](adr/0024-undocumented-pro-scope.md) |
| 07 | Admin API only from confirmed public contracts; extensions under `/admin/x/v1` | [0025](adr/0025-admin-api-contracts.md) |
| 08 | No GitHub MCP or Context7; all tooling project-local | [0026](adr/0026-ai-tooling-project-local.md) |
| 09 | Helm deferred until the core is stable | [0027](adr/0027-helm-deferred.md) |
| — | Mandatory differential compatibility suite | [0028](adr/0028-compatibility-test-suite.md) |
| 10 | Project license: Apache-2.0 | [0030](adr/0030-project-license.md) |

## 1. Shape of the system

The system is a single Go binary, the upstream Centrifugo server plus our PRO modules. Every node is identical and shares nothing with the others. All cross-node state lives in pluggable backends (Redis, NATS, PostgreSQL), as upstream ([ADR-0019](adr/0019-distributed-architecture.md)).

Legend: `[U]` inherited from upstream (Apache-2.0 / MIT), `[P]` new PRO-equivalent module, `[E]` our extension, `[B]` names via `internal/brand`.

```
                 ┌──────────────────────────────── node (one process) ────────────────────────────────┐
 clients ──WS/H2/H3──▶ transports [U] ─┐                                                              │
 (SDKs)  ──SSE/HTTP-stream/uni──────▶  │ frames  ┌─────────────────┐ cmds ┌──────────────────────────┐ │
         ──uni gRPC [U][B]──────────▶  ├────────▶│ protocol codecs │─────▶│ client session [U]       │ │
                                       │◀────────│ [U] centrifugal/│◀─────│ (centrifuge: queue,      │ │
                                       │ batches │ protocol v0.23.1│      │  batching, ping, limits) │ │
                                       │         └─────────────────┘      └────────────┬─────────────┘ │
                                       │   authn [U] JWT/JWKS/proxy ◀──────────────────┤               │
                                       │   authz [U] ns perms │ [P] caps/CEL/patterns/labels/blocking  │
                                       │   [P] rate limits, tracing ◀── OnCommandRead/Processed hooks  │
                                       │                                   ┌───────────▼─────────────┐ │
 backend ──HTTP/gRPC server API [U][B]─▶ api executor [U] + [P] methods ──▶│ centrifuge.Node [U]     │ │
 backend ◀── proxy HTTP/gRPC [U][B] ◀────────────────────────────────────── │ hub, fan-out, positions,│ │
 queues ──▶ consumers [U] (Kafka, PG outbox, NATS JS, Redis Streams, …) ──▶│ recovery, delta, tags   │ │
                                                                            └──┬──────┬──────┬──────┬─┘ │
                                    per-channel engine selection [P] via GetBroker/GetPresenceManager/   │
                                    GetMapBroker;  Controller via SetController [P]                      │
                                         Broker [U] │ PresenceManager [U] │ MapBroker [U] │ Controller [U]│
 admin UI [P] (web/admin, ours) ◀─▶ admin API [U] + [P] OIDC/RBAC/audit/snapshots/config viewer + [E] /x │
 push API [P] ──▶ push service [P] ──▶ queue (Redis Streams / PG) ──▶ workers ──▶ FCM/APNs/HMS/WebPush   │
 analytics exporter [P] ──▶ ClickHouse        observability [U] Prometheus, OTel, zerolog  (+[P] metrics)│
                 └──────────────────────────────────────────┬───────────────────────────────────────────┘
                                                            ▼
                       Redis (standalone/Sentinel/Cluster) · NATS · PostgreSQL · ClickHouse
```

## 2. Packages

Upstream packages are listed as they exist at v6.10.3 (verified tree listing). New packages are marked NEW.

| Layer | Package | Origin | Responsibility |
|---|---|---|---|
| Entry / composition | `main.go`, `internal/app` | U, edited at marked `// cedrazeka:pro` points | config → constructors → run; the only place PRO modules are wired |
| Names | `internal/brand` | NEW | default names, compatibility mode (ADR-0022) |
| Config / CLI | `internal/config`, `configtypes`, `confighelpers`, `configredact`, `cli` | U | viper + envconfig, `defaultconfig`/`configdoc`, redaction. PRO keys are added to `configtypes` |
| Transports | `internal/websocket`, `uniws`, `unisse`, `unihttpstream`, `unigrpc`, `wt` (+ centrifuge HTTP-stream/SSE) | U | framing only |
| Core | `github.com/centrifugal/centrifuge` (module) | U, unmodified | node, sessions, hub, recovery, delta, engine interfaces |
| Engines | centrifuge memory/redis; `internal/natsbroker`, `redisnatsbroker`, `pgstreambroker`, `pgmapbroker`, `pgschema`, `pgshard`, `redisshard`, `redisiam`, `controllers` | U | Broker, PresenceManager, MapBroker, Controller implementations |
| Auth | `internal/jwtverify`, `jwks`, `jwtutils` | U | JWT and JWKS |
| Proxy | `internal/proxy`, `proxyproto` | U | backend calls |
| Server API | `internal/api`, `apiproto` | U | HTTP and gRPC executor |
| Consumers | `internal/consuming`, `pgoutbox`, `redisqueue` | U | async consumers |
| Admin | `internal/admin` | U | OSS admin endpoints |
| Observability | `internal/metrics`, `logging`, `telemetry` | U | Prometheus, zerolog, OTel |
| PRO modules | `internal/pro/{authz, labels, cel, patterns, blocking, userstatus, connections, ratelimit, tracing, snapshots, configview, oidc, rbac, audit, analytics, engines, controller, push, …}` | NEW | one package per PRO feature group (ADR-0029) |
| Admin UI | `web/admin` | NEW | our SPA (phase 5) |
| Compatibility suite | `test/compat` | NEW, separate module | ADR-0028 |

Interfaces are added only where at least two implementations or a test seam are known (CLAUDE.md "explicit interfaces" together with "no speculative abstractions").

## 3. Hot path: a publication reaches subscribers (inherited)

1. A server API `publish` is authorized (API key; JWKS is PRO-12) and goes to the executor in `internal/api`.
2. `centrifuge.Node` picks the broker for the channel: `Config.GetBroker(channel)` when set (PRO-21), otherwise the default broker.
3. `Broker.Publish` appends to history (when configured) and returns `StreamPosition{offset, epoch}`, then publishes over the inter-node PUB/SUB.
4. Each node delivers to its local subscribers. Tags filter, delta and batching are applied in centrifuge. PRO server-side tags filter (PRO-25) and grouping (PRO-31) hook in there.
5. Each session's writer drains the byte-bounded queue (`client.queue_max_size`). On overflow the client gets disconnect `3008 slow`.

Delivery and ordering are as in `docs/getting-started/design.md`:
- at-most-once by default;
- at-least-once within the history window when positioning or recovery is on;
- per-channel ordering for sequential publishes;
- no exactly-once claim.

## 4. Recovery (inherited)

The centrifuge implementation is used as is. The externally visible results are:
- the missed publications with `recovered: true`;
- `recovered: false` plus a new position (gap, limit or epoch change);
- the latest publication in cache mode.

They are proven by the CS-REC scenarios against `ref` ([compatibility-suite.md](../testing/compatibility-suite.md)), including a concurrent-publish race scenario.

## 5. Control plane

Node info, user-targeted commands, emulation routing, surveys (snapshots, connections, user status) and PRO-15 propagation all go through `centrifuge.Controller` and the survey/notification hooks. By default the Redis broker doubles as controller (`SetBroker` picks it up when it implements `Controller`). The OSS PostgreSQL controller lives in `internal/controllers`. PRO-22 adds an explicit Redis or NATS controller via `Node.SetController` ([ADR-0007](adr/0007-controller.md)).

## 6. Cross-cutting

- **Limits everywhere** (upstream limits are inherited; new PRO limits follow the same rule): every limit has a config key, a metric and an error code.
- **Backpressure:** bounded per-session queues (inherited). New PRO workers (push, ClickHouse, tracing) use bounded pools and drop counters.
- **Graceful shutdown:** the inherited order. PRO modules register into the upstream `closeOnStop` sequence.
- **Tenant readiness (DEC-02):** new PRO storage carries a constant `scope` (EXT-05); no tenant singletons outside `internal/app`.
- **Security:** [threat-model.md](../security/threat-model.md).
- **Provenance and licensing:** [provenance.md](../legal/provenance.md), [naming-and-trademarks.md](../legal/naming-and-trademarks.md).

## 7. Broker independence check (user requirement F)

Verified against centrifuge `ae446d43faf0` and OSS v6.10.3 `internal/app/engine.go`:

| Interface | Methods (verbatim signatures, abbreviated types) | Implementations available |
|---|---|---|
| `Broker` | `RegisterBrokerEventHandler(h)`, `Subscribe(ch...)`, `Unsubscribe(ch...)`, `Publish(ch, data, PublishOptions) (PublishResult, error)`, `PublishJoin(ch, *ClientInfo)`, `PublishLeave(ch, *ClientInfo)`, `History(ch, HistoryOptions) ([]*Publication, StreamPosition, error)`, `RemoveHistory(ch)` | memory, Redis (centrifuge); NATS (`natsbroker`), Redis+NATS (`redisnatsbroker`), PostgreSQL (`pgstreambroker`) |
| `PresenceManager` | `Presence(ch)`, `PresenceStats(ch)`, `AddPresence(ch, clientID, *ClientInfo)`, `RemovePresence(ch, clientID, userID)` | memory, Redis |
| `MapBroker` | `RegisterEventHandler`, `Subscribe`, `Unsubscribe`, `Publish(ctx, ch, key, opts)`, `Remove`, `ReadStream`, `ReadState`, `Stats`, `Clear` | memory, Redis (centrifuge); PostgreSQL (`pgmapbroker`) |
| `Controller` | `RegisterControlEventHandler(h)`, `PublishControl(data, nodeID, shardKey)` | Redis broker (implicit), PostgreSQL (`controllers`); PRO-22 adds Redis/NATS explicit |

Findings:
- **Core logic depends only on these interfaces.** `centrifuge.Node` holds interface values set via `SetBroker`, `SetPresenceManager`, `SetMapBroker` and `SetController`. Backend choice happens only in `internal/app/engine.go` (`switch cfg.Broker.Type { memory | redis | nats | postgres | redisnats }`, with similar switches for presence and map broker).
- **Capability differences are real and documented, not hidden:**
  - NATS has no history (so no recovery) and no presence;
  - PostgreSQL has no presence;
  - Redis covers every role.
  
  Config validation must reject impossible combinations. Upstream does this; CS-HIST-03 checks it.
- **Per-channel selection** for PRO-21 needs no core change: `Config.GetBroker`, `GetPresenceManager` and `Map.GetMapBroker` already exist.
- **Gap to watch:** the selection switch in `internal/app/engine.go` is a closed list. PRO-21 named engines will add a registry (`internal/pro/engines`) that builds named instances with the same constructors. No new interface is needed.

Conclusion: the architecture is not tied to one broker. Redis, NATS and PostgreSQL plug in through the same four interfaces, and the remaining differences are capability limits that upstream documents.

## 8. Implementation order

See [feature-matrix.md § Implementation order](../compatibility/feature-matrix.md#implementation-order) and the detailed [phase 1 plan](../plan/phase-1.md).

## 9. Risks (top)

| ID | Risk | Mitigation |
|---|---|---|
| R-MERGE | Upstream upgrades conflict with our edits | minimal edits, hooks first, marked wiring points, upgrade procedure (ADR-0029) |
| R-LEGAL | Naming and trademark use in compatibility mode; license mix | naming-and-trademarks.md, legal review before release, DEC-10/12 |
| R-LIB | A PRO feature needs a `centrifuge` change | check upstream hooks first; vendor fork only via ADR |
| R-SPEC | PRO behavior is under-documented (snapshots, OIDC sessions, ClickHouse DDL, push device IDs) | DEC-06: open question, never guessed; EXTENSION label |
| R-SEC | Push SSRF, OIDC, CEL bounds, admin RBAC | threat model, fuzz, deny-by-default tests |
| R-PERF | PRO hooks on the hot path (rate limits, tracing) slow fan-out | benchmarks before and after each hook; hooks off by default |
| R-SUITE | Compatibility-suite flakiness hides regressions | determinism rules (no sleeps, deadlines), golden mode fallback |
