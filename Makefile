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

# Regenerate Go bindings under internal/api/pb/v1 from proto/*.proto.
# Requires:
#   apt-get install -y protobuf-compiler           # /usr/bin/protoc
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
# Both go-installed binaries land under $$(go env GOBIN) (default
# $$HOME/go/bin); ensure that's on $$PATH for protoc to find them.
PROTO_OUT := internal/api/pb/v1

proto:
	@command -v protoc >/dev/null || { echo "protoc not installed"; exit 1; }
	@command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go missing; run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc missing; run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }
	mkdir -p $(PROTO_OUT)
	protoc -I proto \
	    --go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
	    --go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
	    proto/*.proto
