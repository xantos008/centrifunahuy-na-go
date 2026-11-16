# Engines: backends and their licenses

Cedrazeka connects to Redis, NATS and PostgreSQL as external services. None of them is bundled with Cedrazeka: the binary, `Dockerfile`, release files and `THIRD_PARTY_LICENSES` contain only the Go client libraries (e.g. `rueidis`, Apache-2.0), not a server. Operators choose and license the backend they run.

| Backend | Server license | Where this project uses it |
|---|---|---|
| Redis 8 | tri-license RSALv2 / SSPLv1 / AGPLv3. RSAL and SSPL are not OSI open-source licenses; AGPLv3 is. Using Redis only for tests does not make its license irrelevant (phase-2 condition C1) | test compose files only (`test/compat`, image pinned in `docs/ai/tooling-lock.md`) |
| Valkey | BSD-3-Clause | planned: library integration job (phase-2 step 2.10) |
| NATS server | Apache-2.0 | test compose files only: compat profiles `n`, `n-raw` (image pinned in `docs/ai/tooling-lock.md`) |
| PostgreSQL | PostgreSQL License | test compose files only: compat profiles `p`, `p-notify` (image pinned in `docs/ai/tooling-lock.md`) |

## Redis topologies tested

Verified black-box against Centrifugo v6.10.3 in the compatibility suite (phase-2 step 2.4), Redis 8.10.3:

| Topology | Address form | Test setup |
|---|---|---|
| Standalone | `redis://user:password@host:6379` | profiles `r`, `r-opt`, `r-lists` |
| Sentinel | `redis+sentinel://user:password@sentinel:26379?sentinel_master_name=mymaster` (credentials are the master's) | 1 master, 2 replicas, 1 sentinel |
| Cluster | plain `redis://user:password@node:6379`; Cluster is detected automatically | 3 masters, no replicas |
| Sharding | `address` array of standalone addresses | 2 standalone Redis |

Sentinel and Cluster tell clients the addresses that the Redis nodes announce. Servers must be able to reach exactly those addresses. In containers this means the nodes must announce the network address the servers use, not 127.0.0.1.

## NATS broker: what changes for clients

Observed black-box against Centrifugo v6.10.3 (phase-2 step 2.5; NATS 2.15.3, Redis presence manager). Ref and Cedrazeka behave the same:

- No history. Namespaces with `history_size`/`history_ttl` still load and the server starts. At runtime, every history-based operation answers `108 not available`: `history`, `history_remove`, subscribe with recovery, cache recovery, `force_positioning`, and fossil delta.
- Publish results carry no `offset`/`epoch`. `version` suppression and `idempotency_key` deduplication do not apply: every publication is delivered.
- Tags filters (`tf`) and presence via the Redis presence manager work as on Redis.
- Raw mode: messages go to and from NATS subjects unchanged, after `channel_replacements`. Tags are not carried. A push's `pub.channel` holds the NATS subject (e.g. `pub.raw-x` for channel `pub:raw-x`). With `allow_wildcards`, a client can subscribe to e.g. `news.*`.

## PostgreSQL broker: what changes for clients

Observed black-box against Centrifugo v6.10.3 (phase-2 step 2.6; PostgreSQL 18.7, PG broker and PG controller, memory presence). Ref and Cedrazeka behave the same:

- **Asynchronous delivery.** Publications reach subscribers through an outbox worker: by polling, or by `LISTEN/NOTIFY` with `use_notify`. A client that subscribes right after a publish can receive that publication as a push. In 50 publish-then-subscribe rounds with 3 publications each, ref pushed 149 of 150 and Cedrazeka 145.
- **Position is always tracked**, including namespaces without history. Publications carry offset/epoch, and `skip_history` does not keep a publication out of history. The subscribe reply on a stream that has no publications yet has no epoch.
- **Meta TTL.** History reads do not refresh it (docs). An expired position is removed by the periodic cleanup (`broker.postgres.cleanup_interval`, default 1m), not at the TTL itself, so it can outlive `history_meta_ttl` by up to that interval. The test configs set 1s; with that, the position resets after the TTL as on Redis. A history read of the reset stream does not create a new position: the next publish starts yet another epoch.
- **Payloads go through JSONB.** The bytes are normalized: `{"text":"x"}` comes back as `{"text": "x"}`. Fossil delta is not applied, so subscribers get full payloads. Use `binary_data` for payloads that must stay byte-exact.
- **Version suppression and idempotency work.** Pushes carry no `version` field.
- **Transactional publishing.** The `cf_stream_*` functions have identical signatures on both servers, and commit/rollback behaves as documented (KD-11, CS-PUB-07).

## Multi-node

Observed black-box against Centrifugo v6.10.3 (phase-2 step 2.7), two nodes sharing one backend. Ref and Cedrazeka behave the same:

- **Redis** (standalone, Sentinel, Cluster, sharding) and **NATS + Redis presence**: publications, join/leave, presence, server API calls by user, and history/recovery all work across nodes. A node stopped gracefully (`docker compose stop`) left the other's `info` list within a second in CS-ENG-03. A killed node's clients leave presence within `presence_ttl` + `client.presence_update_interval`; no leave push is sent for them.
- **PostgreSQL broker:**
  - Multi-node needs `controller` (type `postgres`). Without it the server starts and publications still cross nodes, but nodes do not discover each other: `info` lists only the node asked.
  - Presence uses the default memory manager, so it is per node: each node reports only its own clients. Join/leave pushes do cross nodes.

## Backend outage

Observed black-box against Centrifugo v6.10.3 (phase-2 step 2.8). Ref and Cedrazeka behave the same:

| Broker backend down | API publish | New client subscribe | After the backend is back |
|---|---|---|---|
| Redis | `100 internal server error`, `temporary: true` | connection closed, `3004 internal server error` | works again within seconds |
| NATS | same | succeeds (the client library buffers it during reconnect) | works again |
| PostgreSQL | same | succeeds (local subscription, delivery from the outbox) | works again; the schema must still exist, since servers create it only at startup |

Sentinel failover: after a new master is elected, publishing works again within seconds, and history written before the failover is on the new master (asynchronous replication; nothing was lost in the test). Live delivery is not clean at once: from about 3.5 s to 17.5 s after the failover, about one publication per second did not reach its subscriber, on ref and Cedrazeka alike. Redis PUB/SUB is at-most-once while servers reconnect. Clients that must not miss publications should use history recovery.

Engine configuration and failure behavior are documented in [phase 2](../plan/phase-2.md) as the profiles are verified.
