# ADR-0028: Mandatory differential compatibility test suite

- Status: **accepted** (user requirement D, 2026-11-15)
- Date: 2026-11-15
- Feature IDs: every UPSTREAM row; PRO rows in doc-expectation mode
- Spec: [docs/testing/compatibility-suite.md](../../testing/compatibility-suite.md)

## Context
After the fork (ADR-0002), OSS behavior is inherited. Rebranding, PRO modules and upstream upgrades can still break it silently. The user requires one set of scenarios runnable against Centrifugo v6.10.3 and against our server, comparing:
- protocol;
- replies and errors;
- authorization;
- subscriptions;
- history, recovery, presence;
- server API.

## Decision
- A separate Go module `test/compat` keeps SDK and test dependencies out of the server module.
- **Targets:**
  - `ref`: the official image `centrifugo/centrifugo:v6.10.3`, pinned by digest `sha256:33fdbbce7d195dde316cbde17bc1d2a17b9e345de82799d1ec3c2bb8916e3803`;
  - `sut`: our build.
  
  Both are started by `test/compat/docker-compose.yml` with the same backends (Redis, NATS, PostgreSQL). `sut` runs in compatibility mode (ADR-0022), so env names, metrics and gRPC service names line up.
- **Scenarios** are plain Go functions `func(t *testing.T, tg Target) Transcript`. A Transcript is an ordered list of normalized wire events: frames sent and received, HTTP and gRPC requests and responses, disconnect codes.
- **Diff mode** (`make compat`) runs every scenario on both targets and diffs the normalized transcripts.
- **Golden mode** (`make compat-golden`) diffs `sut` against transcripts recorded from `ref` and committed in `testdata/golden/`. It runs where Docker Hub is unavailable. `-update` re-records only from `ref`.
- Normalization replaces only volatile values (client IDs, epochs, node IDs, timestamps, versions, ordering of unordered sets) with stable placeholders. Every rule is listed in the spec. Masking any other field needs a KD entry.
- A diff fails the build unless the scenario is mapped to an accepted entry in `known-differences.md`.
- PRO and EXTENSION scenarios run against `sut` only. PRO expected transcripts are hand-written from the docs, with source citations. EXTENSION scenarios also assert that default configs do not change any UPSTREAM transcript.
- Clients:
  - a raw protocol client (JSON and Protobuf framing) for byte-level checks;
  - the official SDK `centrifuge-go` v0.13.0 (MIT) for behavior checks;
  - `net/http` for the HTTP API;
  - a client generated from `api.proto` for gRPC.

## Alternatives considered
- testcontainers-go: a large dependency tree and a Docker socket requirement inside tests. docker compose plus env URLs is simpler. Rejected.
- A YAML scenario DSL: an extra interpreter to maintain. Go functions are enough. Rejected.
- Record-only golden tests, no live `ref`: misses changes on re-baseline. Golden mode is kept as a fallback, not the default.

## Consequences
- The suite is part of implementation phase 1 (harness plus the first scenarios). It gates every later phase.
- The upstream unit and integration tests inherited with the fork also stay green. They complement this suite but do not replace it, because they test internals, not the wire.

## Sources
docker.io/centrifugo/centrifugo:v6.10.3 (digest verified 2026-11-15); github.com/centrifugal/centrifuge-go v0.13.0 (MIT, 2026-10-21).
