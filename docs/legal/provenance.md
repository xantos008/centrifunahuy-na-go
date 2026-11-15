# Component provenance and license obligations

Status: **pre-import**. No upstream code is in the repo yet. This register lists what the Phase 1 import will bring in, verified against the upstream tree at `v6.10.3` on 2026-11-15. Each row is updated at import time with the exact commit and the first repo commit that contains it.

Rules (ADR-0002, user requirement E):
- **Never delete or rewrite** a license file, copyright line or license header of imported code. That includes "Copyright (c) Centrifugal Labs LTD" headers and the gorilla/Kelsey Hightower notices.
- An Apache-2.0 file that we modify gets this line under its existing header, never in place of it (Apache-2.0 §4(b)):

  `// Modified by the Cedrazeka authors, 2026. See NOTICE.`

- New files that we write carry our own header, per ADR-0030.
- Every upstream file we modify or delete is listed in §3, in the same commit as the change.
- A new third-party dependency (Go module, npm package, vendored code) needs a row in §1 or §2 and a license check before merge. Allowed: MIT, BSD-2/3, Apache-2.0, ISC, MPL-2.0 (file-level). Anything else needs an ADR. GPL is allowed only for dev tools that are not distributed (golangci-lint, already recorded).

## 1. Imported source (in-tree)

| ID | Component | Origin | Version / commit | License | Copyright notice to keep | Repo location | Our changes planned |
|---|---|---|---|---|---|---|---|
| PRV-01 | Centrifugo OSS server | github.com/centrifugal/centrifugo | tag `v6.10.3`, `4b6d86e23452a6fa1bd424ffe4526862efae85b8` | Apache-2.0 (root `LICENSE`, **no NOTICE file upstream**) | "Copyright 2023 Centrifugal Labs LTD" (LICENSE), "Copyright (c) Centrifugal Labs LTD" (`main.go` header) | repo root | module path rename, branding (ADR-0022), PRO wiring (ADR-0029), web UI removal |
| PRV-02 | WebSocket implementation (fork of gorilla/websocket, synced from centrifuge `internal/websocket`) | github.com/gorilla/websocket via centrifugal/centrifuge | as in PRV-01 | BSD-2-Clause (`internal/websocket/LICENSE`, `AUTHORS`) | "Copyright (c) 2013 The Gorilla WebSocket Authors" and the per-file headers | `internal/websocket/` | none planned |
| PRV-03 | envconfig (fork) | github.com/kelseyhightower/envconfig | as in PRV-01 | MIT (`internal/config/envconfig/LICENSE`) | "Copyright (c) 2013 Kelsey Hightower" | `internal/config/envconfig/` | none planned. The env prefix is passed in by callers |
| PRV-04 | Embedded admin web UI bundle (statik) | github.com/centrifugal/web `main` (built `dist/`) | as embedded in PRV-01 | MIT (GitHub API `spdx_id`, 2026-11-15) | in bundle | `internal/webui/statik.go` | **removed** in Phase 1: branded UI assets (ADR-0016, ADR-0022) |
| PRV-05 | Swagger UI bundle (statik) | github.com/swagger-api/swagger-ui `v4.18.1` | as embedded in PRV-01 | Apache-2.0 | in bundle | `internal/swaggerui/statik.go` | the spec title is rebranded through the generated `api.swagger.json`. The bundle itself is unchanged |
| PRV-06 | Google API annotation protos | github.com/googleapis/googleapis | as in PRV-01 | Apache-2.0 (file headers) | "Copyright 2015 Google LLC" | `internal/apiproto/swagger/google/api/*.proto` | none |
| PRV-07 | Server API / proxy / uni-stream schemas | part of PRV-01 | as in PRV-01 | Apache-2.0 | as PRV-01 | `internal/apiproto/api.proto`, `internal/proxyproto/proxy.proto`, `internal/unigrpc/unistream/unistream.proto` | in default mode, copies under our package names are generated next to the originals. The originals stay unchanged for compatibility mode (ADR-0022) |

## 2. Go module dependencies (not in-tree)

| ID | Module | Version | License | Use |
|---|---|---|---|---|
| PRV-10 | github.com/centrifugal/centrifuge | v0.40.2-0.20261031074606-ae446d43faf0 | MIT | core library, unmodified (ADR-0002) |
| PRV-11 | github.com/centrifugal/protocol | v0.23.1 | MIT | client protocol types and codecs |
| PRV-12 | all other modules in upstream `go.mod` | as pinned in `go.sum` | per module | listed with license text in `THIRD_PARTY_LICENSES`, generated at import. The generator tool is to be vetted (phase-1 plan step 1.6) |
| PRV-13 | github.com/centrifugal/centrifuge-go (test module only) | v0.13.0 | MIT | compatibility suite client. Not distributed in the binary |

## 3. Upstream files modified or deleted

Empty until import. Format: `path | change | commit | ADR`.

## 4. Attribution files to create at import

- `NOTICE`: our project name and copyright; "This product includes software developed by Centrifugal Labs LTD (Centrifugo, https://github.com/centrifugal/centrifugo), licensed under the Apache License 2.0"; and the PRV-02..06 attributions.
- `THIRD_PARTY_LICENSES`: full license texts for PRV-02, 03, 05, 06, 10, 11, 12.
- `upstream/BASE`: `centrifugo v6.10.3 4b6d86e23452a6fa1bd424ffe4526862efae85b8`.

## 5. Not imported, never to be imported

- Centrifugo PRO code or binaries, in any form.
- Centrifugo logos, trademarks and UI text (PRV-04 is removed for this reason).
- The upstream packaging and release scripts that publish to upstream channels (packagecloud, upstream Docker Hub). They are removed in a separate commit and listed in §3.
