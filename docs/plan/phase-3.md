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
- 3.1 now includes the keyed server gRPC API in CS-API-03. It compares requests and replies for all 21 OSS methods, a missing API key and the 19 PRO-only methods against the pinned reference on `m`. CORE-OQ-04 is resolved. CS-API-05 covers the transport options below.
- CS-TRANS-02 covers both unidirectional WebSocket connection forms on memory. A client connects with its request in the first frame or in `cf_connect`, receives a publication on its server-side subscription and observes a disconnect push followed by a WebSocket close frame with the same code and reason.
- CS-PROXY-01..03 passed in diff and golden mode on `m`, `r` and `p`. They exercise connect, subscribe and publish callbacks over HTTP on memory and PostgreSQL and over gRPC on Redis. The tests cover the connect cache override, presence metadata, subscribe approval and rejection, and publication rewriting, tags and history. PostgreSQL retained both delivered publications when the callback set `skip_history` on the second. Memory and Redis retained only the first.
- CS-RPC-02 passed in diff and golden mode on `r` and `p`. It exercises namespaced RPC callbacks over gRPC on Redis and HTTP on PostgreSQL. It checks echoed data and application error 470. With the RPC proxy enabled for `probe` only, an unknown unqualified method returns 108 and an unknown namespace returns 104.
- CS-PROXY-04 and CS-PROXY-05 exercise server-managed connection and subscription refresh over gRPC on `r` and HTTP on `p`. The callbacks require JWT connection meta. The scenarios check the renewed metadata in presence, delivery after renewal, connection close 3005 and single-channel unsubscribe 2501.
- CS-RPC-03 uses the same two callback transports to check the one-second RPC deadline, temporary error 100, backend observation of cancellation, and another successful RPC on the same WebSocket.
- CS-PROXY-06 compared named subscribe, publish and RPC routing on `r` (gRPC metadata) and `p` (HTTP paths and headers). CS-PROXY-07 compared both directions of gRPC subscription streams on `r`, including application rejection and interruption. Both passed diff and golden.

