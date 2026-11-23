# Compatibility test suite (mandatory)

Decision: [ADR-0028](../architecture/adr/0028-compatibility-test-suite.md). The harness and first scenarios landed in implementation phase 1 ([plan](../plan/phase-1.md)).

## 1. Goal
Run the same scenarios against:
- `ref`: Centrifugo OSS v6.10.3, official image, pinned by digest;
- `sut`: our server.

The suite compares the wire behavior of both in these areas:
- protocol;
- replies and errors;
- authorization;
- subscriptions;
- history;
- recovery;
- presence;
- server API (HTTP and gRPC).

A diff fails CI unless the scenario is mapped to an **accepted** entry in [known-differences.md](../compatibility/known-differences.md).

## 2. Layout (`test/compat/`, separate Go module)

```
test/compat/
├── go.mod                     module …/test/compat; requires gorilla/websocket v1.5.3, nats.go v1.55.0 (more deps only when a scenario needs them)
├── docker-compose.yml         ref (image@digest) and sut (our linux binary from .build/, mounted into alpine:3.24: no image build,
│                              no network inside Docker); per profile: node B (ref-b, sut-b) and one backend set per
│                              side (redis-ref / redis-sut …) on separate networks ref-net / sut-net, Compose `profiles:`
├── configs/<profile>/         one config per engine profile (m, r, …), identical for ref and sut (sut adds --compat=centrifugo)
├── harness/
│   ├── target.go              Target{Name, BaseURL}; COMPAT_REF_URL / COMPAT_SUT_URL (default 127.0.0.1:18000 / :28000); WaitReady
│   ├── client.go              Session; raw WS client (exact JSON frames, batches, pings answered, close codes); HTTP API calls
│   ├── transcript.go          Transcript, Event, Normalize, Diff (line diff of canonical JSON, no HTML escaping)
│   ├── profile.go             profiles (nodes, backends), Profiles(t, …) filter, node lists, side-bound Session.Compose/Exec
│   ├── isolation.go           CS-ISO-01 (C2), run by TestMain before any scenario of a profile with backends
│   ├── run.go                 Run(t, name, scenario) per COMPAT_MODE = diff | golden | record; golden load/save
│   └── token.go               HS256 test tokens (RS/ES/EdDSA added with CS-AUTH-01)
├── scenarios/                 one file per area (below), plain Go test functions
└── testdata/golden/<profile>/ recorded ref transcripts per profile (committed)
```

Run modes (Makefile targets):

