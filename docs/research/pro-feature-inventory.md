<!-- Generated in Phase 0 from public docs only (centrifugal.dev @ 69da714, PRO release notes v6.0.0..v6.10.3). Summaries, not copied text. Open questions OQ-nn here are mirrored in ../research/open-questions.md as PRO-OQ-nn. -->

# Centrifugo PRO feature inventory (clean-room research)

- Baseline: Centrifugo PRO v6.10.3 (GitHub release `centrifugal/centrifugo-pro` v6.10.3, 2026-11-08), built on OSS v6.10.3.
- Docs source: local clone of `centrifugal/centrifugal.dev`, commit `69da71466274f741bfe924bbf4a62a9124ec5776` (2026-11-13). Accessed 2026-11-15.
- Release notes: `gh release view <tag> --repo centrifugal/centrifugo-pro` for every v6.0.0..v6.10.3 tag (accessed 2026-11-15). URL pattern: `https://github.com/centrifugal/centrifugo-pro/releases/tag/<tag>`.
- Method: public docs only. These are summaries, not copied text. Anything the docs do not cover is listed as an OPEN QUESTION (section 5).
- Path to URL rule: the URL uses the page's front-matter `id`, which is not always the filename. Notable cases:
  - `client_msg_batching.md` → `/docs/pro/client_message_batching`
  - `channel_events.md` → `/docs/pro/channel_state_events`
  - `draft_client_publications.md` → `/docs/pro/client_publications`
- Base URL for the docs is `https://centrifugal.dev/docs/pro/<id>`. RN = release notes.

## 0. Page status (sidebar = `src/sidebars.js` "Pro" list)

| File | id / URL | In sidebar | Status / note |
|---|---|---|---|
| overview.md | /docs/pro/overview | yes | Feature list and sandbox limits (20 conns, 2 nodes, 5 API rps) |
| install_and_run.md | /docs/pro/install_and_run | yes | Packaging, `license` key, cosign verification since v6.10.3 |
| push_notifications.md | /docs/pro/push_notifications | yes | GA, except that `update_push_status` is marked experimental |
| tracing.md | /docs/pro/tracing | yes | Admin trace API documented as "not stable" |
| analytics.md | /docs/pro/analytics | yes | GA |
| rate_limiting.md | /docs/pro/rate_limiting | yes | GA |
| admin_ui.md | /docs/pro/admin_ui | yes | Snapshots: beta. Flight recorder: experimental |
| client_authentication.md | /docs/pro/client_authentication | yes | meta_from_claim and multi-JWKS: beta (since v6.5.0) |
| server_api_enhancements.md | /docs/pro/server_api_enhancements | yes | GA |
| user_status.md, connections.md, access_revoke.md | same ids | yes | GA |
| event_hooks.md | /docs/pro/event_hooks | yes | Channel state events: **preview**. Cache empty: GA |
| capabilities.md, channel_patterns.md, cel_expressions.md | same ids | yes | GA |
| performance.md | /docs/pro/performance | yes | `client.batch_periodic_events` is EXPERIMENTAL |
| scalability.md | /docs/pro/scalability | yes | Node-grouped sharded PUB/SUB is experimental |
| bandwidth_optimizations.md | /docs/pro/bandwidth_optimizations | yes | Channel compaction: beta |
| client_msg_batching.md | /docs/pro/client_message_batching | yes | GA |
| observability_enhancements.md | /docs/pro/observability_enhancements | yes | GA |
| server_tags_filter.md | /docs/pro/server_tags_filter | yes | GA |
| map_subscriptions.md | /docs/pro/map_subscriptions | yes | **Experimental**: the whole map subscription feature |
| shared_poll.md | /docs/pro/shared_poll | yes | **Experimental**: the whole shared poll feature. Relay is **in progress / not officially supported** |
| admin_idp_auth.md | /docs/pro/admin_idp_auth | no (orphan) | Stub that points to the admin_ui SSO section |
| process_stats.md | /docs/pro/process_stats | no (orphan) | CPU/RSS per node in admin UI and in the `info` API |
| delta_at_most_once.md | /docs/pro/delta_at_most_once | no (orphan) | Duplicate of the bandwidth_optimizations section |
| namespace_engines.md | /docs/pro/namespace_engines | no (orphan) | Older duplicate of scalability#per-namespace-engines |
| channel_events.md | /docs/pro/channel_state_events | no (orphan) | Older copy. **Conflict:** it says the `redis` engine is required, while event_hooks.md also allows a separate redis presence manager. event_hooks.md is newer |
| channel_cache_empty.md | /docs/pro/channel_cache_empty | no (orphan) | Older copy of the event_hooks cache empty section |
| draft_client_publications.md | /docs/pro/client_publications | **no** | Unlisted. No `draft:` flag, but no release note mentions it. **Release status unknown** (OQ-01) |
| dictionary_compression.md | (not built) | no | `draft: true`. A comment in the file says "Not released… excluded from production builds". Marked experimental. **Not part of the v6.10.3 baseline** |

## 1. Full PRO inventory

Columns: ID | Feature / sub-feature | Status | Min PRO ver | Config keys | API / protocol fields | Storage | Source | Notes.

Min PRO version notes:
- "≤6.0.0" means the feature predates the scanned range, which starts at v6.0.0.
- "unknown" means no release note or doc gives the version.

### PRO-01 Push notifications (push_notifications.md → /docs/pro/push_notifications)

