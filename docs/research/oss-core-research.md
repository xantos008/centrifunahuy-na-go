<!-- Phase 0 research annex. Derived from public docs/OSS source at the pinned baseline. OQ-nn here are mirrored in open-questions.md as CORE-OQ-nn. Paths: DOCS=centrifugal.dev@69da714, OSS=centrifugo@v6.10.3, PROTO=protocol@v0.23.1, LIB=centrifuge@8888c6e (see OQ-18). -->

# Centrifugo OSS core: compatibility research (clean-room, documentation-derived)

- Target baseline: Centrifugo OSS **v6.10.3**. The `c-oss` clone is at commit `4b6d86e` ("Prepare v6.10.3 (#1313)", 2026-11-08), and `git describe` gives `v6.10.3`.
- Date accessed: 2026-11-15.
- Scope: OSS client protocol, transports, SDK spec, channels/namespaces, permissions, server API, server-side subscriptions, configuration/CLI, TLS, admin UI, delta compression, publication filtering, map subscriptions and shared poll. Engines, proxies, consumers and observability get only short rows here because other research covers them.
- Method: read-only reading of public docs and public OSS source. Nothing was built or executed. Every statement cites a path. A statement taken from **source code rather than docs** is labelled `[src]`, and it is evidence of implementation behaviour, not a documented contract.

## 0. Sources and path conventions

