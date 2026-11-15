# ADR-0023: No mixed clusters with Centrifugo nodes — DEC-05

- Status: **accepted** (user decision DEC-05, 2026-11-15)
- Date: 2026-11-15
- Feature IDs: OSS-ENG-*, PRO-21/22/26/40; KD-10

## Context
After DEC-01 our nodes start with the upstream Redis key, Lua, NATS subject and PG table layout. That layout is an internal detail. It is undocumented and can change in any upstream release, and our PRO modules will add their own keys and control messages.

## Decision
- Running our nodes and Centrifugo nodes against the same broker, presence, controller or map backend is **not supported**, in any mode.
- Default key prefixes differ from upstream (`cedrazeka` vs `centrifugo`, ADR-0022), so accidental sharing is unlikely in default mode. In compatibility mode an operator could point both at one prefix. The docs warn against it, and no detection is promised.
- Our internal control-message envelopes may diverge from upstream without an ADR.
- The exception is the public SQL surface for PostgreSQL transactional publishing (KD-11). It is a documented application contract, not inter-node state, and it stays compatible.

## Alternatives considered
- Support mixed clusters: requires freezing an undocumented layout across upstream upgrades. Rejected.

## Consequences
Migrating from Centrifugo is a blue/green cutover: clients reconnect and recover from history only if history lives in a backend both sides read. That is not supported, so recovery across a cutover gives `recovered=false`. This is documented in operations (Phase 9).

## Sources
User decision DEC-05; OQ-ENG-01.