| ID | Sub-feature | Status | Min ver | Config keys | API / fields | Storage | Notes |
|---|---|---|---|---|---|---|---|
| PRO-01.01 | Enable and providers | GA | ≤6.0.0 | `push_notifications.enabled`, `push_notifications.enabled_providers` (`fcm`,`hms`,`apns`,`webpush`), `dry_run`, `dry_run_latency`, `max_inactive_device_interval`, `read_from_replica` | — | PG (`database.postgresql.dsn`, `replica_dsn`) | Without `database`, only raw-token, native-topic and condition sends work. Device and topic APIs are unavailable |
| PRO-01.02 | FCM | GA | ≤6.0.0 | `push_notifications.fcm.credentials_file`, `.tokens_batch_size` (default 500, max 500) | `notification.fcm.message` (native FCM Message) | — | Since v6.10.0, startup fails if batch size >500 (RN v6.10.0) |
| PRO-01.03 | HMS | GA | ≤6.0.0 | `push_notifications.hms.app_id`, `.app_secret`, `.auth_endpoint`, `.push_endpoint`, `.tokens_batch_size` (default 1000, max 1000) | `notification.hms.message` | — | v6.9.0 fixes: OAuth token race, timeouts, partial success (RN v6.9.0) |
| PRO-01.04 | APNs | GA | ≤6.0.0 | `push_notifications.apns.endpoint` (`development`/`production`/custom https), `.bundle_id`, `.auth_type` (`token`/`cert`), `.token_key_file`, `.token_key_pem`, `.token_key_id`, `.token_team_id`, `.cert_p12_file`, `.cert_p12_b64`, `.cert_p12_password`, `.tokens_batch_size` (100) | `notification.apns.headers`, `.payload` | — | `apns-push-type` defaults to `alert`. APNs request timeout is 10s (RN v6.10.0) |
| PRO-01.05 | Web Push (VAPID) | GA | 6.8.2 | `push_notifications.webpush.vapid_public_key`, `.vapid_private_key`, `.subject`, `.tokens_batch_size` (100), `.allowed_endpoint_origins`, `.extra_allowed_endpoint_origins` | `notification.webpush.headers` (`TTL`,`Urgency`,`Topic`), `.payload`. The device `token` is the PushSubscription serialized as a JSON string | PG | RFC 8030/8291/8292/8188. Startup fails if the keys do not match (RN v6.10.0) |
| PRO-01.06 | Web Push SSRF protection | GA | 6.8.2 (docs) | `allowed_endpoint_origins`, `extra_allowed_endpoint_origins` | — | — | Always on, no opt-out: https only, no IP literals, private/link-local/loopback resolved addresses refused at connect time, no redirects. The default origin allowlist has 4 push-service globs |
| PRO-01.07 | Device storage and lifecycle | GA | ≤6.0.0 | `max_inactive_device_interval` | `device_register`, `device_update`, `device_remove`, `device_list` | PG | IDs are server-generated (embed timestamp and provider). `provider`+`token` is unique. Register is full-state replace. Errors 107/113 on a bad stored id. Auto-removal on FCM `UNREGISTERED`/`INVALID_ARGUMENT`, APNs 410/`ExpiredToken`, WebPush 404/410 (RN v6.10.0) |
| PRO-01.08 | Device topics | GA | ≤6.0.0 | — | `device_topic_update` (op add/remove/set, optional `user` guard), `device_topic_list` | PG | Topic sends read the device topic list |
| PRO-01.09 | User topics | GA | ≤6.0.0 | — | `user_topic_update` (`user` may be `""` = global binding), `user_topic_list` | PG | Applied immediately to the user's registered devices. The global binding is applied at next register |
| PRO-01.10 | Send push | GA | ≤6.0.0 | — | `send_push_notification` (recipient: `filter`/`fcm_tokens`/`fcm_topic`/`fcm_condition`/`hms_tokens`/`hms_topic`/`hms_condition`/`apns_tokens`/`webpush_tokens`) | queue | Exactly one recipient field may be set |
| PRO-01.11 | Queue: Redis Streams (default) | GA | ≤6.0.0 | `push_notifications.queue.type`=`redis`, `.queue.redis.{address…, reuse_from_engine, consumer_concurrency (64), max_stream_length (100000)}` | — | Redis | Full Redis options (Sentinel/Cluster/TLS) |
| PRO-01.12 | Queue: PostgreSQL | GA | ≤6.0.0. Dedicated PG since 6.0.0 | `queue.type`=`postgresql`, `.queue.postgresql.{dsn, reuse_from_database, consumer_concurrency (16), scheduler_consumer_concurrency (16), prefix}` | — | PG | Switching queues loses in-flight jobs |
| PRO-01.13 | Delayed push / cancel | GA | ≤6.0.0 | — | `send_at` (unix seconds), `uid`, `cancel_push{uid}` | queue | Cancel only works before processing starts. The reply is always empty. Limit-delayed pushes and retries cannot be cancelled |
| PRO-01.14 | Expiry / retry | GA | changed 6.10.0 | — | `notification.expire_at` | — | Without it, sending stops 24h after queueing. FCM/HMS whole-batch outages are retried with backoff (6.10.0). `optimize_for_reliability` flag exists. **No dead-letter queue is documented** |
| PRO-01.15 | Timezone-aware push | GA | ≤6.0.0 | — | device `timezone` (IANA, `Local` rejected), `limit_strategy.time_limit{send_after_time, send_before_time, no_tz_send_now}` | PG | Only with `filter` sends. Windows that cross midnight are rejected. Requires `expire_at` |
| PRO-01.16 | Templating | GA | ≤6.0.0 | — | `use_templating`, `use_meta` | PG | Go-template style `{{.device.meta.x}}`, `.device.id`, `.device.user`. String literals use `[[...]]`. Only with `filter` sends |
| PRO-01.17 | Localizations | GA | ≤6.0.0 | — | `localizations{lang:{translations{}}}`, device `locale` (`ll-CC`) | PG | The `.l10n` template variable. A default value is required |
| PRO-01.18 | Push rate limits (per device; key may use the user) | GA | ≤6.0.0 | `distributed_rate_limit.enabled`, `distributed_rate_limit.storage_type` (must be `redis`, RN v6.10.0). Other keys undocumented | `limit_strategy.rate_limit{key (vars `device.id`,`device.user`), policies[{rate, interval_ms}], drop_if_rate_limited}` | Redis | Limits apply per device. A key using `device.user` gives per-user limits. Only with `filter` sends |
| PRO-01.19 | Push status / interaction | **experimental** | unknown | needs `clickhouse_analytics.export.notifications.enabled` | `update_push_status{analytics_uid, status (delivered/interacted), device_id, msg_id}`, `analytics_uid` on send | ClickHouse | Returns `not available` without ClickHouse |
| PRO-01.20 | Push metrics | GA | various | — | `centrifugo_push_notification_count`, `_scheduled_request_count`, `_deferred_batch_count` (6.10.0), `_dropped_job_count` (6.10.0), `_queue_consuming_lag`, `_consuming_inflight_jobs`, `_job_duration_seconds` (deprecated Summary), `_job_duration_seconds_histogram` (6.8.1) | — | Also observability_enhancements.md |
| PRO-01.21 | Push UI in admin | GA | unknown | — | — | — | Mentioned in push and admin docs |
| PRO-01.22 | Paged sends for limits/templating | GA | 6.10.0 | — | — | — | RN v6.10.0 |
| PRO-01.23 | Graceful shutdown of push consumers | GA | 6.10.3 | `shutdown.timeout` | — | — | RN v6.10.3 |

### PRO-02 Tracing (tracing.md → /docs/pro/tracing)

| ID | Sub-feature | Status | Min ver | Config | API | Notes |
|---|---|---|---|---|---|---|
| PRO-02.01 | Channel and user trace streams (admin UI) | GA | ≤6.0.0 | — | — | Real-time protocol frames |
| PRO-02.02 | Admin trace HTTP endpoint | "not stable" | ≤6.0.0. `admin.api_key` auth since 6.6.5 | `admin.api_key` | `POST /admin/trace` body `{"type":"user"\|"channel","entity":"…"}`. Auth: `X-API-Key` or `Authorization: token <admin token>` (`POST /admin/auth` password=) | Stream stays open until the client closes (fix in 6.6.5) |
| PRO-02.03 | Save trace to file | GA | file download in UI 6.2.5 | — | curl `-o file`, UI download | RN v6.2.5. The /pro table says "Traces can be saved to files" |
| PRO-02.04 | CEL filter in tracing UI | GA | 6.2.5 | — | — | RN v6.2.5 (replaced `filterx`). Filter variables are undocumented |
| PRO-02.05 | Trace ends on shutdown | GA | 6.10.3 | — | — | RN v6.10.3 |

### PRO-03 ClickHouse analytics (analytics.md → /docs/pro/analytics)

| ID | Sub-feature | Status | Min ver | Config | Notes |
|---|---|---|---|---|---|
| PRO-03.01 | Export core | GA | ≤6.0.0 | `clickhouse_analytics.enabled`, `.clickhouse_dsn[]` (`tcp://user:pass@host:port`), `.clickhouse_database`, `.clickhouse_cluster`, `.skip_schema_initialization`, `.skip_ping_on_start`, `.tls` (6.6.4) | Native TCP only. DB taken from the DSN as fallback (6.6.4) |
| PRO-03.02 | Export types | GA | ≤6.0.0 | `.export.{connections,subscriptions,operations,publications,notifications}.enabled` | Tables `connections`, `subscriptions`, `operations`, `publications`, `notifications`, plus `*_distributed` |
| PRO-03.03 | Export filters | GA | 6.2.2 | `.export.connections.http_headers`, `.export.connections.grpc_metadata`, `.export.{connections,subscriptions,operations}.export_users`, `.export.publications.export_channels` | — |
| PRO-03.04 | Per-export tuning | GA | unknown | `max_buffer_size` (1e6), `flush_interval` (10s), `flush_size` (100000), `ttl` ("7 DAY") | Drops when the buffer is full, counted by metric |
| PRO-03.05 | Schema migration v6.8.2 columns | GA | 6.8.2 | auto ALTER unless `skip_schema_initialization` | `labels Map`, `namespace`, `channel` (subscriptions now one row per subscription), `latency`, `node`, `protocol`, `connected_at` |
| PRO-03.06 | Namespace resolution in export | GA | 6.8.2 | none | RPC namespace is taken from `rpc_namespace_boundary` |
| PRO-03.07 | Metrics | GA | — | — | `centrifugo_clickhouse_analytics_drop_count`, `_flush_duration_seconds` (+`_histogram`), `_batch_size` (+`_histogram`) |
| PRO-03.08 | Admin dashboards: Trends (+leaderboards), User explorer, Channel explorer | GA | 6.8.2 | — | admin_ui.md#analytics-dashboards |
| PRO-03.09 | Flight recorder (CSV export) | **experimental** | 6.8.2 | — | Per-connection or per-user operation timeline |
| PRO-03.10 | Graceful flush on shutdown | GA | 6.10.3 | — | RN v6.10.3 |

### PRO-04 Operation rate limits (rate_limiting.md → /docs/pro/rate_limiting)

