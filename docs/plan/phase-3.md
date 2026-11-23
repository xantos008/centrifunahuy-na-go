# Implementation phase 3 — verify the remaining OSS core

Follow the implementation order in the [feature matrix](../compatibility/feature-matrix.md#implementation-order). This plan expands the remaining OSS work from SPEC §8 phases 2–3. Phase-2 evidence remains tracked in [its own plan](phase-2.md).

## Sequence

| Step | Scope | Evidence |
|---|---|---|
| 3.1 | Server API over gRPC and remaining transport profiles | Compare gRPC status and reply payloads, connection setup, push delivery and shutdown. Cover SSE, HTTP-stream, unidirectional WebSocket/gRPC, SockJS, emulation and WebTransport according to the pinned matrix and ADR-0021. |
| 3.2 | HTTP and gRPC proxies, subscription lifecycle and RPC | Compare callback request mapping, namespace selection, replies, errors, deadlines and built-in RPC ping with local fakes. |
| 3.3 | Async consumers | Use isolated provider emulators or local backends. Verify consumer-to-API delivery, retry and acknowledgement behavior. Record provider limitations separately. |
| 3.4 | Personal channels, map, shared poll and subscription streams | Run against the applicable memory, Redis and PostgreSQL profiles. Check push order, reconnect state, pagination and cross-node propagation. |
| 3.5 | Review and close the OSS core | Run affected diff and golden suites and the upstream server tests. Update the feature matrix and open questions. List any environment-dependent checks with their scope. |

Use [ADR-0028](../architecture/adr/0028-compatibility-test-suite.md) for transcripts. A package's inherited tests are supporting evidence. Mark a contract verified when its applicable reference comparisons pass. Track any product changes in their own commits with provenance where applicable.

The exit state is a stable OSS core with completed compatibility evidence for the relevant OSS-TRANS, OSS-PROXY, OSS-CONS, OSS-API-02 and OSS-SUB rows. DEC-09 defers Helm until this point. Phase 4 then begins the independently implemented authorization features.

## Progress

- 3.2 started with CS-RPC-01. A custom ping method returns an empty RPC result. The default name and an unknown method return 104. The scenario matched Centrifugo OSS v6.10.3 in diff mode and passed against the recorded transcript on `m`. Proxy RPC and the remaining lifecycle cases are pending.
- 3.1 started with CS-TRANS-01. The unidirectional SSE and HTTP-stream POST endpoints send a connect push with the requested subscription, then deliver an API publication. Both passed in diff and golden mode on `m`. The remaining transport and gRPC cases are pending.
- 3.1 now includes the keyed server gRPC API in CS-API-03. It compares requests and replies for all 21 OSS methods, a missing API key and the 19 PRO-only methods against the pinned reference on `m`. CORE-OQ-04 is resolved. TLS, reflection and transport error mode still need separate coverage.
