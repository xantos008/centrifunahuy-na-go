# Threat model (Phase 0, pre-implementation)

Method: list the trust boundaries, then STRIDE per boundary, then mitigations that map to feature rows and tests. Review this again at the end of every phase.

## Assets

| Asset | Sensitivity |
|---|---|
| Token secrets and keys (HMAC secrets, RSA/EC/Ed25519 keys, JWKS), API keys, `admin.secret`/password, OIDC client secret | critical |
| Push provider credentials (FCM service account, APNs .p8/.p12, HMS secret, VAPID private key) | critical |
| Backend DSNs (Redis, PG, NATS, ClickHouse, Kafka, SQS, Azure, GCP) | critical |
| Publication payloads, presence (who is online), connection metadata (`info`, `meta`, labels, headers) | PII / customer data |
| Device tokens, user topics, timezones, locales | PII |
| Analytics and snapshots in ClickHouse; trace streams | PII, high volume |
| Revocation/block state, audit log | integrity-critical |
| Availability of fan-out | business-critical |

## Trust boundaries

1. **Internet clients → transports.** Untrusted. Sources: WS, SSE, HTTP-stream, emulation, WebTransport, uni transports, uni gRPC.
2. **App backend → server API.** HTTP `/api`, gRPC API. Semi-trusted: authenticated by key or JWKS.
3. **Server → app backend (proxies).** HTTP/gRPC to configured endpoints. The targets come from configuration.
4. **Server → backends.** Redis, PG, NATS, ClickHouse, queues.
5. **Server → external providers.** FCM, APNs, HMS, Web Push endpoints (**user-supplied URLs**), JWKS URLs (possibly templated from token claims), OIDC issuer.
6. **Operators → admin UI/API.** Password/token/API key/OIDC; roles admin and viewer.
7. **Config and env → process.** Trusted, but contains secrets.
8. **Supply chain.** Go modules, frontend npm packages, container base image, CI, AI tooling.

## STRIDE by boundary

