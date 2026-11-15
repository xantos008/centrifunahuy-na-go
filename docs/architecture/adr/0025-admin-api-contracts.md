# ADR-0025: Admin API only from confirmed public contracts — DEC-07

- Status: **accepted** (user decision DEC-07, 2026-11-15)
- Date: 2026-11-15
- Feature IDs: OSS-ADM-01, PRO-02, PRO-05..08, PRO-37; KD-16

## Context
After DEC-01 the OSS admin backend (`internal/admin`: `/admin/auth`, `/admin/api`, password, `admin.secret` and API-key auth) is inherited open-source code. Its behavior is a confirmed public contract because the code is public. PRO admin features (OIDC, roles, audit, tracing, snapshots, config viewer, inspector) are documented only at the level of the docs pages.

## Decision
- The OSS admin endpoints are inherited and kept compatible. They are covered by the compatibility suite (category UPSTREAM).
- PRO admin features are implemented to the documented contract: config keys, roles `admin`/`viewer`, documented endpoints and events. They are category PRO.
- Anything we need beyond that (pagination, export, partial-result fields, durable audit sink, CSRF/state/nonce hardening, readiness) is category **EXTENSION**. It sits under our own versioned path prefix (`/admin/x/v1/...`) and is never described as compatible.
- The admin UI is our own (ADR-0016). It uses only the endpoints above.

## Sources
User decision DEC-07; docs/server/admin_web.md, docs/pro/admin_ui.md; OSS `internal/admin`.
