# Implementation phase 1 — upstream import, licensing, names, compatibility-suite harness

Status: **done** (2026-11-15). Exit criteria results are at the end of this document.
Decisions: DEC-01…DEC-12 (2026-11-15). License: Apache-2.0 (DEC-10). Names: Cedrazeka (DEC-12). Nothing is blocking.

## Goal and exit criteria
At the end of phase 1 the repo builds our binary from the Centrifugo OSS v6.10.3 base under our module path and names. Upstream behavior is unchanged in compatibility mode, and a working differential compatibility suite proves it for the ★ scenarios.

Exit criteria (all must hold; exact commands and results go in the phase report):
1. The pristine import commit equals upstream `v6.10.3`, byte for byte: `git diff --stat <import-commit> <upstream v6.10.3 tree>` is empty, excluding the three root collisions documented in step 1.1.
2. Upstream unit tests pass after our changes, with the same pass/fail set as on the pristine import (baseline recorded in step 1.1).
3. Upstream integration tests (`--tags=integration`, docker compose services) have the same pass/fail set as the baseline.
4. `make compat` (ref vs sut, memory profile) is green for every ★ scenario in [compatibility-suite.md](../testing/compatibility-suite.md). `make compat-golden` is green.
5. CS-NAME-01 is green: own names in default mode, upstream names in compatibility mode.
6. provenance.md §3 lists every modified or deleted upstream file. `NOTICE`, `THIRD_PARTY_LICENSES` and `upstream/BASE` exist.
7. Quality gates pass: gofmt, go vet, golangci-lint (new issues only, measured against the import commit), govulncheck, buf lint.

Out of scope for phase 1: any PRO feature, the admin UI, Helm, renaming human-readable log and help strings beyond those listed in step 1.5 (tracked as a phase 9 task), and engines other than memory in the compatibility suite (phase 2).

## Steps

Each step is one reviewable commit or PR, made by the user from a Markdown message drafted by Claude. Order matters.

### 1.0 Commit Phase 0 docs (precondition)
- The user makes every commit. Claude asks for each commit and drafts its message as Markdown. This step commits the current docs, `.claude/`, `tools/`, `CLAUDE.md`, `SPEC.md`, `README.md` and `.gitignore`.

### 1.1 Pristine upstream import
- `git fetch --depth 1 https://github.com/centrifugal/centrifugo.git refs/tags/v6.10.3`, then verify the peeled commit is `4b6d86e23452a6fa1bd424ffe4526862efae85b8`.
- Import the tree at the root with `git read-tree -u --prefix=` / `git checkout FETCH_HEAD -- .` on branch `phase-1/import`.
- **Root collisions** (upstream files that exist in our repo):
  - `LICENSE`: take upstream verbatim (Apache-2.0);
  - `README.md`: take upstream verbatim in this commit, ours is restored in 1.2;
  - `.gitignore`: take upstream verbatim, merged in 1.2.
  
  `CLAUDE.md`, `SPEC.md`, `docs/`, `tools/` and `.claude/` do not collide (upstream has none of them).
- Commit message: `import: centrifugo v6.10.3 (4b6d86e) verbatim`.
- **Record the baseline:**
  - `go build ./...`;
  - `go test -count=1 -race $(misc/scripts/test_packages.sh unit)`;
  - `docker compose up -d` (upstream compose), then `go test -count=1 -race --tags=integration $(misc/scripts/test_packages.sh integration)`.
  
  Save the pass/fail lists to `docs/testing/baseline-upstream-tests.md`. No fixing at this step.

### 1.2 Licensing and attribution (DEC-10: Apache-2.0)
- Apache-2.0 repo-wide (DEC-10): the upstream `LICENSE` stays as is, and the old MIT LICENSE is retired. Our copyright line goes in `NOTICE`.
- Create:
  - `NOTICE`: project, copyright "Cedrazeka authors", and the Centrifugo OSS attribution plus PRV-02..06;
  - `THIRD_PARTY_LICENSES`;
  - `upstream/BASE`.
- Restore our `README.md`, adding "Based on Centrifugo OSS v6.10.3 (Apache-2.0) by Centrifugal Labs LTD" and a link to NOTICE. Merge `.gitignore` (upstream + ours).
- Fill provenance.md §1 with the import commit hash.

