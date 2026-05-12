# GopherTrunk TUI

A full-screen operator console over the daemon's REST + SSE API.
The TUI ships in the same binary as the daemon — no separate
install — and points at a running daemon over HTTP. Eleven panels
cover every read surface; the Scanner cockpit, talkgroup table,
and a curated set of mutations are interactive when the daemon is
started with `api.allow_mutations: true` and the TUI with
`--write`.

## Quick start

```bash
# In one terminal: run the daemon as usual.
gophertrunk run -config config.yaml

# In another: open the TUI.
gophertrunk tui
```

By default the TUI connects to `http://127.0.0.1:8080`. Override
with `-server`:

```bash
gophertrunk tui -server http://10.0.0.5:8080
gophertrunk tui -server https://radio.example.com -insecure
```

## Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `-server URL` | `http://127.0.0.1:8080` | daemon base URL |
| `-insecure` | `false` | skip TLS certificate verification (for self-signed test daemons) |
| `-timeout DURATION` | `5s` | per-request timeout; SSE streams are unaffected |
| `-no-color` | `false` | strip ANSI colour, useful when piping or on a monochrome terminal |
| `-write` | `false` | surface mutation keybindings; daemon must also have `api.allow_mutations: true` |

## Panels

| # | Panel | What it shows |
| --- | --- | --- |
| 1 | Dashboard | At-a-glance health, active call count, recent events, recent tone alerts. Default landing panel. |
| 2 | Systems | Configured trunking systems: name, protocol, control channels, IDs (WACN / SystemID / RFSS-Site). |
| 3 | Talkgroups | Full talkgroup table from the daemon. Substring filter (`/`) and a sort cycle (`s`: ID → Alpha → Priority). |
| 4 | Active calls | Calls currently being followed. Driven by a 1 s poll plus live `call.start` / `call.end` events for instant updates. |
| 5 | Call history | Ended calls from `/api/v1/calls/history`. On-demand reload (`r`); the panel does not poll continuously. Row formatting runs off the Update goroutine so the reducer never blocks on history rebuilds. |
| 6 | Events | The full SSE feed in chronological order. Substring filter (`/`), pause auto-scroll (`p`), clear filter (`c`). 500-entry ring buffer. |
| 7 | Tone alerts | Just `tone.alert` events with profile / device / matched frequencies. 100-entry ring buffer. |
| 8 | Metrics | Curated subset of `/metrics`: calls active / total, grant total, CC locked, SSE clients, devices attached, tone alerts. |
| 9 | Devices | The SDR pool snapshot: serial, driver, tuner, role, configured gain / PPM / bias-tee, attach state. Refreshed on `sdr.attached` / `sdr.detached` events for instant updates. |
| 0 | Scanner | Police-scanner cockpit: per-trunked-system CC hunter state, conventional FM scan list (current dwell highlighted), talkgroup scan-list summary, audio cockpit. |
| — | Settings | Tabbed read-only inspector of the live daemon configuration (Daemon · Storage · Audio · Recording · Tones · API · Vocoders · SDR · FEC). Reach via `Tab` / `Shift+Tab` or the command palette — there's no number shortcut. |

## Keybindings

### Global

| Key | Action |
| --- | --- |
| `Tab` | next panel |
| `Shift+Tab` | previous panel |
| `1`–`9`, `0` | jump directly to a panel (0 = Scanner; Settings has no number) |
| `Ctrl+P` | open the fuzzy command palette (see below) |
| `Ctrl+T` | toggle theme (dark ↔ monochrome) |
| `?` | toggle help overlay |
| `q` / `Ctrl+C` | quit |

### Inside tables

| Key | Action |
| --- | --- |
| `j` / `↓` | next row |
| `k` / `↑` | previous row |
| `g` | top |
| `G` | bottom |
| `Page Down` / `Page Up` | scroll a page |

### Panel-local

