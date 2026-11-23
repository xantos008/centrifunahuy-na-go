# Source register

Initial source access was on **2026-11-15**. Later reference observations are listed below. Version pins are in [baseline.md](../compatibility/baseline.md). Hierarchy follows CLAUDE.md: official docs and release notes come first, then official repositories, then provider docs, then black-box experiments. Secondary sources are leads only.

## 1. Repositories (read-only clones; nothing executed)

| Source | URL | Version / commit | License | Used for |
|---|---|---|---|---|
| Docs site | https://github.com/centrifugal/centrifugal.dev | `69da71466274f741bfe924bbf4a62a9124ec5776` (2026-11-13) | Apache-2.0 | All OSS/PRO feature semantics (`docs/`), blog posts (`blog/`), /pro comparison table (`src/pages/components/ComparisonTable.jsx`), sidebar status (`sidebars.js`) |
| Centrifugo OSS | https://github.com/centrifugal/centrifugo | tag `v6.10.3`, `4b6d86e23452a6fa1bd424ffe4526862efae85b8` | Apache-2.0 | Config keys/defaults (`internal/configtypes`), server API schema (`internal/apiproto/api.proto`), proxy schema (`internal/proxyproto/proxy.proto`), uni gRPC schema (`internal/unigrpc/unistream`), CLI. Cited as `[src]` = observed implementation, not contract |
| Client protocol | https://github.com/centrifugal/protocol | v0.23.1 (pinned by OSS go.mod); HEAD read `4455d8ea79cfd62206cc4ee6f32025da63fa1654` | MIT | `definitions/client.proto`: Command/Reply/Push wire schema |
| centrifuge library | https://github.com/centrifugal/centrifuge | OSS pins `v0.40.2-0.20261031074606-ae446d43faf0`; HEAD read `8888c6eee1c4e3a390d2fac511ccee19456f4eb4` (newer, see CORE-OQ-18) | MIT | Cross-check of codes, transports, engine capabilities (`[src]` only) |
| Centrifugo PRO releases | https://github.com/centrifugal/centrifugo-pro | README/SECURITY only, `0b35c13595ac6a77c2aa8a7abda5fdbab9ce9abc`; **no binaries downloaded** | proprietary product, public repo has no code | Versioning policy |

| centrifuge library (pinned) | module proxy | `v0.40.2-0.20261031074606-ae446d43faf0` (downloaded 2026-11-15) | MIT | Engine interfaces and extension points (overview §7, ADR-0029), re-verified at the exact pinned version |
| Admin web UI | https://github.com/centrifugal/web | `main` (embedded in OSS as `internal/webui`) | MIT (GitHub API) | Provenance PRV-04 only. Removed from our build; never copied |
| Swagger UI | https://github.com/swagger-api/swagger-ui | `v4.18.1` (per `misc/scripts/update_swagger_web.sh`) | Apache-2.0 | Provenance PRV-05 |
| Reference image | docker.io/centrifugo/centrifugo | `v6.10.3`, digest `sha256:33fdbbce7d195dde316cbde17bc1d2a17b9e345de82799d1ec3c2bb8916e3803` (verified 2026-11-15) | Apache-2.0 | Compatibility suite `ref` target (ADR-0028) |
| Go SDK | https://github.com/centrifugal/centrifuge-go | `v0.13.0` (2026-10-21) | MIT | Compatibility suite client |

## 2. Release notes

| Source | URL | Range read | Used for |
|---|---|---|---|
| OSS releases | https://github.com/centrifugal/centrifugo/releases | v6.0.0 … v6.10.3 (via `gh release view`) | Minimum versions, behavior changes (e.g. strict JSON 6.10.0, per-ns limits 6.10.3) |
| PRO releases | https://github.com/centrifugal/centrifugo-pro/releases | v6.0.0 … v6.10.3 | PRO min versions, status changes, conflicts (PRO-OQ-02/04) |
| Release-note pages in the docs repo | `release_notes/v6.8.0_oss_day1.md`, `v6.8.0_pro_day1.md` | — | v6.8.0 feature set |

