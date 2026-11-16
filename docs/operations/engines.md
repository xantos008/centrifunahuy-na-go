# Engines: backends and their licenses

Cedrazeka connects to Redis, NATS and PostgreSQL as external services. None of them is bundled with Cedrazeka: the binary, `Dockerfile`, release files and `THIRD_PARTY_LICENSES` contain only the Go client libraries (e.g. `rueidis`, Apache-2.0), not a server. Operators choose and license the backend they run.

| Backend | Server license | Where this project uses it |
|---|---|---|
| Redis 8 | tri-license RSALv2 / SSPLv1 / AGPLv3. RSAL and SSPL are not OSI open-source licenses; AGPLv3 is. Using Redis only for tests does not make its license irrelevant (phase-2 condition C1) | test compose files only (`test/compat`, image pinned in `docs/ai/tooling-lock.md`) |
| Valkey | BSD-3-Clause | planned: library integration job (phase-2 step 2.10) |
| NATS server | Apache-2.0 | planned: compat profiles `n`, `n-raw` (step 2.5) |
| PostgreSQL | PostgreSQL License | planned: compat profiles `p`, `p-notify` (step 2.6) |

## Redis topologies tested

Verified black-box against Centrifugo v6.10.3 in the compatibility suite (phase-2 step 2.4), Redis 8.10.3:

| Topology | Address form | Test setup |
|---|---|---|
| Standalone | `redis://user:password@host:6379` | profiles `r`, `r-opt`, `r-lists` |
| Sentinel | `redis+sentinel://user:password@sentinel:26379?sentinel_master_name=mymaster` (credentials are the master's) | 1 master, 2 replicas, 1 sentinel |
| Cluster | plain `redis://user:password@node:6379`; Cluster is detected automatically | 3 masters, no replicas |
| Sharding | `address` array of standalone addresses | 2 standalone Redis |

Sentinel and Cluster tell clients the addresses that the Redis nodes announce. Servers must be able to reach exactly those addresses. In containers this means the nodes must announce the network address the servers use, not 127.0.0.1.

Engine configuration and failure behavior are documented in [phase 2](../plan/phase-2.md) as the profiles are verified.
