# ADR-0004: Transport layer

- Status: superseded by ADR-0002 (inherited)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
Transports are inherited from OSS v6.10.3 and centrifuge. That includes the in-tree WebSocket implementation `internal/websocket` (a fork of gorilla/websocket, BSD-2-Clause; see provenance.md), plus quic-go v0.64.1 and webtransport-go v0.13.0. The library spike is cancelled. There is no bidirectional gRPC transport (ADR-0021).

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-TRANS-01..16

## Context
Bidirectional transports:
- WebSocket: permessage-deflate, decompression limit, RFC 8441 over HTTP/2 (experimental).
- HTTP-streaming and SSE, both with an `/emulation` back channel routed to the owning node.
- WebTransport (experimental, HTTP/3).

Unidirectional transports: uni WS, uni SSE, uni HTTP-stream and uni gRPC.

There is **no bidirectional gRPC client transport** in OSS (CORE-OQ-20). SockJS is removed in v6.

## Decision
- One `Transport` interface: `Name()`, `Protocol()`, `Unidirectional()`, `Emulation()`, `Write(frames)`, `Close(disconnect)`. The session layer never sees `net/http`.
- WebSocket library: `github.com/coder/websocket` vs `github.com/gorilla/websocket` will be decided by a Phase 1 spike. Hard requirements:
  - permessage-deflate with a configurable level;
  - prepared-message reuse (PRO-34 compression cache);
  - read limits;
  - access to HTTP/2 extended CONNECT.
  gorilla is the current default candidate (BSD-3, compression support, the same lineage as the reference). The spike result becomes ADR-0004a.
- HTTP/3 and WebTransport use `quic-go` + `webtransport-go` (MIT). They sit behind a build/feature flag and are delivered last (experimental upstream).
- uni gRPC uses the public `unistream.proto` (Apache-2.0, OSS) with `google.golang.org/grpc`.
- Emulation is a node-addressed control message over `Controller` (ADR-0007), keyed by `node` + `session` from ConnectResult.

## Alternatives
- gobwas/ws (zero-copy, lower-level): more code for compression. Fallback if the benchmarks demand it.
- A bidirectional gRPC transport: not in baseline. Only if the user requests it (known-differences entry).

## Consequences
Uni transports reuse the same session core with `Unidirectional()=true`. That means connect-error → disconnect-code mapping (`connect_code_to_unidirectional_disconnect`) lives in the session layer.

## Sources
docs/transports/*.md; OSS go.mod (quic-go, webtransport-go).
