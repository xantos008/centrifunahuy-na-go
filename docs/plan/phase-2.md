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
| C2 | Two independent Docker networks for ref and sut. Same hostnames resolve inside each network; no shared volumes for Redis, NATS, PostgreSQL; stop/restart actions are symmetric; tests cannot reach the other side's backend by accident | (a) Backend actions exist only as `s.Compose(action, base)` and `s.Exec(base, cmd…)` on a scenario `Session`: they resolve the service of the session's **own** side (`redis` → `redis-ref` or `redis-sut`); there is no API that names the other side. Every scenario runs the same code on ref and on sut, so actions are symmetric by construction, and each action is recorded as a note in the transcript, so a missing action on one side shows up as a diff. (b) An isolation check runs at the start of every profile with backends (`CS-ISO-01`): inside each server container `getent hosts redis` (and `nats`, `postgres`) resolves to an address of that side's backend; the two answers differ; `docker inspect` shows no volume or bind mount shared between the sides' backends; a probe publication through ref's backend (e.g. NATS raw subject or a Redis key) is invisible to sut and vice versa. The profile fails before any scenario if the check fails. (c) Backends have no host ports except the loopback ports listed below, and each port belongs to exactly one side |
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
- **Profiles:** `COMPAT_PROFILE` (default `m`). `harness.Profiles(t, "m", "r", …)` skips a scenario outside its profiles. The doc's `COMPAT_ENGINE` is renamed accordingly. Configs live in `configs/<profile>/` (the phase-1 `configs/memory` is now `configs/m`).
- **Targets:** `RefNodes()` / `SUTNodes()` return one or two nodes; single-node scenarios use node A.
- **Golden (C3):** `testdata/golden/<profile>/`. The phase-1 files move to `golden/m/` unchanged. Recording takes an explicit scenario list (`COMPAT_RECORD=CS-HIST-01,…`); nothing re-records automatically. During review, `diff -r golden/m golden/<p>` shows what differs between engines (expected: nothing for the shared scenarios).
- **Backend control (C2):** `s.Compose(action, base, args…)` runs `docker compose -f ../docker-compose.yml --profile <profile> <action> <service>` (restart, stop, start, kill) and `s.Exec(base, cmd…)` runs `exec -T`, both on the session's own side only. Node B is `node-b`. One compose project (`cedrazeka-compat`) with Compose `profiles:`; `down -v` removes only this project's volumes.
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
- `Session.Compose` / `Session.Exec` (side-bound); CS-ISO-01 lands in step 2.2, the first profile with backends.
- Exit: `m` green in diff and golden; harness unit tests for the profile filter and the record list.

**Done (2026-11-16).** Deviations and findings:
- `Both(t, action)` was replaced by side-bound `Session.Compose` / `Session.Exec`: a scenario cannot name the other side at all, which is stricter for C2, and the action note in the transcript makes asymmetry visible as a diff.
- One compose project with Compose `profiles:` instead of a project per profile; per-side networks `ref-net` / `sut-net`; config dir chosen by `COMPAT_PROFILE`.
- **Pre-existing flake fixed (test, not server):** CS-API-02 recorded `info.nodes[].uptime`, which both servers omit while a node is younger than 1 s (zero value). The golden came from an older ref, so golden mode failed when it ran right after startup; phase-1 CI never hit it because golden ran after the diff run. `uptime` is already volatile in the normalization table; it is now dropped like `metrics`. `golden/m/CS-API-02.json` re-recorded with `COMPAT_RECORD=CS-API-02`: the diff is exactly one removed line (`"uptime"`).
- Checks: harness unit tests (profile filter, record list, side-bound service names); `make compat-all` (m) 3× green; golden mode right after a fresh start 3× green; golangci-lint on test/compat 0 issues.

### 2.2 Redis standalone (`r`)
- `configs/r/config.json`, Redis services (ACL user/password), node B for ref and sut.
- Tags on CS-PUB-01, CS-HIST-01, CS-REC-01/02, CS-PRES-01; golden recorded from ref.
- Check: `diff -r golden/m golden/r` reviewed and explained in the commit message.

