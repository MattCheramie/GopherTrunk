---
title: "The Conventional Scanner, Part 1: What 'Conventional' Means to a Trunking Scanner — The Dwell Loop"
description: "How GopherTrunk's conventional scanner turns a fixed-frequency scan list into calls a trunking engine can record: the scanning / dwell / held states, the 100 ms scan window, the synthetic grant and its 0x80000000-or-talkgroup_id identity, the hangtime countdown with its 3 dB hysteresis and 50 ms debounce, and the 3 s stream-stall guard from issue 1184."
category: tutorials
keywords: conventional scanner sdr, analog fm scan list, scanner dwell loop, hangtime squelch scanner, synthetic call trunking engine, scanner.conventional config, talkgroup_id conventional channel, stream stall timeout, rtl-sdr fm scanner go, gophertrunk conventional scanner
tags: [conventional-scanner, analog-fm, scanning, squelch, hangtime, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 1
---

*Part 1 of **The Conventional Scanner**, a 14-part operator's tutorial on the
half of GopherTrunk that has no control channel: `scanner.conventional`. It
stands on three earlier series — [The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }})
for what the samples mean, [Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }})
for the data bursts that ride analog channels, and the
[Cookbook's analog FM recipe]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
for a working config — and reads the log the way the
[Field Notebook]({{ '/blog/series/field-notebook/' | relative_url }}) does.
This part is the frame: what a trunking engine does with a channel that
never announces itself, and what one visit to one channel looks like inside
`internal/scanner/conventional`.*

> **TL;DR:** A trunked system tells GopherTrunk where the voice is; a
> conventional channel does not, so `conventional.Scanner.Run` visits each
> scan-list entry in turn, opens the SDR's IQ stream, and watches the
> channel's power for `MinDwellPerChannel` (100 ms default). When the squelch
> opens — and, on a tone-gated channel, the CTCSS/DCS detector agrees — it
> publishes a **synthetic grant** through `Engine.HandleSyntheticCall`
> (protocol `fm-conv` or `am-conv`, `GroupID` = `talkgroup_id` or
> `0x80000000 | index`) and the recorder treats the dwell like any other call.
> The dwell ends when power stays below `squelch_dbfs − 3 dB`
> (`SquelchHysteresisDb`) for `hangtime_ms` (1500 default); a blip shorter
> than `ActivityDebounce` (50 ms) cannot restart that countdown
> ([#1090](https://github.com/MattCheramie/GopherTrunk/issues/1090)). A
> stream that delivers no chunk for `StreamStallTimeout` (3 s) ends the call
> with `reason=error` — the phantom 23 s call of
> [#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184). States:
> `scanning`, `dwell`, `held`, read back at `GET /api/v1/scanner`.

**Key takeaways**

- **The engine never learns anything from the carrier.** The scanner
  manufactures the grant itself; `HandleSyntheticCall` and
  `EndSyntheticCall` are the only entry points, and everything downstream
  sees an ordinary call.
- **A channel visit is a timer and a comparison.** 100 ms minimum per
  channel, one in-channel power reading per IQ chunk, one threshold. Tone
  gating and data decoders are extra conditions on the same chunk.
- **Hangtime is debounced on both edges.** The squelch opens at
  `squelch_dbfs`, counts down only once power drops 3 dB below it, and a
  single above-threshold chunk cannot reset the countdown.
- **The call's identity is a config decision, not a list position.** An
  unset `talkgroup_id` means `0x80000000 | index`, which shifts whenever the
  list is edited ([#1105](https://github.com/MattCheramie/GopherTrunk/issues/1105)).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The loop | pick → tune → open stream → scan window → dwell | `internal/scanner/conventional/scanner.go` (`Run`) |
| Scan window | wait up to `MinDwellPerChannel` (100 ms) for the squelch to open | `scanWindow`, raised by `ctcssMinDwell` 350 ms / `dcsMinDwell` 600 ms / `amMinDwell` 100 ms |
| The grant | `Protocol` `fm-conv`/`am-conv`, `GroupID`, `GroupLabel`, `System: "scanner"` | `beginDwell`, `conventionalProtocol`, `trunking.Engine.HandleSyntheticCall` |
| End of call | hangtime after power < open − `SquelchHysteresisDb`, debounced by `ActivityDebounce` | `beginDwell`, `TestConvScannerBriefBlipsDoNotHoldSquelchOpen` |
| Stalled stream | no chunk for `StreamStallTimeout` (3 s) → `EndReasonError`, re-open | `DefaultStreamStallTimeout`, `TestConvScannerStalledStreamEndsDwell` |
| Operator state | `scanning` / `dwell` / `held`, cursor, per-channel `active` / `locked_out` | `Snapshot`, `GET /api/v1/scanner` (`ConvScannerStatusDTO`) |

## In this post

- **Why a trunking engine carries a scanner** — the synthetic call and what the engine does with it.
- **One channel visit, state by state** — `Run`, `pickNextChannel`, the three states.
- **The scan window** — `MinDwellPerChannel`, chunk size, and the tone-driven minimums.
- **From dwell to call** — the grant, the hangtime countdown, hysteresis and debounce.
- **When the stream stops** — the stall guard that issue #1184 added.

## Why a trunking engine carries a scanner

Everything else in GopherTrunk starts with a control channel: a P25, DMR
Tier III or TETRA site broadcasts a grant naming a talkgroup, a source and a
frequency, and the engine turns it into a voice-device allocation, a
recording and a call-log row. A fire dispatch frequency, a marine VHF channel
or a GMRS repeater offers none of that — a carrier appears and vanishes on a
fixed frequency, and the only way to follow it is to go and look.

The conventional scanner is the part that goes and looks. The package
comment on `squelch.go` states the design in one line: a state machine
cycles through operator-configured channels, measures power on each
tune-and-dwell, and on squelch break hands off to the trunking engine via
`Engine.HandleSyntheticCall` so the recorder writes audio like any other
call. The engine side is small. `HandleSyntheticCall` builds an `ActiveCall`
keyed by the scanner's device serial, keeps it in the engine's `synthetic`
map rather than the voice pool, publishes `KindCallStart` on the bus and logs
`synthetic call started device=… grant=…`. `EndSyntheticCall` publishes
`KindCallEnd` with the reason and logs `synthetic call ended`.

What the scanner does **not** do is produce audio. `beginDwell` keeps
reading IQ for carrier-drop detection and runs no demodulator. Audio comes
from the voice composer, which subscribes to the same `KindCallStart`,
classifies the protocol (`fm-conv` → `voiceKindFM`, `am-conv` →
`voiceKindAM` in `classifyVoiceKind`) and runs its analog chain on a
*subscription* to the scanner's IQ broker — the fan-out
[issue #1075's postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
introduced after the chain tried to open a second stream on a device the
scanner already held. The scanner is the broker's primary `StreamIQ`
consumer; `convScanVoiceSource` hands the composer a `broker.Subscribe()`
copy of the same chunks.

One consequence from the same block: the scanner needs a `role: voice` SDR
of its own and takes the **last** one in the pool. With channels configured
and no voice device, the daemon logs
`daemon: scanner.conventional / manual_tune_enabled configured but no Voice SDRs in the pool; skipping`
and the scan list is absent from everything that follows.

## One channel visit, state by state

`Run` is a loop with no hidden goroutines of its own. Each iteration asks
`pickNextChannel` for an index and a copy of the channel, applies any
per-channel gain (`applyChannelGain`, Part 4), tunes with
`Tuner.SetCenterFreq`, opens a fresh IQ stream under a per-dwell context,
resets the channel's detectors and meters so nothing from the previous
frequency leaks into the first reading, and runs the scan window. No break
means cancel the stream and advance; a break means `beginDwell` holds the
channel until the call ends.

```go
// internal/scanner/conventional/scanner.go (shape) — Run
idx, ch, ok := s.pickNextChannel()
s.applyChannelGain(ch)
if err := s.opts.Tuner.SetCenterFreq(ch.FrequencyHz); err != nil {
    s.log.Warn("conv: tune failed", "freq_hz", ch.FrequencyHz, "err", err)
    s.sleep(ctx, 100*time.Millisecond)
    continue
}
streamCtx, cancel := context.WithCancel(ctx)
stream, err := s.opts.IQ.StreamIQ(streamCtx)
if det := s.detectorFor(idx); det != nil { det.Reset() }
if pm := s.powerMeterFor(idx); pm != nil { pm.reset() }
broken := s.scanWindow(ctx, idx, ch, stream)
if !broken { cancel(); continue }
s.beginDwell(idx, ch, stream, streamCtx, cancel)
```

Three states are visible from outside. `StateScanning` is the loop above.
`StateDwell` is set at the top of `beginDwell` with `dwellIndex` pointing at
the channel and cleared by `endDwell`. `StateHeld` is an operator action:
`Hold()` pins the scanner, and `Run` checks `IsHeld()` every iteration and
idles in 100 ms ticks until `Resume()`. `DwellOn(idx)` sets
`forcedDwellIndex`, which `pickNextChannel` honours ahead of the cursor and
ahead of a lockout (an explicit "listen now"). Lockouts and priority interleave are Part 12's subject; for
now, `pickNextChannel` walks at most `n` steps for an unlocked channel and
returns `ok=false` when there is none, at which point `Run` idles.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Timeline of one channel visit: tune, a 100 millisecond scan window, squelch break and synthetic call start, a dwell held while power stays within 3 decibels of the open level, a 1500 millisecond hangtime countdown that ignores a blip shorter than 50 milliseconds, then the call ends with reason normal and the scanner advances.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one channel visit: scan window → dwell → hangtime → advance</text>
  <line x1="30" y1="70" x2="650" y2="70" stroke="var(--fg-muted)"/>
  <rect x="30" y="40" width="40" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="50" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tune +</text>
  <text x="50" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">StreamIQ</text>
  <rect x="70" y="40" width="90" height="60" fill="none" stroke="currentColor"/>
  <text x="115" y="60" text-anchor="middle" fill="currentColor" font-size="8">scanWindow</text>
  <text x="115" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">≥ MinDwellPerChannel</text>
  <text x="115" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">100 ms</text>
  <line x1="160" y1="30" x2="160" y2="110" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="160" y="26" text-anchor="middle" fill="var(--accent)" font-size="8">power ≥ squelch_dbfs → HandleSyntheticCall</text>
  <rect x="160" y="40" width="250" height="60" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="285" y="60" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">dwell · StateDwell</text>
  <text x="285" y="76" text-anchor="middle" fill="currentColor" font-size="8">active while power ≥ open − SquelchHysteresisDb (3 dB)</text>
  <rect x="410" y="40" width="190" height="60" fill="none" stroke="currentColor" stroke-dasharray="4 2"/>
  <text x="505" y="60" text-anchor="middle" fill="currentColor" font-size="9">hangtime countdown</text>
  <text x="505" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="8">belowSince … 1500 ms</text>
  <rect x="470" y="86" width="4" height="10" fill="var(--accent)"/>
  <text x="472" y="126" text-anchor="middle" fill="var(--accent)" font-size="8">blip &lt; ActivityDebounce 50 ms: ignored</text>
  <line x1="600" y1="30" x2="600" y2="110" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="600" y="26" text-anchor="middle" fill="currentColor" font-size="8">EndSyntheticCall reason=normal</text>
  <rect x="600" y="40" width="50" height="60" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="625" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">next</text>
  <text x="30" y="160" fill="var(--fg-muted)" font-size="8">squelchState for the composer:</text>
  <rect x="160" y="168" width="250" height="12" fill="none" stroke="var(--accent)"/>
  <text x="285" y="177" text-anchor="middle" fill="var(--accent)" font-size="8">open</text>
  <rect x="410" y="168" width="190" height="12" fill="none" stroke="var(--fg-muted)"/>
  <text x="505" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="8">closed</text>
  <text x="30" y="210" fill="var(--fg-muted)" font-size="8">no chunk for StreamStallTimeout (3 s) → reason=error, stream re-opened</text>
</svg>
<figcaption>A visit is a 100 ms look, then — if the squelch opens — a dwell that ends only after a full hangtime of quiet. The published squelch state follows the same debounced decision.</figcaption>
</figure>

## The scan window

`scanWindow` arms one timer for `MinDwellPerChannel` and reads chunks until
either the timer fires (return `false`, advance) or a chunk clears the
squelch (return `true`, dwell). Each chunk goes through `squelchMeasure`,
which returns the channel's meter and threshold — the in-channel power meter
and `squelch_dbfs` for FM, the carrier-to-noise meter and `squelch_cn_db`
for AM. Parts 4 and 8 cover the meters; here, **the comparison is per chunk and
the window is per channel**. The chunk is whatever the stream delivers
(`Options.DwellChunkLen` documents the 4096-sample reference, ≈1.7 ms at
2.4 MS/s, that the 50 ms debounce was sized against).

The 100 ms minimum is a floor. `New` raises it when any channel needs longer
to decide: `ctcssMinDwell` (350 ms) covers the CTCSS detector's 250 ms
Goertzel block plus chunk margin — without it the scanner advanced before
the detector ever reported and a tone-gated channel could never open;
`dcsMinDwell` (600 ms) covers a DCS codeword plus confirmation;
`amMinDwell` (100 ms) covers the AM meter's first Welch average. The bump is
scanner-wide, so one DCS channel makes every window 600 ms. A tone-gated
window also requires `det.Process(iq)` to report the tone — power alone is
not "right system" — and a data decoder mid-burst when the timer fires
extends the window in `dataScanHoldStep` steps (Part 11).

## From dwell to call

`beginDwell` publishes the open squelch state, marks `StateDwell`,
synthesises the grant, runs the hangtime loop, and on every exit path calls
`endDwell`, which ends the synthetic call with a reason and returns the
scanner to `StateScanning`.

```go
// internal/scanner/conventional/scanner.go (shape) — beginDwell
gid := ch.TalkgroupID
if gid == 0 {
    gid = uint32(0x80000000) | uint32(idx)
}
g := trunking.Grant{
    System:      s.opts.SystemName,       // "scanner" in the daemon
    Protocol:    conventionalProtocol(ch), // "fm-conv" | "am-conv"
    GroupID:     gid,
    GroupLabel:  ch.Label,
    SourceID:    0,
    FrequencyHz: ch.FrequencyHz,
    At:          now,
}
s.opts.Engine.HandleSyntheticCall(g, s.opts.DeviceSerial)
```

The identity line matters. Bit 31 keeps a positional ID clear of any real
trunked group, but the positional ID is the **list index**, so inserting a
channel above an existing one renames every call that channel has logged.
`talkgroup_id` pins it
([#1105](https://github.com/MattCheramie/GopherTrunk/issues/1105)), and
`validateScanner` rejects two channels whose *effective* IDs collide.
`TestConvScannerUsesExplicitTalkgroupID` pins the override;
`TestConvScannerBreaksSquelchAndEndsOnHangtime` pins the default
(`0x80000001` for the second channel) and the `fm-conv` protocol.

The hangtime loop reads chunks against a different threshold from the one
that opened the squelch: `keepAlive := open − ch.SquelchHysteresisDb` (3 dB
by default). A chunk counts as *inactive* only once power falls 3 dB below
the opening level, so a signal sitting on the threshold cannot flicker the
countdown. On a tone-gated channel an active chunk also needs
`det.Process(iq)` true — a transmitter that drops its CTCSS hangs up like a
carrier drop. Two timestamps do the accounting: `belowSince` marks the start
of the current below-threshold run, and `aboveSince` is the debounce
accumulator. Activity resets the countdown only when it has been continuous
for `ActivityDebounce` or when no countdown is running; a lone
above-threshold chunk sets `aboveSince`, the next inactive chunk clears it,
and `belowSince` survives. That is the
[#1090](https://github.com/MattCheramie/GopherTrunk/issues/1090) fix — before
it, any single above-threshold chunk zeroed the countdown and a channel
peppered with ~2 ms blips never released.
`TestConvScannerBriefBlipsDoNotHoldSquelchOpen` feeds one loud chunk in
thirty at a 2 ms cadence against a 300 ms hangtime and demands
`EndReasonNormal`.

The same debounced decision is published as `squelchState` and read by the
composer through `SquelchOpen(serial)`: open while no countdown runs, closed
while one does, `ok=false` outside a dwell or for another serial (which the
composer treats as "no gating available", never as closed). During the tail
the FM chain freezes its audio AGC and fades the PCM to silence instead of
amplifying receiver noise — the audible tail the #1090 reporter measured.
`TestConvScannerPublishesSquelchState` pins all four states.

## When the stream stops

Hangtime measures silence in chunks that arrive. A stream that stops
delivering without closing — a wedged USB pump — gives nothing to measure,
and the 250 ms ticker kept touching the watchdog, so the call stayed open.
That is what the
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) FleetSync
coexistence log showed: 627 ms after start the scanner opened a synthetic
call on a silent 146.67 MHz channel that ran 23 s and ended `reason=error`
only at Ctrl-C.

`Options.StreamStallTimeout` (zero selects `DefaultStreamStallTimeout`, 3 s)
closes the hole. `beginDwell` stamps `lastChunk` on every delivered chunk and
the ticker compares it:

```go
// internal/scanner/conventional/scanner.go (shape) — beginDwell ticker
case <-ticker.C:
    if stalled := s.opts.Now().Sub(lastChunk); stalled >= s.opts.StreamStallTimeout {
        s.log.Warn("conv: IQ stream stalled during dwell — ending call and re-opening the stream",
            "freq_hz", ch.FrequencyHz, "label", ch.Label,
            "no_samples_for", stalled.Round(time.Millisecond))
        s.endDwell(idx, trunking.EndReasonError)
        return
    }
    s.opts.Engine.Touch(s.opts.DeviceSerial)
```

Three seconds never trips on an RTL-SDR's few-millisecond chunks or a
throttled remote stream, and releases a dead pump before an operator
notices. The reason is `error`, not `normal` — a stall is not a hangtime —
and the loop then re-tunes and re-opens. `TestConvScannerStalledStreamEndsDwell`
drives a `stallIQ` that delivers one loud chunk and then nothing, asserts
`EndReasonError` inside the timeout, and asserts the stream is opened at
least twice. What the guard does not explain is *why* both of that
reporter's dongles stalled at the same instant; that returns in Part 13.

## Where it shows

From the engine, `synthetic call started device=<serial> grant=<…>` and
`synthetic call ended device=<serial> reason=<…>` bracket every dwell; from
the scanner, `conv: tune failed` and `conv: StreamIQ failed` mark a visit
that never got a window, and the stall WARN a dwell cut short.
`EndReason.String()` gives the words: `normal` is a hangtime release,
`error` a stream problem, `lockout` the operator locking out a dwelling
channel, `manual` a temporary channel removed mid-dwell.

`GET /api/v1/scanner` returns a `conventional` object
(`ConvScannerStatusDTO`) mirroring `Scanner.Snapshot`: `state`,
`device_serial`, `cursor_index`, and per channel `index`, `label`,
`frequency_hz`, `mode`, `active`, `locked_out`, `last_break_at`. The routes
`POST /api/v1/scanner/conventional/hold`, `/resume`, `/{index}/dwell`,
`/{index}/lockout` and `/{index}/unlockout` are `Hold`, `Resume`, `DwellOn`,
`LockoutChannel` and `UnlockoutChannel` one for one.

### How the dwell shaped the Go code

- **Interfaces the size of the tests.** `Tuner`, `IQSource`, `Engine` and
  `Recorder` expose one to three methods each, so `scanner_test.go` runs the
  real loop against `fakeTuner`, `fakeIQ` and `fakeEngine`.
- **Defaults applied in one place.** `New` fills `SquelchDbFS` −50,
  `Hangtime` 1500 ms, `ActivityDebounce` 50 ms, `SquelchHysteresisDb` 3 dB
  and `Mode` `fm`, so the daemon passes zeros through.
- **Every exit path clears the published squelch state.** `beginDwell`
  defers `squelchState.Store(squelchNoDwell)`.

## Where this goes next

The loop reads a `Channel` the daemon builds from one
`scanner.conventional[]` entry, and every field on it is an operator
decision. [Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }})
takes the scan list key by key — `mode`, `squelch_dbfs`, `hangtime_ms`,
`priority`, `talkgroup_id`, `gain`, `tone`, `decoders` — verifies each
against the config struct and `config.example.yaml`, and ends with the
config-discovery lines that decide whether the daemon is even reading the
file you edited.

## FAQ

**What does "conventional" mean in GopherTrunk?**
A fixed-frequency analog channel with no control channel. The scanner in
`internal/scanner/conventional` visits each `scanner.conventional` entry,
measures the channel's power, and on squelch break publishes a synthetic
grant through `Engine.HandleSyntheticCall`, so the recorder and call log
treat the dwell as an ordinary call.

**How long does the scanner stay on each channel?**
At least `MinDwellPerChannel` (100 ms) while scanning, raised to 350 ms when
any channel uses CTCSS or 600 ms with DCS. Once the squelch opens it stays
until the channel has been below `squelch_dbfs − 3 dB` for `hangtime_ms`
(1500 ms default), or until the stream stalls for 3 s.

**Why does my conventional call show a talkgroup like 2147483649?**
That is `0x80000001`, the positional default `0x80000000 | index` for the
second channel. Set `talkgroup_id` on each channel so the ID survives
reordering (issue #1105); `validateScanner` refuses two channels whose
effective IDs collide.

**Why did a conventional call end with reason=error?**
`EndReasonError` is a stream problem, not a hangtime release: the stream
closed, the dwell context was cancelled, or no chunk arrived for
`StreamStallTimeout` (3 s). The last case logs
`conv: IQ stream stalled during dwell — ending call and re-opening the stream`
with `no_samples_for`.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Scan List as Config — Channels, Modes, Hangtime, Priority]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }})
