---
name: extension-architecture
description: Design Cedrazeka extensions, map upstream behavior to code, or review architectural boundaries and migration.
---

# Extension architecture

Start with `docs/architecture/overview.md`, ADR-0029 and the relevant feature-matrix row. Trace one real path from transport or API to policy and storage, including ownership, failure and shutdown. Follow the existing extension seams and keep upstream compatibility verification alongside the feature. For a material decision use the repository ADR template and identify rollback and data migration. Public documentation and black-box observations are the evidence for independently implemented PRO behavior.