**Done (2026-11-16).**
- Redis `redis:8.10.3-alpine@sha256:7958cfbb…35c3` (same digest as `8-alpine` on that day), no persistence, `default` user off, ACL user `compat` with a test-only password; services `redis-ref` / `redis-sut`, alias `redis` on their own network only. Node B: `ref-b` (18100), `sut-b` (28100). Servers wait for their side's Redis healthcheck (`depends_on … required: false`, so profile `m` starts without it).
- `configs/r/config.json` = `configs/m` + `engine` (type `redis`, `presence_ttl` 5s) + `client.presence_update_interval` 2s.
- CS-ISO-01 in `harness/isolation.go`, called from `TestMain`; negative runs: attaching `sut` to `ref-net`, and pointing `sut-net`'s `redis` alias at `redis-ref`, both abort the profile with a CS-ISO-01 error. Probes exist for Redis only; NATS and PostgreSQL probes come with steps 2.5 and 2.6.
- Golden recorded from ref with `COMPAT_RECORD=CS-PUB-01,CS-HIST-01,CS-REC-01,CS-REC-02,CS-PRES-01`: all five files are byte-identical to `golden/m` (the engine does not show in these transcripts).
- Checks: `make compat-all` (m r) 3× green; golangci-lint on test/compat 0 issues. No server code changed.

### 2.3 New single-node scenarios
CS-HIST-02, CS-HIST-04, CS-REC-03, CS-REC-04, CS-REC-05, CS-PUB-03, CS-PUB-04, CS-PUB-05, CS-ENG-CFG on `m` and `r`; variants `r-opt`, `r-lists`.

**Done (2026-11-16).**
- **Scenarios:**
  - CS-HIST-02 and CS-HIST-04 in `history_test.go`;
  - CS-REC-03, CS-REC-04 and CS-REC-05 in `recovery_test.go`;
  - CS-PUB-03, CS-PUB-04 and CS-PUB-05 in `pub_test.go`;
  - CS-ENG-CFG in `engine_test.go`.
  - Profile tags follow the tables above. `history_remove` is not repeated in CS-HIST-02: CS-HIST-01 already covers it.
- **Harness:**
  - `Session.APIUntil`: polls without recording and records the last answer.
  - `WS.RecvUntil(marker)`.
  - `Session.StartFails(env…)`: `docker compose run --rm --no-deps` of the side's node A with env overrides; records the exit status and the fatal log line. New event kind `exit`.
- **Configs:**
  - New namespaces in `m` (copied to all Redis profiles): `hmeta`, `hlim`, `hnoperm`, `cache`, `acache`, `pos`, `delta`, `deltap`, `tags`, `ver`.
  - Global `client.history_max_publication_limit` and `client.recovery_max_publication_limit` set to 8.
  - All existing golden files pass unchanged with the new configs.
  - `r` = `m` + engine. `r-opt` = `r` + `presence_hash_field_ttl` + `presence_user_mapping`. `r-lists` = `r` + `history_use_lists`. All are generated with `jq` from `m`.
  - Profiles `r-opt` and `r-lists` have 1 node each.
- **Golden differences between engines:**
  - `r` vs `m`: only the CS-REC-03 note (`restart redis` vs `restart node`).
  - `r-opt` CS-PRES-01 equals `m`.
  - `r-lists` CS-HIST-01 differs in one place: `reverse: true` returns `[1,2]` instead of `[3,2]`. Redis Lists do not support reverse order (docs, `engines.md` §`history_use_lists`; the option is deprecated). Ref and sut agree.
- **Test defects found and fixed before recording; no server cause:**
  1. CS-HIST-02 polled history while waiting for expiry, and every read extends the meta TTL (centrifuge `broker_history_stream.lua`, memory `historyHub`), so the position never expired. It now waits out the TTLs (see compatibility-suite.md §5).
  2. The first fixed waits were too tight for the memory engine's whole-second TTLs: `history_meta_ttl` was raised from 3s to 5s, with waits of 2.5 s and 6.5 s.
  3. CS-REC-04 on `r` gave a ref/sut diff: sut subscribes failed with 3004 right after CS-REC-03 restarted Redis. Ref logs show the same `pub/sub connection temporary unavailable` errors, and with equal time after a restart both sides match. Ref only had more time because its restart happened earlier. CS-REC-03 now waits until a probe subscription succeeds.
  4. Cache-mode subscribe requests lacked `recoverable: true`, which SDKs send together with `recover` for an empty `since`.
