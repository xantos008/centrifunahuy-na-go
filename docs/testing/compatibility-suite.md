# Compatibility test suite (mandatory)

Decision: [ADR-0028](../architecture/adr/0028-compatibility-test-suite.md). Status: specified. The harness and the first scenarios are built in implementation phase 1 ([plan](../plan/phase-1.md)).

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
├── go.mod                     module …/test/compat; requires gorilla/websocket v1.5.3 (more deps only when a scenario needs them)
├── docker-compose.yml         ref (image@digest) and sut (our linux binary from .build/, mounted into alpine:3.24: no image build,
│                              no network inside Docker); backends are added with the phase-2 engine profiles
├── configs/<engine>/          one config per engine profile, identical for ref and sut (sut adds --compat=centrifugo)
├── harness/
│   ├── target.go              Target{Name, BaseURL}; COMPAT_REF_URL / COMPAT_SUT_URL (default 127.0.0.1:18000 / :28000); WaitReady
│   ├── client.go              Session; raw WS client (exact JSON frames, batches, pings answered, close codes); HTTP API calls
│   ├── transcript.go          Transcript, Event, Normalize, Diff (line diff of canonical JSON, no HTML escaping)
│   ├── run.go                 Run(t, name, scenario) per COMPAT_MODE = diff | golden | record; golden load/save
│   └── token.go               HS256 test tokens (RS/ES/EdDSA added with CS-AUTH-01)
├── scenarios/                 one file per area (below), plain Go test functions
└── testdata/golden/           recorded ref transcripts (committed)
```

Run modes (Makefile targets):

| Target | What it does |
|---|---|
| `make compat-up` / `make compat-down` | cross-compile the linux binary into `test/compat/.build/`, start / stop both targets |
| `make compat` | run all scenarios on ref and sut, diff. **CI default** |
| `make compat-golden` | sut vs committed golden transcripts (no ref container needed) |
| `make compat-record` | re-record golden from ref (only on re-baseline; diff reviewed in the PR) |
| `COMPAT_ENGINE=redis make compat` | restrict to one engine profile |

## 3. Transcript and normalization
An Event is one of:
- `ws_send{frame}` / `ws_recv{frame}` (JSON, or decoded Protobuf rendered as canonical JSON);
- `ws_close{code, reason}`;
- `http{method, path, status, headers(allowlist), body}`;
- `grpc{method, code, body}`.

Normalization replaces **only** these values with stable placeholders (`<client:1>`, `<epoch:1>`…). Values are numbered in order of first appearance, so equality *within* a transcript is still checked:

| Field | Why volatile |
|---|---|
| `client`, `session` IDs | random UUIDs |
| `epoch` | random per stream |
| node `uid`, `name`, `version`, `uptime`, metrics in `info` | per process |
| timestamps (`time`, `expires`, `ttl` countdowns) | wall clock |
| order of map-valued results (presence, channels) | unordered upstream |
| `Server`/`Date` HTTP headers | per process |

Never normalized: codes, messages, field presence or absence, offsets, `recovered`/`was_recovering`, data payloads, order of frames within a connection. Adding a normalization rule requires a KD entry or an amendment to this doc in the same PR.

## 4. Scenario catalogue (phase 1 delivers the ★ subset; the rest grows with each phase)

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

### CS-HIST: history (OSS-HIST-*)
- ★ CS-HIST-01: `history` with `limit` unset (position only), `limit`, `since`, `reverse`; `history_size`/`history_ttl` trimming; max limit 300. M R P
- CS-HIST-02: `history_remove` keeps the top position; `history_meta_ttl`. M R P
- CS-HIST-03: history on a NATS-backed namespace is rejected at config load (capability). N

### CS-REC: recovery and positioning (OSS-SUB-08/09, OSS-HIST)
- ★ CS-REC-01: reconnect with `recover`, `epoch`, `offset` within the window → `recovered=true`, exact missed publications, no duplicates. M R P
- ★ CS-REC-02: gap larger than the history → `recovered=false` and the new position. M R P
- CS-REC-03: epoch change (history reset, backend restart for memory) → `recovered=false`. M R
- CS-REC-04: `recovery_max_publication_limit`; cache recovery mode (latest publication). M R
- CS-REC-05: concurrent publish during subscribe-with-recover (1000 iterations, loss/duplicate check). R P

### CS-PRES: presence (OSS-PRES-*)
- ★ CS-PRES-01: `presence` and `presence_stats` via client and API; join/leave pushes. M R
- CS-PRES-02: presence TTL expiry after an abrupt disconnect (presence is eventually consistent; bounded wait). R

### CS-API: server API (OSS-API-01..04)
- ★ CS-API-01: auth: `X-API-Key`, `?api_key=`, `Authorization: apikey`; bad key → 401 with empty body; non-POST → 405; bad JSON → 400; unknown path → 404. M
- ★ CS-API-02: every HTTP route (21) with a minimal valid request and one invalid request (error code and message). M
- CS-API-03: the same set over gRPC (`centrifugal.centrifugo.api.CentrifugoApi`); PRO-only rpc behavior on OSS (CORE OQ-04, recorded from ref). M
- CS-API-04: `error_mode` variants; `disconnect`, `unsubscribe`, `refresh` by user across 2 nodes. M R

### CS-TRANS: other transports (phase 3)
HTTP-stream, SSE with emulation, uni WS / SSE / HTTP-stream / gRPC: CS-PROTO-01 run over each.

### CS-ENG: multi-node (phase 2)
Two sut nodes against each of R, N, P versus two ref nodes: cross-node fan-out, presence, `info` node list, user-targeted disconnect.

### CS-NAME: names (sut-only, ADR-0022)
- ★ CS-NAME-01: default mode reads `CEDRAZEKA_*` env and exposes `cedrazeka_*` metrics; compatibility mode reads `CENTRIFUGO_*` and exposes `centrifugo_*`; the gRPC API answers on both service names in compatibility mode.

### PRO and EXTENSION
- PRO scenarios are sut-only. Expected transcripts are hand-written from the docs; each file cites the doc path and the docs commit.
- EXTENSION scenarios are sut-only, plus a check that every UPSTREAM golden transcript is unchanged with default config.

## 5. Determinism rules
- No sleeps as synchronization. Wait for an expected event with a deadline (default 5 s; recorded in the transcript on timeout).
- Fixed test keys and secrets live in `harness/token.go` (test-only values, not secrets).
- Each scenario uses unique channel names (`cs-<id>-<rand>`) and normalizes the random suffix, so scenarios can run in parallel against one target.
- Backends are flushed between engine profiles, not between scenarios.

## 6. Exit criteria per implementation phase
Each phase in the [feature matrix](../compatibility/feature-matrix.md#implementation-order) names the CS-* groups that must be green on all applicable engine profiles. A matrix row moves to `verified` only when its scenarios are green in `make compat` (diff mode).
