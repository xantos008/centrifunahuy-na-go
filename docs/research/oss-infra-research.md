<!-- Phase 0 research annex. Derived from public docs/OSS source at the pinned baseline. Open questions here are mirrored in open-questions.md as INFRA-OQ-*. -->

# OSS infrastructure research: engines, history/recovery, presence, auth, proxy, consumers, observability, ops

Scope: Centrifugo OSS **v6.10.3** (released 2026-11-08). Clean-room research: documentation and public behavior only. No code was copied. Source code was read only to confirm config keys, defaults and wire field names, and to find places where docs and code disagree.

Date accessed: 2026-11-15.

## Source legend

| Prefix | Meaning | Public URL mapping |
|---|---|---|
| `D:` | `centrifugal.dev` docs clone, commit `69da714`, `docs/` = v6 | `docs/server/X.md` -> `https://centrifugal.dev/docs/server/X` (same for `getting-started/`, `pro/`) |
| `B:` | `centrifugal.dev/blog/` | `blog/YYYY-MM-DD-slug.md` -> `https://centrifugal.dev/blog/YYYY/MM/DD/slug` |
| `O:` | Centrifugo OSS clone, commit `4b6d86e` ("Prepare v6.10.3 (#1313)") | `https://github.com/centrifugal/centrifugo/blob/v6.10.3/<path>` |
| `L:` | `centrifugal/centrifuge` library clone | `https://github.com/centrifugal/centrifuge/blob/master/<path>` (version pinned by `O:go.mod`) |
| `R:` | GitHub release notes, `gh release view vX --repo centrifugal/centrifugo` | `https://github.com/centrifugal/centrifugo/releases/tag/vX` |

Status vocabulary: **GA** means the docs do not mark it experimental or deprecated. **EXP** means the docs say "Experimental". **DEP** means deprecated. **PRO** means a PRO-only extension that is mentioned in OSS docs; it is listed only so the boundary is clear.

---

## 1. Feature table

### 1.1 Engines / brokers / presence managers / controller (OSS-ENG)

| ID | Feature | Status | Min ver | Config keys | API / protocol | Storage | Sources | Notes |
|---|---|---|---|---|---|---|---|---|
| OSS-ENG-01 | Engine = Broker + Presence Manager, configured as one entity | GA | <v6 | `engine.type` (`memory` default, `redis`) | — | — | D:server/engines.md §What is Engine, §engine.type; O:internal/configtypes/engine.go `Engine` | Engine type values in code: `memory`, `redis` only (O:internal/app/engine.go `configureEngines`). |
| OSS-ENG-02 | Memory engine | GA | <v6 | `engine.type=memory` | — | process memory | D:server/engines.md §Memory engine | Single node only. History and presence are lost on restart. Supports "all engine features". v6.10.0: memory history split into 64 locked parts (R:v6.10.0). |
| OSS-ENG-03 | Redis engine (standalone) | GA | <v6 | `engine.redis.address` (string or array, default `redis://127.0.0.1:6379`), `prefix` (`centrifugo`), `force_resp2`, `tls`, `presence_ttl` (`60s`), `connect_timeout` (`1s`), `io_timeout` (`4s`), `db` (0), `user`, `password`, `client_name` | — | Redis Streams (history), ZSET+HASH (presence), PUB/SUB | D:server/engines.md §Redis engine, §engine.redis; O:internal/configtypes/redis.go | **Minimum Redis 6.2.0** (D). Schemes: `redis://`, `rediss://` (TLS, since v6.1.0, R:v6.1.0), `unix`. URL query params seen in code: `connect_timeout`, `io_timeout`, `tls_enabled`, `force_resp2`, `sentinel_*`, `replica_client_enabled`, `addr` (O:internal/redisshard/shard.go). `connect_timeout`, `io_timeout`, `db`, `user`, `password`, `client_name` are code-only keys: not in engines.md. |
| OSS-ENG-04 | Redis Sentinel | GA | <v6 | address `redis+sentinel://[[user]:pass@]host:port?sentinel_master_name=X&addr=...&sentinel_user=&sentinel_password=`; `sentinel_tls`; code also has `sentinel_address[]`, `sentinel_master_name`, `sentinel_client_name` | — | — | D:server/engines.md §Redis Sentinel; O:internal/configtypes/redis.go | `sentinel_master_name` is required. Code also accepts `rediss+sentinel://`. v6.9.2: periodic Sentinel topology refresh (R:v6.9.2). v6.9.0: the Sentinel TLS config is now used for the Sentinel connection (R:v6.9.0). |
| OSS-ENG-05 | Redis app-level sharding | GA | <v6 | `engine.redis.address` as array | — | — | D:server/engines.md §Redis sharding | Jump consistent hash over channels and keys. Do not use DB numbers as shards, because PUB/SUB is shared across DBs. |
| OSS-ENG-06 | Redis Cluster | GA | <v6 (auto-detect v6.1.0) | plain address; seed nodes via `?addr=`; sharding across clusters with an address array; code also has `cluster_address[]` and the `redis+cluster://` scheme | — | hash-tagged keys | D:server/engines.md §Redis Cluster support; R:v6.1.0 | Detected automatically since v6.1.0. Sharded PUB/SUB and replicas are **PRO** (D:pro/scalability.md). v6.9.7: channels starting with `}` are refused with bad request (R:v6.9.7). |
| OSS-ENG-07 | Redis-compatible storages | GA ("reported to work") | — | as Redis | — | — | D:server/engines.md §Redis compatible storages | AWS ElastiCache and Google Memorystore are "reported to work, test failover". KeyDB "should work fine". DragonflyDB "should work fine", try `force_resp2`, cluster emulation untested. Valkey "should work fine… no tests performed by Centrifugal Labs". **Conflict:** R:v6.0.3 says Valkey was added to the Redis engine integration test suite. |
| OSS-ENG-08 | Redis IAM auth (AWS/GCP/Azure) | EXP | v6.10.0 | `aws_iam_auth{enabled,replication_group_id \| serverless_cache_name,region}` + `user`; `gcp_iam_auth{enabled}`; `azure_entra_auth{enabled}` | — | — | D:server/engines.md §Redis IAM authentication; R:v6.10.0 | Only one may be enabled. Cannot be combined with `password`. TLS is required. Applies to engine, broker, presence_manager, map broker and redis_stream consumer. |
| OSS-ENG-09 | Redis presence via hash field TTL | GA (opt-in) | ≤v6.0 (exact min not stated) | `engine.redis.presence_hash_field_ttl` | — | HASH with per-field TTL (Redis ≥7.4) | D:server/engines.md | Saves up to 1.6x memory (D). |
| OSS-ENG-10 | Redis presence user mapping | GA (opt-in) | (not stated) | `engine.redis.presence_user_mapping` | speeds up `presence_stats` | extra per-user ZSET+HASH | D:server/engines.md | Up to 2x presence memory (D). |
| OSS-ENG-11 | Redis history in Lists | DEP | — | `engine.redis.history_use_lists` | — | Redis Lists | D:server/engines.md | No reverse history and no delta compression (D). Code adds: no pagination and no idempotent publishing (O:configtypes/engine.go). See CONFLICT-04. |
| OSS-ENG-12 | Separate broker and presence manager | GA | v6.0.0 | `broker.enabled`, `broker.type` (`redis`, `nats`, `postgres`; code also `memory`), `broker.redis`, `broker.nats`, `broker.postgres`; `presence_manager.enabled`, `presence_manager.type` (`memory` default, `redis`), `presence_manager.redis` | — | — | D:server/engines.md §Separate broker and presence manager; R:v6.0.0; O:internal/app/engine.go | If both are enabled, the engine is not initialized. Per-namespace engines are **PRO**. |
| OSS-ENG-13 | NATS broker | GA | <v6 | `broker.nats.url` (`nats://localhost:4222`), `prefix` (`centrifugo`), `dial_timeout` (`1s`), `write_timeout` (`1s`), `tls`, `allow_wildcards` | — | core NATS PUB/SUB | D:server/engines.md §Nats broker | **At most once only. No history, recovery or cache.** No JetStream. Join/leave does not work in wildcard channels. Tags in publications since v6.2.0 (R:v6.2.0). |
| OSS-ENG-14 | NATS raw mode | GA | <v6 | `broker.nats.raw_mode.{enabled, channel_replacements, prefix}` | — | — | D:server/engines.md §Nats raw mode | No Centrifugo wrapping, so no join/leave. v6.10.0: payloads not matching `publication_data_format` json/json_object are logged and dropped. |
| OSS-ENG-15 | PostgreSQL stream broker (transactional publish) | EXP | v6.8.0 | `broker.type=postgres`; `broker.postgres.{dsn (req), pool_size 16, num_shards 8, cleanup_interval 1m, idempotent_result_ttl 5m, binary_data false, table_prefix "cf", stream_retention 24h, use_notify false, notify_dsn, skip_schema_init, partition_lookahead_days 2, partition_retention_days 7, outbox.poll_interval 100ms, outbox.batch_size 1000}`; code also has `tls` | SQL: `cf_stream_publish`, `_publish_strict`, `_publish_join`, `_publish_leave`, `_remove_history`, `_top_position` | PG ≥16; daily-partitioned stream table plus outbox polling and LISTEN/NOTIFY | D:server/engines.md §PostgreSQL broker; R:v6.8.0; B:2026-05-24-pg-stream-broker-benefits.md | Always tracks offset and epoch, even without history config. Meta TTL is not refreshed on reads. Each node polls (N nodes = N times the reads). Built-in SQL defaults when called directly: history_ttl 1 min, size 100. Supports version suppression (`p_version`, `p_version_epoch`) and `p_idempotency_key`. Read replicas and broker fan-out are **PRO**. |
| OSS-ENG-16 | PostgreSQL controller | EXP | v6.8.0 | `controller.enabled`, `controller.type=postgres`, `controller.postgres.{dsn, pool_size 8, num_shards 1, table_prefix cf, poll_interval 50ms, use_notify, notify_dsn, partition_retention_days 1, partition_lookahead_days 2, partition_cleanup_interval 1m, skip_schema_init}`; code also has `tls` and `batch_size` (1000) | — | PG ≥16; `cf_controller_messages` (partitioned), `cf_controller_shard_lock`, `cf_controller_schema_version`, `cf_controller_publish()` | D:server/engines.md §PostgreSQL controller; O:internal/controllers/; B:2026-05-25-pg-controller-multi-node.md | Carries heartbeats, discovery, surveys, unsubscribe/disconnect propagation. Does not carry publications. `replica` is PRO ("In Centrifugo OSS `replica` is not a known option"). Redis/NATS custom controllers are **PRO**. See CONFLICT-01 for the `use_notify` default. |
| OSS-ENG-17 | Controller concept (implicit) | GA | — | none (implicit) | — | — | L:controller.go; L:node.go `SetBroker` | If the broker implements `Controller`, it doubles as the controller. That is the case for the Redis broker (L:broker_redis.go) and the NATS broker (O:internal/natsbroker/broker.go). The PG stream broker does **not** implement it (no `PublishControl` in O:internal/pgstreambroker). See OQ-ENG-03. |
| OSS-ENG-18 | Map broker backends (cross-ref) | EXP (map subs) | v6.8.0 | `map_broker.type` (`memory`, `redis`, `postgres`) | — | — | O:configtypes/engine.go `MapBroker`; D:server/engines.md §PostgreSQL-only multi-node | Details belong to the map subscriptions scope. Listed here only for the capability matrix. |
| OSS-ENG-19 | Redis replica client (key exists in OSS) | PRO consumers | — | `*.redis.replica_client.enabled`, URL `replica_client_enabled` | — | — | O:configtypes/redis.go; D:pro/scalability.md | OSS parses it, but the users (`subscribe_on_replica`, `presence_read_from_replica`) are PRO. |
| OSS-ENG-20 | `redisnats` broker | unreleased | — | `broker.type=redisnats` + `enable_unreleased_features` | — | — | O:internal/app/engine.go; O:configtypes/engine.go `RedisNats` (`expose:"-"`) | Hidden. Do not implement; record it only. |

