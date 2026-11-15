# Compatibility baseline

Pinned 2026-11-15. Changing any row requires an ADR and a re-run of the Phase 0 verification for affected features.

**Role of the baseline after DEC-01 (2026-11-15):** OSS v6.10.3 is both the behavior reference *and* the code base of the project. The project is a fork (ADR-0002). PRO v6.10.3 is a behavior reference only, from public docs (ADR-0024). Category definitions (UPSTREAM / PRO / EXTENSION) are in [feature-matrix.md](feature-matrix.md).

| Component | Pinned version | Released | Source / commit |
|---|---|---|---|
| Centrifugo OSS (reference server, behavior) | **v6.10.3** | 2026-11-08 | https://github.com/centrifugal/centrifugo/releases/tag/v6.10.3 — repo HEAD read: `4b6d86e23452a6fa1bd424ffe4526862efae85b8` (2026-11-08) |
| Centrifugo PRO (public docs + release notes only) | **v6.10.3** | 2026-11-08 | https://github.com/centrifugal/centrifugo-pro/releases/tag/v6.10.3 — release notes state it is built on OSS v6.10.3 |
| Client protocol schema | **centrifugal/protocol v0.23.1** | — | go.mod of OSS v6.10.3; repo HEAD read: `4455d8ea79cfd62206cc4ee6f32025da63fa1654` |
| centrifuge (Go library used by OSS) | v0.40.2-0.20261031074606-ae446d43faf0 (pseudo-version) | 2026-10-31 | go.mod of OSS v6.10.3; latest tag v0.40.1; repo HEAD read `8888c6eee1c4e3a390d2fac511ccee19456f4eb4` |
| Reference image (compatibility suite `ref`) | `centrifugo/centrifugo:v6.10.3` | 2026-11-08 | digest `sha256:33fdbbce7d195dde316cbde17bc1d2a17b9e345de82799d1ec3c2bb8916e3803` (verified 2026-11-15) |
| Official Go SDK (compatibility suite client) | centrifuge-go **v0.13.0** (MIT) | 2026-10-21 | https://github.com/centrifugal/centrifuge-go |
| Documentation | centrifugal.dev `docs/` (current = v6) | 2026-11-13 | https://github.com/centrifugal/centrifugal.dev commit `69da71466274f741bfe924bbf4a62a9124ec5776` |

OSS and PRO currently share the version number v6.10.3; this is a coincidence of the release process, not a rule (the PRO README states versions may differ).

## Scope rules

- **In GA scope:** features documented as GA in the pinned docs.
- **Separate status, behind feature flags:** features the docs mark `experimental`, `beta` or `preview`. Examples: map subscriptions and shared poll (the whole feature), channel state events (preview), snapshots (beta), multi-JWKS and `meta_from_claim` (beta), channel compaction (beta), flight recorder (experimental), `update_push_status` (experimental), node-grouped sharded PUB/SUB (experimental), `client.batch_periodic_events` (experimental). We implement them with the same status label.
- **Out of scope until officially released:**
  - shared poll relay (WIP, "not officially supported");
  - dictionary compression (`draft: true`, unreleased);
  - ephemeral client publications (unlisted page). Pending decision: PRO-OQ-01.
- **Excluded by decision:** bidirectional gRPC client transport (DEC-03), mixed clusters with Centrifugo nodes (DEC-05), undocumented PRO behavior (DEC-06), Helm until the core is stable (DEC-09).
- **Never in scope:** PRO licensing (license keys, sandbox limits, `centrifugo_license_expiration_timestamp_seconds`), Centrifugo branding, PRO binaries (not downloaded, run or decompiled).
- **Deprecated in baseline:** implement only when needed for config compatibility, and log deprecation warnings. Examples: Summary metrics (removal planned in v7), the legacy `method_override` map. Removed upstream and not implemented: SockJS and Tarantool (removed in v6).

## Reference behavior

The compatibility suite (ADR-0028) runs against the pinned image above and official open-source SDKs. Upstream source is also read and inherited (Apache-2.0). PRO-only behavior is specified from public docs alone; ambiguities go to `docs/research/open-questions.md` and are never filled with guesses.

## Re-baselining

1. Diff the docs between the pinned commit and the new commit: `git diff <old>..<new> -- docs/`.
2. Read the OSS and PRO release notes in between.
3. Apply the upstream code diff `vOLD..vNEW` (ADR-0029), update `upstream/BASE` and provenance.md.
4. Re-pin the reference image digest, then re-record golden transcripts (`make compat-record`) and review the diff.
5. Update the source register, feature matrix and known differences.
6. Record the change in a new ADR.
