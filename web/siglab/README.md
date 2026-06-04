# GopherTrunk Signal Lab (web)

Standalone, offline signal-analysis web console. It talks to the `/api/v1/siglab/*`
REST API (see `internal/api/handlers_siglab.go`) and runs entirely in the
browser — no Node.js at runtime. The built bundle is embedded into the
`gophertrunk` binary (`web/siglab/embed.go`) and served by:

- `gophertrunk siglab serve` — a self-contained offline server (default
  `127.0.0.1:8099`), and
- the daemon, which also mounts the same API routes.

## What it does

Parity with the siglab CLI/TUI plus visualization and comparison:

- Upload a raw IQ capture (u8/f32) or **synthesize** an idealized/impaired one
  (the `gen` surface: SNR, carrier offset/drift, multipath, DC, I/Q imbalance).
- Configure and **run** the engine (protocol, sample rate, tune, auto-tune,
  conjugate, IQ-correct, P25 deep knobs) with a **live SSE event stream**.
- **Identify** the protocol of an unknown capture (ranked candidates).
- A rich results dashboard: symbol histogram, constellation, PSD,
  spectrogram/waterfall, eye diagram, sync-landscape heatmap, receiver-state
  series, grants, and event timeline.
- **Compare** multiple analyzed captures — overlay their power spectra and diff
  their metrics — against each other or a synthesized idealized reference.
- **Export** results (JSON/JSONL/YAML/CSV) via the engine's own serializers.

## Libraries

- **Chart.js** + **D3** — histograms, line/series, constellation, eye diagram
  (always in the initial chunk).
- **Plotly.js** — spectrogram/waterfall and sync-landscape heatmaps
  (lazy-loaded chunk).
- **TensorFlow.js** — numpy-like array/FFT math for the PSD and spectrogram
  (lazy-loaded chunk).
- **stdlib** (`@stdlib/stats-base-*`) — SciPy-like summary statistics; with the
  Hann window in `src/dsp/stats.ts`.
- **Observable Plot** — available for alternate statistical plots.

Heavy libraries are code-split via `manualChunks` and dynamic `import()`, so the
initial download stays small while the offline embed still ships everything.

## Develop

```bash
# terminal 1 — backend
gophertrunk siglab serve            # API + (stale) SPA on :8099

# terminal 2 — Vite dev server with HMR, proxying /api to :8099
make siglab-web-dev                 # http://127.0.0.1:5273

# build + embed into the binary
make siglab-web-build               # → web/siglab/dist
make build                          # binary embeds the bundle

# tests / typecheck
make siglab-web-test
cd web/siglab && npm run typecheck
```

Requires Node.js (npm 9+). The bundle fetches no CDN assets and runs fully
offline.