### 1.2 History and recovery (OSS-HIST)

| ID | Feature | Status | Min ver | Config keys | API / protocol | Sources | Notes |
|---|---|---|---|---|---|---|---|
| OSS-HIST-01 | Per-channel history stream (offset uint64 + epoch string) | GA | <v6 | ns `history_size` (0), `history_ttl` (0s) | server `history`, client history | D:server/history_and_recovery.md §The history stream; D:server/channels.md §history_size/§history_ttl | **Both** must be > 0. `history_ttl` must be < `history_meta_ttl`. History is "a cache, not your source of truth". |
| OSS-HIST-02 | History meta TTL | GA | <v6 | `channel.history_meta_ttl` (default `720h`), ns `history_meta_ttl` | — | D:server/channels.md §history_meta_ttl; O:configtypes/types.go | v6.10.0: now also applied to token/proxy-authorized subscriptions (R:v6.10.0). |
| OSS-HIST-03 | Stream recovery (mode `stream`) | GA | <v6 | ns `force_recovery`, `allow_recovery`, `force_recovery_mode` (`stream` default) | subscribe `recover`, `epoch`, `offset`; reply `recovered`, `was_recovering`, `publications`, `epoch`, `offset` | D:server/history_and_recovery.md §How recovery works, §The recovery decision; D:server/channels.md §force_recovery | Recovery succeeds only if the epoch matches **and** the whole gap is in history. There is never partial recovery. `recovered:true` can come with zero publications. Enabling recovery also enables positioning. Clients should tolerate duplicates. Code decision rule: L:client.go `isStreamRecovered`. |
| OSS-HIST-04 | Cache recovery (mode `cache`) | GA | v5.4.0 | ns `force_recovery_mode: cache` (+ recovery + history, typically `history_size:1`) | client empty `since` -> `recover:true` | D:server/cache_recovery.md | Delivers only the latest publication. Intermediate ones may be lost. |
| OSS-HIST-05 | Auto cache recover | GA | v6.8.3 | ns `auto_cache_recover` (requires `force_recovery` + `force_recovery_mode: cache`) | connect-proxy `subs[ch].cache_recover` (per subscription) | D:server/cache_recovery.md §Automatic cache recovery; D:server/channels.md; R:v6.8.3 | Works for server-side subscriptions and unidirectional clients. |
| OSS-HIST-06 | Positioning | GA | <v6 | ns `force_positioning`, `allow_positioning` | disconnect `3010 insufficient state` | D:server/history_and_recovery.md §Positioning; D:server/channels.md | Requires history. The subscribe reply exposes top offset and epoch. |
| OSS-HIST-07 | Max publications per recovery: global and per-namespace | GA | global <v6; ns v6.10.3 | `client.recovery_max_publication_limit` (300), ns `recovery_max_publication_limit` (0 = inherit) | — | D:server/channels.md §recovery_max_publication_limit; D:server/configuration.md; R:v6.10.3 | If the client missed more than the limit: `recovered:false` and **no** publications. Applies to server-side subscriptions too. |
| OSS-HIST-08 | Max publications per client history call: global and per-namespace | GA | global <v6; ns v6.10.3 | `client.history_max_publication_limit` (300), ns `history_max_publication_limit` | client history | D:server/channels.md §history_max_publication_limit; R:v6.10.3 | — |
| OSS-HIST-09 | History iteration API (`limit`, `since`, `reverse`) | GA | <v6 | — | server `history` (limit 0 = only top position, -1 = all up to the limit) | D:server/history_and_recovery.md §History iteration API | Reverse is unsupported with Redis Lists (OSS-ENG-11). |
| OSS-HIST-10 | Unrecoverable outcome | GA | — | — | `recovered:false`; error/disconnect **112** `unrecoverable position` exists (L:errors.go) | D:server/history_and_recovery.md; L:client.go (~5389) | For stream subscriptions the documented outcome is `recovered:false`. The code also has a client flag `subscriptionFlagRejectUnrecovered` that turns this into a 112 disconnect. Undocumented in this scope, see OQ-HIST-02. |
| OSS-HIST-11 | Recovery buffering bounded by `client.queue_max_size` per connection | GA | v6.9.7 | `client.queue_max_size` | — | R:v6.9.7 | Behavioral hardening. |
| OSS-HIST-12 | History permission gating | GA | <v6 | ns `allow_history_for_client`, `_subscriber`, `_anonymous` | 108 if history is not configured, 103 if denied | D:server/channel_permissions.md §History permission model | — |

### 1.3 Presence and join/leave (OSS-PRES)

