# Implementation phase 2 — verify the inherited engines

Status: **approved** (2026-11-16). D2-1…D2-6 accepted as recommended; the user's conditions C1–C6 below are binding for every step.
Inputs: [phase-1.md](phase-1.md) (done), [compatibility-suite.md](../testing/compatibility-suite.md), ADR-0006/0008/0009/0010/0011/0019/0023/0028, feature matrix §Implementation order row 2, open questions OQ-ENG-01..05, OQ-HIST-01..03, INFRA-CONFLICT-04/15, CORE-OQ-12.

## Goal
Prove, black-box against Centrifugo OSS v6.10.3, that the inherited engine code behaves the same on every OSS backend and topology, in single-node and multi-node setups, including restarts and backend failures. Phase 2 writes tests, configs and docs. **It is not expected to change server code**: a ref/sut diff means either a regression we introduced (names, cleanup) or a harness bug, and is fixed as such, with a KD entry only if the difference is deliberate.

## What is being verified, and where the code lives
Facts from the code survey (2026-11-16, file references in the survey notes of step 2.0):

| Backend | Code | Tested today |
|---|---|---|
| memory broker/presence | centrifuge library | compat suite (M), library unit tests |
| Redis broker/presence (standalone, Sentinel, Cluster, sharded) | **centrifuge library** (`broker_redis.go`, `presence_redis.go`); config → shards in `internal/confighelpers/redis.go`, `internal/redisshard` | our repo: Redis standalone only (`internal/app` leak test). Sentinel/Cluster/sharding: config-building unit tests only. The library's own integration suite (Redis 7/8, Valkey 8/9, Cluster, Sentinel, DragonflyDB) is **not run by our CI** |
| NATS broker (+ raw mode, wildcards) | ours: `internal/natsbroker` | our integration tests |
| PostgreSQL stream broker, controller, map broker | ours: `internal/pgstreambroker`, `internal/controllers`, `internal/pgmapbroker`, `internal/pgschema` | our integration tests (PG 16) |
| `redisnats` broker | ours, unreleased (`enable_unreleased_features`) | our integration tests; not in scope beyond "rejected without the flag" |

Facts that shape the profiles:
- `controller.type` has one value, `postgres`. Without a controller, the Redis and NATS brokers act as controller; the memory and PostgreSQL brokers do not, so **multi-node PostgreSQL needs `controller.type=postgres`** (answers the code side of OQ-ENG-03; the black-box side is scenario CS-ENG-03b).
- PostgreSQL has no presence manager. Presence is memory (per node) unless `presence_manager.type=redis`.
- NATS has no history, idempotency or version suppression (`ErrorNotAvailable` at runtime, not a config-load error as the catalogue assumed for CS-HIST-03).
- Node info is published every 3 s, peers expire after 7 s (cleanup every 9 s). `/health` never checks the engine (KD-30).
- PG read replicas and PG fan-out exist in code but are not reachable from config (PRO-26/40, phase 7).
- Map brokers (memory/redis/postgres) belong to map subscriptions: verified in phase 3 together with OSS-SUB-03..07 (OSS-ENG-18 moves there).

## Decisions (accepted 2026-11-16, all as recommended)

| # | Question | Decision | Why |
|---|---|---|---|
| D2-1 | Do ref and sut share backend instances? | **No.** Each target gets its own backend containers on its own Docker network, with the same hostname (`redis`, `nats`, `postgres`), so both read one identical config file | ADR-0023 forbids mixed clusters; a shared Redis would make the two servers deliver each other's publications |
| D2-2 | Backend versions in the compat suite | Redis **8** (alpine), NATS **2.10**, PostgreSQL **16**; images pinned by digest | upstream compose uses Redis 7 / NATS 2.10 / PG 16; the library CI uses Redis 7+8. Redis ≥ 7.4 is required for `presence_hash_field_ttl`. Redis 7 stays covered by our integration tests and the library job |
| D2-3 | Run the centrifuge library's own integration tests at our pinned version? | **Yes**, one CI job: Redis 8 + Valkey 9, Cluster, Sentinel, isolated runner. DragonflyDB excluded (docs: "should work fine", not a guarantee) | the Redis engine is library code; today nothing of ours runs it against Cluster or Sentinel |
| D2-4 | KD-07 `history_use_lists` (deprecated upstream) | **Keep inherited behavior** (works as upstream), verify it on an R-lists variant, change KD-07 to "inherited, deprecated, not recommended" | the Phase 0 proposal "fail validation" would itself be a compatibility break |
| D2-5 | Redis IAM auth (OSS-ENG-08), managed Redis (OSS-ENG-07: ElastiCache, Memorystore) | **Not verifiable locally**: stay `inherited` with a note; config-parse tests only | needs cloud accounts and credentials (CLAUDE.md: ask first; not worth it now) |
| D2-6 | Valkey in the compat suite | **No**: library job only | the server code path is identical to Redis; the library already tests Valkey |

