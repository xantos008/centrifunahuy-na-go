# Component provenance and license obligations

Status: **imported** on 2026-11-15. Repo commit `66d1418` holds the upstream tree; the local ref `refs/upstream/v6.10.3` points at upstream commit `4b6d86e`. The rows below were verified against the upstream tree at `v6.10.3`.

Rules (ADR-0002, user requirement E):
- **Never delete or rewrite** a license file, copyright line or license header of imported code. That includes "Copyright (c) Centrifugal Labs LTD" headers and the gorilla/Kelsey Hightower notices.
- An Apache-2.0 file that we modify gets this line under its existing header, never in place of it (Apache-2.0 §4(b)):

  `// Modified by the Cedrazeka authors, 2026. See NOTICE.`

- New files that we write carry our own header, per ADR-0030.
- Every upstream file we modify or delete is listed in §3, in the same commit as the change.
- A new third-party dependency (Go module, npm package, vendored code) needs a row in §1 or §2 and a license check before merge. Allowed: MIT, BSD-2/3, Apache-2.0, ISC, MPL-2.0 (file-level). Anything else needs an ADR. GPL is allowed only for dev tools that are not distributed (golangci-lint, already recorded).

## 1. Imported source (in-tree)

| ID | Component | Origin | Version / commit | License | Copyright notice to keep | Repo location | Our changes planned |
|---|---|---|---|---|---|---|---|
| PRV-01 | Centrifugo OSS server | github.com/centrifugal/centrifugo | tag `v6.10.3`, `4b6d86e23452a6fa1bd424ffe4526862efae85b8` | Apache-2.0 (root `LICENSE`, **no NOTICE file upstream**) | "Copyright 2023 Centrifugal Labs LTD" (LICENSE), "Copyright (c) Centrifugal Labs LTD" (`main.go` header) | repo root | module path rename, branding (ADR-0022), PRO wiring (ADR-0029), web UI removal |
| PRV-02 | WebSocket implementation (fork of gorilla/websocket, synced from centrifuge `internal/websocket`) | github.com/gorilla/websocket via centrifugal/centrifuge | as in PRV-01 | BSD-2-Clause (`internal/websocket/LICENSE`, `AUTHORS`) | "Copyright (c) 2013 The Gorilla WebSocket Authors" and the per-file headers | `internal/websocket/` | none planned |
| PRV-03 | envconfig (fork) | github.com/kelseyhightower/envconfig | as in PRV-01 | MIT (`internal/config/envconfig/LICENSE`) | "Copyright (c) 2013 Kelsey Hightower" | `internal/config/envconfig/` | none planned. The env prefix is passed in by callers |
| PRV-04 | Embedded admin web UI bundle (statik) | github.com/centrifugal/web `main` (built `dist/`) | as embedded in PRV-01 | MIT (GitHub API `spdx_id`, 2026-11-15) | in bundle | `internal/webui/statik.go` | **removed** in step 1.3: branded UI assets (ADR-0016, ADR-0022) |
| PRV-05 | Swagger UI bundle (statik) | github.com/swagger-api/swagger-ui `v4.18.1` | as embedded in PRV-01 | Apache-2.0 | in bundle | `internal/swaggerui/statik.go` | the spec title is rebranded through the generated `api.swagger.json`. The bundle itself is unchanged |
| PRV-06 | Google API annotation protos | github.com/googleapis/googleapis | as in PRV-01 | Apache-2.0 (file headers) | "Copyright 2015 Google LLC" | `internal/apiproto/swagger/google/api/*.proto` | none |
| PRV-07 | Server API / proxy / uni-stream schemas | part of PRV-01 | as in PRV-01 | Apache-2.0 | as PRV-01 | `internal/apiproto/api.proto`, `internal/proxyproto/proxy.proto`, `internal/unigrpc/unistream/unistream.proto` | in default mode, copies under our package names are generated next to the originals. The originals stay unchanged for compatibility mode (ADR-0022) |

## 2. Go module dependencies (not in-tree)

| ID | Module | Version | License | Use |
|---|---|---|---|---|
| PRV-10 | github.com/centrifugal/centrifuge | v0.40.2-0.20261031074606-ae446d43faf0 | MIT | core library, unmodified (ADR-0002) |
| PRV-11 | github.com/centrifugal/protocol | v0.23.1 | MIT | client protocol types and codecs |
| PRV-12 | all other modules in upstream `go.mod` | as pinned in `go.sum` | per module | listed with license text in `THIRD_PARTY_LICENSES`, generated at import. The generator tool is to be vetted (phase-1 plan step 1.6) |
| PRV-13 | github.com/centrifugal/centrifuge-go (test module only) | v0.13.0 | MIT | compatibility suite client. Not distributed in the binary |

## 3. Upstream files modified or deleted

