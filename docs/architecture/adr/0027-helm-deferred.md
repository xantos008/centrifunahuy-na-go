# ADR-0027: Helm chart deferred until the core is stable — DEC-09

- Status: **accepted** (user decision DEC-09, 2026-11-15)
- Date: 2026-11-15

## Decision
- No Helm work, and no research of `centrifugal/helm-charts`, until the core is stable. That means UPSTREAM rows for engines, transports and server API are `verified` by the compatibility suite (end of implementation phase 3).
- Until then, deployment artifacts are limited to a Dockerfile and a `docker-compose` setup for development and the compatibility suite.
- OQ-OPS-01 stays open and is re-opened at that milestone.

## Sources
User decision DEC-09.