## Conditions (user, 2026-11-16)

| # | Condition | How it is enforced |
|---|---|---|
| C1 | Redis 8 is not open source under an OSI license (RSALv2 / SSPLv1 / AGPLv3). Using it only for tests does not make the license irrelevant. Redis is never part of our distributed artifacts | the license status is stated in this plan, in `docs/operations/engines.md` and in the tooling notes; Redis images appear only in test compose files; step 2.11 checks that `Dockerfile`, release files and `THIRD_PARTY_LICENSES` contain no Redis server component (the Go client `rueidis` is Apache-2.0 and is a separate thing) |
| C2 | Two independent Docker networks for ref and sut. Same hostnames resolve inside each network; no shared volumes for Redis, NATS, PostgreSQL; stop/restart actions are symmetric; tests cannot reach the other side's backend by accident | (a) `harness.Backend(side)` is the only way to address a backend; a side is chosen by the target, never by a literal port. (b) Backend actions go through `harness.Both(t, action)`, which applies the action to ref and sut and records it once; there is no single-side variant for backend actions. (c) An isolation check runs at the start of every profile (`CS-ISO-01`): inside each server container `getent hosts redis` (and `nats`, `postgres`) resolves to an address of that side's backend; the two answers differ; `docker inspect` shows no volume or bind mount shared between the sides' backends; a probe publication through ref's backend (e.g. NATS raw subject or a Redis key) is invisible to sut and vice versa. The profile fails before any scenario if the check fails. (d) Backends have no host ports except the loopback ports listed below, and each port belongs to exactly one side |
| C3 | Golden files are never updated automatically. A diff is investigated, its cause established, and only then accepted | `make compat-record` is never called by CI or by other targets; it records only the scenarios named in `COMPAT_RECORD` (no "record all" default); every golden change is reviewed with `git diff`, and the commit message states the cause per changed file. The move to `golden/m/` in step 2.1 is a pure `git mv` with zero content diff |
| C4 | The library is tested at an exact, recorded version and commit; library results are never mixed with server results | centrifuge `v0.40.2-0.20261031074606-ae446d43faf0`, commit `ae446d43faf0e6f34619e42c11935340257669a7` (from the module cache `.info` origin, 2026-11-16); the runner checks that the copied module matches this version before running. Separate make target (`library-test`), separate CI job, separate baseline file `docs/testing/baseline-library-tests.md`; library numbers never appear in the server baseline or in compat results |
| C5 | No automatic Docker cleanup. Before step 2.10 check disk usage; only then clear unneeded build cache. Never `docker system prune --volumes` | before 2.10: `docker system df` and free space are reported to the user; `docker builder prune` runs only after the user confirms; no `system prune`, no volume prune, no image prune of images not created by this project |
| C6 | No server code change without a separate decision | a ref/sut mismatch is first classified: test, config, infrastructure, or our code. If it is our code, work stops, the finding is reported with evidence, and a fix needs the user's decision; the fix is its own commit (and PR), never mixed into a test step. Phase-2 test commits must not touch files outside `test/`, `docs/`, `Makefile`, `.github/` |

## Engine profiles
One config directory per profile in `test/compat/configs/<profile>/`, identical for ref and sut (sut adds `--compat=centrifugo`). Every profile keeps the phase-1 namespaces and keys, plus the engine sections and the namespaces the new scenarios need.

