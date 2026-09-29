# Gat 1400 Simulator - project Makefile
#
# Targets:
#   make dev       - run protocol + bff in dev mode (foreground)
#   make build     - build binary to ./bin/gat1400-sim
#   make test      - run unit + integration tests
#   make e2e       - run end-to-end tests (requires a free port)
#   make lint      - run golangci-lint
#   make fmt       - go fmt ./...
#   make tidy      - go mod tidy
#   make docs      - regenerate docs (placeholder)
#   make release   - cross-compile for darwin/linux/windows
#   make clean     - clean build artefacts
#
# Variables (override on the command line):
#   BINDIR  output directory for build (default bin/)
#   LDFLAGS linker flags passed to go build (default -s -w)

BINDIR    ?= bin
LDFLAGS   ?= -s -w
PKG       := ./...
CMD       := ./cmd/gat1400-sim
BIN       := $(BINDIR)/gat1400-sim

GO        ?= go

.PHONY: dev build test e2e lint fmt tidy docs release clean

dev:
	$(GO) run $(CMD)

build: $(BINDIR)/$(notdir $(CMD))
$(BINDIR)/$(notdir $(CMD)):
	mkdir -p $(BINDIR)
	$(GO) build -ldflags '$(LDFLAGS)' -o $@ $(CMD)

test:
	$(GO) test -race -count=1 $(PKG)

e2e:
	$(GO) test -tags=e2e -race -count=1 ./test/e2e/...

lint:
	golangci-lint run $(PKG)

fmt:
	$(GO) fmt $(PKG)

tidy:
	$(GO) mod tidy

docs:
	@echo "docs are hand-maintained under docs/"

release:
	mkdir -p $(BINDIR)
	GOOS=darwin  GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/gat1400-sim-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/gat1400-sim-darwin-arm64  $(CMD)
	GOOS=linux   GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/gat1400-sim-linux-amd64    $(CMD)
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/gat1400-sim-windows-amd64.exe $(CMD)

clean:
	rm -rf $(BINDIR)