| Alias | Local clone (under `$SRC=/private/tmp/claude-501/.../scratchpad/src`) | Version / commit | Public URL mapping |
|---|---|---|---|
| DOCS | `$SRC/centrifugal.dev` | commit `69da714` (2026-11-13). `docs/` = current v6 docs | `docs/<p>.md` -> `https://centrifugal.dev/docs/<p>` (e.g. `docs/server/server_api.md` -> https://centrifugal.dev/docs/server/server_api). `blog/<YYYY-MM-DD-slug>.md` -> `https://centrifugal.dev/blog/<YYYY>/<MM>/<DD>/<slug>` (slug from filename; front matter may override it, so verify) |
| OSS | `$SRC/c-oss` | tag `v6.10.3`, `4b6d86e` | `https://github.com/centrifugal/centrifugo/blob/v6.10.3/<path>` |
| PROTO | `$SRC/c-proto` | `centrifugal/protocol` v0.23.1 (commit `4455d8e`). `c-oss/go.mod` pins `github.com/centrifugal/protocol v0.23.1` | `https://github.com/centrifugal/protocol/blob/v0.23.1/<path>` |
| LIB | `$SRC/c-lib` | `centrifugal/centrifuge` HEAD `8888c6e` (2026-11-11). **Note:** `c-oss/go.mod` pins `v0.40.2-0.20261031074606-ae446d43faf0`, which differs from the clone HEAD | `https://github.com/centrifugal/centrifuge/blob/<commit>/<path>` |

Below, paths are written relative to the alias, e.g. `DOCS:docs/server/codes.md` or `OSS:internal/apiproto/api.proto`.

---

## 1. Feature table

Status legend:
- **GA**: no experimental, deprecated or removed marker in the docs.
- **experimental**: the docs say so explicitly.
- **removed**: explicitly removed.
- **draft/WIP**: unreleased doc.

| ID | Feature | Status | Min version (if stated) | Config keys | API / protocol surface | Source path(s) | Notes |
|---|---|---|---|---|---|---|---|
| OSS-PROTO-01 | Bidirectional client protocol: Command/Reply envelopes; `id` (uint32, client-chosen, echoed); `id==0` marks a push | GA | - | - | `Command`, `Reply`, `Push`, `Error` | DOCS:docs/transports/client_protocol.md; PROTO:definitions/client.proto | No `oneof`. The server takes the first set request field. Connect must be the first command, and any other command before a successful connect gives a disconnect (proto comment). A command with no request (other than a pong) gives a bad-request disconnect (proto comment) |
| OSS-PROTO-02 | JSON encoding: `bytes` fields carry raw embedded JSON, not base64 | GA | - | - | - | DOCS:docs/transports/client_protocol.md; PROTO:definitions/client.proto (header) | Exception: `Dictionary.data_b64` is base64 (PRO draft feature, see OSS-PROTO-14) |
| OSS-PROTO-03 | Protobuf encoding | GA | - | - | WS subprotocol `centrifuge-protobuf` | DOCS:docs/transports/websocket.md; DOCS:docs/transports/client_sdk.md | [src] LIB:handler_websocket.go also accepts the query params `format=protobuf` or `cf_protocol=protobuf`, and offers the subprotocols `centrifuge-json` and `centrifuge-protobuf`. [src] OSS:internal/wt/handler.go uses `cf_protocol=protobuf` for WebTransport. [src] LIB:handler_http_stream.go and LIB:emulation.go pick Protobuf when the request has `Content-Type: application/octet-stream`. [src] The bidirectional SSE transport is always JSON (LIB:handler_sse.go `Protocol()` returns JSON) |
| OSS-PROTO-04 | Frame batching: JSON is newline-delimited; Protobuf is varint length-delimited | GA | - | - | Many Commands per frame (client to server). Many Replies per frame (server to client), and a frame may mix replies and pushes | DOCS:docs/transports/client_protocol.md | |
| OSS-PROTO-05 | Strict JSON command decoding (RFC 8259): exact-case keys; duplicate key, invalid UTF-8 or trailing data is an error. Protobuf strings must be valid UTF-8. An undecodable command gives disconnect `3501` | GA | v6.10.0 | - | - | DOCS:docs/transports/client_protocol.md | Breaking for non-official SDKs |
| OSS-PROTO-06 | App-level ping/pong: server sends an empty Reply (`{}`) as ping; client answers with an empty Command when `ConnectResult.pong=true` | GA | - | `client.ping_interval` (default `25s`), `client.pong_timeout` (default `8s`). `ping_interval` must be greater than `pong_timeout` | `ConnectResult.ping`, `ConnectResult.pong` | DOCS:docs/transports/overview.md; DOCS:docs/transports/client_protocol.md; PROTO:definitions/client.proto | A pong without a preceding ping gives a bad-request disconnect (proto). No pong gives disconnect `3012`. Proto says `Command.ping`/`PingRequest` is "not supported by the server at the moment" |
| OSS-PROTO-07 | WebSocket native frame ping/pong debug mode | GA (documented for debugging only) | - | URL param `cf_ws_frame_ping_pong=true` | - | DOCS:docs/transports/websocket.md | |
| OSS-PROTO-08 | Error replies `{code,message,temporary}`. `temporary` drives retry behaviour | GA | - | - | `Error` | DOCS:docs/transports/client_protocol.md; DOCS:docs/server/codes.md | See section 2.3 |
| OSS-PROTO-09 | Disconnect / unsubscribe codes with reconnect semantics | GA | - | - | `Disconnect`, `Unsubscribe` pushes; WS close frame | DOCS:docs/server/codes.md; DOCS:docs/transports/client_api.md; PROTO:definitions/client.proto | Code ranges in the docs and the proto conflict (OQ-03) |
| OSS-PROTO-10 | Client name/version in ConnectRequest (observability only) | GA | - | - | `ConnectRequest.name` (max 16 chars), `ConnectRequest.version` (max 64 chars) | DOCS:docs/transports/client_protocol.md | |
| OSS-PROTO-11 | Headers emulation: `ConnectRequest.headers` is turned into proxy HTTP headers | GA | v6.0.0 | (proxy header config) | `ConnectRequest.headers` | DOCS:blog/2025-01-16-centrifugo-v6-released.md (Headers emulation); DOCS:docs/server/proxy.md ("HTTP headers emulation"); DOCS:docs/transports/uni_client_protocol.md | SDK matrix: js, dart, rust, c# |
| OSS-PROTO-12 | Server time in connect reply/push | GA | - | `client.connect_include_server_time` | `ConnectResult.time`, `Connect.time` (Unix ms) | DOCS:docs/server/configuration.md | |
| OSS-PROTO-13 | Subscription feature flags in `SubscribeRequest.flag`: bit0 = numeric channel id, bit1 = reject unrecovered (gives error 112) | proto-defined | - | - | `SubscribeRequest.flag`, `SubscribeResult.id`, `Push.id` | PROTO:definitions/client.proto | The numeric channel id relates to PRO "channel compaction" (DOCS:docs/pro/bandwidth_optimizations.md). The OSS behaviour of bit1 is not described in the docs (OQ-07) |
| OSS-PROTO-14 | Connection feature flags / dictionary compression: `ConnectRequest.flag` bit0, `ConnectRequest.dict`, `ConnectRequest.profile`, `ConnectResult.flag`, `ConnectResult.dict` | **draft/WIP (PRO, unreleased)** | - | - | `Dictionary` message | PROTO:definitions/client.proto; DOCS:docs/pro/dictionary_compression.md (front matter `draft: true`, "Not released") | Do NOT implement as GA. Decode these fields and ignore them |
| OSS-PROTO-15 | Recovery fields: `recover`, `epoch`, `offset`, `recovered`, `was_recovering`, `positioned`, `recoverable` | GA | - | channel options (section 3) | SubscribeRequest/Result | DOCS:docs/transports/client_protocol.md; DOCS:docs/transports/client_api.md; DOCS:docs/server/history_and_recovery.md | |
| OSS-PROTO-16 | Strict client protocol version 2 (only v2 is current) | GA | v4 introduced v2 | - | - | DOCS:docs/getting-started/migration-v4.md (`cf_protocol_version=v2` used during the v4 migration); [src] LIB:transport.go `ProtocolVersion2` "current stable" | The v6 docs do not document `cf_protocol_version` (OQ-08) |
| OSS-TRANS-01 | Bidirectional WebSocket | GA, enabled by default | - | `websocket.disabled`, `handler_prefix` (default `/connection/websocket`), `message_size_limit` (65536), `decompressed_message_size_limit` (0 = 10x the size limit, since 6.8.4), `read_buffer_size`, `write_buffer_size`, `use_write_buffer_pool`, `compression`, `compression_min_size`, `compression_level` (default 1), `write_timeout` (config struct only, undocumented) | `/connection/websocket` | DOCS:docs/transports/websocket.md; DOCS:docs/server/configuration.md; [src] OSS:internal/configtypes/types.go | permessage-deflate. A message over the decompressed limit closes the connection with "message too big" |
| OSS-TRANS-02 | WebSocket over HTTP/2 (RFC 8441), WS and uni WS | **experimental** | v6.5.0 | env `GODEBUG=http2xconnect=1`; TLS, or `http_server.h2c_external` | - | DOCS:docs/transports/websocket.md; DOCS:docs/transports/uni_websocket.md | The config struct also has `http_server.external_http2_extended_connect`, which is undocumented (OQ-09) |
| OSS-TRANS-03 | HTTP-streaming with bidirectional emulation | GA | v4 | `http_stream.enabled`, `handler_prefix` (`/connection/http_stream`), `max_request_body_size` (65536) | `/connection/http_stream` + `/emulation` | DOCS:docs/transports/http_stream.md | Docs say only centrifuge-js implements it. The SDK matrix lists bidirectional emulation for js, rust and c#. Supports Protobuf |
| OSS-TRANS-04 | SSE (EventSource) with bidirectional emulation | GA | v4 | `sse.enabled`, `handler_prefix` (`/connection/sse`), `max_request_body_size` (65536, POST only) | `/connection/sse` + `/emulation` | DOCS:docs/transports/sse.md | [src] JSON only. [src] Connect payload comes from the `cf_connect` URL param on GET, or the body on POST (LIB:handler_sse.go) |
| OSS-TRANS-05 | Emulation endpoint (client-to-server half of the emulation) | GA | v4 | `emulation.handler_prefix` (`/emulation`), `emulation.max_request_body_size` (65536). Auto-enabled when `sse` or `http_stream` is enabled | `EmulationRequest{node, session, data}`. `ConnectResult.session` and `ConnectResult.node` are set for emulation | DOCS:docs/server/configuration.md; PROTO:definitions/client.proto; DOCS:blog/2022-07-19-centrifugo-v4-released.md | Requests are proxied to the node that owns the connection |
| OSS-TRANS-06 | WebTransport (HTTP/3, one bidirectional stream) | **experimental** | - | `webtransport.enabled`, `handler_prefix` (`/connection/webtransport`), `message_size_limit` (v6); needs `http_server.http3.enabled` and TLS | `/connection/webtransport` | DOCS:docs/transports/webtransport.md; DOCS:blog/2025-01-16-centrifugo-v6-released.md | JSON and Protobuf. Known issue: disconnect advice is not always delivered |
| OSS-TRANS-07 | HTTP/3 server | **experimental** | - | `http_server.http3.enabled` | - | DOCS:docs/server/configuration.md | Required for WebTransport |
| OSS-TRANS-08 | SockJS | **removed** | removed in v6 | (was `sockjs`, `sockjs_url`) | (was `/connection/sockjs`) | DOCS:docs/transports/sockjs.md; DOCS:docs/getting-started/migration-v6.md | Do not implement |
| OSS-TRANS-09 | Unidirectional protocol: server sends bare `Push` frames; first push is `connect`; empty push = ping | GA | - | - | `ConnectRequest` (token, data, name, version, subs, headers); `subs[ch]` uses only recover/epoch/offset (plus `delta`, per proto) | DOCS:docs/transports/uni_client_protocol.md; PROTO:definitions/client.proto | `Connect.pong` is never set on uni transports |
| OSS-TRANS-10 | Unidirectional WebSocket | GA | - | `uni_websocket.enabled`, `handler_prefix` (`/connection/uni_websocket`), `message_size_limit`, `decompressed_message_size_limit` (6.8.4), `join_push_messages` (6.0.3), `disable_closing_handshake` (6.5.1), `disable_disconnect_push` (6.5.1), `compression`, `compression_min_size`, `compression_level`, plus buffer options and `write_timeout` in the struct | first WS message = ConnectRequest, or `cf_connect` URL param (since 6.3.0) | DOCS:docs/transports/uni_websocket.md | JSON only. Uses both native WS ping and app-level `{}` pings |
| OSS-TRANS-11 | Unidirectional HTTP streaming | GA | - | `uni_http_stream.enabled`, `handler_prefix`, `max_request_body_size`, `connect_code_to_http_response` | POST with a JSON ConnectRequest body; newline-delimited JSON pushes | DOCS:docs/transports/uni_http_stream.md; DOCS:docs/server/proxy.md | JSON only |
| OSS-TRANS-12 | Unidirectional SSE | GA | - | `uni_sse.enabled`, `handler_prefix`, `max_request_body_size`, `connect_code_to_http_response` | GET with `cf_connect` (JSON in the URL, keep the URL under 2048 chars) or POST body; `data: {...}` frames | DOCS:docs/transports/uni_sse.md | `Last-Event-Id` is not supported. JSON only |
| OSS-TRANS-13 | Unidirectional gRPC | GA | - | `uni_grpc.enabled`, `port` (11000), `address`, `max_receive_message_size` (65536), `tls` | proto `internal/unigrpc/unistream/unistream.proto`, published to Buf `buf.build/centrifugo/unistream` with per-release labels since v6.10.0 | DOCS:docs/transports/uni_grpc.md; OSS:internal/unigrpc/ | Docs say "JSON and binary". `client.connection_limit` and `client.connection_rate_limit` do not apply to it |
| OSS-TRANS-14 | Unidirectional connect error to disconnect-code mapping | GA | - | `client.connect_code_to_unidirectional_disconnect{enabled,transforms[{code,to{code,reason}}]}`; proxy `status_to_code_transforms[].to_disconnect` | - | DOCS:docs/server/proxy.md (~l.1395-1480) | By default any proxy error becomes `3004` for uni clients |
| OSS-TRANS-15 | Connection init endpoint (`GET`, returns `{}`) | GA | v6.5.0 | `init.enabled`, `init.handler_prefix` (`/connection/init`) | - | DOCS:docs/server/configuration.md | Used as an LB ping or to warm HTTP/2 |
| OSS-TRANS-16 | H2C on external endpoints | GA | v6.5.0 | `http_server.h2c_external` | - | DOCS:docs/server/configuration.md | |
| OSS-SDK-01 | Client SDK spec: connection states disconnected/connecting/connected/closed; subscription states; backoff with full jitter; token refresh via a getToken callback; `UnauthorizedError` semantics | GA | - | - | - | DOCS:docs/transports/client_api.md | |
| OSS-SDK-02 | Official SDK list: centrifuge-js, -go, -dart, -swift, -java, -python, -rust, -csharp | GA | - | - | - | DOCS:docs/transports/client_sdk.md | SDK feature matrix is in the same file |
| OSS-SDK-03 | Subscription `getState` callback (SDK-side app-owned state) | GA | v6.8.0 | - | uses history/recovery | DOCS:docs/transports/client_api.md; DOCS:blog/2026-07-27-app-owned-state-stream-subscriptions.md | Matrix marks all SDKs. The blog lists js, dart, swift, go, java and python |
| OSS-SDK-04 | Server SDKs: gocent, pycent, jscent, phpcent, rubycent, rustcent, csharpcent | GA | - | - | HTTP API | DOCS:docs/server/server_api.md; DOCS:docs/server/server_sdks.md | |
| OSS-CH-01 | Channel name rules: ASCII only, max length | GA | - | `channel.max_length` (255) | - | DOCS:docs/server/channels.md | Reserved symbols: `:` `#` `$` `/` (PRO patterns), `*` `&` (future) |
| OSS-CH-02 | Namespaces: `name` must match `^[-a-zA-Z0-9_]{2,}$`; options from `channel.without_namespace` and `channel.namespaces[]`; an unknown namespace gives 102 | GA | - | `channel.namespaces`, `channel.without_namespace`, `channel.namespace_boundary` (struct only) | - | DOCS:docs/server/channels.md; [src] OSS:internal/configtypes/types.go | Namespace = part before the first `:` after stripping the private prefix |
| OSS-CH-03 | User-limited channels `#` with comma-separated users | GA | - | `allow_user_limited_channels`; `channel.user_boundary` and `channel.user_separator` (struct only, undocumented) | - | DOCS:docs/server/channels.md; [src] OSS:internal/configtypes/types.go | Limits subscribing only |
| OSS-CH-04 | Private channel prefix `$`: subscribing without a token is rejected | GA (legacy compatibility) | - | `channel.private_prefix` (default `$`) | - | DOCS:docs/server/channels.md; DOCS:docs/server/channel_permissions.md | |
| OSS-CH-05 | Channel options (full list in section 3) | GA, except where marked | various | section 3 | - | DOCS:docs/server/channels.md; OSS:internal/configtypes/namespace.go | |
| OSS-CH-06 | Publication data format validation | GA | v6.5.2 | `publication_data_format` (`""`, `json`, `json_object`, `binary`), at top level (`channel.publication_data_format`) or per namespace | - | DOCS:docs/server/channels.md | With the default, invalid JSON delivered to a JSON client gives disconnect `3506` |
| OSS-PERM-01 | Subscribe permission decision order: private gate, token, user-limited, subscribe proxy, stream proxy, [PRO caps], allow_subscribe_for_client/anonymous, client.insecure, [PRO CEL] | GA | - | namespace `allow_*` | 103/102/109/3500 outcomes | DOCS:docs/server/channel_permissions.md | Publish, history and presence each have their own models in the same doc |
| OSS-PERM-02 | Positioning / recovery / join-leave permission models | GA | v6.10.0 also applies to subscribe proxy | `force_*`, `allow_positioning`, `allow_recovery`, `force_push_join_leave` | - | DOCS:docs/server/channel_permissions.md; DOCS:docs/server/channels.md | Subscribe-stream-proxy subscriptions never get positioning or recovery |
| OSS-SUB-01 | Client-side stream subscriptions | GA | - | - | subscribe/unsubscribe | DOCS:docs/transports/client_api.md | One subscription per channel per connection (105) |
| OSS-SUB-02 | Server-side subscriptions: JWT `channels`, connect proxy `channels`, uni connect `subs`, server API subscribe | GA | - | - | `Push.subscribe`/`unsubscribe`; `ConnectResult.subs`; `ConnectRequest.subs` | DOCS:docs/server/server_subs.md; DOCS:docs/transports/client_api.md | Uni `subs` never subscribes to private-prefix, proxy-authorized or non-`stream` namespaces |
| OSS-SUB-03 | Automatic personal channel subscription and single connection per user | GA | - | `client.subscribe_to_user_personal_channel.{enabled, personal_channel_namespace, single_connection}` | - | DOCS:docs/server/server_subs.md | single_connection needs presence and is best-effort |
| OSS-SUB-04 | Map subscriptions (types `map`, `map_clients`, `map_users`; modes ephemeral/recoverable/persistent) | **experimental** | v6.8.0 | `subscription_type`, `map.*`, `map_broker.type` (memory/redis/postgres), `map_clients_presence_channel_prefix`, `map_users_presence_channel_prefix` | `SubscribeRequest.type` (1/2/3), `phase`, `cursor`, `limit`, `asc`; `SubscribeResult.state`, `phase`, `cursor`; `PublishRequest.type`/`key`/`removed`; `Publication.key`/`removed`/`score`; API `map_*` | DOCS:docs/server/map_subscriptions.md; PROTO:definitions/client.proto | Only centrifuge-js supports it. Redis Cluster for map needs PRO sharded PUB/SUB. PRO adds per-namespace map brokers |
| OSS-SUB-05 | Map publish/remove proxy | **experimental** (part of map) | v6.8.0 | `map.publish_proxy_enabled`/`_name`, `map.remove_proxy_enabled`/`_name` | proxy events `map_publish`, `map_remove` | DOCS:docs/server/map_subscriptions.md; DOCS:docs/server/proxy.md | Cannot be combined with `map.client_key` |
| OSS-SUB-06 | Shared poll subscriptions (HMAC-signed tracked keys; versionless/versioned modes; direct publish) | **experimental** | v6.8.0 | top-level `shared_poll.{hmac_secret_key, hmac_previous_secret_key, hmac_previous_secret_key_valid_until, concurrency_limit}`; namespace `shared_poll.{proxy_name, refresh_interval, refresh_batch_size, max_keys_per_connection, mode, channel_shutdown_delay, track_expired_extra_delay, publish_enabled}`; `channel.proxy.shared_poll_refresh` | `SubscribeRequest.type=4`; `SubRefreshRequest.track`/`untrack` (`TrackBatch{signature, items[KeyedItem{key, version}]}`); `SubRefreshResult.items`; `Publication.version`; API `shared_poll_publish` | DOCS:docs/server/shared_poll.md; PROTO:definitions/client.proto; OSS:internal/configtypes/namespace.go | Only centrifuge-js supports it. `shared_poll_publish` needs `publish_enabled: true`, otherwise 108 |
| OSS-SUB-07 | Proxy subscription streams (unidirectional/bidirectional gRPC streams to the backend) | **experimental** | - | `subscribe_stream_proxy_enabled`/`_name`/`_bidirectional`, `channel.proxy.subscribe_stream` | - | DOCS:docs/server/proxy_streams.md | |
| OSS-SUB-08 | Cache recovery mode / auto cache recover | GA | v5.4.0 / auto since v6.8.3 | `force_recovery_mode: cache`, `auto_cache_recover` | - | DOCS:docs/server/cache_recovery.md; DOCS:docs/server/channels.md | |
| OSS-SUB-09 | Delta compression (Fossil) | GA (no experimental marker) | v5.x (no version stated in the doc) | `allowed_delta_types: ["fossil"]`, `delta_publish`; publish/broadcast `delta` | `SubscribeRequest.delta="fossil"`, `SubscribeResult.delta`, `Publication.delta` | DOCS:docs/server/delta_compression.md | Needs history plus positioning or recovery. Not available with Redis `history_use_lists` or with a tags filter. `skip_history` is not allowed. History responses are not delta-encoded. All official SDKs support it |
| OSS-SUB-10 | Publication filtering by tags (client-side filter) | GA (no experimental marker) | v6.4.0 | `allow_tags_filter` | `SubscribeRequest.tf` (`FilterNode{op,key,cmp,val,vals,nodes}`); publish `tags` | DOCS:docs/server/publication_filtering.md; PROTO:definitions/client.proto | Operators: eq, neq, in, nin, ex, nex, sw, ew, ct, gt, gte, lt, lte; logical and/or/not. An invalid filter gives 107. Docs on SDK support conflict (OQ-05) |
| OSS-SUB-11 | Client RPC (proxied) and RPC namespaces | GA | - | `rpc.without_namespace`, `rpc.namespaces[{name, proxy_enabled, proxy_name}]`, `rpc.namespace_boundary` (":"), `rpc.proxy` | `RPCRequest{data, method}` / `RPCResult{data}` | DOCS:docs/server/configuration.md; DOCS:docs/server/proxy.md; OSS:internal/configtypes/rpc_namespace.go | |
| OSS-SUB-12 | Built-in RPC `ping` extension | GA | - | `rpc.ping.enabled`, `rpc.ping.method` (default `ping`) | RPC returns an empty reply | DOCS:docs/server/configuration.md; OSS:internal/app/app.go:555 | |
| OSS-SUB-13 | Async send (client to server, no reply) | GA | - | - | `SendRequest{data}`, sent with `id=0` | DOCS:docs/transports/client_protocol.md; PROTO:definitions/client.proto | Without a server message handler the result is disconnect 3508 (codes.md) |
| OSS-SUB-14 | Version-based publish dedup (`version`, `version_epoch`) and `idempotency_key` (5 min window) | GA | version since v6.2.0 | - | publish/broadcast fields | DOCS:docs/server/server_api.md | NATS broker ignores `idempotency_key`. Version checks need history |
| OSS-API-01 | HTTP server API: POST `/api/<method>`, JSON body | GA | - | `http_api.{disabled, handler_prefix ("/api"), key, error_mode, external, insecure}` | section 2.1 | DOCS:docs/server/server_api.md; OSS:internal/api/handler.go | Auth: header `X-API-Key`, or `?api_key=`; [src] also `Authorization: apikey <KEY>` (OSS:internal/middleware/auth.go), which DOCS:docs/server/map_subscriptions.md examples use. Bad key gives 401 with an empty body; non-POST gives 405; invalid JSON gives 400 with plain text; unknown path gives 404 |
| OSS-API-02 | gRPC server API | GA | - | `grpc_api.{enabled, error_mode, address, port (10000), key, tls, reflection, max_receive_message_size}` | `service CentrifugoApi` in OSS:internal/apiproto/api.proto; Buf `buf.build/centrifugo/apiproto` | DOCS:docs/server/server_api.md | **No auth by default.** Key goes in metadata `authorization: apikey <KEY>`. Not compatible with Buf Connect HTTP |
| OSS-API-03 | Transport error mode (HTTP status / gRPC code mapping) | GA | v5.1.0 (111 mapping since v6.10.0) | `http_api.error_mode`, `grpc_api.error_mode` = `transport`; per request: header `X-Centrifugo-Error-Mode: transport` or gRPC metadata `x-centrifugo-error-mode` | section 2.2 | DOCS:docs/server/server_api.md; OSS:internal/apiproto/errors.go | |
| OSS-API-04 | `Error.temporary` in server API responses | GA | v6.10.0 | - | - | DOCS:docs/server/server_api.md | Set for 100 and 111. Requests that can never succeed now return a specific code instead of 100 (v6.10.0) |
| OSS-API-05 | Swagger UI for the HTTP API | GA | - | `swagger.enabled`, `swagger.handler_prefix` (`/swagger`) | - | DOCS:docs/server/configuration.md | Internal endpoint |
| OSS-CFG-01 | Config file formats JSON / YAML / TOML (picked by extension) | GA | - | `-c/--config` (default `config.json`) | - | DOCS:docs/server/configuration.md; OSS:internal/app/cmd.go | |
| OSS-CFG-02 | Precedence: file < env < flags | GA | - | - | - | DOCS:docs/server/configuration.md | Empty env vars count as unset |
| OSS-CFG-03 | Env naming: `CENTRIFUGO_` + upper-cased path with `_` for nesting; bools via Go ParseBool; `[]string` space-separated; arrays of objects as JSON; MapStringString as JSON (keys lower-cased); StringKeyValues | GA | - | - | - | DOCS:docs/server/configuration.md | `CENTRIFUGO_VAR_*` extrapolation into MapStringString/StringKeyValues values (since 6.3.0) |
| OSS-CFG-04 | `.env` file support | GA | v6.0.0 | - | - | DOCS:blog/2025-01-16-centrifugo-v6-released.md | Lookup path and precedence are not documented (OQ-10) |
| OSS-CFG-05 | Validation on start (exit code 1); WARN logs for unknown keys | GA | - | `checkconfig --strict` fails on unknown keys [src] | - | DOCS:docs/server/configuration.md; OSS:internal/cli/checkconfig.go | |
| OSS-CFG-06 | Config types: Duration (`ms`, `s`, `m`, `h`; 1ms resolution, some options second precision), MapStringString, StringKeyValues, arrays of objects, TLS object | GA | - | - | - | DOCS:docs/server/configuration.md | |
| OSS-CFG-07 | SIGHUP reload (token secrets and channel options only). A reload that needs a new proxy is rejected (v6.10.3) | GA | reject since v6.10.3 | - | - | DOCS:docs/server/configuration.md | |
| OSS-CFG-08 | Graceful shutdown on SIGINT/SIGTERM | GA | - | `shutdown.timeout` (30s); a timeout gives exit 1 | - | DOCS:docs/server/configuration.md | Order: stop accepting, let consumers ack, close clients, close backends |
| OSS-CFG-09 | `--watch-config-file`: in-process restart on config file change (development) | GA | v6.10.3 | flag | - | DOCS:docs/server/configuration.md; OSS:internal/app/cmd.go | Keeps the PID. An invalid change is ignored |
| OSS-CFG-10 | Endpoint prefixes and internal port | GA | - | `*.handler_prefix`, `http_server.{internal_port, internal_address, internal_tls}`, `http_api.external`, `admin.external` | - | DOCS:docs/server/configuration.md | Internal endpoints: `/api`, admin (`/`, `/admin/auth`, `/admin/api`), `/metrics`, `/health`, `/debug/pprof`, `/swagger` |
| OSS-CFG-11 | Anonymous usage stats (every 24h) | GA | - | `usage_stats.disabled` | - | DOCS:docs/server/configuration.md | A clean-room server should not send telemetry; record as a known difference |
| OSS-CFG-12 | PID file | GA | - | `pid_file`, flag `--pid_file` | - | DOCS:docs/server/configuration.md | |
| OSS-CFG-13 | Debug pprof endpoints | GA | - | `debug.enabled`, `debug.handler_prefix` | `/debug/pprof/` | DOCS:docs/server/configuration.md | |
| OSS-CFG-14 | Node options | GA | - | `node.name`, `node.info_metrics_aggregate_interval` (60s) | `info` metrics | DOCS:docs/server/server_api.md; [src] OSS:internal/configtypes/types.go | |
| OSS-CFG-15 | `enable_unreleased_features` top-level key | undocumented | - | - | - | [src] OSS:internal/config/config.go | OQ-11 |
| OSS-CLI-01 | Console commands | GA | defaultconfig/defaultenv since v6.0 | section 2.6 | - | DOCS:docs/server/console_commands.md; OSS:internal/cli/*.go | `serve` exists in code but not in the docs |
| OSS-TLS-01 | TLS object `{enabled, cert_pem, key_pem, server_ca_pem, client_ca_pem, insecure_skip_verify, server_name}`; PEM given as raw, then base64, then file path (in that priority) | GA | unified in v6 | `http_server.tls`, `grpc_api.tls`, `uni_grpc.tls`, `http_server.internal_tls`, `http_server.tls_external` | - | DOCS:docs/server/configuration.md; DOCS:docs/server/tls.md | Also used for proxies, NATS, PG, Kafka |
| OSS-TLS-02 | ACME autocert (Let's Encrypt) | GA | - | `http_server.tls_autocert.{enabled, host_whitelist, cache_dir, email, http, http_addr (":80")}` | - | DOCS:docs/server/tls.md; DOCS:docs/server/configuration.md | |
| OSS-ORIG-01 | Origin check / CORS | GA | - | `client.allowed_origins` (patterns with `*`; requests without `Origin` pass) | - | DOCS:docs/server/configuration.md | Also enables CORS for HTTP-based uni transports |
| OSS-CLI-02 | Client connection limits and tuning | GA | - | `client.{channel_limit (128), user_connection_limit, connection_limit, connection_rate_limit (gives 503), concurrency, queue_max_size (1MiB), stale_close_delay, history_max_publication_limit (300), recovery_max_publication_limit (300)}` | - | DOCS:docs/server/configuration.md | Struct-only, undocumented: `expired_close_delay`, `expired_sub_close_delay`, `presence_update_interval`, `channel_position_check_delay`, `channel_position_max_time_lag` (OQ-12) |
| OSS-AUTH-00 | Auth modes adjacent to the protocol | GA | - | `client.user_id_http_header`, `client.allow_anonymous_connect_without_token`, `client.disallow_anonymous_connection_tokens`, `client.insecure`, `client.insecure_skip_token_signature_verify`, `client.token.*`, `client.subscription_token.enabled` | - | DOCS:docs/server/configuration.md; [src] OSS:internal/configtypes/types.go | JWT details are out of scope here |
| OSS-ADM-01 | Admin web UI (embedded React SPA) | GA | - | `admin.{enabled, password, secret, insecure, handler_prefix, web_path, web_proxy_address, external}` | admin endpoints `/`, `/admin/auth`, `/admin/api` | DOCS:docs/server/admin_web.md; DOCS:docs/server/configuration.md | Commands it can run: publish, broadcast, subscribe, unsubscribe, disconnect, history, history_remove, presence, presence_stats, info, channels. Failed logins are throttled per IP. Tracing, analytics and push device views are PRO. The UI source is a separate repo (github.com/centrifugal/web); **do not copy its UI assets or text** (CLAUDE.md) |
| OSS-ENG-00 | Engines (summary only) | memory, redis = GA; nats broker = GA (at-most-once); PostgreSQL broker = **experimental** (PG16+); PostgreSQL controller = **experimental** (PG16+); Redis IAM auth = **experimental** (v6.10.0); Tarantool = **removed** in v6 | - | `engine`, `broker`, `presence_manager`, `map_broker`, `controller` | - | DOCS:docs/server/engines.md; DOCS:docs/getting-started/migration-v6.md | Covered elsewhere. Listed for gap analysis |

---

## 2. Complete lists

### 2.1 Server API methods (HTTP `/api/<path>` + gRPC `CentrifugoApi`)

Source: DOCS:docs/server/server_api.md, DOCS:docs/server/map_subscriptions.md, DOCS:docs/server/shared_poll.md, OSS:internal/api/handler.go (HTTP route table), OSS:internal/apiproto/api.proto (gRPC service), OSS:internal/api/api.go (OSS executor).

Every response has the form `{error?: Error{code, message, temporary?}, result?: ...}`.

**OSS-implemented (21 HTTP routes, and the same set in gRPC):**

| # | HTTP path | gRPC rpc | Request fields | Result fields | Doc |
|---|---|---|---|---|---|
| 1 | `/publish` | Publish | `channel`*, `data`* (JSON), `skip_history`, `tags` map, `b64data`, `idempotency_key`, `delta`, `version` (6.2.0+), `version_epoch` | `offset`, `epoch` | server_api.md |
| 2 | `/broadcast` | Broadcast | `channels`*[], `data`*, plus the same optional fields as publish | `responses[]PublishResponse` | server_api.md |
| 3 | `/subscribe` | Subscribe | `user`*, `channel`*, `info`, `b64info`, `client`, `session`, `data`, `b64data`, `recover_since{offset, epoch}`, `expire_at`, `override{presence, join_leave, force_push_join_leave, force_positioning, force_recovery}` (each a BoolValue `{value}`); PRO-only: `label_filter`, `all_users` | empty | server_api.md |
| 4 | `/unsubscribe` | Unsubscribe | `user`*, `channel`*, `client`, `session`; PRO: `label_filter`, `all_users` | empty | server_api.md |
| 5 | `/disconnect` | Disconnect | `user`*, `client`, `session`, `whitelist[]`, `disconnect{code, reason}`; PRO: `label_filter`, `all_users` | empty | server_api.md |
| 6 | `/refresh` | Refresh | `user`*, `client`, `session`, `expired`, `expire_at`, `info` (applied only with a future `expire_at`); PRO: `label_filter`, `all_users` | empty | server_api.md. Caution: without `expire_at` the connection becomes non-expiring |
| 7 | `/presence` | Presence | `channel`* | `presence` map clientId -> `ClientInfo{client, user, conn_info, chan_info}` | server_api.md |
| 8 | `/presence_stats` | PresenceStats | `channel`* | `num_clients`, `num_users` | server_api.md |
| 9 | `/history` | History | `channel`*, `limit` (unset = position only), `since{offset, epoch}`, `reverse` | `publications[]`, `offset`, `epoch` | server_api.md; history_and_recovery.md "History iteration API" |
| 10 | `/history_remove` | HistoryRemove | `channel`* | empty (top stream position is kept) | server_api.md |
| 11 | `/channels` | Channels | `pattern` (gobwas/glob) | `channels` map -> `{num_clients}` | server_api.md. No pagination |
| 12 | `/info` | Info | - | `nodes[]{uid, name, version, num_clients, num_users, num_subs, num_channels, uptime, metrics{interval, items}, process (PRO)}` | server_api.md |
| 13 | `/batch` | Batch | `commands[]` (each `{<method>: req}`), `parallel` | `replies[]` | server_api.md |
| 14 | `/rpc` | RPC | `method`*, `params` (bytes) | `RPCResult` | **Not documented** in server_api.md. Present in OSS:internal/api/handler.go:54 and OSS:internal/apiproto/api.proto:732. Dispatches to registered server-side RPC extensions (`SetRPCExtension`), and an unknown method gives 104 not found [src OSS:internal/api/api.go:868]. OQ-01 |
| 15 | `/map_publish` | MapPublish | `channel`, `key`, `data`, `b64data`, `key_mode` (`if_new`/`if_exists`), `idempotency_key`, `tags`, `version`, `version_epoch`, `delta` | `offset`, `epoch`, `suppressed`, `suppress_reason` (`idempotency`, `version`, `key_exists`, `key_not_found`) | map_subscriptions.md (experimental) |
| 16 | `/map_remove` | MapRemove | `channel`, `key`, `idempotency_key` | same as map_publish (`idempotency`, `key_not_found`) | map_subscriptions.md |
| 17 | `/map_read_state` | MapReadState | `channel`, `cursor`, `limit` (-1 = all, 0 = position only), `key`, `revision_offset`, `revision_epoch` (epoch change gives 112) | `entries[]{key, data, tags, offset, removed, time}`, `offset`, `epoch`, `cursor` | map_subscriptions.md |
| 18 | `/map_read_stream` | MapReadStream | `channel`, `since_offset`, `since_epoch`, `limit`, `reverse` | `entries[]`, `offset`, `epoch` (gap gives 112) | map_subscriptions.md |
| 19 | `/map_stats` | MapStats | `channel` | `num_keys` | map_subscriptions.md |
| 20 | `/map_clear` | MapClear | `channel` | (not described) | map_subscriptions.md |
| 21 | `/shared_poll_publish` | SharedPollPublish | per shared_poll.md "Direct publish / API" (fields not fully tabulated here, OQ-02) | - | shared_poll.md (experimental). Needs `publish_enabled` |

(*) = required per the docs.

**In api.proto but NOT in the OSS HTTP routes or OSS executor (PRO surface):** Connections, UpdateUserStatus, GetUserStatus, DeleteUserStatus, BlockUser, UnblockUser, RevokeToken, InvalidateUserTokens, DeviceRegister, DeviceUpdate, DeviceRemove, DeviceList, DeviceTopicList, DeviceTopicUpdate, UserTopicList, UserTopicUpdate, SendPushNotification, UpdatePushStatus, CancelPush (OSS:internal/apiproto/api.proto:25-43; DOCS:docs/pro/server_api_enhancements.md, DOCS:docs/pro/connections.md). What OSS returns for these over gRPC is not documented (OQ-04).

**Legacy single-endpoint API** (`{id, method, params}`): `OSS:internal/api/handler_legacy.go` defines types for it, but no route registration was found in a grep of `internal/`. Treat it as not part of v6.10.3 (OQ-13).

### 2.2 Server API error codes and transport mapping

Source: DOCS:docs/server/server_api.md ("Error", "Transport error mode"); OSS:internal/apiproto/errors.go.

| Code | Message | HTTP (transport mode) | gRPC (transport mode) | temporary |
|---|---|---|---|---|
| 100 | internal server error | 500 | Internal | yes |
| 102 | unknown channel | 404 | NotFound | |
| 104 | not found | 404 | NotFound | |
| 107 | bad request | 400 | InvalidArgument | |
| 108 | not available | 400 | InvalidArgument | |
| 111 | too many requests | 429 (since 6.10.0) | ResourceExhausted | yes (since 6.10.0) |
| 112 | unrecoverable position | 416 | OutOfRange | |
| 113 | conflict | 409 | AlreadyExists | |
| other | - | 500 | Internal | |

Note: server API 104 = "not found" while client 104 = "method not found", and server API 113 = "conflict" while client 113 = "concurrent pagination". Same numbers, different messages (OQ-06).

### 2.3 Client protocol: commands, results, pushes

Source: PROTO:definitions/client.proto; DOCS:docs/transports/client_protocol.md; DOCS:docs/transports/uni_client_protocol.md.

**Command fields (field number):**
- `id` (1)
- `connect` (4) ConnectRequest
- `subscribe` (5)
- `unsubscribe` (6)
- `publish` (7)
- `presence` (8)
- `presence_stats` (9)
- `history` (10)
- `ping` (11, *not supported by the server*)
- `send` (12, no reply, id may be 0)
- `rpc` (13)
- `refresh` (14)
- `sub_refresh` (15)
- An empty Command is a pong.

**Reply fields:**
- `id` (1)
- `error` (2)
- `push` (4)
- results: `connect` (5), `subscribe` (6), `unsubscribe` (7), `publish` (8), `presence` (9), `presence_stats` (10), `history` (11), `ping` (12), `rpc` (13), `refresh` (14), `sub_refresh` (15)
- An empty Reply is a ping.

**Request and result messages:**

| Message | Fields |
|---|---|
| ConnectRequest | token 1, data 2, subs 3 (map<string,SubscribeRequest>), name 4, version 5, headers 6, flag 7 (PRO draft), dict 8 (PRO draft), profile 9 (PRO draft) |
| ConnectResult | client 1, version 2, expires 3, ttl 4, data 5, subs 6, ping 7, pong 8, session 9, node 10, time 11, flag 12, dict 13 |
| RefreshRequest | token 1 |
| RefreshResult | client 1 (unset), version 2 (unset), expires 3, ttl 4 |
| SubscribeRequest | channel 1, token 2, recover 3, epoch 6, offset 7, data 8, positioned 9, recoverable 10, join_leave 11, delta 12, tf 13 (FilterNode), flag 14, type 15 (0 stream, 1 map, 2 map_clients, 3 map_users, 4 shared_poll), phase 16, cursor 17, limit 18, asc 19 |
| SubscribeResult | expires 1, ttl 2, recoverable 3, epoch 6, publications 7, recovered 8, offset 9, positioned 10, data 11, was_recovering 12, delta 13, id 14, type 15, phase 16, cursor 17, state 18, publish_debounce 19 (PRO client publish debouncing, per the SDK matrix) |
| SubRefreshRequest | channel 1, token 2, type 3, track 4 (TrackBatch[]), untrack 5 |
| SubRefreshResult | expires 1, ttl 2, items 3 |
| UnsubscribeRequest / UnsubscribeResult | channel 1 / {} |
| PublishRequest / PublishResult | channel 1, data 2, type 3, key 4, removed 5 / {} |
| PresenceRequest / PresenceResult | channel / presence map<string,ClientInfo> |
| PresenceStatsRequest / PresenceStatsResult | channel / num_clients, num_users |
| HistoryRequest / HistoryResult | channel 1, limit 7, since 8 (StreamPosition{offset, epoch}), reverse 9 / publications, epoch, offset |
| RPCRequest / RPCResult | data 1, method 2 / data 1 |
| SendRequest | data 1 |
| EmulationRequest | node 1, session 2, data 3 |

**Shared types:**

| Type | Fields |
|---|---|
| ClientInfo | user, client, conn_info, chan_info |
| Publication | data 4, info 5, offset 6, tags 7, delta 8, time 9 (inter-node only), channel 10 (wildcard subs), key 11, removed 12, score 13, epoch 14, prev_data 15 (never sent to clients), version 16 |
| FilterNode | op, key, cmp, val, vals, nodes |

**Push types** (`Push{id 1 (numeric channel id), channel 2, ...}`):

| Push field | Meaning | Bidirectional | Unidirectional |
|---|---|---|---|
| `pub` (4) Publication | publication | yes | yes |
| `join` (5) {info} | client joined | yes | yes |
| `leave` (6) {info} | client left | yes | yes |
| `unsubscribe` (7) {code, reason} | server unsubscribed the client | yes | yes |
| `message` (8) {data} | async server-to-client message | yes | proto yes. Not listed in the uni doc list (OQ-14) |
| `subscribe` (9) {recoverable, epoch, offset, positioned, data} | server-side subscribe | yes | yes |
| `connect` (10) {client, version, data, subs, expires, ttl, ping, pong, session, node, time} | first push on uni transports | n/a | yes |
| `disconnect` (11) {code, reason, reconnect (deprecated, unset)} | server closes the connection | may be replaced by the WS close frame | yes (uni WS can disable it via `disable_disconnect_push`) |
| `refresh` (12) {expires, ttl} | connection refreshed server-side | yes | yes |

### 2.4 Error codes, unsubscribe codes and disconnect codes

Source: DOCS:docs/server/codes.md; PROTO:definitions/client.proto; DOCS:docs/transports/client_api.md; cross-checked [src] with LIB:errors.go and LIB:disconnect.go (same code set).

**Client error codes.** Range [0, 399] is reserved by Centrifuge and [400, 1999] is for applications. Client-side SDK errors use 0-100.

| Code | Message | Temporary | Meaning |
|---|---|---|---|
| 100 | internal server error | yes | server-side failure |
| 101 | unauthorized | | request unauthorized |
| 102 | unknown channel | | namespace not defined |
| 103 | permission denied | | access not allowed |
| 104 | method not found | | unknown command method |
| 105 | already subscribed | | duplicate subscription, client-side or server-side |
| 106 | limit exceeded | | some limit hit |
| 107 | bad request | | malformed; retrying does not help |
| 108 | not available | | feature not enabled (e.g. history off) |
| 109 | token expired | | SDK refreshes the token |
| 110 | expired | | connection expired (no token involved) |
| 111 | too many requests | yes | rate limited |
| 112 | unrecoverable position | | stream lacks the range / wrong epoch |
| 113 | concurrent pagination | yes | another pagination is in progress on the same stream |

**Unsubscribe codes.** Server range is [2000, 2999]. Codes >= 2500 mean resubscribe, codes < 2500 mean go to unsubscribed. [2000-2099] and [2500-2599] are reserved by Centrifuge. Client-side codes are < 2000.

| Code | Reason | Resubscribe | Note |
|---|---|---|---|
| 2000 | server unsubscribe | no | server API unsubscribe |
| 2500 | insufficient state | yes | message loss detected |
| 2501 | subscription expired | yes | refresh the sub token |
| 2502 | server tags filter changed (codes.md) / "state invalidated" (proto) | yes | client must drop its sub token and cached state. Currently sent for PRO map server tags filter changes (since v6.10.0) |

[src] LIB:unsubscribe.go also defines client-side codes `UnsubscribeCodeClient=0` and `UnsubscribeCodeDisconnect` (value not captured here).

**Disconnect codes.** Proto ranges:
- [0, 2999]: client and transport use
- [3000, 3499]: Centrifuge, reconnect
- [3500, 3999]: Centrifuge, terminal
- [4000, 4499]: app, reconnect
- [4500, 4999]: app, terminal
- >= 5000: Centrifuge, reconnect (proto only; conflicts with client_api.md, see OQ-03)

| Code | Reason | Reconnect | Meaning |
|---|---|---|---|
| 3000 | connection closed | (special) | closed without server advice |
| 3001 | shutdown | yes | node shutting down |
| 3004 | internal server error | yes | server error. Also the default uni connect-proxy error |
| 3005 | connection expired | yes | |
| 3006 | subscription expired | yes | |
| 3008 | slow | yes | client cannot read fast enough |
| 3009 | write error | yes | |
| 3010 | insufficient state | yes | bad stream position |
| 3011 | force reconnect | yes | |
| 3012 | no pong | yes | |
| 3013 | too many requests | yes | |
| 3014 | state invalidated | yes | client fetches a fresh token and resyncs |
| 3500 | invalid token | no | |
| 3501 | bad request | no | malformed frames, or a JSON command that fails the strict decode |
| 3502 | stale | no | not authenticated in time (`client.stale_close_delay`) |
| 3503 | force disconnect | no | |
| 3504 | connection limit | no | e.g. personal-channel `single_connection` |
| 3505 | channel limit | no | |
| 3506 | inappropriate protocol | no | e.g. binary data to a JSON client |
| 3507 | permission denied | no | |
| 3508 | not available | no | e.g. async message with no handler |
| 3509 | too many errors | no | |

Codes 3002, 3003 and 3007 are not defined in the docs or in LIB:disconnect.go.

The WebSocket close frame carries code and reason, and the reason must be under 127 bytes (proto). An oversized WS message is closed with a "message too big" close code (websocket.md). The numeric value is not stated (OQ-15).

### 2.5 Official client SDKs

centrifuge-js (browser, NodeJS, React Native), centrifuge-go, centrifuge-dart, centrifuge-swift, centrifuge-java, centrifuge-python, centrifuge-rust (Tokio and WASM; WebSocket and HTTP-streaming), centrifuge-csharp (.NET, MAUI, Blazor, Unity). Source: DOCS:docs/transports/client_sdk.md.

Format support:
- JSON default: js
- JSON and Protobuf: go, rust (Protobuf default)
- Protobuf only: dart, swift, java

The docs list community SDKs separately.

### 2.6 Console commands (exact list)

Source: DOCS:docs/server/console_commands.md; OSS:internal/cli/*.go. Root server command flags: OSS:internal/config/config.go:139-171 and OSS:internal/app/cmd.go.

| Command | Flags [src] | Doc |
|---|---|---|
| (root, run server) | `-c/--config`, `--watch-config-file`, `--pid_file`, `-a/--http_server.address`, `-p/--http_server.port` (8000), `--http_server.internal_address`, `--http_server.internal_port`, `--engine.type`, `--broker.enabled`, `--broker.type`, `--presence_manager.enabled`, `--presence_manager.type`, `--map_broker.type`, `--log.level`, `--log.file`, `--debug.enabled`, `--admin.enabled`, `--admin.external`, `--admin.insecure`, `--prometheus.enabled`, `--swagger.enabled`, `--health.enabled`, `--uni_websocket.enabled`, `--uni_sse.enabled`, `--uni_http_stream.enabled`, `--sse.enabled`, `--http_stream.enabled`, `--init.enabled`, `--client.insecure`, `--http_api.insecure`, `--http_api.external`, `--grpc_api.enabled`, `--grpc_api.port` (10000), `--uni_grpc.enabled`, `--uni_grpc.port` (11000) | configuration.md (says "see `centrifugo -h`") |
| `version` | - | yes |
| `genconfig` | `-c/--config` (format by extension) | yes |
| `checkconfig` | `-c/--config`, `-s/--strict` | yes (`--strict` is undocumented) |
| `defaultconfig` | `-c`, `-b/--base`, `-d/--dry-run` | yes |
| `defaultenv` | `-b/--base`, `--base-non-zero-only` | yes |
| `configdoc` | `-m/--markdown`, `-s/--section`, `-p/--port` (6060), `-l/--base-level` | yes |
| `gentoken` | `-c`, `-u/--user`, `-t/--ttl` (default 604800 s; -1 = no exp), `-q/--quiet` | yes (`-q` and `-1` are undocumented) |
| `gensubtoken` | `-c`, `-u`, `-s/--channel`, `-t`, `-q` | yes |
| `checktoken` | `-c` + positional TOKEN | yes |
| `checksubtoken` | `-c` + positional TOKEN | yes |
| `serve` | `-d/--dir`, `-p/--port` (3000), `-a/--address`: static file server (development only) | **not documented** |

### 2.7 Default endpoints

From DOCS:docs/server/configuration.md "Default endpoints / Customize handler prefixes", transport docs, and server_api.md:

| Endpoint | Path or port |
|---|---|
| WebSocket | `/connection/websocket` |
| HTTP streaming | `/connection/http_stream` |
| SSE | `/connection/sse` |
| Emulation | `/emulation` |
| WebTransport | `/connection/webtransport` |
| Unidirectional SSE | `/connection/uni_sse` |
| Unidirectional HTTP streaming | `/connection/uni_http_stream` |
| Unidirectional WebSocket | `/connection/uni_websocket` |
| Connection init | `/connection/init` |
| HTTP API | `/api` |
| Prometheus | `/metrics` |
| Health | `/health` |
| Swagger | `/swagger` |
| Debug | `/debug/pprof` |
| Admin | `/` (plus `/admin/auth`, `/admin/api`) |
| gRPC API | port 10000 |
| uni gRPC | port 11000 |
| HTTP server | port 8000 |

---

## 3. Namespace / channel option keys (exact names)

Primary source: OSS:internal/configtypes/namespace.go (struct `ChannelOptions`, `MapConfig`, `SharedPollConfig`). Cross-checked with DOCS:docs/server/channels.md, DOCS:docs/server/map_subscriptions.md and DOCS:docs/server/shared_poll.md. These keys are valid in `channel.without_namespace` and in each `channel.namespaces[]` entry. The namespace entry also takes `name`.

**OSS `ChannelOptions` (in struct order):**

| Group | Keys |
|---|---|
| Presence | `presence`, `join_leave`, `force_push_join_leave`, `map_clients_presence_channel_prefix`, `map_users_presence_channel_prefix` |
| History | `history_size`, `history_ttl`, `history_meta_ttl`, `history_max_publication_limit` (v6.10.3), `recovery_max_publication_limit` (v6.10.3) |
| Positioning / recovery | `force_positioning`, `allow_positioning`, `force_recovery`, `allow_recovery`, `force_recovery_mode` (`stream`/`cache`), `auto_cache_recover` (v6.8.3) |
| Delta / filtering | `allowed_delta_types`, `allow_tags_filter` (v6.4.0), `delta_publish` |
| Subscribe permissions | `allow_subscribe_for_anonymous`, `allow_subscribe_for_client` |
| Publish permissions | `allow_publish_for_anonymous`, `allow_publish_for_subscriber`, `allow_publish_for_client` |
| Presence permissions | `allow_presence_for_anonymous`, `allow_presence_for_subscriber`, `allow_presence_for_client` |
| History permissions | `allow_history_for_anonymous`, `allow_history_for_subscriber`, `allow_history_for_client` |
| Channel rules | `allow_user_limited_channels`, `channel_regex` (namespace prefix excluded; checked for client-side subs only), `publication_data_format` (v6.5.2) |
| Proxies | `subscribe_proxy_enabled`, `subscribe_proxy_name`, `publish_proxy_enabled`, `publish_proxy_name`, `sub_refresh_proxy_enabled`, `sub_refresh_proxy_name`, `subscribe_stream_proxy_enabled`, `subscribe_stream_proxy_name`, `subscribe_stream_proxy_bidirectional` |
| Subscription type | `subscription_type` (`stream` default, `map`, `map_clients`, `map_users`, `shared_poll`), `map` (object), `shared_poll` (object) |

**`map` object keys:** `mode`, `key_ttl`, `stream_size`, `stream_ttl`, `meta_ttl`, `remove_client_on_unsubscribe`, `client_key` (`""` / `client_id` / `user_id`), `allow_publish_for_client`, `allow_publish_for_subscriber`, `allow_publish_for_anonymous`, `allow_remove_for_client`, `allow_remove_for_subscriber`, `allow_remove_for_anonymous`, `publish_proxy_enabled`, `publish_proxy_name`, `remove_proxy_enabled`, `remove_proxy_name`, `default_page_size` (100), `min_page_size` (100), `max_page_size` (1000), `live_transition_max_publication_limit`, `subscribe_catch_up_timeout` (5s).

**`shared_poll` object keys (namespace):** `proxy_name`, `refresh_interval` (10s), `refresh_batch_size` (1000), `max_keys_per_connection` (5000), `mode` (`versionless`/`versioned`), `channel_shutdown_delay` (1s), `track_expired_extra_delay` (25s), `publish_enabled` (false).

**Documented in channels.md but PRO only (absent from the OSS struct):** `cache_empty_proxy_enabled`, `cache_empty_proxy_name`, `state_proxy_enabled`, `state_events`, `shared_position_sync`, `subscribe_cel`, `publish_cel`, `history_cel`, `presence_cel`, `batch_max_size`, `batch_max_delay`, `batch_flush_latest`, `allow_channel_compaction`. channels.md does not mark the three `batch_*` keys as "PRO only" in their headings, but they link to PRO docs and are not in the OSS struct.

**`channel.*` top-level keys (OSS struct):** `proxy` (subscribe, publish, sub_refresh, subscribe_stream, shared_poll_refresh; `state` and `cache_empty` are PRO), `without_namespace`, `namespaces`, `history_meta_ttl` (720h), `publication_data_format`, `max_length` (255), `private_prefix` (`$`), `namespace_boundary`, `user_boundary`, `user_separator` (the last three are undocumented, OQ-16).

**RPC namespace keys:** `name`, `proxy_enabled`, `proxy_name` (OSS:internal/configtypes/rpc_namespace.go).

---

## 4. Gaps: documented OSS features missing from the SPEC OSS list

The SPEC list was supplied by the caller. Items marked (exp) are experimental.

1. **Delta compression** (Fossil; `allowed_delta_types`, `delta_publish`, publish `delta`). OSS-SUB-09
2. **Publication filtering by tags** (`allow_tags_filter`, `tf` FilterNode, publish `tags`). OSS-SUB-10
3. **Server API methods beyond publish:** `broadcast`, `subscribe`/`unsubscribe`/`disconnect`/`refresh` (user, client and session targeting), `history_remove`, `channels`, `info`, `batch` (with `parallel`), undocumented `rpc`, the `map_*` family (6 methods), and `shared_poll_publish`
4. **Transport error mode** (HTTP/gRPC status mapping) and `Error.temporary` in API responses (v6.10.0)
5. **Idempotent publish** (`idempotency_key`) and **versioned publish** (`version`, `version_epoch`, v6.2.0)
6. **Unidirectional WebSocket, unidirectional SSE and unidirectional HTTP-streaming** as separate transports (SPEC lists only generic "WebSocket/SSE/HTTP streaming/gRPC")
7. **gRPC: OSS has only *unidirectional* gRPC streaming plus the gRPC server API.** There is no bidirectional gRPC client transport in OSS docs. SPEC's "gRPC transport" needs clarifying
8. **Emulation endpoint** `/emulation` and `EmulationRequest` (part of bidirectional emulation, but it is a separate endpoint and config)
9. **WebSocket permessage-deflate compression** and the decompression-bomb limit (`decompressed_message_size_limit`, v6.8.4)
10. **WebSocket over HTTP/2 (RFC 8441)** (exp, v6.5.0), **H2C** (`h2c_external`), **HTTP/3** (exp)
11. **`cf_ws_frame_ping_pong` debug mode**
12. **Uni WS options** `join_push_messages`, `disable_closing_handshake`, `disable_disconnect_push`; uni WS `cf_connect` URL param (v6.3.0)
13. **Unidirectional connect error mapping**: `client.connect_code_to_unidirectional_disconnect`, `uni_sse|uni_http_stream.connect_code_to_http_response`, proxy `status_to_code_transforms`
14. **Connection init endpoint** `/connection/init` (v6.5.0)
15. **Headers emulation** (`ConnectRequest.headers` to proxy headers; v6)
16. **User-limited channels (`#`)** and **channel_regex** (SPEC has only "private channels")
17. **Positioning** (`force_positioning` / `allow_positioning`) as a feature separate from recovery; `recovery_max_publication_limit` / `history_max_publication_limit` per namespace (v6.10.3)
18. **Automatic personal channel subscription** and **single connection per user**
19. **`publication_data_format`** validation (v6.5.2)
20. **Sub refresh proxy** (`sub_refresh_proxy_*`); **map publish/remove proxy** (exp); **shared poll refresh proxy** (exp); **subscribe stream proxy, bidirectional** (exp)
21. **Map presence types** `map_clients` / `map_users` and the presence channel prefixes (exp)
22. **RPC namespaces** (`rpc.namespaces`, `rpc.namespace_boundary`) and the built-in **RPC ping extension** (`rpc.ping`)
23. **Async `send` command** (client to server message without a reply)
24. **SDK `getState` callback** pattern (v6.8.0) and the SDK spec (states, backoff), which together define what client behaviour the server must support
25. **Config features:** `.env` file, `CENTRIFUGO_VAR_` extrapolation, MapStringString/StringKeyValues env encodings, SIGHUP partial reload, `--watch-config-file` (v6.10.3), `shutdown.timeout`, `pid_file`, unknown-key warnings
26. **CLI commands beyond genconfig/checkconfig/gentoken:** `version`, `defaultconfig`, `defaultenv`, `configdoc`, `gensubtoken`, `checktoken`, `checksubtoken`, and undocumented `serve`
27. **Endpoint prefix customisation and internal port** (`http_server.internal_*`, `*.handler_prefix`, `*.external`)
28. **Swagger UI**, **debug pprof**, **Graphite metrics**, **OpenTelemetry** config section, **anonymous usage stats** (`usage_stats.disabled`)
29. **Auth conveniences:** `client.user_id_http_header`, `allow_anonymous_connect_without_token`, `disallow_anonymous_connection_tokens`, `client.insecure`, `insecure_skip_token_signature_verify`, `http_api.insecure`, `admin.insecure`
30. **Connection limits:** `connection_limit`, `connection_rate_limit` (503), `user_connection_limit`, `channel_limit`, `client.concurrency`, `queue_max_size`
31. **Engines-adjacent OSS items missing from SPEC:** PostgreSQL **controller** (exp), Redis **IAM auth** (exp, v6.10.0), NATS **raw mode**, separate `broker` / `presence_manager` / `map_broker` sections. Cover them in the engines research
32. **Server SDKs** (gocent etc.) and **Buf schema registry** publication of apiproto and unistream (per-release labels since v6.10.0)

---

## 5. Open questions (do not guess; resolve by black-box tests against v6.10.3 or by a decision)

| ID | Question | Evidence |
|---|---|---|
| OQ-01 | Server API `/api/rpc` (gRPC `RPC`) exists in OSS code but is undocumented. Is it part of the public compatibility surface? Which extensions are registered in OSS? `rpc.ping` is registered on the *client* handler (OSS:internal/app/app.go:555), and it is unclear whether the API executor has any extensions in OSS | OSS:internal/api/handler.go:54, OSS:internal/api/api.go:57,854-868 |
| OQ-02 | Exact request/response fields of `shared_poll_publish`, `map_clear` result, and per-field required flags for the `map_*` methods (docs list "options" informally) | DOCS:docs/server/shared_poll.md §Direct publish/API; DOCS:docs/server/map_subscriptions.md §Server API. Resolve from OSS:internal/apiproto/api.proto |
| OQ-03 | Disconnect code ranges conflict. The proto says >=5000 is reserved by Centrifuge with reconnect; client_api.md says custom codes must be in [3000, 4999] and anything other than 3000-3499 / 4000-4499 goes to disconnected | PROTO:definitions/client.proto (Disconnect comment) vs DOCS:docs/transports/client_api.md §Disconnect codes |
| OQ-04 | What does OSS return for PRO-only gRPC methods in the shared `CentrifugoApi` service (Unimplemented? 104?), and over HTTP (404)? | OSS:internal/apiproto/api.proto:25-43; OSS:internal/api/handler.go |
| OQ-05 | Publication tags filter SDK support conflict. client_sdk.md matrix: all SDKs. publication_filtering.md: "only centrifuge-js". Same kind of conflict for bidirectional emulation (http_stream.md says "only JS", the matrix also has rust and c#) | DOCS:docs/transports/client_sdk.md vs DOCS:docs/server/publication_filtering.md, DOCS:docs/transports/http_stream.md |
| OQ-06 | Code 104/113 messages differ between the client protocol ("method not found" / "concurrent pagination") and the server API ("not found" / "conflict"). Confirm this is intentional and keep two code tables | DOCS:docs/server/codes.md vs DOCS:docs/server/server_api.md; OSS:internal/apiproto/errors.go |
| OQ-07 | OSS behaviour of `SubscribeRequest.flag` bit0 (numeric channel id, PRO compaction?) and bit1 (reject unrecovered) | PROTO:definitions/client.proto only |
| OQ-08 | Does v6.10.3 still accept or require `cf_protocol_version` URL param, or the `format` and `cf_protocol` query params? Only source code and v4 migration docs mention them | DOCS:docs/getting-started/migration-v4.md; [src] LIB:handler_websocket.go:191 |
| OQ-09 | `http_server.external_http2_extended_connect`: purpose and relation to `GODEBUG=http2xconnect=1` (undocumented) | [src] OSS:internal/configtypes/types.go |
| OQ-10 | `.env` file: lookup path, precedence relative to real env vars | DOCS:blog/2025-01-16-centrifugo-v6-released.md (one bullet only) |
| OQ-11 | `enable_unreleased_features` top-level key: semantics | [src] OSS:internal/config/config.go |
| OQ-12 | Undocumented `client.*` keys: `expired_close_delay`, `expired_sub_close_delay`, `presence_update_interval`, `channel_position_check_delay`, `channel_position_max_time_lag`. Defaults and semantics (use `defaultconfig`/`configdoc` output from a reference binary) | [src] OSS:internal/configtypes/types.go |
| OQ-13 | Legacy `{method, params}` API types in `handler_legacy.go`: are they reachable in v6.10.3? | OSS:internal/api/handler_legacy.go |
| OQ-14 | Do uni transports deliver `message` pushes (from the client-message server API)? The uni doc list omits `message` | DOCS:docs/transports/uni_client_protocol.md vs PROTO:definitions/client.proto |
| OQ-15 | Numeric WS close code for "message too big" (presumably the RFC 6455 value; not stated) | DOCS:docs/transports/websocket.md |
| OQ-16 | `channel.namespace_boundary`, `user_boundary`, `user_separator`: are they configurable in OSS v6 or internal? configuration.md mentions `channel.namespace_boundary` only by analogy in `rpc.namespace_boundary` | DOCS:docs/server/configuration.md §rpc.namespace_boundary; [src] OSS:internal/configtypes/types.go |
| OQ-17 | `uni_grpc` "Supported data formats: JSON and binary". How does a uni gRPC client select JSON? | DOCS:docs/transports/uni_grpc.md; OSS:internal/unigrpc/unistream/unistream.proto (not read in depth) |
| OQ-18 | The `c-lib` clone HEAD (`8888c6e`) is newer than the centrifuge version pinned by v6.10.3 (`ae446d43faf0`). Re-verify the [src] claims taken from LIB against the pinned commit | OSS:go.mod:18 |
| OQ-19 | Delta compression minimum version is not stated in the current docs (introduced in v5.x per the blog). Confirm for the matrix | DOCS:docs/server/delta_compression.md; DOCS:blog/2024-05-30-real-time-data-compression-experiments.md |
| OQ-20 | The SPEC item "gRPC transport": does it mean only OSS `uni_grpc`, or is a bidirectional gRPC client transport expected (none is documented in OSS)? Needs a scope decision | DOCS:docs/transports/overview.md |
| OQ-21 | Unsubscribe code 2502 reason string differs: codes.md says "server tags filter changed", proto comment says "state invalidated". Which string goes on the wire? | DOCS:docs/server/codes.md vs PROTO:definitions/client.proto |
| OQ-22 | The admin UI is an embedded SPA from the separate `centrifugal/web` repo. A clean-room build needs its own UI. Confirm the admin API contract (`/admin/auth`, `/admin/api`) is in compatibility scope | DOCS:docs/server/admin_web.md; DOCS:docs/server/configuration.md §Custom internal port |