| Profile | Config | Backends per target | Nodes per target |
|---|---|---|---|
| `m` | memory (existing) | – | 1 |
| `r` | `engine.type=redis`, one address; `presence_ttl` 5s, `client.presence_update_interval` 2s (short, for CS-PRES-02); Redis user + password (ACL) | redis | 2 |
| `r-opt` | as `r` + `presence_hash_field_ttl`, `presence_user_mapping` | redis | 1 |
| `r-lists` | as `r` + `history_use_lists` | redis | 1 |
| `rs` | Sentinel: `redis+sentinel://…?sentinel_master_name=mymaster` | 1 master, 2 replicas, 1 sentinel | 2 |
| `rc` | Cluster: plain address of one node (auto-detect, OQ-ENG-02) | 3-master cluster | 2 |
| `rsh` | sharding: `address` array of 2 standalone Redis | redis ×2 | 2 |
| `n` | `broker.type=nats` + `presence_manager.type=redis` (the documented pairing) | nats, redis | 2 |
| `n-raw` | `n` + `raw_mode` + `allow_wildcards` | nats, redis | 1 |
| `p` | `broker.type=postgres` + `controller.type=postgres`, presence memory | postgres | 2 |
| `p-notify` | `p` + `use_notify: true` (broker and controller) | postgres | 1 |

Ports, all on 127.0.0.1: ref nodes 18000 / 18100, sut nodes 28000 / 28100, sut-default 28001 (as now). For scenarios that act on a backend directly: PostgreSQL via `docker compose exec … psql` (no driver in the test module); NATS via loopback ports 14222 (ref) / 24222 (sut) and `nats.go` (Apache-2.0, already in the root go.sum). Redis and Sentinel commands via `docker compose exec … redis-cli`.

