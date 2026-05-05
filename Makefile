VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -X github.com/MattCheramie/GopherTrunk/internal/version.Version=$(VERSION)
TAGS    ?=

GO      ?= go
PKGS    := ./...

.PHONY: all build test lint tidy vet clean run proto

all: build

build:
	$(GO) build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o bin/gophertrunk ./cmd/gophertrunk

test:
	$(GO) test -tags "$(TAGS)" -race -count=1 $(PKGS)

vet:
	$(GO) vet -tags "$(TAGS)" $(PKGS)

lint: vet
	@command -v staticcheck >/dev/null && staticcheck $(PKGS) || echo "staticcheck not installed; skipping"

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin/ dist/

run: build
	./bin/gophertrunk

proto:
	@echo "proto generation lands in Phase 8"