## 3. Documentation pages → feature IDs

The URL uses the page front-matter `id`. Each ID points to a row in [feature-matrix.md](../compatibility/feature-matrix.md); detail lives in the annexes.

| URL | Repo path | Requirements supported (feature IDs) |
|---|---|---|
| https://centrifugal.dev/docs/getting-started/design | `docs/getting-started/design.md` | delivery/ordering/presence model (architecture overview §3, ADR-0008, ADR-0019) |
| https://centrifugal.dev/docs/getting-started/ecosystem | `docs/getting-started/ecosystem.md` | OSS-OPS-13 |
| https://centrifugal.dev/docs/getting-started/highlights | `docs/getting-started/highlights.md` | (read; context only) |
| https://centrifugal.dev/docs/getting-started/installation | `docs/getting-started/installation.md` | OSS-OPS-01, OSS-OPS-02, OSS-OPS-03, OSS-OPS-04, OSS-OPS-05, OSS-OPS-06, OSS-OPS-07 |
| https://centrifugal.dev/docs/getting-started/migration_v4 | `docs/getting-started/migration-v4.md` | OSS-PROTO-16 |
| https://centrifugal.dev/docs/getting-started/migration_v6 | `docs/getting-started/migration-v6.md` | OSS-ENG-00, OSS-TRANS-08 |
| https://centrifugal.dev/docs/pro/access_revoke | `docs/pro/access_revoke.md` | PRO-15.01, PRO-15.02, PRO-15.03 |
| https://centrifugal.dev/docs/pro/admin_idp_auth | `docs/pro/admin_idp_auth.md` | orphan duplicate page (not in sidebar); superseded by the sidebar page, see pro-feature-inventory §0 |
| https://centrifugal.dev/docs/pro/admin_ui | `docs/pro/admin_ui.md` | PRO-05.01, PRO-05.02, PRO-05.03, PRO-06.01, PRO-06.02, PRO-07.01, PRO-08.01, PRO-09.05, PRO-37.01, PRO-37.02, PRO-37.03 |
| https://centrifugal.dev/docs/pro/analytics | `docs/pro/analytics.md` | PRO-03.01, PRO-03.02, PRO-03.03, PRO-03.04, PRO-03.05, PRO-03.06, PRO-03.07, PRO-03.08, PRO-03.09, PRO-03.10 |
| https://centrifugal.dev/docs/pro/bandwidth_optimizations | `docs/pro/bandwidth_optimizations.md` | OSS-PROTO-13, PRO-23.01, PRO-23.02, PRO-23.03, PRO-23.04 |
| https://centrifugal.dev/docs/pro/capabilities | `docs/pro/capabilities.md` | PRO-17.01, PRO-17.02 |
| https://centrifugal.dev/docs/pro/cel_expressions | `docs/pro/cel_expressions.md` | PRO-19.01 |
| https://centrifugal.dev/docs/pro/channel_cache_empty | `docs/pro/channel_cache_empty.md` | orphan duplicate page (not in sidebar); superseded by the sidebar page, see pro-feature-inventory §0 |
| https://centrifugal.dev/docs/pro/channel_state_events | `docs/pro/channel_events.md` | orphan duplicate page (not in sidebar); superseded by the sidebar page, see pro-feature-inventory §0 |
| https://centrifugal.dev/docs/pro/channel_patterns | `docs/pro/channel_patterns.md` | PRO-18.01 |
| https://centrifugal.dev/docs/pro/client_authentication | `docs/pro/client_authentication.md` | PRO-10.01, PRO-10.02, PRO-11.01, PRO-11.02, PRO-11.03, PRO-11.04 |
| https://centrifugal.dev/docs/pro/client_message_batching | `docs/pro/client_msg_batching.md` | PRO-24.01 … PRO-24.06 |
| https://centrifugal.dev/docs/pro/connections | `docs/pro/connections.md` | PRO-14.01 |
| https://centrifugal.dev/docs/pro/delta_at_most_once | `docs/pro/delta_at_most_once.md` | orphan duplicate page (not in sidebar); superseded by the sidebar page, see pro-feature-inventory §0 |
| (not published: `draft: true`) | `docs/pro/dictionary_compression.md` | OSS-PROTO-14, PRO-30 (excluded) |
| https://centrifugal.dev/docs/pro/client_publications | `docs/pro/draft_client_publications.md` | PRO-29 (unlisted; excluded) |
| https://centrifugal.dev/docs/pro/event_hooks | `docs/pro/event_hooks.md` | PRO-16.01, PRO-16.02 |
| https://centrifugal.dev/docs/pro/install_and_run | `docs/pro/install_and_run.md` | PRO-36, PRO-42 |
| https://centrifugal.dev/docs/pro/map_subscriptions | `docs/pro/map_subscriptions.md` | PRO-26.01, PRO-26.02, PRO-26.03, PRO-26.04, PRO-26.05, PRO-26.06 |
| https://centrifugal.dev/docs/pro/namespace_engines | `docs/pro/namespace_engines.md` | orphan duplicate page (not in sidebar); superseded by the sidebar page, see pro-feature-inventory §0 |
| https://centrifugal.dev/docs/pro/observability_enhancements | `docs/pro/observability_enhancements.md` | PRO-09.01, PRO-09.02, PRO-09.03, PRO-09.04, PRO-09.06, PRO-09.07, PRO-09.08, PRO-09.09, PRO-09.10, PRO-09.11, PRO-35 |
| https://centrifugal.dev/docs/pro/overview | `docs/pro/overview.md` | PRO feature list, sandbox limits (PRO-36, not applicable) |
| https://centrifugal.dev/docs/pro/performance | `docs/pro/performance.md` | PRO-20.01, PRO-20.02, PRO-32.01, PRO-32.02, PRO-33.01, PRO-33.02, PRO-33.03, PRO-34.01, PRO-34.02, PRO-34.03, PRO-34.04, PRO-34.05 |
| https://centrifugal.dev/docs/pro/process_stats | `docs/pro/process_stats.md` | PRO-09.05 (orphan page) |
| https://centrifugal.dev/docs/pro/push_notifications | `docs/pro/push_notifications.md` | PRO-01.01, PRO-01.02, PRO-01.03, PRO-01.04, PRO-01.05, PRO-01.06, PRO-01.07, PRO-01.08, PRO-01.09, PRO-01.10, PRO-01.11, PRO-01.12, PRO-01.13, PRO-01.14 … (+9) |
| https://centrifugal.dev/docs/pro/rate_limiting | `docs/pro/rate_limiting.md` | PRO-04.01, PRO-04.02, PRO-04.03, PRO-04.04, PRO-04.05, PRO-04.06, PRO-04.07, PRO-04.08, PRO-04.09 |
| https://centrifugal.dev/docs/pro/scalability | `docs/pro/scalability.md` | OSS-ENG-06, OSS-ENG-19, PRO-21.01, PRO-21.02, PRO-22.01, PRO-40.01, PRO-40.02 |
| https://centrifugal.dev/docs/pro/server_api_enhancements | `docs/pro/server_api_enhancements.md` | PRO-12.01, PRO-12.02, PRO-31 |
| https://centrifugal.dev/docs/pro/server_tags_filter | `docs/pro/server_tags_filter.md` | PRO-25.01 |
| https://centrifugal.dev/docs/pro/shared_poll | `docs/pro/shared_poll.md` | PRO-27.01, PRO-27.02, PRO-27.03, PRO-27.04, PRO-27.05 |
| https://centrifugal.dev/docs/pro/tracing | `docs/pro/tracing.md` | PRO-02.01, PRO-02.02, PRO-02.03, PRO-02.04, PRO-02.05 |
| https://centrifugal.dev/docs/pro/user_status | `docs/pro/user_status.md` | PRO-13.01, PRO-13.02, PRO-13.03 |
| https://centrifugal.dev/docs/server/admin_web | `docs/server/admin_web.md` | OSS-ADM-01 |
| https://centrifugal.dev/docs/server/authentication | `docs/server/authentication.md` | OSS-AUTH-01, OSS-AUTH-02, OSS-AUTH-03, OSS-AUTH-04, OSS-AUTH-05, OSS-AUTH-06, OSS-AUTH-07, OSS-AUTH-08 |
| https://centrifugal.dev/docs/server/cache_recovery | `docs/server/cache_recovery.md` | OSS-HIST-04, OSS-HIST-05, OSS-SUB-08 |
| https://centrifugal.dev/docs/server/channel_permissions | `docs/server/channel_permissions.md` | OSS-AUTH-12, OSS-AUTH-13, OSS-AUTH-14, OSS-AUTH-15, OSS-CH-04, OSS-HIST-12, OSS-PERM-01, OSS-PERM-02, OSS-PRES-06 |
| https://centrifugal.dev/docs/server/channel_token_auth | `docs/server/channel_token_auth.md` | OSS-AUTH-09, OSS-AUTH-10, OSS-AUTH-16 |
| https://centrifugal.dev/docs/server/channels | `docs/server/channels.md` | OSS-CH-01, OSS-CH-02, OSS-CH-03, OSS-CH-04, OSS-CH-05, OSS-CH-06, OSS-HIST-01, OSS-HIST-02, OSS-HIST-03, OSS-HIST-05, OSS-HIST-06, OSS-HIST-07, OSS-HIST-08, OSS-PERM-02 … (+3) |
| https://centrifugal.dev/docs/server/codes | `docs/server/codes.md` | OSS-PROTO-08, OSS-PROTO-09 |
| https://centrifugal.dev/docs/server/configuration | `docs/server/configuration.md` | OSS-ADM-01, OSS-API-05, OSS-AUTH-00, OSS-AUTH-11, OSS-CFG-01, OSS-CFG-02, OSS-CFG-03, OSS-CFG-05, OSS-CFG-06, OSS-CFG-07, OSS-CFG-08, OSS-CFG-09, OSS-CFG-10, OSS-CFG-11 … (+20) |
| https://centrifugal.dev/docs/server/console_commands | `docs/server/console_commands.md` | OSS-CLI-01 |
| https://centrifugal.dev/docs/server/consumers | `docs/server/consumers.md` | OSS-CONS-01, OSS-CONS-02, OSS-CONS-03, OSS-CONS-04, OSS-CONS-05, OSS-CONS-06, OSS-CONS-07, OSS-CONS-08, OSS-CONS-09, OSS-CONS-10, OSS-CONS-11 |
| https://centrifugal.dev/docs/server/delta_compression | `docs/server/delta_compression.md` | OSS-SUB-09 |
| https://centrifugal.dev/docs/server/engines | `docs/server/engines.md` | OSS-ENG-00, OSS-ENG-01, OSS-ENG-02, OSS-ENG-03, OSS-ENG-04, OSS-ENG-05, OSS-ENG-06, OSS-ENG-07, OSS-ENG-08, OSS-ENG-09, OSS-ENG-10, OSS-ENG-11, OSS-ENG-12, OSS-ENG-13 … (+5) |
| https://centrifugal.dev/docs/server/history_and_recovery | `docs/server/history_and_recovery.md` | OSS-HIST-01, OSS-HIST-03, OSS-HIST-06, OSS-HIST-09, OSS-HIST-10, OSS-PROTO-15 |
| https://centrifugal.dev/docs/server/infra_tuning | `docs/server/infra_tuning.md` | OSS-OPS-10 |
| https://centrifugal.dev/docs/server/load_balancing | `docs/server/load_balancing.md` | OSS-OPS-09 |
| https://centrifugal.dev/docs/server/map_subscriptions | `docs/server/map_subscriptions.md` | OSS-API-01, OSS-SUB-04, OSS-SUB-05 |
| https://centrifugal.dev/docs/server/monitoring | `docs/server/monitoring.md` | OSS-OBS-01, OSS-OBS-03 |
| https://centrifugal.dev/docs/server/observability | `docs/server/observability.md` | OSS-OBS-01, OSS-OBS-02, OSS-OBS-04, OSS-OBS-05, OSS-OBS-07, OSS-OBS-08, PRO-09.02 |
| https://centrifugal.dev/docs/server/presence | `docs/server/presence.md` | OSS-PRES-01, OSS-PRES-02, OSS-PRES-03, OSS-PRES-04, OSS-PRES-07 |
| https://centrifugal.dev/docs/server/proxy | `docs/server/proxy.md` | OSS-PROTO-11, OSS-PROXY-01, OSS-PROXY-02, OSS-PROXY-03, OSS-PROXY-04, OSS-PROXY-05, OSS-PROXY-06, OSS-PROXY-08, OSS-PROXY-09, OSS-PROXY-10, OSS-PROXY-11, OSS-PROXY-12, OSS-PROXY-13, OSS-PROXY-14 … (+9) |
| https://centrifugal.dev/docs/server/proxy_streams | `docs/server/proxy_streams.md` | OSS-PROXY-07, OSS-SUB-07 |
| https://centrifugal.dev/docs/server/publication_filtering | `docs/server/publication_filtering.md` | OSS-SUB-10 |
| https://centrifugal.dev/docs/server/server_api | `docs/server/server_api.md` | OSS-API-01, OSS-API-02, OSS-API-03, OSS-API-04, OSS-CFG-14, OSS-SDK-04, OSS-SUB-14 |
| https://centrifugal.dev/docs/server/server_sdks | `docs/server/server_sdks.md` | OSS-OPS-12, OSS-SDK-04 |
| https://centrifugal.dev/docs/server/server_subs | `docs/server/server_subs.md` | OSS-SUB-02, OSS-SUB-03 |
| https://centrifugal.dev/docs/server/shared_poll | `docs/server/shared_poll.md` | OSS-SUB-06 |
| https://centrifugal.dev/docs/server/tls | `docs/server/tls.md` | OSS-OPS-08, OSS-TLS-01, OSS-TLS-02 |
| https://centrifugal.dev/docs/transports/client_api | `docs/transports/client_api.md` | OSS-PROTO-09, OSS-PROTO-15, OSS-SDK-01, OSS-SDK-03, OSS-SUB-01, OSS-SUB-02 |
| https://centrifugal.dev/docs/transports/client_protocol | `docs/transports/client_protocol.md` | OSS-PROTO-01, OSS-PROTO-02, OSS-PROTO-04, OSS-PROTO-05, OSS-PROTO-06, OSS-PROTO-08, OSS-PROTO-10, OSS-PROTO-15, OSS-SUB-13 |
| https://centrifugal.dev/docs/transports/client_sdk | `docs/transports/client_sdk.md` | OSS-PROTO-03, OSS-SDK-02 |
| https://centrifugal.dev/docs/transports/http_stream | `docs/transports/http_stream.md` | OSS-TRANS-03 |
| https://centrifugal.dev/docs/transports/overview | `docs/transports/overview.md` | OSS-PROTO-06 |
| https://centrifugal.dev/docs/transports/sockjs | `docs/transports/sockjs.md` | OSS-TRANS-08 |
| https://centrifugal.dev/docs/transports/sse | `docs/transports/sse.md` | OSS-TRANS-04 |
| https://centrifugal.dev/docs/transports/uni_client_protocol | `docs/transports/uni_client_protocol.md` | OSS-PROTO-11, OSS-TRANS-09 |
| https://centrifugal.dev/docs/transports/uni_grpc | `docs/transports/uni_grpc.md` | OSS-TRANS-13 |
| https://centrifugal.dev/docs/transports/uni_http_stream | `docs/transports/uni_http_stream.md` | OSS-TRANS-11 |
| https://centrifugal.dev/docs/transports/uni_sse | `docs/transports/uni_sse.md` | OSS-TRANS-12 |
| https://centrifugal.dev/docs/transports/uni_websocket | `docs/transports/uni_websocket.md` | OSS-TRANS-02, OSS-TRANS-10 |
| https://centrifugal.dev/docs/transports/websocket | `docs/transports/websocket.md` | OSS-PROTO-03, OSS-PROTO-07, OSS-TRANS-01, OSS-TRANS-02 |
| https://centrifugal.dev/docs/transports/webtransport | `docs/transports/webtransport.md` | OSS-TRANS-06 |