| Target | What it does |
|---|---|
| `make compat-up` / `make compat-down` | cross-compile the linux binary into `test/compat/.build/`, start / stop the targets and backends of `PROFILE` (default `m`; `down -v` flushes that profile's backends) |
| `make compat` | run all scenarios on ref and sut, diff. **CI default** |
| `make compat-golden` | sut vs committed golden transcripts (no ref container needed) |
| `make compat-record COMPAT_RECORD=CS-…[,…]` | re-record the **named** scenarios from ref, only after the diff is investigated and its cause established; there is no record-all (phase-2 condition C3) |
| `make compat PROFILE=r` | run against one engine profile (scenarios outside it are skipped) |
| `make compat-all COMPAT_PROFILES="m r …"` | for each profile: down, up, diff, golden, down |

## 3. Transcript and normalization
An Event is one of:
- `ws_send{frame}` / `ws_recv{frame}` (JSON, or decoded Protobuf rendered as canonical JSON);
- `ws_close{code, reason}`;
- `http{method, path, status, headers(allowlist), body}`;
- `grpc{method, code, body}`.

Normalization replaces **only** these values with stable placeholders (`<client:1>`, `<epoch:1>`…). Values are numbered in order of first appearance, so equality *within* a transcript is still checked. Scenarios must not use the volatile key names inside their own payloads.

| Field | Why volatile |
|---|---|
| `client`, `session` IDs | random UUIDs |
| `epoch` | random per stream |
| node `uid`, `name`, `version`, `uptime`, metrics in `info` | per process |
| timestamps (`time`, `expires`, `ttl` countdowns) | wall clock |
| order of map-valued results (presence, channels) | unordered upstream |
| `Server`/`Date` HTTP headers | per process (headers are not recorded at all) |
| `token` values in sent commands | tokens embed `exp` times and random ECDSA signatures; only the server's reaction is compared |
| the per-run suffix of channel names (`Session.Ch`) → `RUN` | keeps long-running targets' memory state (history, presence) apart between runs |
| keys of maps keyed by client ID (presence results) | numbered like `client` values, in a stable order (by `user`, then ID) |
| `info`: `metrics` and `uptime` dropped, all other values reduced to their type (`KeysOnly`) | node counters and observed-metric sets depend on process history; `uptime` is omitted (zero) while a node is younger than 1 s, so its presence depends on timing |

Never normalized: codes, messages, field presence or absence, offsets, `recovered`/`was_recovering`, data payloads, order of frames within a connection. Adding a normalization rule requires a KD entry or an amendment to this doc in the same PR.

## 4. Scenario catalogue (phase 1 delivers the ★ subset; the rest grows with each phase)

**Status 2026-11-15:** all ★ scenarios are implemented and green in diff mode (ref vs sut) and golden mode on the memory profile, plus CS-PUB-01, CS-HIST-01 (memory only; the R/P profiles come in phase 2). Test-config specifics: `ping_interval` 2s / `pong_timeout` 1s (CS-PROTO-04), and `allow_publish_for_client` in `pub:` so publisher and subscriber are separate sockets. CS-PROTO-03 records only the first reaction per malformed frame.

**Status 2026-11-16 (phase 2, step 2.2):** profile `r` (Redis 8 standalone, two nodes per target) is green in diff and golden mode for CS-PUB-01, CS-HIST-01, CS-REC-01/02, CS-PRES-01; `golden/r` is byte-identical to `golden/m` for these five.

**Status 2026-11-16 (phase 2, step 2.3):** these new scenarios are green in diff and golden mode:
- CS-HIST-02, CS-HIST-04, CS-REC-03, CS-REC-04, CS-PUB-03, CS-PUB-04 and CS-PUB-05 on `m` and `r`;
- CS-REC-05 on `r`;
- CS-ENG-CFG on `m`, plus the variants `r-opt` and `r-lists`.

**Status 2026-11-16 (phase 2, step 2.4):** Redis topologies `rs` (Sentinel), `rc` (Cluster, 3 masters) and `rsh` (2 shards), two nodes per target, are green in diff and golden mode for their tagged scenarios. Their golden files are byte-identical to `golden/r`.

**Status 2026-11-16 (phase 2, step 2.5):** NATS profiles `n` (NATS broker + Redis presence manager, two nodes per target) and `n-raw` (raw mode + wildcards, one node) are green in diff and golden mode. On `n` these are: CS-PUB-01, CS-PUB-04, CS-PUB-05, CS-PRES-01 and the new CS-HIST-03. On `n-raw` this is the new CS-PUB-06.

**Status 2026-11-16 (phase 2, step 2.6):** PostgreSQL profiles are green in diff and golden mode:
- `p`: PG broker and PG controller without NOTIFY, memory presence, two nodes per target. Tagged scenarios: CS-HIST-01/02/04, CS-PUB-01/03/04/05, CS-REC-01/02/04/05 and the new CS-PUB-07.
- `p-notify`: the same with `use_notify` on, one node. CS-PUB-07 only; its golden is byte-identical to `p`.

`sut-default` does not run on `p*`: the PG table prefix is `cf` in both modes, so it would join the sut cluster through the controller. Where a scenario publishes and then subscribes, it calls `s.AwaitDelivery()`. On `p*` this waits 500 ms for the outbox and records a note; elsewhere it does nothing.

**Status 2026-11-16 (phase 2, step 2.7):** the multi-node scenarios CS-ENG-01..05, CS-ENG-03b and CS-PRES-02 are green in diff and golden mode on their profiles. For `rs`, `rc`, `rsh` and `n`, their golden files are byte-identical to `golden/r`. For `p`, CS-ENG-03 and CS-ENG-04 are identical to `golden/r`. CS-ENG-01 and CS-ENG-05 differ only by the always-tracked position. In CS-ENG-02, presence is per node.

**Status 2026-11-16 (phase 2, step 2.8):** CS-ENG-06 on `r`, `n`, `p` and CS-ENG-07 on `rs` are green in diff and golden mode. While the backend is down, publish fails on all three. Subscribe fails only on Redis; NATS and PG accept it.

**Status 2026-11-16 (phase 2, step 2.9):** CS-NAME-02 (sut-only) is green on `r` and `n`.

Configs: `configs/r` is `configs/m` plus the engine section; `rs`, `rc`, `rsh` differ from `r` only in `engine.redis.address`; `r-opt` and `r-lists` are `r` plus one option each. Regenerate them with `jq` when `m` changes (see the step 2.3 notes in the plan). Note: `version` is a volatile key, so publication versions in CS-PUB-05 appear as placeholders. Equal versions keep equal placeholders, which is all the scenario needs.

**Phase 2** ([plan](../plan/phase-2.md)) adds engine profiles (`r`, `r-opt`, `r-lists`, `rs`, `rc`, `rsh`, `n`, `n-raw`, `p`, `p-notify`), two nodes per target, and the CS-ENG, CS-ISO and new CS-HIST/REC/PUB scenarios; see the profile and scenario tables below.

Each scenario lists the engine profiles it runs on: `M` memory, `R` Redis, `N` NATS (broker only), `P` PostgreSQL (broker and controller only; no presence).

### CS-PROTO: protocol and codecs (OSS-PROTO-*)
- ★ CS-PROTO-01: JSON connect, subscribe, publish, receive; batch of several commands in one frame (newline-delimited). M
- ★ CS-PROTO-02: the same over Protobuf (`?format=protobuf`, varint-delimited). M
- ★ CS-PROTO-03: strict JSON → 3501: duplicate keys, invalid UTF-8, trailing data, unknown command. M
- ★ CS-PROTO-04: empty-frame ping/pong; pong timeout disconnect code. M
- CS-PROTO-05: message size limit, decompression limit, permessage-deflate negotiation. M
- CS-PROTO-06: draft fields `flag`/`dict`/`profile` are tolerated. M

### CS-AUTH: connection authentication (OSS-AUTH-*)
- ★ CS-AUTH-01: valid HS256 token; RS256, ES256, EdDSA with configured public keys. M
- ★ CS-AUTH-02: expired token → 109; bad signature → 3500 (verify code table); `alg=none` rejected; alg confusion (RS key, HS token). M
- CS-AUTH-03: `aud`/`iss` checks; `nbf` handling (INFRA-CONFLICT-10); JWKS from a local static server. M
- CS-AUTH-04: refresh flow (`expires` → refresh command); `info` and `channels` claims. M
- CS-AUTH-05: connect proxy (HTTP backend fake, identical for both targets). M

### CS-SUB: subscriptions and permissions (OSS-PERM, OSS-SUB-*, OSS-CH)
- ★ CS-SUB-01: namespace permission matrix: `allow_subscribe_for_client`, `allow_subscribe_for_anonymous`, protected, user-limited `#` channels, unknown namespace → 102. M
- ★ CS-SUB-02: subscription token (valid, expired, wrong channel). M
- CS-SUB-03: server-side subscriptions (`subs` claim, `subscribe` API) and join/leave pushes. M R
- CS-SUB-04: channel limit (128), name length (255), already subscribed → 105. M
- CS-SUB-05: subscribe proxy; tags filter (`tf`); delta (fossil) negotiation. M

### CS-PUB: publication (OSS-API-01, OSS-SUB)
- ★ CS-PUB-01: client publish allowed or denied by namespace; server API publish with `tags`, `idempotency_key`, `skip_history`. M R N P
- CS-PUB-02: `broadcast` to many channels; batch API, ordered and `parallel`. M R
- CS-PUB-07: transactional publishing through the documented SQL surface of the PostgreSQL broker (KD-11). `cf_stream_publish` inside an application transaction: COMMIT delivers, ROLLBACK leaves no trace (the offset is reused). Also `p_idempotency_key` and `p_version` suppression (`out_suppressed`, `out_suppress_reason`), `cf_stream_publish_strict` raising `publish suppressed: version`, `cf_stream_top_position`, `cf_stream_remove_history`, and the signatures of all `cf_stream_*` functions. P (`p`, `p-notify`)
- CS-PUB-06: NATS raw mode (`channel_replacements` `:`→`.`) and `allow_wildcards`. A raw NATS message reaches channel subscribers unchanged; API and client publications reach the raw subject without wrapping; a wildcard channel receives matching subjects. N (`n-raw`)

### CS-HIST: history (OSS-HIST-*)
- ★ CS-HIST-01: `history` with `limit` unset (position only), `limit`, `since`, `reverse`; `history_size`/`history_ttl` trimming; max limit 300. M R P
- CS-HIST-02: `history_remove` keeps the top position; `history_meta_ttl`. M R P
- CS-HIST-03: history-based features on a NATS broker. Observed on ref: the config loads and startup succeeds. At runtime, API and client `history`, `history_remove`, subscribe with `recoverable`/`recover`, cache recovery, auto cache recovery, forced positioning and fossil delta each give `108 not available`. A plain subscribe and a publish succeed, and publish results carry no offset/epoch. N

### CS-REC: recovery and positioning (OSS-SUB-08/09, OSS-HIST)
- ★ CS-REC-01: reconnect with `recover`, `epoch`, `offset` within the window → `recovered=true`, exact missed publications, no duplicates. M R P
- ★ CS-REC-02: gap larger than the history → `recovered=false` and the new position. M R P
- CS-REC-03: epoch change (history reset, backend restart for memory) → `recovered=false`. M R
- CS-REC-04: `recovery_max_publication_limit`; cache recovery mode (latest publication). M R
- CS-REC-05: concurrent publish during subscribe-with-recover (1000 iterations, loss/duplicate check). R P
- CS-REC-06: a connection JWT creates a server-side subscription in a namespace with automatic cache recovery. The WebSocket and unidirectional HTTP-stream connect replies each contain only the latest publication. The clients send neither a recovery request nor a previous position. The `p` profile waits for outbox delivery before connecting. M R P
- CS-REC-07 waits for the stream metadata TTL to elapse while a positioned server-side subscriber answers pings. The next publication starts a fresh epoch at offset 1 and the WebSocket closes with code 3010. M R P

### CS-PRES: presence (OSS-PRES-*)
- ★ CS-PRES-01: `presence` and `presence_stats` via client and API; join/leave pushes. M R
- CS-PRES-02: presence TTL expiry after a node crash (`docker kill` of B): B's client leaves A's `presence_stats` within `presence_ttl` + `presence_update_interval` + 3 s. Bounded poll, last answer recorded. R (`r`)
- CS-PRES-03: `presence` and `presence_stats` return error 103 before the client subscribes to a channel with subscriber-only presence. After subscribing, the presence reply carries `conn_info` from the connection JWT and `chan_info` from the subscription JWT. The `p` profile uses per-node memory presence. M R P

### CS-API: server API (OSS-API-01..04)
- ★ CS-API-01: auth: `X-API-Key`, `?api_key=`, `Authorization: apikey`; bad key → 401 with empty body; non-POST → 405; bad JSON → 400; unknown path → 404. M
- ★ CS-API-02: every HTTP route (21) with a minimal valid request and one invalid request (error code and message). M
- CS-API-03: the 21 OSS methods over the keyed gRPC API on M. The harness reuses CS-API-02 inputs and the generated service descriptor. An unauthenticated Info gets status 16. All 19 PRO-only methods return status 12 and their HTTP paths return 404 on the pinned reference. The loopback endpoints for ref and sut default to ports 18010 and 28010 and can be overridden with `COMPAT_REF_GRPC_ADDR` and `COMPAT_SUT_GRPC_ADDR`.
- CS-API-04: `error_mode` variants; `disconnect`, `unsubscribe`, `refresh` by user across 2 nodes. M R

### CS-RPC: client RPC (phase 3)

- CS-RPC-01 runs on the ping-only `m` profile. The custom RPC ping method returns an empty result. The default name and an unknown method return error 104. Verified in diff and golden mode.
- CS-RPC-02 exercises the RPC proxy over gRPC on `r` and HTTP on `p`. The `probe` namespace echoes the request data or returns application error 470. An unqualified unknown method returns 108 with proxying outside namespaces disabled. An unknown namespace returns 104. Verified in diff and golden mode on both profiles.
- CS-RPC-03 calls a backend that waits for cancellation. On `r` and `p`, the one-second proxy deadline produces temporary error 100. A second RPC on the same WebSocket checks that the backend observed its request context ending. A third RPC echoes data.

### CS-PROXY: application callbacks (phase 3)

Each server calls a separate instance of `cmd/proxyfixture` on its private Compose network. Profiles `m` and `p` use HTTP JSON. Profile `r` uses the generated gRPC proxy service. The fixture has no published host ports. CS-PROXY-01..03 passed in diff and golden mode on all three profiles.

- CS-PROXY-01 uses a tokenless connect callback to grant a server-side subscription. The namespace has automatic cache recovery disabled. Both WebSocket and unidirectional HTTP-stream connect with the callback override off and on. The scenario checks the latest publication at offset 2 and the connection/subscription info in presence.
- CS-PROXY-02 uses a connection JWT and subscribes to a proxied channel. It checks reply data, connection info from the JWT and channel info from the subscribe callback in presence, and propagation of application error 470 when the callback rejects the subscription.
- CS-PROXY-03 publishes through the client protocol. The callback rewrites the data, attaches a tag and sets `skip_history` on the second publication. A separate subscriber checks both deliveries. Memory and Redis retain only the first publication. The PostgreSQL stream broker retains both with consecutive offsets. A third request receives the callback's rejection.
- CS-PROXY-04 starts two connections with expiring JWTs. One receives updated connection info in presence and can still receive a publication. The other closes with code 3005 after the callback reports expiry. Both callbacks require the JWT connection meta. Runs on `r` and `p`.
- CS-PROXY-05 starts two expiring subscriptions on one WebSocket. Renewal updates channel info in presence and preserves the original connection info. Expiry produces an unsubscribe push (2501) for the second channel. The first still receives a publication. Both callbacks require the channel name and JWT connection meta. In both refresh scenarios the PostgreSQL stream broker includes the stream epoch and offset in delivered publications. Redis omits those fields on these channels.

### CS-TRANS: other transports (phase 3)

- CS-TRANS-01: unidirectional SSE and HTTP-stream POST connect, initial subscription, framing and API publication delivery. Verified in diff and golden mode on `m`.
- CS-TRANS-02 exercises unidirectional WebSocket via a first-frame connect request and via `cf_connect`. Both clients receive an API publication, a disconnect push and a WebSocket close frame with code 4001 and the configured reason. M
- Remaining transport coverage includes emulated SSE and HTTP-stream, unidirectional gRPC, and the other transports in the phase-3 plan.

### CS-ENG: multi-node (phase 2)
Two sut nodes against two ref nodes. Each scenario first waits until `info` on both nodes lists two nodes. `Session.OnB` points calls and new connections at node B.
- CS-ENG-01: cross-node fan-out. The subscriber is on A; on B, a client publish, an API publish, and a second subscriber whose join and leave A receives. R (`r`, `rs`, `rc`, `rsh`) N P
- CS-ENG-02: `presence` and `presence_stats` from the client and the API, on both nodes. On P they are per node (memory presence). R (`r`, `rs`, `rc`, `rsh`) N P
- CS-ENG-03: `info` lists both nodes; after `docker compose stop` of B, A lists one node within 20 s; B is started again. R N P
- CS-ENG-03b: B replaced by a one-off copy with `configs/p-noctl` (PG broker without controller, OQ-ENG-03). Neither node lists the other, and an API publish on B still reaches A. P
- CS-ENG-04: `subscribe`, `unsubscribe`, `refresh` and `disconnect` by user, called on A, for a user connected to B. R N P
- CS-ENG-05: position on A, publications via A's API, then `history` and recovery on B. R (`r`, `rc`) P
- CS-ENG-06: the broker backend is stopped; the API publish error and one subscribe attempt are recorded. After the start, publish and subscribe work again within 30 s, and a fresh channel delivers. R (`r`) N P
- CS-ENG-07: `SENTINEL FAILOVER mymaster`. Once a new master is elected, publishing works again within 30 s, and history holds the publications made before the failover. Then delivery must work for 5 s without a miss (unrecorded probes, ~20/s) before one delivered publication is recorded. R (`rs`)

### CS-NAME: names (sut-only, ADR-0022)
- ★ CS-NAME-01: default and compatibility modes expose their respective metrics prefixes. The branded HTTP error-mode headers are exercised here. The environment and gRPC service-name checks are covered by the separate server tests in `internal/brand/e2e`.
- CS-NAME-02: on `r` and `n`, default mode prefixes Redis keys and channels and NATS subjects with `cedrazeka`, compatibility mode with `centrifugo`; checked in the shared sut backends (Redis presence key, Redis `PUBSUB CHANNELS`, NATS `/subsz`). PG `table_prefix` is `cf` in both modes.

### PRO and EXTENSION
- PRO scenarios are sut-only. Expected transcripts are hand-written from the docs; each file cites the doc path and the docs commit.
- EXTENSION scenarios are sut-only, plus a check that every UPSTREAM golden transcript is unchanged with default config.

## 5. Determinism rules
- No sleeps as synchronization. Wait for an expected event with a deadline (default 5 s; recorded in the transcript on timeout).
- Fixed test keys and secrets live in `harness/token.go` (test-only values, not secrets).
- Each scenario uses unique channel names (`cs-<id>-<rand>`) and normalizes the random suffix, so scenarios can run in parallel against one target.
- Backends are flushed between engine profiles, not between scenarios.
- **Exception: waiting out a configured TTL.** CS-HIST-02 sleeps for fixed periods, because every history read extends the meta TTL, so polling would keep the state alive. Each wait leaves more than a second of margin, because the memory engine counts TTLs in whole seconds and sweeps once a second. Waits for a state change (a backend coming back) poll with `Session.APIUntil` and record only the last answer.
- **Outcome scenarios** (CS-REC-05) do their work in a throw-away session and record one note with the checked counts.
- **CS-ISO-01 (isolation, phase-2 condition C2)** runs in `TestMain` for every profile with backends, before any scenario; a failure aborts the profile. It checks that every server container is attached only to its side's network, that each backend hostname (`redis`, …) resolves inside every server container to that side's backend container, that the two sides' backends share no mount, and that a probe key written through one side's Redis is absent on the other. Verified to fail (2026-11-16) when `sut` is also attached to `ref-net` and when the `redis` alias on `sut-net` points to `redis-ref`.

## 6. Exit criteria per implementation phase
Each phase in the [feature matrix](../compatibility/feature-matrix.md#implementation-order) names the CS-* groups that must be green on all applicable engine profiles. A matrix row moves to `verified` only when its scenarios are green in `make compat` (diff mode).

When running an additional local Compose stack, set `COMPAT_COMPOSE_FILE` and `COMPOSE_PROJECT_NAME` to the file and project used for startup. Point the HTTP and gRPC overrides at that stack's published ports. Before a lifecycle action such as restart, the harness checks the side's node A HTTP address against the published Compose binding. This check also applies when a scenario controls a backend or node B.