- **Answered (black box, ref v6.10.3):** OQ-ENG-04 (history part), OQ-HIST-01 (Redis), OQ-HIST-03; see open-questions.md.
- **Observed, not a difference:** in compatibility mode the sut's development-build warning names Cedrazeka. Per ADR-0022, logs are product surface and log texts are deferred to phase 9.
- **Checks:** `make compat-all` (m r r-opt r-lists) 3× green; golangci-lint on test/compat 0 issues. No server code changed.

### 2.4 Redis topologies (`rs`, `rc`, `rsh`)
Sentinel and Cluster containers announcing container addresses; sharding with two Redis. The phase-2 single-node scenarios run on them per the tables.

**Done (2026-11-16).**
- Topologies, one set per side, same image as `r`, every node only on its side's network:
  - `rs`: master `redis`, replicas `replica1`/`replica2` (`--replicaof redis 6379`), one `sentinel` (quorum 1, `resolve-hostnames yes`, `down-after-milliseconds` 2000 for step 2.8). Address `redis+sentinel://compat:…@sentinel:26379?sentinel_master_name=mymaster`; the URL credentials authenticate to the master, the sentinel has no password. The sentinel is healthy only once it has discovered both replicas.
  - `rc`: three masters `rc1..rc3` with `cluster-enabled yes`; a one-shot `rc-init-<side>` resolves the nodes with `getent` on its own network, runs `redis-cli --cluster create`, and exits once all three report `cluster_state:ok`. Servers wait for it (`service_completed_successfully`). Address: plain `redis://compat:…@rc1:6379`.
  - `rsh`: `redis` and `redis2`, `address` array of two.
- Addresses: Sentinel and Cluster report addresses of the side's own network (e.g. ref 172.18.x, sut 172.20.x in one run), never 127.0.0.1. No `announce-ip` setting is needed because nothing runs on the host network.
- Both servers log `engine_mode` `sentinel`, `cluster` and `sharded(2):standalone-x2`. On `rsh` both shards hold keys after the run (ref 3 + 5, sut 3 + 5).
- CS-ISO-01: profiles list their writable probe entry points (`probe`; cluster: any node, `redis-cli -c`). Every backend hostname, including replicas and the sentinel, is checked for network, resolution and mounts. A new harness test requires a probe for every profile with backends. Negative check: `sut-b` attached to `ref-net` on `rs` fails CS-ISO-01.
- Golden: `golden/rs` and `golden/rsh` (5 files each) and `golden/rc` (7 files: plus CS-HIST-02, CS-PUB-03) are byte-identical to `golden/r`.
- OQ-ENG-02 answered from ref: a plain address of one cluster node is detected as Cluster.
- **Checks:** `make compat-all` (m r r-opt r-lists rs rc rsh) 3× green; harness unit tests; golangci-lint on test/compat 0 issues. No server code changed.

### 2.5 NATS (`n`, `n-raw`)
NATS + Redis presence; CS-HIST-03; raw mode (publish to a raw subject with `nats.go`, receive on a channel) and wildcard subscriptions; CS-PUB-04/05 on `n`.

