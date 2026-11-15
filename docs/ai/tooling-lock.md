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
- Phase 1 candidate to vet before use: `github.com/google/go-licenses/v2` (Apache-2.0), for `THIRD_PARTY_LICENSES` (phase-1 plan step 1.6). Not installed yet.
- Side effect, recorded for transparency: verifying centrifuge at the pinned pseudo-version (`go mod download`) put the module into the standard Go module cache (`$GOMODCACHE`), the same cache the dev tools use. This is a download cache, not an install.
