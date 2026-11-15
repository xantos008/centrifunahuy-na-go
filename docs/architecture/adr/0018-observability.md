# ADR-0018: Observability

- Status: revised 2026-11-15 (DEC-01, DEC-04)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
Logging is the inherited zerolog v1.35.1, not slog. Prometheus (client_golang) and OTel are inherited. The default metric namespace comes from `internal/brand` (`cedrazeka` / `centrifugo` in compatibility mode). The cardinality guard applies to new PRO metrics.

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-OBS-*, PRO-09, PRO-28, PRO-35, PRO-38, PRO-39

## Decision
- Prometheus via `github.com/prometheus/client_golang`. Metric names follow the documented `centrifugo_*` families where compatibility matters (dashboards and alerts); naming is subject to user decision Q4.
- Cardinality guard: label values come only from bounded sets: namespace (resolved, not channel), command, code, transport, client name from an allowlist, and configured client labels with a value cap → `other`. No user or channel IDs ever.
- Native histograms supported; Summary metrics only where the docs still have them (deprecated).
- Tracing and metrics via the OpenTelemetry SDK with an OTLP exporter configured by the standard `OTEL_*` env.
- Logging uses stdlib `log/slog` with JSON/text handlers and a redaction helper for tokens, DSNs and keys.
- Sentry (PRO-35) is behind an interface and off by default.

## Sources
docs/server/observability.md, monitoring.md, docs/pro/observability_enhancements.md.
