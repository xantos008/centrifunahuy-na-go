# ADR-0019: Distributed architecture

- Status: proposed
- Date: 2026-11-15

## Decision
- Nodes are stateless with respect to each other. All shared state (PUB/SUB, history, presence, map state, control) lives in backends. Any node can serve any client, so there are no sticky sessions, except emulation, which is routed by `node` through the controller.
- Failure model:
  - A backend outage degrades the features that depend on it: publish fails with 100 temporary, recovery reports `recovered=false`, and presence goes stale up to its TTL (documented ~1 min).
  - Clients reconnect with SDK backoff.
  - Timeouts, retries with jitter and circuit breaking apply per adapter.
- Scaling:
  - consistent client-side sharding across Redis setups, Redis Cluster, and PRO sharded PUB/SUB partitions;
  - NATS clustering;
  - PG replicas plus fan-out.
- Consistency is per channel: ordered, eventually consistent presence, and eventually consistent node registry and revocation sets (propagation delay documented as a bound in config).
- No in-process consensus.

## Sources
docs/getting-started/design.md, docs/server/engines.md, docs/pro/scalability.md.