| ID | Sub-feature | Status | Min ver | Config | Notes |
|---|---|---|---|---|---|
| PRO-04.01 | Per-connection in-memory token bucket | GA | ≤6.0.0 | `client.rate_limit.client_command.{enabled, dry_run, total, default, <cmd>}`. Each bucket is `{enabled, buckets:[{interval, rate}]}` | Commands: subscribe, unsubscribe, publish, history, presence, presence_stats, refresh, sub_refresh, rpc, map_publish, map_remove, track, untrack. Error 111 |
| PRO-04.02 | Per-user in-memory | GA | ≤6.0.0 | `client.rate_limit.user_command.*` (adds `connect`) | Authenticated users only |
| PRO-04.03 | Per-user Redis (cluster-wide) | GA | ≤6.0.0 | `client.rate_limit.redis_user_command.*` + `.redis{…, reuse_from_engine}` | No `total`. No unsubscribe/untrack (config error since 6.10.0). One round trip since 6.9.7 |
| PRO-04.04 | RPC method overrides | GA | array format 6.8.0 | `rpc.method_overrides[{method, enabled, buckets}]`. Legacy `method_override` map is deprecated until v7 | Both at once = startup error |
| PRO-04.05 | Namespace overrides | GA | unknown | `<cmd>.namespace_overrides[{namespace_name, enabled, buckets}]` | Replaces the base bucket. `total` is still checked |
| PRO-04.06 | Error limit / disconnect | GA | ≤6.0.0 | `client.rate_limit.client_error.{enabled, dry_run, total}` | Disconnect code = no-reconnect. Internal errors are not counted |
| PRO-04.07 | dry_run | GA | 6.10.0 | `dry_run` per layer | Metric `centrifugo_rate_limit_client_over_limit_count{layer,command,namespace,dry_run}` |
| PRO-04.08 | unsubscribe/untrack counted, never rejected | GA | 6.10.0 | — | Counts as a client error in client_command |
| PRO-04.09 | Validation | GA | 6.10.0 | — | Interval must be 1s–1h, rate >0, override needs `namespace_name` |

### PRO-05 / PRO-06 / PRO-07 / PRO-08 Admin UI (admin_ui.md → /docs/pro/admin_ui)

| ID | Sub-feature | Status | Min ver | Config | Notes |
|---|---|---|---|---|---|
| PRO-05.01 | OIDC SSO (PKCE + Discovery) | GA | ≤6.0.0 | `admin.oidc.{enabled, display_name, issuer, client_id, audience, redirect_uri, extra_scopes, access_cel}` | Access token must be a JWT with matching `aud`. Checked on every request. `openid` scope always sent |
| PRO-05.02 | Server-side OIDC flow | GA | 6.5.2 | `admin.oidc.client_secret`, requires `admin.secret` ≥32 chars | Redirect to `/admin/oidc/callback`. ID token encrypted in a Secure HttpOnly cookie. Login bound to the starting browser (6.10.0) |
| PRO-05.03 | Starts when the IdP is unreachable | GA | 6.10.0 | — | RN v6.10.0 |
| PRO-06.01 | Roles admin/viewer | GA | 6.10.0 | `admin.oidc.roles[{role, cel}]`, `admin.oidc.default_role` | Roles are unioned. No roles = admin. Password/API key access = admin. Viewer requests are rejected server-side |
| PRO-06.02 | Audit log (to process log) | GA | 6.10.0 | `admin.audit.log.enabled` | Log line with `component=admin_audit`: action, actor, auth, role, ip, target, result, time. Bypasses `log.level`. Payloads never logged. **Server API calls are not audited** |
| PRO-07.01 | Channels and connections snapshots | **beta** | 6.3.0 | `clickhouse_analytics.snapshots.enabled`, `.snapshots.ttl` ("14 DAY"), plus `database` (PG) and admin enabled | Tables `snapshot_channels`, `snapshot_connections`. `label_filter` applied at gather time. Anonymous and all-connections gather since 6.9.0 |
| PRO-08.01 | Config viewer | GA | 6.8.2 | none | Read-only effective config of the serving node. Secrets and URL/DSN credentials masked server-side. "Only non-defaults" toggle |
| PRO-37.01 | Inspector (channel/user) | GA | 6.9.1 | none | Shows resolved namespace/pattern options, presence/history/map widgets, user block state, device count, connections, actions with an in-memory/durable badge |
| PRO-37.02 | `admin.instance_name` | GA | 6.10.0 | `admin.instance_name` | Shown before login |
| PRO-37.03 | Status page extras | GA | 6.2.1 (aggregates), 6.10.0 (license) | `node.info_metrics_aggregate_interval` (OSS key) | Client-name distribution and frame/connect/subscribe/API/publication rates |
| PRO-09.05 | CPU/RSS per node | GA | unknown | — | process_stats.md. Also exposed in the `info` API (field names undocumented) |

### PRO-09 / PRO-28 Observability (observability_enhancements.md → /docs/pro/observability_enhancements)

| ID | Sub-feature | Status | Min ver | Config | Notes |
|---|---|---|---|---|---|
| PRO-09.01 | Client name resolution | GA | ≤6.0.0 | `prometheus.additional_client_names[]` | Official SDK names are built in. Inflight conns/subs metrics (v6 blog) |
| PRO-09.02 | Channel namespace resolution | GA | 6.0.0 | `prometheus.channel_namespace_resolution` | Adds a `channel_namespace` label to OSS metrics (docs/server/observability.md) |
| PRO-09.03 | Transport accept protocol label | GA | 6.5.0 | `prometheus.expose_transport_accept_protocol` | Label `accept_protocol`: h1/h2/h3/unknown/mem |
| PRO-09.04 | Client labels as dimensions | GA | 6.8.2 | `prometheus.client_labels[]` | Exported with an `app_` prefix. Missing label = "" |
| PRO-09.06 | OTel metrics bridge | GA | 6.8.1 | `opentelemetry.enabled`, `opentelemetry.metrics`, `OTEL_EXPORTER_OTLP_*` env | `service.instance.id` = node ID (6.8.3) |
| PRO-09.07 | GCP ADC auth | GA | 6.8.2 | `opentelemetry.google_cloud_adc_auth` | Base option is OSS for traces. PRO adds it for metrics |
| PRO-09.08 | Cloud resource detectors | GA | 6.8.3 | `opentelemetry.resource_detectors` (`gcp`,`aws`) | Startup fails if metadata lookup fails on the platform |
| PRO-09.09 | Summaries deprecated | deprecated | 6.8.1 | `prometheus.native_histograms` | Removal planned in v7 |
| PRO-09.10 | PRO metric families | GA | various | — | push, ClickHouse, `centrifugo_database_pool_*` (6.9.0), `centrifugo_bus_*`, rate limit, `centrifugo_rate_limit_hits_over_limit`, `centrifugo_channel_state_events_queue_consuming_lag_milliseconds`, `centrifugo_license_expiration_timestamp_seconds` (6.10.0), `centrifugo_shared_poll_relay_*`, `*_redis_node_grouped_*` (6.8.4) |
| PRO-09.11 | Official Grafana dashboard "PRO ·" rows | GA | unknown | — | — |
| PRO-35 | Sentry | GA | 6.0.0 | `sentry.{enabled, dsn, environment, sample_rate}` | — |

### PRO-10 / PRO-11 Client authentication (client_authentication.md → /docs/pro/client_authentication)

