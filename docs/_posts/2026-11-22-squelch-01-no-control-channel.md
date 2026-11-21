---
title: "Squelch, Part 1: No Control Channel — How a Scan List Decides a Call Exists"
description: "A trunked system announces every call on its control channel; a scan list has to infer one from the air. This opener walks GopherTrunk's conventional dwell loop — pick, tune, scan window, synthetic call, hangtime — and maps the gates the series takes apart, almost all of which began as a level threshold and became a statistic."
category: deep-dives
keywords: conventional scanner sdr, scan list squelch, scanner dwell loop hangtime, synthetic call trunking engine, carrier squelch dbfs, squelch activity debounce, iq fan-out broker, analog fm scanner rtl-sdr, how a scanner detects a call, gophertrunk squelch
tags: [squelch, conventional, scanner, analog-fm, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 1
---

*Part 1 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens: channel power, noise
squelch, CTCSS and DCS, AM carrier-to-noise, and the per-protocol call gates
for P25, DMR, System Fusion, NXDN and D-STAR. This opener is the loop
itself: how `internal/scanner/conventional` turns "something is on this
frequency" into a recorded call, which SDR it does that on, and the map of
the thirteen parts that follow.*

> **TL;DR:** A trunked decoder is *told* about calls: a control-channel
> grant reaches `Engine.HandleGrant`. A scan list is told nothing.
> `Scanner.Run` in `internal/scanner/conventional/scanner.go` picks a channel
> (`pickVisit`), tunes, opens a fresh IQ stream and gives the channel one
> scan window — `MinDwellPerChannel`, 100 ms by default — to pass every gate
> it carries. If it does, `beginDwell` builds a `trunking.Grant` and calls
> `Engine.HandleSyntheticCall` (`synthetic call started` in the log), so the
> recorder, call log and API treat it like any trunked call. The dwell ends
> after `Hangtime` (1500 ms) without activity; `ActivityDebounce` (50 ms)
> stops a one-chunk blip resetting that countdown. The scanner owns its SDR,
> the last `role: voice` device, behind an IQ broker the voice chain
> subscribes to. Every gate that began as a level threshold has since been
> replaced by a statistic or paired with one.

**Key takeaways**

- **A scan list infers what a control channel announces.** No grant, no
  talkgroup, no release message: the only evidence is the IQ, and each
  channel gets about a tenth of a second to show it.
- **One loop makes two different decisions.** The scan window decides
  whether a channel opens; the hangtime countdown decides when it closes.
- **The scanner manufactures a grant, not a recording path.** A dwell is a
  `trunking.Grant` handed to the engine a trunked system feeds, so
  everything downstream is shared code.
- **"Is something there?" is not one question.** Power in the channel, a
  carrier rather than noise, the right tone, a call rather than an idle
  repeater: each is its own gate.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Channel rotation | round robin skipping lockouts; forced dwell and priority interleave first | `pickNextChannelLocked` in `internal/scanner/conventional/scanner.go` |
| Scan window | one visit's time to open: 100 ms, raised to 350 / 600 / 500 ms when the list has a CTCSS / DCS / P25 NAC gate | `scanWindow`, `Options.MinDwellPerChannel`, `minToneDwell` |
| Synthetic call | a grant flagged conventional, with bit 31 set in its group ID | `beginDwell` → `Engine.HandleSyntheticCall` |
| Call end | hangtime countdown, 50 ms activity debounce, 3 dB close-side hysteresis | `Channel.Hangtime`, `ActivityDebounce`, `SquelchHysteresisDb` |
| Stream ownership | the scanner is the IQ broker's primary consumer; the voice chain subscribes | `buildConvScanner`, `convScanVoiceSource` in `cmd/gophertrunk/daemon.go` |
| Dead stream | 3 s without an IQ chunk ends the call | `Options.StreamStallTimeout` |

## In this post

- **What a control channel gives you for free** — and what a scan list infers instead.
- **The loop: pick, tune, window** — `Run`, `pickVisit` and the scan window.
- **The dwell: a synthetic call and a countdown** — the grant, hangtime, debounce.
- **The SDR the scanner owns** — the last voice device, and the IQ broker.
- **The gate map** — one table for the whole series, and its thesis.

## What a control channel gives you for free

On a trunked system the hard question is answered by someone else. The site's
[control channel]({{ '/reference/control-channel/' | relative_url }})
transmits a [grant]({{ '/reference/channel-grant/' | relative_url }}) naming
the talkgroup, the radio and the voice frequency; `Engine.HandleGrant` binds
a `role: voice` SDR and retunes it to a frequency where the system has
already said there is a call
([SDR in Pure Go Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})).
The receiver's work starts *after* the call's existence is settled.

