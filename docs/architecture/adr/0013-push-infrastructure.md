# ADR-0013: Push notification infrastructure

- Status: proposed
- Date: 2026-11-15
- Feature IDs: PRO-01.*

## Decision
- Components:
  - **Registry**: PostgreSQL devices, device topics and user topics; cursor pagination as documented.
  - **Queue**: `JobQueue` interface with Redis Streams (default) and PostgreSQL implementations, consumer concurrency from config.
  - **Scheduler**: `send_at`, `cancel_push(uid)` before processing.
  - **Expander**: filter → device pages; templates, localizations, timezone windows, rate limits.
  - **Provider adapters**.
- Provider adapters are our own minimal HTTP clients, not vendor SDKs, to control timeouts, batching and secrets:
  - FCM HTTP v1 with OAuth2 service account via `golang.org/x/oauth2/google`;
  - APNs over HTTP/2 with token (ES256 JWT) or certificate auth;
  - HMS OAuth + push endpoint;
  - Web Push using RFC 8291 encryption and VAPID (RFC 8292), via a vetted library (`SherClockHolmes/webpush-go`, MIT) **or** our own implementation, decided after a security review.
- Web Push SSRF protections are mandatory and on by default: https only, origin allowlist, no IP literals, private-address refusal at dial time, no redirects. This is documented upstream.
- Invalid-token cleanup follows the documented provider codes. Retries use backoff until `expire_at` (default 24h). No DLQ (not documented upstream); dropped jobs are counted and logged.
- Templating uses `text/template` with the documented `[[...]]` literal delimiters and an explicit variable allowlist, bounded by a size limit.

## Sources
docs/pro/push_notifications.md, PRO release notes v6.8.2–v6.10.3.