Blog posts cited by the annexes (e.g. `blog/2025-01-16-centrifugo-v6-released.md`, `blog/2026-07-27-app-owned-state-stream-subscriptions.md`, `blog/2026-08-17-tuning-idle-connections`) are treated as secondary to docs. Where they conflict with the docs, the conflict is recorded (PRO-OQ-08).

## 4. Third-party / provider specifications (to be read in the phase that needs them)

| Topic | Authoritative source | Needed in phase |
|---|---|---|
| WebSocket | RFC 6455, RFC 7692 (permessage-deflate), RFC 8441 (WS over HTTP/2) | 1, 3 |
| JWT / JWK / JWKS | RFC 7519, RFC 7515, RFC 7517, RFC 7518, RFC 8037 (EdDSA) | 1, 4 |
| OIDC / OAuth | OpenID Connect Core 1.0, Discovery 1.0, RFC 7636 (PKCE), RFC 9700 (OAuth security BCP) | 5 |
| CEL | https://github.com/google/cel-spec (language), cel-go docs (cost limits) | 4 |
| SSE | WHATWG HTML Living Standard §Server-sent events | 3 |
| HTTP/3 / WebTransport | RFC 9114, draft-ietf-webtrans-http3 | 3 (experimental) |
| Redis | https://redis.io/docs (Streams, PUB/SUB, sharded PUB/SUB, Cluster, Sentinel, hash-field TTL 7.4) | 2 |
| PostgreSQL | https://www.postgresql.org/docs/16/ (LISTEN/NOTIFY, advisory locks) | 2 |
| NATS | https://docs.nats.io (core, JetStream consumers/acks) | 2, 3 |
| ClickHouse | https://clickhouse.com/docs (MergeTree, TTL, Distributed, native protocol) | 6 |
| FCM | https://firebase.google.com/docs/cloud-messaging (HTTP v1 API) | 8 |
| APNs | https://developer.apple.com/documentation/usernotifications (token/cert auth, HTTP/2) | 8 |
| HMS Push Kit | https://developer.huawei.com/consumer/en/hms/huawei-pushkit | 8 |
| Web Push | RFC 8030, RFC 8291, RFC 8292, RFC 8188 | 8 |
| OpenTelemetry | https://opentelemetry.io/docs/specs/ (OTLP, env config) | 6 |
| Prometheus | https://prometheus.io/docs/practices/naming/, native histograms | 1, 6 |

These are recorded as the sources we will use; their content was **not** verified in Phase 0 beyond what the Centrifugo docs reference. Each phase re-verifies the relevant spec and records the version here.

## Reference observations on 2026-11-23

The pinned Centrifugo OSS reference image in §1 was run locally against the memory profile. CS-RPC-01 exercises OSS-SUB-12 and CS-TRANS-01 exercises the POST paths of OSS-TRANS-11 and OSS-TRANS-12 and the corresponding push framing in OSS-TRANS-09. See the [scenario catalogue](../testing/compatibility-suite.md) for scope.
