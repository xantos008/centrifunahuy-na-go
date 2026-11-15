# ADR-0008: Stream positions, history and recovery

- Status: superseded by ADR-0002 (inherited)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
History, positions and the subscribe/recover race handling are inherited from centrifuge. We do not redesign the algorithm. The recovery matrix becomes compatibility-suite scenarios (no loss, no duplicates, correct `recovered`, epoch change, cache mode) against `ref`. PRO-20 (singleflight, shared position sync) is added on top.

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-HIST-*, OSS-SUB-08, OSS-PROTO-15, PRO-20

## Context
Per-channel streams have an `epoch` (a string that changes when the stream is lost or reset) and a monotonic `offset`, with `history_size`, `history_ttl` and `history_meta_ttl`.

Recovery can end in three ways:
- it returns the missed publications;
- it reports `recovered=false` (gap, epoch change, or limit exceeded);
- it returns the latest publication (cache mode).

Positioning detects loss in the at-most-once PUB/SUB layer through periodic position checks. The relevant options are `channel_position_check_delay` and `channel_position_max_time_lag`; their semantics are CORE-OQ-12.

## Decision
- Positions come from the history storage atomically with the append (memory: a mutex-protected ring; Redis: our own Lua scripts / stream commands; PostgreSQL: a transaction).
- Subscribe-with-recover algorithm:
  1. Subscribe the node to the broker channel first.
  2. Start buffering live publications for that channel/client.
  3. Read history since the client's position.
  4. Merge by offset, dropping duplicates.
  5. Flush, then switch to live.
  Both the subscribe→history race and the merge are property-tested with injected interleavings.
- Shared position sync and singleflight (PRO-20) are optional wrappers around `Broker.History`/`Presence`, keyed by channel. They are a pure optimization and do not change semantics.
- No exactly-once delivery: at-least-once within the history window, duplicates suppressed by offset on the client side as upstream does.

## Sources
docs/server/history_and_recovery.md, cache_recovery.md, design.md, docs/pro/scalability.md.
