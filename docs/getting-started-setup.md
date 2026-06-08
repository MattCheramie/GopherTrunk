---
layout: page
title: Get started
description: From download to your first decoded call in a few steps
nav_group: Getting Started
---

# Get started

A straightforward path from a fresh download to your first decoded call. New to
the project? Read [What is GopherTrunk?](getting-started.html) first for the
big picture. The steps below use Linux x86_64; macOS, Windows, and ARM64 follow
the same shape — see the per-OS pages linked in step 2.

## 1. Connect a radio

Plug in an SDR dongle. An **RTL-SDR** is the cheapest place to start. Attach an
antenna suited to your band (for ~700–900 MHz public-safety trunking, a simple
discone or a tuned whip works). Device-by-device guidance — gain, bias-tee,
remote backends — is in the [Hardware guide](hardware.html).

## 2. Download the binary

```sh
# Linux x86_64 — see https://gophertrunk.org/downloads.html for macOS, Windows, ARM64.
VERSION=v0.3.5
curl -L -o gophertrunk.tar.gz \
  https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/gophertrunk-${VERSION}-linux-amd64.tar.gz
tar xzf gophertrunk.tar.gz && cd gophertrunk-${VERSION}-linux-amd64
```

On other platforms, follow the per-OS recipe instead — Windows ships a one-click
installer that bundles Zadig for WinUSB setup, and macOS ships notarised
tarballs:

- [Linux install](install-linux.html)
- [macOS install](install-macos.html)
- [Windows install](install-windows.html)
- [All downloads & checksums](downloads.html)

## 3. Verify the binary and your radio

```sh
./gophertrunk version          # confirms the binary runs
./gophertrunk sdr list         # should list your dongle(s)
```

If `sdr list` doesn't see your dongle, run the diagnostics:

```sh
./gophertrunk sdr doctor -v    # explains why a dongle isn't recognized
```

## 4. Create your config

```sh
cp config.example.yaml config.yaml
```

`config.example.yaml` is heavily annotated. The one thing a newcomer must set is
**at least one system** under `trunking:` — its protocol and control-channel
frequency. The shape is simple:

```yaml
trunking:
  systems:
    - name: "Example-P25"
      protocol: p25
      control_channels:
        - 851_000_000
      talkgroup_file: "/etc/gophertrunk/talkgroups-p25.csv"   # optional alpha tags
```

Don't hand-edit if you don't have to:

- **[Import (PDF / CSV)](import.html)** — pull a system straight from a
  RadioReference PDF/CSV (`gophertrunk import-pdf`, or the Import panel in any
  UI).
- **Config Builder** — `gophertrunk config serve` opens a guided editor in your
  browser.
- **[Live edits](live-edits.html)** — most settings can be changed from a
  running UI without restarting the daemon.

## 5. Run it

```sh
./gophertrunk -config config.yaml
```

On a terminal this drops into the **launcher** — pick **[1] TUI**, **[2] Web**,
or **[3] Headless**. Skip the prompt with a flag:

```sh
./gophertrunk -tui  -config config.yaml   # in-process terminal console
./gophertrunk -web  -config config.yaml   # open the browser console
./gophertrunk -headless -config config.yaml   # silent daemon
```

More on the menu and how each mode attaches: [Launcher](launcher.html).

## 6. Watch your first call

Once running, you're looking for this sequence:

1. **CC locks** — the daemon synchronises to the control channel.
2. **A grant fires** — the system assigns a talkgroup to a voice channel.
3. **A call records** — audio is captured and written to your recordings
   directory (set under `recordings:` in the config) and logged to the SQLite
   call database.

Read the panels — active calls, history, control-channel activity — via the
[TUI](tui.html) or the [Web console](web.html).

## 7. Next steps

- **Share your feed** — stream completed calls to Broadcastify Calls,
  RdioScanner, OpenMHz, or Icecast/ShoutCast (`broadcast:` config).
- **[Hunt](hunt.html)** — discover and map an unknown trunked system.
- **[Hardening](hardening.html)** — API auth, TLS, and safe remote access.
- **[Opt-in features](opt-in-features.html)** — paging, APRS, AIS, ADS-B, and
  other extras.

## Troubleshooting

- **Dongle not found** → `./gophertrunk sdr doctor -v`, then the
  [Hardware guide](hardware.html).
- **No audio on playback** → `./gophertrunk audio list` to check output devices.
- **Voice grants dropped on DMR Tier III** → the system needs a `dmr_band_plan`
  (see the annotated `config.example.yaml`).