**Done (2026-11-16).**
- NATS `nats:2.15.3-alpine` (Apache-2.0, pinned by digest). Started with a test-only user, monitoring port for the healthcheck, no JetStream. Services `nats-ref`/`nats-sut` have the alias `nats` on their own network. Loopback client ports are 14222 (ref) and 24222 (sut).
- `configs/n` = `r` without `engine`, plus `broker` nats and `presence_manager` redis (`presence_ttl` 5s). `configs/n-raw` = `n` plus `allow_wildcards` and `raw_mode` with `channel_replacements` `{":": "."}`.
- Harness: profiles `n` (2 nodes) and `n-raw` (1 node), backends nats + redis. The CS-ISO-01 NATS probe subscribes on both sides and publishes through one; only that side may receive. `Session.NATSPublish`/`NATSSubscribe` use the session's own side (C2) and record notes. The test module gains nats.go v1.55.0, at the versions already pinned in the root `go.mod` (provenance PRV-14).
- Negative checks: `sut-b` on `ref-net` fails CS-ISO-01 (from step 2.4); pointing the sut NATS URL at ref's port makes the probe fail ("probe via nats-ref reached nats-sut").
- CS-HIST-03 (new): the catalogue said history on NATS is "rejected at config load". Ref shows otherwise: the server starts, and every history-based operation answers `108 not available` at runtime. The catalogue now follows ref.
- CS-PUB-06 (new, `n-raw`): raw message → channel push; API and client publish → raw subject unchanged; wildcard channel `wild-….*`. The push's `pub.channel` holds the NATS subject.
- Golden `n` vs `r`, every difference is explained by "NATS has no history":
  - CS-PRES-01 and CS-PUB-04 are byte-identical;
  - CS-PUB-01: publish results without offset/epoch, history `108`;
  - CS-PUB-05: no version suppression, and a repeated `idempotency_key` is delivered again (both servers).
- **Checks:** `make compat-all` (all 9 profiles) 3× green; harness unit tests; golangci-lint on test/compat 0 issues. No server code changed.

### 2.6 PostgreSQL (`p`, `p-notify`)
PG broker + PG controller. Transactional publishing (KD-11): an application transaction calls the documented SQL surface through `psql`; commit → clients receive, rollback → nothing. The exact functions and tables come from the docs (`docs/server/engines.md` §PostgreSQL broker) and are checked against both servers. KD-11 becomes `accepted` when green.

**Done (2026-11-16).**
- PostgreSQL `postgres:18.7-alpine` (PostgreSQL License, pinned by digest). Services `postgres-ref`/`postgres-sut` with the alias `postgres`, a test-only user, healthcheck `pg_isready`. Data was on tmpfs in step 2.6; step 2.8 moved it to a per-container anonymous volume (see there).
- `configs/p` = `r` without `engine`, plus `broker` postgres and `controller` postgres, both with `use_notify: false`. Presence is memory (no presence manager). `configs/p-notify` = `p` with `use_notify: true`; `pg_stat_activity` shows 2 LISTEN connections per side on `p-notify` and none on `p`.
- `sut-default` is not started on `p*`. The PG `table_prefix` is `cf` in both modes, so it would join the sut cluster through the controller as a third node. CS-ISO-01 skips it there (`noSUTDefault`).
- Harness:
  - CS-ISO-01 PG probe: a table created on one side must not exist on the other. Negative check: pre-creating the table on sut fails it.
  - `Session.SQL(stmts…)`: psql on the session's own side, all statements in one session; records statements and output, or the ERROR line. Notices are off.
  - `Session.AwaitDelivery()`: see the race below.
- **Race found and fixed (test, not server).** CS-HIST-04 once differed on `p`: a fresh subscriber got publications made just before it subscribed. A throwaway experiment (not committed): 50 rounds of "publish 3, subscribe at once". Ref pushed 149 of 150, sut 145. This is outbox behavior on both servers, not a difference. Scenarios that publish and then subscribe (`recoverAfter`, CS-HIST-04, CS-REC-04) now call `s.AwaitDelivery()`. On `p*` it waits 500 ms and records a note; elsewhere it is a no-op, so the `m`/`r` goldens did not change. `p` then passed 3× in a row.
- CS-PUB-07 (new, KD-11 → accepted):
  - commit delivers;
  - rollback leaves nothing, and its offset is reused by the next publication;
  - idempotency reports `t|idempotency`, version suppression reports `t|version`;
  - `_strict` raises `publish suppressed: version`;
  - top position and history removal behave as documented;
  - the `cf_stream_*` signatures are recorded.
  - The `p-notify` golden is byte-identical to `p`.
- Golden `p` vs `r` (all identical on ref and sut, explained in `operations/engines.md` §PostgreSQL broker):
  - position always tracked (offset/epoch in every push, an empty stream has no epoch);
  - `skip_history` has no effect;
  - pushes carry no `version` field;
  - the HIST-02 position survived meta TTL + 1.5 s. Corrected in step 2.7: this depended on the timing of the 1-minute cleanup, see there;
  - JSONB normalizes payloads and fossil delta is not sent;
  - CS-HIST-01 and CS-REC-05 are identical.
