# GopherTrunk daemon — multi-stage Docker build.
#
#   Stage 1 (builder)  pure-Go build (CGO_ENABLED=0). No C toolchain
#                      or system libraries required.
#   Stage 2 (runtime)  carries only the daemon binary on a slim base.
#
# USB pass-through is the operator's responsibility; see
# docs/hardening.md for the udev + docker run / compose recipe.

FROM golang:1.24-bookworm AS builder
WORKDIR /src

# Cache deps before copying the rest of the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=docker
ENV CGO_ENABLED=0
RUN go build -trimpath \
        -ldflags "-s -w -X github.com/MattCheramie/GopherTrunk/internal/version.Version=${VERSION}" \
        -o /out/gophertrunk ./cmd/gophertrunk

# ---------------------------------------------------------------

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

# Non-root user. /dev/bus/usb access is configured at runtime via the
# host's udev rules; see docs/hardening.md.
RUN useradd --system --create-home --shell /usr/sbin/nologin gopher
USER gopher
WORKDIR /home/gopher

COPY --from=builder /out/gophertrunk /usr/local/bin/gophertrunk

# Default ports: HTTP API on 8080, gRPC on 50051. Override with config.
EXPOSE 8080 50051

ENTRYPOINT ["/usr/local/bin/gophertrunk"]
CMD ["run", "-config", "/etc/gophertrunk/config.yaml"]
