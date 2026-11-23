---
name: isolated-ci
description: Maintain Cedrazeka Compose test infrastructure, GitHub Actions, dependency checks and release gates.
---

# Isolated ci

Compare against `Makefile`, relevant test Compose files, `.github/workflows/test.yml` and `docs/ai/tooling-lock.md`. Pin actions by full SHA and test images by digest. Keep test services scoped to the test project and check port ownership and volume isolation. Keep Redis server out of release artifacts. Compare exact test inventories with the applicable baseline files. Use existing Go vulnerability and license targets before adding scanners.

When delegating compatibility tests to a parallel stack, assign unique loopback ports and pass the stack's `COMPAT_COMPOSE_FILE`, `COMPOSE_PROJECT_NAME`, `COMPAT_PROFILE`, and endpoints to the runner. Start Compose with that file, project name, and profile. Run `go test` with the same settings and the matching `COMPAT_REF_URL`, `COMPAT_SUT_URL`, `COMPAT_SUT_DEFAULT_URL`, `COMPAT_REF_GRPC_ADDR`, and `COMPAT_SUT_GRPC_ADDR` as applicable. Before a restart, compare `docker compose port <service> 8000` for that project with the HTTP endpoint the test uses. Save complete logs for each run in a verification series.
