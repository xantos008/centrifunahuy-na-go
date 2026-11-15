# ADR-0001: Clean-room process and pinned compatibility baseline

- Status: accepted, revised 2026-11-15 (DEC-01)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
Clean-room now applies to **PRO only**. OSS v6.10.3 source is the code base of the project (ADR-0002), not just a reference. The pinned baseline, source register, status carry-over and the ban on PRO code/binaries are unchanged. The OSS reference binary is now the `ref` target of the compatibility suite (ADR-0028).

The original Phase 0 text below is kept for history.

## Context
We target protocol/API/behavior compatibility with Centrifugo OSS and functional equivalents of PRO features. The licenses differ:
- PRO is proprietary.
- OSS server (`centrifugal/centrifugo`) is Apache-2.0.
- `centrifugal/centrifuge` and `centrifugal/protocol` are MIT.
- The docs site is Apache-2.0.

## Decision
- Pin OSS v6.10.3, PRO v6.10.3 and protocol v0.23.1, plus the docs commit `69da714` (see `docs/compatibility/baseline.md`).
- PRO behavior comes only from public docs and release notes. We never use PRO binaries.
- Every behavior gets a source register entry. Undocumented behavior becomes an open question plus a black-box experiment against the OSS reference binary.
- Experimental/beta/preview status is carried over 1:1 and gated by feature flags. WIP/draft features are excluded.

## Alternatives considered
- "Latest docs, no pin": rejected. The docs change weekly (the docs commit is newer than the release), so behavior would not be reproducible.

## Consequences
Re-baselining is an explicit ADR. Feature rows record their source commit.

## Sources
centrifugo-pro README (versions may differ); LICENSE files of the four repos (read 2026-11-15).