| ID | Feature | Status | Min ver | Config keys | API | Sources | Notes |
|---|---|---|---|---|---|---|---|
| OSS-PRES-01 | Online presence per channel | GA | <v6 | ns `presence` | server `presence`, client `presence` | D:server/presence.md; D:server/channels.md §presence | Entries: `client`, `user`, `conn_info`, `chan_info`. |
| OSS-PRES-02 | Presence stats (num clients, num users) | GA | <v6 | — | server/client `presence_stats` | D:server/presence.md | Redis optimization: OSS-ENG-10. |
| OSS-PRES-03 | Join/leave events | GA | <v6 | ns `join_leave`, `force_push_join_leave` | client `join`/`leave` push | D:server/channels.md §join_leave; D:server/presence.md | **At most once.** Not delivered unless forced or requested by a client that has presence permission. N² risk on mass reconnect. v6.10.0: token/proxy override of `join_leave` honored (R:v6.10.0). |
| OSS-PRES-04 | Presence/channel info sources | GA | <v6 | — | conn token `info`/`b64info`, connect proxy `info`; sub token `info`, subscribe proxy `info` | D:server/presence.md §Connection and channel custom info | `info` and `chan_info` are attached to presence and join/leave. |
| OSS-PRES-05 | Presence TTL (Redis) and update interval | GA | — | `engine.redis.presence_ttl` (60s); `client.presence_update_interval` (27s, **code-only**) | — | D:server/engines.md; O:configtypes/types.go | `presence_update_interval` is not documented. See CONFLICT-15. |
| OSS-PRES-06 | Presence permission gating | GA | <v6 | `allow_presence_for_client`, `_subscriber`, `_anonymous` | 108 if presence is disabled | D:server/channel_permissions.md §Presence permission model | — |
| OSS-PRES-07 | Per-user status API | PRO | — | — | — | D:server/presence.md (tip) | Out of OSS scope. |

### 1.4 Authentication and channel permissions (OSS-AUTH)

| ID | Feature | Status | Min ver | Config keys | Sources | Notes |
|---|---|---|---|---|---|---|
| OSS-AUTH-01 | Connection JWT | GA | <v6 | `client.token.*` | D:server/authentication.md | Claims in §2.1. |
| OSS-AUTH-02 | Algorithms HS256/384/512, RS256/384/512, ES256/384/512, EdDSA (Ed25519) | GA | EdDSA PEM v6.10.2 | `client.token.hmac_secret_key`, `rsa_public_key`, `ecdsa_public_key`, `ed25519_public_key` (PEM) | D:server/authentication.md; R:v6.10.2; O:internal/jwtverify/token_verifier_jwt.go `newAlgorithms` | Allowlist is built from the configured keys. An unconfigured algorithm is rejected. Before v6.10.2, EdDSA was available only via JWKS. |
| OSS-AUTH-03 | Audience / issuer checks | GA | <v6 | `client.token.audience`, `issuer` | D:server/authentication.md §aud/§iss | Off by default. Strongly recommended for external IdPs. Also applies to subscription tokens unless a separate config is used. |
| OSS-AUTH-04 | JWKS | GA | <v6 (EC v5.1.0, optional `alg` v6.2.0, OKP) | `client.token.jwks_public_endpoint` | D:server/authentication.md §JSON Web Key support; R:v6.2.0, v6.10.0, v6.10.2 | kty RSA/EC/OKP(Ed25519). Cache 5 min (1 h before v6.10.2). Refetch on unknown `kid` at most once per 5 s. GET with 1 s timeout and 1 retry, not configurable. Unknown kid -> treated as expired (v6.10.2+). Endpoint down -> temporary error. Once set, **all** tokens are verified via JWKS. |
| OSS-AUTH-05 | Dynamic JWKS endpoint (`issuer_regex`/`audience_regex` + `{{var}}` template) | GA (breaking change v6.7.0) | v6.7.0 safety rules | `client.token.issuer_regex`, `audience_regex`, `insecure_skip_jwks_endpoint_safety_check` | D:server/authentication.md §Dynamic JWKs endpoint; R:v6.7.0 | Named groups used in the template must be an explicit alternation of fixed strings, otherwise startup fails. Cannot be combined with `issuer`/`audience` (startup error). Claims are parsed **before** signature verification (O:jwtverify `extractTokenVars`). |
| OSS-AUTH-06 | HMAC key rotation | GA | v6.6.1 | `client.token.hmac_previous_secret_key`, `hmac_previous_secret_key_valid_until` (unix) | D:server/authentication.md §HMAC key rotation; R:v6.6.1 | Also available for subscription tokens. |
| OSS-AUTH-07 | Custom user ID claim | GA | <v6 | `client.token.user_id_claim` (regex `^[a-zA-Z_]+$`) | D:server/authentication.md §Custom token user id claim | — |
| OSS-AUTH-08 | Connection expiration and refresh | GA | <v6 | `exp`/`expire_at` | D:server/authentication.md §Connection expiration | 25 s grace (D). Connect reply carries `ttl`. Refresh via SDK or refresh proxy. |
| OSS-AUTH-09 | Subscription JWT | GA | <v6 | same keys; `channel.private_prefix` (`$`) | D:server/channel_token_auth.md | Claims in §2.2. |
| OSS-AUTH-10 | Separate subscription token config | GA | <v6 | `client.subscription_token.enabled` + all token keys | D:server/channel_token_auth.md §Separate subscription token config | — |
| OSS-AUTH-11 | Anonymous / insecure modes | GA | <v6 | `client.allow_anonymous_connect_without_token`, `disallow_anonymous_connection_tokens`, `insecure`, `insecure_skip_token_signature_verify` | O:configtypes/types.go; D:server/configuration.md | Insecure modes are for development only. |
| OSS-AUTH-12 | Subscribe permission order | GA | — | ns `allow_subscribe_for_client`, `allow_subscribe_for_anonymous`, `allow_user_limited_channels`, `subscribe_proxy_enabled`, `subscribe_stream_proxy_enabled` | D:server/channel_permissions.md §Subscribe permission model | Order: known ns (102) -> private-prefix gate -> token (alone decides; 109 expired, 103 unverifiable, 3500 wrong channel/user/client) -> user-limited `#` -> subscribe proxy -> stream proxy -> [PRO caps] -> allow_subscribe_* -> client.insecure -> [PRO CEL]. |
| OSS-AUTH-13 | Publish / history / presence permission order | GA | — | `allow_publish_for_client`, `_subscriber`, `_anonymous`, `publish_proxy_enabled`; history/presence analogs | D:server/channel_permissions.md | The publish proxy takes precedence over everything. In a stream-proxy namespace, publishing is allowed only into one's own bidi stream (108 otherwise). |
| OSS-AUTH-14 | Positioning / recovery / join-leave permission | GA | v6.10.0 (proxy-authorized) | `allow_positioning`, `allow_recovery`, history permission | D:server/channel_permissions.md §Positioning/Recovery/Join-Leave permission model | Stream-proxy subscriptions never get positioning or recovery. |
| OSS-AUTH-15 | Capabilities (`caps`, `allow`) | PRO | — | — | D:server/channel_permissions.md §Capabilities | Out of OSS scope, but the wire fields exist in proxy.proto. |
| OSS-AUTH-16 | `gensubtoken` CLI | GA | <v6 | — | D:server/channel_token_auth.md | HS256, default TTL 168h (example output). |

### 1.5 Proxy (OSS-PROXY)