| Panel | Keys |
| --- | --- |
| Systems | `Enter` open detail card |
| Talkgroups | `/` filter, `s` cycle sort, `l` toggle lockout, `S` toggle scan flag, `+` / `-` priority up/down, `Enter` open detail card, `Esc` exit filter input |
| Active calls | `e` end highlighted call (write) |
| Call history | `r` reload |
| Events | `/` filter, `p` pause auto-scroll, `c` clear filter |
| Tone alerts | `R` reset detector for highlighted device (write) |
| Metrics | `S` run retention sweep now (write) |
| Devices | (table navigation only) |
| Scanner | `j` / `k` move row, `h` hold/resume highlighted row, `r` force re-hunt (Systems section, confirms), `Enter` dwell on highlighted conv channel, `L` lockout / unlockout highlighted conv channel (skipped from scan rotation, runtime-only), `m` cycle scan_mode, `+` / `-` volume up/down (5% step), `M` mute toggle, `R` recording toggle, `f` manual tune (type frequency in MHz, Enter to listen, Esc to cancel) |
| Settings | `[` / `]` / `h` / `l` / `←` / `→` cycle through inspector tabs |

## Mouse

The tab strip is clickable, and every table-backed panel
(Systems · Talkgroups · Active · History · Devices · Tones ·
Metrics) reacts to body clicks and scroll-wheel ticks:

- **Left-click on a tab** switches to that panel.
- **Left-click on a data row** moves the cursor onto that row.
  Follow up with `Enter` to open the detail card, or with a
  mutation key (`e` to end the highlighted call, `R` to reset the
  highlighted tone detector, `L` to toggle conventional channel
  lockout, etc.).
- **Scroll-wheel up/down** advances the cursor one row at a time
  in the same panels. Bubbles v1.0.0's `table.Update` is
  `KeyMsg`-only, so wheel forwarding is plumbed by the TUI itself
  via a shared `MouseAware` interface.

Chrome clicks (border / title / column header) are ignored;
out-of-range clicks clamp to the last row.

## Command palette

`Ctrl+P` opens a centered fuzzy-matched action list. Type to filter;
`↑` / `↓` or `Ctrl+J` / `Ctrl+K` to move; `Enter` to run; `Esc` or
another `Ctrl+P` to dismiss. The list is rebuilt every time you
open the palette so newly-arrived systems / talkgroups / devices
are reachable without a restart.

Discovered actions:

- **Panel jumps** for every panel (including Settings, which has no
  number shortcut).
- **System / talkgroup / device drill-ins** — pre-positions the
  destination panel's cursor on the matching row before opening the
  detail modal, so keyboard and mouse paths converge on the same
  selection. Talkgroup actions cap at the first 200 to keep the
  list legible on large rosters.
- **Audio mutations** — volume ±5%, mute toggle, recording toggle.
- **Retention sweep** (confirms).
- **Scanner** — toggle scan_mode list ↔ all, per-system hold/resume.
- **Theme toggle**, **help overlay**, **quit**.

## Theme

`Ctrl+T` cycles between the dark and monochrome palettes at runtime.
`-no-color` starts the TUI on monochrome and pins the lipgloss
renderer to a nil writer so ANSI is stripped even when piping
output — `Ctrl+T` flips the palette in that mode but the renderer
won't re-emit colour.

Panels that own a cached `bubbles/table` style bundle handle a
`ThemeChangedMsg` broadcast by re-applying `tableStyles()` so the
swap takes effect on the next render without a restart.

## Settings panel

Read-only inspector of the live daemon configuration, fetched once
at startup and refreshed every 30 s from `GET /api/v1/runtime`.
Cycle through tabs with `[` / `]` (or `h` / `l` / `←` / `→`):

| Tab | Coverage |
| --- | --- |
| Daemon | version, log level / format, metrics enabled |
| Storage | call-log DB path, CC cache file, retention windows (days) + sweeper interval |
| Audio | enabled, device, sample rate, buffer ms, available backends, auto-fallback disable flag (Linux) |
| Recording | output dir, sample rate, raw vocoder frames, CMA equalizer (taps + step size) |
| Tones | per-profile name, cooldown, frequencies |
| API | HTTP / gRPC / WebSocket / metrics listener addresses, mutations allowed |
| Vocoders | per-protocol vocoder mapping |
| SDR | registered backend names, per-device serial / role / gain |
| FEC | per-system FEC state — every protocol's chain is on by default, this view surfaces explicit `<key>: off` opt-outs and the current channel-coding / colour-code parameters |

