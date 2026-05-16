VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/MattCheramie/GopherTrunk/internal/version.Version=$(VERSION) \
           -X github.com/MattCheramie/GopherTrunk/internal/version.Commit=$(COMMIT) \
           -X github.com/MattCheramie/GopherTrunk/internal/version.BuildTime=$(BUILD_TIME)
TAGS    ?=

GO      ?= go
PKGS    := ./...

.PHONY: all build test test-dvsi test-integration integration integration-cc integration-cc-grant integration-cc-nxdn integration-cc-dmr integration-cc-dpmr integration-cc-edacs integration-cc-motorola integration-cc-tetra integration-cc-p25p2 integration-cc-mpt1327 integration-cc-ltr integration-cc-ysf lint tidy vet vulncheck licenses clean run proto cross-build release-archives release-dry-run web-build web-dev web-clean web-test

all: build

build:
	$(GO) build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" -o bin/gophertrunk ./cmd/gophertrunk

test:
	$(GO) test -tags "$(TAGS)" -race -count=1 $(PKGS)

# test-dvsi runs the DVSI hardware-backend tests under the -tags dvsi
# build. The Vocoder + Transport + USB-enumeration codepath is gated
# behind the build tag (patent-encumbered AMBE+2 hardware backend
# documented in docs/vocoders.md). The tagged unit tests use a
# scripted mock Transport and a software-loopback Transport so the
# wire protocol + Vocoder state machine + voice.Vocoder interface
# conformance all exercise in CI without a real DVSI USB-3000.
test-dvsi:
	$(GO) test -tags "dvsi $(TAGS)" -race -count=1 ./internal/voice/dvsi/...

# integration boots the wired daemon (no real SDR) end-to-end and asserts
# the engine + recorder + call log + metrics + API agree on a synthetic
# call. Build-tagged so default `make test` stays a fast unit run.
integration:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 ./cmd/gophertrunk/...

# test-integration is the full-tree variant — runs every
# integration-tagged test across the codebase, not just the ones in
# cmd/gophertrunk/. Future-proofs against integration-tagged tests
# landing in other packages without an explicit CI / Makefile change.
test-integration:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 $(PKGS)

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

# integration-cc-motorola boots the daemon with synthesized 3600-baud 2-FSK
# GFSK IQ (BT = 0.5) carrying a 24-bit outbound sync + 128-bit BCH(64, 16, 11)
# OSW pair encoding an OpSystemIDExtended announcement. Reuses the GFSKModulator
# from the EDACS PR; differences are framing + per-codeword BCH instead of
# whole-CCW BCH.
integration-cc-motorola:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesMotorola ./cmd/gophertrunk/...

# integration-cc-tetra boots the daemon with synthesized π/4-DQPSK IQ
# (18000 sym/s, α = 0.35) carrying a 38-dibit normal training sequence +
# 108-dibit SCH/HD burst encoding an MLE SYSINFO PDU through the full ETSI
# EN 300 392-2 §8.3.1 channel-coding chain. First integration test to
# exercise the π/4-DQPSK modulator primitive shipped alongside this PR.
integration-cc-tetra:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesTETRA ./cmd/gophertrunk/...

# integration-cc-p25p2 boots the daemon with synthesized H-DQPSK IQ
# (6000 sym/s, α = 0.20, π/8 rotation) carrying a 20-dibit outbound sync
# + 146-dibit trellis-coded MAC PDU and asserts the production
# newP25Phase2Pipeline + p25_phase2_trellis_mode + supervisor + API +
# metrics chain recovers the lock. Reuses the π/4-DQPSK modulator from
# PR #154.
integration-cc-p25p2:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesP25Phase2 ./cmd/gophertrunk/...

