---
layout: page
title: Web console
description: Setup and quick-start guide for the standalone browser-based GopherTrunk operator console — runs on any laptop, tablet, or phone over a LAN
nav_group: Operate
---

# Web console

GopherTrunk ships a full browser-based operator console alongside the
TUI. Two delivery paths:

- **Embedded.** When the daemon binary was linked against the built
  SPA, the daemon hosts the console at the HTTP API's `/` (no
  separate static server). Run `gophertrunk -web` (or pick `[2] Web`
  from the launcher menu) and your default browser opens against the
  daemon URL automatically.
- **Standalone static bundle.** A pure HTML / CSS / JS folder that
  works without the embedded build. Open `index.html` in any modern
  browser, point it at a running daemon's URL on the connect screen,
  and operate. Useful when you don't want to (re)build the daemon
  binary alongside the SPA.

Every Bubbletea TUI panel has a browser counterpart:

| Panel       | What it does                                                    |
| ----------- | --------------------------------------------------------------- |
| Dashboard   | Live WebSocket event feed, audio cockpit with PCM-over-WAV playback |
| Active      | Active-call list with live elapsed ticker + end-call mutation   |
| History     | Filterable call-log explorer + retention-sweep mutation         |
| Systems     | Trunked-system browser with detail modal                        |
| Talkgroups  | Sortable / filterable list with scan / lockout / priority edits |
| Radio IDs   | Per-RID browser merging the operator alias catalogue with the live affiliation tracker; detail modal with last 50 calls per radio; alias / watch / lockout / priority edits |
| Devices     | SDR pool inspector (live attach / detach)                       |
| Events      | Live ring-buffer viewer (filter / pause / JSON expansion)       |
| Tones       | `tone.alert` feed with per-device reset                         |
| Metrics     | Curated `gophertrunk_*` Prometheus tiles + Chart.js trend       |
| Scanner     | CC hunter, conventional channels, manual VFO tune, scan_mode    |
| Settings    | Theme, write-mode toggle, **live config editing** (PATCH /api/v1/settings) |
| Import      | Upload PDFs / CSV bundles, preview, commit into config.yaml     |

The headline scenario: run the daemon on a Raspberry Pi (or any host
with the RTL-SDR attached), then operate from a laptop, tablet, or
phone anywhere on the LAN.

## Quick path: `gophertrunk -web`

When the daemon binary was built with the SPA embedded (the default
in release builds), the launcher's `-web` flag wires everything up
for you:

```sh
gophertrunk -config config.yaml -web
```

The daemon binds the HTTP API, registers the SPA at `/`, and opens
your default browser at the daemon URL. Same flow when you run
`gophertrunk` with no flags and pick `[2] Web` from the interactive
launcher menu (see [`launcher.md`]({{ '/launcher.html' | relative_url }})).

The daemon also serves the auxiliary consoles alongside the operator
console, each on its own path, so they are reachable from the main
console's nav (**System** group) and cross-linked in each console's
header: the **Config Builder** at `/config/`, **Signal Lab** at
`/siglab/`, **RF Scope** at `/rfscope/`, and — in a `-tags cryptolab`
build — **Crypto Lab** at `/cryptolab/`. Each link only appears once
the daemon actually mounts that console (advertised via
`GET /api/v1/runtime`). The standalone `gophertrunk <console> serve`
commands still serve each console on its own port for offline use.

On a headless host (SSH session, no `$DISPLAY`) the launcher prints
the URL + a hint instead of trying to launch a browser, so a remote
operator can open the URL from their laptop.

If you're building the daemon yourself, use **`make dist`** — it
runs `make web-build` and `make build` in the right order so the
`//go:embed all:dist` snapshot in `web/embed.go` picks up a real
bundle and the daemon serves the SPA at `/`. Plain `make build`
(or a bare `go build`) skips the SPA build, so the embed contains
only the `.gitkeep` sentinel and the daemon answers a helpful 404
at `/` explaining the fix. The REST and WebSocket APIs work
normally in both cases — only the embedded SPA differs. The
launcher's standalone-bundle fallback (below) also still works.

## 1. Get the bundle

The web console ships as a sibling directory inside every release
archive — same archive that contains the `gophertrunk` binary:

```
gophertrunk-<version>-<os>-<arch>/
├── gophertrunk              # the daemon / CLI binary
├── gophertrunk-web/         # standalone web consoles
│   ├── index.html           # standard operator console
│   ├── assets/…             # bundled React / Tailwind / Chart.js
│   ├── favicon.svg
│   ├── manifest.webmanifest
│   ├── sw.js                # PWA service worker
│   ├── siglab/              # Signal Lab (offline signal analysis) console
│   ├── rfscope/             # RF Scope (protocol-agnostic RF analysis) console
│   └── configbuilder/       # Config Builder / editor console
├── config.example.yaml
└── samples/…
```

