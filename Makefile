# Modified by the Cedrazeka authors, 2026. See NOTICE.
TESTFOLDERS := $(shell go list ./... | grep -v /misc/)
# Packages to test, all by default. To run a part of them like CI does:
#   make test-integration TEST_PACKAGES="$(bash misc/scripts/test_packages.sh integration)"
TEST_PACKAGES ?= $(TESTFOLDERS)

all: test

test:
	go test -count=1 -v $(strip $(TEST_PACKAGES)) -cover -race

test-integration:
	go test -count=1 -v $(strip $(TEST_PACKAGES)) -cover -race --tags=integration

generate:
	bash misc/scripts/generate.sh
	go generate ./...

swagger-web:
	make generate
	./misc/scripts/update_swagger_web.sh

sync-websocket:
	rsync -av --exclude '*/' ../centrifuge/internal/websocket/ internal/websocket/

update-deps:
	./misc/scripts/update-deps.sh

deps:
	go mod tidy

local-deps:
	go mod tidy
	go mod download
	go mod vendor

# Dev tools are project-local, pinned in tools/go.mod and built into .bin/
# (docs/ai/tooling-lock.md). Nothing is installed globally.
.PHONY: tools lint vuln licenses licenses-check compat-up compat-down compat compat-golden compat-record

# Compatibility suite (ADR-0028): ref = centrifugo v6.10.3 image, sut = our build.
# The sut binary is cross-compiled on the host and mounted into alpine, so
# nothing is downloaded or built inside Docker.
COMPAT_GOARCH ?= $(shell go env GOARCH)

compat-up:
	mkdir -p test/compat/.build
	GOOS=linux GOARCH=$(COMPAT_GOARCH) CGO_ENABLED=0 go build -o test/compat/.build/cedrazeka .
	cd test/compat && docker compose up -d --wait

compat-down:
	cd test/compat && docker compose down -v

compat:
	cd test/compat && COMPAT_MODE=diff go test -count=1 ./...

compat-golden:
	cd test/compat && COMPAT_MODE=golden go test -count=1 ./scenarios/

compat-record:
	cd test/compat && COMPAT_MODE=record go test -count=1 ./scenarios/


# THIRD_PARTY_LICENSES from the modules linked into the binary (offline; fails on unknown licenses).
licenses:
	go run scripts/thirdpartylicenses.go

licenses-check:
	go run scripts/thirdpartylicenses.go -check

tools:
	go -C tools build -o ../.bin/ github.com/golangci/golangci-lint/v2/cmd/golangci-lint golang.org/x/vuln/cmd/govulncheck github.com/bufbuild/buf/cmd/buf

lint: tools
	.bin/golangci-lint run --timeout 10m0s

vuln: tools
	.bin/govulncheck ./...

# Compares what clients, backends and operators observe between BASE (a git
# ref) and the current tree. See misc/snapshot/README.md.
#   make snapshot-diff BASE=master [SNAPSHOT_FLAGS="-with redis,postgres"]
snapshot-diff:
	BASE="$(BASE)" SNAPSHOT_FLAGS="$(SNAPSHOT_FLAGS)" bash misc/snapshot/snapshot_diff.sh

snapshot-test:
	cd misc/snapshot && go vet ./... && go test -count=1 ./...

# The module path is the repository address; the binary is always named cedrazeka.
build:
	CGO_ENABLED=0 go build -o cedrazeka .

