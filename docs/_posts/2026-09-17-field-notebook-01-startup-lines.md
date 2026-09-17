---
title: "The Field Notebook, Part 1: Startup — The Lines Before the First Lock"
description: An operator's reading of GopherTrunk's startup log block — the two "gophertrunk starting" lines, device opened and gain set, and the config WARNs the daemon prints before its first lock, including the gain-in-tenths trap, two systems on one control tuner, and the pool gate that used to log nothing.
category: tutorials
keywords: gophertrunk startup log, sdr scanner won't lock, gain tenths of a db, rtl-sdr gain 50, two systems one sdr, config warnings daemon, sdr list probe, sdr doctor zadig, read debug log, gophertrunk field notebook
tags: [field-notebook, logs, startup, config, sdr, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 1
---

*Part 1 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field
measures, what a healthy rig prints, which number is a noise meter and which
one means traffic, and which deep dive to open when a line goes wrong.
[The Operator's Cookbook]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
built rigs and showed the lines that prove each one alive;
[From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
explained why those lines are designed as instruments. This series is the
reading guide between them, starting with the block between launch and the
first control-channel lock — where most "it doesn't decode" reports are
already decided.*

> **TL;DR:** A healthy startup prints, in order: `gophertrunk starting
> version=…` (`cmd/gophertrunk/main.go`), a `diagnostics banner=` block,
> one `device opened` + one `sdr: gain set` per dongle
> (`internal/sdr/pool.go`), `daemon: config summary` plus one `daemon:
> system config` per system, a second `gophertrunk starting` carrying
> `systems=` and `voice_devices=`, `api: listening`, then `ccdecoder:
> digital down-converter configured` as the hunt begins. The WARNs between
> them are **config verdicts, not RF**: `gain looks like dB, not
> tenths-of-dB` (`gain: 50` is 5 dB — `did_you_mean=500`), `N systems (…)
> share ONE control SDR` (time-multiplexed, never concurrent), and
> `channels:/voice_taps/tuner_strategy … are being IGNORED` off a wideband
> role. The dangerous case logs *nothing*: a source list missing from the
> pool-construction gate in `daemon.go` registers no driver, and the daemon
> starts quietly with no radio.

**Key takeaways**

- **Startup is a checklist the daemon reads aloud.** Each stage announces
  itself once, so a missing line localises the fault before you touch RF.
- **`gain:` is in tenths of a dB, and the log tells you.** `gain: 50`
  parses to 5.0 dB; the WARN prints `parsed_db=5 did_you_mean=500`.
- **Two systems on one `role: control` tuner never decode at once.** The
  hunter camps on the first lock and hunts the second only after losing it;
  the concurrent path is `role: wideband`.
- **Silence can be the symptom.** A source list left out of the pool gate
  registers no driver and logs nothing; a block with no `device opened` line
  is the tell.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Process banner | version, host and dongle diagnostics | `cmd/gophertrunk/main.go` (`gophertrunk starting`, `diagnostics`) |
| Dongle claim | driver, serial, role, rate, ppm | `internal/sdr/pool.go` (`device opened`, `sdr: gain set`) |
| Gain-unit trap | bare integer ≤ 50 tenths reads as dB | `cmd/gophertrunk/daemon.go` (`warnGainUnits`, `gainLooksLikeDBMistake`) |
| One control tuner, N systems | time-multiplexed hunt, WARN at startup | `daemon.go` (`cchuntSystems` block), `TestTwoSystemsOnOneControlSDRWarns` |
| Dead wideband keys | `channels:`/`voice_taps`/`tuner_strategy` off a wideband role | `daemon.go` (`being IGNORED` WARN) |
| The silent gate | every source list must be named or no driver registers | `daemon.go` (pool construction), `TestRemoteOnlySDRConfigsRegisterTheirDriver` |

## In this post

- **What this line family is telling you** — the startup block, stage by stage.
- **What healthy looks like** — a real block and its fields.
- **The WARNs that mean config, not RF** — gain units, shared tuners, dead keys.
- **The gate that logs nothing** — why an absent line is the loudest symptom.
- **When it doesn't look like that** — symptom → cause → fix.

## What this line family is telling you

Startup is a sequence of claims, each logged by the layer that made it — the
daemon knows what it opened, parsed and wired, not whether the radio will
lock. `main.go` logs `gophertrunk starting version=…` the moment
the logger exists, then `diagnostics banner=` — `version`, `os`, `host`
and the dongles the pool claimed, so a pasted log carries its own context
([Running It For Real Part 6]({{ '/blog/deep-dives/running-it-for-real-06-diagnostics-reporter/' | relative_url }})).
The pool prints one `device opened driver= serial= role= rate_hz= ppm=
bias_tee=` receipt per dongle — kept so an operator can grep whether the
`ppm` they typed landed on the serial driving the hunt (issue #264) — then
`sdr: gain set … gain_db=`, the *dB* figure the tenths value became.

`daemon: config summary` and one `daemon: system config name= protocol=
control_channels= enhancements=` per system restate the config **as the
receiver was built**, not as the empty YAML strings that mean "default".
`Daemon.Run` then logs the second `gophertrunk starting` with `http_addr=
grpc_addr= systems= voice_devices=`, the API binds (`api: listening addr=
tls=`), and the hunt begins with `ccdecoder: digital down-converter
configured sdr_rate_hz= pipeline_rate_hz= lo_offset_hz= autotune_hz=`.
Everything after that belongs to
[Part 2]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Timeline of GopherTrunk's six startup log stages with callouts marking where each startup WARN attaches, and a dashed pool-gate box before device opened where a missing source list produces no line at all.">
  <line x1="20" y1="70" x2="660" y2="70" stroke="var(--fg-muted)"/>
  <g fill="none" stroke="currentColor">
    <rect x="20" y="48" width="86" height="44" rx="4"/>
    <rect x="126" y="48" width="86" height="44" rx="4"/>
    <rect x="246" y="48" width="100" height="44" rx="4"/>
    <rect x="366" y="48" width="100" height="44" rx="4"/>
    <rect x="486" y="48" width="86" height="44" rx="4"/>
    <rect x="592" y="48" width="76" height="44" rx="4"/>
  </g>
  <text x="63" y="66" text-anchor="middle" fill="currentColor" font-size="9">gophertrunk</text>
  <text x="63" y="78" text-anchor="middle" fill="currentColor" font-size="9">starting version=</text>
  <text x="169" y="66" text-anchor="middle" fill="currentColor" font-size="9">diagnostics</text>
  <text x="169" y="78" text-anchor="middle" fill="currentColor" font-size="9">banner=</text>
  <text x="296" y="66" text-anchor="middle" fill="currentColor" font-size="9">device opened</text>
  <text x="296" y="78" text-anchor="middle" fill="currentColor" font-size="9">sdr: gain set</text>
  <text x="416" y="66" text-anchor="middle" fill="currentColor" font-size="9">daemon: config</text>
  <text x="416" y="78" text-anchor="middle" fill="currentColor" font-size="9">summary / system</text>
  <text x="529" y="66" text-anchor="middle" fill="currentColor" font-size="9">starting systems=</text>
  <text x="529" y="78" text-anchor="middle" fill="currentColor" font-size="9">voice_devices=</text>
  <text x="630" y="66" text-anchor="middle" fill="currentColor" font-size="9">api: listening</text>
  <text x="630" y="78" text-anchor="middle" fill="currentColor" font-size="9">DDC configured</text>
  <rect x="212" y="40" width="34" height="60" rx="3" fill="none" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="229" y="34" text-anchor="middle" fill="var(--accent)" font-size="9">pool gate</text>
  <line x1="229" y1="100" x2="229" y2="128" stroke="var(--accent)"/>
  <text x="229" y="140" text-anchor="middle" fill="var(--accent)" font-size="9">source list missing ⇒ no driver, NO line</text>
  <line x1="296" y1="92" x2="296" y2="160" stroke="var(--fg-muted)"/>
  <text x="296" y="172" text-anchor="middle" fill="currentColor" font-size="9">WRN gain looks like dB, not tenths-of-dB</text>
  <line x1="416" y1="92" x2="416" y2="200" stroke="var(--fg-muted)"/>
  <text x="416" y="212" text-anchor="middle" fill="currentColor" font-size="9">WRN N systems share ONE control SDR</text>
  <line x1="529" y1="92" x2="529" y2="230" stroke="var(--fg-muted)"/>
  <text x="529" y="242" text-anchor="middle" fill="currentColor" font-size="9">WRN no voice source</text>
</svg>
<figcaption>The startup block is a stage-by-stage receipt: every WARN hangs off the stage that produced it, and the one failure that produces no WARN sits in the pool gate.</figcaption>
</figure>

## What healthy looks like

A one-dongle P25 rig from Cookbook Part 1 at `log.level: info` starts like
this (long fields trimmed with `…`; every field name is real):

```
INF gophertrunk starting version=v1.1.5
INF diagnostics banner="GopherTrunk diagnostics\nversion : v1.1.5 …"
INF device opened driver=rtlsdr serial=00000001 role=wideband rate_hz=2400000 ppm=0 bias_tee=false
INF sdr: gain set serial=00000001 role=wideband gain_db=49.6
INF daemon: system config name=Metro-P25 protocol=p25 control_channels=2 enhancements=demod=c4fm …
INF gophertrunk starting http_addr=127.0.0.1:8080 systems=1 voice_devices=2
INF api: listening addr=127.0.0.1:8080 tls=false
INF ccdecoder: digital down-converter configured sdr_rate_hz=2400000 pipeline_rate_hz=48000 lo_offset_hz=0 autotune_hz=0
```

Read three things off it. **`role=wideband` with `voice_devices=2`** means
the virtual voice taps auto-enabled. **`gain_db=49.6`** is the tenths value
`"496"` in dB — if it matches the number you typed, you typed dB.
**`pipeline_rate_hz=48000`** is the per-protocol channel rate (48 kHz for the
4800-baud family, 144 kHz for TETRA), which is why decode is invariant to
your capture rate
([Weak-Signal Engineering Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})).

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `version=` | the binary actually running | the release you installed | stale — [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}) rung 0 |
| `device opened … serial= role= ppm=` | the pool's claim on one dongle | one per configured device | a device missing, or `configured SDR not present on the bus` |
| `sdr: gain set … gain_db=` | the applied gain in dB | 15–49.6 for a manual RTL-SDR | single digits, or no line |
| `system config … enhancements=` | what the receiver was built with | the flags you expect | `unrecognised …; falling back to` WARNs above it |
| `starting … voice_devices=` | voice sources the composer can bind | ≥ 1 per trunked system | `0` with `systems>0` — grants drop |
| `… pipeline_rate_hz=` | the DDC's output rate | 48000 or 144000 by protocol | absent — no hunt started |

## The WARNs that mean config, not RF

Every startup WARN in `daemon.go` is a verdict on the file, reached before a
single sample was demodulated. Fix these before touching the antenna.

**Gain in tenths.** `gain:` is tenths of a dB — `"320"` is 32 dB — because
that is the unit the tuner driver speaks. SDRTrunk, OP25 and gqrx take whole
dB, and the habit lands `gain: 32` in first-run configs.
`gainLooksLikeDBMistake` flags a bare integer at or below 50 tenths:

```
WRN daemon: gain looks like dB, not tenths-of-dB — radio may be effectively deaf serial=00000001 configured=50 parsed_db=5 did_you_mean=500 hint="gain: is in TENTHS of a dB (\"320\" = 32 dB). SDRTrunk/OP25/gqrx users multiply dB by 10. …"
```

That line was in a 13 Sep field log: a TETRA DMO rig ran `gain: 50`, its
control channel sat at −70 dBFS, and `gain: 500` was the first thing to try.
The sibling `warnLowGain` covers 51–149 tenths, and its hint adds that a
non-zero `iq_clip_ratio` means the radio is already too *hot* — see
[Analog Edge Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
and [SDR gain overload]({{ '/reference/sdr-gain-overload/' | relative_url }}).

**Two systems, one control tuner.** The `cchunt.Supervisor` is a time
multiplexer: it hunts systems round-robin and `parkUntilUnlocked` on the
first that locks — on a healthy site, forever. A 10 Sep report listed two
TETRA systems inside one 200 kHz span and saw only one ever decoding, with
no line saying why. Now the daemon says so once (wrapped for width):

```
WRN daemon: 2 systems (250_013, 250_208) share ONE control SDR: the single-tuner hunter decodes them one at a time and camps on the first that locks, so the others are hunted only after it loses lock. To decode them concurrently host every control channel on a `role: wideband` device with one `channels:` entry per system …
WRN daemon: sdr.devices[00000001] has role "auto" but sets channels:/voice_taps/tuner_strategy — those keys only apply to role: wideband and are being IGNORED …
```

The second line is the companion the same rig carried: a `channels:` plan on
a `role: auto` device that nothing read, because those keys are gated on
`Role == "wideband"`. `TestTwoSystemsOnOneControlSDRWarns` pins both; the
concurrent recipe is
[Cookbook Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}).

**No voice source.** `no voice source configured but trunking systems are
defined; voice grants will be dropped — no audio and no recordings` (issue
#379) fires once from `Daemon.Run` when `systems>0` and the voice pool is
empty — error-shaped, but also a legitimate monitor-only rig. Its cousin `trunking.systems configured but sdr.devices is empty` is
the pool gate's `else` branch, the setup for the next section.

## The gate that logs nothing

The most instructive startup failure prints no WARN. The SDR pool is built
inside one condition in `daemon.go`, and the network drivers are registered
*inside* that block:

```go
// cmd/gophertrunk/daemon.go (shape) — the pool-construction gate
// EVERY source list has to be named here: a source missing from the
// condition is never registered at all.
if len(cfg.SDR.Devices) > 0 || len(cfg.Baseband.Replay) > 0 ||
    len(cfg.SDR.RTLTCP) > 0 || len(cfg.SDR.SoapyRemote) > 0 ||
    len(cfg.SDR.Ka9qRadio) > 0 || len(cfg.SDR.Sidecar) > 0 {
    d.pool = sdr.NewPool(log) // … drivers registered, pool.OpenWith …
}
```

The gate once read `SDR.Devices || Baseband.Replay || SDR.RTLTCP`. A config
with only `sdr.soapy_remote` (or only `ka9q_radio`) registered no driver, and
the daemon came up quietly with no radio — no `device opened`, no pool error,
and not even the `sdr.devices is empty` WARN, because `SoapyRemote` was
non-empty and the `else` never ran. The unhealthy block's signature is what
is missing:

```
INF gophertrunk starting version=v1.1.5
INF diagnostics banner="GopherTrunk diagnostics\n…\ndongles : none detected"
INF gophertrunk starting http_addr=127.0.0.1:8080 systems=1 voice_devices=0
WRN no voice source configured but trunking systems are defined; voice grants will be dropped — … systems=1
```

No `device opened`. No `sdr: gain set`. No `ccdecoder: digital down-converter
configured`, because the hunt supervisor is only built when `d.pool != nil`.
`TestRemoteOnlySDRConfigsRegisterTheirDriver` walks each network-only config
through `NewDaemon` and asserts on driver **registration**, not on `d.pool`,
which is `nil` whenever the pool fails to open.

The reading rule behind this series is
[From the Issue Tracker Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})'s:
**a success-only log carries no information, so an absent line is
evidence.** Startup is where it applies without a counter, because the
healthy sequence is fixed.

## When it doesn't look like that

| Symptom | Likely cause | Fix / read |
|---|---|---|
| `sdr: gain set … gain_db=5` plus the `gain looks like dB` WARN | `gain: 50` meant 50 dB | `gain: "500"` or `auto`; [Cookbook 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}) explains the unit |
| `share ONE control SDR` WARN; one system never locks | two systems on a `role: control`/`auto` tuner are time-multiplexed | Both CCs on a `role: wideband` device, one `channels:` entry each — [Cookbook 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }}) |
| `configured SDR not present on the bus; check the cable / dmesg / lsusb` | serial typo, or wrong kernel driver bound | `gophertrunk sdr list` for the serial, `sdr doctor` for the binding — [Running It For Real 7]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }}) |

