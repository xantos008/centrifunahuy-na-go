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
.PHONY: library-test tools lint vuln licenses licenses-check compat-build compat-up compat-down compat compat-golden compat-record compat-all

# Compatibility suite (ADR-0028): ref = centrifugo v6.10.3 image, sut = our build.
# The sut binary is cross-compiled on the host and mounted into alpine, so
# nothing is downloaded or built inside Docker. PROFILE selects the engine
# profile (docs/plan/phase-2.md); `compat-all` runs every profile in turn.
# Golden files are re-recorded only for the scenarios named in COMPAT_RECORD.
COMPAT_GOARCH ?= $(shell go env GOARCH)
PROFILE ?= m
COMPAT_PROFILES ?= m r r-opt r-lists rs rc rsh n n-raw p p-notify cloud kafka
COMPAT_ENV = COMPAT_PROFILE=$(PROFILE)
COMPAT_COMPOSE = cd test/compat && $(COMPAT_ENV) docker compose --profile $(PROFILE)

compat-build:
	mkdir -p test/compat/.build
	GOOS=linux GOARCH=$(COMPAT_GOARCH) CGO_ENABLED=0 go build -o test/compat/.build/cedrazeka .
	GOOS=linux GOARCH=$(COMPAT_GOARCH) CGO_ENABLED=0 go -C test/compat build -o .build/proxyfixture ./cmd/proxyfixture

compat-up: compat-build
ifeq ($(PROFILE),cloud)
	$(COMPAT_COMPOSE) up -d --wait pubsub-ref pubsub-sut
	go -C test/compat run ./cmd/pubsubsetup
endif
ifeq ($(PROFILE),p-notify)
	$(COMPAT_COMPOSE) up -d --wait postgres-ref postgres-sut
	$(COMPAT_COMPOSE) exec -T postgres-ref psql -X -v ON_ERROR_STOP=1 -U compat -d compat < testdata/postgres/consumer-outbox.sql
	$(COMPAT_COMPOSE) exec -T postgres-sut psql -X -v ON_ERROR_STOP=1 -U compat -d compat < testdata/postgres/consumer-outbox.sql
endif
ifeq ($(PROFILE),rsh)
	sh test/compat/configs/rsh/tls.sh
endif
	$(COMPAT_COMPOSE) up -d --wait

compat-down:
	$(COMPAT_COMPOSE) down -v
ifeq ($(PROFILE),rsh)
	$(RM) test/compat/configs/rsh/tls/ca.crt test/compat/configs/rsh/tls/server.crt test/compat/configs/rsh/tls/server.key
endif

compat:
	cd test/compat && $(COMPAT_ENV) COMPAT_MODE=diff go test -count=1 ./...

compat-golden:
	cd test/compat && $(COMPAT_ENV) COMPAT_MODE=golden go test -count=1 ./scenarios/

compat-record:
	@test -n "$(COMPAT_RECORD)" || { echo "COMPAT_RECORD=<scenario>[,…] is required: golden files are never re-recorded wholesale"; exit 1; }
	cd test/compat && $(COMPAT_ENV) COMPAT_MODE=record COMPAT_RECORD=$(COMPAT_RECORD) go test -count=1 ./scenarios/

compat-all: compat-build
	@set -e; for p in $(COMPAT_PROFILES); do \
		echo "== profile $$p"; \
		$(MAKE) --no-print-directory compat-down PROFILE=$$p >/dev/null 2>&1 || true; \
		$(MAKE) --no-print-directory compat-up compat compat-golden PROFILE=$$p || { $(MAKE) --no-print-directory compat-down PROFILE=$$p; exit 1; }; \
		$(MAKE) --no-print-directory compat-down PROFILE=$$p; \
	done

# THIRD_PARTY_LICENSES from the modules linked into the binary (offline; fails on unknown licenses).
licenses:
	go run scripts/thirdpartylicenses.go

licenses-check:
	go run scripts/thirdpartylicenses.go -check

tools:
	go -C tools build -o ../.bin/ github.com/golangci/golangci-lint/v2/cmd/golangci-lint golang.org/x/vuln/cmd/govulncheck github.com/bufbuild/buf/cmd/buf

# Upstream code is linted for new issues only, relative to the import commit
# (phase-1 plan); test/compat is our code and is linted fully.
LINT_BASE ?= 66d1418

lint: tools
	.bin/golangci-lint run --timeout 10m0s --new-from-rev=$(LINT_BASE) ./...
	cd test/compat && ../../.bin/golangci-lint run --timeout 10m0s ./...

# The centrifuge library's own integration tests at the pinned version and
# commit (condition C4; docs/testing/baseline-library-tests.md). Results go to
# test/library/out/ and are never mixed with the server's test results.
LIBRARY_VERSION := v0.40.2-0.20261031074606-ae446d43faf0
LIBRARY_COMMIT := ae446d43faf0e6f34619e42c11935340257669a7
LIBRARY_COMPOSE = LIBRARY_VERSION=$(LIBRARY_VERSION) LIBRARY_COMMIT=$(LIBRARY_COMMIT) HOSTMOD=$$(go env GOMODCACHE) docker compose -f test/library/compose.isolated.yml
library-test:
	@test "$$(go list -m -f '{{.Version}}' github.com/centrifugal/centrifuge)" = "$(LIBRARY_VERSION)" || { echo "go.mod centrifuge != LIBRARY_VERSION: update the pin and re-record the baseline"; exit 1; }
	go mod download && go mod verify
	# The library's own test-only dependencies; the container runs with GOPROXY=off.
	cd $$(go env GOMODCACHE)/github.com/centrifugal/centrifuge@$(LIBRARY_VERSION) && go mod download
	mkdir -p test/library/out
	$(LIBRARY_COMPOSE) up -d --wait || { $(LIBRARY_COMPOSE) down -v; exit 1; }
	$(LIBRARY_COMPOSE) --profile tests run --rm -T tests || { $(LIBRARY_COMPOSE) down -v; exit 1; }
	$(LIBRARY_COMPOSE) down -v
	go run test/library/summary.go test/library/out/library.json > test/library/out/library-tests.txt
	grep -c . test/library/out/library-tests.txt; grep -v 'pass$$' test/library/out/library-tests.txt || true
	diff docs/testing/baseline/library-$(LIBRARY_VERSION)-tests.txt test/library/out/library-tests.txt

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

