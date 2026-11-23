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

- 3.2 started with CS-RPC-01. On the ping-only `m` profile, the custom ping method returns an empty RPC result. The default name and an unknown method return 104. The scenario matched Centrifugo OSS v6.10.3 in diff mode and passed against the recorded transcript on `m`.
- 3.1 started with CS-TRANS-01. The unidirectional SSE and HTTP-stream POST endpoints send a connect push with the requested subscription, then deliver an API publication. Both passed in diff and golden mode on `m`.
- 3.1 now includes the keyed server gRPC API in CS-API-03. It compares requests and replies for all 21 OSS methods, a missing API key and the 19 PRO-only methods against the pinned reference on `m`. CORE-OQ-04 is resolved. TLS, reflection and transport error mode still need separate coverage.
- CS-TRANS-02 covers both unidirectional WebSocket connection forms on memory. A client connects with its request in the first frame or in `cf_connect`, receives a publication on its server-side subscription and observes a disconnect push followed by a WebSocket close frame with the same code and reason.
- CS-PROXY-01..03 passed in diff and golden mode on `m`, `r` and `p`. They exercise connect, subscribe and publish callbacks over HTTP on memory and PostgreSQL and over gRPC on Redis. The tests cover the connect cache override, presence metadata, subscribe approval and rejection, and publication rewriting, tags and history. PostgreSQL retained both delivered publications when the callback set `skip_history` on the second. Memory and Redis retained only the first.
- CS-RPC-02 passed in diff and golden mode on `r` and `p`. It exercises namespaced RPC callbacks over gRPC on Redis and HTTP on PostgreSQL. It checks echoed data and application error 470. With the RPC proxy enabled for `probe` only, an unknown unqualified method returns 108 and an unknown namespace returns 104.
- CS-PROXY-04 and CS-PROXY-05 exercise server-managed connection and subscription refresh over gRPC on `r` and HTTP on `p`. The callbacks require JWT connection meta. The scenarios check the renewed metadata in presence, delivery after renewal, connection close 3005 and single-channel unsubscribe 2501.
- CS-RPC-03 uses the same two callback transports to check the one-second RPC deadline, temporary error 100, backend observation of cancellation, and another successful RPC on the same WebSocket.

## Remaining work

Phase 3 is open. Continue 3.2 with named proxy selection and gRPC subscription streams. The RPC deadline scenario covers cancellation triggered by the deadline. Peer disconnect during a pending callback and retry after a refresh backend failure still need independent evidence. Step 3.1 still needs the gRPC configuration variants listed above and the remaining transports from its scope. Steps 3.3 and 3.4 need consumer delivery, retries and acknowledgement scenarios, and coverage for personal channels, map subscriptions, shared poll and subscription streams. Finish with the 3.5 acceptance review and merge the completed phase into `main` before branching for phase 4.