Two commands belong beside this table. `gophertrunk sdr list` prints
`DRIVER IDX SERIAL TUNER PRODUCT gains(0.1 dB)` — the ladder in the config's
own tenths unit. `gophertrunk sdr doctor` inspects the USB bus
and driver binding ([Zadig]({{ '/reference/zadig/' | relative_url }}) on
Windows); a dongle that appears then vanishes is
[USB recovery]({{ '/reference/rtlsdr-usb-recovery/' | relative_url }}) territory.

### How this line shapes operator practice

- **Paste the whole startup block into a report.** A reader diagnoses by
  what is missing; a trimmed excerpt hides it.
- **Read `gain_db=` before any dBFS figure.** A single-digit gain explains a
  −70 dBFS control channel completely; no equalizer or antenna will.
- **Treat every startup WARN as a config bug.** They are computed from the
  file alone; RF troubleshooting starts once the block prints clean.

## Where this goes next

Once the down-converter line prints, the hunt supervisor owns the log.
[Part 2]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }})
reads the lock family — `cc-hunt: trying`, `cc-hunt: locked`, the
per-protocol `cc locked` lines, `cchunt: hunt failed` with its `diagnosis`
field, and the `camped on conventional channel` idle state.

## FAQ

**Why does GopherTrunk log "gophertrunk starting" twice?**
Two layers announce themselves: `main.go` logs `gophertrunk starting
version=…` as soon as the logger exists, and `Daemon.Run` logs `gophertrunk
starting http_addr= systems= voice_devices=` once the pool, systems and voice
devices are built. The second line says what the daemon will do.