A [conventional]({{ '/reference/conventional-radio/' | relative_url }})
channel is a frequency and nothing else. The scanner has to answer three
questions from the IQ alone, on every channel, on every pass: is anything
here right now; is it what this scan-list entry is for; and when did it
stop.

This blog has covered that twice, briefly:
[Protocol Decoders Part 9]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
gave the scanner one section, and
[The Operator's Cookbook Part 6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
one recipe. Since then the squelch moved from the SDR's span to the channel,
FM gained a noise squelch, AM a carrier-to-noise squelch, and the scan list
learned five digital protocols. The loop barely changed, so it comes first.

## The loop: pick, tune, window

`Scanner.Run` is one goroutine and a plain `for`:

```go
// internal/scanner/conventional/scanner.go (shape)
for {
    if s.IsHeld() { s.sleep(ctx, 100*time.Millisecond); continue }
    v, ch, ok := s.pickVisit()           // next channel + its visit, one lock
    if !ok { s.sleep(ctx, 100*time.Millisecond); continue }
    s.applyChannelGain(ch)               // per-channel gain, before the tune
    s.opts.Tuner.SetCenterFreq(ch.FrequencyHz)
    stream, _ := s.opts.IQ.StreamIQ(streamCtx) // a fresh stream every visit
    // reset this channel's gate, data decoders, meters and level gauge
    if !s.scanWindow(streamCtx, idx, ch, stream, pos) { cancel(); continue }
    s.beginDwell(v, ch, stream, streamCtx, cancel)
}
```

**Pick.** `pickNextChannelLocked` tries three things in order: a forced
dwell (`DwellOn`, the TUI's Enter key), which beats even a lockout for one
cycle; then, with `PriorityInterleave` set, the next channel carrying a
`Priority` ([priority scan]({{ '/reference/priority-scan/' | relative_url }}),
Part 13); otherwise the round-robin cursor, skipping `lockedOut`. An empty
or fully locked-out list idles in 100 ms ticks. (The operator's view of
those controls is
[Cookbook Volume 2 Part 9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }}).)

**Tune.** The stream is cancelled and reopened every visit, "to drop any
in-flight samples from the previous channel", and every per-channel meter
and gate is reset for the same reason.

**Window.** `scanWindow` starts a timer for `MinDwellPerChannel` and reads
chunks until the channel opens (`true`) or the timer fires (`false`, next
channel). The default is 100 ms, and `New` raises it when a gate on the list
needs longer to report: 350 ms for CTCSS (`ctcssMinDwell`), 600 ms for DCS
(`dcsMinDwell`), 500 ms for a P25 NAC filter (`nacMinDwell`). It is one
value per scanner, not per channel: ten untoned channels are about a second
a lap before tuning time, and one CTCSS entry makes that three and a half.

The window can also be *extended*, under one rule: only a channel that has
cleared its level threshold may ask — "a silent channel costs no extra
time", as the `callGate` comment puts it.

| Extension | Who asks | Bound |
|---|---|---|
| Data burst in progress | an MDC1200, FleetSync or ACARS framer that has locked sync | 50 ms steps (`dataScanHoldStep`) up to 500 ms, 1 s with ACARS |
| Call gate settling | a DMR, YSF, NXDN or D-STAR channel whose power squelch has broken | once, to the gate's `minDwell()`: 500 ms DMR, YSF, NXDN96; 1 s NXDN48; 1200 ms D-STAR |
| Noise squelch's first reading | an FM channel over its level before the meter has a window | once: 60 ms (`fmNoiseSettle`) |

When a channel opens the scanner logs what it measured: `conv: squelch
opened` with `level_dbfs` and `squelch_dbfs`, or `conv: call gate opened`
for a DMR, NXDN, D-STAR or System Fusion gate. Read that line first when a
channel opens wrongly.

