# GopherTrunk Config Builder (web/configbuilder)

A standalone, browser-based **Config Builder/Editor** for GopherTrunk's
`config.yaml`. It is the GUI counterpart to hand-editing the file against
`internal/config/config.go`:

- Browse and load existing config files (the standard discovery
  directories, or a `-config-dir` you pass).
- Build a new config section-by-section, each section with inline
  instructions, a live validator, and a **Docs ↗** link.
- Browse/search **RadioReference.com** and import a system (control
  channels + talkgroups) straight into your draft.
- Import from a RadioReference **PDF** or **CSV** export.
- Live `config.yaml` preview; save to disk (with overwrite + external-edit
  guards) and per-system talkgroup CSV sidecars.

## Running

Standalone (no daemon, no SDR required):

```sh
make configbuilder-web-build          # build the SPA into dist/
go run ./cmd/gophertrunk config serve -open
# → http://127.0.0.1:8077/
```

The daemon also serves it at `/config/`, linked from the main web UI's nav
("🛠 Config Builder"), so you can edit config in a new tab without tearing
down the live operator session.

RadioReference browse/import uses `GOPHERTRUNK_RR_KEY` / `GOPHERTRUNK_RR_USER`
/ `GOPHERTRUNK_RR_PASS` (or, on the daemon, the `radioreference` config
block).

## Development

```sh
make configbuilder-web-dev   # Vite dev server, proxies /api → 127.0.0.1:8077
make configbuilder-web-test  # vitest
```

The SPA talks to the `/api/v1/config/*` routes (see
`internal/api/handlers_config_builder.go`). The config is exchanged as JSON
using Go field names (the config structs carry yaml, not json, tags), so the
TypeScript types in `src/api/types.ts` use those exact names.
