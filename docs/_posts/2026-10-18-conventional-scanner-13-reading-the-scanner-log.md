---
title: "The Conventional Scanner, Part 13: Reading the Scanner's Log — Lines, Symptoms, and What Is Still Open"
description: "An operator's reading of every line the conventional scanner prints — the config-source and LO-offset lines at startup, synthetic call started and ended with its reason, the DCS gate and opposite-polarity WARNs, the stalled-stream and overload WARNs — plus the /api/v1/scanner snapshot fields and the honest list of what is not yet verified."
category: tutorials
keywords: gophertrunk scanner log, synthetic call ended reason, conv scanner lo offset tuning, dcs code heard with the opposite polarity, iq stream stalled during dwell, front end overloaded warn, api v1 scanner conventional, config source origin, scanner troubleshooting sdr, gophertrunk conventional scanner
tags: [conventional-scanner, logs, scanner, troubleshooting, dcs, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 13
---

*Part 13 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's non-trunked scan list — what you configure, what the code does
with it, what the log prints, and what is verified on air.
[Part 12]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }})
finished the rotation with priority interleave and frequency-keyed lockouts.
This part collects every line the scanner prints, in the
[Field Notebook]({{ '/blog/tutorials/field-notebook-14-field-card/' | relative_url }})
format — line, meaning, healthy, worry — then the `/api/v1/scanner`
snapshot, a symptom table, and what is still open.*