Every config knob the daemon reads has a touch-point here. Mutating
the settings still requires editing `config.yaml` and restarting
the daemon.

## Polling cadences

The TUI keeps SharedState fresh with a fan of polling Cmds:

| Endpoint | Period |
| --- | --- |
| `/api/v1/calls/active` | 1 s |
| `/api/v1/health` | 2 s |
| `/api/v1/scanner` | 2 s |
| `/api/v1/audio` | 3 s |
| `/metrics` | 5 s |
| `/api/v1/systems` | 10 s |
| `/api/v1/devices` | 10 s (also refreshed on `sdr.attached` / `sdr.detached`) |
| `/api/v1/talkgroups` | 30 s |
| `/api/v1/runtime` | 30 s |
| `/api/v1/calls/history` | on-demand (`r` in History panel) |
| `/api/v1/systems/{name}` | on-demand (Systems panel `Enter`) |
| `/api/v1/talkgroups/{id}` | on-demand (Talkgroups panel `Enter`) |

A long-lived `/api/v1/events` SSE stream complements the polls, so
`call.start` / `call.end` / `cc.locked` / `tone.alert` /
`audio.state` arrive instantly. If the SSE stream drops, the TUI
shows a transient toast and reconnects with exponential backoff
(1 s → 30 s cap).

## Mutations

The TUI drives a curated set of operator actions when both sides
opt in:

```bash
# Daemon side: enable in config.yaml.
api:
  http_addr: 127.0.0.1:8080   # bind to loopback if you can
  allow_mutations: true

# TUI side: pass --write to surface the keybindings.
gophertrunk tui --write
```

Both ends are gated separately on purpose. The daemon's HTTP API
has no authentication today, so any host that can reach the
listener can call mutations once the gate is open — bind to
`127.0.0.1` and trust the operator before flipping it on.

| Panel | Key | Action | Confirm? |
| --- | --- | --- | --- |
| Active calls | `e` | End the highlighted call (`POST /api/v1/calls/{serial}/end`, reason=manual) | yes |
| Talkgroups | `l` | Toggle lockout on the highlighted talkgroup (`PATCH /api/v1/talkgroups/{id}`) | no — reversible |
| Talkgroups | `S` | Toggle scan flag (relevant when `scan_mode: list`) | no |
| Talkgroups | `+` / `-` | Bump priority up / down (clamped 0–99) | no |
| Tone alerts | `R` | Reset tone-out match progress on the highlighted device (`POST /api/v1/devices/{serial}/tone-reset`) | yes |
| Metrics | `S` | Run a retention sweep now (`POST /api/v1/retention/sweep`) | yes |
| Scanner | `h` | Hold/resume highlighted system or conv channel (`POST /api/v1/scanner/hunt/{system}/{hold,resume}` or `/conventional/{hold,resume}`) | no |
| Scanner | `r` | Force re-hunt the highlighted system (`POST /api/v1/scanner/hunt/{system}/retune`) | yes |
| Scanner | `Enter` | Dwell on the highlighted conv channel (`POST /api/v1/scanner/conventional/{index}/dwell`) | no |
| Scanner | `L` | Lockout / unlockout the highlighted conv channel (`POST /api/v1/scanner/conventional/{index}/{lockout,unlockout}`) | no — reversible |
| Scanner | `m` | Cycle global scan_mode list ↔ all (`PATCH /api/v1/scanner`) | no |
| Scanner | `f` | Manual tune: type a frequency in MHz and Enter to listen (`POST /api/v1/scanner/manual_tune`) | no |
| Scanner | `+` / `-` / `M` / `R` | Volume ± 5%, mute toggle, recording toggle (`PATCH /api/v1/audio`) | no |

When a key requires confirmation, a centered modal opens. The
modal captures keyboard focus until you press:

- `y` or `Enter` — fire the action and close the modal
- `n` or `Esc` — cancel and close the modal

On success the status bar shows `<label> ok` and the dependent
read panels refresh immediately rather than waiting for the next
poll tick. Audio mutations additionally publish an `audio.state`
SSE event so a second TUI / `curl` PATCH converges in one round-trip
instead of waiting for the next 3 s poll. On failure the status bar
shows the HTTP error.