### 1.3 Remove upstream-only plumbing and the branded UI
Each deletion is listed in provenance.md §3. Read every file before deleting it.

| Path | Action | Reason |
|---|---|---|
| `internal/webui/` (statik bundle) and its wiring in `internal/admin` / `internal/app` | delete; admin serves only `admin.web_path` until our UI exists | branded UI assets (ADR-0016, ADR-0022) |
| `misc/scripts/update_web.sh`, Makefile `web` target | delete | pulls the branded UI |
| `.github/workflows/release.yml`, `misc/scripts/{release_*.sh, cosign_images.sh, package.sh}`, `misc/release/`, `misc/packaging/`, `misc/rpm/`, `.goreleaser.yml`, Makefile `package*` / `packagecloud*` targets, `.github/FUNDING.yml` | delete (after reading them) | they publish to upstream channels. `snapshot.yml` is **kept**: a before/after behavior check, no publishing |
| Makefile `install-lint`, `vuln` | replace with project-local `.bin/` tools | they ran a global `go install` |
| `.github/workflows/test.yml` | keep, adapted in 1.8 | CI |
| `internal/usage/` reporter | keep the code, disable the start call in `internal/app` (KD-06); accept the config key, never send | no telemetry |
| `Dockerfile` | keep; binary name via 1.5 | |

### 1.4 Module path rename
- `go.mod`: `module github.com/centrifugal/centrifugo/v6` → `module github.com/xantos008/centrifunahuy-na-go`. This is the repository address; the product, binary and every user-visible name is `cedrazeka` (step 1.5 sets the binary name explicitly, since Go would default to `centrifunahuy-na-go`).
- Rewrite imports with `gofmt -r` / `go mod edit` plus a scripted import rewrite. Update `go_package` options in `internal/apiproto/*.proto`, `internal/proxyproto/*.proto` and `internal/unigrpc/unistream/*.proto` and in their generated `*.pb.go`. Result (2026-11-15): every `go_package` is relative (`./;apiproto` etc.), so neither `.proto` nor `.pb.go` contains the module path, and regeneration is **not needed** for the rename. Regenerating with upstream `generate.sh` needs protoc v7.34.0 plus 5 Go plugins installed globally; this is deferred until a `.proto` actually changes, and the tools will then be pinned project-locally.
- One mechanical commit. Provenance §3 records it as "module path, all files".
- Tests: build plus the unit test set equals the baseline.

### 1.5 Names and compatibility mode (ADR-0022, EXT-01)

**New package `internal/brand`:**
- `brand.go`:
  - `type Mode uint8` (`ModeDefault`, `ModeCentrifugo`);
  - name constants (product, binary, env prefix, metrics namespace, key prefix, Kafka client ID, OTel service name, gRPC service names);
  - `func Resolve(args []string, getenv func(string) string) (Mode, error)`, reading `--compat` and `CEDRAZEKA_COMPAT`;
  - a `Names` struct returned by `func (m Mode) Names() Names`.
  
  No globals beyond constants. `Names` is passed down from `main` through `internal/app`.
- `brand_test.go`: table tests for Resolve precedence (flag over env, unknown value → error) and for `Names()` in both modes.

**Edit points in upstream code.** Every edit gets the Apache §4(b) notice line and an entry in provenance §3. Exact lines are re-checked at import time.

| File (v6.10.3) | Change |
|---|---|
| `main.go` | resolve the mode first and pass `Names` into `cli` / `app` |
| `internal/config/config.go` (l.231–443: `envconfig` prefix, `CENTRIFUGO_VAR_`, k8s regex) | prefix from `Names.EnvPrefix` |
| `internal/configredact/configredact.go` (l.362–465) | env name derivation from `Names.EnvPrefix` |
| `internal/cli/defaultenv.go`, `defaultconfig.go`, `configdoc` | prefix and binary name from `Names` |
| `internal/configtypes/redis.go:55`, `types.go:238` (Redis and NATS `prefix` default `centrifugo`), PG prefix defaults, `types.go:342` (Graphite) | defaults applied from `Names.KeyPrefix` at load time, since struct-tag defaults cannot depend on mode; `doc` strings neutralized |
| `internal/metrics/metrics.go:10`, `internal/app/node.go:23`, `internal/app/app.go:403` | default namespace from `Names.MetricsNamespace` (an explicit config value still wins) |
| `internal/consuming/kafka.go` (client ID l.122, `centrifugo-*` header names) | from `Names` |
| `internal/telemetry/provider.go:49` | OTel service name from `Names` |
| `internal/api/grpc.go`, `internal/unigrpc` | register the generated service with our service name. In compatibility mode, also register the upstream `ServiceDesc`. Implemented by copying `grpc.ServiceDesc` with a different `ServiceName` (same handlers and message types), no duplicate codegen |
| `internal/proxy` gRPC client | method path prefix from `Names` (upstream path in compatibility mode) |
| `internal/cli/version.go`, root command help, startup log line | product and binary name |
| `Dockerfile` | `USER`, `WORKDIR` and binary from the build arg `BINARY=cedrazeka` |
| `internal/apiproto/swagger/api.swagger.json` (`info.title`) | own title |