- **Checks:** `make compat-all` (all 11 profiles) 3× green; harness unit tests; golangci-lint on test/compat 0 issues. No server code changed.

### 2.7 Multi-node
CS-ENG-01..05 and CS-PRES-02 on the profiles in the table.

**Done (2026-11-16).**
- Harness:
  - `Session.OnB(fn)` runs calls and dials inside `fn` against node B of the session's own side (C2). Connections keep their node.
  - `Session.AwaitNodes(n, timeout)` polls `/api/info` and records the node count it reached.
  - `Session.ReplaceB(config)` stops B and starts a one-off copy of B, on the same port, with `configs/<config>`. Its test cleanup removes the copy and starts B again even after a failure, so later scenarios still find two nodes.
- Scenarios in `scenarios/multinode_test.go`. Each first waits for 2 nodes on A and on B. After a stop or kill, B is started again and the scenario waits for 2 nodes on both.
- **Test defects found while reading the first recordings, fixed before recording:**
  1. In v6, a subscriber gets join/leave only if it asks with `join_leave: true`.
  2. The subscriber's own join arrives first. Without reading it, the later reads were shifted by one and the leave was never checked. The transcript was deterministic all the same, which is why the recordings were read and not only compared.
  3. In CS-ENG-03b, an open connection that was not read during the 5 s waits was closed with `3012 no pong` (ping 2 s, pong timeout 1 s). The waits now come before the client connects.
- Golden:
  - `r`: 6 files.
  - `rs`, `rsh`: CS-ENG-01/02. `rc`: CS-ENG-01/02/05. `n`: CS-ENG-01..04. All byte-identical to `r`.
  - `p`: CS-ENG-01..05 and CS-ENG-03b. Differences from `r`, identical on ref and sut:
    - CS-ENG-01: offset/epoch in pushes (position always tracked).
    - CS-ENG-02: presence per node, each node sees only its own client.
    - CS-ENG-05: the empty stream's subscribe reply has no epoch; recovery on B still gives `recovered: true` with the three publications.
- **Answered (black box, ref v6.10.3):**
  - OQ-ENG-03: no startup error; no node discovery; delivery across nodes works.
  - OQ-ENG-05, multi-node part.
  - INFRA-CONFLICT-15: presence entries of a killed node expire within TTL + interval.
  - See open-questions.md.
- **Side observation:** `CENTRIFUGO_CONTROLLER_ENABLED=false` also drops the PG broker from the file config. Ref and sut behave identically; recorded in OQ-ENG-03.
- **Flaky test found by the control run, fixed in config (test, not server):** the first `make compat-all` failed CS-HIST-02 on `p` in golden mode. Diff mode had passed: ref and sut agreed in that run. The PG broker removes expired meta in a cleanup pass every `broker.postgres.cleanup_interval` (default 1m). Whether that pass fell inside the 6.5 s wait was chance, so the step 2.6 golden had caught a "position survived" run. `configs/p`, `p-notify` and `p-noctl` now set `cleanup_interval: "1s"`; both servers accept the key without a warning. CS-HIST-02 on `p` then passed 5× in diff mode with a reset position, and it was re-recorded (`COMPAT_RECORD=CS-HIST-02`). It equals `golden/r` except one epoch placeholder: a history read on PG does not create the new position, so the next publish gets another epoch. All other `p` goldens are unchanged.
- `configs/p-noctl` = `p` without `controller`. It is not a profile and is used only by CS-ENG-03b.
- **Checks:** `make compat-all` (all 11 profiles) 3× green; harness unit tests; golangci-lint on test/compat 0 issues. No server code changed.

### 2.8 Failure
CS-ENG-06 (backend stop/start) and CS-ENG-07 (Sentinel failover).

