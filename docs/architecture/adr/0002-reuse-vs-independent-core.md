# ADR-0002: Build on Centrifugo OSS v6.10.3 (fork) — DEC-01

- Status: **accepted** (user decision DEC-01, 2026-11-15)
- Date: 2026-11-15
- Supersedes: the Phase 0 recommendation "A: independent code" in this ADR
- Feature IDs: all `OSS-*` rows (category UPSTREAM)

## Context
Licenses verified in Phase 0 (LICENSE files at the pinned commits):
- `centrifugal/centrifugo` (server) is Apache-2.0, "Copyright 2023 Centrifugal Labs LTD". There is **no NOTICE file** at v6.10.3.
- `centrifugal/centrifuge` (core library, a Go module dependency of the server) and `centrifugal/protocol` are MIT.
- Centrifugo PRO is proprietary. Its public repo holds only README and SECURITY.

## Decision
- The project is a **fork of Centrifugo OSS v6.10.3** (tag `v6.10.3`, commit `4b6d86e23452a6fa1bd424ffe4526862efae85b8`). We do not rewrite the OSS server from scratch.
- `github.com/centrifugal/centrifuge` stays an **unmodified Go module dependency**, at the exact pseudo-version pinned in upstream `go.mod`. It is forked into the repo (`third_party/centrifuge` + a `replace` directive) only when a PRO feature needs a change that its public extension points cannot provide. Each such fork is recorded in [provenance.md](../../legal/provenance.md) and in an ADR.
- PRO-equivalent features are written by us from public docs only. PRO code and binaries are never used (unchanged from ADR-0001).
- Apache-2.0 and MIT obligations are kept for every imported file:
  - license texts and copyright notices are kept;
  - modified files carry a change notice (Apache-2.0 §4(b));
  - attribution goes in `NOTICE` and `THIRD_PARTY_LICENSES`.
  
  The process is in [provenance.md](../../legal/provenance.md).
- Import method: one **pristine import commit** of the upstream tree at the tag, with no changes. All our changes come in later, separate commits. Upgrades to a new upstream tag are applied as `git diff vA..vB` from upstream with `git apply -3`. This keeps the upstream base reproducible and diffable (ADR-0029).

## Alternatives considered
| Option | Why not |
|---|---|
| A. Independent code, schemas only | Highest compatibility risk and effort. Rejected by the user. |
| B. Our own server on the `centrifuge` library | We would re-implement config, API, proxies, consumers and engines that OSS already provides under Apache-2.0. |
| D. Git merge of the full upstream history | Imports about 10 years of history and branding into our log. A pristine import commit plus upstream tag diffs gives the same reproducibility. Can be revisited. |

## Consequences
- OSS compatibility becomes "inherited, then proven": UPSTREAM rows in the feature matrix start as `inherited` and reach `verified` only through the compatibility suite (ADR-0028).
- Several Phase 0 ADRs that assumed independent code are revised: 0003–0011, 0016–0018 and 0020. Each now has a "Revision (DEC-01)" section.
- The CLAUDE.md "Mission" wording ("independent, clean-room") no longer matches this decision. Clean-room now applies **only to PRO**. CLAUDE.md was updated accordingly (DEC-11).
- The project license must be compatible with the Apache-2.0 base. See ADR-0030 (DEC-10).
- Upstream branding inside the code must be replaced under DEC-04 (ADR-0022) without touching license notices.

## Sources
`LICENSE` in centrifugo@v6.10.3, centrifuge@ae446d43faf0, protocol@v0.23.1. Apache License 2.0 §4. User decision DEC-01 (2026-11-15).
