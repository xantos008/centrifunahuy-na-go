# Engines: backends and their licenses

Cedrazeka connects to Redis, NATS and PostgreSQL as external services. None of them is bundled with Cedrazeka: the binary, `Dockerfile`, release files and `THIRD_PARTY_LICENSES` contain only the Go client libraries (e.g. `rueidis`, Apache-2.0), not a server. Operators choose and license the backend they run.

| Backend | Server license | Where this project uses it |
|---|---|---|
| Redis 8 | tri-license RSALv2 / SSPLv1 / AGPLv3. RSAL and SSPL are not OSI open-source licenses; AGPLv3 is. Using Redis only for tests does not make its license irrelevant (phase-2 condition C1) | test compose files only (`test/compat`, image pinned in `docs/ai/tooling-lock.md`) |
| Valkey | BSD-3-Clause | planned: library integration job (phase-2 step 2.10) |
| NATS server | Apache-2.0 | planned: compat profiles `n`, `n-raw` (step 2.5) |
| PostgreSQL | PostgreSQL License | planned: compat profiles `p`, `p-notify` (step 2.6) |

Engine configuration and failure behavior are documented in [phase 2](../plan/phase-2.md) as the profiles are verified.
