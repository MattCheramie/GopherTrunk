---
layout: page
title: Windows user guide
description: Comprehensive walkthrough of every GopherTrunk setting, feature, and command on Windows 11 — from install through daily operation
nav_group: Operate
---

# GopherTrunk on Windows — User Guide

This guide takes a Windows 11 operator from a fresh download to a
fully configured, daily-driven scanner. It consolidates the install
walkthrough, every CLI subcommand, every `config.yaml` knob, the TUI
and web console, importing systems from RadioReference, hardening,
voice-decoder calibration, and troubleshooting into one
Windows-focused reference.

Where this document distills information from elsewhere in the docs,
the source page is linked inline. Operators on Linux or macOS should
prefer the platform-specific install guides
([Linux]({{ '/install-linux.html' | relative_url }}),
[macOS]({{ '/install-macos.html' | relative_url }})) plus the
shared [TUI]({{ '/tui.html' | relative_url }}) /
[web console]({{ '/web.html' | relative_url }}) /
[hardening]({{ '/hardening.html' | relative_url }}) pages.

## Contents

1. [What you need before you start](#1-what-you-need-before-you-start)
2. [Install GopherTrunk](#2-install-gophertrunk)
3. [Bind the RTL-SDR to WinUSB with Zadig](#3-bind-the-rtl-sdr-to-winusb-with-zadig)
4. [Verify the install](#4-verify-the-install)
5. [Build your first config](#5-build-your-first-config)
6. [Run the daemon](#6-run-the-daemon)
7. [Configure as a Windows service](#7-configure-as-a-windows-service)
8. [Operate from the TUI](#8-operate-from-the-tui)
9. [Operate from the web console](#9-operate-from-the-web-console)
10. [Import trunked systems from RadioReference](#10-import-trunked-systems-from-radioreference)
11. [Configuration reference (`config.yaml`)](#11-configuration-reference-configyaml)
12. [Scanner subsystems](#12-scanner-subsystems)
13. [Audio playback and recordings](#13-audio-playback-and-recordings)
14. [Tone-out paging-tone alerts](#14-tone-out-paging-tone-alerts)
15. [API authentication and hardening](#15-api-authentication-and-hardening)
16. [Metrics and health endpoints](#16-metrics-and-health-endpoints)
17. [CLI reference](#17-cli-reference)
18. [Vocoders and voice calibration](#18-vocoders-and-voice-calibration)
19. [Upgrading and uninstalling](#19-upgrading-and-uninstalling)
20. [Troubleshooting](#20-troubleshooting)

---

## 1. What you need before you start

- **Windows 11 x64** (Windows 10 x64 22H2 also works, untested below
  that). The portable ZIP also has an ARM64 variant for Surface /
  Snapdragon X laptops.
- **An RTL-SDR dongle.** Any RTL2832U-based dongle with an R820T,
  R820T2, R828D, E4000, FC0012/13, or FC2580 tuner. The reference
  units are the **NooElec NESDR Smart v5** and **RTL-SDR Blog v3 /
  v4**. See [`hardware.md`]({{ '/hardware.html' | relative_url }})
  for the full matrix.
- **Administrator access** for the install wizard and the one-time
  Zadig driver swap.
- **Windows Terminal or PowerShell.** Every shell command below is
  PowerShell.
- **A modern browser** (Edge, Chrome, Firefox, or any Chromium fork)
  if you plan to use the web operator console.

## 2. Install GopherTrunk

The full walkthrough lives at
[`install-windows.md`]({{ '/install-windows.html' | relative_url }}).
The condensed version:

1. **Download** `gophertrunk-<version>-windows-amd64-setup.exe` from
   the [releases page](https://github.com/MattCheramie/GopherTrunk/releases).
   The matching `…-windows-amd64.zip` is identical contents if you'd
   rather skip the wizard. ARM64 portable ZIPs are also published.

2. **Verify the SHA-256 checksum** before running the installer (see
   [`downloads.html#verify-your-download`]({{ '/downloads.html#verify-your-download' | relative_url }})):

   ```powershell
   $expected = (Get-Content SHA256SUMS | Select-String "windows-amd64-setup.exe").ToString().Split(" ")[0]
   $actual = (Get-FileHash gophertrunk-<version>-windows-amd64-setup.exe -Algorithm SHA256).Hash.ToLower()
   if ($actual -ne $expected) { throw "checksum mismatch" }
   ```

3. **Run the installer.** Builds are unsigned today — if SmartScreen
   blocks the download or run, click **More info → Run anyway**, or
   right-click the file → Properties → check **Unblock**. The wizard
   does the following:

   - Copies `gophertrunk.exe` to `C:\Program Files\GopherTrunk\`
     (single static binary, no DLLs).
   - Adds Start Menu entries for the daemon, the config template,
     and the install walkthrough.
   - Offers to install the **browser-based web operator console** to
     a folder of your choice (default
     `%USERPROFILE%\Documents\GopherTrunk Web Console`). Untick the
     "Install the web operator console" task on the Tasks page for
     a headless install.
   - Optionally adds `C:\Program Files\GopherTrunk` to your **system
     PATH** so `gophertrunk` is reachable from any PowerShell window
     (off by default — tick the "Add GopherTrunk to my PATH" option
     during install if you want it).

When the wizard finishes it offers to open this guide, a console
window, and (if installed) the web console.

## 3. Bind the RTL-SDR to WinUSB with Zadig

Windows ships an RTL-SDR DVB-T driver by default — that's the
broadcast-TV driver, and it's the wrong driver for SDR work. You
need to swap it to **WinUSB** on a per-device basis with **Zadig**.
You only do this once per dongle.

1. Plug in the RTL-SDR dongle.
2. Download Zadig from <https://zadig.akeo.ie> (single .exe, no
   install). Run as **Administrator**.
3. **Options → List All Devices** so the RTL-SDR shows up.
4. From the dropdown, pick the dongle — typically **Bulk-In,
   Interface (Interface 0)** or **RTL2832U** (the NESDR Smart v5
   reports as `RTL2838UHIDIR`).
5. With **WinUSB** selected as the target driver, click **Replace
   Driver** (or **Install Driver** on first run).
6. Wait ~10 seconds for the success dialog.

To restore the original DVB-T driver later (e.g. to watch broadcast
TV again), re-run Zadig and pick the manufacturer driver.

## 4. Verify the install

Open **Windows Terminal** and run:

```powershell
gophertrunk version
gophertrunk sdr list
```

`gophertrunk version` prints the build version, git SHA, and build
timestamp (all pinned at link time via `-ldflags`).

`sdr list` prints one row per attached dongle with its driver,
index, serial, and product string. The TUNER and gains columns are
blank by default — `sdr list` only reads USB descriptors, so it's
fast and never collides with a running daemon. Pass `--probe` when
you want those columns populated (it opens each device briefly to
run the demod + tuner bring-up):

```
> gophertrunk sdr list
DRIVER    IDX  SERIAL            TUNER     PRODUCT   gains(0.1 dB)
rtlsdr    0    00000001                    Generic   []

> gophertrunk sdr list --probe
DRIVER    IDX  SERIAL            TUNER     PRODUCT   gains(0.1 dB)
rtlsdr    0    00000001          R820T2    NESDR Sm  [0 9 14 ... 496]
```

If you see `no SDR devices found`:

- Confirm the dongle is plugged in (LED on, Device Manager shows it).
- Re-run Zadig with **Options → List All Devices** and verify the
  **Driver** column shows **WinUSB**. If it shows
  `RTL2832UUSB` / `RTL28xxBDA`, the swap didn't take.
- If you didn't add GopherTrunk to PATH, run from the install folder:
  ```powershell
  cd "C:\Program Files\GopherTrunk"
  .\gophertrunk.exe sdr list
  ```

List audio output devices too — useful when you plan to enable live
playback (§ 13):

```powershell
gophertrunk audio list
```

## 5. Build your first config

The installer asked you for an "editable files folder" (default
`Documents\GopherTrunk`), seeded a `config.yaml` there, and pinned
the path in `HKCU\Environment\GOPHERTRUNK_CONFIG` so the daemon
discovers it automatically. Open it from the Start Menu shortcut
"Edit my config.yaml (Notepad)" or directly:

```powershell
notepad "$env:USERPROFILE\Documents\GopherTrunk\config.yaml"
```

A read-only reference copy of the full annotated template stays at
`C:\Program Files\GopherTrunk\config.example.yaml` (Start Menu →
"Configuration template").

The full schema reference is in § 11. The bare minimum to get a
working scanner is:

- `sdr.devices[].serial` — match the serial from `sdr list`.
- `trunking.systems[].name` — display name for your trunked system.
- `trunking.systems[].protocol` — `p25`, `dmr`, `nxdn`, `tetra`,
  `motorola`, `edacs`, `ltr`, `mpt1327`, `dpmr`, `dstar`, or `ysf`.
- `trunking.systems[].control_channels` — list of control-channel
  frequencies in Hz.
- `trunking.systems[].talkgroup_file` — path to a Trunk-Recorder-style
  talkgroup CSV. Generate one with `gophertrunk import-pdf` (§ 10)
  or hand-author per
  [`import.md#csv-format`]({{ '/import.html#csv-format' | relative_url }}).

### Interactive wizard

First-time operators can skip hand-editing entirely:

```powershell
gophertrunk import-pdf -wizard
```

The wizard asks one question per config section (log level, API
bind, auth mode, CORS, storage, recordings, retention, SDR devices,
scanner cockpit, audio playback) and writes a fully-annotated
`config.yaml`. Defaults match `config.example.yaml`, so pressing
Enter through every screen still produces a valid file.

Combine with a RadioReference import to bootstrap a region in one
pass:

```powershell
gophertrunk import-pdf -wizard -pdf maricopa.pdf
```

Full reference: [`import.md`]({{ '/import.html' | relative_url }}).

## 6. Run the daemon

```powershell
gophertrunk run
```

The daemon walks `$GOPHERTRUNK_CONFIG` →
`%APPDATA%\GopherTrunk\config.yaml` →
`%USERPROFILE%\Documents\GopherTrunk\config.yaml` → `.\config.yaml`
and loads the first one it finds, printing `config: loaded <path>`
on startup so you can confirm the choice. If you keep multiple
configs in the editable-files folder (e.g. `config.yaml` plus a
`prod.yaml`), the daemon prints a numbered menu and asks which to
load — Enter alone picks #1. A non-interactive launch (Windows
service, Scheduled Task) auto-selects the first match with a
stderr warning instead of hanging.

Override discovery any time:

```powershell
gophertrunk run -config "C:\path\to\other.yaml"
```

Logs stream to the terminal. Press `Ctrl+C` to stop cleanly — the
daemon installs a `signal.NotifyContext` for `SIGINT` / `SIGTERM`,
drains active calls (every `ActiveCall` gets a final `CallEnd`
event so the call log captures it), and closes the database before
exit.

**Daemon flags:**

| Flag | Description |
| --- | --- |
| `-config <path>` | Path to `config.yaml`. Optional — when omitted the daemon walks `$GOPHERTRUNK_CONFIG` → `%APPDATA%\GopherTrunk` → `Documents\GopherTrunk` → cwd and loads the first match (built-in defaults if nothing found). |
| `-log-level <lvl>` | Override `log.level` (`debug` / `info` / `warn` / `error`). |
| `-log-format <fmt>` | Override `log.format` (`text` / `json`). |

The startup log includes a one-line patent-posture banner about
AMBE+2 voice decoding (§ 18). Suppress with the environment
variable `GOPHERTRUNK_QUIET_BANNER=1` if it's noise in your
deployment.

## 7. Configure as a Windows service

For a long-running deployment, register GopherTrunk as a Windows
service with [**NSSM**](https://nssm.cc) — that's the simplest path
until a native service manifest ships.

```powershell
# Download and extract nssm from https://nssm.cc
nssm install GopherTrunk "C:\Program Files\GopherTrunk\gophertrunk.exe" `
  run -config "C:\ProgramData\GopherTrunk\config.yaml"
nssm set GopherTrunk AppDirectory "C:\Program Files\GopherTrunk"
nssm set GopherTrunk DisplayName "GopherTrunk Trunking Daemon"
nssm set GopherTrunk Start SERVICE_AUTO_START
nssm start GopherTrunk
```

Inspect the service:

```powershell
Get-Service GopherTrunk
sc.exe query GopherTrunk
```

NSSM redirects stdout/stderr to its own log files; configure
`nssm set GopherTrunk AppStdout "C:\ProgramData\GopherTrunk\daemon.log"`
plus the matching `AppStderr` to point them somewhere persistent.
Set `nssm set GopherTrunk AppRotateFiles 1` to rotate the logs on
restart.

To stop and remove later:

```powershell
nssm stop GopherTrunk
nssm remove GopherTrunk confirm
```

## 8. Operate from the TUI

The Bubbletea-based terminal UI is built into the same binary as
the daemon — no separate install. It connects to a running daemon
over HTTP and renders eleven panels (Dashboard · Systems ·
Talkgroups · Active · History · Events · Tones · Metrics ·
Devices · Scanner · Settings).

### Launch

In a second PowerShell window with the daemon running in the first:

```powershell
gophertrunk tui
```

Default target is `http://127.0.0.1:8080`. Override with `-server`:

```powershell
gophertrunk tui -server http://10.0.0.5:8080
gophertrunk tui -server https://radio.example.com -insecure
```

**TUI flags:**

| Flag | Default | Purpose |
| --- | --- | --- |
| `-server URL` | `http://127.0.0.1:8080` | daemon base URL |
| `-insecure` | `false` | skip TLS certificate verification |
| `-timeout DURATION` | `5s` | per-request timeout (SSE streams unaffected) |
| `-no-color` | `false` | strip ANSI colour |
| `-write` | `false` | surface mutation keybindings (requires `api.auth.mode != disabled` plus a token or loopback bypass on the daemon) |

### Keybindings — global

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | next / previous panel |
| `1`-`9`, `0` | jump directly to a panel (0 = Scanner) |
| `Ctrl+P` | open the fuzzy command palette |
| `Ctrl+T` | toggle theme (dark ↔ monochrome) |
| `?` | toggle help overlay |
| `q` / `Ctrl+C` | quit |

### Keybindings — inside tables

| Key | Action |
| --- | --- |
| `j` / `↓` | next row |
| `k` / `↑` | previous row |
| `g` / `G` | top / bottom |
| `Page Down` / `Page Up` | scroll a page |

### Keybindings — panel-local

| Panel | Keys |
| --- | --- |
| Systems | `Enter` open detail |
| Talkgroups | `/` filter, `s` cycle sort, `l` toggle lockout, `S` toggle scan, `+` / `-` priority ± 1, `Enter` detail |
| Active calls | `e` end highlighted call (write) |
| Call history | `r` reload (no continuous poll) |
| Events | `/` filter, `p` pause auto-scroll, `c` clear filter |
| Tone alerts | `R` reset detector for highlighted device (write) |
| Metrics | `S` run retention sweep now (write) |
| Scanner | `j` / `k` move, `h` hold/resume, `r` force re-hunt, `Enter` dwell, `L` lockout, `m` cycle scan_mode, `+` / `-` / `M` / `R` volume±/mute/record, `f` manual VFO tune |
| Settings | `[` / `]` cycle tabs |

The TUI is mouse-aware: click a tab to switch panels, click a row
to move the cursor, scroll-wheel up/down to advance rows.

### Settings panel

A read-only inspector of the live daemon configuration, fetched
once at startup and refreshed every 30 s from
`GET /api/v1/runtime`. Cycle tabs (Daemon · Storage · Audio ·
Recording · Tones · API · Vocoders · SDR · FEC) with `[` / `]`.
Every config knob the daemon reads has a touch-point here.

### Mutations (write mode)

Mutation keybindings are hidden by default. To unlock them, both
the daemon and the TUI must opt in:

```yaml
# config.yaml (daemon side)
api:
  http_addr: "127.0.0.1:8080"
  auth:
    mode: "auto"        # loopback bypass; or `required` + token
```

```powershell
# TUI side
gophertrunk tui --write
```

When a mutation requires confirmation, a centered modal opens —
`y` / `Enter` to fire, `n` / `Esc` to cancel.

Full reference: [`tui.md`]({{ '/tui.html' | relative_url }}).

## 9. Operate from the web console

GopherTrunk ships a full browser-based operator console (a
standalone static SPA — pure HTML/CSS/JS, no Node.js, no embedded
server in the daemon). Every TUI panel has a browser counterpart.

### Launch on the same machine as the daemon

Open File Explorer to the folder you picked during install
(default `%USERPROFILE%\Documents\GopherTrunk Web Console`) and
double-click `index.html`. The Start Menu shortcut points at the
same file. On the connect screen, enter:

- **Server URL:** `http://127.0.0.1:8080`
- **Bearer token:** the contents of `api.auth.token_file` (or empty
  if `auth.mode: disabled`)
- **Remember on this device:** tick to store the token in
  `localStorage` instead of `sessionStorage`.

### Operate from another device on the LAN

Canonical "headless box, operate from the couch" scenario.

1. Edit `config.yaml`:

   ```yaml
   api:
     http_addr: "0.0.0.0:8080"
     cors:
       allowed_origins:
         - "null"          # SPA opened via file:// on the laptop
     auth:
       mode: "required"
       token_file: "C:\\ProgramData\\GopherTrunk\\api-token"
   ```

2. Restart the daemon.

3. Copy the `GopherTrunk Web Console` folder to the operating device
   (USB stick, file share, or download the matching release archive
   on that device and use its `gophertrunk-web/` directory).

4. Double-click `index.html`, enter the daemon's LAN URL and the
   bearer token on the connect screen.

### Install as a PWA

The SPA is a Progressive Web App. After connecting once:

- **Desktop Edge / Chrome:** click the install icon in the address
  bar (right side, small monitor glyph).
- **Android Chrome / Edge:** an install banner appears once the
  service worker registers — accept it, or use the browser menu →
  "Install app".
- **iOS Safari:** Share button → "Add to Home Screen".

The installed PWA still talks to whichever daemon URL you set on
the connect screen.

### Write mode in the browser

Mutation buttons (end-call, talkgroup edits, scanner controls,
retention sweep, tone-detector reset) are hidden by default.
Unlock them under **Settings → Allow mutations from this browser**.
If the daemon reports `allow_mutations: false`, fix the
`api.auth.mode` / `trusted_networks` config first (§ 15).

### Audio playback

The audio cockpit lives at the top of the Dashboard. The browser
streams live PCM from `GET /api/v1/audio/stream` — a continuous
open-ended WAV body, no JavaScript decoding required.

iOS / Android browsers require a one-shot "Tap to enable audio"
gesture per session because of autoplay rules. The SPA shows a
prompt on first connect.

Full reference: [`web.md`]({{ '/web.html' | relative_url }}).

## 10. Import trunked systems from RadioReference

The `import-pdf` subcommand parses two source types and merges them
into your `config.yaml`, generating Trunk-Recorder-style talkgroup
CSVs as it goes:

- **RadioReference.com PDF exports** — the "Export PDF" button on
  any P25 trunking-system page.
- **Structured CSV bundles** — a single multi-section CSV file per
  system (format documented in
  [`import.md#csv-format`]({{ '/import.html#csv-format' | relative_url }})).

### Quick start — RadioReference PDF

1. Sign in to RadioReference, open the trunking-system page (e.g.
   "Maricopa County"), click **Export PDF** in the footer, save the
   file.
2. Run:

   ```powershell
   gophertrunk import-pdf `
     -pdf maricopa.pdf `
     -config "$HOME\gophertrunk.yaml"
   ```

3. The review TUI launches. Toggle Include on each site, edit
   Scan / Lockout / Priority on talkgroups, press `w` to write or
   `q` to discard.

### Quick start — CSV bundle

```powershell
gophertrunk import-pdf `
  -csv my-system.csv `
  -config "$HOME\gophertrunk.yaml"
```

### Review TUI keybindings

| View | Key | Action |
| --- | --- | --- |
| Any | `w` | Write merged config + CSVs and exit |
| Any | `q` / `Ctrl+C` | Quit without writing |
| Systems list | `↑` / `↓` | Move cursor |
| Systems list | `Enter` | Open system |
| System (Sites tab) | `Space` | Toggle site Include flag |
| System (any tab) | `Tab` | Switch Sites ↔ Talkgroups |
| Talkgroups | `s` | Toggle Scan |
| Talkgroups | `L` | Toggle Lockout |
| Talkgroups | `0`-`9` | Set Priority (0 clears) |
| Talkgroups | `e` | Edit Alpha Tag |
| System view | `Esc` / `h` | Back to systems list |

### Import flags

| Flag | Description |
| --- | --- |
| `-pdf <file>` | RadioReference PDF (repeatable). |
| `-csv <file>` | Structured CSV bundle (repeatable). |
| `-config <path>` | Existing `config.yaml`, merged in place. Default `./config.yaml`. |
| `-csv-dir <dir>` | Where to write talkgroup CSVs. Default: directory of `-config`. |
| `-no-tui` | Skip the review TUI; merge from parsed defaults. |
| `-dry-run` | Print planned changes and exit without writing. |
| `-force` | Overwrite an existing `trunking.systems[]` entry with the same name. |
| `-wizard` | Launch the interactive config-builder wizard. |

Writes are atomic: each CSV and the config are written to a temp
file in the destination directory and renamed into place after both
struct-level and node-level YAML schema validations pass. Comments
and unrelated blocks in your existing `config.yaml` are preserved
verbatim.

Full reference: [`import.md`]({{ '/import.html' | relative_url }}).

## 11. Configuration reference (`config.yaml`)

Every section of the daemon config, mapped to the schema in
`config.example.yaml`. Defaults are what you get when the key is
omitted entirely.

### `log`

```yaml
log:
  level: info       # debug | info | warn | error
  format: text      # text | json
```

`text` is human-readable; `json` is structured for ingestion into
log aggregators.

### `api`

```yaml
api:
  http_addr: "127.0.0.1:8080"    # HTTP REST + SSE + WebSocket
  grpc_addr: "127.0.0.1:50051"   # gRPC
  allow_mutations: false         # legacy gate; prefer auth.mode
  auth:
    mode: "auto"                 # auto | required | disabled
    # token: "inline-token-here"
    # token_file: "C:\\ProgramData\\GopherTrunk\\api-token"
    # trusted_networks:
    #   - "10.0.0.0/8"
    #   - "192.168.0.0/16"
  cors:
    allowed_origins: []
  tls_cert: ""                   # PEM cert path; pair with tls_key
  tls_key: ""                    # PEM key path
```

See § 15 for the full authentication policy discussion.

### `metrics`

```yaml
metrics:
  enabled: true     # mounts /metrics on the HTTP API
```

See § 16 for the Prometheus series exposed.

### `storage`

```yaml
storage:
  path: "C:\\ProgramData\\GopherTrunk\\calls.db"
  cc_cache_file: "C:\\ProgramData\\GopherTrunk\\cc-cache.json"
```

Use forward slashes or escape backslashes in YAML strings. The
SQLite database holds the call-log history (queried by the TUI
History panel and the web History tab). The CC cache file persists
last-known control-channel frequencies across restarts so the
hunter can lock faster on next boot.

### `recordings`

```yaml
recordings:
  dir: "C:\\ProgramData\\GopherTrunk\\recordings"
  sample_rate: 8000
  write_raw: true       # also append a .raw sidecar with vocoder frames
  equalizer:
    enabled: false      # CMA blind equalizer (simulcast mitigation)
    taps: 8
    step_size: 0.0001
```

Per-call recordings land under
`<dir>\<system>\<talkgroup>\<UTC>_src<id>.wav`. With `write_raw:
true`, a sibling `<UTC>_src<id>.raw` carries the per-frame
compressed vocoder stream — 11 bytes/frame for IMBE, 7 bytes/frame
for AMBE+2 — for offline decoding through DSD-FME, OP25, or
`gophertrunk decode` (§ 18).

The CMA equalizer is **opt-in** because simulcast mitigation costs
CPU and may distort clean-RF capture. Operators not on a simulcast
site shouldn't enable it.

### `retention`

```yaml
retention:
  call_log_days: 30   # 0 disables call-log row sweep
  files_days: 14      # 0 disables filesystem sweep
  interval: "1h"      # how often the sweeper runs
```

The sweeper deletes rows from `calls.db` older than
`call_log_days` and recording files older than `files_days`. Trigger
a sweep on demand from the TUI Metrics panel (`S`) or via
`POST /api/v1/retention/sweep`.

### `sdr`

```yaml
sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"   # match `gophertrunk sdr list`
      role: control        # control | voice | auto
      ppm: 0               # 0 is fine for TCXO-equipped units
      gain: "auto"         # "auto" or tenths-of-dB ("496" = 49.6 dB)
      bias_tee: false      # enable 5V bias-tee for external LNA
    - serial: "00000002"
      role: voice
      ppm: 0
      gain: "auto"
      bias_tee: false
```

- **Roles.** `control` dongles dwell on a system's control channel
  and decode signalling. `voice` dongles follow grants and decode
  voice payloads. `auto` lets the pool assign on first attach.
- **PPM.** Reference-clock offset in parts per million. TCXO units
  (NESDR Smart v5, RTL-SDR Blog v3+) measure within ±0.5 ppm out of
  the box; plain DVB-T sticks usually need a calibration value
  somewhere in ±20 ppm.
- **Gain.** `"auto"` uses the tuner's AGC. A numeric string is the
  tuner gain in tenths of a dB (e.g. `"496"` = 49.6 dB). The
  supported values per tuner are listed in `gophertrunk sdr list`
  under "gains(0.1 dB)".
- **Bias-tee.** Powers an external LNA via the SMA. Only enable on
  dongles that ship with the bias-tee circuit (NESDR Smart v5,
  RTL-SDR Blog v3+ — see [`hardware.md`]({{ '/hardware.html' | relative_url }})).

### `trunking`

```yaml
trunking:
  systems:
    - name: "Example-P25"
      protocol: p25
      control_channels:
        - 851_000_000
        - 852_000_000
      talkgroup_file: "C:\\ProgramData\\GopherTrunk\\talkgroups-p25.csv"
```

**Supported protocols:** `p25` (Phase 1 + Phase 2 share the parent
key — Phase 2 is selected by setting `p25_phase2_*` opt-ins),
`dmr`, `nxdn`, `tetra`, `motorola` (Type II), `edacs`, `ltr`,
`mpt1327`, `dpmr`, `dstar`, `ysf`.

#### FEC opt-outs

Every protocol's forward-error-correction chain is **on by default**.
Operators feeding pre-stripped capture files (DSD-FME `-r` dumps,
OP25 fixtures, MMDVMHost / DSDcc test data) opt out per-system:

| Protocol | YAML key | Opt-out value |
| --- | --- | --- |
| TETRA | `tetra_channel_coding` | `off` |
| LTR FCS | `ltr_fcs_mode` | `off` |
| LTR Manchester | `ltr_manchester_mode` | `off` / `nrz` |
| P25 Phase 2 trellis | `p25_phase2_trellis_mode` | `off` |
| P25 Phase 2 RS | `p25_phase2_rs_mode` | `on` enables verification |
| P25 Phase 2 PN44 scrambler | `p25_phase2_scrambler_mode` | `on` / `probe` |
| NXDN Viterbi | `nxdn_viterbi_mode` | `off` |
| EDACS BCH | `edacs_bch_mode` | `off` |
| MPT 1327 BCH | `mpt1327_bch_mode` | `off` |
| MPT 1327 CWSC tolerance | `mpt1327_cwsc_tolerance` | `0` / `exact` / `off`, or `0`-`15` for custom |
| Motorola Type II BCH | `motorola_bch_mode` | `off` |
| D-STAR FEC | `dstar_fec_mode` | `on` enables the JARL DV-mode chain |

TETRA additionally requires `tetra_colour_code: <non-zero>` for
non-BSCH channels — descrambling produces garbage without it.

#### Receiver clock recovery

P25 Phase 2 and TETRA route through the Gardner symbol-timing-
recovery loop by default. Operators with sample-aligned synthesized
IQ fixtures can opt back to the naive sps-th-sample decimator:

| Receiver | YAML key | Opt-out |
| --- | --- | --- |
| P25 Phase 2 | `p25_phase2_clock_mode` | `naive` / `off` |
| TETRA | `tetra_clock_mode` | `naive` / `off` |

Full reference: [`opt-in-features.md`]({{ '/opt-in-features.html' | relative_url }}).

### `scanner`

See § 12 for the full scanner subsystem reference.

### `audio`

See § 13 for the live-playback config.

### `tone_out`

See § 14 for the paging-tone detector config.

## 12. Scanner subsystems

GopherTrunk runs three scanner subsystems on top of the trunking
engine: the engine-level scan-mode gate, the multi-system control-
channel hunter, and the conventional FM scan list.

```yaml
scanner:
  scan_mode: all       # all | list
  cc_hunt:
    enabled: true
    dwell_ms: 3000
    backoff_ms: 5000
    max_backoff_ms: 60000
  manual_tune_enabled: false
  manual_tune_disabled: false
  conventional: []
```

### `scan_mode`

- **`all`** (default) — follow every non-locked-out grant. The
  backwards-compatible behaviour.
- **`list`** — follow only grants whose talkgroup carries
  `Scan: true` in its CSV row. Emergency grants bypass the gate
  regardless.

Cycle at runtime from the TUI Scanner panel (`m`) or via
`PATCH /api/v1/scanner` body `{"scan_mode":"list"}`.

### `cc_hunt`

Multi-system control-channel hunter. When enabled, the daemon
rotates a free control SDR through every configured system's
control channels until each system locks. Per-system hold / resume /
force-retune from the TUI Scanner panel:

- `h` hold or resume the highlighted system
- `r` force re-hunt (confirms)

### Conventional FM scan list

A fixed-frequency analog FM scanner. Requires a dedicated Voice
SDR — the last Voice device in the pool is used.

```yaml
scanner:
  conventional:
    - label: "Sheriff Repeater"
      frequency_hz: 155895000
      mode: fm
      squelch_dbfs: -48
      hangtime_ms: 1500
      priority: 4
      tone:
        mode: ctcss        # ctcss | dcs | none
        ctcss_hz: 100.0
        # dcs_code: "023"  # 3-digit octal for DCS
```

**Per-channel knobs:**

| Knob | Description |
| --- | --- |
| `label` | Display name |
| `frequency_hz` | Tuner centre frequency |
| `mode` | `fm` (default), `nfm`, `am`; protocols extend over time |
| `squelch_dbfs` | IQ-power squelch threshold in dBFS |
| `hangtime_ms` | Carrier-lost dwell before hopping |
| `priority` | Integer 0–10 for scan order |
| `tone` | Optional CTCSS / DCS sub-audible squelch gate |

With `tone` configured, the scanner only opens when both the
carrier is present **and** the configured tone is detected, so
adjacent-system traffic on the same frequency doesn't trigger a
false dwell. Omit the block (or set `mode: none`) for plain
carrier-only squelch.

### Manual VFO tune

Press `f` on the Scanner panel to enter a frequency in MHz and
listen immediately (a runtime VFO channel is added and the scanner
dwells). The TUI surfaces the input when:

- **Auto-detect.** ≥ 2 Voice SDRs are present in the pool — the
  daemon constructs the conventional scanner off the spare.
- **Forced.** `manual_tune_enabled: true` constructs the scanner
  even with a single Voice SDR (steals it from the trunking pool).
- **Vetoed.** `manual_tune_disabled: true` overrides the auto-detect.

Web console exposes the same control under the Scanner tab.

## 13. Audio playback and recordings

Live audio playback routes decoded PCM to your Windows sound
device. **Disabled by default** so headless / service deployments
stay silent; WAV recording is unaffected.

```yaml
audio:
  enabled: false       # set true to play decoded calls live
  device: ""           # empty = system default; "null" forces no-op
  sample_rate: 8000    # must match recordings.sample_rate
  buffer_ms: 80        # playback queue depth; higher = more jitter-tolerant
  volume: 0.8          # 0..1 software gain
  muted: false
```

List the available output devices first:

```powershell
gophertrunk audio list
```

Set `device` to one of the listed names to pin output, or leave
empty for the system default sink. The Windows backend uses
WASAPI; failure to initialise (no sound device, exclusive-mode
contention) falls back to a silent player automatically so the
daemon stays running.

Recordings always land on disk per § 11 (`recordings.dir`). The
`.raw` sidecar (per-call vocoder frames) makes it easy to re-decode
post hoc through DSD-FME / OP25 or `gophertrunk decode` (§ 18).

Mutate audio at runtime from the TUI Scanner panel:

| Key | Effect |
| --- | --- |
| `+` / `-` | Volume ± 5% |
| `M` | Mute toggle |
| `R` | Recording toggle |

The web console exposes the same controls on the Dashboard audio
cockpit. Live PCM streams to the browser via
`GET /api/v1/audio/stream` — a continuous open-ended WAV body.

## 14. Tone-out paging-tone alerts

GopherTrunk fires `tone.alert` events when configured paging tones
are detected on a Voice device's PCM stream — the most common use
case is two-tone sequential (Motorola Quick Call II) fire/EMS
dispatch.

```yaml
tone_out:
  profiles:
    - name: "station-1-engine"
      alpha_tag: "Station 1 Engine"
      cooldown: "30s"
      tones:
        - frequency_hz: 1042.2
          min_duration: "250ms"
          max_duration: "1500ms"
        - frequency_hz: 1297.4
          min_duration: "2.5s"
          max_duration: "5s"
```

**Per-profile schema:**

| Key | Description |
| --- | --- |
| `name` | Required, unique within `profiles` |
| `alpha_tag` | Human-readable label (UI / webhook / log) |
| `tones[]` | Ordered tone list, ≥ 1 entry |
| `tones[].frequency_hz` | Target tone in Hz |
| `tones[].min_duration` | Required dwell before counting |
| `tones[].max_duration` | Caps the on-time; 0 = no upper bound |
| `tolerance_hz` | Frequency drift tolerance (default 15 Hz) |
| `magnitude_threshold` | Goertzel-magnitude floor (default 0.05) |
| `max_gap` | Silence allowed between tones (default 200ms) |
| `cooldown` | Re-fire suppression window (default 30s) |
| `system` | Restrict to one trunked system (empty = all) |
| `group_id` | Restrict to one talkgroup (0 = all) |

Multiple profiles coexist; the detector fires the first one whose
tone sequence matches. The bundled `config.example.yaml` includes
commented-out single-tone, system-scoped, and tight-tolerance
examples.

View live tone alerts in the TUI **Tones** panel or the web
console **Tones** tab. Reset the detector for a device (`R` in the
TUI) when you want to re-arm after a false positive.

## 15. API authentication and hardening

Every HTTP mutation endpoint (end-call, talkgroup priority / lockout /
scan, retention sweep, tone-detector reset, scanner cockpit, audio
cockpit, manual tune) is gated by `api.auth`.

### Policy modes

- **`auto`** (default) — require a bearer token on non-loopback
  binds; bypass the check on loopback (`127.0.0.1` / `::1`).
  Reasonable for single-host operator boxes — kernel-enforced
  reachability is a peer-cred proxy. The daemon refuses to start in
  `auto` mode on a public bind without a configured token.
- **`required`** — every mutation request must carry a valid Bearer
  token, even from loopback. Use when the daemon shares a host
  with untrusted users.
- **`disabled`** — wide-open mutations, no auth. Equivalent to the
  legacy `allow_mutations: true`. Only safe behind an external
  proxy that does its own auth.

### Generating a token

```powershell
# 32 bytes of random, hex-encoded — 64 ASCII chars.
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
([System.BitConverter]::ToString($bytes) -replace '-', '').ToLower() `
  | Out-File -Encoding ascii "C:\ProgramData\GopherTrunk\api-token"
```

Then in `config.yaml`:

```yaml
api:
  http_addr: "0.0.0.0:8080"
  auth:
    mode: "required"
    token_file: "C:\\ProgramData\\GopherTrunk\\api-token"
```

The daemon re-reads `token_file` on every mutation request, so
rotation is a one-step file overwrite — no restart, no SIGHUP.

### Client usage

```powershell
$token = Get-Content "C:\ProgramData\GopherTrunk\api-token"
curl.exe -sS http://daemon:8080/api/v1/mutations
curl.exe -sS -X POST `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json" `
  -d '{"reason":"manual"}' `
  http://daemon:8080/api/v1/calls/00000001/end
```

`GET /api/v1/mutations` is always open and reports `auth_mode` +
`can_mutate` for the current request, so TUIs / scripts can light
up write-side keybindings without probing real endpoints.

### Trusted networks (LAN bypass)

For LAN deployments where you trust the whole segment:

```yaml
api:
  http_addr: "192.168.1.10:8080"
  auth:
    mode: "auto"
    trusted_networks:
      - "192.168.0.0/16"
```

The middleware honours `RemoteAddr` only — `X-Forwarded-For` is
intentionally ignored so the bypass isn't forgeable by a hostile
upstream proxy.

### CORS (browser access)

```yaml
api:
  cors:
    allowed_origins:
      - "null"                          # SPA opened via file://
      - "http://laptop.local:8000"      # static-host alternative
      # - "*"                           # any origin (loopback only)
```

`"null"` covers the canonical case where the web console is opened
via `file://` from File Explorer. The browser sends the literal
string `null` as the Origin for those loads.

### TLS

```yaml
api:
  http_addr: ":8080"
  grpc_addr: ":50051"
  tls_cert: "C:\\ProgramData\\GopherTrunk\\tls\\cert.pem"
  tls_key:  "C:\\ProgramData\\GopherTrunk\\tls\\key.pem"
```

Both keys must be set together — setting one without the other is
a config error the daemon refuses to start with. The same cert /
key pair is used for both the HTTP and gRPC listeners. Cert
rotation requires a daemon restart.

Generate a test cert with OpenSSL for Windows or via WSL:

```powershell
openssl req -x509 -newkey rsa:2048 -nodes -days 365 `
  -keyout C:\ProgramData\GopherTrunk\tls\key.pem `
  -out    C:\ProgramData\GopherTrunk\tls\cert.pem `
  -subj "/CN=gophertrunk.example.com"
```

Full reference: [`hardening.md`]({{ '/hardening.html' | relative_url }}).

## 16. Metrics and health endpoints

### `GET /metrics`

Prometheus exposition (when `metrics.enabled: true`):

| Series | Type | Description |
| --- | --- | --- |
| `gophertrunk_events_total{kind=...}` | counter | Every event observed on the internal bus |
| `gophertrunk_calls_total{reason=...}` | counter | Calls completed by EndReason |
| `gophertrunk_calls_active` | gauge | Currently-active call count |
| `gophertrunk_control_channel_locked{system=...}` | gauge | 1 while CC locked |
| `gophertrunk_sdr_iq_underruns_total{driver,serial}` | counter | IQ pipeline drops |
| `gophertrunk_sdr_usb_reconnects_total{driver,serial}` | counter | USB re-opens |
| `gophertrunk_decode_errors_total{protocol,stage}` | counter | Decode failures |
| `gophertrunk_sdr_attached{driver,serial,role}` | gauge | 1 per attached SDR |
| `gophertrunk_build_info{version}` | gauge | Always 1; build version label |

### `GET /api/v1/health`

Always open (no token required) so liveness / readiness probes can
hit it from outside the auth boundary:

```json
{
  "status":              "ok",
  "now":                 "2026-05-13T19:00:00Z",
  "version":             "v1.2.3",
  "pool_attached_count": 2,
  "active_calls":        1,
  "db_connected":        true,
  "metrics_enabled":     true,
  "auth_mode":           "auto"
}
```

Probe semantics:

| Probe type | Condition |
| --- | --- |
| Liveness | HTTP 200 + body decodes |
| Readiness | `status == "ok"` AND `pool_attached_count >= 1` AND `db_connected == true` |

## 17. CLI reference

```
gophertrunk [run] [-config path]    run the daemon (default)
gophertrunk sdr list                list discovered SDR devices
gophertrunk audio list              list audio output devices
gophertrunk tui [-server URL]       open the operator TUI
gophertrunk decode [flags]          decode a captured .raw frame stream into a WAV
gophertrunk import-pdf [flags]      import a RadioReference PDF / CSV bundle
gophertrunk version                 print build version + git SHA + build time
gophertrunk help                    show usage
```

### `decode`

Run the registered in-binary vocoders against a `.raw` frame stream
out-of-band:

```powershell
gophertrunk decode -in call.raw  -out call.wav  -vocoder imbe
gophertrunk decode -in dmr.raw   -out dmr.wav   -vocoder ambe2
gophertrunk decode -list-vocoders        # enumerate registered names
```

Stdin / stdout work via `-in -`, so capture pipelines stream into
the decoder without a temporary file. See § 18 for the supported
vocoders.

## 18. Vocoders and voice calibration

GopherTrunk ships pure-Go IMBE (P25 Phase 1) and AMBE+2 (P25 Phase
2, DMR Tier II/III, NXDN, dPMR, D-STAR voice) decoders. Both are
on by default; AMBE+2 is **patent-encumbered** in some
jurisdictions (DVSI IPR portfolio), and the legal responsibility
for operating it falls on the deployer — see
[`vocoders.md`]({{ '/vocoders.html' | relative_url }}) §"Patent
posture".

| Backend | Build tag | Default? | Notes |
| --- | --- | --- | --- |
| `null` (silence) | none | yes | Always available |
| `imbe` (pure-Go) | none | yes | P25 Phase 1 LDU1 / LDU2 |
| `ambe2` (pure-Go) | none | yes | P25 Phase 2 / DMR / NXDN / dPMR / D-STAR |
| `dvsi` (USB-3000 chip) | `-tags dvsi` | no | Wire protocol shipping; USB transport stub |

Live-pipeline auto-decode maps `Grant.Protocol` to a vocoder per
the [default mapping]({{ '/vocoders.html' | relative_url }}#live-pipeline-auto-decode);
override with `RecorderOptions.VocoderForProtocol` if you're
embedding the recorder.

### Voice calibration

To tune the in-tree decoders' loudness against DSD-FME / OP25
reference output, follow the recipe in
[`voice-calibration.md`]({{ '/voice-calibration.html' | relative_url }}):

1. Record a reference call with `recordings.write_raw: true`.
2. Decode the `.raw` through DSD-FME / OP25 to get a reference WAV.
3. Run `cmd/voice-calibrate` (or the in-tree
   `internal/voice/calibrate` unit test) to compute RMS-ratio (dB)
   and best-alignment cross-correlation.
4. Tune `internal/voice/mbe/agc.go::TargetPeak` if the RMS-ratio is
   outside ±3 dB.

Acceptance: `|RMSRatioDb| < 3.0` and `PeakXcorr > 0.85`.

## 19. Upgrading and uninstalling

### Upgrade

Run a newer installer in place — it overwrites
`C:\Program Files\GopherTrunk\gophertrunk.exe` and refreshes the
Start Menu entries. Your `config.yaml`, recordings, and call-log DB
(wherever you wrote them) are left alone. If `gophertrunk` is
running as an NSSM service, stop it first:

```powershell
nssm stop GopherTrunk
.\gophertrunk-<version>-windows-amd64-setup.exe
nssm start GopherTrunk
```

After upgrade, confirm the new build:

```powershell
gophertrunk version
```

### Uninstall

**Settings → Apps → Installed apps → GopherTrunk → Uninstall.**
The uninstaller removes the install folder, every Start Menu
entry, and undoes the PATH change if you opted in.

Recordings under your `recordings.dir`, the SQLite call log, and
the CC cache file are **not** removed — they live under
`ProgramData` or your home directory and remain on disk. Delete
them manually if you want a clean slate:

```powershell
Remove-Item -Recurse "C:\ProgramData\GopherTrunk"
Remove-Item "$HOME\gophertrunk.yaml"
```

If you registered an NSSM service, remove it before uninstall:

```powershell
nssm stop GopherTrunk
nssm remove GopherTrunk confirm
```

## 20. Troubleshooting

| Symptom | Likely cause + fix |
| --- | --- |
| `gophertrunk` not recognised in PowerShell | PATH wasn't added during install — open a fresh terminal, or run from `C:\Program Files\GopherTrunk` directly. |
| `sdr list` prints `no SDR devices found` | Zadig WinUSB swap didn't take. Re-run Zadig with **Options → List All Devices** and verify the **Driver** column shows **WinUSB**. |
| `usb: device disconnected` mid-stream | The DVB driver re-attached itself, or Windows USB selective-suspend kicked in. Re-run Zadig; in Device Manager, disable "Allow the computer to turn off this device" under the USB hub's Power Management tab. |
| `WinUsb_Initialize` fails | The dongle is bound to the wrong driver — re-run Zadig and pick **WinUSB**. |
| SmartScreen blocks the installer | Right-click → Properties → Unblock, or **More info → Run anyway**. |
| Audio plays as silence | `audio.enabled: false` by default — set `true` in config. Confirm the device name in `gophertrunk audio list`. |
| `daemon unreachable` in the TUI | Daemon isn't running, or the `-server` URL points at the wrong host / port. Run `curl.exe http://127.0.0.1:8080/api/v1/health` to confirm. |
| Web console connect screen shows "Failed to fetch" | Daemon not reachable. Confirm `gophertrunk run` is up and the URL is right. |
| Browser console shows "CORS preflight rejected" | Daemon hasn't allow-listed your SPA's origin. Add it under `api.cors.allowed_origins` and restart. |
| 401 on every web-console request | Wrong bearer token. Re-check `api.auth.token_file` contents and re-enter on the connect screen. |
| Mutation buttons invisible in the web UI | Write mode is off or the daemon rejects mutations. Settings → tick "Allow mutations". |
| iOS / Android audio stops after 2 seconds | Autoplay block. Tap the "Tap to enable audio" prompt on the Dashboard. |
| NSSM service exits immediately | Stdout/stderr aren't redirected — `nssm set GopherTrunk AppStdout C:\ProgramData\GopherTrunk\daemon.log` and re-check the log. |
| TUI event stream toast keeps reappearing | SSE stream dropping. Check the daemon log; corporate proxies sometimes truncate SSE. |

For anything else, open an issue at
<https://github.com/MattCheramie/GopherTrunk/issues> with:

- `gophertrunk version` output
- the first ~50 lines of the daemon log
- `gophertrunk sdr list` output
- relevant excerpts of your `config.yaml`

## See also

- [Windows install (5-minute path)]({{ '/install-windows.html' | relative_url }})
- [Hardware setup]({{ '/hardware.html' | relative_url }})
- [TUI]({{ '/tui.html' | relative_url }}) / [Web console]({{ '/web.html' | relative_url }})
- [Import (PDF / CSV)]({{ '/import.html' | relative_url }})
- [Hardening & operations]({{ '/hardening.html' | relative_url }})
- [Opt-in features]({{ '/opt-in-features.html' | relative_url }})
- [Vocoders]({{ '/vocoders.html' | relative_url }}) / [Voice calibration]({{ '/voice-calibration.html' | relative_url }})
- [Architecture]({{ '/architecture.html' | relative_url }})
