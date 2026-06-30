# RF Scope web console

Standalone browser console for GopherTrunk's protocol-agnostic RF network
analysis (`internal/rfscope`). Upload an IQ capture and view its RF scene:
protocol hierarchy, per-channel I/O graph, top talkers, conversations, and
expert-info anomalies, plus a one-click cryptolab `ks` frames download for any
unknown payloads.

## Develop

    make rfscope-web-dev      # Vite dev server on :5272, proxies /api to :8098
    gophertrunk rfscope serve # the API backend (default 127.0.0.1:8098)

## Build

    make rfscope-web-build    # produces web/rfscope/dist/, embedded into the binary

The built bundle is embedded via `embed.go`; `gophertrunk rfscope serve` then
serves it at `/`.