| ID | Feature | Status | Min ver | Config keys | Sources | Notes |
|---|---|---|---|---|---|---|
| OSS-PROXY-01 | Connect proxy | GA | <v6 | `client.proxy.connect.{enabled,…}` | D:server/proxy.md §Connect proxy | Called only when the client has **no JWT**. Bidirectional and unidirectional. |
| OSS-PROXY-02 | Refresh proxy | GA | <v6 | `client.proxy.refresh.{enabled,…}` | D:server/proxy.md §Refresh proxy | On error, Centrifugo retries later. Custom error/disconnect are ignored. |
| OSS-PROXY-03 | Subscribe proxy | GA | <v6 | `channel.proxy.subscribe` + ns `subscribe_proxy_enabled`, `subscribe_proxy_name` | D:server/proxy.md §Subscribe proxy | Not called for token subscriptions or user-limited channels. Bidirectional only. |
| OSS-PROXY-04 | Publish proxy | GA | <v6 | `channel.proxy.publish` + ns `publish_proxy_enabled`, `publish_proxy_name` | D:server/proxy.md §Publish proxy | Can replace data and set skip_history. Since v6.8.0 also tags, idempotency_key, delta, version, version_epoch. |
| OSS-PROXY-05 | Sub refresh proxy | GA | <v6 | `channel.proxy.sub_refresh` + ns `sub_refresh_proxy_enabled`, `sub_refresh_proxy_name` | D:server/proxy.md §Sub refresh proxy | Only for subscriptions that have an expiration. Request delay up to ~1 min. |
| OSS-PROXY-06 | RPC proxy | GA | <v6 | `rpc.proxy`, `rpc.without_namespace.proxy_enabled`, `rpc.namespaces[].{name,proxy_enabled,proxy_name}` | D:server/proxy.md §Client RPC proxy | RPC namespace boundary `:`. Name regex `^[-a-zA-Z0-9_.]{2,}$`. |
| OSS-PROXY-07 | Subscribe stream proxy (unidirectional and bidirectional GRPC) | EXP | v5.1.0 | `channel.proxy.subscribe_stream` (grpc only), ns `subscribe_stream_proxy_enabled`, `_name`, `subscribe_stream_proxy_bidirectional` | D:server/proxy_streams.md; CHANGELOG v5.1.0 | The first message must contain `SubscribeResponse`. `timeout` = wait for that first message. Client-side subscriptions only. Cannot be combined with subscribe/publish/sub_refresh proxy in one ns. On backend link loss, unsubscribe with `insufficient state`. No history semantics. |
| OSS-PROXY-08 | HTTP and GRPC proxy protocols | GA | <v6 | `endpoint` `http(s)://` or `grpc://` | D:server/proxy.md §Supported proxy protocols | HTTP is always POST with JSON and `Content-Type: application/json`. GRPC is unary, schema `proxy.proto` (also on BSR, versioned labels since v6.10.0). |
| OSS-PROXY-09 | Proxy config object | GA | v6.0.0 shape | `endpoint`, `timeout` (1s), `http_headers`, `emulated_headers`, `grpc_metadata`, `include_connection_meta`, `binary_encoding`, `http.{static_headers,status_to_code_transforms}`, `grpc.{tls,credentials_key,credentials_value,compression,static_metadata}` | D:server/proxy.md §Proxy configuration object; O:configtypes/types.go `ProxyCommon` | `static_metadata` since v6.5.2 (R:v6.5.2). |
| OSS-PROXY-10 | Header emulation, split into `emulated_headers` | GA (security change) | emulation v6.0.0; split v6.9.0 | `emulated_headers` | D:server/proxy.md §HTTP headers emulation; R:v6.9.0 | Precedence: transport header > emulated > static. Emulated values are untrusted input. |
| OSS-PROXY-11 | Header/metadata translation rules | GA | — | — | D:server/proxy.md §Header proxy rules | HTTP<->GRPC mapping table. |
| OSS-PROXY-12 | Named proxies and per-namespace proxies | GA | <v6 | top-level `proxies[]` with `name` (regex above) | D:server/proxy.md §Per-namespace custom proxies | For subscribe, publish, sub_refresh, subscribe_stream and rpc. v6.10.3: a SIGHUP reload that references a proxy not created at start is rejected (R:v6.10.3). |
| OSS-PROXY-13 | Custom error / disconnect | GA | <v6 | — | D:server/proxy.md §Return custom error/§Return custom disconnect | Error codes 400–1999. Disconnect 4000–4499 = reconnect, 4500–4999 = terminal. Reason ≤32 bytes. An empty response **allows** the request. Honored for connect, subscribe, publish and rpc (and map publish/remove). |
| OSS-PROXY-14 | Unexpected error handling and code transforms | GA | <v6 | `http.status_to_code_transforms[{status_code,to_error{code,message,temporary},to_disconnect{code,reason}}]`, `client.connect_code_to_unidirectional_disconnect{enabled,transforms}`, `uni_sse`/`uni_http_stream.connect_code_to_http_response` | D:server/proxy.md §Unexpected error handling | Defaults: bidirectional 100 (temporary), unidirectional disconnect 3004. |
| OSS-PROXY-15 | Binary encoding mode | GA | <v6 | `binary_encoding` | D:server/proxy.md §Binary encoding mode | `encoding:"binary"` in requests, `b64*` fields. |
| OSS-PROXY-16 | Include connection meta | GA | <v6 | `include_connection_meta` | D:server/proxy.md §Include connection meta | No-op for connect. |
| OSS-PROXY-17 | Map publish/remove proxy, shared poll refresh proxy | EXP (cross-ref) | v6.8.0 | see map_subscriptions/shared_poll | D:server/proxy.md §Map publish/remove proxy; O:internal/proxyproto/proxy.proto | Out of this scope. Listed for completeness. |
| OSS-PROXY-18 | Cache empty / state proxies | PRO | — | — | D:server/proxy.md | RPCs exist in proxy.proto (`NotifyCacheEmpty`, `NotifyChannelState`). |
| OSS-PROXY-19 | No unsubscribe/disconnect hooks | GA (design) | — | — | D:server/proxy.md (tip) | Do not implement disconnect webhooks as OSS-compatible behavior. |

### 1.6 Async consumers (OSS-CONS)

| ID | Consumer | Status | Min ver | Sources | Notes |
|---|---|---|---|---|---|
| OSS-CONS-01 | Common framework: `consumers[]` `{enabled,name,type,<type>{…}}` | GA | <v6 | D:server/consumers.md §consumers[]; O:configtypes/types.go `Consumer` | Name regex `^[a-zA-Z0-9_]{2,}$`. Env `CENTRIFUGO_CONSUMERS` (JSON) and `CENTRIFUGO_CONSUMERS_<NAME>_<OPT>`. |
| OSS-CONS-02 | PostgreSQL outbox | GA | <v6 (pre-v6; exact min not checked) | D:server/consumers.md §PostgreSQL outbox consumer | §4. |
| OSS-CONS-03 | Kafka | GA | <v6; publication data mode v6.0.0 | D:server/consumers.md §Kafka consumer; R:v6.0.0, v6.2.3, v6.7.1, v6.8.2 | §4. |
| OSS-CONS-04 | NATS JetStream | GA | v6.2.0 | D:server/consumers.md §Nats Jetstream; R:v6.2.0, v6.5.2 | §4. |
| OSS-CONS-05 | Redis Streams | GA | v6.2.0 | D:server/consumers.md §Redis Stream | §4. |
| OSS-CONS-06 | Google Cloud Pub/Sub | GA | v6.2.0 | D:server/consumers.md §Google Cloud PUB/SUB | §4. |
| OSS-CONS-07 | AWS SQS (incl. SNS envelope) | GA | v6.2.0 | D:server/consumers.md §AWS SQS | §4. |
| OSS-CONS-08 | Azure Service Bus | GA | v6.2.0 | D:server/consumers.md §Azure Service Bus | §4. |
| OSS-CONS-09 | Command mode vs publication data mode; method header/attribute | GA | method header v6.2.0 | D:server/consumers.md §How consumers work, §Publication data mode; R:v6.2.0 | Only state-modifying commands. Reads are ignored. `batch` is unsupported. JSON payloads only. |
| OSS-CONS-10 | Error and retry semantics | GA | v6.10.0 rule | D:server/consumers.md §How consumers work; R:v6.10.0 | Retries only errors 100 and 111. Other errors are logged and the message is marked processed. A broadcast with any temporary per-channel failure is retried whole. |
| OSS-CONS-11 | Graceful drain on shutdown | GA | v6.10.3 | D:server/consumers.md; R:v6.9.2, v6.10.3 | Stop taking new messages, finish and ack in-flight ones within `shutdown.timeout`. |
| OSS-CONS-12 | OTel tracing of consuming | (code only) | — | O:configtypes/types.go `OpenTelemetry.consuming` | Key exists, no docs found. OQ-OBS-01. |

### 1.7 Observability (OSS-OBS)

| ID | Feature | Status | Min ver | Config keys | Sources | Notes |
|---|---|---|---|---|---|---|
| OSS-OBS-01 | Prometheus `/metrics` | GA | <v6 | `prometheus.{enabled, handler_prefix "/metrics", instrument_http_handlers, recovered_publications_histogram, native_histograms}` | D:server/monitoring.md; D:server/observability.md; O:configtypes/types.go | Metric list in §1.7.1. |
| OSS-OBS-02 | Native histograms; Summaries deprecated (removal in v7) | GA (opt-in; `client_golang` marks native histograms experimental) | v6.8.1 | `prometheus.native_histograms` | D:server/observability.md §Native histograms; R:v6.8.1 | When on, Summaries are dropped and `_histogram` companions switch to the native schema. |
| OSS-OBS-03 | Graphite export | GA | <v6 | `graphite.{enabled, host localhost, port 2003, prefix centrifugo, interval 10s, tags}` | D:server/monitoring.md; O:configtypes/types.go | `prefix` and `tags` are code-only keys. |
| OSS-OBS-04 | OpenTelemetry tracing of server API (HTTP and GRPC) | GA | <v6; GRPC exporter v5.0.3 | `opentelemetry.{enabled, api}`; `OTEL_*` env | D:server/observability.md §OpenTelemetry | Default exporter `http/protobuf`. `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` is supported. `service.name=centrifugo`. `service.instance.id` = node ID (v6.8.3). |
| OSS-OBS-05 | OTel Google Cloud ADC auth | GA | v6.8.2 | `opentelemetry.google_cloud_adc_auth` | D:server/observability.md §Export to Google Cloud | — |
| OSS-OBS-06 | Health endpoint | GA | <v6 | `health.{enabled, handler_prefix "/health"}` | D:server/configuration.md §health; O:internal/health/health.go | Code returns HTTP 200 `{}` unconditionally, with no dependency checks. Internal port. |
| OSS-OBS-07 | Logging | GA | v6 naming | `log.level` (`none`,`trace`,`debug`,`info` default,`warn`,`error`), `log.file` | D:server/configuration.md §log; D:server/observability.md §Logs | See CONFLICT-06 (`log_level` in observability.md). Secret redaction: R:v6.0.2, v6.9.0. |
| OSS-OBS-08 | Grafana dashboard 13039 | GA | updated v6.9.5 | — | D:server/observability.md §Grafana dashboard | Uses `_histogram` metrics only. |
| OSS-OBS-09 | Init endpoint (`/connection/init` per R) | GA | v6.5.0 | `init.enabled` | D:server/configuration.md §init; R:v6.5.0 | GET returns `{}`. Usable as an LB ping. External port. |
| OSS-OBS-10 | Debug pprof, Swagger UI | GA | <v6 | `debug.{enabled,handler_prefix /debug/pprof}`, `swagger.{enabled,handler_prefix /swagger}` | O:configtypes/types.go; D:server/configuration.md | Internal endpoints. |