| ID | Sub-feature | Status | Min ver | Config | Fields | Notes |
|---|---|---|---|---|---|---|
| PRO-10.01 | meta_from_claim | **beta** | 6.5.0 | `client.token.meta_from_claim` (StringKeyValues `[{key,value}]`) | — | Key must match `^[A-Za-z_][A-Za-z0-9_]*$`. Dot paths with `\` escaping. Missing paths are skipped. Overrides the `meta` claim. Connection tokens only |
| PRO-10.02 | Multiple JWKS providers | **beta** | 6.5.0. Same-iss/different-aud since 6.5.2 | `client.token.jwks.{enabled, providers[{name, enabled, endpoint, issuer, audience, tls, meta_from_claim, labels_from_claim}]}` and `client.subscription_token.jwks.*` | — | Routing by iss (+aud). Mutually exclusive with `jwks_public_endpoint`. When enabled, HMAC/RSA/ECDSA/Ed25519 keys are ignored. Empty providers = reject all |
| PRO-11.01 | Labels from the `labels` claim | GA | 6.8.2 | — | JWT `labels` (map string→string) | Immutable for the connection lifetime |
| PRO-11.02 | labels_from_claim | GA | 6.8.2 | `client.token.labels_from_claim`, per-provider | — | gjson stringification. Overrides the `labels` claim. Not allowed for subscription tokens |
| PRO-11.03 | Labels from connect proxy | GA | 6.8.2 | — | connect result `labels` | JWT and proxy are never used together for one connection |
| PRO-11.04 | Labels on outgoing proxy requests | GA | 6.8.2 | none | top-level `labels` on publish/subscribe/rpc/refresh/sub_refresh/map_publish/map_remove | Always on |

### PRO-12 Server API enhancements (server_api_enhancements.md)

| ID | Sub-feature | Status | Min ver | Config / fields | Notes |
|---|---|---|---|---|---|
| PRO-12.01 | JWKS auth for HTTP/GRPC API | GA | 6.4.0 (RN). The doc says 6.3.2, **conflict** (OQ-02) | `http_api.jwks.{enabled, endpoint, audience, issuer, scope, tls}`, same under `grpc_api.jwks` | Only `use:sig` keys. `Authorization: Bearer`. Checks exp/nbf, optional aud/iss/scope. A bearer header overrides an API key. 503/`Unavailable` when no key can be fetched (6.10.2). Token verification cached (6.10.2). 1s clock skew (6.10.0) |
| PRO-12.02 | label_filter + all_users | GA | 6.8.2 | `label_filter` (FilterNode), `all_users` on `subscribe`, `unsubscribe`, `disconnect`, `refresh`. `label_filter` also on `connections` | FilterNode ops: eq, neq, in, nin, ex, nex, sw, ew, ct, gt, gte, lt, lte, and, or, not. Linear scan. Older nodes ignore it |
| PRO-31 | Grouped publications | GA (default on) | 6.9.7 | namespace `publication_grouping_disabled`, `publication_grouping_max_delay` (Redis/Memory only) | Order kept per channel, not across channels in a batch. Grouping on Redis/Memory only |

### PRO-13 User status (user_status.md)

| ID | Sub-feature | Status | Min ver | Config | API |
|---|---|---|---|---|---|
| PRO-13.01 | Client RPC `update_user_status` | GA | ≤6.0.0 | `user_status.disable_for_client` | client RPC method `update_user_status`, data must be `{}` (empty data is rejected). No-op for anonymous connections |
| PRO-13.02 | Server API | GA | ≤6.0.0 | `user_status.{enabled, redis{…, reuse_from_engine}, expire_interval (720h default), storage_type (redis only)}` | `update_user_status{users[]}`, `get_user_status{users[]}` → `statuses[{user, active?, online?}]`, `delete_user_status{users[]}` |
| PRO-13.03 | Online auto-update | GA | non-blocking connect since 6.9.7 | — | `online` updated periodically while connected |

### PRO-14 Connections API (connections.md)

| ID | Status | Min ver | API |
|---|---|---|---|
| PRO-14.01 | GA | ≤6.0.0. `label_filter` since 6.8.2 | `connections{user?, expression? (CEL), label_filter?}`. At least one is required. Result `connections{clientId: ConnectionInfo{app_name, app_version, transport, protocol, user, state{channels{ch:{source}}, connection_token{uid, issued_at}, subscription_tokens{ch:{uid, issued_at}}, meta}, connected_at_ms, ping_pong_latency_ms (-1 when unknown), labels}}`. Meta comes from the JWT `meta` claim or connect proxy `meta`. **No pagination is documented** |

### PRO-15 User blocking and token revocation (access_revoke.md)

| ID | Status | Min ver | Config | API |
|---|---|---|---|---|
| PRO-15.01 Block | GA | ≤6.0.0 | `user_block.{storage_type: redis\|database, redis{…}}` (single shard only, no `reuse_from_engine`) | `block_user{user, expire_at?}`, `unblock_user{user}` |
| PRO-15.02 Revoke by jti | GA | ≤6.0.0. Works everywhere since 6.10.0 | `token_revoke.{storage_type, redis}` | `revoke_token{uid, expire_at?}` |
| PRO-15.03 Invalidate by iat | GA | ≤6.0.0. `channel` field since 6.10.0 (unknown; implied by RN) | `user_tokens_invalidate.{storage_type, redis}` | `invalidate_user_tokens{user, issued_before?, expire_at?, channel?}` |

Notes for PRO-15:
- State is in memory by default. Nodes periodically load all entries from persistent storage.
- Entries expire at `expire_at` (fixed in 6.10.0). A negative `expire_at` is a bad request.
- `database` storage means PostgreSQL only.

### PRO-16 Event hooks (event_hooks.md)

| ID | Sub-feature | Status | Min ver | Config | Protocol |
|---|---|---|---|---|---|
| PRO-16.01 | Channel state events occupied/vacated | **preview** | ≤6.0.0. Improved in 6.0.0 | `channel.proxy.state.endpoint` (HTTP or `grpc://`), namespace `state_proxy_enabled`, `presence: true`. `engine.redis.channel_state.{vacated_event_delay (5s), num_partitions (128)}` or `presence_manager.redis.channel_state.*` | GRPC `NotifyChannelState`. HTTP body `{"events":[{channel, type, time_ms}]}`. Reply `{"result":{}}`. Retries with exponential backoff and jitter. Bounded queue drops the oldest events. Best-effort. Requires Redis presence. Redis must be noeviction/volatile-* |
| PRO-16.02 | Cache empty events | GA | ≤6.0.0 | `channel.proxy.cache_empty.{endpoint, timeout}`, namespace `cache_empty_proxy_enabled` | `NotifyCacheEmptyRequest{channel}` → `{populated?}`. If populated, recovery is retried once. Proxy failure disconnects the client with 3004 |

### PRO-17 Capabilities (capabilities.md)

| ID | Status | Min ver | Fields |
|---|---|---|---|
| PRO-17.01 Connection caps | GA | ≤6.0.0 | JWT `caps`, connect/refresh proxy `caps`: `[{channels[], allow[sub,pub,prs,hst], match: ""(exact)\|wildcard\|regex}]` |
| PRO-17.02 Subscription caps | GA | ≤6.0.0. Enable positioning/recovery/join-leave since 6.10.0 | subscription JWT `allow`, subscribe proxy `allow` |

Notes for PRO-17:
- Caps are evaluated per operation as a union of grants. Order does not matter and caps never deny.
- Regex is not anchored.
- Refresh with different caps disconnects with a reconnect code. A refresh proxy result without `caps` means no caps.
- Uni transport `subs` are checked against caps and CEL since 6.5.1.

### PRO-18 Channel patterns (channel_patterns.md)

- Status GA. Min ver ≤6.0.0, generalized in 6.0.0.
- Config: `channel.patterns: true` plus namespace `pattern: "/a/:x"`.

Notes:
- Only ASCII is allowed. Mid-segment variables are allowed (`user_:user`).
- Static segments take precedence, with backtracking.
- Same-shape conflicts and duplicate variables are rejected at startup.
- A channel with no matching pattern gets `102 unknown channel`.
- Patterns are not allowed in `without_namespace`.
- With patterns, `channel_regex` matches the whole channel.
- Pattern namespaces inherit the global `publication_data_format` (6.10.0).

### PRO-19 CEL (cel_expressions.md)

- Status GA. Min ver ≤6.0.0.
- Namespace options: `subscribe_cel`, `publish_cel`, `history_cel`, `presence_cel`.
- Variables: `subscribed`, `user`, `meta`, `channel`, `vars`, `labels` (labels since 6.8.2).

Notes:
- CEL is an additional allow path only: it runs when nothing else allowed the operation, and never when a proxy handles the operation.
- `history_cel` and `presence_cel` are not consulted for recovery or join/leave.
- Since 6.10.0, `subscribe_cel` no longer applies on connect to private, proxied or non-stream channels.
- Other CEL uses:
  - `connections.expression`
  - `admin.oidc.access_cel`
  - `admin.oidc.roles[].cel`
  - tracing UI filter
  - ephemeral publication tags (OQ-01)

### PRO-20 / PRO-32 / PRO-33 / PRO-34 Performance and scalability (performance.md, scalability.md)

| ID | Sub-feature | Status | Min ver | Config |
|---|---|---|---|---|
| PRO-20.01 | Singleflight (history, presence, presence stats) | GA | ≤6.0.0 | `singleflight.enabled` / `CENTRIFUGO_SINGLEFLIGHT_ENABLED` |
| PRO-20.02 | Shared position sync | GA | ≤6.0.0 | namespace `shared_position_sync` |
| PRO-32.01 | Subscribe on Redis replica (engine, broker, map broker) | GA | 6.0.0 | `*.redis.replica_client.enabled`, `*.redis.subscribe_on_replica` (Sentinel/Cluster only) |
| PRO-32.02 | Presence read from replica | GA | 6.0.0 | `replica_client.enabled`, `presence_read_from_replica`, needs `presence_hash_field_ttl` (Redis 7.4+) |
| PRO-33.01 | Redis Cluster sharded PUB/SUB | GA | 6.0.0 | `engine.redis.sharded_pub_sub_partitions` (also `map_broker.redis`) |
| PRO-33.02 | Precomputed partition tags | GA | 6.8.3 | `use_precomputed_partition_tags` (partitions ∈ {16..4096 powers of 2}) |
| PRO-33.03 | Node-grouped sharded PUB/SUB | **experimental** | 6.8.0 | `group_sharded_pub_sub_by_node` |
| PRO-34.01 | Batch periodic events | **EXPERIMENTAL** | 6.2.0 | `client.batch_periodic_events` |
| PRO-34.02 | Commands off the read loop | GA | 6.9.2 | `websocket.process_commands_off_read_loop` |
| PRO-34.03 | Fast HTTP proxy client | GA | 6.0.3. All HTTP proxies since 6.10.0 | `<proxy>.http.use_fast_client` (HTTP/1.1 only) |
| PRO-34.04 | Prepared WS compression message cache | GA | unknown | `websocket.compression_prepared_message_cache_size` (bytes) |
| PRO-34.05 | Faster JSON/Protobuf codecs (API, proxy, consumers, JWT, uni GRPC) | GA | unknown | none (internal) |

