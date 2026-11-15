# ADR-0009: Redis adapter

- Status: superseded by ADR-0002 (inherited)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The Redis adapter is inherited (rueidis, upstream Lua and key layout). Our own key layout is cancelled. Mixed clusters stay unsupported (ADR-0023). The default key prefix changes under ADR-0022. PRO Redis features (replicas, sharded PUB/SUB, user status, block/revoke, rate limits) are new code that reuses the upstream shard abstraction (`internal/redisshard`).

The original Phase 0 text below is kept for history.

## Context
Required capabilities:
- standalone, Sentinel and Cluster;
- client-side consistent sharding across several Redis/Cluster setups;
- Redis-compatible stores (as documented);
- IAM auth (AWS/GCP/Azure, experimental);
- PRO: replica reads/subscribes, sharded PUB/SUB with partitions.

The broker, presence, history, map, rate limits, user status, block/revoke and the push queue all use Redis.

## Decision
- Client library: `github.com/redis/rueidis` (Apache-2.0): pipelining and auto-batching, RESP3, Cluster, sharded PUB/SUB. Fallback: `go-redis/v9`. Final choice by a benchmark spike in Phase 3.
- Lua scripts are written by us. The key schema is ours and documented in `docs/architecture/redis-keys.md` (Phase 3). We aim for operational equivalence, not key-level compatibility: mixing our nodes with reference nodes on one Redis is **not** supported (known-differences).
- One shared Redis "shard set" abstraction. Each subsystem (`engine.redis`, `user_status.redis`, …) gets its own configured set or reuses the engine's (`reuse_from_engine`), as the docs describe.

## Alternatives
- Key-level compatibility with the reference server: undocumented internals, high cost, no user value. Rejected unless requested.

## Sources
docs/server/engines.md, docs/pro/scalability.md; OSS go.mod shows rueidis (informational only).
