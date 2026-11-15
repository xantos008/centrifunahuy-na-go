# ADR-0012: ClickHouse analytics pipeline

- Status: proposed
- Date: 2026-11-15
- Feature IDs: PRO-03, PRO-07, PRO-01.19

## Context
The docs specify:
- export of connections, subscriptions, operations, publications and notifications;
- native TCP DSNs, optional cluster;
- per-export `max_buffer_size`, `flush_interval`, `flush_size`, `ttl`;
- dropping on a full buffer, with a drop metric;
- automatic schema creation and migration;
- snapshots stored in ClickHouse.

The exact upstream DDL is not public (PRO-OQ-10).

## Decision
- `github.com/ClickHouse/clickhouse-go/v2` (Apache-2.0), native protocol, batch insert.
- One bounded ring buffer per export type with a single flusher goroutine. A full buffer drops the newest row and increments `..._drop_count`. This is never blocking the hot path; it mirrors the documented behavior.
- Our own versioned schema: MergeTree, `PARTITION BY toYYYYMMDD(time)`, `TTL` from config, `*_distributed` tables when a cluster is set. Schema version table, forward-only migrations.
- PII controls: the documented `export_users` / `export_channels` / header allowlists, plus our own documented extension for field suppression (known-differences).

## Alternatives
- Kafka in front of ClickHouse: more infrastructure; not in the docs. Rejected.

## Sources
docs/pro/analytics.md, docs/pro/admin_ui.md.