<figure class="lab-figure">
<svg viewBox="0 0 680 222" width="680" height="222" role="img" aria-label="The conventional scanner's dwell loop: pick the next channel, tune and open a fresh IQ stream, then run a 100 millisecond scan window. If nothing opens, the loop moves to the next channel. If every gate passes, a synthetic call starts; when activity stops, a 1500 millisecond hangtime countdown runs, then scanning resumes.">
  <text x="252" y="13" text-anchor="middle" fill="var(--fg-muted)" font-size="8">window expired, nothing open → next channel</text>
  <path d="M425 34 L425 20 L80 20 L80 34" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <rect x="10" y="34" width="140" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="80" y="50" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">pickVisit</text>
  <text x="80" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="8">next channel, skip lockouts</text>
  <line x1="150" y1="53" x2="180" y2="53" stroke="currentColor"/>
  <rect x="180" y="34" width="140" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="250" y="50" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">tune + StreamIQ</text>
  <text x="250" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gain, LO, reset the meters</text>
  <line x1="320" y1="53" x2="350" y2="53" stroke="currentColor"/>
  <rect x="350" y="34" width="150" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="425" y="50" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">scanWindow</text>
  <text x="425" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="8">100 ms; a gate may extend it</text>
  <line x1="500" y1="53" x2="530" y2="53" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="515" y="46" text-anchor="middle" fill="var(--accent)" font-size="8">open</text>
  <rect x="530" y="34" width="140" height="38" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="600" y="50" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">beginDwell</text>
  <text x="600" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="8">synthetic call started</text>
  <line x1="515" y1="57" x2="515" y2="96" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <rect x="150" y="96" width="372" height="78" rx="4" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="160" y="111" fill="var(--accent)" font-size="9" font-weight="bold">the open edge: every gate the channel carries must pass</text>
  <text x="160" y="125" fill="currentColor" font-size="8">fm / nfm: level ≥ squelch_dbfs AND noise ratio ≤ 0.5</text>
  <text x="160" y="138" fill="currentColor" font-size="8">with a tone: AND the CTCSS or DCS detector reports present</text>
  <text x="160" y="151" fill="currentColor" font-size="8">am: carrier-to-noise ≥ squelch_cn_db (12 dB by default)</text>
  <text x="160" y="164" fill="currentColor" font-size="8">digital modes: level AND the call gate (p25: only with nac set)</text>
  <line x1="600" y1="72" x2="600" y2="112" stroke="currentColor"/>
  <text x="594" y="95" text-anchor="end" fill="var(--fg-muted)" font-size="8">activity stops</text>
  <rect x="530" y="112" width="140" height="38" rx="4" fill="none" stroke="currentColor"/>
  <text x="600" y="128" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">hangtime</text>
  <text x="600" y="141" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1500 ms; 50 ms debounce</text>
  <path d="M600 150 L600 196 L80 196 L80 72" fill="none" stroke="currentColor"/>
  <text x="340" y="211" text-anchor="middle" fill="var(--fg-muted)" font-size="8">hangtime elapsed → synthetic call ended → scanning resumes</text>
</svg>
<figcaption>One loop, two decisions: the scan window decides whether a channel opens, and the hangtime countdown decides when it closes.</figcaption>
</figure>

## The dwell: a synthetic call and a countdown

When the window returns `true`, `beginDwell` takes over the still-open stream
and records nothing. It writes a grant.

```go
// internal/scanner/conventional/scanner.go (shape)
g := trunking.Grant{
    System:       s.opts.SystemName,        // "scanner" in the daemon
    Protocol:     conventionalProtocol(ch), // fm-conv, am-conv, p25, dmr-tier2 …
    GroupID:      gid,                      // talkgroup_id, else 0x80000000|index
    Conventional: true,
    FrequencyHz:  ch.FrequencyHz,
}
s.opts.Engine.HandleSyntheticCall(g, s.opts.DeviceSerial)
```

`HandleSyntheticCall` registers a call from a source that "already owns its
SDR — no VoicePool binding, no re-tune, no preemption logic", and publishes
the same `CallStart` event a trunked grant produces. From the engine outward
nothing knows the difference: the composer reads the protocol and builds the
matching chain (`fm-conv` and `am-conv` go to `runFMChain`, the digital
modes to their own voice chains), the recorder opens a file, the call log
gets a row. Two lines from the reporter's run in
[From the Issue Tracker Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }}):

```text
synthetic call started  device=SDRV4-03 grant="scanner/fm-conv tg=2147483648 src=0 freq=146670000"
synthetic call ended    device=SDRV4-03 reason=normal
```

