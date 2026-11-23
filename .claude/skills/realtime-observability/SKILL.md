---
name: realtime-observability
description: Implement or validate Cedrazeka tracing, ClickHouse analytics, realtime performance or dashboards.
---

# Realtime observability

Start from ADR-0012/0018, the relevant PRO contract and the existing metrics. Define what is counted and at which lifecycle point. Check cardinality and redact frame credentials before export. Exercise backpressure and cancellation when the exporter is unavailable. For performance changes use the same workload and runtime for both measurements and include allocations and concurrency settings.