| Path | Change | Step / commit | ADR |
|---|---|---|---|
| `README.md` | replaced by the Cedrazeka README, with an origin/license section; the upstream text stays in the import commit | 1.2 | ADR-0002 |
| `.gitignore` | merged: all upstream rules except `.claude/` (our project skills are tracked), plus `/cedrazeka` and our rules | 1.2 | ADR-0020 |
| `misc/snapshot/go.mod`, `misc/snapshot/go.sum` | **history note, no change**: import commit `66d1418` accidentally contains IDE-made dependency bumps. The next commit (`fixup!`, `4edc28d`) restores the upstream bytes. From `4edc28d` on, the tree equals upstream for all upstream paths | 1.1 | — |
| `misc/snapshot/go.mod`, `misc/snapshot/go.sum` | **history note, no change**: commit `73a6f3c` (step 1.2) again carries the IDE-made bumps; step 1.3 restores the upstream bytes. Local guard: `git update-index --skip-worktree` on both files | 1.2 / 1.3 | — |
| `internal/webui/statik.go` | **deleted**: embedded branded upstream admin UI bundle (PRV-04) | 1.3 | ADR-0016, ADR-0022 |
| `misc/scripts/update_web.sh` | **deleted**: rebuilt the branded UI from centrifugal/web | 1.3 | ADR-0016 |
| `internal/admin/handlers.go` | modified: without `admin.web_path` / `web_proxy_address`, the web prefix answers 404 with a hint instead of serving the embedded UI. The admin API routes are unchanged | 1.3 | ADR-0016 |
| `internal/app/app.go` | modified: the usage-stats sender is never created (no telemetry); the `usage_stats` key is still accepted | 1.3 | KD-06 |
| `.github/workflows/release.yml`, `.goreleaser.yml`, `.github/FUNDING.yml`, `misc/release/`, `misc/scripts/{release_preflight,release_package,release_packagecloud,cosign_images,package}.sh`, `misc/packaging/`, `misc/rpm/` | **deleted**: the upstream release pipeline. It publishes to upstream channels (GitHub `centrifugal/centrifugo`, Docker Hub `centrifugo/centrifugo`, packagecloud `FZambia/centrifugo`, the Homebrew tap, Buf BSR) and injects the usage-stats endpoint and token. Our own release pipeline comes in phase 9 | 1.3 | ADR-0020 |
| `Makefile` | modified: removed `web`, `package`, `packagecloud*` and the `VERSION`/`ITERATION` vars; `lint` and `vuln` use project-local `.bin/` tools via a new `tools` target, replacing the global `go install` (`install-lint` removed) | 1.3 | ADR-0026 |
| all 221 Go files that import the module, `go.mod`, `misc/snapshot/go.mod` | modified: module path `github.com/centrifugal/centrifugo/v6` → `github.com/xantos008/centrifunahuy-na-go` (the repository address; the product name stays Cedrazeka). Same for `…/misc/snapshot`, whose replaced-module version becomes `v0.0.0-…` because the path has no `/vN` suffix. Each file gets the change notice (in `main.go` it sits below the upstream license header). Imports are re-sorted by `gofmt` in 8 files. No `.proto` or `.pb.go` change was needed: every `go_package` is relative (`./;apiproto` etc.) | 1.4 | ADR-0020 |
| `main.go`, `internal/app/{cmd,app,node}.go`, `internal/config/config.go`, `internal/config/envconfig/envconfig.go` (PRV-03), `internal/configtypes/stringmap.go`, `internal/configredact/configredact.go`, `internal/cli/{configdoc,defaultenv,defaultconfig,checkconfig,checktoken,checksubtoken,version}.go`, `internal/metrics/metrics.go`, `internal/telemetry/provider.go`, `internal/consuming/kafka.go`, `internal/api/{grpc,handler,api}.go`, `internal/unigrpc/grpc.go`, `internal/proxy/*_grpc.go` (10), `internal/tools/test_helpers.go`, `Dockerfile`, `Makefile` (`build -o cedrazeka`); new test files `internal/{app,cli,config,configredact,configtypes,telemetry}/brand_compat_test.go` pin compatibility mode for the upstream tests that assert upstream names | modified: names come from `internal/brand`; Cedrazeka by default, upstream names with `--compat=centrifugo` | 1.5 | ADR-0022 |
| `api/proto/cedrazeka/{api,proxy,unistream}.proto` | **new derived files**: copies of the upstream schemas (PRV-07) with the package and service renamed; header states origin and modification | 1.5 | ADR-0022 |

## 4. Attribution files to create at import

- `NOTICE`: created in step 1.2. It reproduces the Swagger UI NOTICE text, as Apache-2.0 §4(d) requires; upstream Centrifugo has no NOTICE file.
- `THIRD_PARTY_LICENSES`: created in step 1.2 with PRV-02, 03, 10 and 11. The Apache-2.0 components (PRV-01, 05, 06) are covered by `LICENSE`. PRV-12 comes in step 1.6. PRV-04 is removed in step 1.3.
- `upstream/BASE`: created in step 1.2.

## 5. Not imported, never to be imported

- Centrifugo PRO code or binaries, in any form.
- Centrifugo logos, trademarks and UI text (PRV-04 is removed for this reason).
- The upstream packaging and release scripts that publish to upstream channels (packagecloud, upstream Docker Hub). They are removed in a separate commit and listed in §3.
