# AI & dev tooling lock

Inventory date: 2026-11-15. Policy: CLAUDE.md "Claude Code skills and MCP setup" + SPEC.md §10. Nothing installed globally in this phase.

## Environment (inspected, nothing changed)

| Item | Value |
|---|---|
| Claude Code | 2.1.325 |
| Go | go1.27.4 darwin/arm64 |
| Docker | 29.9.3 (daemon running) |
| redis-server (host) | 8.4.0 — not used by the project; integration tests will use containers |
| psql (host) | 14.20 client only |
| nats-server, clickhouse | not installed (will use containers in Phase 2/5) |
| node / npx, uv / uvx | present (not used by any installed tool) |

### Pre-existing, user-level (not installed by this project, not modified)

| Kind | Name | Version / source | Note |
|---|---|---|---|
| plugin | ponytail | 4.9.0, user scope | minimalism guidance |
| skill | devops-engineer | 1.2.0, MIT, github.com/Jeffallan | user skill |
| skill | graphify | user skill | referenced by user's global CLAUDE.md |
| MCP | claude.ai Claude Docs / Google Drive / Google Sheets / Google Docs | claude.ai connectors | account-level; **not used for this repo** — no source code is sent to them |

No project-level `.claude/settings*.json` or `.mcp.json` existed.

## Installed by this project (project-local)

### Go dev tools — `tools/go.mod` (separate module, does not touch the product module)

Installed with `go get -tool` (Go ≥1.24 tool directives), checksums in `tools/go.sum`, `go mod verify` → `all modules verified`. Binaries built into gitignored `.bin/` with:
`go -C tools build -o ../.bin/ github.com/golangci/golangci-lint/v2/cmd/golangci-lint golang.org/x/vuln/cmd/govulncheck github.com/bufbuild/buf/cmd/buf`

| Tool | Version | License | Source | Purpose | Network | Smoke test |
|---|---|---|---|---|---|---|
| golangci-lint | v2.15.0 | GPL-3.0 (dev tool only, never linked into product) | github.com/golangci/golangci-lint | lint gate | none | `version` → 2.15.0 ✔ |
| govulncheck | v1.9.0 | BSD-3-Clause | golang.org/x/vuln (Go team) | vuln gate | queries vuln.go.dev (module paths/versions only, no source) | `-version` ✔ |
| buf | v1.75.0 | Apache-2.0 | github.com/bufbuild/buf | protobuf lint/breaking checks for client/server API schemas | none for lint/breaking (BSR only if explicitly used) | `--version` → 1.75.0 ✔ |

Side effect: modules were downloaded into the standard Go module cache (`$GOMODCACHE`), which is per-user cache, not an installation; no PATH, shell or global config changes.

### Project skills — `.claude/skills/` (written in-repo, no third-party code)

| Skill | Purpose |
|---|---|
| `compat-research` | how to verify behavior against the pinned baseline and update source register / matrix / open questions |
| `go-quality-gates` | exact gate commands from CLAUDE.md using the pinned tools |

## Evaluated, NOT installed (with reason)

SPEC §10 lists 28 skills. Not installed as separate skills: general Go/distributed/Redis/PostgreSQL/NATS/ClickHouse/gRPC/WebSocket/JWT/CEL/OTEL/frontend/K8s/tech-writing knowledge is already covered by the base model; a third-party skill adds supply-chain risk for no verified gain. Existing built-in skills cover `security-review`, `code-review`, `simplify`; user skill `devops-engineer` covers Docker/K8s/CI. Revisit per phase: add an in-repo skill only when a project-specific procedure exists (e.g. `protocol-golden` in Phase 1).

| MCP | Decision | Reason / what is needed |
|---|---|---|
| filesystem | not needed | Claude Code built-in Read/Write/Edit already scoped to repo |
| fetch / HTTP reader | not needed | built-in WebFetch; docs are read from pinned git clones |
| github (github/github-mcp-server) | **declined (DEC-08, 2026-11-15)** | requires a GitHub token; `gh` CLI already works read-only for release notes |
| context7 | **declined (DEC-08, 2026-11-15)** | third-party hosted service (queries leave the machine) |
| postgres / redis / clickhouse | deferred to Phase 2/5 | no local databases exist yet; reference `modelcontextprotocol/servers` postgres server is archived — pick a maintained one then, read-only, dev container DSN only |
| docker | not needed | `docker` CLI via Bash suffices |
| playwright | deferred to Admin UI phase | no UI yet |
| kubernetes/helm | deferred to Phase 7 | no dev cluster |
| memory, sequential-thinking | not installed | optional, no value over repo docs |

If any deferred MCP requires `npx`/`uvx` it would populate user-level caches (`~/.npm`, `~/.cache/uv`); per the "no global install" rule that will be raised with the user before installing.

## Decisions 2026-11-15 (ADR-0026)

- DEC-08: GitHub MCP and Context7 are not installed. Everything stays project-local: `tools/go.mod` → `.bin/`, `.claude/skills/`, and a future project `.mcp.json` only.
- `github.com/google/go-licenses/v2`: **not adopted** (step 1.6). It was replaced by the in-repo stdlib-only generator `scripts/thirdpartylicenses.go`. No new tool installed.
- Side effect, recorded for transparency: verifying centrifuge at the pinned pseudo-version (`go mod download`) put the module into the standard Go module cache (`$GOMODCACHE`), the same cache the dev tools use. This is a download cache, not an install.

