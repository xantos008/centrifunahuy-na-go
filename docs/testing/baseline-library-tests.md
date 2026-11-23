# Centrifuge library integration baseline

Recorded 2026-11-23 from an unmodified copy of `github.com/centrifugal/centrifuge` at `v0.40.2-0.20261031074606-ae446d43faf0` (commit `ae446d43faf0e6f34619e42c11935340257669a7`, module zip checksum `h1:iM7SvBHUfmQTOSSkQvfEDB/NN3m5y/gzn3Z4BKe+ONg=`). [Full test inventory](baseline/library-v0.40.2-0.20261031074606-ae446d43faf0-tests.txt).

Go 1.27.4 on Linux arm64, with the digest-pinned Redis 8 and Valkey 9 services in [the isolated Compose stack](../../test/library/compose.isolated.yml). The stack covers standalone Redis and Valkey, Redis Cluster and Sentinel, and Valkey Cluster. DragonflyDB cases inside the tests are excluded under D2-3. Go emits 1,531 top-level results including 1,301 in the root package.

| Passed | Skipped | Failed | Test events elapsed |
|---:|---:|---:|---:|
| 1526 | 5 | 0 | 6m 47s |

The five top-level skips are `TestFuzzLifecycleEnumerate`, `TestFuzzLifecycleReplay`, `TestPreShardedSlots`, `TestRedisMemoryUsage`, and `TestSharedPollEncodeFailureRollback_TODO`.

The first attempt at four shards produced a killed test process. The complete baseline used twelve root-package processes, `-parallel 2`, and the same race detector and integration tag. The baseline file was absent at the comparison step on this first complete run. After recording the inventory, the same `diff` command completed with exit code 0.

Run `make library-test` to reproduce and compare the exact inventory. It writes raw JSON, version information and the normalized inventory under `test/library/out/`. A second complete local run exited successfully and matched all 1,531 results. [Server baseline](baseline-upstream-tests.md).

The first GitHub Actions run for `04b4414` on 2026-11-23 matched the baseline exactly, with 1,526 passes, five skips and zero failures. The job took 9m 19s, including 9m 07s for `make library-test` and its comparison. [Library CI job](https://github.com/xantos008/centrifunahuy-na-go/actions/runs/45218406597/job/137845696075).