# integration-cc-mpt1327 boots the daemon with synthesized FFSK IQ
# (audio-band CCIR FFSK at 1200 baud — mark = 1200 Hz, space = 1800 Hz —
# inside an FM channel) carrying a BCH(63, 38)-encoded ALH codeword and
# asserts the production newMPT1327Pipeline + mpt1327_bch_mode + supervisor
# + API + metrics chain recovers the lock. First integration test to
# exercise the FFSK modulator primitive shipped alongside this PR.
integration-cc-mpt1327:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesMPT1327 ./cmd/gophertrunk/...

# integration-cc-ltr boots the daemon with synthesized sub-audible NRZ IQ
# (300 baud below ~300 Hz, FM-modulated) carrying back-to-back LTR Status
# words and asserts the production newLTRPipeline + supervisor + API +
# metrics chain recovers the lock. Last item on the "lights up live
# trunked reception" punch list — closes per-protocol coverage of the
# trunked-radio families gophertrunk decodes.
integration-cc-ltr:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesLTR ./cmd/gophertrunk/...

# integration-cc-ysf boots the daemon with synthesized 4800-baud C4FM IQ
# carrying back-to-back YSF FSW-bearing frames and asserts the production
# newYSFPipeline + supervisor + API + metrics chain recovers the lock.
# Same C4FM modulator as P25 P1 / NXDN / DMR / dPMR (480-dibit YSF frame
# layout with FSWPattern at offset 0); closes per-protocol coverage of
# the trunked-radio families gophertrunk decodes.
integration-cc-ysf:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesYSF ./cmd/gophertrunk/...

# integration-cc-grant extends the cc.locked path with the full
# status → IdentifierUpdate → GroupVoiceChannelGrant TSBK chain on
# the P25 Phase 1 control channel. Uses ccdecoder.SetTestFactory to
# install a stub pipeline that pumps the synthesized dibit stream
# directly into a real phase1.ControlChannel (bypassing the
# Mueller-Müller clock loop, which reliably lands cc.locked but not
# every subsequent FSW + NID + 98-dibit TSBK trellis window in one
# streaming pass). Verifies the band-plan dispatch, KindGrant
# publication with resolved FrequencyHz, scanner state-locked
# transition, and the metrics handler's grant counter.
integration-cc-grant:
	$(GO) test -tags "integration $(TAGS)" -race -count=1 -run TestDaemonCCDecodesP25Phase1GrantChain ./cmd/gophertrunk/...

vet:
	$(GO) vet -tags "$(TAGS)" $(PKGS)

lint: vet
	@command -v staticcheck >/dev/null && staticcheck $(PKGS) || echo "staticcheck not installed; skipping"

tidy:
	$(GO) mod tidy