The TUI fetches `GET /api/v1/mutations` once at startup to learn
which subsystems the daemon has wired (engine / retention / tone
detector / scanner / audio). If the daemon doesn't recognise the
route (older build), the TUI assumes mutations are off and
`--write` is a no-op.

### Endpoints reference

| Method | Path | Body | Notes |
| --- | --- | --- | --- |
| `GET` | `/api/v1/mutations` | — | Capability probe. Always 200. |
| `GET` | `/api/v1/runtime` | — | Sanitized snapshot of `config.Config`. Powers the Settings panel. |
| `GET` | `/api/v1/audio` | — | Current backend / volume / mute / recording state. |
| `PATCH` | `/api/v1/audio` | `{"volume":0.7}` / `{"muted":true}` / `{"recording_enabled":true}` | Any combination of the three knobs. |
| `GET` | `/api/v1/scanner` | — | Unified scanner snapshot — systems, conventional, scan_mode, scan-list counts. |
| `PATCH` | `/api/v1/scanner` | `{"scan_mode":"list"}` | Cycle scan_mode at runtime. |
| `POST` | `/api/v1/scanner/hunt/{system}/{hold,resume,retune}` | — | Per-system CC-hunt control. |
| `POST` | `/api/v1/scanner/conventional/{hold,resume}` | — | Conventional scanner control. |
| `POST` | `/api/v1/scanner/conventional/{index}/{dwell,lockout,unlockout}` | — | Per-channel control. |
| `POST` | `/api/v1/scanner/manual_tune` | `{"frequency_hz":155895000,"label":"...","mode":"fm"}` | Adds a runtime VFO channel and dwells. |
| `POST` | `/api/v1/calls/{deviceSerial}/end` | `{"reason":"manual"}` | 404 if no active call on that device. |
| `PATCH` | `/api/v1/talkgroups/{id}` | `{"priority":3,"lockout":true,"scan":false}` | All fields optional; supply at least one. Returns the updated TalkgroupDTO. |
| `POST` | `/api/v1/retention/sweep` | — | Synchronous; 503 if no call-log persistence configured. |
| `POST` | `/api/v1/devices/{serial}/tone-reset` | — | 503 if the tone-out detector isn't wired. |

Every mutation endpoint returns `403 Forbidden` with
`{"error":"mutations disabled (set api.allow_mutations: true to enable)"}`
when the daemon was started without the gate.

## Troubleshooting

- **"daemon unreachable"**: check the daemon is running and the
  `-server` URL is reachable. If the daemon binds to a non-default
  address or port, pass `-server` explicitly.
- **"event stream disconnected"**: the TUI auto-reconnects. If the
  toast keeps reappearing, check the daemon logs and the connection
  in front of it (corporate proxies sometimes truncate SSE).
- **Garbled colours**: pass `-no-color`, or set `TERM` to a value
  your terminal supports (e.g. `xterm-256color`). `Ctrl+T` also
  cycles to the monochrome palette at runtime.

## Implementation notes

- Library: `charmbracelet/bubbletea` (Elm-style model/update/view) +
  `bubbles` (table, textinput, viewport, help) + `lipgloss`
  (styling).
- DTOs are mirrored from `internal/api/types.go` into
  `internal/tui/client/types.go` rather than imported, keeping the
  TUI a pure HTTP client of the wire protocol.
- The SSE pump uses bubbletea's canonical channel pattern: a
  goroutine reads the stream, writes typed `Event`s into a buffered
  channel, and a `listenSSE` Cmd blocks on `<-ch` and re-arms itself
  in the model's Update.
- All colour decisions flow through `internal/tui/theme`. Panels
  read `theme.Theme()` at render time; the cached lipgloss styles
  on `*Model` and inside each `bubbles/table` re-pick up the
  palette via `panels.ThemeChangedMsg`.
- Optional panel interfaces:
  - `panels.Revealer` — pre-position a panel's cursor on a key
    (system name, talkgroup ID, device serial, scanner row).
    Driven from the command palette.
  - `panels.MouseAware` — left-press + scroll-wheel handling on
    body rows. Driven from the root model's mouse hit-test.
  - Both are opt-in; panels that don't implement them are skipped
    silently by the dispatcher.
