# Cedrazeka

- `SPEC.md` — full technical specification and OSS/PRO feature inventory.
- `CLAUDE.md` — mandatory instructions for Claude Code.

Start Claude Code in this directory. It must read every Markdown file before changing code, establish a pinned compatibility baseline, build the feature matrix, and review the AI tooling catalogue before installing anything.

The project is based on Centrifugo OSS v6.10.3 (Apache-2.0, decision DEC-01). PRO-equivalent features are written independently from public documentation. It must not copy proprietary source code or bypass commercial licensing.

## Phase 0 outputs

- Baseline: `docs/compatibility/baseline.md` (OSS v6.10.3, PRO v6.10.3, protocol v0.23.1)
- Feature matrix: `docs/compatibility/feature-matrix.md`; known differences: `docs/compatibility/known-differences.md`
- Research: `docs/research/` (source register, open questions with user decisions DEC-01…09, OSS/PRO annexes)
- Architecture: `docs/architecture/overview.md`, ADRs in `docs/architecture/adr/`
- Security: `docs/security/threat-model.md`; AI/dev tooling: `docs/ai/tooling-lock.md`
- Decisions DEC-01…09 (2026-11-15): ADR-0002, 0014, 0021–0027; compatibility suite ADR-0028; upstream sync ADR-0029; license ADR-0030 (Apache-2.0)
- Licensing/provenance: `docs/legal/provenance.md`, `docs/legal/naming-and-trademarks.md`
- Compatibility suite spec: `docs/testing/compatibility-suite.md`; phase 1 plan: `docs/plan/phase-1.md`

## Origin and license

Cedrazeka is based on [Centrifugo OSS](https://github.com/centrifugal/centrifugo) v6.10.3 by Centrifugal Labs LTD, licensed under the Apache License 2.0. The imported base is recorded in `upstream/BASE`. Every borrowed component and every change to upstream files is listed in `docs/legal/provenance.md`. Cedrazeka is not affiliated with or endorsed by Centrifugal Labs LTD.

Cedrazeka is licensed under the Apache License 2.0 (`LICENSE`). It is free to use, modify and redistribute, and it is provided "AS IS", without warranty or liability (sections 7–8). Attribution notices are in `NOTICE`, and third-party license texts are in `THIRD_PARTY_LICENSES`.

The upstream README is available in the import commit and at the upstream repository.