### PRO-21 / PRO-22 / PRO-40 Engines (scalability.md)

| ID | Sub-feature | Status | Min ver | Config |
|---|---|---|---|---|
| PRO-21.01 | Named brokers per namespace | GA | ≤6.0.0. `postgres` type fixed in 6.8.1 | `brokers[{enabled, name, type: redis\|nats\|postgres, redis/nats/postgres{…}}]`, namespace `broker_name` |
| PRO-21.02 | Named presence managers | GA | ≤6.0.0 | `presence_managers[{enabled, name, type: redis, redis{}}]`, namespace `presence_manager_name` |
| PRO-22.01 | Custom controller | GA | 6.2.3 | `controller.{enabled, type: redis\|nats, redis{}/nats{}}`. `postgres` is OSS and works in PRO since 6.10.0 |
| PRO-40.01 | PG stream broker read replicas and fan-out | GA | 6.10.0 | `broker.postgres.replica.{enabled, dsn[]}`, `broker.postgres.broker_fanout.{enabled, type, redis/nats, advisory_lock_base_id}` |
| PRO-40.02 | PG controller read replica | unknown | 6.8.0 (RN) | **The doc section is missing from the current docs** (OQ-03) |

### PRO-23 Bandwidth (bandwidth_optimizations.md)

| ID | Sub-feature | Status | Min ver | Config |
|---|---|---|---|---|
| PRO-23.01 | Delta for at-most-once | GA | unknown | namespace `keep_latest_publication` (+ `allowed_delta_types`) |
| PRO-23.02 | Channel compaction | **beta** | 6.5.0 (missed the 6.4.0 tag) | namespace `allow_channel_compaction`. The SDK negotiates it. Varint channel id in protobuf |
| PRO-23.03 | Client publish debouncing | GA (only `centrifuge-js` supports it) | 6.8.0 | namespace `client_publish_debounce_interval`. Returned in the subscribe result |
| PRO-23.04 | Drop intermediary publications | GA | 6.2.0 | → `batch_flush_latest` |

### PRO-24 Message batching (client_message_batching)

| ID | Sub-feature | Status | Min ver | Config |
|---|---|---|---|---|
| PRO-24.01 | Write delay | GA | ≤6.0.0 | `client.write_delay` |
| PRO-24.02 | Timer-based writes | GA | 6.6.0 | `client.write_with_timer` (needs write_delay) |
| PRO-24.03 | Queue shrink delay | GA | 6.6.0 | `client.queue_shrink_delay` |
| PRO-24.04 | Max messages per frame | GA | unknown | `client.max_messages_in_frame` (16, -1 = unlimited) |
| PRO-24.05 | Replies bypass the queue | GA | unknown | `client.reply_without_queue` |
| PRO-24.06 | Channel batching | GA | 6.2.0 | namespace `batch_max_size`, `batch_max_delay`, `batch_flush_latest` |

Notes for PRO-24:
- `batch_flush_latest` keeps the latest publication per key for map subscriptions and keeps join/leave.
- Combining it with `allowed_delta_types` is rejected since 6.9.7.

### PRO-25 Server tags filter (server_tags_filter.md)

- Status GA. Min ver 6.8.0.
- Set via:
  - subscribe proxy `server_tags_filter`
  - subscription JWT `server_tags_filter`
  - connection JWT `subs.<ch>.server_tags_filter`
  - connect proxy `subs.<ch>.server_tags_filter`
- Filter language: FilterNode.

Notes:
- Filtering applies to live delivery, history recovery and cache recovery. For map subscriptions it applies to the state, stream and live phases.
- Map removal events carry the tags of the removed entry.
- The filter is evaluated before the client filter.
- Client `history` on a filtered subscription gets 103 (6.10.0).
- Updating the filter on sub refresh: hot swap for streams, resubscribe for maps (actually works since 6.10.0). Refresh cannot remove the filter.
- Incompatible with delta compression.
- A `null` node refuses the subscription (6.10.0).

### PRO-26 Map subscription enhancements (map_subscriptions.md). The whole feature is EXPERIMENTAL and only `centrifuge-js` supports it.

| ID | Sub-feature | Min ver | Config |
|---|---|---|---|
| PRO-26.01 | In-memory cache | 6.8.0 | `map_broker.cache.{enabled, max_channels (10000/-1), idle_timeout (5m/"-1s", ≥1s), sync_interval (30s), sync_concurrency (64), sync_batch_size (1000), load_timeout (5s), stream_size (1000)}` |
| PRO-26.02 | Full-state delta sync | 6.8.0 | namespace `map.full_state_channel_prefix`, `map.full_state_tick_interval` (250ms). Needs the cache and a fossil stream namespace |
| PRO-26.03 | PG read replicas (map and stream broker) | 6.8.0 (map), 6.10.0 (stream) | `*.postgres.replica.{enabled, dsn[]}` (both required) |
| PRO-26.04 | PG broker fan-out | 6.8.0 | `*.postgres.broker_fanout.{enabled, type: redis\|nats, …, advisory_lock_base_id}` |
| PRO-26.05 | Redis Cluster map broker via sharded PUB/SUB | 6.8.0 | `map_broker.redis.sharded_pub_sub_partitions` (>0), plus node-grouped, subscribe-on-replica and precomputed tags |
| PRO-26.06 | Per-namespace map brokers | 6.8.0 | `map_brokers[{name, enabled, type, …, cache}]`, namespace `map.broker_name` |

### PRO-27 Shared poll enhancements (shared_poll.md). The whole feature is EXPERIMENTAL and only `centrifuge-js` supports it.

| ID | Sub-feature | Status | Min ver | Config |
|---|---|---|---|---|
| PRO-27.01 | Cached latest data / instant data | experimental | 6.8.0 | namespace `shared_poll.keep_latest_data` (instant data needs `versioned` mode) |
| PRO-27.02 | Delta compression | experimental | 6.8.0 | `keep_latest_data` + `allowed_delta_types:["fossil"]` |
| PRO-27.03 | Notification fast path | experimental | 6.8.0 | `shared_poll.notification.{enabled, type: redis\|nats, redis{…,prefix}, nats{…,prefix}, channel ("shared_poll_notify"), batch_max_size, batch_max_delay}`, plus a per-namespace `shared_poll.notification.{batch_max_size, batch_max_delay}`. Message format `{"items":[{channel,key}]}` |
| PRO-27.04 | Adaptive backpressure | experimental | 6.8.0 | namespace `shared_poll.backpressure_max_interval` |
| PRO-27.05 | Shared poll relay | **IN PROGRESS / not officially supported** (shipped in binaries) | 6.8.0 | `shared_poll_relay.{enabled, endpoint, http_server{enabled,address,port 9090}, grpc_server{enabled,address,port 9091}, history_size (3), item_ttl (90s), notification{…}}`. CLI `--mode=shared_poll_relay`. Ready channel `shared_poll_ready` / `<channel>_ready`. Metrics `centrifugo_shared_poll_relay_*` |

### New items not in SPEC PRO-01..28 (PRO-29+)

