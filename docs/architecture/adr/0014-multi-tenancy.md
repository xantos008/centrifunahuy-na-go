# ADR-0014: Multi-tenancy model

- Status: **accepted** (user decision DEC-02, 2026-11-15): option 1
- Date: 2026-11-15

## Context
SPEC requires "tenant isolation in all layers". **Neither Centrifugo OSS nor PRO has a first-class tenant entity.** We searched docs/server and docs/pro at the baseline.

The docs do describe multi-tenant *patterns*:
- dynamic JWKS with `{{tenant}}` templated from `issuer_regex` (docs/server/authentication.md, ~l.592);
- PostgreSQL `table_prefix` for multi-tenant deployments sharing one instance (engines.md, map_subscriptions.md);
- key prefixes (`engine.redis.prefix`, `broker.nats.prefix`);
- namespaces;
- per-namespace engines (PRO-21);
- separate deployments.

A first-class tenant would be an extension beyond compatibility.

## Options
1. **Deployment-per-tenant, or a shared deployment using the documented patterns** (recommended for v1): separate processes, or shared processes with per-tenant prefixes, JWKS templating and namespaces. Zero protocol change; compatible.
2. **Namespace-as-tenant convention**: tenant = namespace (or pattern var) plus per-namespace engines, limits and RBAC. Compatible, but admin/analytics need tenant-scoped views (extension).
3. **First-class tenant ID** (in JWT claim and API key → tenant, propagated through every layer): strongest, but an incompatible extension that touches every interface.

## Decision (DEC-02)
Option 1 for v1: deployment-per-tenant. Isolation is by process, credentials and backend key prefixes or databases.

The architecture must not block options 2 and 3 later:
- No global singletons that assume one tenant per process beyond the composition root (`internal/app`).
- Every backend adapter already takes a prefix or schema (`engine.redis.prefix`, `broker.nats.prefix`, PG `table_prefix`), so co-location stays possible.
- New PRO subsystems (push registry, analytics, audit, admin) put an opaque `scope` column or key segment in their storage schemas from day one. It is constant (`"default"`) in v1. Cost: one column; it saves a data migration later.
- Admin RBAC is resource-scoped (role → actions on resource patterns), so tenant-scoped roles can be added without a redesign.

Original recommendation text:
Option 1 now. Keep `tenant` out of core interfaces. Revisit option 2 when the admin/analytics phases begin. Record it in known-differences.

## Sources
grep "tenant" over docs/ at commit 69da714: only the JWKS templating and `table_prefix` patterns above.
