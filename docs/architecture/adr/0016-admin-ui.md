# ADR-0016: Admin UI and admin API

- Status: revised 2026-11-15 (DEC-01, DEC-04, DEC-07)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The OSS admin backend (`internal/admin`) is inherited. The embedded upstream UI (`internal/webui`, built from `centrifugal/web`, MIT, branded) is removed from our build in Phase 1. Until our UI exists, `admin.web_path` can serve a custom build. Our own UI (`web/admin`) talks only to the inherited OSS admin endpoints, documented PRO endpoints, and `/admin/x/v1/*` extensions (ADR-0025).

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-ADM-01, PRO-02, PRO-05..08, PRO-37

## Context
The upstream admin UI is a separate repo (`centrifugal/web`). Its assets and text must not be copied (CLAUDE.md). The compatibility surface is the admin endpoints (`/admin/auth`, `/admin/api`, `/admin/trace`, `/admin/oidc/callback`) and their auth modes: password, `admin.secret` token, API key, OIDC.

## Decision
- A separate frontend package `web/admin`: React + TypeScript + Vite, our own design system. The build output is embedded with `go:embed`, and `admin.web_path` serves a custom build.
- The backend enforces auth and RBAC on **every** endpoint. The viewer role is read-only at the handler level, via a single middleware with an allowlist of non-mutating actions and a deny-by-default test.
- OIDC: `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`. Authorization Code + PKCE, state and nonce, a server-side flow option, encrypted HttpOnly Secure cookies, CSRF tokens on mutations. Some of this goes beyond the docs (PRO-OQ-11) and is recorded as hardening, not as a compatibility claim.
- Audit: a structured log event `component=admin_audit` (documented), plus an optional durable sink (our extension).
- The admin API schema is versioned and documented by us; compatibility with upstream admin internals is not promised (CORE-OQ-22).

## Sources
docs/server/admin_web.md, docs/pro/admin_ui.md.
