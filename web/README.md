# GopherTrunk Web

A pure-browser operator console for the GopherTrunk daemon. Runs entirely
client-side: React + Tailwind CSS + Chart.js, bundled into a static
folder. No Node.js at runtime. Talks to the daemon's HTTP/SSE/WebSocket
API directly, with all settings stored in browser storage.

## What ships

A release archive (produced by `make release-archives`) contains:

```
gophertrunk-v…/
├── gophertrunk           # the daemon binary
├── gophertrunk-web/      # this SPA, pre-built (open index.html in a browser)
│   ├── index.html
│   ├── assets/…
│   ├── favicon.svg
│   └── manifest.webmanifest
└── …
```

The `gophertrunk-web/` directory is self-contained — every dependency
(React, React Router, Tailwind CSS, Zustand, Chart.js, D3 scale,
Workbox runtime, …) is bundled into the JS/CSS files. There are no
CDN fetches; everything works on an offline LAN.

## Quick start (operators)

### Daemon on the same machine

1. Run the daemon: `gophertrunk daemon -config config.yaml`.
2. Make sure the daemon allows your browser:

   ```yaml
   api:
     http_addr: "127.0.0.1:8080"
     cors:
       allowed_origins:
         - "null"           # required when opening index.html via file://
   ```

3. Open `gophertrunk-web/index.html` in any browser. On the
   connect screen enter `http://127.0.0.1:8080` and (if you set
   one) your bearer token.

### Daemon on a Raspberry Pi, web UI on your laptop

This is the headline scenario: keep the SDR-attached host
running quietly in a closet; operate from anywhere on the LAN.

1. On the Pi, set `api.host: 0.0.0.0` (or pick a specific LAN
   address) and add a CORS origin for the laptop:

   ```yaml
   api:
     http_addr: "0.0.0.0:8080"
     auth:
       mode: "required"
       token_file: "/etc/gophertrunk/api-token"
     cors:
       allowed_origins:
         - "null"
   ```

2. Start the daemon: `gophertrunk daemon -config config.yaml`.
3. Copy `gophertrunk-web/` to the laptop (USB stick, `scp`, or
   unpack the same release archive locally).
4. Double-click `gophertrunk-web/index.html`. On the connect
   screen enter the Pi's URL (e.g. `http://192.168.1.42:8080`)
   and the token.

The browser remembers the URL; the token sits in `sessionStorage`
unless you check "Remember on this device" (then it moves to
`localStorage`).

### Install on a phone

The SPA is a Progressive Web App. After connecting once:

- **Android (Chrome/Edge):** an "Install GopherTrunk" banner
  appears once the service worker registers — accept it. Or use
  the browser menu → "Install app".
- **iOS (Safari):** open the SPA in Safari, tap the Share button,
  then "Add to Home Screen". iOS will launch the app full-screen
  with a dedicated icon.

The installed PWA still talks to whichever daemon URL you set on
the connect screen. Audio playback works on both platforms after
a one-time "Tap to enable audio" gesture (required by iOS/Android
autoplay rules).

## Quick start (developers)

```sh
# from the repository root:
make web-dev    # starts Vite at http://127.0.0.1:5173 with proxy to :8080
make web-build  # produces web/dist/ — the shipping artifact
make web-clean  # removes node_modules/, dist/, and SW dev-dist/

# From web/ directly:
npm install     # one-time
npm test        # Vitest + React Testing Library against panels
npm run typecheck
```

Tested with Node.js 20 LTS and npm 10. Older Node versions may
work but aren't in CI.

The dev server proxies `/api/*` and `/metrics` to
`127.0.0.1:8080`, so the SPA running on `:5173` looks
same-origin to the browser and you don't need CORS during
development.

### Tests

`npm test` runs Vitest in CI mode against `src/**/*.{test,spec}.{ts,tsx}`.
Tests use jsdom for the DOM and React Testing Library for component
interaction; `vi.mock` replaces the `api/client` and `api/write`
modules so no network is involved. New panel tests should follow
the `Import.test.tsx` / `Settings.test.tsx` pattern: mock the API,
reset the zustand store between tests, and drive the component
through `userEvent`.

