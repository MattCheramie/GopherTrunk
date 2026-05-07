# GopherTrunk TUI

A read-only, full-screen operator view over the daemon's REST + SSE
API. The TUI ships in the same binary as the daemon — no separate
install — and points at a running daemon over HTTP.

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

## Panels

| # | Panel | What it shows |
| --- | --- | --- |
| 1 | Dashboard | At-a-glance health, active call count, recent events, recent tone alerts. Default landing panel. |
| 2 | Systems | Configured trunking systems: name, protocol, control channels, IDs (WACN / SystemID / RFSS-Site). |
| 3 | Talkgroups | Full talkgroup table from the daemon. Substring filter (`/`) and a sort cycle (`s`: ID → Alpha → Priority). |
| 4 | Active calls | Calls currently being followed. Driven by a 1 s poll plus live `call.start` / `call.end` events for instant updates. |
| 5 | Call history | Ended calls from `/api/v1/calls/history`. On-demand reload (`r`); the panel does not poll continuously. |
| 6 | Events | The full SSE feed in chronological order. Substring filter (`/`), pause auto-scroll (`p`), clear filter (`c`). 500-entry ring buffer. |
| 7 | Tone alerts | Just `tone.alert` events with profile / device / matched frequencies. 100-entry ring buffer. |
| 8 | Metrics | Curated subset of `/metrics`: calls active / total, grant total, CC locked, SSE clients, devices attached, tone alerts. |

## Keybindings

### Global

| Key | Action |
| --- | --- |
| `Tab` | next panel |
| `Shift+Tab` | previous panel |
| `1`–`8` | jump directly to a panel |
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
| Talkgroups | `/` filter, `s` cycle sort, `Esc` exit filter input |
| Active calls | (table navigation only) |
| Call history | `r` reload |
| Events | `/` filter, `p` pause auto-scroll, `c` clear filter |
| Tone alerts | (table navigation only) |
| Metrics | (table navigation only) |

## Polling cadences

The TUI keeps SharedState fresh with a fan of polling Cmds:

| Endpoint | Period |
| --- | --- |
| `/api/v1/calls/active` | 1 s |
| `/api/v1/health` | 2 s |
| `/metrics` | 5 s |
| `/api/v1/systems` | 10 s |
| `/api/v1/talkgroups` | 30 s |
| `/api/v1/calls/history` | on-demand |

A long-lived `/api/v1/events` SSE stream complements the polls, so
`call.start` / `call.end` / `cc.locked` / `tone.alert` arrive
instantly. If the SSE stream drops, the TUI shows a transient toast
and reconnects with exponential backoff (1 s → 30 s cap).

## Mutations

The TUI can drive five operator actions when both sides opt in:

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
| Talkgroups | `+` / `-` | Bump priority up / down (clamped 0–99) | no |
| Tone alerts | `R` | Reset tone-out match progress on the highlighted device (`POST /api/v1/devices/{serial}/tone-reset`) | yes |
| Metrics | `S` | Run a retention sweep now (`POST /api/v1/retention/sweep`) | yes |

When a key requires confirmation, a centered modal opens. The
modal captures keyboard focus until you press:

- `y` or `Enter` — fire the action and close the modal
- `n` or `Esc` — cancel and close the modal

On success the status bar shows `<label> ok` and the dependent
read panels (active calls + talkgroups) refresh immediately rather
than waiting for the next poll tick. On failure the status bar
shows the HTTP error.

The TUI fetches `GET /api/v1/mutations` once at startup to learn
which subsystems the daemon has wired (engine / retention / tone
detector). If the daemon doesn't recognise the route (older build),
the TUI assumes mutations are off and `--write` is a no-op.

### Endpoints reference

| Method | Path | Body | Notes |
| --- | --- | --- | --- |
| `GET` | `/api/v1/mutations` | — | Capability probe. Always 200. |
| `POST` | `/api/v1/calls/{deviceSerial}/end` | `{"reason":"manual"}` | 404 if no active call on that device. |
| `PATCH` | `/api/v1/talkgroups/{id}` | `{"priority":3,"lockout":true}` | Both fields optional; supply at least one. Returns the updated TalkgroupDTO. |
| `POST` | `/api/v1/retention/sweep` | — | Synchronous; 503 if no call-log persistence configured. |
| `POST` | `/api/v1/devices/{serial}/tone-reset` | — | 503 if the tone-out detector isn't wired. |

Every mutation endpoint returns `403 Forbidden` with `{"error":"mutations disabled (set api.allow_mutations: true to enable)"}` when the daemon was started without the gate.

## Troubleshooting

- **"daemon unreachable"**: check the daemon is running and the
  `-server` URL is reachable. If the daemon binds to a non-default
  address or port, pass `-server` explicitly.
- **"event stream disconnected"**: the TUI auto-reconnects. If the
  toast keeps reappearing, check the daemon logs and the connection
  in front of it (corporate proxies sometimes truncate SSE).
- **Garbled colours**: pass `-no-color`, or set `TERM` to a value
  your terminal supports (e.g. `xterm-256color`).

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