| Boundary | Threat | Mitigation | Feature / test |
|---|---|---|---|
| 1 | Spoofing: forged or replayed JWT, `alg=none`, alg confusion (HS vs RS) | Algorithm allowlist bound to key type; reject `none`; check exp/nbf (upstream maps nbf to expired, INFRA-CONFLICT-10), aud/iss when configured; revocation by `jti`/`iat` (PRO-15) | OSS-AUTH, PRO-15; auth matrix tests, JWT fuzz |
| 1 | Tampering: malformed frames, duplicate JSON keys, invalid UTF-8 | Strict decoder (upstream since 6.10.0) → 3501; size limits before decode | OSS-PROTO-05; codec fuzz |
| 1 | DoS: connection floods, huge frames, decompression bombs, slow readers, subscription floods, expensive history/presence calls | `connection_limit`, `connection_rate_limit`, `message_size_limit`, `decompressed_message_size_limit`, byte-bounded write queue → 3008, `channel_limit`, `client.concurrency`, history/recovery limits (300), PRO rate limits | OSS-TRANS-01, OSS-CLI-02, PRO-04; slow-consumer and bounded-memory tests |
| 1 | Elevation: subscribing to private or user channels | Decision pipeline per channel_permissions doc, fail closed; caps grant-only; CEL fail closed | OSS-PERM, PRO-17/19; decision-table tests |
| 1 | Information disclosure: cross-origin WebSocket hijacking | `allowed_origins` | OSS-ORIG-01 |
| 1 | Server-side filter bypass | Server tags filter enforced server-side for live, recovery and cache (never rely on the client filter) | PRO-25 |
| 2 | Spoofing: API without auth (gRPC API has **no auth by default** upstream) | Keep the default for compatibility; WARN at start when bound to a non-loopback address without a key; JWKS auth option | OSS-API-02, PRO-12, KD-25 |
| 2 | DoS / mistakes: `all_users` or broad `label_filter` fan-out | Explicit flag required (upstream); bounded executor concurrency; metrics | PRO-12.02 |
| 3 | SSRF via proxy endpoints | Endpoints come only from config, never from client input; timeouts; header allowlist (`emulated_headers`, split for security in 6.9.0) | OSS-PROXY |
| 3 | Credential leakage in forwarded headers | Forward only configured headers; never forward Authorization unless configured | OSS-PROXY-11 |
| 4 | Credential exposure in logs, errors or config viewer | Central redaction for DSN userinfo, keys and tokens; config viewer masks server-side | PRO-08, OBS |
| 4 | Backend outage leading to unbounded buffering | Bounded queues; drop and count metrics (ClickHouse, channel state events); timeouts and circuit breaking | PRO-03, PRO-16 |
| 5 | **SSRF through Web Push endpoints** (user-supplied URLs) | https only, origin allowlist, no IP literals, private/loopback/link-local refusal at dial time, no redirects (documented upstream, always on) | PRO-01.06 |
| 5 | SSRF through JWKS URLs (templated from claims, OSS dynamic JWKS) | Upstream template safety rule, plus our hardening KD-20 (redirect, size and IP limits) | OSS-AUTH (JWKS), PRO-10 |
| 5 | Provider credential theft | Load from file/env/secret manager; never logged; never sent to the UI | PRO-01 |
| 5 | Template injection in push templating | `text/template` with a fixed variable set, no functions beyond an allowlist, size limit | PRO-01.16 |
| 6 | Viewer mutating state; missing per-request checks | One authorization middleware, deny by default, per-action allowlist; tests enumerate every admin route | PRO-06 |
| 6 | OIDC: code interception, CSRF, session fixation | PKCE, state/nonce, Secure HttpOnly SameSite cookies, rotation on login, CSRF tokens | PRO-05, KD-21 |
| 6 | Trace and snapshot data exposure | RBAC, bounded trace buffers, redaction of tokens in frames | PRO-02, PRO-07 |
| 6 | Repudiation | Audit every privileged mutation (`component=admin_audit`), optional durable sink | PRO-06, KD-22 |
| 7 | Secrets committed in config examples | Examples use placeholders; `.env` is gitignored; secret scanning in CI | Phase 9 |
| 8 | Malicious dependency or AI tool | Pinned modules (`go.sum`), govulncheck, SBOM, minimal deps; AI tools vetted and locked (docs/ai/tooling-lock.md) | OSS-OPS |
| 8 | Tampered upstream import or upgrade diff (DEC-01) | Fetch by tag, verify the peeled commit hash against `upstream/BASE`; pristine import commit; upgrade diffs reviewed like our own code; reference image pinned by digest | ADR-0002, ADR-0029 |
| 7 | Compatibility mode silently changes env/metric names in production | Mode only by explicit flag or env; logged at startup; CS-NAME-01 | ADR-0022 |

## Resource and abuse limits (checklist, every limit needs a config key, a metric and an error)

Payload size; channel name length (255); channels per connection (128); connections per user; queue bytes (1 MiB); labels (count and size, KD-24); `meta` size; history page (300); presence response size; API batch size; `broadcast` channel count; consumer batch; CEL cost; push batch sizes (FCM 500, HMS 1000, APNs/WebPush 100 per docs); ClickHouse buffer (1e6 rows); trace buffer.

## Tenant isolation

Upstream has no first-class tenant; only patterns (ADR-0014). Per DEC-02 (deployment-per-tenant in v1), isolation means:
- a deployment per tenant;
- distinct key prefixes;
- separate credentials.

## Open security questions

INFRA OQ-AUTH-01 (clock skew), OQ-AUTH-02 (JWKS SSRF), PRO-OQ-11 (OIDC session/CSRF), PRO-OQ-14 (client JWKS cache), PRO-OQ-15 (label limits).