| ID | Feature | Status | Min ver | Config / fields | Source |
|---|---|---|---|---|---|
| PRO-29 | Ephemeral client publications: JSON Schema (draft 2020-12) / `empty_binary` validation, tag extraction (`path`/`cel`/`if`), `exclude_client_info` | **unlisted, release unknown** | unknown | top-level `schemas[{name, type, definition (inline or file path)}]`, namespace `client_publication.{schemas[], tags[{key, path\|cel, if}], exclude_client_info}`. CEL vars: data, timestamp_ms, schema_name, user, client, meta, vars, labels | draft_client_publications.md → /docs/pro/client_publications |
| PRO-30 | Dictionary compression (WebSocket, PG-backed profiles, training, admin-UI-driven) | **draft / unreleased / experimental** | not released | `dictionary_compression.{enabled, frame_cache_size, max_live_codecs, max_sampled_connections, frame_budget_per_second, size_ladder_bytes, snapshot_poll_interval, flush_interval, maintenance_interval, max_concurrent_sessions, candidate_ttl, denied_value_names}`, JWT/connect-proxy `profile` | dictionary_compression.md (not published) |
| PRO-31 | Grouped publications | GA | 6.9.7 | see PRO-12 | server_api_enhancements.md#grouped-publications |
| PRO-32 | Redis replica offload | GA | 6.0.0 | see above | scalability.md |
| PRO-33 | Redis Cluster sharded PUB/SUB (engine), precomputed tags, node-grouped | GA / experimental | 6.0.0 / 6.8.3 / 6.8.0 | see above | scalability.md. Also a separate row on the /pro comparison table |
| PRO-34 | Connection runtime and codec performance options | mixed | 6.0.3–6.9.2 | see above | performance.md |
| PRO-35 | Sentry integration | GA | 6.0.0 | `sentry.*` | observability_enhancements.md#sentry-integration |
| PRO-36 | License key and sandbox mode | N/A for us (licensing; do not replicate) | — | `license` / `CENTRIFUGO_LICENSE`, multiple keys (6.2.1) | install_and_run.md, overview.md |
| PRO-37 | Admin UI extras: Inspector, instance name, status page aggregates/license, push device UI | GA | 6.2.1–6.10.0 | `admin.instance_name` | admin_ui.md |
| PRO-38 | OTel metrics export, GCP ADC, resource detectors | GA | 6.8.1–6.8.3 | see PRO-09 | observability_enhancements.md |
| PRO-39 | Extra Prometheus labels: accept_protocol, client names, client labels | GA | 6.0.0–6.8.2 | see PRO-09 | observability_enhancements.md |
| PRO-40 | PostgreSQL stream broker replicas and fan-out. PG controller read replica | GA / unknown | 6.10.0 / 6.8.0 | see above | RN v6.10.0, RN v6.8.0 |
| PRO-41 | PostgreSQL pool sizing and metrics for PRO storage | GA | 6.9.0 | `database.postgresql.pool_max_size` (16), `database.postgresql.replica_dsn[]`. Migration tables `cf_migrations`, `cf_queue_migrations` | RN v6.9.0, observability_enhancements.md |
| PRO-42 | Release signing and SBOM (PRO artifacts) | GA | 6.10.0 | — | install_and_run.md (says v6.10.3), RN v6.10.0 (says 6.10.0). **Conflict** (OQ-04) |

## 2. PRO server API methods (HTTP `POST /api/<method>`, also GRPC)

Push (PRO-01):

- `device_register`: `id?`, `provider`* (fcm/hms/apns/webpush), `token`*, `platform`* (ios/android/web), `user?`, `timezone?`, `locale?` (ll-CC), `topics[]?`, `meta{string:string}?`. Returns `{id}`.
- `device_update`: one of `ids[]`/`users[]`, plus `user_update{user}`, `timezone_update{timezone}`, `locale_update{locale}`, `meta_update{meta}` (only devices that already have meta), `topics_update{op add|remove|set, topics[]}`. Returns `{}`.
- `device_remove`: one of `ids[]`/`users[]`. Returns `{}`.
- `device_list`:
  - Request: `filter{ids, providers, platforms, users, topics}`, `cursor`, `limit`, `include_total_count`, `include_topics`, `include_meta`, `include_webpush_keys`.
  - Returns `{items[Device], next_cursor?, total_count?}`.
  - Device fields: `id`, `provider`, `token`, `platform`, `user`, `timezone`, `locale`, `created_at` (ms), `updated_at` (ms), `topics`, `meta`, `webpush_keys`.
- `device_topic_update`: `device_id`*, `op`* (add/remove/set), `topics[]`, `user?` (ownership guard: conflict or not found). Returns `{}`.
- `device_topic_list`:
  - Request: `filter{device_ids, device_providers, device_platforms, device_users, topics, topic_prefix}`, `cursor`, `limit`, `include_device`, `include_total_count`.
  - Returns `{items[{id, topic, device?}], next_cursor?, total_count?}`.
- `user_topic_update`: `user`* (may be `""`), `op`*, `topics[]`. Returns `{}`.
- `user_topic_list`: `filter{users, topics, topic_prefix}`, `cursor`, `limit`, `include_total_count`. Returns `{items[{id, user, topic}], next_cursor?, total_count?}`.
- `send_push_notification`:
  - Request: `recipient`*, `notification`*, `uid?`, `send_at?`, `optimize_for_reliability?`, `limit_strategy?`, `analytics_uid?`, `localizations?`, `use_templating?`, `use_meta?`. Returns `{uid}`.
  - `recipient` is exactly one of: `filter` (DeviceFilter), `fcm_tokens`, `fcm_topic`, `fcm_condition`, `hms_tokens`, `hms_topic`, `hms_condition`, `apns_tokens`, `webpush_tokens`.
  - `notification` fields: `expire_at?`, `fcm{message}`, `hms{message}`, `apns{headers?, payload}`, `webpush{headers?, payload}`.
  - `limit_strategy` fields: `rate_limit{key?, policies[{rate, interval_ms}], drop_if_rate_limited?}` and `time_limit{send_after_time, send_before_time, no_tz_send_now?}`.
- `cancel_push`: `uid`*. Returns `{}` (always).
- `update_push_status` (experimental): `analytics_uid`*, `status`* (delivered/interacted), `device_id`*, `msg_id?`. Returns `{}`.

User status (PRO-13):
- `update_user_status{users[]}`
- `get_user_status{users[]}` → `{statuses[{user, active?, online?}]}`
- `delete_user_status{users[]}`
- Also a client RPC: `update_user_status` with data `{}`.

Connections (PRO-14): `connections{user?, expression?, label_filter?}` → `{connections{clientId: ConnectionInfo}}`.

Blocking and revocation (PRO-15):
- `block_user{user, expire_at?}`
- `unblock_user{user}`
- `revoke_token{uid, expire_at?}`
- `invalidate_user_tokens{user, issued_before?, expire_at?, channel?}`
- All return `{}`.

OSS methods with PRO fields (PRO-12):
- `subscribe`, `unsubscribe`, `disconnect` and `refresh` gain `label_filter` and `all_users` (docs/server/server_api.md marks these "Centrifugo PRO").
- `publish`/`broadcast` behavior changes with grouping (6.9.7). There are no new fields.

Methods that do not exist in the docs:
- No rate-limit management API.
- No snapshot server API. Snapshots are created through an admin endpoint whose path is undocumented (OQ-05).
- No `device_topic_*` or `user_topic_*` methods beyond `update` and `list`.
- No `get_device`.

Admin (non-server-API) endpoints that are documented:
- `POST /admin/auth` (password)
- `POST /admin/trace` (`type`, `entity`)
- `/admin/oidc/callback`

Proxy / protocol additions:
- State proxy: `NotifyChannelState{events[{channel, type, time_ms}]}`.
- Cache empty proxy: `NotifyCacheEmpty{channel}` → `{populated}`.
- Connect result: `caps`, `labels`, `subs.<ch>.server_tags_filter`, `meta`.
- Refresh result: `caps`.
- Subscribe result: `allow`, `server_tags_filter`.
- Sub refresh result: `server_tags_filter`.
- JWT claims: `caps`, `labels`, `meta` (+mapping), `subs.<ch>.server_tags_filter`. Subscription JWT: `allow`, `server_tags_filter`. Revocation uses `jti`/`iat`.
- Subscribe reply carries `client_publish_debounce_interval`. The field name inside the client protocol is undocumented (OQ-06).
- Channel compaction is negotiated in the subscribe exchange. Wire details are not in PRO docs (OQ-07).

## 3. /pro comparison matrix

Source: `src/pages/components/ComparisonTable.jsx`, rendered on https://centrifugal.dev/pro.

The component supports `'soon'` and `preview` badges, but **no row uses them**. Statuses below come from the description text.

