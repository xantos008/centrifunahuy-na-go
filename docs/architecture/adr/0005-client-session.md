# ADR-0005: Client session, write path, batching and backpressure

- Status: superseded by ADR-0002 (inherited)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The session, write queue, batching and `3008 slow` behavior are inherited from centrifuge. PRO-24 and PRO-34 are wired through `GetChannelBatchConfig` and existing upstream options (ADR-0029).

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-PROTO-06, OSS-CLI-02, PRO-24, PRO-34

## Decision
- One reader goroutine per connection (the transport-owned read loop). It dispatches commands in order, or off the read loop when `websocket.process_commands_off_read_loop` is set (PRO-34).
- One writer goroutine per connection drains a **byte-bounded** queue (`client.queue_max_size`, default 1 MiB). Overflow gives disconnect `3008 slow`. The queue is never unbounded.
- Batching knobs map 1:1 to the documented config: `write_delay`, `write_with_timer`, `max_messages_in_frame`, `reply_without_queue`, `queue_shrink_delay`. Defaults reproduce OSS behavior (no delay).
- Pings are timer-driven per connection. ponytail: per-connection timers to start; a shared timing wheel if benchmarks at 1M connections show timer overhead.
- `client.concurrency` bounds concurrent command processing per connection.

## Alternatives
- A shared writer pool (an N:M scheduler): fewer goroutines, but more complex fairness. Revisit only with benchmark evidence.

## Sources
docs/server/configuration.md (client.*), docs/pro/client_msg_batching.md, docs/pro/performance.md.