#### 1.7.1 Documented Prometheus metrics (namespace `centrifugo_`)
Source: D:server/observability.md §Exposed metrics. "(PRO)" marks a label that only PRO adds. "Dep" means the Summary is deprecated and has a `_histogram` companion added in v6.8.1.

| Metric | Type | Labels (OSS) |
|---|---|---|
| node_messages_sent_count / node_messages_received_count | Counter | type |
| node_action_count | Counter | action |
| node_num_clients, node_num_users, node_num_subscriptions, node_num_nodes, node_num_channels | Gauge | — |
| node_build | Gauge | version |
| node_survey_duration_seconds (Dep) / `_histogram` | Summary / Histogram | op |
| client_num_reply_errors | Counter | method, code |
| client_num_server_unsubscribes | Counter | code |
| client_num_server_disconnects | Counter | code |
| client_command_duration_seconds (Dep) / `_histogram` | Summary / Histogram | method |
| client_recover | Counter | recovered, has_recovered_publications (v6.2.4) |
| client_recovered_publications | Histogram (opt-in, v6.2.4) | (see doc) |
| node_client_connection_limit | Counter | — |
| client_connections_accepted / client_connections_inflight | Counter / Gauge | transport, client_name, client_version |
| client_subscriptions_inflight | Gauge | client_name |
| client_ping_pong_duration_seconds | Histogram | transport |
| transport_messages_sent / _sent_size / _received / _received_size | Counter | transport, frame_type, channel_namespace |
| transport_outgoing_close_count (v6.8.4) | Counter | transport, code |
| transport_frame_size (v6.9.4) | Histogram | transport |
| proxy_duration_seconds (Dep) / `_histogram`, proxy_errors, proxy_inflight_requests | Summary/Histogram/Counter/Gauge | protocol, type, name |
| api_command_duration_seconds (Dep) / `_histogram` | — | protocol, method |
| api_command_errors_total | Counter | protocol, method, error |
| api_rpc_duration_seconds (Dep) / `_histogram` | — | protocol, method |
| node_pub_sub_lag_seconds | Histogram | — |
| node_broadcast_duration_seconds | Histogram | type |
| node_tags_filter_dropped_publications | Counter | — |
| broker_redis_pub_sub_errors / _dropped_messages / _buffered_messages | Counter/Counter/Gauge | broker_name, error / channel_type / pub_sub_processor |
| broker_publish_suppressed_count | Counter | reason |
| broker_postgres_cleanup_removed_total | Counter | broker_name, pass |
| broker_postgres_outbox_cursor_lag_seconds | Gauge | broker_name, shard |
| broker_postgres_partitions | Gauge | broker_name |
| map_broker_* (redis pub/sub, suppressed, cleanup, postgres lag/partitions) | various | cross-ref, map scope |
| broker_redis_node_grouped_* | — | PRO only |
| consumers_messages_processed_total / consumers_errors_total | Counter | consumer_name |
| node_incoming_http_requests_total | Counter | path (route pattern since v6.9.0), method, status; needs `instrument_http_handlers` |
| shared_poll_* | — | cross-ref, shared poll scope |

Present in code, missing from docs: `centrifugo_client_subscriptions_accepted` (L:metrics.go; fixed to actually increment in v6.9.5, R:v6.9.5). See CONFLICT-16.

### 1.8 Ops / deployment (OSS-OPS)

| ID | Feature | Status | Min ver | Sources | Notes |
|---|---|---|---|---|---|
| OSS-OPS-01 | Binary releases: linux amd64/arm64/armv6, darwin amd64/arm64, windows amd64, freebsd amd64; `install.sh` | GA | — | D:getting-started/installation.md | — |
| OSS-OPS-02 | Docker image `centrifugo/centrifugo:vX` (Alpine base) | GA | — | D:getting-started/installation.md; R:v6.9.2 (Alpine 3.24) | Since v6.10.0, pre-releases are published only under their exact tag. `latest`/`v6`/`v6.10` point to stable releases. Docs advise a high `nofile` ulimit. |
| OSS-OPS-03 | Helm chart (`centrifugal/helm-charts`; GHCR since v6.3.0; v13 in v6.6.1; Gateway API in 13.3.0) | GA | — | D:getting-started/installation.md; R:v6.3.0, v6.6.1, v6.8.2 | Separate repo, not in this clone. |
| OSS-OPS-04 | RPM/DEB via packagecloud (Debian 10–12, Ubuntu 20.04/22.04/24.04, CentOS 7) | GA | Noble v6.1.0 | D:getting-started/installation.md; R:v6.1.0 | O:misc/packaging/centrifugo.service exists. |
| OSS-OPS-05 | Homebrew tap | GA | — | D:getting-started/installation.md | — |
| OSS-OPS-06 | Sigstore-signed checksums, SPDX SBOMs, signed images | GA | v6.10.0 | D:getting-started/installation.md §Verifying release artifacts; R:v6.10.0 | — |
| OSS-OPS-07 | `genconfig`, `version`, `gensubtoken` CLI; `--watch-config-file` (dev) | GA | watch v6.10.3 | D:getting-started/installation.md; R:v6.10.3 | CLI scope is shared with another agent. |
| OSS-OPS-08 | TLS: file/PEM, ACME autocert, unified TLSConfig | GA | unified v6.0.0 | D:server/tls.md | TLSConfig `{enabled, cert_pem, key_pem, server_ca_pem, client_ca_pem, insecure_skip_verify, server_name}`. PEM fields accept raw, base64 or a path (detected in that order). Autocert `{enabled, host_whitelist, cache_dir, email, http, http_addr :80}`; code also has `server_name`. `grpc_api.tls`, `uni_grpc.tls`. |
| OSS-OPS-09 | Load balancing guidance | GA (docs) | — | D:server/load_balancing.md | WS upgrade, no buffering for SSE/HTTP-stream, forward `/emulation`, idle timeout > ping interval (25 s). **No sticky sessions needed**: an emulation POST can hit any node and is forwarded via the broker. Configs for nginx/HAProxy/Envoy/Caddy/Traefik. |
| OSS-OPS-10 | Infra tuning | GA (docs) | — | D:server/infra_tuning.md | nofile, ephemeral ports, TIME_WAIT, proxy max connections, conntrack, DDoS layers. |
| OSS-OPS-11 | Internal port separation | GA | — | D:server/configuration.md §Custom internal port | `http_server.internal_port`. Internal endpoints: /api, admin, /metrics, /health, /debug/pprof, /swagger. |
| OSS-OPS-12 | Server SDKs (gocent, pycent, jscent, phpcent, rubycent, rustcent, csharpcent) | GA (ecosystem) | doc page v6.10.3 | D:server/server_sdks.md | Shared spec (gocent spec.md). Retry with the same `idempotency_key`. 10 s default timeout. Not a server feature, but relevant for compatibility tests. |
| OSS-OPS-13 | Ecosystem (Centrifuge lib, framework integrations) | info | — | D:getting-started/ecosystem.md | Community integrations: Laravel, Symfony, Django, RoadRunner, Spiral. |
| OSS-OPS-14 | Flow diagrams | info (unlisted page) | — | D:flow_diagrams.md | swimlanes.io sources for JWT/connect-proxy/publish flows. No normative content. |

---

## 2. JWT claim tables

### 2.1 Connection token
Sources: D:server/authentication.md §Connection JWT Claims; wire names confirmed in O:internal/jwtverify/token_verifier_jwt.go `ConnectTokenClaims` (+ embedded RFC 7519 registered claims).