# vulncheck runs golang.org/x/vuln/cmd/govulncheck against the
# project's direct + transitive dependencies. CI runs this on every
# PR; the binary lives at $(go env GOBIN)/govulncheck after
#   go install golang.org/x/vuln/cmd/govulncheck@latest
vulncheck:
	@command -v govulncheck >/dev/null || { \
	  echo "govulncheck not installed; run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	  exit 1; \
	}
	govulncheck ./...

# licenses regenerates the machine-readable transitive-deps inventory
# (THIRD_PARTY_LICENSES.csv) using google/go-licenses. The
# hand-curated direct-deps table in THIRD_PARTY_LICENSES.md stays in
# sync with go.mod; this target backstops it with the full graph.
# Requires:
#   go install github.com/google/go-licenses/v2@latest
licenses:
	@command -v go-licenses >/dev/null || { \
	  echo "go-licenses not installed; run: go install github.com/google/go-licenses/v2@latest"; \
	  exit 1; \
	}
	go-licenses csv ./... > THIRD_PARTY_LICENSES.csv 2>/dev/null || \
	  echo "go-licenses reported errors; review THIRD_PARTY_LICENSES.csv for what landed"

# cross-build produces a static, dependency-free binary for every
# common (OS, arch) pair the daemon supports. CGO_ENABLED=0 is safe:
# the project uses purego for every system FFI (ALSA / WASAPI /
# CoreAudio / IOKit / WinUSB) and modernc.org/sqlite for the call
# log, so no cgo is required. Each output lands under dist/.
GOOSES   ?= linux darwin windows
GOARCHES ?= amd64 arm64

cross-build:
	@mkdir -p dist
	@for os in $(GOOSES); do \
	    for arch in $(GOARCHES); do \
	        ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	        echo "  → CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch"; \
	        CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
	            $(GO) build -tags "$(TAGS)" -ldflags "$(LDFLAGS)" \
	            -o dist/gophertrunk-$$os-$$arch$$ext ./cmd/gophertrunk || exit 1; \
	    done; \
	done
	@ls -lh dist/

# release-dry-run rehearses the release.yml linux job locally so an
# operator can validate ldflags wiring + version-string injection
# before pushing a tag. Skips the GitHub Release publish step; just
# produces the tarball + checksum under dist/. Run with
#   make release-dry-run VERSION=v0.99.0
# to exercise a prerelease build; default VERSION picks up the same
# git-describe value the release workflow would compute.
release-dry-run:
	@echo "→ Rehearsing release build for version $(VERSION)"
	@echo "→ COMMIT=$(COMMIT) BUILD_TIME=$(BUILD_TIME)"
	@rm -rf dist/dry-run
	@mkdir -p dist/dry-run
	CGO_ENABLED=0 $(GO) build -trimpath \
	    -ldflags "$(LDFLAGS)" \
	    -o dist/dry-run/gophertrunk \
	    ./cmd/gophertrunk
	@echo "→ Built binary reports:"
	@dist/dry-run/gophertrunk version
	@cd dist/dry-run && sha256sum gophertrunk > SHA256SUMS && cat SHA256SUMS
	@echo "✓ Dry-run complete. Output: dist/dry-run/"

# release-archives wraps the cross-build outputs in per-target
# tarballs (linux/darwin) and zips (windows). Run `make cross-build`
# first, or chain: `make cross-build release-archives`.
release-archives:
	@command -v zip >/dev/null || { echo "zip not installed"; exit 1; }
	@cd dist && for f in gophertrunk-*; do \
	    case "$$f" in \
	        *windows*) zip -q $${f%.exe}.zip $$f ;; \
	        *) tar czf $$f.tar.gz $$f ;; \
	    esac; \
	done
	@ls -lh dist/

clean:
	rm -rf bin/ dist/

run: build
	./bin/gophertrunk

# --- Web UI -----------------------------------------------------------------
# The browser-side operator console lives in web/. It's a Vite + React + Tailwind
# SPA whose build output is plain HTML/JS/CSS — no Node.js runtime, no
# server-side rendering. The release pipeline ships web/dist/ alongside the
# daemon binary as a sibling `gophertrunk-web/` folder; operators open
# `index.html` in any browser and point it at their daemon.
#
# `make web-build` is the only target the release pipeline needs. Targets
# require a local Node.js (npm 9+ recommended); see web/README.md for the
# tested versions.

# web-build produces the shippable static bundle in web/dist/. Idempotent —
# re-running it without source changes is a no-op after the npm cache is warm.
web-build:
	@command -v npm >/dev/null || { echo "npm not installed; see web/README.md"; exit 1; }
	cd web && npm ci --no-audit --no-fund
	cd web && npm run build

# web-dev runs the Vite dev server with HMR. The dev server proxies
# /api/* and /metrics to a daemon listening on 127.0.0.1:8080 so the SPA
# can talk to a live backend without manual CORS config.
web-dev:
	@command -v npm >/dev/null || { echo "npm not installed; see web/README.md"; exit 1; }
	cd web && npm install --no-audit --no-fund
	cd web && npm run dev

web-clean:
	rm -rf web/dist web/node_modules web/dev-dist

# web-test runs Vitest in CI mode against the SPA panels. Uses
# `npm install` (not `npm ci`) so devs that already ran web-dev
# don't pay the full reinstall cost.
web-test:
	@command -v npm >/dev/null || { echo "npm not installed; see web/README.md"; exit 1; }
	cd web && npm install --no-audit --no-fund
	cd web && npm test


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
