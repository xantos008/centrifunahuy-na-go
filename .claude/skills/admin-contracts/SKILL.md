---
name: admin-contracts
description: Build Cedrazeka admin APIs and a distinct admin interface for roles, audit, snapshots or configuration.
---

# Admin contracts

Read ADR-0016/0025 and the threat model. Keep documented compatibility routes and Cedrazeka extension routes under the contracts in ADR-0025. Enforce role checks and audit in the backend, including export and pagination endpoints. Use independent visual assets and wording. Exercise viewer and operator flows with development data and verify redaction and partial-result behavior.
