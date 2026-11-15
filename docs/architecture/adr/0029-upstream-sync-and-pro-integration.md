# ADR-0029: Upstream sync and how PRO modules attach to the fork

- Status: proposed
- Date: 2026-11-15
- Feature IDs: all PRO rows

## Context
Every line changed in an upstream file makes the next upstream merge more expensive. PRO features need hooks into the server.

The `centrifuge` library already exposes public extension points that match several PRO features (verified at `ae446d43faf0`):
- `Config.GetBroker`, `GetPresenceManager`, `Map.GetMapBroker`: per-channel engine choice (PRO-21, PRO-26);
- `GetChannelBatchConfig`: PRO-24;
- `GetChannelMediumOptions`: PRO-20;
- `Node.SetController`: PRO-22;
- `OnSurvey`, `OnNotification`: cross-node snapshots, connections and user status (PRO-07, PRO-13, PRO-14);
- `OnCommandRead`, `OnCommandProcessed`, `OnTransportWrite`: tracing, rate limits, granular metrics (PRO-02, PRO-04, PRO-09).

## Decision
- **Placement.** PRO code lives in new packages `internal/pro/<module>/` (for example `internal/pro/ratelimit`, `internal/pro/push`). Upstream packages stay where they are.
- **Wiring.** Modules are wired in one place: the upstream composition root `internal/app`, through a small set of calls marked `// cedrazeka:pro`. Hooks are preferred in this order:
  1. a `centrifuge` public extension point;
  2. a new optional field or interface in an upstream package;
  3. a direct edit of upstream logic.
  
  Options 2 and 3 are listed in provenance.md.
- **Config.** PRO config keys are added to upstream `configtypes` with the documented PRO key names, so `defaultconfig`, `configdoc` and the config viewer pick them up.
- **Upstream sync.**
  - `upstream/BASE` records the imported tag and commit.
  - An upgrade is one branch: apply `git diff <old-tag>..<new-tag>` from upstream with `git apply -3`, resolve conflicts, update baseline.md, provenance.md and the feature matrix, re-record golden transcripts from the new `ref`, and land the result with an ADR.
- **Library fork trigger.** If a PRO feature needs a `centrifuge` change, first check whether upstream `centrifuge` has or accepts the hook. Otherwise vendor the library into `third_party/centrifuge` with a `replace` directive (MIT notice kept) and record it in an ADR.

## Alternatives considered
- Patching PRO behavior directly into upstream files: fastest at first, worst merge cost. Rejected as the default.
- A runtime plugin system (Go `plugin`, gRPC plugins): fragile and unneeded, since we ship one binary. Rejected.

## Sources
centrifuge `config.go` l.205–430, `node.go` l.274–300 and l.2060–2128 (commit ae446d43faf0).
