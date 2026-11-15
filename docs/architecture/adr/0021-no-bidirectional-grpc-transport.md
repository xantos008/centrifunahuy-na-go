# ADR-0021: No bidirectional gRPC client transport — DEC-03

- Status: **accepted** (user decision DEC-03, 2026-11-15)
- Date: 2026-11-15
- Feature IDs: OSS-TRANS (uni gRPC), OSS-API-02 (gRPC server API); KD-08

## Context
SPEC mentions a "gRPC transport". OSS v6.10.3 has only:
- the **unidirectional** gRPC client transport (`unistream.proto`);
- the gRPC **server API** (`api.proto`);
- gRPC **proxy** calls to the backend (`proxy.proto`).

No bidirectional gRPC client transport exists upstream (CORE-OQ-20). There is no SDK for one.

## Decision
- Do not implement a bidirectional gRPC client transport.
- Keep the inherited uni gRPC transport, the gRPC server API and the gRPC proxy as they are.
- SPEC wording is corrected (SPEC §14). KD-08 becomes `accepted`.

## Alternatives considered
- Add one as an EXTENSION: no client SDKs, no upstream contract, extra attack surface. Rejected.

## Consequences
None for the transport layer: the fork already has exactly the upstream transport set.

## Sources
docs/transports/overview.md, docs/transports/uni_grpc.md at docs@69da714. OSS `internal/unigrpc/unistream/unistream.proto`.
