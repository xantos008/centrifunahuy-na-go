# ADR-0015: Authentication, authorization pipeline and CEL

- Status: proposed
- Date: 2026-11-15
- Feature IDs: OSS-AUTH-*, OSS-PERM-*, PRO-10, PRO-11, PRO-15, PRO-17, PRO-18, PRO-19, PRO-25

## Decision
- **JWT**: use a vetted library with an explicit algorithm allowlist (HS256/384/512, RS*, ES*, EdDSA per docs) and reject `none`. Candidates: `github.com/golang-jwt/jwt/v5` (MIT) or `github.com/cristalhq/jwt/v5` (MIT, used upstream). Choose after a fuzz and benchmark spike.
- **JWKS**: own cache. TTL and background refresh; keep the last good key set on fetch failure (documented upstream for the server API JWKS; client JWKS behavior is PRO-OQ-14). SSRF guard: https only by default, response size cap, timeout, no redirects to other hosts.
- **Pipeline**: an ordered, pure function `Decide(op, conn, channel, nsOpts) → allow|deny(code)`. It follows the documented subscribe order:
  1. private gate;
  2. token;
  3. user-limited;
  4. subscribe proxy;
  5. stream proxy;
  6. caps;
  7. allow_* options;
  8. insecure;
  9. CEL.
  It is table-tested against the channel_permissions doc.
- **Capabilities**: union of grants and never deny. Matchers are exact, wildcard (gobwas-glob-compatible semantics; verify) and regex (unanchored, per docs).
- **Channel patterns**: segment trie with static-over-variable precedence and backtracking. Conflicts are rejected at load. ASCII only.
- **CEL**: `github.com/google/cel-go` (Apache-2.0).
  - Compile at config load; cache the programs.
  - Limits: `CostLimit`, an interrupt check frequency with a context deadline, and an expression length cap.
  - Fail closed.
  - Variables are only those documented (`subscribed, user, meta, channel, vars, labels`).
- **Revocation/blocking**: in-memory sets refreshed from Redis/PG with periodic full load. They are checked on connect, refresh, subscribe and sub_refresh, and applied to live connections through Controller.

## Sources
docs/server/authentication.md, channel_token_auth.md, channel_permissions.md, docs/pro/{capabilities,channel_patterns,cel_expressions,access_revoke,client_authentication}.md.