**What does the "gain looks like dB, not tenths-of-dB" warning mean?**
GopherTrunk's `gain:` value is tenths of a dB, so `gain: 50` is 5 dB and the
radio is nearly deaf. The WARN prints `parsed_db` and `did_you_mean` (the
value ×10). Set `gain: "500"` for 50 dB, or `gain: auto`, and check the
ladder with `sdr list`.

**Can two trunked systems share one control SDR in GopherTrunk?**
Only one at a time. A single `role: control` or `auto` tuner is hunted
round-robin and parks on the first system that locks; the second is hunted
only after that lock is lost, and the daemon warns at startup. For concurrent
decode, host every control channel on a `role: wideband` device, one
`channels:` entry per system.

**The daemon starts but never prints "device opened" — what happened?**
No SDR driver registered. Either the configured serial is not on the bus
(`configured SDR not present` follows), the pool failed to open (`SDR pool
failed to open`), or — on older builds — a network-only source such as
`soapy_remote` was missing from the pool gate and registered nothing
silently. Upgrade first, then check `sdr list`.

**Do startup warnings appear anywhere besides the log?**
Yes. Every `addWarning` message is collected into `startupWarnings`, printed
by the launcher as `!`-prefixed lines on stderr, and shown on the TUI
dashboard, so a headless operator sees the same config verdicts the log
carries; `preflight.go`'s warnings join the same list.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Lock Lines — Locked, Lost & Transitions]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }})