**Done (2026-11-16).** Scenarios are in `scenarios/failure_test.go`. Backend actions use `Session.Compose`/`Exec` on the session's own side (C2), so ref and sut go through the same sequence.
- **CS-ENG-06** (`r`, `n`, `p`): the broker backend (`redis`, `nats`, `postgres`) is stopped. The API publish is polled until it fails and only the last answer is recorded. Then one subscribe attempt is recorded. The backend is started; publish and subscribe are polled until they work (deadline 30 s). Finally a subscriber on a fresh channel receives an API publication. Observed, identical on ref and sut:
  - Publish while down: `100 internal server error`, `temporary: true`, on all three.
  - Subscribe while down: on `r` the connection is closed with `3004 internal server error`. On `n` it succeeds, because nats.go buffers subscriptions while it reconnects. On `p` it succeeds, because the PG broker subscribes locally; delivery is from the outbox.
  - After the start, everything works again within the deadline. Goldens: `n` differs from `r` only in the subscribe answer and the service name; `p` differs in the subscribe answer, the service name and offset/epoch.
- **CS-ENG-07** (`rs`): two publications, then `SENTINEL FAILOVER mymaster`. The test waits until the sentinel reports another master. Publishing works again (polled, 30 s), and history on the new master holds all three publications with the same epoch. The test then waits until delivery to a new subscriber has worked for 5 s without a miss (unrecorded probes at ~20/s, deadline 60 s), and records one delivered publication. The old master stays a replica until `compat-down`.
- **Infrastructure and test defects found, fixed before recording:**
  1. PostgreSQL data was on tmpfs. `docker stop` empties a tmpfs, so the restarted database had no schema. The servers create the schema only at startup, so neither recovered (`relation "cf_stream" does not exist` on ref). A real PostgreSQL restart keeps its data. Data now lives in the image's anonymous volume, one per container: the sides still share nothing (CS-ISO-01 checks mounts), and `compat-down` (`down -v`) removes it.
  2. Waiting for a subscribe *failure* timed out on `n`, where subscribes do not fail (see above). It is now one recorded attempt after the publish error.
  3. **Lost publications after a Sentinel failover (ref and sut alike).** The first control run failed CS-ENG-01 and CS-PUB-01 on `rs`, which run right after CS-ENG-07. CS-ENG-07 had checked publish and history, but not delivery. A throwaway probe (not committed) sent subscribe + API publish to fresh channels at ~20/s for 90 s after the failover. On both sides, from ~3.5 s to ~17.5 s after the failover, exactly one delivery in about every 1.07 s was lost: 14 on ref and 14 on sut, then none for 70 s. Waiting until the sentinel reported the failover finished did not shorten the window. Both logs show `pub/sub connection error EOF` at the failover; Redis PUB/SUB is at-most-once. The behavior is the same on both servers, so it is not a difference; the cause inside the shared library code was not investigated. CS-ENG-07 now waits for 5 s of loss-free delivery. 3 s was not enough: such a window could fit between the election and the first loss at ~3.5 s, and in 1 of 3 runs it did.
     Probe pitfalls on the way, both in throwaway code:
     - a gorilla/websocket connection is unusable after a read deadline expires, which first looked like "no delivery for 90 s";
     - an unthrottled probe (~500 connections/s) produced losses of its own and exhausted the host's ephemeral ports.
  4. The next control run failed CS-ISO-01 in the golden pass on `rs`, which reuses the stack after the diff pass. After the failover, `redis` is a replica, so the probe's `SET` there was refused. The `rs` probe now asks the side's sentinel for the current master and writes there (probe entry `sentinel`). Three failovers in a row on one stack (CS-ENG-07, diff, golden) then passed.
  5. On `p`, the publish that first succeeds after the restart can still be in the outbox and reach a new subscriber of the same channel as a push. The final check therefore uses a fresh channel.
- **Checks:** `make compat-all` (all 11 profiles) 3× green; harness unit tests; golangci-lint on test/compat 0 issues. No server code changed.

### 2.9 Names
CS-NAME-02 (sut-default on `r` and `n`).

