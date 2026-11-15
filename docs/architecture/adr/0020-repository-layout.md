# ADR-0020: Repository and module layout

- Status: proposed, revised 2026-11-15 (DEC-01)
- Date: 2026-11-15

## Decision
The upstream tree is imported at the repo root, because its `go.mod`, `main.go` and `internal/` expect that. Our files are added alongside it.

```
/                          upstream root (go.mod, main.go, Makefile, Dockerfile, .golangci.yml, buf.yaml, misc/)
├── internal/              upstream packages (modified only per ADR-0029)
│   ├── brand/             NEW: names + compatibility mode (ADR-0022)
│   └── pro/<module>/      NEW: PRO-equivalent modules (ADR-0029)
├── web/admin/             NEW (implementation phase 6): our admin UI
├── test/compat/           NEW: compatibility suite, separate Go module (ADR-0028)
├── third_party/           NEW, only if needed: vendored forks (e.g. centrifuge), each with LICENSE
├── upstream/BASE          NEW: imported upstream tag + commit
├── tools/                 our dev-tool module (exists)
├── docs/                  our docs (exists; upstream has no docs/ dir)
├── LICENSE                per ADR-0030 (DEC-10)
├── NOTICE                 NEW: attribution (Apache-2.0 §4(d))
└── THIRD_PARTY_LICENSES   NEW: license texts of embedded third-party code
```

- Go module path: `github.com/xantos008/centrifunahuy-na-go` (from `origin`), replacing `github.com/centrifugal/centrifugo/v6`. The rename is one mechanical commit, listed in provenance.md.
- Root files that collide with upstream are merged by hand in the import step:
  - `README.md`: ours, plus an "Includes Centrifugo OSS" attribution section;
  - `LICENSE`: per DEC-10;
  - `.gitignore`: union of both;
  - `CLAUDE.md` and `SPEC.md`: ours (upstream has none).
- Upstream-only tooling that is not ours is removed in a separate commit, recorded in provenance.md: packagecloud targets, upstream release scripts that publish to upstream channels, `.github/` workflows that push to upstream registries.
- Generated code stays committed. Generators are pinned in `tools/go.mod` (buf) or by the upstream `misc/scripts/generate.sh` versions.

## Sources
OSS v6.10.3 tree (root listing verified 2026-11-15).