`tg=2147483648` is `0x80000000`: the first scan-list entry, with bit 31 set
so a positional ID cannot collide with a real
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}); `talkgroup_id`
pins it against list reordering (#1105).

Then the scanner stays put and decides, chunk by chunk, whether the channel
is still active: the open edge's tests, except that the level only has to
stay above `keepAlive`, 3 dB under the open threshold
(`SquelchHysteresisDb`), so a signal sitting on the threshold does not
chatter. The result drives a countdown:

```go
// internal/scanner/conventional/scanner.go (shape)
if active {
    if aboveSince.IsZero() { aboveSince = now }
    if belowSince.IsZero() || now.Sub(aboveSince) >= ch.ActivityDebounce {
        belowSince = time.Time{} // countdown cleared
    }
} else {
    aboveSince = time.Time{}
    if belowSince.IsZero() {
        belowSince = now // countdown starts
    } else if now.Sub(belowSince) >= ch.Hangtime {
        s.endDwell(v, trunking.EndReasonNormal)
        return
    }
}
```

`Hangtime` defaults to 1500 ms (`hangtime_ms`), so a pause or a reply stays
one call. The debounce (50 ms, `activity_debounce_ms`) exists because of
#1090. The original loop cleared the countdown on *any* chunk above the
threshold; a chunk at 2.4 MS/s lasts under two milliseconds, so a blip
recurring faster than the hangtime — a carrier tail, impulse noise, an
intermod product — held the call open indefinitely. Now activity must be
continuous for 50 ms, about thirty chunks, before it counts as renewed; a
lone blip sets `aboveSince`, the next quiet chunk clears it, and the silence
already counted survives (`TestConvScannerBriefBlipsDoNotHoldSquelchOpen`).

The edges are asymmetric on purpose: the trailing edge is debounced, the
leading edge is not, because a late open loses the start of a transmission
(`squelch_open_ms` is the opt-in leading-edge equivalent).

Two more things run inside the dwell. The debounced decision is published
through `SquelchOpen`, so the composer's FM chain mutes during the countdown
instead of recording 1.5 s of receiver noise on the end of every call. And a
250 ms ticker checks that chunks are still arriving: `StreamStallTimeout`
(3 s) without one ends the call with reason `error`, because hangtime cannot
fire on a stream that delivers nothing (Part 10).

Meanwhile the scanner is deaf to every other channel. Both ways out are
more receivers: channels can name a `device:` so each SDR runs its own
scanner (#1260), and a `role: wideband` dongle can carry conventional
channels as fixed taps that never hop (#1288).

## The SDR the scanner owns

The daemon builds a conventional scanner when `scanner.conventional` has
entries, when `manual_tune_enabled` is set, or when it finds a spare voice
SDR. `planConvScanGroups` gives it **the last `role: voice` device** in the
pool, so the trunking engine keeps the others. With no voice SDR the
scanner is skipped with a WARN (`daemon: scanner.conventional /
manual_tune_enabled configured but no Voice SDRs in the pool; skipping`),
the first line to grep for when a scan list does nothing.

Owning the SDR creates the problem From the Issue Tracker Part 16 (linked
above) is about. On a trunked system the monitor and the recorder are
different devices. On a scan list the monitored channel *is* the recorded
channel, and a device hands out one `StreamIQ` at a time. So
`buildConvScanner` gives the scanner the device's `iqtap` broker, not the
device, and makes it the broker's primary consumer. When a call starts the
voice chain gets a `convScanVoiceSource`, which calls `broker.Subscribe()`
and receives copies of exactly the IQ the squelch is measuring.

Between broker and scanner sits one more wrapper, `convScannerFrontEnd`,
which tunes the hardware a few hundred kilohertz *below* each channel and
mixes it back to centre, so the tuner's own artefacts land outside what the
gates measure (Part 4).

## The gate map

Everything else in this series hangs off the "open" edge in the figure:

| Gate | Question it answers | Statistic | Part |
|---|---|---|---|
| Channel power | is there energy in *this channel*? | RMS power behind a ±8 kHz filter, against `squelch_dbfs` | [2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }}) |
| Noise squelch (fm, nfm) | is that energy a carrier, or noise? | discriminator noise over what an empty channel gives: open at 0.5, hold to 0.6 | [3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }}) |
| Offset tuning | not a gate: keeps the tuner's own products out of the channel | an LO offset searched so DC spur, image and clipping products miss it | [4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }}) |
| CTCSS | is it this system's tone? | exact-frequency Goertzel over 250 ms blocks, against neighbouring tones | [5]({{ '/blog/deep-dives/squelch-05-ctcss/' | relative_url }}) |
| DCS | is it this system's code? | the 23-bit codeword, in the configured polarity | [6]({{ '/blog/deep-dives/squelch-06-dcs/' | relative_url }}) |
| AM carrier-to-noise | is there an AM carrier? | strongest bin within ±4.5 kHz over the 20th-percentile floor; opens at 12 dB | [7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }}) |
| P25 NAC, DMR call gate | is it a call, on this system? | a NID carrying the NAC; a voice sync, Voice LC Header or PI header in the last second | [8]({{ '/blog/deep-dives/squelch-08-call-gates-p25-dmr/' | relative_url }}) |
| YSF, NXDN, D-STAR gates | is it that protocol's traffic? | a frame sync; a confirmed frame pair; a CRC-valid header or slow-data sync pair | [9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}) |
| Burst bridge, stall timeout | is this gap a release? | a 100 ms bridge on DMR; 3 s with no samples at all | [10]({{ '/blog/deep-dives/squelch-10-bursty-carriers/' | relative_url }}) |
| Data decoders | is a burst still arriving? | a framer that has locked sync | [11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }}) |

