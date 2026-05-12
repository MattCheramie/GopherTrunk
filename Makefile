VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -X github.com/MattCheramie/GopherTrunk/internal/version.Version=$(VERSION)
TAGS    ?=

GO      ?= go
PKGS    := ./...

.PHONY: all build test integration integration-cc integration-cc-nxdn integration-cc-dmr integration-cc-dpmr integration-cc-edacs lint tidy vet clean run proto

all: build

build:
	$(GO) build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o bin/gophertrunk ./cmd/gophertrunk

test:
	$(GO) test -tags "$(TAGS)" -race -count=1 $(PKGS)

# integration boots the wired daemon (no real SDR) end-to-end and asserts
# the engine + recorder + call log + metrics + API agree on a synthetic
# call. Build-tagged so default `make test` stays a fast unit run.
integration:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 ./cmd/gophertrunk/...

# integration-cc is the focused "lights up live trunked reception" check:
# boots the daemon with a mock SDR + a stubbed P25 Phase 1 pipeline factory
# that injects synthesized dibits into the real phase1.ControlChannel, and
# asserts the full chain above the IQ→dibit demod (supervisor →
# ccdecoder → state machine → bus → API + metrics) recovers the lock. The
# sibling `make integration` target covers a wider engine+recorder loop;
# this one isolates the CC-decoder critical path.
integration-cc:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesP25Phase1 ./cmd/gophertrunk/...

# integration-cc-nxdn is the NXDN-specific sibling of integration-cc. Boots
# the daemon with synthesized C4FM IQ replaying a SITE_INFO RCCH message
# through the NXDN-TS-1-A §4.5.1.1 spec FEC chain (`nxdn_viterbi_mode: spec`)
# and asserts the production newNXDNPipeline + the supervisor + API + metrics
# all recover the lock.
integration-cc-nxdn:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesNXDN ./cmd/gophertrunk/...

# integration-cc-dmr boots the daemon with synthesized 4800-baud 4-FSK IQ
# carrying a 132-dibit DMR Tier III burst (Aloha CSBK through BPTC(196, 96)
# inside a BS-Data sync + slot-type Hamming(20, 8) frame) and asserts the
# production newDMRTier3Pipeline + supervisor + API + metrics chain recovers
# the lock.
integration-cc-dmr:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesDMRTier3 ./cmd/gophertrunk/...

# integration-cc-dpmr boots the daemon with synthesized 2400-baud 4-FSK IQ
# carrying a 24-dibit FS3 sync + 40-dibit StandingServiceStatus CSBK and
# asserts the production newDPMRPipeline + supervisor + API + metrics
# chain recovers the lock.
integration-cc-dpmr:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesDPMR ./cmd/gophertrunk/...

# integration-cc-edacs boots the daemon with synthesized 9600-baud 2-FSK
# GFSK IQ (BT = 0.3, ±2.4 kHz deviation) carrying a 24-bit outbound sync
# + 40-bit BCH(40, 28, 2)-encoded CmdSystemID CCW and asserts the
# production newEDACSPipeline + edacs_bch_mode: on + supervisor + API
# + metrics chain recovers the lock. First non-C4FM protocol integration
# test; exercises the new GFSKModulator primitive in internal/dsp/demod.
integration-cc-edacs:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesEDACS ./cmd/gophertrunk/...

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