Download the matching release archive from the
**[Downloads page]({{ '/downloads.html' | relative_url }})** and unpack
it. That's the only file you need on the device that will run the
browser; nothing else is installed.

> **Windows installer:** Setup copies this same tree into the `web\`
> subfolder of your chosen data folder (default
> `%USERPROFILE%\Documents\GopherTrunk\web`), and adds Start Menu
> shortcuts for all four consoles. The paths below that say
> `gophertrunk-web/` map to `…\GopherTrunk\web\` on an installed
> Windows box.

## 2. Configure the daemon for browser access

Two daemon-side knobs control browser access — both already on the
existing `api:` block in `config.yaml`. Edit your `config.yaml`:

```yaml
api:
  # Bind 0.0.0.0 so a laptop on the same network can reach the daemon.
  # Use a specific LAN address like 192.168.1.42 if you prefer.
  http_addr: "0.0.0.0:8080"

  # Browsers send a different Origin than the daemon's bind address,
  # so the daemon needs an explicit allow-list. Choose ONE of:
  cors:
    allowed_origins:
      - "null"                      # SPA opened via file:// (most common)
      # - "http://192.168.1.7:9000" # SPA hosted by a static web server on the laptop
      # - "*"                        # any origin (loopback-only daemons; not for public binds)

  # Require a bearer token when the daemon binds to a non-loopback
  # interface. The SPA prompts for the token on the connect screen.
  auth:
    mode: "required"
    token_file: "/etc/gophertrunk/api-token"
```

Restart the daemon for the changes to take effect:

```bash
./gophertrunk run -config config.yaml
```

> **Why CORS?** The SPA is loaded from a different origin than the
> daemon (either `file://` or whatever static host you use), so the
> browser performs a CORS preflight on every request. `"null"`
> covers the `file://` case; explicit URLs cover the rest.

## 3. Open the SPA

### On the same machine as the daemon

Double-click `gophertrunk-web/index.html` to open it in your default
browser. On the connect screen enter:

- **Server URL:** `http://127.0.0.1:8080`
- **Bearer token:** the contents of your `token_file` (or empty if
  `auth.mode: disabled`)
- **Remember on this device:** check it if you want the token to
  survive a browser restart (it lands in `localStorage` instead of
  `sessionStorage`).

### From a laptop, tablet, or phone on the same LAN

This is the canonical "headless Pi, operate from the couch"
scenario.

1. On the host running the daemon, set `api.http_addr: "0.0.0.0:8080"`
   (or a specific LAN IP) and restart.
2. Copy `gophertrunk-web/` to the device that will run the browser —
   USB stick, `scp`, or just unpack the same release archive on that
   device.
3. Double-click `gophertrunk-web/index.html`. On the connect screen
   enter the daemon's URL (e.g. `http://192.168.1.42:8080`) and
   the bearer token.

The browser remembers the URL across visits.

### Hosted by a static web server (optional)

If you'd rather serve the SPA over HTTP than open `file://`, point
any static-file HTTP server at `gophertrunk-web/`:

```bash
# Quick and dirty — any laptop with Python
cd gophertrunk-web/
python3 -m http.server 9000
```

```bash
# Caddy one-liner
caddy file-server --root gophertrunk-web --listen :9000
```

Then add the server's origin to the daemon's CORS list:

```yaml
api:
  cors:
    allowed_origins:
      - "http://192.168.1.7:9000"
```

Open `http://192.168.1.7:9000/` in any browser on the network and
follow the connect-screen flow as above.

## 4. Install as a phone app (optional)

The SPA is a Progressive Web App. After connecting once:

- **Android (Chrome / Edge):** An "Install GopherTrunk" banner
  appears once the service worker registers — accept it. Or use the
  browser menu → "Install app".
- **iOS (Safari):** Open the SPA in Safari, tap the Share button,
  then "Add to Home Screen". iOS launches the installed app full-
  screen with a dedicated icon.
- **Desktop Chrome / Edge:** Click the install icon in the address
  bar (right side, looks like a small monitor) to install as a
  desktop app.

The installed PWA still talks to whichever daemon URL you set on the
connect screen — installation doesn't change the API target. Audio
playback works on both platforms after a one-time "Tap to enable
audio" gesture (required by iOS / Android autoplay rules).

## Quick reference

### Connect screen field map