**Done (2026-11-16).**
- `scenarios/names_test.go` `TestCS_NAME_02`, sut-only like CS-NAME-01 (skipped in record mode, no golden file). `sut` and `sut-default` share the sut side's backends. Each subscribes a client to its own channel; then the test checks the backends directly:
  - Redis presence key `<prefix>.presence.data.<ch>` exists for the mode's prefix and not for the other (both profiles);
  - the broker subscription for the client channel uses only the mode's prefix: Redis `PUBSUB CHANNELS <prefix>.client.*` on `r`, NATS monitoring `/subsz?subs=1` subject `<prefix>.client.<ch>` on `n`.
  - Default mode: `cedrazeka`; compatibility mode: `centrifugo`. PostgreSQL is not covered: `table_prefix` is `cf` in both modes.
- Negative check: with `COMPAT_SUT_DEFAULT_URL` pointed at the compatibility-mode server, the test fails on both profiles (wrong presence key, wrong broker subscription).
- **Checks:** see the commit; no server code changed.

### 2.10 Library integration tests and CI
- Precondition (C5): report `docker system df` and free space; clean build cache only after the user confirms.
- `test/library/compose.isolated.yml`: Redis 8 standalone, Cluster, Sentinel, Valkey 9 in one network namespace (no host ports, as `test/integration`). The test container copies the centrifuge module from the read-only module cache at the pinned version and runs `go test -race -tags integration` over the module with a bounded number of parallel tests and the root package split across processes.
- First complete run = baseline (pass/fail set recorded separately in `docs/testing/baseline-library-tests.md`), the same way phase 1 recorded the server's own tests.
- CI (`test.yml`): the `compat` job becomes a matrix over profiles; a new `library` job. Images pinned by digest. Record local duration in the library baseline and CI job durations after the first workflow run.

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

## Local revalidation on 2026-11-23

The handoff at `8d75f1b` had steps 2.0–2.9 committed and step 2.10 in the working tree. The separate library baseline is now recorded in [baseline-library-tests.md](../testing/baseline-library-tests.md). The compatibility workflow has one matrix job per engine profile and a separate library job.

| Check | Local result |
|---|---|
| `go test -count=1 -race -json $(bash misc/scripts/test_packages.sh unit)` | 549 upstream results match the phase-1 inventory, plus 9 passing Cedrazeka tests. |
| Isolated server Compose stack with `go test -count=1 -race -json -tags=integration $(bash misc/scripts/test_packages.sh integration)` | 299 pass and 2 skip, exact phase-1 inventory. |
| `make library-test` | 1,526 passes and five skips on the first complete run. A subsequent complete run exited 0 and matched the recorded inventory. |
| `go vet ./...`, `go -C test/compat vet ./...`, `go -C test/compat test -race ./harness`, `make lint`, `make licenses-check`, `.bin/govulncheck ./...` | Passed. No lint findings or vulnerabilities reported. |
| Redis release scope | The Dockerfile and release pipeline carry no Redis server. Snapshot CI uses Redis as a test service. THIRD_PARTY_LICENSES records the Go client. |

The first current compatibility run passed memory, all Redis profiles and both NATS profiles in diff and golden mode. PostgreSQL `p` passed diff and failed golden when CS-PUB-07 missed a WebSocket pong during its SQL sequence. The harness now reads concurrently during that sequence and records the buffered pushes at the original transcript position. Existing transcripts were preserved. The targeted rerun, `make compat-all COMPAT_PROFILES="p p-notify"`, passed both modes on both profiles. On `p-notify`, `go test -race -count=1 -run '^TestCS_PUB_07$' ./scenarios` also passed in diff and golden mode.

A second all-profile attempt exposed a missing cross-node join in NATS golden mode. The NATS broker queues subscriptions asynchronously. CS-ENG-01 now waits for both of the receiving node’s subjects to appear in the side’s NATS subscription monitor before the other node publishes. The rerun with `COMPAT_PROFILES="n n-raw p p-notify"` passed in diff and golden mode on all four profiles against the existing transcripts.