**Our .proto for default mode:** `api/proto/cedrazeka/{api,proxy,unistream}.proto`. These are copies of the upstream files with only `package` and `go_package` changed, header kept, listed in provenance (PRV-07). They are published for users who generate clients in default mode. The server does not need generated code from them, because it uses the `ServiceDesc` copy above.

**Tests:**
- unit tests: brand; config env loading in both modes (`internal/config` table test with `CEDRAZEKA_` vs `CENTRIFUGO_` variables); metrics namespace; Kafka headers;
- gRPC test over `bufconn`: default mode serves `/cedrazeka.api.v1.Api/Publish` and rejects the upstream path; compatibility mode serves both;
- compatibility scenario CS-NAME-01.

**Done (2026-11-15), deviations from the plan above:**
- The mode is process-wide, set in `main`, rather than a `Names` struct passed down (ADR-0022 implementation notes).
- Defaults are mapped in envconfig, not after load.
- The upstream tests that assert upstream names run in compatibility mode.
- The gRPC rename test lives in `internal/brand/e2e`.
- Reflection limitation: KD-26.
- Operator doc: `docs/operations/compat-mode.md`.

### 1.6 THIRD_PARTY_LICENSES generation
- Candidate: `github.com/google/go-licenses/v2` (Apache-2.0). Vet it per CLAUDE.md (source, license, maintenance, network use), pin it in `tools/go.mod`, and record it in `docs/ai/tooling-lock.md` **before** use. If vetting fails, fall back to a ~40-line script using `go list -m -json all` plus module-cache LICENSE files, in `scripts/`.
- Add a Makefile target `licenses`. CI checks that the file is up to date.

**Done (2026-11-15):** `go-licenses` was not adopted. A ~200-line stdlib-only generator (`scripts/thirdpartylicenses.go`, `//go:build ignore`) avoids vetting and pinning another tool with its own dependency tree. It is offline: `go list` plus the module cache. Targets: `make licenses`, `make licenses-check`.

### 1.7 Build image for the compatibility suite
- `test/compat/Dockerfile.sut`: multi-stage build (`golang:1.27` builder → the same `alpine:3.24` base as upstream). It builds `./` into `/usr/local/bin/cedrazeka`.

### 1.8 Compatibility-suite harness (ADR-0028)

**New Go module `test/compat`.**

