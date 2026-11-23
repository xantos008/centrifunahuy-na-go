---
name: isolated-ci
description: Maintain Cedrazeka Compose test infrastructure, GitHub Actions, dependency checks and release gates.
---

# Isolated ci

Compare against `Makefile`, relevant test Compose files, `.github/workflows/test.yml` and `docs/ai/tooling-lock.md`. Pin actions by full SHA and test images by digest. Keep test services scoped to the test project and check port ownership and volume isolation. Keep Redis server out of release artifacts. Compare exact test inventories with the applicable baseline files. Use existing Go vulnerability and license targets before adding scanners.