> **TL;DR:** The scanner's log has three layers. **Startup**: `config:
> source path= origin=` (or the `no config file found` / `multiple config
> files found` WARNs from `cmd/gophertrunk/main.go`), `conv: scanner LO
> offset tuning serial= lo_offset_hz= mode=`, `conv: restored persisted
> lockouts count=`, and the construction WARNs that each end in "every signal
> passes the gate" or "decoders disabled" — config verdicts, not RF.
> **Dwell**: `synthetic call started device= grant=` and `synthetic call
> ended device= reason=` (`internal/trunking/engine.go`; reasons `normal`,
> `error`, `lockout`, `manual`), `conv: DCS gate opened … nrz_inverted=`,
> and the per-channel WARNs — `DCS code heard with the opposite polarity`,
> `IQ stream stalled during dwell … no_samples_for=`, `front end overloaded
> … clipped_fraction=`. **Snapshot**: `GET /api/v1/scanner` →
> `conventional.state` (`scanning` / `dwell` / `held`) and one row per
> channel with `active`, `locked_out`, `last_break_at`. Still open: a
> noise-quieting FM squelch (#1239), the #1184 two-dongle USB stall, and the
> ACARS and FleetSync scan-list live runs.

**Key takeaways**

- **Most "it doesn't open" reports are decided before the first dwell.** A
  construction WARN ending in "every signal passes the gate" means a zero
  sample rate reached the scanner — config, not RF.
- **`reason=` is the dwell's verdict.** `normal` is hangtime, `error` a
  stream that closed or stalled, `lockout` and `manual` operator acts; a run
  of `error` points at the pump, not the channel.
- **The opposite-polarity WARN is the DCS answer key.** It names the code,
  the configured polarity and the one heard, once per channel.
- **The overload WARN names the fix.** `clipped_fraction` over 0.002 means
  reduce gain — never raise it.

## Cheat sheet

| Line / key | Meaning | Healthy / worry when |
|---|---|---|
| `config: source path= origin=` | which file backs the daemon | present once; worry at `no config file found` or `multiple config files found` |
| `conv: scanner LO offset tuning` | `lo_offset_hz`, `mode=auto|configured|…` | `mode=auto` with a non-zero offset; worry at `no room for an LO offset` |
| `conv: restored persisted lockouts count=` | lockout memory applied before the first pick | matches what you locked; absent is fine with no lockouts or no `storage.path` |
| `synthetic call started/ended` | the dwell as the engine sees it | `reason=normal`; worry at repeated `error` |
| `conv: DCS gate opened` | the gate matched, with `nrz_inverted` | one per keyup; worry if never, with carrier present |
| `conv: IQ stream stalled during dwell` | no chunks for `StreamStallTimeout` (3 s) | never; worry at every dwell — the pump |
| `conv: front end overloaded` | > 0.2 % of raw samples on the ADC rail | never; worry at any — lower the gain |
| `GET /api/v1/scanner` | `conventional.state`, per-channel `active` / `locked_out` | `scanning` ↔ `dwell` cycling; worry at `held` you did not set |

## In this post

- **Startup** — config source, the LO offset line, lockouts, and the construction WARNs.
- **The dwell** — call started and ended, the reasons, the DCS lines.
- **The channel WARNs** — stalled stream, tune and stream failures, gain, overload.
- **The snapshot and the panels** — `/api/v1/scanner`, TUI keys, web columns.
- **Symptom table and what is still open** — the honest list.

## Startup

Before anything tunes, `main.go` says which file it read. After an update
that changes discovery precedence this is the first line to check — a daemon
silently running built-in defaults, or a different file than intended,
explains missing squelch, gain or recording settings with no other symptom
(the #1184 lesson, and the
[#836](https://github.com/MattCheramie/GopherTrunk/issues/836) fix that
added the lowercase `~/.config/gophertrunk` path to `candidateDirs`):

```text
INF config: source path=/home/op/.config/gophertrunk/config.yaml origin=discovered
WRN config: no config file found — running built-in defaults hint="pass -config, set GOPHERTRUNK_CONFIG, or place config.yaml in a discovered directory (gophertrunk doctor lists them)"
WRN config: multiple config files found; using the one above and ignoring the others using=… ignored=[…]
```

The daemon then decides whether a conventional scanner exists at all — with
`scanner.conventional` listed, `manual_tune_enabled: true`, or two or more
`role: voice` SDRs (`daemon: scanner: auto-enabling manual tune (spare Voice
SDR detected) voice_sdrs=2`) it takes the **last** voice SDR; with none it
prints `daemon: scanner.conventional / manual_tune_enabled configured but no
Voice SDRs in the pool; skipping`. Next comes the offset line
([Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})):

```text
INF conv: scanner LO offset tuning serial=00000002 lo_offset_hz=… mode=auto sample_rate_hz=2400000
```

`mode` is one of `auto`, `configured`, `disabled (scanner.lo_offset_hz < 0)`,
`no room for an LO offset at this sample rate` or `scanner.lo_offset_hz
exceeds 35% of the sample rate` (`convScannerLOOffsetHz`); the last two mean
the scanner is tuning on-channel, where a clipped front end whistles at 4×
the carrier offset. Then `conv: restored persisted lockouts count=N` if the
store had rows, and the construction-time WARNs from `New`:

| WARN (prefix `conv:`) | Cause |
|---|---|
| `tone gating configured but scanner sample rate is zero; tone gate disabled — every signal passes the gate` | `SampleRateHz` 0 reached the scanner |
| `CTCSS detector failed to initialise; tone gate disabled — every signal passes the gate` (DCS twin) | constructor refused the inputs |
| `scanner sample rate is zero; squelch_dbfs measures the whole SDR span instead of each channel` | the #1239 meter cannot be built |
| `AM channel but the scanner sample rate is zero; using the squelch_dbfs power squelch instead of carrier-to-noise` | same, for an AM channel |
| `data decoders configured but the scanner has no decoder factory or sample rate; decoders disabled` / `data decoder failed to initialise; channel scans without it` | #1220 decoders without a bus or rate, or a constructor error |
| `channel gain looks like whole dB, but gain is tenths of a dB — "28" is 2.8 dB; write "280" or "28.0" for 28 dB` / `ignoring unparseable channel gain` | per-channel `gain` (#1239) in the wrong unit |

Every one of these is a config verdict. The daemon passes `sdr.sample_rate`
through as `SampleRateHz`, so if the zero-rate family prints on a real rig,
the config that reached the daemon is not the one you edited — go back to the
`config: source` line.

## The dwell

A dwell is bracketed by two engine lines. `beginDwell` synthesises a
`trunking.Grant` (protocol `fm-conv` or `am-conv`, `GroupID` =
`talkgroup_id` or `0x80000000 | index`) and `HandleSyntheticCall` logs it;
`endDwell` hands the engine an `EndReason` and `EndSyntheticCall` logs that:

```text
INF synthetic call started device=00000002 grant=…
INF synthetic call ended device=00000002 reason=normal
INF recorder: call ended device=00000002 wav=… duration=… reason=normal
```

The reasons the scanner can produce are four. `normal` — `Hangtime` of
debounced silence elapsed. `error` — the stream channel closed, the stream
context ended, or the stall watchdog fired. `lockout` — the operator locked
out the channel mid-dwell. `manual` — a temporary (VFO) channel was removed
while dwelling. The `recorder: call ended` line that follows carries the same `reason` from
the recorder's side
([Field Notebook Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
reads that family). The dwell prints nothing per chunk — squelch state goes
to the composer through `SquelchOpen`, not the log.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A timeline of one conventional dwell with the log lines placed where they occur. The scan window opens after a tune; when the in-channel power clears the squelch and, for a tone-gated channel, the DCS gate opened line prints, the engine logs synthetic call started. Activity continues through the dwell; a short blip inside the activity debounce window does not reset the hangtime countdown. After hangtime of debounced silence the engine logs synthetic call ended with reason normal, followed by the recorder's call ended line. Below, three alternative exits are marked: stalled stream after three seconds giving reason error, lockout giving reason lockout, and a removed VFO channel giving reason manual.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one dwell, and where each line prints</text>
  <line x1="30" y1="70" x2="650" y2="70" stroke="var(--fg-muted)"/>
  <rect x="30" y="56" width="90" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="75" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="8">scanWindow</text>
  <rect x="120" y="50" width="330" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="285" y="66" text-anchor="middle" fill="currentColor" font-size="9">beginDwell: carrier (+ tone) active</text>
  <text x="285" y="81" text-anchor="middle" fill="var(--fg-muted)" font-size="8">a blip shorter than ActivityDebounce never resets the countdown</text>
  <rect x="450" y="56" width="120" height="28" fill="none" stroke="var(--accent)"/>
  <text x="510" y="74" text-anchor="middle" fill="var(--accent)" font-size="8">hangtime countdown</text>
  <line x1="120" y1="90" x2="120" y2="118" stroke="currentColor"/>
  <text x="120" y="130" text-anchor="middle" fill="currentColor" font-size="8">conv: DCS gate opened (if gated)</text>
  <text x="120" y="142" text-anchor="middle" fill="currentColor" font-size="8" font-weight="bold">synthetic call started</text>
  <line x1="570" y1="90" x2="570" y2="118" stroke="var(--accent)"/>
  <text x="570" y="130" text-anchor="middle" fill="var(--accent)" font-size="8" font-weight="bold">synthetic call ended reason=normal</text>
  <text x="570" y="142" text-anchor="middle" fill="var(--accent)" font-size="8">recorder: call ended reason=normal</text>
  <line x1="30" y1="165" x2="650" y2="165" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="30" y="182" fill="var(--fg-muted)" font-size="8">other exits:</text>
  <text x="130" y="182" fill="currentColor" font-size="8">no chunks for 3 s → "IQ stream stalled" WARN → reason=error</text>
  <text x="130" y="198" fill="currentColor" font-size="8">LockoutChannel → reason=lockout · RemoveTemporaryChannel → reason=manual · stream closed → reason=error</text>
</svg>
<figcaption>The dwell prints at its two edges only; everything between them is state the snapshot and the composer read, not the log.</figcaption>
</figure>

A DCS-gated channel adds two lines of its own
([Part 6]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }})).
When the gate matches inside `scanWindow`:

```text
INF conv: DCS gate opened freq_hz=154190000 label="Fire Dispatch" dcs_code=023 nrz_inverted=false
```

`nrz_inverted` is the sense the detector matched in — the field that pinned
the on-air polarity on 24 Sep (D025N → `false`, D025I → `true`). And when the
code is heard in the sense the gate does **not** accept, once per channel
(`dcsPolarityWarned`), the scanner says exactly what to change:

```text
WRN conv: DCS code heard with the opposite polarity; gate kept shut — set tone.dcs_polarity to match the radio (N = normal, I = inverted) or both freq_hz=… label=… index=… dcs_code=023 dcs_polarity=normal heard=inverted
```

The gate stays shut by design; `tone.dcs_polarity: both` opens on either
sense (and, as the config comment notes, `023` "both" also opens on 047N).

## The channel WARNs

Four WARNs can print during a dwell or a tune, and each names its own fix.

**`conv: IQ stream stalled during dwell — ending call and re-opening the
stream freq_hz= label= no_samples_for=`.** The 250 ms ticker in `beginDwell`
compares the last chunk's arrival to `StreamStallTimeout`
(`DefaultStreamStallTimeout` = 3 s). Hangtime counts silent chunks that never
arrive, so before #1184 a pump that stopped without closing held a phantom
call open — the reporter's 23 s synthetic call on a silent 146.67 MHz
channel. Now it ends `reason=error` and `Run` re-opens the stream
(`TestConvScannerStalledStreamEndsDwell`).

**`conv: tune failed freq_hz= err=`** and **`conv: StreamIQ failed err=`**
come from `Run`; both sleep briefly and retry the next pick. A steady stream
of them on one dongle is the USB stall class under "still open".

**`conv: per-channel gain write failed; scanning at the device's current
gain freq_hz= label= gain_tenth_db= err=`** prints once per distinct gain
value (`gainWarned`), and the scan carries on at whatever the device holds
([Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})).

**`conv: front end overloaded — IQ pinned to the ADC rail … Reduce gain or
add attenuation (do NOT raise gain). issue #1184 serial= channel_hz=
clipped_fraction= lo_offset_hz=`** comes from
`convScannerFrontEnd.observeRaw`: the *raw* pre-mix samples over
`convClipWindow` (1 s), warning when the rail-pinned fraction exceeds
`convClipWarnFrac` = 0.002, at most once per `convClipWarnGap` (5 min). The
offset mix hides the 4δ whistle; a clipped front end still loses weak
signals.

## The snapshot and the panels

`GET /api/v1/scanner` returns the unified cockpit view; its `conventional`
object is `ConvScannerStatusDTO` (`internal/api/server.go`):

```json
"conventional": {
  "enabled": true,
  "state": "scanning",
  "device_serial": "00000002",
  "cursor_index": 3,
  "channels": [
    {"index": 0, "label": "Sheriff Repeater", "frequency_hz": 155895000,
     "mode": "nfm", "active": false, "locked_out": false,
     "last_break_at": "2026-10-18T14:02:11Z"}
  ]
}
```

`state` is `scanning`, `dwell` or `held` (`conventional.State`); `active` is
true on exactly the row whose index is `dwellIndex`; `last_break_at` is the
last time that channel broke squelch; `locked_out` is the runtime flag from
[Part 12]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }}).
Not in the DTO: the per-channel `decoders` list the scanner's own
`Snapshot()` carries, and any live power reading — squelch state goes to the
composer, not the API. The top-level `hold` and `avoids` are the engine's
talkgroup controls, which never apply to a conventional call. The TUI
Scanner panel renders the same rows with `h`, `Enter`, `L`, `f` and `m`
(the [TUI reference]({{ '/tui.html' | relative_url }}) lists each key's
route); the web Scanner panel shows `#`, Label, Freq, Mode and State with
Dwell and Lock/Unlock buttons.

