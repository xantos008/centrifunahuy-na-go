# ADR-0030: Project license after the fork — DEC-10

- Status: **accepted** (user decision DEC-10, 2026-11-15): option A, Apache-2.0
- Date: 2026-11-15

## Context
- The repo LICENSE is MIT, "Copyright (c) 2026 Igor Rykin".
- After DEC-01 most of the code will be Apache-2.0 upstream files, plus vendored components under other licenses:
  - gorilla/websocket, BSD-2-Clause;
  - envconfig, MIT;
  - Swagger UI, Apache-2.0.
- Apache-2.0 lets us license our own additions differently, but the imported files stay under Apache-2.0 with its notice and change-marking duties.

## Options
| Option | Notes |
|---|---|
| A. **Apache-2.0 for the whole repo** (our new code too); upstream copyright kept; add our own `NOTICE` | One license for one combined work. The patent grant is consistent. Simplest to audit. **Recommended** |
| B. Keep MIT for our new files, Apache-2.0 for upstream-derived files | Legal but noisy: a per-file license header is needed, and modified upstream files stay Apache-2.0 anyway |
| C. Dual license | No benefit identified |

## Decision (DEC-10)
The user's requirements: the software is free for anyone, and the responsibility for using it lies with the users, not with us. Apache-2.0 meets both. It allows use, modification and redistribution free of charge, including commercial use (§2). It ships "AS IS" with no warranty (§7), and the authors are not liable for damages (§8). The upstream base is Apache-2.0 anyway. No license removes every liability under every jurisdiction, but this is the standard disclaimer for open source.

Option A. Replace the root `LICENSE` with the Apache-2.0 text. Add `NOTICE` (our project, plus the attribution that the work includes Centrifugo OSS by Centrifugal Labs LTD under Apache-2.0) and `THIRD_PARTY_LICENSES`. The change happens in the import step of implementation phase 1, only after the user confirms.

## Sources
Apache License 2.0 §4; LICENSE files listed in provenance.md.
