---
name: go-quality-gates
description: Run the mandatory quality gates (format, lint, tests, race, fuzz, vuln) before marking any Go task complete. Use after changing Go code.
---

# Go quality gates

Tools are pinned project-locally in `tools/go.mod` (see `docs/ai/tooling-lock.md`). Build once into the gitignored `.bin/`:

```sh
go -C tools build -o ../.bin/ github.com/golangci/golangci-lint/v2/cmd/golangci-lint golang.org/x/vuln/cmd/govulncheck github.com/bufbuild/buf/cmd/buf
```

```sh
gofmt -l . | tee /dev/stderr | (! read)                     # no unformatted files
go vet ./...
.bin/golangci-lint run ./...
go test ./<changed pkgs>/...
go test -race ./<concurrency-sensitive pkgs>/...
go test -run=^$ -fuzz=Fuzz<Name> -fuzztime=30s ./<pkg>     # parsers/auth/config/patterns/CEL
.bin/govulncheck ./...
.bin/buf lint                      # when .proto files change
```

Integration tests needing Redis/PostgreSQL/NATS/ClickHouse run against local containers only (never production DSNs).

Report the exact commands and their real output. If a gate was not run, say so.