One local `make compat-all` exited 0 after both harness fixes. All 11 profiles passed in diff and golden mode in that run. The three consecutive runs recorded in steps 2.6–2.8 apply to the earlier harness. On 2026-11-24 an overlapping local memory test restarted the standard reference container while a repeat was exercising PostgreSQL. That interrupted attempt was discarded. After isolating the other stack, three consecutive complete `make compat-all` runs from an archive of `04b4414` each exited 0. Each covered all 11 profiles in both modes (66 profile/mode results across the three runs). The feature branch adds a Compose endpoint check before lifecycle actions and backend isolation probes. A separate complete run on the branch code at `3c57d44` passed all 11 profiles in diff and golden, including the additional positioning, cache and presence scenarios on their applicable profiles.

## GitHub Actions results for `04b4414`

The 2026-11-23 push to `phase-2/engines` passed all 14 jobs in 9m 27s. Each compatibility profile passed `make compat` against Centrifugo OSS v6.10.3 and `make compat-golden` against the recorded reference. The separate `snapshot` workflow runs on pull requests labeled `snapshot`. [GitHub Actions run](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597).

| Job | Duration | Result |
|---|---:|---|
| Go 1.27.4 tests | 4m 31s | `make test` and `make test-integration` passed with the race detector. [Job](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845695387) |
| Lint, vulnerabilities, licenses | 4m 05s | Formatting, server and compatibility module vet, lint, govulncheck and license checks passed. [Job](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696187) |
| `m` | 2m 16s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696231) |
| `r` | 2m 55s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696924) |
| `r-opt` | 1m 58s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696162) |
| `r-lists` | 2m 03s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845695885) |
| `rs` | 3m 10s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696092) |
| `rc` | 2m 33s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845695863) |
| `rsh` | 1m 43s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696040) |
| `n` | 1m 48s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696035) |
| `n-raw` | 2m 03s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696218) |
| `p` | 4m 24s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696100) |
| `p-notify` | 1m 51s | [Diff and golden passed](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696230) |
| Centrifuge library integration | 9m 19s | `make library-test` took 9m 07s and matched the recorded inventory of 1,526 passes and five skips. [Job](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696075) |

## Acceptance evidence before proxy verification (2026-11-24)

On `m`, `r` and `p`, CS-REC-06 covers automatic cache recovery for token-driven server-side subscriptions over WebSocket and unidirectional HTTP-stream. CS-PRES-03 checks error 103 from `presence` and `presence_stats` before subscription and checks both connection and subscription JWT metadata in the presence response. CS-REC-07 confirms close 3010 after a positioned server-side subscription loses its stream epoch on all three profiles. Of the 37 matrix rows in criterion 4, 32 are verified. The five open rows are OSS-HIST-05/11, OSS-PRES-04/06 and OSS-SUB-08. They still need the connect-proxy cache recovery override, a server scenario that reaches the configured recovery queue limit, connection JWT `b64info`, proxy-supplied presence metadata, and metadata on join/leave pushes. The `allow_presence_for_client` and `allow_presence_for_anonymous` flags also need dedicated scenarios. The pinned library suite covers the queue limit and overflow. The open-question register contains the reference observations for criterion 5. [Feature matrix](../compatibility/feature-matrix.md#phase-2-compatibility-evidence-2026-11-23) · [Open questions](../research/open-questions.md).

## Remaining acceptance evidence

On 2026-11-24, CS-PROXY-01 and CS-PROXY-02 passed in diff mode against Centrifugo OSS v6.10.3 and in golden mode on memory (`m`), Redis (`r`) and PostgreSQL (`p`). CS-PROXY-01 checks that the connect-proxy `subs[ch].cache_recover` override set to `false` returns no cached publications and set to `true` returns only the latest publication. Both values are checked on WebSocket and unidirectional HTTP-stream. CS-PROXY-01 and CS-PROXY-02 also cover connect- and subscribe-proxy metadata in presence responses.

Of the 37 matrix rows in criterion 4, 34 are verified. Three remain partially covered and carry the `inherited` status. [Feature matrix](../compatibility/feature-matrix.md#phase-2-compatibility-evidence-2026-11-23).

- OSS-HIST-11 needs a server scenario that reaches the configured recovery queue limit. The pinned library suite covers the limit and overflow.
- OSS-PRES-04 needs coverage for connection JWT `b64info` and both metadata fields in join/leave pushes.
- OSS-PRES-06 needs dedicated scenarios for `allow_presence_for_client` and `allow_presence_for_anonymous`.
