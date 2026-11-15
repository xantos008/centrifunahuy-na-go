# ADR-0010: PostgreSQL usage

- Status: revised 2026-11-15 (DEC-01)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The OSS PostgreSQL broker, map broker, controller, outbox consumer and their schemas are inherited (pgx v5.12.0, upstream migrations in `internal/pgschema`). The in-house migration runner applies only to new PRO schemas (push registry/queue, block/revoke storage), reusing the upstream schema helper where it fits. Every new table carries the `scope` column (ADR-0014).

The original Phase 0 text below is kept for history.

## Context
PostgreSQL appears in several roles:
- OSS: broker (experimental, PG 16+, transactional publishing), controller (experimental), map broker, outbox consumer.
- PRO: device/topic registry, push queue, read replicas, broker fan-out via Redis/NATS, user block/revoke `database` storage, snapshot metadata.

## Decision
- Driver `github.com/jackc/pgx/v5` (MIT) with `pgxpool`. Pool size, timeouts and replicas come from config.
- Migrations are embedded SQL (`embed.FS`), applied with a small in-house runner (a version table plus an advisory lock). ponytail: no migration framework dependency until we need down-migrations.
- Notifications use LISTEN/NOTIFY where low latency is needed. Ordering and consistency come from table reads, so a lost NOTIFY is never data loss. Exact OSS PostgreSQL broker semantics are taken from engines.md during Phase 3 (see OSS-ENG rows).
- The schema is ours. Transactional publishing (an app inserting into our table inside its own transaction) **is** a public contract. Its table/function shape must match the docs, so it gets golden SQL tests.

## Sources
docs/server/engines.md#postgresql-broker, docs/server/consumers.md, docs/pro/push_notifications.md.
