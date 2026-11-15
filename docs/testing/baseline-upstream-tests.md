# Upstream test baseline: Centrifugo OSS v6.10.3 pristine import

Recorded 2026-11-15 for phase 1, step 1.1. Every later step must reproduce exactly this pass/fail set (phase 1 exit criteria 2–3). The full per-test list is in [baseline/upstream-v6.10.3-tests.txt](baseline/upstream-v6.10.3-tests.txt). It holds top-level tests only, one line per test: `kind  package  test  result`.

## Code under test
- Upstream tag `v6.10.3`, commit `4b6d86e23452a6fa1bd424ffe4526862efae85b8`, fetched from github.com/centrifugal/centrifugo.
- The repo index equals the upstream tree for all 565 upstream paths (`git diff --cached --quiet refs/upstream/v6.10.3 -- <upstream paths>`; the local ref `refs/upstream/v6.10.3` keeps the fetched commit from being garbage-collected and is not pushed by default).
- Before the tests: `go build ./...` OK, `go mod verify` "all modules verified".

## Results

| Suite | Environment | Command | Packages | Top-level tests | Tests incl. subtests | Failures |
|---|---|---|---|---|---|---|
| unit | host, go1.27.4 darwin/arm64 | `go test -count=1 -race -json $(misc/scripts/test_packages.sh unit)` | 52 (32 with tests, 20 without) | 549 pass | 1081 pass | **0** |
| integration | container `golang:1.27.4-bookworm` linux/arm64, services per upstream compose | `go test -count=1 -race -json --tags=integration $(misc/scripts/test_packages.sh integration)` | 11, all pass | 299 pass, 2 skip | 442 pass, 2 skip | **0** |

Skipped tests:
- `internal/consuming` `TestKafkaConsumer_WorksCorrectlyInLoadedTopic`;
- `internal/consuming` `TestKafkaConsumer_RollingRestart`.

Both are skipped unconditionally upstream (`t.Skip()` at `internal/consuming/kafka_test.go:685` and `:1166`). This is not an environment problem.

Integration package times (s): app 21.5, consuming 23.2, pgmapbroker 7.1, pgstreambroker 3.7, pgoutbox 2.8, pgschema 2.8, controllers 2.6, natsbroker 1.4, redisqueue 1.2, redisnatsbroker 1.1, pgshard 1.1.

## How integration tests are run (isolated)
The upstream tests hardcode `localhost:6379`, `localhost:5432`, `localhost:4222`, `localhost:29092`, `localhost:8085` and `localhost:4566`. Those ports may be taken by other services on a developer machine. So the tests run with [test/integration/compose.isolated.yml](../../test/integration/compose.isolated.yml):
- All services and the test container share one network namespace (`network_mode: service:netns`). The hardcoded `localhost` addresses reach the test services, and **no host ports are published**.
- The service set equals upstream CI (`docker compose up` default profile), minus `postgres-replica`, which no test uses (no reference to port 5433 in the tests). Kafka's controller listener is moved from `kafka:9093` to `localhost:9093` because of the shared namespace.
- Modules come from the host module cache, mounted read-only, with `GOPROXY=off`. Run `go mod download` on the host first. Downloads from inside the container failed intermittently with "connection refused" on 2026-11-15.

```
go mod download
cd test/integration
export REPO=$(git rev-parse --show-toplevel) HOSTMOD=$(go env GOMODCACHE)
docker compose -f compose.isolated.yml up -d --wait
docker compose -f compose.isolated.yml --profile tests run --rm -T tests \
  sh -c 'go test -count=1 -race --tags=integration $(bash misc/scripts/test_packages.sh integration)'
docker compose -f compose.isolated.yml down -v
```

Images used:
- `alpine:3.24`
- `redis:7-alpine`
- `nats:2.10-alpine`
- `postgres:16`
- `gcr.io/google.com/cloudsdktool/google-cloud-cli:572.0.0-emulators`
- `localstack/localstack:4.14`
- `confluentinc/cp-kafka:8.0.0`
- `golang:1.27.4-bookworm`

They are tag-pinned as upstream does. Digest pinning comes with the CI work in step 1.9.