| File | Content |
|---|---|
| `test/compat/go.mod` | `module github.com/xantos008/centrifunahuy-na-go/test/compat`. Requires `github.com/centrifugal/centrifuge-go v0.13.0`, `github.com/centrifugal/protocol v0.23.1`, `google.golang.org/grpc` (the version already in upstream go.sum), and a WebSocket client: whatever centrifuge-go already depends on, so no new library |
| `test/compat/docker-compose.yml` | `ref: centrifugo/centrifugo:v6.10.3@sha256:33fdbbce…3803`; `sut` from `Dockerfile.sut` with `--compat=centrifugo`; config mounted from `configs/memory/`. Each on its own port; redis/nats/postgres services defined but unused until phase 2 |
| `test/compat/configs/memory/config.json` | namespaces covering the CS-SUB-01 permission matrix, history on `hist:` namespace, test HMAC secret and test RSA/EC/Ed25519 public keys, API key `compat-test-key` (test-only values) |
| `test/compat/harness/target.go` | `Target` struct, `FromEnv(prefix)`. Reads `COMPAT_REF_*` / `COMPAT_SUT_*` |
| `test/compat/harness/rawclient.go` | raw WebSocket client: connect with `?format=json|protobuf`, send exact bytes, receive frames with deadline, record events |
| `test/compat/harness/transcript.go` | `Event`, `Transcript`, `Normalize()` (rules table from the spec §3), `Diff(a, b) string` |
| `test/compat/harness/golden.go` | `-update` flag; load/store `testdata/golden/<name>.json` |
| `test/compat/harness/token.go` | test token builders HS256/RS256/ES256/EdDSA, plus `alg=none` and alg-confusion builders; test keys embedded as constants |
| `test/compat/harness/run.go` | `Run(t, name, scenario)`: diff mode (ref and sut) or golden mode, chosen by `COMPAT_MODE` |
| `test/compat/scenarios/proto_test.go` | CS-PROTO-01..04 |
| `test/compat/scenarios/auth_test.go` | CS-AUTH-01..02 |
| `test/compat/scenarios/sub_test.go` | CS-SUB-01..02 |
| `test/compat/scenarios/pub_test.go` | CS-PUB-01 (memory) |
| `test/compat/scenarios/history_test.go` | CS-HIST-01 (memory) |
| `test/compat/scenarios/recovery_test.go` | CS-REC-01..02 (memory) |
| `test/compat/scenarios/presence_test.go` | CS-PRES-01 (memory) |
| `test/compat/scenarios/api_test.go` | CS-API-01..02 (all 21 HTTP routes) |
| `test/compat/scenarios/names_test.go` | CS-NAME-01 (sut only) |
| `test/compat/harness/transcript_test.go` | unit tests for normalization (placeholders are stable and numbered; non-volatile fields are untouched) and for Diff |
| `test/compat/testdata/golden/*.json` | recorded from ref with `make compat-record` |
| Makefile (root) | targets `compat`, `compat-golden`, `compat-record` |

**Progress (2026-11-15), part A done:** harness, `docker-compose.yml`, the memory config, the Makefile targets, CS-PROTO-01 and CS-API-01. All three modes are verified (diff, golden, record), including a negative check: a tampered golden fails. Deviations:
- No `Dockerfile.sut` (step 1.7). Docker DNS is unreliable on the dev machine (`apk`, `proxy.golang.org`), so the sut is the host-compiled linux binary mounted into `alpine:3.24`.
- `rawclient.go` and `golden.go` are merged into `client.go` and `run.go`.
- The client is gorilla/websocket instead of centrifuge-go, until behavior scenarios need the SDK.

**Part B done (2026-11-15):** CS-PROTO-02/03/04, CS-AUTH-01/02, CS-SUB-01/02, CS-PUB-01, CS-HIST-01, CS-REC-01/02, CS-PRES-01, CS-API-02 (all 20 non-info routes valid and invalid, plus `info` by shape) and CS-NAME-01 (a third container, `sut-default`). Stable over 3 diff and 2 golden runs. Harness additions:
- Protobuf codec (`centrifugal/protocol` v0.23.1);
- `Next`, `RecvPing`, `Last`;
- per-run channel suffix;
- `KeysOnly` and `Omit`;
- presence-key normalization;
- RS256, ES256, EdDSA, alg=none and alg-confusion tokens (test-only keys in `harness/keys.go`).

### 1.9 CI
`.github/workflows/test.yml`, adapted from upstream:
- jobs `lint`, `unit` and `integration` (upstream services);
- a new job `compat` (docker compose; `make compat`; on failure, the diff is uploaded as an artifact);
- `govulncheck`;
- a `licenses` freshness check.

No publishing jobs. Third-party actions are pinned by commit SHA.

**Done (2026-11-15):** `test.yml` now has three jobs:
- `checks`: gofmt, vet, `make lint`, `make vuln`, `make licenses-check`;
- `test`: upstream unit and integration tests, unchanged;
- `compat`: `make compat-up`, `compat`, `compat-golden`.

Pinned actions:
- checkout v7.0.1 `3d3c42e5…`;
- setup-go v7.0.0 `b7ad1dad…`;
- cache v6.1.0 `55cc8345…`.

