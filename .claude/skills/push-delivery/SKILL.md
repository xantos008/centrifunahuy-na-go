---
name: push-delivery
description: Design and verify Cedrazeka push device lifecycle, provider contracts and queue workers.
---

# Push delivery

Read ADR-0013 and the matching push rows in the feature matrix. Model device ownership, token rotation and revocation before wiring providers. Use a local fake for provider responses and verify bounded retries, deadline, cancellation and idempotency at the queue boundary. Keep delivery state distinct from provider acceptance and avoid exposing tokens in diagnostics.
