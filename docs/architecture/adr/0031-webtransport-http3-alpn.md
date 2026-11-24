# ADR-0031 WebTransport negotiates HTTP/3 in both naming modes

Accepted on 2026-11-24 for OSS-TRANS-06 and OSS-TRANS-07.

## Observation

With a static TLS certificate and WebTransport enabled, the pinned Centrifugo v6.10.3 listener fails the QUIC handshake with TLS alert 120 (`no_application_protocol`). The client offers `h3`. Both JSON and Protobuf connections stop before any application data is exchanged.

## Decision

Cedrazeka advertises HTTP/3 on the QUIC listener in its default and Centrifugo-compatible naming modes. The application passes the WebTransport server a TLS configuration prepared by `http3.ConfigureTLSConfig`. The library clones the listener configuration and installs the HTTP/3 ALPN identifier. The TCP listener continues to use its own configuration.

## Acceptance

On profile `rsh`, CS-TRANS-06 checks the pinned server against the recorded TLS alert. On both Cedrazeka modes it requires a verified TLS handshake, an HTTP/3 session, a client subscription and delivery from a second client over JSON and Protobuf. The two successful Cedrazeka transcripts must match. This exception is scoped to CS-TRANS-06. [Known differences](../../compatibility/known-differences.md)
