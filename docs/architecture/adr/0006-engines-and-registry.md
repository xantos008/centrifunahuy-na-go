# ADR-0006: Broker / PresenceManager / MapBroker interfaces and engine registry

- Status: revised 2026-11-15 (DEC-01)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The engine interfaces are the inherited centrifuge ones: `Broker`, `PresenceManager`, `MapBroker`, `Controller` (see overview §7 for the verified method sets). We define no parallel interfaces. The per-namespace registry (PRO-21) is built on `centrifuge.Config.GetBroker` / `GetPresenceManager` / `Map.GetMapBroker`. The capability validation idea is kept, as a config-time check in `internal/pro/engines`. The shared engine contract test suite is kept, as compatibility scenarios run per backend (ADR-0028).

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-ENG-*, PRO-21, PRO-26, PRO-32, PRO-33, PRO-40

## Context
OSS has an `engine` section (memory|redis), separate `broker` and `presence_manager` sections, and a `map_broker` section. PRO adds named `brokers[]`, `presence_managers[]` and `map_brokers[]` selected per namespace. Backends differ in capability; for example, NATS has no history and PostgreSQL has no presence manager. The full matrix is in feature-matrix §Engine capabilities.

## Decision
- Small interfaces, split by responsibility:
  - `Broker`: Publish, Subscribe, Unsubscribe, History, RemoveHistory, and PublishJoin/PublishLeave.
  - `PresenceManager`: Add, Remove, Presence, Stats.
  - `MapBroker`: Publish, Remove, ReadState, ReadStream, Stats, Clear.
- A capability descriptor per implementation, validated at config load. For example, `history` on a namespace routed to a NATS broker is a startup error, which mirrors the docs' constraints.
- `EngineRegistry` maps namespace → named broker/presence/map broker. Single-engine OSS configs are the degenerate case (one entry named `default`). Building it in from Phase 1 costs one map lookup and avoids a later refactor for PRO-21.
- Adapters are shipped in this order: memory (Phase 1) → Redis → NATS → PostgreSQL (Phase 3), each with the shared contract test suite (`internal/engine/enginetest`).

## Alternatives
- A single fat `Engine` interface (the historical shape): rejected. It forces fake implementations for capabilities a backend does not have.

## Sources
docs/server/engines.md, docs/pro/scalability.md, docs/server/map_subscriptions.md.