## Test container images (phase 2, pinned by digest)

Pulled into the local Docker image store; used only by test compose files, never part of a Cedrazeka artifact.

| Image | Digest | License | Used by | Since |
|---|---|---|---|---|
| `redis:8.10.3-alpine` | `sha256:7958cfbb9d152f87dadfbc89f1b0b15938f0b81ff8982feb327476f4911935c3` | Redis 8: RSALv2 / SSPLv1 / AGPLv3 (tri-license; RSAL and SSPL are not OSI open source). Condition C1: using it only for tests does not make its license irrelevant; Redis is never shipped | `test/compat/docker-compose.yml` (`redis-ref`, `redis-sut`); `test/library/compose.isolated.yml` | 2.2 |
| `nats:2.15.3-alpine` | `sha256:558394a23dd3c36f2d3a64b51865fd09908010e80a542cea5e75c2af543ed62a` | NATS server: Apache-2.0. Current stable release on 2026-11-16 (same digest as `2-alpine`) | `test/compat/docker-compose.yml` (`nats-ref`, `nats-sut`) | 2.5 |
| `postgres:18.7-alpine` | `sha256:b6972e1115a4ff8eba5d7f7d19184926d1e16d7d1136ae1e1a7ebf286da0f015` | PostgreSQL License. Current release on 2026-11-16 (same digest as `18-alpine`); the docs require PostgreSQL 16+ | `test/compat/docker-compose.yml` (`postgres-ref`, `postgres-sut`) | 2.6 |
| `valkey/valkey:9.1.3-alpine` | `sha256:bf09d58e96d6b4524a15396b65b242bbe18beebbcc15a04d8ee75ddac62fae0b` | Valkey: BSD-3-Clause. Current release on 2026-11-16 (same digest as `9-alpine`) | `test/library/compose.isolated.yml` (library job, D2-6) | 2.10 |
| `golang:1.27.4-bookworm` | `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea` | Go: BSD-3-Clause; Debian packages under their own licenses | `test/library/compose.isolated.yml` (`tests`) | 2.10 |

Phase 2, step 2.10: the centrifuge library's own test-only modules (e.g. `github.com/quagmt/udecimal`, `github.com/prometheus/common`) were added to `$GOMODCACHE` by `go mod download` in the library module (`make library-test`). Download cache, not an install.

CI action added in step 2.10: `actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9` (v7.0.2, tag SHA from `git ls-remote` on 2026-11-16), library job results only.

### Phase 3 consumer test images

The compatibility suite starts queue providers in a dedicated Compose project. Each reference and Cedrazeka instance has its own network and data store.

| Profile | Image | Pinned manifest |
|---|---|---|
| cloud, Pub/Sub | `gcr.io/google.com/cloudsdktool/google-cloud-cli:572.0.0-emulators` | `sha256:727bf8869594d039959bddce6727aae1ea0fd464f3e5b31df55ab6fbb8e687c5` |
| cloud, SQS | `localstack/localstack:4.14` | `sha256:3ebc37595918b8accb852f8048fef2aff047d465167edd655528065b07bc364a` |
| kafka | `confluentinc/cp-kafka:8.0.0` | `sha256:9cdc8119cb39c45f0efa6da8e2220058020c7433e3ba3a3ce11a8006a18cb336` |
| azure, Service Bus | `mcr.microsoft.com/azure-messaging/servicebus-emulator` | `sha256:5a96d893b245031740f7d46e0fe5ff282d24b78c4b7d761dd57590f3f010a9b3` |
| azure, SQL Server | `mcr.microsoft.com/mssql/server:2022-latest` (linux/amd64) | `sha256:4525e3efac9f3e241cb003bb2ff2a82818b502f37e12049d788b1a9fa7e13b7f` |

The opt-in Azure profile requires operator acceptance of the Microsoft terms for both images. SQL Server uses Developer edition and a new password for each local run.

## Shared project skills

The instructions live in `.claude/skills/`. Each corresponding directory in `.agents/skills/` is a relative symlink for Codex discovery. ADR-0026 continues to govern project-local tooling. These instructions are authored in this repository and have no executable hooks, credentials, network permissions, or third-party packages.

| Skill | Workflow |
|---|---|
| `protocol-compatibility` | Wire behavior, SDKs and reference transcripts |
| `engine-integration` | Backend isolation, state, failover and library tests |
| `extension-architecture` | Extension seams, migration and decisions |
| `defensive-security-review` | Server-side policy, redaction, and bounded work |
| `isolated-ci` | Compose, pinned CI, dependency checks and release gates |
| `realtime-observability` | Traces, analytics, backpressure and benchmarks |
| `push-delivery` | Device lifecycle, queue behavior and provider contracts |
| `admin-contracts` | Admin API, role checks, audit and interface flows |

The architecture, operations and defensive review workflows adapt practices inspected in the local SinoMotors repositories. The instructions are specific to Cedrazeka’s Go runtime, test profiles and compatibility contract.

On 2026-11-24, all ten project skills passed `quick_validate.py` through their Codex paths. Each symlink resolved to its corresponding Claude skill file. The validation used PyYAML 6.0.4 from a temporary directory. The `isolated-ci` instructions now cover delegated parallel Compose runs, including endpoint selection and per-run logs.