- CS-API-05 compared server reflection and Publish errors on plaintext `m` and authenticated TLS on `rsh`. It checked the descriptor, verified certificate chain, negotiated HTTP/2, status details and the per-call error-mode override.
- On `m`, CS-TRANS-03 compared binary delivery and stream termination on unidirectional gRPC. CS-TRANS-04 compared bidirectional emulation for SSE and HTTP-stream. CS-TRANS-05 confirmed that the removed SockJS endpoints return 404.
- CS-CONS-01 on Redis Streams (`r-opt`) traced deliveries to the subscriber and inspected acknowledged and retried queue entries. CS-CONS-02 on PostgreSQL (`p-notify`) compared a rollback, two ordered deliveries around an unknown command in one commit, final history and deletion of all committed outbox rows. CS-CONS-03 on NATS (`n`) checked durable redelivery and the acknowledgement floor. All three passed diff and golden.
- On Redis, CS-SUB-06 confirmed automatic personal subscriptions, replacement disconnect 3504 and cross-node delivery. CS-MAP-01 compared live changes and paginated recovery with the memory, Redis and PostgreSQL map brokers on `m`, `r-lists` and `p-notify`. All three yielded the same normalized transcript. CS-POLL-01 verified signed key tracking, cross-node versioned updates, reconnect and untracking on `r`. All three passed diff and golden.
- CS-RPC-04 on `r` used separate caller and observer sockets. Both the HTTP and gRPC backends confirmed that closing the caller canceled a pending callback before its 25-second deadline. CS-PROXY-08 on `r` and `p` observed the connection and subscription renewals retry after transient failures. Both scenarios passed diff and golden.
- On the isolated `cloud` profile, CS-CONS-04 compared SQS delivery and deletion for regular and SNS envelope messages. CS-CONS-05 compared ordered deliveries for method-attribute and command-envelope messages on the Google Pub/Sub emulator. Both passed diff and golden. CS-CONS-06 on the isolated Kafka profile delivered command and raw publication records and verified the committed partition offsets and zero lag for both consumer groups in diff and golden.
- CS-MAP-02 on `r` and `p` compared map publish and remove callbacks receiving base64 data and connection metadata. Rejected writes return 470. Approved writes reach the subscriber under the backend-selected key, although the writer receives error 107. CS-POLL-02 on `r` compared signed versionless tracking, a server-assigned publication version, disabled direct publishing and untracking. Both passed diff and golden.
- CS-TRANS-07 compared successful bidirectional WebSocket connections over HTTP/2 extended CONNECT on `m`. The unidirectional HTTP/2 handler returned while its connection goroutine still used the HTTP response writer, and both isolated processes exited. Cedrazeka now keeps the request alive. Ten positive regression runs delivered the unidirectional connect push and accepted a subsequent connection. [ADR-0032](../architecture/adr/0032-unidirectional-websocket-http2.md)
- CS-TRANS-06 verified the accepted WebTransport fix. The reference returned TLS alert 120. Both Cedrazeka modes negotiated HTTP/3 and delivered a publication to a second client over JSON and Protobuf.
- CS-PROXY-09 compared static callback values, emulated connect headers and WebSocket handshake headers. The HTTP and gRPC callbacks both observed the configured precedence.
- CS-MAP-03 on `r` compared the expiration of ephemeral map state and recovery of persistent map state. CS-MAP-04 confirmed two client entries for two sockets owned by the same user, one user entry, client removal on disconnect and eventual user expiry. Both passed in the full Redis profile in diff and golden modes.
- CS-PROXY-10 and CS-TRANS-08 on `m` compare default and configured callback status, temporary error, close code and HTTP response mappings. CS-TRANS-08 also checks the public init response. CS-TRANS-09 uses binary HTTP-stream commands, Protobuf emulation envelopes and non-UTF-8 publication bytes. All passed diff and golden.
- The server unit suite passed with the race detector (558 top-level tests). The isolated upstream integration suite passed 299 top-level tests with two Kafka skips also present on the pinned baseline. These counts exclude nested subtests.
- All 13 standard profiles passed in diff and golden modes. The final `m` and `rsh` runs used both transport fixes. The licensed Azure profile runs separately.
- After the final callback and test-client changes, full `m`, `r` and `p` passed diff and golden again. On Redis the map fixture uses a four-second deadline to distinguish application replies from a cold gRPC connection timeout. Dedicated RPC and stream cases retain their one-second deadlines.
- CS-CONS-07 passed diff, reference recording and golden on the local Azure profile. Each server consumed two AMQP messages on its own Service Bus emulator and SQL Server instance. Both subscribers observed offsets 1 and 2, and both active and dead-letter queues were empty after settlement.

## Local acceptance and Git status

| Step | Evidence |
|---|---|
| 3.1 | The keyed gRPC API and all remaining transports passed the applicable comparisons. ADR-0031 and ADR-0032 record the two approved transport fixes and their positive checks. |
| 3.2 | CS-PROXY-11..15 and CS-RPC-05 completed the named stream and renewal, token and personal subscriptions, HTTP and gRPC metadata, unqualified RPC and publication-option comparisons. Thirteen combinations of scenario and profile passed diff and golden, including regressions against three existing goldens. |
| 3.3 | All seven consumer backends passed delivery comparisons. The same retry and telemetry tests ran on Cedrazeka and a checkout of the verbatim v6.10.3 import. Both passed permanent and temporary publish errors, mixed broadcast retry, and annotation of an existing span. The official Azure emulator also confirmed in-flight settlement during graceful shutdown on both builds. |
| 3.4 | Personal subscriptions, map storage and recovery, shared poll and subscription streams passed on their applicable profiles. |
| 3.5 | Local evidence covers all phase-3 matrix rows. OSS-TRANS-08 (removed SockJS), OSS-PROXY-18 (PRO hooks) and OSS-PROXY-19 (no OSS unsubscribe/disconnect callback) are marked not applicable. |

Local acceptance covers 51 verified contracts and three inapplicable entries. The single pull request from `phase-3/oss-core` to `main` records the final CI results and merge. Phase 4 starts from the synchronized `main`.