| Claim | Type | Req | Semantics | Source / notes |
|---|---|---|---|---|
| `sub` | string | yes (may be `""`) | User ID. Empty = anonymous. | D. Overridable by `client.token.user_id_claim`. |
| `exp` | int (unix s) | no | Token expiry and, by default, connection expiry. Absent = never expires. | D |
| `iat` | int | no | Issued-at. Useful with PRO revocation. | D |
| `jti` | string | no | Token ID. Useful with PRO revocation. | D |
| `aud` | string/array | no | Checked only if `audience`/`audience_regex` is set. | D |
| `iss` | string | no | Checked only if `issuer`/`issuer_regex` is set. | D |
| `nbf` | int | no | **Not documented.** Code checks `IsValidNotBefore` and returns *token expired* when it fails. | O:jwtverify (~610). CONFLICT-10 |
| `info` | JSON | no | Connection info, shown in presence, join/leave and client publications. | D |
| `b64info` | string (base64) | no | Binary info. Invalid base64 = invalid token, client disconnected (v6.10.0). | D; R:v6.10.0 |
| `channels` | []string | no | Server-side subscriptions. **Not permissions.** | D |
| `subs` | map[channel]SubscribeOptions | no | Server-side subscriptions with options. | D |
| `subs[ch].info` / `b64info` | JSON / string | no | Channel info. | D |
| `subs[ch].data` / `b64data` | JSON / string | no | Data returned in the subscription context of the connect reply. | D |
| `subs[ch].override.{presence,join_leave,force_recovery,force_positioning,force_push_join_leave}` | `{"value":bool}` | no | Per-connection namespace option override. | D |
| `meta` | JSON object | no | Hidden from other clients. Sent to proxies when `include_connection_meta`. Visible to the token owner because a JWT is not encrypted. | D |
| `expire_at` | int | no | Connection expiry, decoupled from `exp`. `0` = no connection expiry (`exp` is still checked). | D |
| `channel` | string | **forbidden** | A connection token containing `channel` is rejected as invalid. | O:jwtverify (~604), undocumented |
| `caps` | array | — | PRO only. | D:server/channel_permissions.md |

### 2.2 Subscription token
Sources: D:server/channel_token_auth.md §Subscription JWT claims; O:jwtverify `SubscribeTokenClaims`.

| Claim | Type | Req | Semantics | Notes |
|---|---|---|---|---|
| `sub` | string | no | Must match the connection's user. Missing = anonymous. | D |
| `channel` | string | **yes** | Channel being subscribed. Missing = invalid token. | D; O |
| `info` / `b64info` | JSON / string | no | Channel info (presence, join/leave, publications). | D |
| `data` / `b64data` | JSON / string | no | Data returned in the subscribe reply. | D |
| `exp` | int | no | Token expiry and subscription expiry. When it expires, the client must refresh. | D says that on subscription expiry the "client connection will be closed". |
| `expire_at` | int | no | Subscription expiry decoupled from `exp`. `0` = never (one-time token). | D |
| `aud` / `iss` | — | no | Same rules as the connection token, unless a separate config is used. | D |
| `iat` / `jti` | — | no | For PRO revocation. | D |
| `nbf` | int | no | Not documented. Checked in code; failure = expired. | O; CONFLICT-10 |
| `override.{presence,join_leave,force_push_join_leave,force_recovery,force_positioning}` | `{"value":bool}` | no | Per-subscriber override. | D |
| `client` | string | no | **Not documented in channel_token_auth.md.** Binds the token to a client ID. On mismatch, a subscribe or sub refresh is rejected. channel_permissions.md mentions "issued for another … client -> 3500 invalid token". | O:internal/client/handler.go (~755, ~879); D:server/channel_permissions.md |
| `allow` | []string | — | PRO only. | D |

Verification-level rules (D and O): the allowlist is built from configured keys. JWKS replaces static keys once it is set. Audience/issuer checks run **after** the signature check (O:`validateClaims`). Dynamic JWKS vars are extracted **before** it. A token's "expired" outcome maps to client error 109 (D:server/channel_permissions.md).

---

## 3. Proxy types and fields
Sources: D:server/proxy.md (tables). Wire truth: O:internal/proxyproto/proxy.proto (service `centrifugal.centrifugo.proxy.CentrifugoProxy`). In HTTP mode, `bytes` fields carry raw JSON. With `binary_encoding`, base64 goes in the `b64*` fields.

Common request header fields (all except cache-empty, state and shared-poll): `client`, `transport`, `protocol` (`json`/`protobuf`), `encoding` (`json`/`binary`).

| Proxy (GRPC rpc) | Request fields (beyond common) | Response | Result fields | error/disconnect honored |
|---|---|---|---|---|
| connect (`Connect`) | `name`, `version`, `data`/`b64data`, `channels[]` | `result`/`error`/`disconnect` | `user` (req), `expire_at`, `info`/`b64info`, `data`/`b64data`, `channels[]`, `subs{ch: SubscribeOptions}`, `meta`, `caps` (PRO), `labels` (PRO), **`profile`** (proto only, undocumented) | yes |
| refresh (`Refresh`) | `user`, `meta`; proto also has `labels` | `result` (docs); proto also has `error`/`disconnect` | `expired`, `expire_at`, `info`/`b64info`, `meta`, `caps` (PRO) | **no** (D). CONFLICT-08 |
| subscribe (`Subscribe`) | `user`, `channel`, `token`, `meta`, `data`/`b64data`, `labels` (PRO) | result/error/disconnect | `info`/`b64info`, `data`/`b64data`, `override{presence,join_leave,force_recovery,force_positioning,force_push_join_leave}`, `expire_at` (v5.0.4), `allow` (PRO), `server_tags_filter` (PRO) | yes |
| publish (`Publish`) | `user`, `channel`, `data`/`b64data`, `meta`, `labels` (PRO) | result/error/disconnect | `data`/`b64data` (replace), `skip_history` (not in delta channels), since v6.8.0: `tags`, `idempotency_key`, `delta`, `version` (uint64), `version_epoch` | yes |
| sub_refresh (`SubRefresh`) | `user`, `channel`, `meta`, `labels` (PRO) | result (docs); proto also has error/disconnect | `expired`, `expire_at`, `info`/`b64info`, `server_tags_filter` (PRO, v6.10.0) | **no** (D) |
| rpc (`RPC`) | `user`, `method` (optional), `data`/`b64data`, `meta`, `labels` (PRO) | result/error/disconnect | `data`/`b64data` | yes |
| subscribe stream unidirectional (`SubscribeUnidirectional`) | `SubscribeRequest` | stream of `StreamSubscribeResponse{subscribe_response, publication{data, tags}}` | first message must carry `subscribe_response` | via the first message |
| subscribe stream bidirectional (`SubscribeBidirectional`) | stream `StreamSubscribeRequest{subscribe_request (first), publication}` | same as above | — | via the first message |
| map publish / map remove | see map scope | — | — | yes |
| shared poll refresh (`SharedPollRefresh`) | `channel`, `items[{key,version}]` | result/error | `items[{key,data,version,removed,prev_data}]`, `epoch` | cross-ref |
| cache empty / channel state | PRO | — | — | — |

Shared types: `SubscribeOptions` (connect `subs`) = `info`, `b64info`, `data`, `b64data`, `override`, `cache_recover` (v6.8.3). The proto also has `expire_at` and `server_tags_filter`, which the docs table omits. `Error{code uint32, message, temporary bool}`: `temporary` is documented only inside status transforms. `Disconnect{code, reason}`.

---

## 4. Consumers and semantics
Common rules (D:server/consumers.md): JSON only. **Command mode** (default) uses `{"method","payload"}`. When the method comes from a header, attribute, property, column or field, the payload must be the raw request JSON. In whole-command form, `payload` may be a JSON string (Debezium). **Publication data mode** means the payload is the publication data, and channels, idempotency key, delta, version, version_epoch and tags come from headers or attributes. Only mutating commands are accepted. `batch` is unsupported. Retries cover 100/111 only. Code backoff: exponential with jitter, 10 ms to 5 s cap (O:internal/consuming/backoff.go, implementation detail, not documented). Delivery is effectively **at-least-once**: docs say "reliable at least once processing" in most cases and point to idempotent publishing for duplicates.