## Symptom table and what is still open

| Symptom | First suspect | Where to read |
|---|---|---|
| Nothing ever dwells; no `synthetic call started` | squelch too high for the in-channel level, or the wrong config file | `config: source`; `squelch_dbfs` vs the channel ([Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})) |
| Every dwell ends `reason=error` | the IQ pump | the `IQ stream stalled` WARN's `no_samples_for`; `tune failed` / `StreamIQ failed` |
| Carrier present, tone-gated channel never opens | polarity (DCS) or rate (CTCSS) | `DCS code heard with the opposite polarity`; any "every signal passes the gate" WARN |
| Audio whistles at a fixed pitch | ADC clipping on-channel | `front end overloaded` → lower gain; `conv: scanner LO offset tuning mode=` |
| A locked-out channel still plays after restart | no `storage.path` | absence of `restored persisted lockouts` |
| Data bursts never appear | decoder never built, or no storage | `data decoders configured but…`; the preflight `storage.path is empty` warning |

What the log cannot yet settle:

- **A noise-quieting FM squelch is not built.** `squelch_dbfs` measures the
  channel, not the span (#1239), but it is still absolute dBFS and moves with
  gain; the gain-independent answer — a discriminator-variance gate like
  `dmrrx.carrierGate` — is design only.
- **DCS is on-air verified** (24 Sep, D025N/D025I on Kenwood NX-300/NX-5000
  with a service monitor); the remaining caveat is inherent aliasing (a
  normal 023 gate opens on 047I).
- **The #1184 two-dongle stall is not root-caused.** With `fleetsync.channels`
  on one RTL-SDR and `scanner.conventional` on another, both stalled 146 ms
  after start (an r82xx PLL I2C write returning EPIPE, "tried chunk sizes
  16,8,4; all stalled"); `runSingleChannelDecoder` now retries
  (`fleetsync: SetCenterFreq failed — retrying`, then `SetCenterFreq
  recovered`) and the stall watchdog ends the phantom dwell, but *why* both
  dongles stalled at once is open — the `device opened` lines and whether
  the two share a USB hub are the next instrument.
- **ACARS has no live run**
  ([Part 10]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})),
  FleetSync on a scan-list channel is capture-pinned, not live-verified
  ([Part 11]({{ '/blog/tutorials/conventional-scanner-11-mdc1200-fleetsync-on-scan-channels/' | relative_url }})),
  and the LO offset fix is pinned by `conv_offset_test.go` with no recorded
  on-air A/B on the reporter's clipped rig.

