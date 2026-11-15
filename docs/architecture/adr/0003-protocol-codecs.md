# ADR-0003: Client protocol codecs (JSON + Protobuf)

- Status: superseded by ADR-0002 (inherited)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The client protocol codecs are inherited: `centrifugal/protocol` v0.23.1 (MIT) via `centrifuge`, which already implements raw-JSON `bytes`, strict decoding (6.10.0) and batch framing. We write no codec. The golden and fuzz requirements stay. They move into the compatibility suite (ADR-0028), and the upstream codec fuzz/unit tests are kept. Draft fields are already tolerated upstream.

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-PROTO-01..16

## Context
- Protocol v2 uses Command/Reply/Push.
- JSON batches are newline-delimited. Protobuf batches are varint length-delimited.
- JSON `bytes` fields carry **raw embedded JSON**, not base64, so a stock `protojson` codec is wrong on the wire.
- Strict JSON decoding is required since v6.10.0: duplicate keys, invalid UTF-8 and trailing data cause disconnect 3501.
- Hot-path encoding cost dominates fan-out.

## Decision
- Generate Protobuf types from `centrifugal/protocol@v0.23.1` `definitions/client.proto` (MIT, attributed) with `protoc-gen-go`, and lint/breaking-check them with `buf`.
- Write our own JSON encoder/decoder for the envelope:
  - The encoder is hand-written, appends to pooled buffers and writes `bytes` fields as raw JSON.
  - The decoder is strict and returns typed errors mapped to the code table.
- Encode a publication **once per protocol (and delta state)** per fan-out, not per client.
- Golden fixtures are captured from the reference server and official SDKs for both formats.
- Unknown and draft fields (`flag`, `dict`, `profile`) are decoded and ignored.

## Alternatives
- `protojson`: wrong `bytes` semantics, slow. Rejected.
- `encoding/json` reflection: lenient by default (duplicate keys accepted), slow. Rejected for the envelope; acceptable for config.
- easyjson/ffjson code generation: an extra generator and an unmaintained risk. Rejected.

## Consequences
The JSON codec needs fuzzing (strictness, round-trip) and golden tests. That is a quality gate in CLAUDE.md.

## Sources
docs/transports/client_protocol.md, protocol `definitions/client.proto` v0.23.1.