| Type | Config keys (default) | Ordering | Ack / offset (doc + code observation) |
|---|---|---|---|
| `postgresql` | `dsn` (req), `outbox_table_name` (req), `num_partitions` 1, `partition_select_limit` 100, `partition_poll_interval` 300ms, `partition_notification_channel`, `partition_notification_dsn`, `tls`, `use_try_lock` false | Strict order **per partition** | Table schema: `id BIGSERIAL, method text, payload JSONB, partition int default 0, created_at timestamptz`. Optional NOTIFY trigger. Code: advisory xact lock per partition, rows **deleted** after processing (O:consuming/postgresql.go). |
| `kafka` | `brokers` (req), `topics` (req), `consumer_group` (req), `max_poll_records` 100, `fetch_max_bytes` 52428800, `fetch_max_wait` 500ms (v6.2.3), `fetch_read_uncommitted` false (ReadCommitted default, v6.2.3), `sasl_mechanism` (`plain`, `scram-sha-256`, `scram-sha-512`, `aws-msk-iam`), `sasl_user`, `sasl_password`, `assume_role_arn` (v6.7.1), `tls`, `publication_data_mode{enabled, channels_header, idempotency_key_header, delta_header}`. **Code-only:** `partition_queue_max_size` 1000, `dial_timeout` 3s (R:v6.8.2), `instance_id`, `method_header` `centrifugo-method`, pub-data `version_header`, `version_epoch_header`, `tags_header_prefix` | Order **within a Kafka partition** | Code: marked-offset auto-commit, per-partition bounded queues with pause/resume backpressure. Redpanda v24.3.6 passed tests. |
| `nats_jetstream` | `url` nats://127.0.0.1:4222, `credentials_file`, `username`, `password`, `token`, `stream_name`, `subjects`, `durable_consumer_name`, `deliver_policy` new (`new`, `all`), `max_ack_pending` 100, `method_header` centrifugo-method, `use_existing_consumer` (v6.5.2), `tls`, `publication_data_mode{enabled, channels_header, idempotency_key_header, delta_header, version_header, version_epoch_header, tags_header_prefix}` (defaults `centrifugo-*`) | Unordered with several instances. Ordered with a single instance. | Does not create streams. Code: Ack on success, NakWithDelay with capped exponential backoff (v6.9.0), AckWait hard-coded 30s (undocumented). |
| `redis_stream` | Redis connection keys (address, timeouts, db, user, password, client_name, force_resp2, cluster/sentinel, tls, IAM since v6.10.0), `streams`, `consumer_group`, `visibility_timeout` 30s, `num_workers` 1, `payload_value` payload, `method_value` method, `publication_data_mode{… _value}` | Unordered with several instances. Ordered with one instance and `num_workers=1`. | Code: XREADGROUP / XACK / XAUTOCLAIM after the visibility timeout. |
| `google_pub_sub` | `project_id`, `subscriptions`, `max_outstanding_messages` 100, `max_outstanding_bytes` 1000000, `auth_mechanism` (`default`, `service_account`), `credentials_file`, `method_attribute`, `publication_data_mode{… _attribute}` | Ordered with an ordering key and an ordering-enabled subscription | Code: Ack / Nack. |
| `aws_sqs` | `queues` (URLs), `sns_envelope`, `region`, `max_number_of_messages` 10, `wait_time_time` 20s (key spelled exactly this way), `visibility_timeout` 30s, `max_concurrency` 1, `credentials_profile`, `assume_role_arn`, `method_attribute`, `localstack_endpoint`, `publication_data_mode{…}` | FIFO queue with group IDs | Code: DeleteMessageBatch for processed messages. |
| `azure_service_bus` | `connection_string` or (`use_azure_identity`, `fully_qualified_namespace`, `tenant_id`, `client_id`, `client_secret`), `queues`, `use_sessions`, `max_concurrent_calls` 1, `max_receive_messages` 1, `method_property`, `publication_data_mode{… _property}` | Ordered with sessions | Code: Complete on success, Abandon on failure (v6.9.0). |

---

## 5. Engine capability matrix (OSS v6.10.3)

| Backend | Broker PUB/SUB (multi-node) | History / recovery / cache | Presence manager | Map broker | Controller (cluster coordination) | Status |
|---|---|---|---|---|---|---|
| memory | single node only | yes | yes | yes (`map_broker.type=memory`) | not applicable (single node; the memory broker does not implement `PublishControl`) | GA |
| redis (standalone/Sentinel/Cluster/sharded) | yes | yes (Streams; Lists deprecated) | yes | yes | yes, implicitly via the broker (L:broker_redis.go `PublishControl`). An explicit Redis controller is PRO. | GA |
| nats | yes (at most once) | **no** | **no** (pair with memory or redis presence) | no | yes, implicitly via the broker (O:natsbroker `PublishControl`). An explicit NATS controller is PRO. | GA |
| postgres | yes (outbox polling; each node polls) | yes (always tracks offset and epoch) | **no** (`presence_manager.type` is memory or redis only) | yes (cross-ref) | yes: `controller.type=postgres` (separate component) | EXP, v6.8.0 |
| redisnats | redis + nats | — | — | — | — | unreleased |

Sources: D:server/engines.md, O:internal/app/engine.go, O:internal/configtypes/engine.go, O:internal/controllers/controllers.go.

---

## 6. Gaps: documented OSS items in this scope that the SPEC OSS list does not name

The SPEC list given covers: memory/Redis/Redis Cluster/NATS/PostgreSQL engines, broker/presence separation, PostgreSQL controller, async consumers (the 7 types), JWT conn/sub tokens with exp/iat/nbf/iss/aud, connect/subscribe/publish/RPC/refresh proxies, history size/TTL, epoch+offset, recovery, cache recovery, presence/stats, join/leave, Prometheus, health, logs, Docker/Helm/packages. Documented items **not** named there:

1. **Redis Sentinel** (`redis+sentinel://`, `sentinel_tls`) and **HAProxy-instead-of-Sentinel** guidance.
2. **Redis app-level sharding** (Jump hash; sharding across several clusters).
3. **Redis-compatible backends** policy (ElastiCache, Memorystore, KeyDB, DragonflyDB + `force_resp2`, Valkey).
4. **Redis IAM auth** (AWS/GCP/Azure), EXP v6.10.0.
5. Redis presence options `presence_ttl`, `presence_hash_field_ttl`, `presence_user_mapping`. Deprecated `history_use_lists`. `force_resp2`, `prefix`, `rediss://`.
6. **NATS raw mode** and `allow_wildcards`. The NATS broker has no history (should be stated explicitly as a limitation).
7. **PostgreSQL stream broker SQL surface** (`cf_stream_*` functions, version suppression, idempotency key, partitions, `use_notify`/`notify_dsn`, `binary_data`, `table_prefix` multi-tenancy, PG ≥16).
8. **Map broker backends** (memory/redis/postgres), needed for a "PostgreSQL-only cluster".
9. **History meta TTL** (`history_meta_ttl`, global 720h).
10. **Positioning** (`force_positioning`, `allow_positioning`, 3010 insufficient state).
11. `allow_recovery`, `force_recovery_mode`, **`auto_cache_recover`** (v6.8.3), connect-proxy `cache_recover`.
12. **Per-namespace `recovery_max_publication_limit` / `history_max_publication_limit`** (v6.10.3) and global client limits (300).
13. History iteration API semantics (`limit` 0/-1, `since`, `reverse`).
14. **JWKS**, **dynamic JWKS with safety rules** (v6.7.0), **HMAC rotation** (v6.6.1), **Ed25519 PEM** (v6.10.2), `user_id_claim`, **separate subscription token config**, `insecure_skip_token_signature_verify`, anonymous-connection options.
15. Connection-token claims beyond the registered ones: `info`, `b64info`, `channels`, `subs` (+ override), `meta`, `expire_at`. Subscription-token claims: `channel`, `info`, `data`, `b64*`, `override`, `expire_at`, undocumented `client`. The `jti` claim.
16. Channel permission model: private prefix `$`, user-limited `#` channels, `allow_*_for_client`, `_subscriber`, `_anonymous`, decision order, error codes 102/103/108/109/3500.
17. **Sub refresh proxy**, **subscribe stream proxy** (uni/bidi, EXP), map publish/remove proxy, shared-poll refresh proxy.
18. Proxy mechanics: **named proxies / per-namespace proxies**, `emulated_headers` (v6.9.0 security split), header translation rules, `static_headers`/`static_metadata`, GRPC TLS/credentials/compression, `binary_encoding`, `include_connection_meta`, `status_to_code_transforms`, `connect_code_to_unidirectional_disconnect`, `connect_code_to_http_response`, custom error/disconnect code ranges, empty response = allow.
19. Consumer **publication data mode**, method header/attribute, retry-only-100/111, graceful drain (v6.10.3), env-var override scheme.
20. **Graphite**, **native histograms** (v6.8.1), **OpenTelemetry tracing for server API** (+ GCP ADC, v6.8.2), Grafana dashboard, `instrument_http_handlers`, `recovered_publications_histogram`.
21. **Init endpoint** (v6.5.0), debug pprof, Swagger UI, internal port separation.
22. **TLS** (unified TLSConfig, ACME autocert), **load-balancing requirements** (no sticky sessions for emulation), **infra tuning**.
23. Binary releases / Homebrew / `install.sh`, **Sigstore signatures + SBOM** (v6.10.0), Docker tag policy for pre-releases.
24. Server SDK spec (gocent/pycent/...). Useful as a compatibility-test client.
25. Code-only, undocumented: `client.presence_update_interval` (27s), Kafka `partition_queue_max_size`/`dial_timeout`/`instance_id`/`method_header`, `opentelemetry.consuming`, `graphite.prefix`/`tags`, autocert `server_name`, PG broker/controller `tls`, controller `batch_size`, metric `client_subscriptions_accepted`, hidden `redisnats` broker.