## Status

TUI parity — every Bubbletea TUI panel has a browser counterpart.

| Panel         | Reads                                           | Mutations (write-mode gated)              |
| ------------- | ----------------------------------------------- | ----------------------------------------- |
| ConnectScreen | `GET /api/v1/health` reachability probe         | —                                         |
| Dashboard     | `/api/v1/{health,calls/active,devices,audio}` + WebSocket + `/api/v1/audio/stream` | `PATCH /api/v1/audio` (volume/mute/record) |
| Active        | `GET /api/v1/calls/active` (live elapsed ticker) | `POST /api/v1/calls/{serial}/end`        |
| History       | `GET /api/v1/calls/history?limit&system&group_id` | `POST /api/v1/retention/sweep`          |
| Systems       | `GET /api/v1/systems`                           | —                                         |
| Talkgroups    | `GET /api/v1/talkgroups`                        | `PATCH /api/v1/talkgroups/{id}` (scan/lockout/priority) |
| Devices       | `GET /api/v1/devices` + `sdr.attached/detached` | —                                         |
| Events        | WebSocket ring buffer                           | —                                         |
| Tones         | `tone.alert` events                             | `POST /api/v1/devices/{serial}/tone-reset` |
| Metrics       | `GET /metrics` (curated tiles + Chart.js trend) | —                                         |
| Scanner       | `GET /api/v1/scanner` (CC hunter + conv + manual tune) | `PATCH /api/v1/scanner`, `POST /api/v1/scanner/{hunt,conventional,manual_tune}/*`, `DELETE …` |
| Settings      | —                                               | theme, write-mode, forget-device          |

All mutations are AND-gated by `selectCanMutate` (the Settings
write-mode toggle ∧ the daemon's `/api/v1/mutations.allow_mutations`
flag). Destructive ones — end-call, channel lockout, retune,
retention sweep — are wrapped in a `ConfirmModal` so an accidental
tap in the bottom nav can't fire one.

## Architecture

- **React 18** + **React Router** (hash mode so `file://` works)
- **Zustand** for shared state — see `src/store/shared.ts`
- **Tailwind CSS** for layout + a tiny `tokens.css` for the
  dark / monochrome themes
- **Chart.js** + **react-chartjs-2** for metrics + scanner
  visualisations; **D3 scale** subpackage for custom axes
- **vite-plugin-pwa** wraps Workbox; the service worker
  pre-caches every shipped asset but never `/api/*` or `/metrics`
- **Web Audio + HTMLAudioElement** for live PCM playback via
  `GET /api/v1/audio/stream` (a continuous WAV body emitted by
  the daemon)
- **Media Session API** for lock-screen metadata while a call is
  active

The daemon-side HTTP surface is unchanged except for two
additions:

1. `GET /api/v1/audio/stream` — open-ended WAV body, plays via
   `<audio src="…">`.
2. `api.cors.allowed_origins` — see the daemon config example.

## Browser support

| Browser            | SPA loads | Audio plays | PWA install |
| ------------------ | --------- | ----------- | ----------- |
| Chrome / Edge ≥ 110 | ✅        | ✅          | ✅          |
| Firefox ≥ 110       | ✅        | ✅          | ⚠ desktop only |
| Safari ≥ 16.4       | ✅        | ✅          | ✅ via Share menu |
| iOS Safari ≥ 16.4   | ✅        | ✅          | ✅ via Share menu |
| Android Chrome ≥ 110 | ✅        | ✅          | ✅          |

Network connectivity is required for the daemon API; the SPA
itself loads from your cache after the first visit (service
worker).

## Privacy

The SPA never phones home. Every HTTP request goes directly to
the daemon URL you entered on the connect screen. The bearer
token lives in `sessionStorage` (default) or `localStorage`
(when "Remember on this device" is checked). Settings →
"Forget this device" clears everything.
