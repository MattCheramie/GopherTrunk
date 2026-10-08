---
title: "The Conventional Scanner, Part 11: MDC1200 and FleetSync on Scan-List Channels — The Data Front End Behind the Dwell"
description: "How a scanner.conventional channel runs the MDC1200 and FleetSync decoders on its own IQ instead of pinning a dongle — the dataFrontEnd that decimates 2.4 MS/s to 48 kHz behind a ±8 kHz filter, the Busy flag that holds the dwell through a burst, the reset on every retune, and the 28 Sep on-air MDC1200 run."
category: tutorials
keywords: mdc1200 scanner channel, fleetsync scanner channel, decoders mdc1200 fleetsync, conventional scanner data decoder, ani decode sdr scanner, kenwood fleetsync unit id, motorola mdc1200 ptt id, scan list data burst, issue 1220, gophertrunk conventional scanner
tags: [conventional-scanner, mdc1200, fleetsync, scanner, dsp, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 11
---

*Part 11 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's non-trunked scan list — what you configure, what the code does
with it, what the log prints, and what is verified on air.
[Part 10]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})
put a coherent MSK decoder on an AM air-band channel. This part covers the two
FFSK decoders that got there first — Motorola MDC1200 and Kenwood FleetSync —
and the piece of the scanner that lets any decoder run without its own
dongle: the data front end behind the dwell
([#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220)).*

> **TL;DR:** `decoders: [mdc1200, fleetsync]` on a `scanner.conventional`
> entry runs the same receivers the dedicated `mdc1200.channels` /
> `fleetsync.channels` sections run, fed the scanner's own IQ. The scanner
> owns the **rate** — `dataFrontEnd` (`data.go`) decimates to ≥
> `dataRefRateHz` (48 kHz) with an `m` dividing the SDR rate
> (`pickDataDecimation`: 2.4 MS/s → ÷50) behind a ±`toneChannelCutoffHz`
> (8 kHz) filter; the **dwell** — a decoder whose `Busy()` is true extends
> the scan window in `dataScanHoldStep` (50 ms) steps up to `dataScanHoldMax`
> (500 ms) and counts as activity against hangtime; and the **retune** —
> every decoder is `Reset()` per tune. Measured on the #1184 real-air
> FleetSync slices at 2.4 MS/s with 0 dB band-wide noise: 0/2 bursts raw,
> 2/2 through the front end (`TestDataFrontEndDecodesRealAirFleetSyncAtSDRRate`).
> The CTCSS/DCS gate does not apply. MDC1200 is on-air verified through this
> path (28 Sep, unit 0x1777 on 447.100 MHz); a burst sent while the scanner
> is parked elsewhere is the trade.

**Key takeaways**

- **The decoder sees a channel, never a span.** `dataFrontEnd` hands every
  decoder one ~48 kHz channel whatever the SDR runs at, so a rad/sample
  threshold calibrated at 48 kHz stays calibrated — the #1184 CTCSS lesson
  applied before it could repeat.
- **A locked sync word owns the channel.** `Busy()` pushes the scan window
  out and refreshes the dwell, bounded by `dataScanHoldMax`, so neither the
  hop nor hangtime can cut a burst that has started framing.
- **Nothing new downstream.** `conv_decoders.go` builds the dedicated-SDR
  receivers with serial and frequency stamped on; the existing bus kinds,
  tables, routes and panels carry the rest.
- **The gate is for voice, not data.** A burst carries its own sync word and
  block check, so it is published even when the tone gate stays shut.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Config key | `decoders: [mdc1200, fleetsync]` (`acars` needs `mode: am`) | `config.ConvChannelConfig.Decoders`, `validateConvChannel` |
| Rate | integer decimation to ≥ 48 kHz with `m` dividing the SDR rate | `pickDataDecimation`, `dataRefRateHz`, `dataMaxResampleL` (`data.go`) |
| Channel filter | ±8 kHz Kaiser lowpass after decimation | `toneChannelCutoffHz` (`tone_frontend.go`), `newDataFrontEnd` |
| Hold | window extended 50 ms at a time, ≤ 500 ms (ACARS 1 s) | `dataScanHoldStep`, `dataScanHoldMax`, `acarsScanHoldMax`, `holdMaxFor` |
| Hangtime | a busy decoder counts as activity in `beginDwell` | `channelData.busy`, `TestDataDecoderBusyHoldsHangtime` |
| Factory | the dedicated-SDR receivers, serial + frequency stamped | `convDataDecoderFactory` (`cmd/gophertrunk/conv_decoders.go`) |
| Real-air pins | FleetSync slices at the SDR rate; MDC1200 unit 0x1777 slice | `data_test.go`, `afsk/testdata/mdc1200_unit1777_pttid_end_48k.cs16` |

## In this post

- **Why the decoder belongs on the dwell** — the design ask, and what the scanner already owns.
- **The data front end** — picking `m`, the ±8 kHz filter, the 0/2 → 2/2 measurement.
- **Busy holds the channel** — scan-window extension and the hangtime refresh.
- **The factory and the shared surface** — serial, frequency, bus kinds, logs, panels.
- **What is on air and what is not** — 28 Sep, the fixtures, the trade.

## Why the decoder belongs on the dwell

Until #1220, MDC1200 and FleetSync each needed a whole SDR: an entry under
`mdc1200.channels` or `fleetsync.channels` pins a dongle to one frequency for
the daemon's lifetime
([Beyond Voice Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
and [Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
built those). The reporter's design ask was the obvious one: the scanner
already holds every channel it monitors, and the burst a radio keys at the
head of a transmission arrives on exactly the channel the scanner just
opened squelch on.

The package comment in `internal/scanner/conventional/data.go` names the
three things the scanner owns so the decoders don't have to. The **sample
rate**: the scanner's IQ arrives at the SDR's full rate — 2.4 MS/s on an
RTL-SDR — and an FM discriminator run on that is dominated by the band's
noise, with every rad/sample threshold downstream reading ~50× off (the
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) CTCSS
lesson). The **dwell**: a burst the scanner walks away from mid-frame is
lost. And the **retune**: decoder state must be `Reset()` on every tune, like
the tone detectors, so one channel's filter history never bleeds into the
next.

The interface the scanner asks a decoder to satisfy is three methods:

```go
// internal/scanner/conventional/data.go
type DataDecoder interface {
    ProcessIQ(iq []complex64) // one chunk of channel IQ at the factory's rate
    Busy() bool               // locked a sync word, part-way through a burst
    Reset()                   // clear DSP state; called on every retune
}
type DataDecoderFactory func(ch Channel, kind string, rateHz float64) (DataDecoder, error)
```

`rateHz` is the front end's output, never the SDR rate —
`TestDataDecoderFactoryGetsChannelRate` pins that a 2.4 MS/s scanner hands
the factory exactly 48000. `validateDecoders` rejects any name outside
`DecoderMDC1200` / `DecoderFleetSync` / `DecoderACARS` and a name listed
twice; config validation adds that `acars` needs `mode: am`.

## The data front end

`newDataFrontEnd(sampleHz)` builds two stages. The first is integer
decimation through `dsp.NewResampler(1, m, m*8+1, 8.6)` — a polyphase
anti-alias FIR — where `m` comes from `pickDataDecimation`:

```go
// internal/scanner/conventional/data.go (shape)
func pickDataDecimation(sampleHz float64) int {
    in := int64(sampleHz)
    if float64(in) != sampleHz || in < 2*dataRefRateHz {
        return 1
    }
    for m := in / dataRefRateHz; m >= 2; m-- {
        if in%m != 0 { continue }
        out := in / m
        if dataFFSKRateHz/gcd64(out, dataFFSKRateHz) <= dataMaxResampleL {
            return int(m)
        }
    }
    return 1
}
```

Three constraints are folded in: `m` must divide the input rate exactly, so
the output is an integer rate; the output stays at or above `dataRefRateHz`
= 48 kHz, the rate the FleetSync real-air slices were verified at; and the
interpolation a decoder would need to reach its 9600 Hz discriminator rate
(`dataFFSKRateHz`, 1200 baud × 8) is bounded by `dataMaxResampleL` = 48.
`TestPickDataDecimation` tabulates it: 2.4 MS/s → `m` = 50 → 48 kHz;
2.048 MS/s → 40 → 51.2 kHz (48 761.9 is not an integer rate); 3.2 MS/s →
64 → 50 kHz; a 48 kHz input is left alone.

The second stage is `filter.LowpassKaiser(63, fc, 8.6)` at
`fc = toneChannelCutoffHz / rate` — the same ±8 kHz the tone front end uses
(`tone_frontend.go`: an FM channel passes, a 12.5/25 kHz neighbour does not),
but kept as IQ because each decoder runs its own discriminator. It is skipped
when the rate is too low to need it (`fc ≥ 0.45`).

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A pipeline from a 2.4 megasample per second SDR stream through the scanner's data front end to the decoders. The scanner's scan window and dwell gate which chunks are fed. The front end decimates by 50 to 48 kilohertz behind an anti-alias filter, then applies a plus or minus 8 kilohertz channel filter. The channel IQ fans out to the MDC1200 and FleetSync receivers, each with its own FM discriminator and FFSK chain, which publish onto the events bus to the SQLite logs, REST routes and web panels. A side label shows the measurement: zero of two bursts raw, two of two through the front end.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">dataFrontEnd: one channel at ~48 kHz, whatever the SDR runs at</text>
  <rect x="12" y="40" width="110" height="44" fill="none" stroke="var(--fg-muted)"/>
  <text x="67" y="58" text-anchor="middle" fill="currentColor" font-size="9">SDR IQ</text>
  <text x="67" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2.4 MS/s, whole band</text>
  <path d="M122 62 L150 62" stroke="currentColor"/>
  <rect x="150" y="40" width="120" height="44" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="210" y="58" text-anchor="middle" fill="currentColor" font-size="9">scanWindow / beginDwell</text>
  <text x="210" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">chunks above squelch</text>
  <path d="M270 62 L298 62" stroke="currentColor"/>
  <rect x="298" y="30" width="150" height="64" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="373" y="48" text-anchor="middle" fill="var(--accent)" font-size="9" font-weight="bold">dataFrontEnd</text>
  <text x="373" y="63" text-anchor="middle" fill="currentColor" font-size="8">÷ m = 50 → 48 kHz (anti-alias FIR)</text>
  <text x="373" y="78" text-anchor="middle" fill="currentColor" font-size="8">±8 kHz Kaiser channel filter</text>
  <path d="M448 50 L500 50" stroke="currentColor"/>
  <path d="M448 74 L500 74" stroke="currentColor"/>
  <rect x="500" y="36" width="168" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="584" y="53" text-anchor="middle" fill="currentColor" font-size="8">mdc1200afsk: FM → FFSK → XOR → framer</text>
  <rect x="500" y="64" width="168" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="584" y="81" text-anchor="middle" fill="currentColor" font-size="8">fleetsyncafsk: FM → FFSK → framer</text>
  <path d="M584 90 L584 118" stroke="currentColor"/>
  <rect x="440" y="118" width="228" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="554" y="135" text-anchor="middle" fill="currentColor" font-size="8">bus → mdc1200_log / fleetsync_log → REST → panel</text>
  <text x="554" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="8">stamped serial + frequency_hz (the Channel column)</text>
  <line x1="12" y1="185" x2="400" y2="185" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="12" y="203" fill="currentColor" font-size="9" font-weight="bold">Measured, #1184 FleetSync slices at 2.4 MS/s, 0 dB band-wide noise:</text>
  <text x="12" y="219" fill="var(--fg-muted)" font-size="9">raw receiver 0/2 bursts · through dataFrontEnd 2/2 bursts</text>
  <text x="373" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Busy() → hold window ≤ 500 ms · reset() on retune</text>
</svg>
<figcaption>The scanner owns the rate, the dwell and the retune; the decoders are the unchanged dedicated-SDR receivers behind them.</figcaption>
</figure>

The measurement that justifies the stage is in `data_test.go`.
`loadFleetSyncAt2400k` interpolates the two committed 48 kHz real-air
FleetSync slices (Fleet 107 / Unit 1772, from the #1184 captures) to the
2.4 MS/s the scanner actually delivers and adds white noise across the
**whole** band at 0 dB relative to the signal — in the ~16 kHz channel that
noise sits ~17 dB down, a comfortable in-channel signal.
`TestDataFrontEndDecodesRealAirFleetSyncAtSDRRate` feeds that stream to a
`fleetsyncafsk` receiver raw at 2 400 000 Hz, where nothing decodes
(measured 0/2 bursts on both slices), and through
`newDataFrontEnd(2_400_000)`, where the same samples decode every burst.

## Busy holds the channel

A scanner that hops every `MinDwellPerChannel` would cut a burst that locked
near the end of its window, so `scanWindow` consults the decoders when its
deadline fires:

```go
// internal/scanner/conventional/scanner.go (shape) — scanWindow
case <-deadline.C:
    if cd != nil && held < cd.holdMax && cd.busy() {
        held += dataScanHoldStep        // 50 ms
        deadline.Reset(dataScanHoldStep)
        continue
    }
    return false
```

`holdMax` is per channel: `dataScanHoldMax` = 500 ms for the FFSK decoders,
sized to the longest burst either frames — an MDC1200 double packet or a
FleetSync II frame, ~250 bits ≈ 210 ms at 1200 baud — so a burst that locks
at the deadline completes, while a false lock on noise delays the scan by at
most half a second; ACARS gets `acarsScanHoldMax` = 1 s for its ~0.86 s
block (`holdMaxFor`). `TestDataDecoderBusyExtendsScanWindow` pins both ends:
an idle decoder's window ends at `MinDwellPerChannel`, a busy one's is
extended by ~`dataScanHoldMax` and no more.

Two more details decide whether a burst survives. The chunk that *opens* the
squelch is fed to the decoders before the dwell starts
(`TestDataDecoderSeesSquelchBreakingChunk`), because a burst keyed at the
head of a transmission starts on the first chunk above threshold. And inside
`beginDwell` the decoders run on every chunk, with `cd.busy()` promoting a
silent chunk to `active` — so hangtime, which counts silent chunks, cannot
end the call mid-burst. `TestDataDecoderBusyHoldsHangtime` drives a 100 ms
hangtime at 10 ms per silent chunk: no burst, the dwell ends after ~10
chunks; a decoder busy for 30 chunks, after more than 40.

`Busy()` itself is the framer's state — `r.st != stateHunt` for MDC1200,
`f.st == stateCapture` for FleetSync. On every retune `Run` calls
`cd.reset()`, clearing the front end and each decoder
(`TestDataDecoderResetOnRetune`), so a burst cut off on one channel cannot
swallow the next channel's bits as its payload.

## The factory and the shared surface

The scanner package never imports a decoder. The daemon supplies
`Options.DataDecoders` from `convDataDecoderFactory` in
`cmd/gophertrunk/conv_decoders.go`:

```go
// cmd/gophertrunk/conv_decoders.go (shape)
source := fmt.Sprintf("%s@%d", serial, ch.FrequencyHz)
switch kind {
case conventional.DecoderMDC1200:
    return mdc1200afsk.New(mdc1200afsk.Options{
        InputRateHz: rate, SourceName: source,
        Serial: serial, FrequencyHz: ch.FrequencyHz,
        Bus: bus, Log: log,
    })
case conventional.DecoderFleetSync: // fleetsyncafsk.New, same fields
case conventional.DecoderACARS:     // acarsrx.New, same fields
}
```

These are the constructors the dedicated-SDR sections call, so the decoders
publish onto the same bus kinds — `events.KindMDC1200Message` with a
`storage.MDC1200Message`, `events.KindFleetSyncMessage` with a
`storage.FleetSyncMessage` — and the existing `mdc1200_log` /
`fleetsync_log` tables, `GET /api/v1/mdc1200/messages` /
`GET /api/v1/fleetsync/messages` and the `/mdc1200` / `/fleetsync` panels
need nothing new. Each burst carries the scanner's SDR serial and the
channel's frequency — what the panels' **Channel** column renders.
`TestConvDataDecoderFactoryPublishesWithChannelIdentity` pins it: the
real-air FS-II slice through the factory's decoder yields a CRC-valid
Fleet 107 / Unit 1772 message with `Serial == "CONV-SDR"` and
`FrequencyHz == 462562500`.

Two startup checks guard the wiring. `buildChannelData` WARNs rather than
silently dropping a decoder it cannot build — `conv: data decoders configured
but the scanner has no decoder factory or sample rate; decoders disabled`, or
`conv: data decoder failed to initialise; channel scans without it`. And
`preflight.go` folds the scan-list decoders into its storage check
(`convChannelDecoders`): with `storage.path` empty the warning names
`mdc1200`, `fleetsync` or `acars` — the decoders run, but their REST routes
return 503 and the panels stay empty. One gap: `ConvChannelStatusDTO` on
`/api/v1/scanner` does not map the scanner snapshot's `Decoders` field.

## What is on air and what is not

MDC1200 through this exact path is **on-air verified**. On 28 Sep the
reporter's live run on the fixed build decoded eleven PTT ID bursts — start
(op 0x01 / arg 0x80) and end (0x01 / 0x00), unit 0x1777, 447.100 MHz, narrow
and wide FM — through a scanner-channel decoder, and their 2.4 MS/s scanner
voice recording replays through `TestMDC1200Replay` (carrier found at
−1758 Hz, CRC-valid PTT ID) despite being ADC-clipped at both rails, a
handheld in the room
([Analog Edge Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
covers why clipped is not the same as undecodable). A 0.7 s channelized slice
is the committed real-air regression,
`internal/radio/mdc1200/afsk/testdata/mdc1200_unit1777_pttid_end_48k.cs16`
(`TestMDC1200RealAirSlice`); the harness takes `GT_MDC1200_IQ`,
`GT_MDC1200_RATE` and `GT_MDC1200_UNIT` for the next capture. That run came
two days after a first live attempt decoded FleetSync 8/8 through the
identical DSP chain while MDC1200 never locked — the XOR-precoded line-code
defect the decoder's own tests had encoded the same wrong way, the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
again.

FleetSync is on-air verified on the reporter's Kenwood lab (16 Sep, FS-I and
FS-II live) through the *dedicated-SDR* section, and capture-pinned on the
scanner path: `TestScannerDecodesRealAirFleetSyncOnChannel` runs the real-air
FS-II slice at the SDR rate through the whole scanner — squelch break, dwell,
front end and a real `fleetsyncafsk` receiver from the factory — and requires
two CRC-valid FS-II messages. Per the repo's #764/#771 rule a green capture
is not a live pass; the Kenwood lab run with `decoders: [fleetsync]` on a
scan-list channel is the remaining gate.

Two limits are by design. A burst sent while the scanner is parked on another
channel is missed — the trade against pinning an SDR, and a
`fleetsync.channels` entry remains the answer for a channel that must never be
missed. And the decoders ignore the CTCSS/DCS gate on purpose: a burst
carries its own sync word and block check, so it is logged even when the
gate stays shut and no voice call opens.

## Where this goes next

The scan list so far has been a plain rotation with a squelch. Hardware
scanners add two things to that: a priority channel sampled several times per
lap, and a lockout memory that survives a power cycle.
[Part 12]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }})
reads `scanner.priority_interleave` and `pickNextChannel`, then the
`conv_lockouts` table that keys a lockout by frequency — never by list index,
which shifts the moment the operator edits the list.

## FAQ

**Why does the scanner decimate before handing IQ to the decoder?**
Because the FFSK discriminators were calibrated on a ~48 kHz channel, and at
2.4 MS/s the band's noise swamps them. `dataFrontEnd` picks an integer `m`
dividing the SDR rate (`pickDataDecimation`: 2.4 MS/s → ÷50) and filters to
±8 kHz; on the #1184 FleetSync slices that is 0/2 versus 2/2 bursts decoded.

**Will the scanner hop away in the middle of a data burst?**
No. A decoder whose `Busy()` is true — a sync word locked, a burst part-way
through — extends the scan window in 50 ms steps up to `dataScanHoldMax`
(500 ms; 1 s for ACARS) and counts as channel activity during the dwell, so
hangtime cannot end the call mid-burst. A false lock on noise costs at most
that bounded delay.

**Does a CTCSS or DCS tone gate block the data decoders?**
No. The decoders run on every chunk that clears the power squelch and on the
whole dwell, independent of the tone gate; a burst carries its own sync word
and block check, so it is published even when no voice call opens. The gate
decides whether the dwell becomes a recording, not whether a burst is logged.

**Is the scanner-channel decoder path verified on air?**
MDC1200 is: the reporter's 28 Sep live run decoded eleven PTT ID bursts (unit
0x1777, 447.100 MHz) through a scan-list channel, and a slice of their
recording is a committed regression. FleetSync is on-air verified on a
dedicated SDR and capture-pinned on the scanner path
(`TestScannerDecodesRealAirFleetSyncOnChannel`); its live scan-list run is
open.

## Series navigation

**Part 11 of 14** · ←
[Part 10: ACARS on a Scanner Channel — Coherent MSK and Parity Repair]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})
· Next →
[Part 12: Priority Interleave and Persistent Lockouts — Keyed by Frequency, Never List Index]({{ '/blog/tutorials/conventional-scanner-12-priority-interleave-and-lockouts/' | relative_url }})