Parts
[12]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }}),
[13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }})
and
[14]({{ '/blog/deep-dives/squelch-14-playbook/' | relative_url }})
cover the shared narrowband front end, the operator's controls, and a
playbook of failure signatures.

The thesis is the Statistic column. The first version of this scanner had
one test: `PowerDbFS` of the stream against a threshold. That is an absolute
level, and
[The Analog Edge Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
already showed what an absolute level does to a decision: it measures how
the operator set the gain, not what is on the air. An idle DMR repeater
holding the scanner on its channel, an analog test tone
[opened as "System Fusion"]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }}),
more than half an hour of noise recorded from an empty channel: each was a
level standing in for a question it could not answer, and each fix answered
it with a ratio, a decoded field or a frame. The package comment in
`squelch.go` now says each mode pairs the level "with a test that does not
depend on gain". `squelch_dbfs` survives as one of two tests, never the only
one, with a single named exception: a `p25` channel configured without `nac`
still opens on the level alone.

### How having no control channel shaped the Go code

- **The scanner speaks the engine's vocabulary.** Its `Engine` dependency is
  three methods (`HandleSyntheticCall`, `EndSyntheticCall`, `Touch`); a fake
  satisfies it in every scanner test.
- **Every gate is one interface.** `toneDetector` is `Process`, `Present`
  and `Reset`; `callGate` adds `gateName` and `minDwell`.
  `buildChannelDetector` chooses by mode, so a new gate is one
  implementation and one branch.
- **Time on a channel is earned.** A call gate or data decoder extends a
  visit only after the level has cleared. The tone and NAC gates are the
  exception: they raise the window for the whole list.

## Where this goes next

The first gate is the oldest, and for most of its life it measured the wrong
thing: `squelch_dbfs` was compared with the power of the scanner's entire
span, so one strong carrier anywhere in view held every channel open.
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
is the fix and the arithmetic behind it.

## FAQ

**How does GopherTrunk know a call exists on a conventional channel?**
It measures the channel's own IQ during a short scan window (100 ms by
default) and opens only if every gate on that channel passes: in-channel
power, plus a noise squelch on FM, carrier-to-noise on AM, a tone detector
if one is set, or a call gate on a digital channel.

**What is a synthetic call in the GopherTrunk log?**
`synthetic call started` is the trunking engine registering a call that did
not come from a control channel: the scanner built a `trunking.Grant` for
the open channel, so recorder, call log and API handle it like any other.
`synthetic call ended` carries the reason: `normal` for hangtime, `error`
for a lost stream, `lockout` or `manual` for an operator action.

**Why does the scanner stay on a channel after the transmission ends?**
That is hangtime: the channel must be inactive for `hangtime_ms` (1500 by
default) before the call ends, so a pause or a reply stays in one recording.
A blip shorter than `activity_debounce_ms` (50) cannot reset the countdown.

**How fast does the conventional scanner scan?**
Each visit gets one scan window, 100 ms by default, plus retune time. The
window is one value per scanner and rises when any channel has a gate that
needs longer: 350 ms with CTCSS, 600 ms with DCS, 500 ms with a P25 NAC
filter.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: The Squelch That Measured the Whole Span]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