## Where this goes next

Thirteen parts of mechanism deserve one page you can copy.
[Part 14]({{ '/blog/tutorials/conventional-scanner-14-annotated-config/' | relative_url }})
is the complete annotated `scanner.conventional` config — every key checked
against `config.example.yaml` and the Go struct tags — with a startup
checklist built from the lines above, and the series wrap-up.

## FAQ

**What does `synthetic call ended reason=error` mean on a conventional channel?**
The dwell ended because the IQ stream closed, its context ended, or no
chunks arrived for `StreamStallTimeout` (3 s) — the `conv: IQ stream stalled
during dwell` WARN prints first in that case. It is a pump verdict, not a
signal one; one line is a hiccup, one per dwell is a dead stream.

**Why does the log say "every signal passes the gate"?**
A tone gate was configured but the detector could not be built — a zero
`SampleRateHz` or a constructor that refused the inputs — so the channel
opens on power alone. On a real rig check the `config: source` line first:
the config that reached the daemon is probably not the one you edited.

**How do I know which DCS polarity my radio sends?**
Read `conv: DCS gate opened … nrz_inverted=` on a match, or the
`DCS code heard with the opposite polarity` WARN on a mismatch, which prints
`dcs_polarity` (configured) and `heard` (received) once per channel. On the
24 Sep Kenwood run D025N read `false` and D025I `true`; set
`tone.dcs_polarity` to match, or `both`.

**What does the "front end overloaded" WARN want me to do?**
Lower the gain or add attenuation — never raise gain. It fires when more than
0.2 % of a one-second window of raw samples sits on the ADC rail
(`convClipWarnFrac`), at most every 5 minutes. The LO offset keeps the 4δ
whistle out of the audio; a clipped front end still loses weak signals.

**Where do I see the scanner's state without the log?**
`GET /api/v1/scanner` → `conventional`: `state` (`scanning` / `dwell` /
`held`), `cursor_index`, and per channel `active`, `locked_out` and
`last_break_at`. The TUI and web Scanner panels render that snapshot;
neither carries a live power reading or the per-channel decoder list.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Priority Interleave and Persistent Lockouts — Keyed by Frequency, Never List Index]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }})
· Next →
[Part 14: The Complete Annotated Conventional Config — Verified Key by Key]({{ '/blog/tutorials/conventional-scanner-14-annotated-config/' | relative_url }})