| Field                    | What to enter                                                    |
| ------------------------ | ---------------------------------------------------------------- |
| Server URL               | `http://<host>:<port>`, e.g. `http://192.168.1.42:8080`          |
| Bearer token             | Contents of `api.auth.token_file` — leave empty if auth is `disabled` |
| Remember on this device  | Move the token from `sessionStorage` → `localStorage` so it survives browser restart |

### Shareable one-click link

The SPA accepts pre-filled credentials via a URL hash:

```
index.html#server=http://192.168.1.42:8080&token=<your-token>
```

Bookmark or share that link (with appropriate care for the token).
The SPA stores the values in browser storage on first load so the
hash isn't needed again.

### Write mode

Mutation controls (end-call, talkgroup edits, scanner hold / resume
/ retune / manual-tune, retention sweep, tone-detector reset) are
hidden by default. To unlock them:

1. Open **Settings**.
2. Tick **Allow mutations from this browser**.
3. If the daemon's `/api/v1/mutations` endpoint reports
   `allow_mutations: false`, the toggle is informational only — fix
   `api.auth.mode` or the trusted-networks list (see
   **[Hardening]({{ '/hardening.html#api-authentication' | relative_url }})**) to actually permit writes.

The browser remembers the write-mode flag locally. Destructive
mutations (end-call, channel lockout, retune, retention sweep)
always prompt for confirmation before firing.

### Audio playback

The audio cockpit lives at the top of the Dashboard:

- **Volume slider** → `PATCH /api/v1/audio { volume }`
- **Mute toggle** → `PATCH /api/v1/audio { muted }`
- **Record toggle** → `PATCH /api/v1/audio { recording_enabled }`
- **Tap to enable** → required once per session on iOS / Android
  per the autoplay rules; the SPA shows a one-shot prompt.

The browser streams live PCM from
`GET /api/v1/audio/stream` — a continuous open-ended WAV body, no
JavaScript decoding required. Reads are unauthenticated by default;
authenticate them by binding the daemon to a trusted network or
requiring TLS + token via your reverse proxy.

### Tabs

Bottom-nav on phones, top-tab strip on desktop:

```
Dashboard · Active · Scanner · Settings              (always visible)
Systems · Talkgroups · Radio IDs · History ·
  Events · Tones · Metrics · Devices                 (desktop overflow row)
```

On phones the overflow row is reachable via the hamburger menu in
the bottom nav. Keyboard shortcuts are not yet wired (a follow-up
PR will add the TUI's command palette + jump-to-panel hotkeys).

## Troubleshooting

| Symptom                                                | Likely cause + fix                                                                                            |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------- |
| "Failed to fetch" on the connect screen                | Daemon not reachable. Confirm `gophertrunk run` is up and the URL is right (`curl http://<url>/api/v1/health`).        |
| Browser console shows "CORS preflight rejected"        | The daemon hasn't allow-listed your SPA's origin. Add it under `api.cors.allowed_origins` and restart.        |
| 401 on every request                                   | Wrong bearer token. Re-check `api.auth.token_file` contents and re-enter on the connect screen.               |
| Connect succeeds but no events / no audio              | The WebSocket couldn't upgrade. If you're behind a reverse proxy, confirm it forwards `Upgrade` / `Connection` headers. |
| Audio plays for 2 s then stops                         | iOS / Android autoplay block. Tap the "Tap to enable audio" prompt on the Dashboard.                          |
| Mutation buttons are invisible                         | Write mode is off or the daemon rejects mutations. Open **Settings** → tick "Allow mutations".                |
| SPA loads but tabs are stuck on a spinner              | Daemon is reachable but the API rejected the token mid-flight. Open the connect screen and re-enter creds.    |
| PWA install prompt never appears (Android)             | First-load only — clear the site's data in the browser and revisit.                                           |

## Building from source

The web console builds with Vite. The daemon's `Makefile` wraps
the npm scripts:

```bash
make dist           # SPA + daemon — the operator binary that serves / itself
make web-build      # produces web/dist/ — the shipped artifact (consumed by `make dist`)
make web-dev        # Vite dev server on :5173 with proxy to :8080
make web-clean      # removes node_modules/, dist/, dev-dist/
```

Tested with Node.js 20 LTS and npm 10. Older Node versions may work
but aren't in CI. **End users never run Node** — the dev server is
a developer convenience only.

The Vite dev server proxies `/api/*` and `/metrics` to
`http://127.0.0.1:8080`, so the SPA running on `:5173` looks
same-origin to the browser and CORS isn't required during
development.

## See also

- [`web/README.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/web/README.md) — full developer reference + architecture
- [Hardening]({{ '/hardening.html#api-authentication' | relative_url }}) — `api.auth.mode`, `token_file`, trusted networks
- [TUI]({{ '/tui.html' | relative_url }}) — full-screen terminal cockpit alternative
