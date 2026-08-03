---
title: "Running It For Real, Part 7: SDR Doctor & Preflight"
description: How GopherTrunk catches a bad dongle, wrong USB driver, or broken config before it costs a call — the read-only sdr doctor that inspects per-device driver binding, the preflight that validates directories and TLS files before the listener binds, and the gain-sanity warnings.
category: deep-dives
keywords: sdr doctor, usb driver binding, zadig winusb, dvb blacklist, config preflight, tls keypair validation, gain sanity check, rtl-sdr diagnostics, gophertrunk running it for real
tags: [running-it-for-real, diagnostics, sdr, deployment, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 7
---

*Part 7 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Part 6 made errors carry their context;
this post is about not reaching the error at all. The most expensive failure for a
scanner is the one you don't see — a dongle bound to the wrong driver, a config
directory that can't be written, a TLS key owned by the wrong user — because it
turns into hours of silence and a missed call. GopherTrunk front-loads those into
two read-only checks that fail loudly *before* the daemon commits to a run.*

> **TL;DR:** Two preventive tools sit in front of the daemon. **`sdr doctor`**
> (`cmd/gophertrunk/sdr_doctor.go`) is a read-only inspector: it walks the known
> RTL-SDR and HackRF VID/PID table, asks the platform which kernel/Windows
> function driver is bound to each matching dongle, and prints a per-device row
> with an actionable next step — the fix for the classic "plugged in but won't
> open" (DVB driver on Linux, no WinUSB on Windows). It never opens or claims a
> device, so it's safe to run alongside a live daemon. **`preflight`**
> (`cmd/gophertrunk/preflight.go`) runs after config load and before `NewDaemon`,
> converting "silently goes bad at runtime" into "fails loudly with a clear
> message": it creates the write directories, validates the TLS keypair actually
> parses, and stat-checks talkgroup/alias files. Layered on top, the daemon's
> construction-time **gain sanity warnings** catch the deaf front-end.

**Key takeaways**

- **`sdr doctor` diagnoses driver binding, read-only.** It inspects each known
  dongle's bound driver and prints OK/BAD plus a next step, without ever opening
  the device — so it runs safely while the daemon holds the claim.
- **Preflight fails before the listener binds.** Unwritable directories and a
  broken TLS keypair become a clear `preflight: …` error up front, not an opaque
  goroutine failure after the port is already open.
- **Non-fatal misconfig becomes a visible warning, not silence.** A missing
  talkgroup CSV or a storage-less paging setup surfaces as a startup warning the
  launcher and TUI pin — the silent-failure class (issue #565) turned actionable.
- **The gain traps have named warnings.** A `32`-for-`320` tenths mistake, a too-
  low manual gain, and a fixed gain on a shared wideband front-end each get a
  specific WARN — the deaf-radio failure the IQ-power gauge alone can't explain.

## Cheat sheet

| Check | Catches | Where it lives |
|---|---|---|
| `sdr doctor` | wrong/unbound USB driver (DVB, no WinUSB) | `cmd/gophertrunk/sdr_doctor.go` |
| driver inspector | per-VID/PID binding, platform-specific | `internal/sdr/rtlsdr/usb` (`DefaultDriverInspector`) |
| preflight dirs | unwritable recordings/storage/cc-cache | `cmd/gophertrunk/preflight.go` |
| preflight TLS | cert/key that don't parse as a keypair | `preflight.go` (`tls.LoadX509KeyPair`) |
| preflight files | missing/empty talkgroup + RID alias CSVs | `preflight.go` |
| gain sanity | deaf front-end (units, too-low, fixed-wideband) | `cmd/gophertrunk/daemon.go` (`warnGainUnits`, `warnLowGain`) |

## In this post

- **The cost of a silent hardware failure** — and why preflight exists.
- **`sdr doctor`** — read-only driver-binding diagnosis with a next step.
- **Preflight** — fail loud, before the listener binds.
- **The gain traps** — the deaf-radio warnings the daemon fires at construction.

## The cost of a silent hardware failure

Every other kind of bug announces itself. A silent hardware failure doesn't — the
daemon starts clean, the logs flow, the metrics look plausible, and no calls come
in. On a laptop you'd notice in seconds and start poking. As an unattended feed,
that state can persist for hours because nothing *errored* — the dongle just never
delivered usable samples. The three classic causes are all catchable before the
fact: the wrong driver is bound to the dongle (so it can't be opened at all), the
config points somewhere unwritable (so recordings silently vanish), or the gain is
set so the front-end is effectively deaf. GopherTrunk turns each into a check that
runs *before* the daemon is trusted to keep a feed alive.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="Two gates stand before a running daemon: sdr doctor read-only checks driver bindings, and preflight fails loud on driver, path, or gain problems, so a bad radio is caught before the listener binds instead of producing a silent no-calls feed">
  <rect x="8" y="66" width="126" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="71" y="85" text-anchor="middle" fill="var(--accent)" font-size="12">sdr doctor</text>
  <text x="71" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">read-only · bindings</text>
  <line x1="134" y1="89" x2="162" y2="89" stroke="currentColor"/><polygon points="162,85 172,89 162,93" fill="currentColor"/>
  <rect x="172" y="66" width="126" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="235" y="85" text-anchor="middle" fill="var(--accent)" font-size="12">preflight</text>
  <text x="235" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">driver · path · gain</text>
  <line x1="298" y1="89" x2="326" y2="89" stroke="currentColor"/><polygon points="326,85 336,89 326,93" fill="currentColor"/>
  <rect x="336" y="66" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="411" y="85" text-anchor="middle" fill="currentColor" font-size="12">bind listener</text>
  <text x="411" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">daemon trusted</text>
  <line x1="486" y1="89" x2="514" y2="89" stroke="currentColor"/><polygon points="514,85 524,89 514,93" fill="currentColor"/>
  <rect x="524" y="66" width="128" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="588" y="92" text-anchor="middle" fill="currentColor" font-size="12">24/7 feed</text>
  <line x1="235" y1="112" x2="235" y2="140" stroke="var(--accent)" stroke-dasharray="4 3"/><polygon points="231,140 235,150 239,140" fill="var(--accent)"/>
  <text x="235" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="10">fail loud here — never a silent no-calls daemon</text>
</svg>
<figcaption>Both checks run before the listener binds. A wrong driver, an unwritable path, or a deaf gain setting stops startup with a message, instead of a daemon that looks healthy and hears nothing.</figcaption>
</figure>

## `sdr doctor`

`gophertrunk sdr doctor` answers one question precisely: *for each USB SDR I know
about, is the right driver bound to it?* On Linux the trap is the kernel's DVB
driver (`dvb_usb_rtl28xxu`) claiming the dongle before GopherTrunk can; on Windows
it's the absence of a WinUSB binding (the Zadig step). The doctor walks the union
of every pure-Go USB SDR's VID/PID table — RTL-SDR plus HackRF — and asks the
platform inspector what's bound:

```go
// cmd/gophertrunk/sdr_doctor.go (shape) — runSDRDoctor
inspector := usb.DefaultDriverInspector()
for _, pair := range knownDoctorDevices() { // RTL-SDR + HackRF VID/PIDs
    bindings, err := inspector.Inspect(pair.VID, pair.PID)
    if errors.Is(err, usb.ErrUnsupportedPlatform) {
        fmt.Fprintf(os.Stderr, "sdr doctor: no driver inspector for %s/%s\n", runtime.GOOS, runtime.GOARCH)
        return
    }
    for _, b := range bindings {
        rows = append(rows, b) // deduped by VID:PID:path
    }
}
printDoctorRows(rows, *verbose)
```

The output is a tab-aligned table — `VID:PID`, `SERIAL`, `DRIVER`, `EXPECTED`,
`STATUS` (OK/BAD), and a `NEXT-STEP` hint — mirroring `sdr list`'s column widths so
the two commands read the same. A HackRF that shows up in `sdr list` but fails to
open surfaces here with a wrong-driver hint instead of being invisible. And when
*no* dongle is found at all, the doctor doesn't just shrug — it prints
platform-specific first steps: on Linux, check `lsusb` for the VID:PID (RTL-SDR is
typically `0bda:2832`/`0bda:2838`, HackRF `1d50:6089`) and confirm
`/sys/bus/usb/devices` is readable; on Windows, open Device Manager, try a USB 2.0
port, and run Zadig with *List All Devices*.

The load-bearing property is in the doc comment: **read-only**. It *never opens or
claims a USB device* — it only inspects the driver binding — so you can run it as a
regular user, alongside a live daemon that already holds the claim, without
disturbing anything. That's what makes it a triage tool you reach for when a
running daemon isn't decoding, not just a first-run setup step.

### How that principle shaped the Go code

- **The inspector is an interface with a platform default.**
  `usb.DefaultDriverInspector()` returns the right implementation per OS, and an
  unsupported platform returns `ErrUnsupportedPlatform` so the command degrades to
  a clear message rather than a panic.
- **Every dongle identity is checked.** Windows binds a function driver *per
  VID/PID at plug time*, so the doctor has to inspect each known identity — a
  dongle that enumerates under an unexpected PID still gets a row and a hint.
- **Rows are deduped by path.** A `VID:PID:path` key means a dongle that matches
  multiple table entries appears once, not three times.
- **`-v` adds the driver description column.** The verbose flag surfaces the raw
  bound-driver name for the cases where the OK/BAD verdict needs backing evidence.

## Preflight

Where `sdr doctor` is a tool you run on demand, `preflight` runs automatically —
after `config.Load` and before `NewDaemon` — and its whole job is to convert a
class of *runtime* failures into *startup* failures. The subsystems it guards
(recorder, storage, cc-cache) each create their directories lazily and validate
their own files, but lazily means the failure lands somewhere in the log after the
daemon is already up and the listener already bound. Preflight moves that forward:

```go
// cmd/gophertrunk/preflight.go (shape) — the fatal checks
// 1. Auto-create the write directories, so a permission problem is one clear
//    error instead of a runtime warning buried in the log.
for _, d := range []struct{ label, path string }{
    {"recordings.dir", cfg.Recordings.Dir},
    {"storage.path (parent)", parentDir(cfg.Storage.Path)},
    {"storage.cc_cache_file (parent)", parentDir(cfg.Storage.CCCacheFile)},
} {
    if d.path == "" { continue }
    if err := os.MkdirAll(d.path, 0o755); err != nil {
        return warnings, fmt.Errorf("preflight: %s: mkdir %q: %w", d.label, d.path, err)
    }
}
// 2. Verify the TLS cert/key actually load as an X.509 keypair, so a typo or a
//    mode-0600-from-another-user surfaces as `preflight: tls …` and NOT as an
//    opaque goroutine error after the listener has already bound.
if cfg.API.TLSCert != "" && cfg.API.TLSKey != "" {
    if _, err := tls.LoadX509KeyPair(cfg.API.TLSCert, cfg.API.TLSKey); err != nil {
        return warnings, fmt.Errorf("preflight: tls cert/key (%s, %s): %w",
            cfg.API.TLSCert, cfg.API.TLSKey, err)
    }
}
```

That TLS check is the second guard on the cert/key pair (Part 3 covered the first,
the both-or-neither XOR at server construction) — and it's the one that catches a
file that *exists* but doesn't parse, which the XOR check can't. Both fire before a
socket opens.

The rest of preflight is *non-fatal* and returns warnings rather than errors,
because these degrade the daemon without breaking it. A talkgroup CSV that's
missing, a directory instead of a file, or empty becomes a warning ("calls on this
system will have no alpha tags") — the daemon runs fine, just without the friendly
labels. The RID alias file gets the same stat-only treatment. And the check that
earns its keep most is the storage-dependency one:

```go
// cmd/gophertrunk/preflight.go (shape) — the silent-failure guard (issue #565)
if cfg.Storage.Path == "" {
    var needs []string // paging / aprs / ais / dsc / mdc1200 / m17 all need storage
    // …collect the configured-but-storage-less decoders…
    if len(needs) > 0 {
        warnings = append(warnings, fmt.Sprintf(
            "storage.path is empty but these decoders need it to surface decoded "+
                "messages: %s — they will run, but their REST endpoints return 503 "+
                "and the web panels stay empty. Set storage.path to enable persistence.",
            strings.Join(needs, ", ")))
    }
}
```

This one exists because a real operator (issue #565) configured POCSAG paging,
saw the pager panel return a bare `503`, and had to discover on their own that the
missing piece was `storage.path`. The decoders run and consume live IQ, but with
no storage their output is never persisted and the REST endpoints 503. Naming the
dependency at load time — alongside the other warnings the launcher and TUI pin —
turns a silent misconfiguration into an actionable one. That's the whole ethos of
this post in one warning: a service should tell you what's wrong *before* you go
looking for why it's quiet.

## The gain traps

The subtlest silent failure is a radio that's technically working but deaf, and it
has three flavours the daemon warns about at construction time — the last line of
defence after the doctor and preflight. The nastiest is a units mistake:
GopherTrunk's `gain:` is in *tenths* of a dB (`320` = 32 dB), but SDRTrunk, OP25,
and gqrx all take whole dB, so an operator's muscle memory lands `32` — which
`SetGain` reads as 3.2 dB and snaps to the bottom of the ladder, leaving the radio
effectively deaf. `warnGainUnits` catches exactly that (a bare integer ≤ 50 tenths)
and prints the fix: *did_you_mean 320*. `warnLowGain` catches the next band up — a
real but too-low manual gain (below ~15 dB) that won't lift a digital signal above
the noise. And `warnFixedGainOnSharedWideband` catches a multi-tap wideband dongle
pinned to one manual gain, where a single value can't serve co-tenant sites of
differing strength and the weak ones sit dead at the ADC floor. Each is a specific,
actionable WARN rather than a mysteriously silent front-end — and each points at
the metric (Part 4's `iq_power_dbfs`, `iq_clip_ratio`) that confirms the diagnosis.

## Where this goes next

Doctor, preflight, and the gain warnings keep a *misconfigured* daemon from
running quietly broken. The next question is scope — what should a healthy daemon
even be *running*? [Part 8]({{ '/blog/deep-dives/running-it-for-real-08-opt-in-feature-matrix/' | relative_url }})
is the opt-in feature matrix: how GopherTrunk keeps optional subsystems (paging,
AIS, ADS-B, broadcast uploads, the extra decoders) off until you ask for them, so
the default daemon is lean and every feature is a deliberate, documented switch.
The [RF Front End]({{ '/blog/series/rf-front-end/' | relative_url }}) series covers
the radio-layer diagnostics in depth, and the
[install-linux]({{ '/install-linux.html' | relative_url }}) guide has the udev/DVB-
blacklist steps `sdr doctor` points you toward.

## FAQ

**Is `sdr doctor` safe to run while the daemon is up?**
Yes — that's a design goal. It only inspects driver bindings and never opens or
claims a device, so it won't fight the running daemon for the USB claim. It's meant
to be a triage tool for a live-but-not-decoding daemon as much as a first-run
check.

**My dongle is plugged in but `sdr doctor` shows BAD — now what?**
Follow the `NEXT-STEP` hint. On Linux that's usually the DVB driver claiming the
device: blacklist `dvb_usb_rtl28xxu` and reload udev. On Windows it's the missing
WinUSB binding: run Zadig. The [install-linux]({{ '/install-linux.html' | relative_url }})
guide and [Hardening]({{ '/hardening.html' | relative_url }}) doc have the full
recipes.

**What's the difference between a preflight error and a warning?**
An error stops startup — unwritable directories and a broken TLS keypair, because
running past them means silent data loss or an insecure downgrade. A warning lets
the daemon run degraded — a missing talkgroup CSV or a storage-less paging setup —
and is pinned by the launcher/TUI so you see it.

**Why does preflight validate TLS when the server already checks it?**
The server's construction check catches the both-or-neither case (one path set,
the other empty). Preflight catches the file that exists but doesn't *parse* — a
typo in the PEM, a permissions problem — before the listener binds, so the error is
`preflight: tls …` and not an opaque failure after the port is open.

**I set a gain but the radio's deaf — is that a bug?**
Almost always a units mistake: `gain:` is in tenths of a dB, so `32` means 3.2 dB,
not 32. The daemon warns about exactly this (`did_you_mean 320`). Prefer
`gain: auto`, or run `gophertrunk sdr list` for the supported ladder — and check
`iq_clip_ratio` before raising gain, since a non-zero clip ratio means it's already
too *high*.

## Series navigation

**Part 7 of 14** · ←
[Part 6: The Diagnostics Reporter]({{ '/blog/deep-dives/running-it-for-real-06-diagnostics-reporter/' | relative_url }})
· Next →
[Part 8: The Opt-In Feature Matrix]({{ '/blog/deep-dives/running-it-for-real-08-opt-in-feature-matrix/' | relative_url }})