Core Messaging:
- PUB/SUB, channels, namespaces: OSS ✓, PRO ✓
- WebSocket, SSE, HTTP-streaming, WebTransport, GRPC transports: OSS ✓, PRO ✓
- WebSocket bidirectional emulation: OSS ✓, PRO ✓
- Stream subscriptions with history, recovery, cache mode: OSS ✓, PRO ✓
- Map subscriptions for state sync: OSS ✓, PRO ✓
- Map subscriptions enhancements: PRO only. Covers the in-memory cache, PG read replicas and fan-out, Redis Cluster sharded PUB/SUB, and per-namespace map brokers
- Shared poll subscriptions: OSS ✓, PRO ✓
- Shared poll enhancements: PRO only. Covers instant data, delta, notification fast path and adaptive backpressure. **The relay is "in progress and not yet officially supported"**
- Presence & join/leave events: OSS ✓, PRO ✓
- Delta compression, client-side publication filtering by tags: OSS ✓, PRO ✓
- Server-side publication tags filter: PRO only
- Proxy events (connect, subscribe, publish, RPC): OSS ✓, PRO ✓
- Official real-time SDKs: OSS ✓, PRO ✓

Security, auth, permissions:
- JWT & proxy authentication: OSS ✓, PRO ✓
- Channel capabilities & CEL expressions: PRO only
- Channel patterns: PRO only
- Operation rate limiting: PRO only (token bucket; disconnect on errors)
- User blocking API: PRO only (optional Redis/DB persistence)
- Token revocation & invalidation: PRO only (jti / iat; in memory with periodic sync)
- Multiple JWKS providers: PRO only. **Its description actually describes claim→meta mapping**
- Client labels: PRO only
- Server API JWKS authentication: PRO only

Scalability and performance:
- Scale with Redis, Redis Cluster and NATS: OSS ✓, PRO ✓
- Per-namespace engines: PRO only
- Singleflight & shared position sync: PRO only
- Performance optimizations: PRO only. Claims "2x less CPU and 2.5x less memory for idle connections" (vendor claim, not verified)
- Sharded PUB/SUB for Redis Cluster: PRO only (includes node-grouped; engine and map broker)
- Bandwidth optimizations: PRO only (delta at-most-once, compaction, debouncing)
- Advanced message write and batching: PRO only (`write_delay`)
- Fast Custom Controllers (Redis, Nats): PRO only. PostgreSQL controller is OSS

Observability:
- Prometheus metrics, admin UI: OSS ✓, PRO ✓
- ClickHouse analytics (current stats, trends, per-connection resolution): PRO only (includes the flight recorder)
- Real-time channel & user tracing: PRO only. "Traces can be saved to files"
- Connection & channel snapshots: PRO only
- SSO/OIDC for admin UI: PRO only (PKCE; checked on every request)
- Admin roles and audit log: PRO only
- Configuration viewer: PRO only
- Enhanced metrics: PRO only (namespace resolution, SDK name, Prometheus→OTEL bridge, CPU/RSS per node)

Engagement:
- Push notifications (FCM, APNs, HMS, WebPush): PRO only (timezone, localization, templating)
- Device token management & topics: PRO only (PG storage, topics, Redis worker queues)

Additional APIs and events:
- Additional event hooks: PRO only (channel state and cache empty)
- User status API: PRO only
- Connections API: PRO only

Support:
- Community support: OSS ✓, PRO ✓
- Prioritized support: PRO only

The `/pro` JSON-LD `featureList` also includes "Idle connection memory tuning". It has no separate table row.

## 4. Gap analysis vs SPEC PRO-01..28

### 4a. Documented PRO features and sub-features missing from the SPEC modules

1. PRO-29: Ephemeral client publications (schema validation, CEL/path tag extraction, `exclude_client_info`). The page is unlisted and no release note mentions it, so its status needs a decision.
2. PRO-30: Dictionary compression. Draft and unreleased, so **exclude it from the v6.10.3 baseline**. Track it only.
3. PRO-31: Grouped publications. Default-on behavior change since 6.9.7, with namespace options `publication_grouping_disabled` and `publication_grouping_max_delay`.
4. PRO-32: Redis replica offload (`subscribe_on_replica`, `presence_read_from_replica`).
5. PRO-33: Redis Cluster sharded PUB/SUB for the engine (`sharded_pub_sub_partitions`, `use_precomputed_partition_tags`, `group_sharded_pub_sub_by_node`). SPEC PRO-20 does not name it, but the /pro table lists it as its own row.
6. PRO-34: Runtime options:
   - `client.batch_periodic_events` (experimental)
   - `websocket.process_commands_off_read_loop`
   - `websocket.compression_prepared_message_cache_size`
   - `use_fast_client`
   - faster codecs (no config)
7. PRO-35: Sentry integration.
8. PRO-37: Admin UI Inspector, `admin.instance_name`, status-page aggregates and license view, push device UI.
9. PRO-38: OTel extras beyond the "bridge" (GCP ADC auth, `resource_detectors`, `service.instance.id`).
10. PRO-39: `accept_protocol` metric label and `additional_client_names`.
11. PRO-40: PostgreSQL stream broker replicas and fan-out (6.10.0). PG controller read replica (RN 6.8.0, doc missing).
12. PRO-41: `database.postgresql.pool_max_size` and `centrifugo_database_pool_*` metrics.
13. Smaller items:
    - Web Push SSRF rules (PRO-01.06), a security requirement.
    - `update_user_status` client RPC with `user_status.disable_for_client`.
    - Uni transport `subs` checked by caps/CEL (6.5.1).
    - `admin.api_key` for the trace API (6.6.5).
    - Graceful shutdown semantics for PRO subsystems (6.10.3).
    - Push metrics `centrifugo_push_deferred_batch_count` and `centrifugo_push_dropped_job_count`.
    - `centrifugo_license_expiration_timestamp_seconds` (licensing, N/A).
    - CPU/RSS in the `info` API.
14. Blog `2026-08-17-tuning-idle-connections` presents `websocket.use_write_buffer_pool` and `websocket.read_buffer_size` as PRO knobs. They are documented as **OSS** in docs/transports/websocket.md, so we classify them as OSS. Record this as a source conflict (OQ-08).

### 4b. SPEC claims checked against the docs