govulncheck reports GO-2026-5932 (`golang.org/x/crypto/openpgp`) only at module level: the code does not call it, so the exit code is 0.

**Not yet verified:** the workflow has not run on GitHub. It runs on the next push.

### 1.10 Docs updated in phase 1
- provenance.md §1 and §3;
- `docs/testing/baseline-upstream-tests.md`;
- `docs/testing/compatibility-suite.md` (any rule changes);
- feature matrix: no status changes beyond `inherited`. `verified` comes only from green diff-mode runs, so ★ rows on memory may move;
- `docs/operations/compat-mode.md` (how to run in compatibility mode, env and metrics names);
- `docs/ai/tooling-lock.md` if go-licenses is added.

## New dependencies in phase 1

| Module | Where | License | Status |
|---|---|---|---|
| none in the server module | — | — | the import brings upstream go.mod/go.sum as is |
| github.com/centrifugal/centrifuge-go v0.13.0 | test/compat | MIT | verified (GitHub API, 2026-11-15) |
| github.com/centrifugal/protocol v0.23.1 | test/compat | MIT | already in upstream go.sum |
| google.golang.org/grpc | test/compat | Apache-2.0 | already in upstream go.sum |
| github.com/google/go-licenses/v2 | tools | Apache-2.0 | **to vet** in step 1.6 |
| docker image centrifugo/centrifugo:v6.10.3 | test/compat | Apache-2.0 | digest pinned |

## Quality-gate commands for phase 1
```
go build ./...
gofmt -l . | (! grep .)
go vet ./...
.bin/golangci-lint run --new-from-rev=<import-commit> ./...    # upstream pins v2.14.0; we run v2.15.0, new issues only
go test -count=1 -race $(misc/scripts/test_packages.sh unit)
docker compose up -d && go test -count=1 -race --tags=integration $(misc/scripts/test_packages.sh integration)
.bin/govulncheck ./...
.bin/buf lint
make compat && make compat-golden
(cd test/compat && go vet ./... && go test ./harness/...)
```

## Risks specific to phase 1
| Risk | Mitigation |
|---|---|
| Upstream tests that are already flaky or need services hide our regressions | baseline recorded on the pristine import (1.1); compare sets, not just "green" |
| Proto regeneration differs from the committed `.pb.go` (generator version drift) | step 1.4 checks for a zero diff first; pin generator versions from `generate.sh` |
| Struct-tag defaults (`default:"centrifugo"`) cannot depend on mode | apply the mode-dependent default after load, only when the value is unset; test both modes |
| Compatibility-suite nondeterminism (timing, ordering) | determinism rules (spec §5); normalization only for listed fields |
| Docker Hub unavailable in CI | golden mode fallback |

## Exit criteria results (2026-11-15)

| # | Criterion | Result |
|---|---|---|
| 1 | Pristine import equals upstream v6.10.3 | Yes, for all 565 upstream paths from `4edc28d` on. Import commit `66d1418` itself carries IDE-made `misc/snapshot/go.{mod,sum}` bumps, reverted in the next commit (provenance.md §3) |
| 2 | Upstream unit tests: same pass/fail set as baseline | Yes. 549/549 identical after every step; last full run at step 1.5. Steps 1.6–1.9 changed no code linked into the binary |
| 3 | Upstream integration tests: same set | Yes. 301/301 identical (isolated runner, step 1.5); green in CI on `7a95c35` |
| 4 | ★ scenarios green, diff and golden, memory profile | Yes. 15 scenarios; locally 3 diff and 2 golden runs; CI run 43723583072 |
| 5 | CS-NAME-01 | Yes |
| 6 | Provenance §3 complete; NOTICE, THIRD_PARTY_LICENSES, upstream/BASE | Yes. 232 modified and 27 deleted upstream files are covered; every modified file that can hold a comment carries the change notice |
| 7 | Quality gates | Yes. gofmt, go vet, golangci-lint (new issues vs import commit: 0; test/compat: 0), govulncheck (exit 0), buf (`api/proto/cedrazeka` builds), licenses-check |

Carried over to later phases:
- R/N/P engine profiles of the suite (phase 2).
- Human-readable text rebranding: logs, metric help, config `doc` strings (phase 9).
- gRPC reflection names (KD-26).
- `misc/snapshot` already fails to build upstream.
