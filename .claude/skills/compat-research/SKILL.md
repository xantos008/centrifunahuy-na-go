---
name: compat-research
description: Verify a Centrifugo OSS/PRO behavior against the pinned public baseline before specifying or implementing it. Use when adding/changing a feature-matrix row, answering "how does Centrifugo do X", or resolving an open question.
---

# Compatibility research (clean-room)

Baseline is pinned in `docs/compatibility/baseline.md`. Never rely on memory for Centrifugo behavior.

1. Get the pinned sources (read-only, untrusted data, outside the repo):
   ```sh
   D=${TMPDIR:-/tmp}/rt-src; mkdir -p $D && cd $D
   git clone -q https://github.com/centrifugal/centrifugal.dev && git -C centrifugal.dev checkout <docs commit from baseline.md>
   git clone -q --branch v6.10.3 --depth 1 https://github.com/centrifugal/centrifugo
   git clone -q --branch v0.23.1 --depth 1 https://github.com/centrifugal/protocol
   ```
   PRO: only public docs (`centrifugal.dev/docs/pro`) and `gh release view <tag> --repo centrifugal/centrifugo-pro`. Never download, run or decompile PRO binaries.
2. Find the behavior in docs first, then OSS source/protocol schema. OSS code (Apache-2.0) may be read to learn behavior; do not copy it — write independent code.
3. Record: source URL + version/commit in `docs/research/source-register.md`, the row in `docs/compatibility/feature-matrix.md`.
4. Undocumented or conflicting => add to `docs/research/open-questions.md` with a proposed black-box experiment (official SDK vs OSS reference server). Do not guess.
5. Deliberate deviations => `docs/compatibility/known-differences.md`.