| SPEC claim | Verdict | Evidence |
|---|---|---|
| PRO-01 HMS provider | **SUPPORTED** | push_notifications.md#hms, config reference |
| PRO-01 Web Push/VAPID | SUPPORTED (since 6.8.2) | push_notifications.md#web-push-vapid, RN v6.8.2 |
| PRO-01 "Per-user/device rate limiting" | **SUPPORTED, with caveats** | Limits apply per device. A `key` using `device.user` gives per-user limits. Requires `distributed_rate_limit` (Redis). Only works with `filter` sends. The `distributed_rate_limit` config block is barely documented (OQ-09) |
| PRO-01 "external token storage, if the baseline provides it" | **PARTIAL / NOT AS A PLUGGABLE STORE** | Docs only allow skipping PG and sending to raw provider tokens and native topics/conditions. No external storage adapter or interface is documented. Implement "raw-token mode" only |
| PRO-01 "retry and dead-letter strategy" | **Retry SUPPORTED, dead-letter NOT DOCUMENTED** | `expire_at` / 24h cutoff, backoff on FCM/HMS outages, drop metrics. There is no DLQ |
| PRO-01 send_at, cancel, timezone, localization, templates, status API, ClickHouse push analytics, provider passthrough | SUPPORTED | push_notifications.md. `update_push_status` is **experimental** |
| PRO-02 "Save trace to file" | **SUPPORTED** | tracing.md (curl `-o`), RN v6.2.5 (UI download), /pro table |
| PRO-02 filters | PARTIAL | CEL filter in the tracing UI (RN v6.2.5). Variables undocumented |
| PRO-02 time windows, sampling, volume limits, redaction, RBAC/audit for tracing, bounded buffers/drop counters | **NOT DOCUMENTED** | Only the viewer role (sees everything) and the audit log of admin changes exist |
| PRO-03 current stats, trends, connection drill-down, flight recorder, TTL | SUPPORTED (flight recorder experimental) | analytics.md, admin_ui.md |
| PRO-03 partitioning, materialized aggregates, analytics API, PII field suppression, tenant isolation | **NOT DOCUMENTED** | Only `export_users`/`export_channels`, header allowlists and TTL exist. Partitioning and engine details are in DDL not reviewed here (OQ-10) |
| PRO-04 limits per connection/user/namespace, RPC method, cluster-wide (per user via Redis), error limits | SUPPORTED | rate_limiting.md |
| PRO-04 limits by client labels, per channel (other than namespace), global/cluster-wide total, admin/API limits | **NOT DOCUMENTED** | — |
| PRO-05 PKCE, issuer/audience checks, per-request checks, claims→roles | SUPPORTED | admin_ui.md |
| PRO-05 state/nonce, CSRF, refresh token, explicit session lifecycle | **NOT DOCUMENTED** (logout with `cookie_domain` is mentioned in RN v6.10.0 only) | OQ-11 |
| PRO-06 roles, server-side enforcement, audit of logins and changes (who/IP/action/target/result) | SUPPORTED (6.10.0) | admin_ui.md#roles, #audit-log |
| PRO-06 correlation/request ID, tamper protection, retention/export/filtering of audit | **NOT DOCUMENTED** | Audit goes only to the process log. Server API calls are not audited |
| PRO-07 cluster snapshot, inspection, labels, filter | SUPPORTED, **beta** | admin_ui.md |
| PRO-07 pagination, export, partial-result indicators | **NOT DOCUMENTED** | — |
| PRO-08 effective config, redaction | SUPPORTED | admin_ui.md#configuration-viewer |
| PRO-08 value source (file/env/flag/default) | **NOT DOCUMENTED** (only "Only non-defaults") | — |
| PRO-09 namespace dims, SDK names, OTel bridge, CPU/RSS, push/ClickHouse/PG pool metrics, dashboard | SUPPORTED | observability_enhancements.md |
| PRO-09 alert rules | NOT DOCUMENTED (except the license expiry example) | — |
| PRO-10 meta/labels from claims, multiple JWKS providers | SUPPORTED, **beta** | client_authentication.md |
| PRO-10 key rotation/cache TTL/background refresh/stale-key policy/algorithm constraints for client JWKS providers | **NOT DOCUMENTED** for client providers | Server API JWKS: keys cached and refreshed, stale keys still used while the endpoint is down, 503 when no key (server_api_enhancements.md) |
| PRO-11 labels sources and consumers, immutability | SUPPORTED | client_authentication.md |
| PRO-11 key/value size limits | **NOT DOCUMENTED** | — |
| PRO-12 JWKS server auth, label_filter/all_users on subscribe/unsubscribe/disconnect/refresh/connections | SUPPORTED | server_api_enhancements.md |
| PRO-12 "and other documented operations", pagination | **NOT DOCUMENTED** beyond the five methods listed | — |
| PRO-13 all items | SUPPORTED (storage Redis only) | user_status.md |
| PRO-14 user/expression/label filter, meta | SUPPORTED | connections.md |
| PRO-14 pagination, node field, targeting from results | **NOT DOCUMENTED** (ConnectionInfo has no `node` field) | — |
| PRO-15 block/unblock, revoke, invalidate, propagation, persistence, TTL, refresh-bypass protection | SUPPORTED | access_revoke.md, RN v6.10.0 |
| PRO-15 "audit trail" | NOT DOCUMENTED for the server API (admin UI actions are audited) | — |
| PRO-16 occupied/vacated, cache empty, retry/timeout | SUPPORTED. Channel state is **preview** | event_hooks.md |
| PRO-16 deduplication, ordering guarantees | **NOT DOCUMENTED** (best effort, batched, per-partition serialization) | — |
| PRO-17 caps, sub/pub/prs/hst, exact/wildcard/regex, union grant-only | SUPPORTED | capabilities.md |
| PRO-18 all | SUPPORTED. Unicode is not allowed (ASCII only). No wildcards | channel_patterns.md |
| PRO-19 subscribe/publish/history/presence CEL. Vars user/meta/channel/vars/labels/subscribed | SUPPORTED | cel_expressions.md |
| PRO-19 "connect" CEL, `operation` variable, resource limits | **NOT DOCUMENTED**. No connect CEL and no `operation` variable exist | — |
| PRO-20 singleflight, shared position sync | SUPPORTED | scalability.md |
| PRO-21/22 | SUPPORTED. Presence managers are Redis only | scalability.md |
| PRO-23 delta at-most-once, compaction (beta), debouncing (js only) | SUPPORTED | bandwidth_optimizations.md |
| PRO-23 bandwidth-savings metrics | **NOT DOCUMENTED** for GA features | — |
| PRO-24 batching size/delay/mode, per-namespace | SUPPORTED | client_message_batching |
| PRO-24 per-transport overrides | **NOT DOCUMENTED** | — |
| PRO-25 | SUPPORTED | server_tags_filter.md |
| PRO-26 | SUPPORTED. **The whole feature is experimental** | map_subscriptions.md |
| PRO-27 relay = WIP | **CONFIRMED** ("in progress", not officially supported, shipped in binaries) | shared_poll.md, ComparisonTable.jsx. The base shared poll feature is also experimental |
| PRO-28 diagnostic endpoints with RBAC; trace/metric/analytics correlation | **NOT DOCUMENTED** as a distinct feature | — |

## 5. Open questions

- **OQ-01:** Is "Ephemeral client publications" (`/docs/pro/client_publications`) released in v6.10.3?
  - The page is unlisted and has no draft flag. No release note mentions `schemas` or `client_publication`.
  - Decide whether it is in scope.
- **OQ-02:** Server API JWKS minimum version.
  - server_api_enhancements.md says PRO v6.3.2, but no v6.3.2 tag exists.
  - RN v6.4.0 announces it, with a dead link `/docs/pro/server_api_auth`.
  - Proposed: use 6.4.0.
- **OQ-03:** PostgreSQL controller read replica (RN v6.8.0 links `scalability#read-replica-support`).
  - The current docs have no such section.
  - Are the config key names and status still valid?
- **OQ-04:** When did release signing start? install_and_run.md says v6.10.3; RN v6.10.0 already lists signed releases and SBOMs. This affects only our own release process.
- **OQ-05:** Snapshot creation and listing endpoints are not documented: paths, request/response shapes, permissions. Only the behavior and `label_filter` are described.
- **OQ-06:** Exact client-protocol fields for:
  - the debounce interval in the subscribe result
  - channel compaction negotiation (the id field in push/publication)
  - the distribution of `keep_latest_publication` deltas
  - These need the public protocol `.proto` (centrifugal/protocol) and black-box tests.
- **OQ-07:** Wire details of channel compaction for the JSON protocol.
- **OQ-08:** The idle-connections blog calls `websocket.use_write_buffer_pool` and `websocket.read_buffer_size` PRO options, but the OSS docs document them. Treat them as OSS?
- **OQ-09:** The full `distributed_rate_limit` configuration block. Only `.enabled` and `.storage_type` (=redis) appear. Redis address and reuse keys are undocumented.
- **OQ-10:** ClickHouse table DDL.
  - analytics.md shows only `SHOW CREATE TABLE` commands, not the column lists. Exact columns are known only for the v6.8.2 additions.
  - Need column lists, engines and partition keys.
- **OQ-11:** Admin OIDC details are not documented:
  - state/nonce handling, logout, `cookie_domain` (appears only in RN v6.10.0), session expiry
  - the CSRF model for server-side flow
  - viewer access to tracing and config
- **OQ-12:** The `info` API fields for PRO CPU/RSS process stats.
- **OQ-13:** Status of `update_push_status`. It is marked experimental with no promotion note. Treat it as experimental.
- **OQ-14:** Client JWKS provider cache behavior: refresh interval, TTL, behavior when the endpoint is unavailable, allowed algorithms. Only the server API JWKS behavior is documented.
- **OQ-15:** Label key/value validation and size limits, and the max number of labels.
- **OQ-16:** Push device ID format. It "embeds timestamp and provider" but the format is not public. We need our own format; compatibility is impossible to verify without black-box tests.
- **OQ-17:** Push `rate_limit.key` variable syntax. The docs say `device.id` and `device.user`; is it template syntax?
- **OQ-18:** The `connections` method: does it have limits or a max result size? Is it exposed in the GRPC API proto?
- **OQ-19:** Channel state events: payload batching size, retry caps, queue bound sizes, the GRPC message schema (proxy.proto is OSS public; verify `NotifyChannelState` is present at OSS v6.10.3).
- **OQ-20:** Release notes mention other keys with no docs:
  - `client.queue_initial_cap` (RN v6.10.0)
  - removed undocumented top-level push `cache` (RN v6.10.0)
  - `emulated_headers` (OSS, RN v6.9.0)
  - Should we support these?
- **OQ-21:** Rate-limit metric naming. `centrifugo_rate_limit_hits_over_limit` (distributed limiter) and `centrifugo_rate_limit_client_over_limit_count` both exist. Confirm the labels of the former (none are documented).
- **OQ-22:** Conflict for channel state events prerequisites. The old page (channel_state_events) says the redis engine is required; event_hooks.md also allows a separate redis presence manager. Proposed: use event_hooks.md (newer).
- **OQ-23:** Dictionary compression (PRO-30) and the `profile` JWT/proxy field are unreleased. Confirm they are excluded from scope.
- **OQ-24:** Minimum versions marked "≤6.0.0" predate the scanned range. Do we need the v5.x release notes for history, or is "≤6.0.0" enough?
