# ADR-0022: Own names by default, opt-in compatibility mode — DEC-04

- Status: **accepted** (user decision DEC-04, 2026-11-15). Names fixed by DEC-12 (2026-11-15): Cedrazeka.
- Date: 2026-11-15
- Feature IDs: OSS-CFG (env), OSS-OBS (metrics), OSS-API-02, OSS-PROXY (gRPC), OSS-CONS (Kafka headers), OSS-TRANS (uni gRPC); KD-17
- Legal notes: [naming-and-trademarks.md](../../legal/naming-and-trademarks.md)

## Context
The fork contains the upstream brand in several places with different weight.

**Wire contracts.** Renaming these breaks existing clients and backends:
- gRPC full service names, which appear in the HTTP/2 `:path`:
  - `centrifugal.centrifugo.api.CentrifugoApi`;
  - `centrifugal.centrifugo.proxy.CentrifugoProxy`;
  - `centrifugal.centrifugo.unistream.CentrifugoUniStream`.
- Kafka header names `centrifugo-*` (`internal/consuming/kafka.go`).

**Operator contracts.** Renaming these breaks existing deployments, not clients:
- the env prefix `CENTRIFUGO_` (`internal/config/config.go`: `envconfig` prefix, `CENTRIFUGO_VAR_`, k8s regex);
- the default metric namespace `centrifugo` (`internal/metrics`, already configurable upstream);
- the default key prefixes (`centrifugo` for Redis, NATS and PG);
- the binary name and CLI help.

**Product surface:** logs, version string, docs links, the embedded admin web UI (`internal/webui`, built from `centrifugal/web`), and the Swagger UI title.

**Not branded:** HTTP API paths and JSON fields, client protocol messages (the protobuf package name never goes on the wire for client frames), error and disconnect codes.

## Decision
1. **One source of names.** A new package `internal/brand` holds the default names as constants: product name, binary, env prefix, metric namespace, default key prefix and gRPC package names. Values (DEC-12): product "Cedrazeka", binary `cedrazeka`, env `CEDRAZEKA_`, metrics `cedrazeka`, key prefix `cedrazeka`, gRPC packages `cedrazeka.*.v1`.
2. **Default mode = own names.** Env `CEDRAZEKA_*`, metrics `cedrazeka_*`, key prefix `cedrazeka`, own gRPC package names (`cedrazeka.api.v1`, …), own Kafka header names.
3. **Compatibility mode** (`compat: centrifugo`) is off by default. It is selected by the flag `--compat=centrifugo` or the env `CEDRAZEKA_COMPAT=centrifugo`, and is read before the rest of config. When on:
   - the env prefix becomes `CENTRIFUGO_` (also `CENTRIFUGO_VAR_` and the k8s regex);
   - the default metric namespace becomes `centrifugo`;
   - default key prefixes become `centrifugo` (still no mixed clusters, ADR-0023);
   - the gRPC server API and uni gRPC additionally **register** the upstream service names, generated from the upstream `.proto` with the original package;
   - gRPC proxy calls use the upstream service name;
   - Kafka headers use the upstream names.
   
   Config file keys are identical in both modes; they are not branded.
4. Compatibility mode exists only for interoperability with existing clients, backends and deployments. It is never used in product marketing, binary names or UI.
5. The embedded upstream web UI (`internal/webui`, MIT, branded) is **removed** from the build in Phase 1 (ADR-0016). The Swagger UI stays (Apache-2.0); its title and spec metadata are rebranded.
6. **License and copyright notices are never renamed.** "Centrifugal Labs LTD" and the upstream LICENSE stay verbatim.

## Alternatives considered
- Keep upstream names (pure drop-in): conflicts with the trademark rule in CLAUDE.md. Rejected by the user.
- Own names only, no compatibility mode: breaks gRPC API clients and proxy backends generated from the upstream `.proto`. Rejected.

## Consequences
- Every renamed default is a compatibility-suite case: run in compatibility mode against upstream, and check the default-mode names.
- The renaming patch touches many upstream files, which is a merge cost on every upstream upgrade (ADR-0029). It is kept mechanical, with call sites going through `internal/brand`.

## Sources
OSS v6.10.3: `internal/config/config.go` l.231–443, `internal/metrics/metrics.go` l.166, `internal/consuming/kafka.go`, `internal/api/grpc.go`, `*.proto` package lines. User decision DEC-04.
