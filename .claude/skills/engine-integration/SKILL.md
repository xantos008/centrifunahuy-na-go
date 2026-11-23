---
name: engine-integration
description: Verify Cedrazeka Redis, Valkey, NATS and PostgreSQL topologies, storage behavior and recovery.
---

# Engine integration

Use `docs/plan/phase-2.md`, `docs/operations/engines.md` and the Compose files under `test/`. Keep ref and sut on independent networks with side-bound backend actions. Wait for observable convergence after a restart. Check delivery and history independently and distinguish broker delivery from controller discovery. Use `make library-test` for the pinned centrifuge library and record its baseline separately from server results.
