# ADR-0007: Control plane (Controller)

- Status: revised 2026-11-15 (DEC-01)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
Uses the inherited `centrifuge.Controller` (`RegisterControlEventHandler`, `PublishControl(data, nodeID, shardKey)`). Default: the broker acts as controller when it implements the interface (Redis); OSS ships the PostgreSQL controller. PRO-22 adds Redis/NATS controller selection via `Node.SetController`. The control envelope is the upstream one. Snapshots and connections use `OnSurvey`.

The original Phase 0 text below is kept for history.
- Feature IDs: PRO-22, OSS PostgreSQL controller, OSS-TRANS-05

## Context
Nodes exchange control messages for:
- node info/heartbeats (`info` API, metrics aggregation);
- user-targeted server API commands (disconnect, subscribe, unsubscribe, refresh), which must reach nodes holding that user's connections;
- emulation requests routed to the owning node;
- snapshots (PRO-07);
- block/revoke propagation (PRO-15).

OSS ships a PostgreSQL controller (experimental). By default control messages go over the broker backend. PRO-22 allows a dedicated Redis or NATS controller.

## Decision
- A `Controller` interface: `Publish(ctx, msg, toNode string)`, `Subscribe(handler)`, plus node-registry helpers. The message envelope is our own versioned protobuf (internal, so not a compatibility surface).
- Default: use the same backend as the default broker. With the `controller` section, use a dedicated backend.
- Node liveness uses heartbeats with a TTL, and the node registry is eventually consistent. Commands are fire-and-forget broadcasts; there is no consensus. Snapshots aggregate with a deadline and report `nodes_responded/nodes_known` (our extension; not documented upstream, see known-differences).

## Alternatives
- Raft/etcd membership: unnecessary. The upstream model is eventually consistent and backend-mediated.

## Sources
docs/pro/scalability.md#setting-custom-controller, docs/server/engines.md (PostgreSQL controller), feature-matrix.
