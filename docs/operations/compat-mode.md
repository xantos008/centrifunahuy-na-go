# Names and Centrifugo compatibility mode

Cedrazeka uses its own names by default. The **compatibility mode** switches operator- and wire-facing names to the upstream Centrifugo ones, so existing deployments, dashboards, gRPC clients and proxy backends keep working (ADR-0022). Config file keys and HTTP API paths are the same in both modes.

## Selecting the mode
- Flag: `cedrazeka --compat=centrifugo …` (works for every subcommand).
- Or env: `CEDRAZEKA_COMPAT=centrifugo`. The flag wins over the env.
- Any other value is a startup error. Empty means default mode.

## What changes

| Surface | Default mode | `--compat=centrifugo` |
|---|---|---|
| Env vars | `CEDRAZEKA_*`, `CEDRAZEKA_VAR_*`, `CEDRAZEKA_CHANNEL_NAMESPACES_<NAME>_*`… | `CENTRIFUGO_*` (same structure) |
| Prometheus metric prefix (not configurable) | `cedrazeka_*` | `centrifugo_*` |
| Default Redis / NATS key prefix, Graphite prefix | `cedrazeka` | `centrifugo` |
| Consumer header/field defaults (Kafka, NATS JetStream, Redis Streams, …) | `cedrazeka-method`, `cedrazeka-channels`, … | `centrifugo-method`, … |
| HTTP API error-mode header / gRPC metadata | `X-Cedrazeka-Error-Mode` / `x-cedrazeka-error-mode` | `X-Centrifugo-Error-Mode` / `x-centrifugo-error-mode` |
| gRPC server API service | `cedrazeka.api.v1.Api` | both `cedrazeka.api.v1.Api` and `centrifugal.centrifugo.api.CentrifugoApi` |
| Uni gRPC service | `cedrazeka.unistream.v1.UniStream` | both names |
| gRPC proxy calls to your backend | `cedrazeka.proxy.v1.Proxy` | `centrifugal.centrifugo.proxy.CentrifugoProxy` |
| OTel service name (unless `OTEL_SERVICE_NAME`), span attributes, Kafka client ID | `cedrazeka`, `cedrazeka.channel` | `centrifugo`, `centrifugo.channel` |
| Binary, Docker image, `version` output, product name | `cedrazeka` / Cedrazeka | same |

Values you set explicitly are never rewritten in either mode.

## Notes
- In default mode, `CENTRIFUGO_*` variables are **not read**, and each one is logged as an unknown environment variable at startup.
- Schemas for generating gRPC clients or proxy servers in default mode: `api/proto/cedrazeka/`. Message types are identical to upstream.
- gRPC server reflection lists only the upstream service descriptors (KD-26).
- Never point Cedrazeka and Centrifugo nodes at the same Redis/NATS/PostgreSQL prefix: mixed clusters are not supported in either mode (ADR-0023).