Topology notes: Cluster and Sentinel announce IP addresses. They must announce the container address, not 127.0.0.1 (which is what the library's compose uses, because its tests run on the host). This is the main infrastructure risk of the phase and is solved first in step 2.4.

## Scenarios
Existing scenarios get profile tags; new ones are added to the catalogue in `compatibility-suite.md`. Engine-independent groups (CS-PROTO, CS-AUTH, CS-SUB-01/02, CS-API-01/02) stay on `m` only.

**Single node**
| ID | Content | Profiles | Rows |
|---|---|---|---|
| CS-PUB-01 ★ | existing | m r rs rc rsh n p | OSS-API-01, OSS-SUB-14 |
| CS-HIST-01 ★ | existing | m r r-lists rs rc rsh p | OSS-HIST-01, 09 |
| CS-HIST-02 | `history_remove` keeps the top position; `history_meta_ttl` expiry (short TTL namespace) | m r rc p | OSS-HIST-02 |
| CS-HIST-03 | history/recovery/idempotency/version on a NATS broker: observed outcome (startup error or runtime error code), recorded from ref | n | OSS-ENG-13, OQ-ENG-05 |
| CS-HIST-04 | history limits: `history_max_publication_limit`, `recovery_max_publication_limit` global and per namespace; permission gating 103/108 | m r p | OSS-HIST-07, 08, 12 |
| CS-REC-01/02 ★ | existing | m r r-lists rs rc rsh p | OSS-HIST-03 |
| CS-REC-03 | epoch change: node restart (m), Redis restart without persistence (r) → `recovered=false`, new epoch | m r | OSS-HIST-03, OQ-HIST-01 |
| CS-REC-04 | cache recovery mode, `auto_cache_recover`, positioning fields in the subscribe reply | m r p | OSS-HIST-04, 05, 06, OSS-SUB-08, OQ-HIST-03 |
| CS-REC-05 | publish concurrently with subscribe-with-recover, 200 iterations: no loss, no duplicates (outcome check on both targets, not a transcript diff) | r p | OSS-HIST-03, ADR-0008 |
| CS-PRES-01 ★ | existing | m r r-opt rs rc rsh n | OSS-PRES-01, 02, 03 |
| CS-PUB-03 | delta (fossil): `allowed_delta_types`, `delta_publish`, subscribe with `delta` | m r rc p | OSS-SUB-09 |
| CS-PUB-04 | tags filter (`tf`) with eq/in/and/or | m r n p | OSS-SUB-10 |
| CS-PUB-05 | `version`/`version_epoch` suppression; `idempotency_key` (NATS ignores it) | m r n p | OSS-SUB-14 |
| CS-ENG-CFG | startup with invalid engine configs: unknown type, `redisnats` without the unreleased flag, sentinel without master name: exit status and first error line (names normalized) | – (no backend) | OSS-ENG-01, 12, 20 |

**Multi-node** (two nodes per target; each scenario first waits until `info` on both nodes lists 2 nodes)
| ID | Content | Profiles |
|---|---|---|
| CS-ENG-01 | cross-node fan-out: subscriber on A, client publish on B and API publish on B; join/leave across nodes | r rs rc rsh n p |
| CS-ENG-02 | presence and `presence_stats` across nodes | r rs rc rsh n p (p records per-node presence) |
| CS-ENG-03 | `info` lists both nodes; a stopped node disappears within the documented cleanup (≤ 20 s). 03b: `p` profile without controller, observed from ref (OQ-ENG-03) | r n p |
| CS-ENG-04 | server API by user across nodes: `disconnect`, `unsubscribe`, `subscribe`, `refresh` (= CS-API-04) | r n p |
| CS-ENG-05 | history and recovery across nodes: publish via A, recover on B | r rc p |
| CS-PRES-02 | presence TTL after a node crash (`docker kill` node B): entries vanish from A within `presence_ttl` + interval | r |

**Failure** (outcome scenarios; each action is applied to ref's and sut's backend alike)
| ID | Content | Profiles |
|---|---|---|
| CS-ENG-06 | backend stopped: API publish error, subscribe error code; backend started: publish and subscribe work again within a deadline | r n p |
| CS-ENG-07 | Sentinel `SENTINEL FAILOVER mymaster`: publishing works again within a deadline, history on the new master | rs |

**Names** (sut only)
| ID | Content |
|---|---|
| CS-NAME-02 | default mode on `r` and `n`: Redis keys and NATS subjects use the `cedrazeka` prefix; compatibility mode uses `centrifugo`. PG `table_prefix` is `cf` in both modes (not a brand name) |

Open questions answered by recording ref behavior: OQ-ENG-02 (cluster detection, `rc` with a plain address), OQ-ENG-03 (CS-ENG-03b), OQ-ENG-04 (memory history TTL, CS-HIST-02 on m), OQ-ENG-05 (CS-HIST-03, CS-ENG-02 on n/p), OQ-HIST-01 (epoch shape per backend, recorded before normalization in a sut-only note), OQ-HIST-03 (CS-REC-04), INFRA-CONFLICT-15 (CS-PRES-02). Answers go to `open-questions.md` as "observed on ref v6.10.3 (black box)".

## Harness changes
- **Profiles:** `COMPAT_PROFILE` (default `m`). `harness.Profiles(t, "m", "r", …)` skips a scenario outside its profiles. The doc's `COMPAT_ENGINE` is renamed accordingly.
- **Targets:** `RefNodes()` / `SUTNodes()` return one or two nodes; single-node scenarios use node A.
- **Golden (C3):** `testdata/golden/<profile>/`. The phase-1 files move to `golden/m/` unchanged. Recording takes an explicit scenario list (`COMPAT_RECORD=CS-HIST-01,…`); nothing re-records automatically. During review, `diff -r golden/m golden/<p>` shows what differs between engines (expected: nothing for the shared scenarios).
- **Backend control (C2):** `harness.Both(t, action)` runs `docker compose -p cedrazeka-compat-<profile> …` (restart, stop, start, kill, exec) for the ref side and the sut side and records the action once. `harness.Backend(side)` resolves a side's backend services and loopback ports from one table. There is no single-side backend action. Node-level actions (kill node B) also go through `Both`.
- **Isolation check CS-ISO-01 (C2):** runs first for every profile with backends; the profile aborts if it fails.
- **Outcome scenarios:** scenarios that cannot have a stable transcript (CS-REC-05, failover timing) record only their checked outcome (counts, codes), so diff and golden modes still apply.
- **Normalization:** no new rules expected. If one is needed, it is added to the table in `compatibility-suite.md` §3 in the same commit, as the spec requires. `info` node counts are **not** normalized.
- **Compose:** one file with Docker Compose `profiles:` per engine profile; `make compat-up PROFILE=r` / `make compat PROFILE=r` / `make compat-down PROFILE=r` (`down -v` between profiles flushes backends, spec §5). `make compat-all` loops over all profiles.

## Steps
Each step is one commit made by the user (or a PR). Every step ends with `make compat` (diff) and `make compat-golden` green on all profiles introduced so far, plus the phase-1 profile.

### 2.0 Plan and decisions
This document with the D2-* decisions and conditions C1–C6; links to it from the feature matrix (implementation order, row 2) and from `compatibility-suite.md`. The full scenario catalogue is rewritten in step 2.11, when the scenarios exist.

### 2.1 Harness: profiles, nodes, golden layout, backend control
- Code: `harness/target.go` (profiles, node lists), `harness/run.go` (golden per profile), `harness/compose.go`, `scenarios/main_test.go` (wait for all nodes of the profile), profile tags on existing scenarios.
- Compose: per-target networks, the `m` profile, ports as above. Makefile: `PROFILE`, `compat-all`.
- Golden files move to `golden/m/` (content unchanged: a re-record must produce zero diff).
- `harness.Both`, `harness.Backend`, CS-ISO-01 (exercised from step 2.2, the first profile with backends).
- Exit: `m` green in diff and golden; harness unit tests for the profile filter and the record list.

### 2.2 Redis standalone (`r`)
- `configs/r/config.json`, Redis services (ACL user/password), node B for ref and sut.
- Tags on CS-PUB-01, CS-HIST-01, CS-REC-01/02, CS-PRES-01; golden recorded from ref.
- Check: `diff -r golden/m golden/r` reviewed and explained in the commit message.

### 2.3 New single-node scenarios
CS-HIST-02, CS-HIST-04, CS-REC-03, CS-REC-04, CS-REC-05, CS-PUB-03, CS-PUB-04, CS-PUB-05, CS-ENG-CFG on `m` and `r`; variants `r-opt`, `r-lists`.

### 2.4 Redis topologies (`rs`, `rc`, `rsh`)
Sentinel and Cluster containers announcing container addresses; sharding with two Redis. The phase-2 single-node scenarios run on them per the tables.

### 2.5 NATS (`n`, `n-raw`)
NATS + Redis presence; CS-HIST-03; raw mode (publish to a raw subject with `nats.go`, receive on a channel) and wildcard subscriptions; CS-PUB-04/05 on `n`.

### 2.6 PostgreSQL (`p`, `p-notify`)
PG broker + PG controller. Transactional publishing (KD-11): an application transaction calls the documented SQL surface through `psql`; commit → clients receive, rollback → nothing. The exact functions and tables come from the docs (`docs/server/engines.md` §PostgreSQL broker) and are checked against both servers. KD-11 becomes `accepted` when green.

### 2.7 Multi-node
CS-ENG-01..05 and CS-PRES-02 on the profiles in the table.

### 2.8 Failure
CS-ENG-06 (backend stop/start) and CS-ENG-07 (Sentinel failover).

### 2.9 Names
CS-NAME-02 (sut-default on `r` and `n`).

### 2.10 Library integration tests and CI
- Precondition (C5): report `docker system df` and free space; clean build cache only after the user confirms.
- `test/library/compose.isolated.yml`: Redis 8 standalone, Cluster, Sentinel, Valkey 9 in one network namespace (no host ports, as `test/integration`). The test container copies the centrifuge module from the read-only module cache at the pinned version and runs `go test -race -tags integration ./...`.
- First run = baseline (pass/fail set recorded in `docs/testing/baseline-upstream-tests.md`), the same way phase 1 recorded the server's own tests.
- CI (`test.yml`): the `compat` job becomes a matrix over profiles; a new `library` job. Images pinned by digest. Runtime per job measured and recorded.

### 2.11 Docs and phase report
- Feature matrix: rows to `verified` per the exit criteria; exceptions with reasons (below).
- `compatibility-suite.md`: catalogue, profiles, per-profile golden layout, status.
- `open-questions.md`: answers from ref observations; `known-differences.md`: KD-07, KD-11.
- `docs/operations/engines.md` (new): supported backend versions (what is actually tested), topologies, the "no mixed clusters" rule (ADR-0023), multi-node PostgreSQL needs the PG controller, presence on PostgreSQL is per node, NATS limits.
- Provenance §3 only if an upstream file changes (none expected).
- Phase report with exact commands and results.

## Exit criteria
1. Every scenario in the tables is green in diff mode and golden mode on each of its profiles, locally (3 consecutive `make compat-all` runs) and in CI.
2. Upstream server tests: unit and integration pass/fail sets equal the phase-1 baseline.
3. Library integration tests: pass/fail set recorded; failures, if any, are pre-existing at the pinned version (same set on a pristine copy) and documented.
4. Matrix rows → `verified`: OSS-ENG-01..06, 09..17; OSS-HIST-01..12; OSS-PRES-01..06; OSS-SUB-08, 09, 10, 14.
   Not verified in phase 2, with reasons recorded in the matrix: OSS-ENG-07 (managed services, D2-5/D2-6; Valkey via the library job only), OSS-ENG-08 (cloud IAM, D2-5), OSS-ENG-18 (map brokers → phase 3), OSS-ENG-19 (PRO-only consumers of the key), OSS-ENG-20 (unreleased; only "rejected without the flag"), OSS-PRES-07 (PRO).
5. Open questions OQ-ENG-02..05, OQ-HIST-01, OQ-HIST-03, INFRA-CONFLICT-15 have recorded answers.
6. Quality gates: gofmt, go vet, golangci-lint (server: new issues only; `test/compat`: full), govulncheck, licenses-check, harness unit tests.

## Out of scope
PRO engine features (per-namespace engines PRO-21, Redis/NATS controller PRO-22, replicas and sharded PUB/SUB PRO-32/33, PG replicas and fan-out PRO-26/40); map subscriptions and map brokers (phase 3); other transports (phase 3); load, soak and chaos beyond the two failure scenarios (phase 9); TLS to backends (phase 9, needs test certificates); DragonflyDB/KeyDB.

## New dependencies
| Item | Where | License | Note |
|---|---|---|---|
| `github.com/nats-io/nats.go` | `test/compat` | Apache-2.0 | already in the server's go.sum (v1.55.0) |
| `redis:8-alpine`, `nats:2.10-alpine`, `postgres:16`, `valkey/valkey:9-alpine` | test compose files only | Redis 8: RSALv2 / SSPLv1 / AGPLv3 (tri-license, not OSI open source under RSAL/SSPL); NATS server: Apache-2.0; PostgreSQL: PostgreSQL License; Valkey: BSD-3-Clause | pinned by digest; used only to run tests, never shipped (C1) |

## Resources
- Docker on the dev machine (2026-11-16): 3.8 GB free of 62.7 GB, 8 GB RAM, 11 CPUs. Profiles run one at a time and are torn down with `down -v` (only this project's compose volumes). New images total < 0.5 GB (redis 8-alpine, valkey 9-alpine; nats and postgres are cached). The full upstream integration run (Kafka, localstack) needs more space: before step 2.10 disk usage is reported, and `docker builder prune` (~5 GB) runs only after the user confirms (C5).
- The largest profile (`rs`): 2×2 server nodes + 2×(3 Redis + sentinel), well within 8 GB.

## Risks
| Risk | Mitigation |
|---|---|
| Cluster/Sentinel announce unreachable addresses inside Docker | announce the container IP; solved first in step 2.4 with a smoke check before scenarios |
| Multi-node timing (node discovery, presence propagation) makes transcripts unstable | wait for the expected state with deadlines; record only the final state; node counts compared, not timing |
| Failure scenarios are timing-dependent | outcome-only recording; generous deadlines; 3 consecutive local runs before commit |
| CI time grows with ~11 profiles | matrix in parallel; golden mode stays as the fallback |
| Library tests have pre-existing failures at the pinned pseudo-version | baseline first, compare sets (as phase 1) |
| Ref behavior contradicts the docs (e.g. NATS history) | ref behavior is the baseline for UPSTREAM rows; the doc conflict is recorded as INFRA-CONFLICT-nn |
