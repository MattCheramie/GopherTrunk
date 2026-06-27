# Crypto Lab web console

The standalone web console for the optional GopherTrunk `cryptolab` research
toolkit. Pick a tool/mode, upload inputs, set options, run, and view the
structured result — the browser counterpart to `gophertrunk cryptolab`.

Stack: React + TypeScript + Vite + Tailwind + Zustand (the same stack and
shared design tokens as the `siglab` and `configbuilder` consoles). The form
is schema-driven: it renders whatever tools/modes/parameters the backend
reports at `GET /api/v1/cryptolab/tools`, so new tools appear automatically.

## Build

```
make cryptolab-web-build           # install + bundle into dist/ (embedded in the binary)
make build TAGS=cryptolab          # build gophertrunk with the toolkit + console linked in
gophertrunk cryptolab serve -open  # launch the console in your browser
```

`dist/` is git-ignored (only `.gitkeep` is tracked); the Go embed picks up a
real build automatically and otherwise reports "SPA not bundled".

## Develop

```
make cryptolab-web-dev   # Vite dev server on :5275, proxying /api to cryptolab serve (:8096)
make cryptolab-web-test  # vitest
```

Requires Node ≥ 18 / npm. The console is offline-first (PWA service worker),
so the single binary serves it with no network access.