---

## 7. Conflicts and open questions

### 7.1 Doc/source conflicts (record; ask for a decision where it matters)

| ID | Conflict | Evidence | Suggested default |
|---|---|---|---|
| CONFLICT-01 | PG controller `use_notify` default: the docs table says `true`. The code struct has no `default:"true"` tag, so the effective Go default is `false`, yet the code doc string says "Default <<true>>". | D:server/engines.md §Configuration options; O:internal/configtypes/engine.go `PostgresController.UseNotify` | Black-box test the reference binary. Until then, follow the docs (`true`) and flag it. |
| CONFLICT-02 | PG controller `batch_size` (1000) and the `tls` key on the PG broker/controller exist in code but not in the docs tables. | O:configtypes/engine.go | Accept them as compatible extras. Mark them undocumented. |
| CONFLICT-03 | engines.md says Sentinel **or Cluster** "must use a special scheme". The same page (and R:v6.1.0) says Cluster is auto-detected from a plain address. Code still accepts `redis+cluster://`. | D:server/engines.md §engine.redis.address info box vs §Redis Cluster support | Support both auto-detection and `redis+cluster://`. |
| CONFLICT-04 | `history_use_lists` limitations: the docs list reverse order and delta. The code doc adds no pagination and no idempotent publishing. | D:server/engines.md; O:configtypes/engine.go | Deprecated: consider not implementing it. Record a known difference. |
| CONFLICT-05 | Kafka `publication_data_mode` header defaults: docs say `""`. Code defaults to `centrifugo-channels`, `centrifugo-idempotency-key`, `centrifugo-delta`, and adds `version_header`, `version_epoch_header`, `tags_header_prefix`, `method_header` (undocumented for Kafka; documented for the other consumers). | D:server/consumers.md §consumers[].kafka; O:configtypes/types.go `KafkaConsumerConfig` | Follow the code defaults, matching the other consumers. Needs a black-box check. |
| CONFLICT-06 | observability.md §Logs uses `log_level`/`log_file` (v5 names). configuration.md and the code use `log.level`/`log.file`. | D:server/observability.md; D:server/configuration.md §log; O:configtypes/types.go | `log.level`/`log.file`. |
| CONFLICT-07 | The subscribe proxy "does not proxy subscriptions with token", yet `SubscribeRequest.token` is documented as "subscription token … when subscription token authorization is used". | D:server/proxy.md §Subscribe proxy info box vs §SubscribeRequest | Open question: when is `token` populated (stream proxy? PRO?). OQ-PROXY-01. |
| CONFLICT-08 | proxy.proto `RefreshResponse`/`SubRefreshResponse` carry `error`/`disconnect`, and `RefreshRequest` has `labels`. The docs say error/disconnect are ignored for refresh, and the docs table omits `labels`. | O:proxyproto/proxy.proto; D:server/proxy.md §Return custom error note | Ignore error/disconnect, as the docs say. Accept the fields on the wire. |
| CONFLICT-09 | proxy.proto has fields the docs omit: `ConnectResult.profile`, `SubscribeOptions.expire_at`, `SubscribeOptions.server_tags_filter`, `Error.temporary`. | O:proxyproto/proxy.proto | Accept on the wire. `profile` semantics are unknown (OQ-PROXY-02). |
| CONFLICT-10 | `nbf`: the SPEC lists it and the docs never mention it. The code enforces it and maps failure to "token expired" (109 / refresh flow), not "invalid". | O:internal/jwtverify/token_verifier_jwt.go (~610, ~784) | Enforce `nbf`. Map to the expired path. Document it as observed behavior. |
| CONFLICT-11 | Subscription-token `client` claim: implemented and referenced indirectly in channel_permissions.md (3500 for a token "issued for another … client"), but absent from channel_token_auth.md. | O:internal/client/handler.go; D:server/channel_permissions.md | Implement it. |
| CONFLICT-12 | `insecure_skip_jwks_endpoint_safety_check`: the code doc string says it disables a check about JWKS endpoints "accessible from the internet". The docs say it bypasses the regex alternation check. | O:configtypes/types.go; D:server/authentication.md | Follow the docs. |
| CONFLICT-13 | Valkey: engines.md says "no tests were performed by Centrifugal Labs". R:v6.0.3 says Valkey was added to the integration test suite. | D:server/engines.md; R:v6.0.3 | Cosmetic. Treat Valkey as supported and test it. |
| CONFLICT-14 | `broker.type` allowed values: docs say `redis`, `nats`, `postgres`. Code default and accepted values also include `memory` (and hidden `redisnats`). | O:internal/app/engine.go | Accept `memory`. |
| CONFLICT-15 | `client.presence_update_interval` (27s) exists in code but is undocumented. Presence liveness depends on it together with `presence_ttl` 60s. | O:configtypes/types.go | Mirror the default. Document it as observed. |
| CONFLICT-16 | Metric `centrifugo_client_subscriptions_accepted` exists in code but is not in the docs metrics list. | L:metrics.go; R:v6.9.5 | Expose it. |
| CONFLICT-17 | The consumer metrics description lists the consumers but omits Azure Service Bus. | D:server/observability.md §centrifugo_consumers_messages_processed_total | Cosmetic. |
| CONFLICT-18 | The health endpoint is advertised for K8s liveness and readiness, but the code always returns 200 `{}` without checking the broker. | D:server/configuration.md §health; O:internal/health/health.go | Match it: unconditional 200 `{}`. Readiness checks are a possible extension and must be flagged as a known difference. |

### 7.2 Open questions (undocumented behavior: do not guess)

- **OQ-ENG-01** Redis key layout, Lua script semantics, PUB/SUB channel naming and the cluster hash-tag scheme. Implementation detail, not documented. Interop with a reference node on the same Redis is out of scope unless the user wants mixed clusters. **Decision needed: must our server share a Redis with real Centrifugo nodes?**
- **OQ-ENG-02** How Redis Cluster is detected automatically. The mechanism is not documented.
- **OQ-ENG-03** A multi-node PG stream broker **without** `controller` enabled: the PG broker does not implement Controller, and the docs do not say what happens (no node discovery, or a startup error?). Needs a black-box test.
- **OQ-ENG-04** Memory engine: history eviction timing (TTL sweep cadence) and presence TTL for the memory presence manager. Not documented.
- **OQ-ENG-05** NATS broker and the PG broker with join/leave and presence: is presence required from a separate manager? Docs: NATS + Redis presence is an example. The behavior with the default memory presence across nodes is undocumented.
- **OQ-HIST-01** Epoch format and generation (docs: "arbitrary string"). The memory epoch changes on restart. Nothing is specified for Redis or PG.
- **OQ-HIST-02** Client "reject unrecovered" subscription flag -> 112 disconnect (L:client.go). Belongs to the client protocol scope. Confirm with the protocol researcher.
- **OQ-HIST-03** Exact interaction of `recovery_max_publication_limit` with cache mode, and with `history_size` larger than the limit.
- **OQ-AUTH-01** Clock-skew leeway for `exp`/`nbf`. The docs are silent and no leeway was observed in the verifier call.
- **OQ-AUTH-02** JWKS SSRF protections beyond the template allowlist (redirect following, response size limit, private-IP blocking). Not documented. Per CLAUDE.md we must add them regardless and record that as a known difference.
- **OQ-AUTH-03** Behavior when both `exp` and `expire_at` are absent but `sub_refresh`/refresh proxies are enabled.
- **OQ-PROXY-01** When is `SubscribeRequest.token` filled? (CONFLICT-07)
- **OQ-PROXY-02** Semantics of `ConnectResult.profile` (proto field 12). Likely PRO. Not documented.
- **OQ-PROXY-03** Refresh-proxy retry interval after a backend error ("retries after some time"). Not quantified.
- **OQ-PROXY-04** Sub-refresh scheduling granularity ("up to one minute").
- **OQ-CONS-01** Exact at-least-once guarantees per consumer after a crash, as opposed to a graceful shutdown. Docs: "in many cases reliable at least once". Must not be claimed as exactly-once.
- **OQ-CONS-02** PostgreSQL outbox: are rows deleted per batch even when some commands fail with non-temporary errors? (Docs: such messages are marked processed, which implies deletion.)
- **OQ-OBS-01** `opentelemetry.consuming` semantics (code-only key).
- **OQ-OBS-02** Full label sets for `client_recovered_publications` and the exact histogram buckets. Not fully documented.
- **OQ-OPS-01** Helm chart values and contract live in a separate repo (`centrifugal/helm-charts`) that is not cloned. A separate research pass is needed if Helm parity is in scope.
