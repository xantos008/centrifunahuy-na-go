# Cedrazeka — project bootstrap

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
