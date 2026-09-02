---
title: "P25 End to End, Part 11: Wideband P25 — Watching the Whole System at Once"
description: One wide capture, many control channels — the tuner-bank architecture behind role wideband, per-call voice taps carved from the same IQ stream, and the twin down-converter paths (the wideband DDCBank vs the single-channel Downconverter) whose drift let the fix for issue 764 miss the replay symptom in issue 771.
category: deep-dives
keywords: wideband p25 monitoring, sdr channelizer, ddc bank, polyphase channelizer, monitor multiple p25 sites one sdr, p25 voice taps, sdr pre-decimation input sample rate, wideband trunking scanner, gophertrunk p25
tags: [p25-end-to-end, p25, wideband, dsp, channelizer, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 11
---

*Part 11 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
built the site map and roamed it one control channel at a time. This part
removes the "one at a time": pin a single SDR across a band and every site
inside its IQ window decodes simultaneously, out of one capture. It is also
where the series' running thread lands hardest — the wideband tuner bank
and the single-channel down-converter are **twin paths**, and the day a fix
landed on one and not the other is the reason this repo has an
issue-closing policy.*

> **TL;DR:** `role: wideband` pins one dongle to a band centre; a
> `tuner.Bank` (`internal/dsp/tuner/`) extracts a 48 kHz narrowband stream
> per configured channel, and `internal/scanner/widebandt2` runs a
> per-channel P25 (or DMR/NXDN) state machine on each — multi-site P25
> means listing each site's CC as its own tap, decoded **in parallel, not
> hunted**. Voice grants ride the same capture via per-call DDC taps
> (`internal/sdr/wbvoice`, `voice_taps: N`). Two Bank implementations
> exist (`DDCBank` per-tap NCO+resampler, `ChannelizerBank` polyphase;
> `tuner_strategy: auto` picks by tap count) — and the wideband `DDCBank`
> is a **different code path** from the single-channel
> `ccdecoder.Downconverter` that `replay -tune-hz` and the hunting daemon
> use. A fix to one does not touch the other: that is how the
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) "fix"
> missed the [#771](https://github.com/MattCheramie/GopherTrunk/issues/771)
> replay symptom. `sdr.input_sample_rate` adds a systemwide pre-decimation
> stage when the front end must run faster than you decode.

**Key takeaways**

- **Parallel beats serial for multi-site.** A hunted single channel visits
  sites one at a time (Part 10); a wideband bank watches every CC in the
  IQ window at once, and grants can be voice-tapped from the same samples
  without retuning anything.
- **Two banks, one interface — and a second twin underneath.** `DDCBank`
  and `ChannelizerBank` implement one `Bank` contract; below them, the
  wideband bank and the single-channel `Downconverter` are parallel DDC
  implementations that have provably drifted apart before.
- **Symbol clocks come from `OutputRateHz()`, not the nominal target.** A
  pathological input rate can land the tap a fraction of a percent off
  48 kHz (issue #550) — a "baud drift" that looks exactly like a demod bug.
- **A wide capture spends a shared budget.** One antenna, one gain, one
  ADC across every tap (issue #749) — and raising the native rate to cover
  more band can *lower* in-channel quality (#764). The levers are
  `gain: "auto"` and `sdr.input_sample_rate`, not more megasamples.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Wideband engine | per-channel receivers + state machines on one SDR | `internal/scanner/widebandt2` (`Engine`) |
| Bank contract | AddTap / Process / OutputRateHz per tap | `internal/dsp/tuner/tuner.go` (`Bank`) |
| Per-tap DDC bank | one NCO mixer + rational resampler per tap | `internal/dsp/tuner/ddc.go` (`DDCBank`) |
| Polyphase bank | shared channelizer + fine-tune DDC, many taps | `internal/dsp/tuner/channelizerbank.go` (`ChannelizerBank`) |
| Voice taps | per-call virtual tuner on the wideband stream | `internal/sdr/wbvoice` (`VirtualTuner`, `voice_taps`) |
| The single-channel twin | live hunting daemon + `replay -tune-hz` DDC | `internal/scanner/ccdecoder/ddc.go` (`Downconverter`) |
| Pre-decimation lever | native rate → decode rate at the Device boundary | `sdr.input_sample_rate` (`internal/sdr/decimate.Device`) |

## In this post

- **One capture, many channels** — the wideband engine and multi-site P25 config.
- **The bank** — two implementations of one contract, and the #764 CPU story.
- **Voice without retuning** — per-call DDC taps as virtual tuners.
- **The twin pair that made a fix miss** — #764, #771, and the drift rules.
- **What a wide capture costs** — shared gain, CPU, and the honest limits.

## One capture, many channels

A 2.4 MS/s dongle sees roughly ±1.08 MHz of usable band after guard — on
an 800 MHz system, easily several sites' control channels plus most of
their voice channels. `role: wideband` pins the dongle to a centre
frequency and lists channels, each referencing a `trunking.systems` entry
whose protocol picks the state machine — `p25` runs the Phase 1 TSBK chain
([Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})),
`p25-phase2` the MAC chain
([Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})),
and DMR/NXDN mix freely on the same dongle. The per-channel receivers are
the same ones the rest of this series met — the twin discipline starts
with reusing them, not forking them.

For **multi-site P25**, each site's control channel becomes its own
`channels:` entry pointing at the same system. Every tap decodes in
parallel out of the one capture — they are *not* hunted one at a time — so
the [Part 10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
site tracker fills in minutes instead of sessions.
[The Hunt Part 9]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
tells this story from the discovery side. Sites of one system can even
differ in modulation: a per-channel `p25_phase1_demod_mode` override
(issue #935) lets a genuinely-linear LSM site decode on the CQPSK path
([Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }}))
while its C4FM siblings keep the default — and the config docs warn, in
bold, not to set `cqpsk` just because a site is *simulcast*: simulcast is a
transmitter-coordination technique, not a modulation.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A wideband capture drawn as a spectrum with four carrier humps inside a 2.4 megasample IQ window. Three DDC taps drop from carriers to parallel 48 kilohertz decoders: two P25 control-channel state machines for site 1 and site 2 and one DMR channel. A fourth, dashed tap labelled per-call voice tap is allocated on a voice grant and feeds the composer. All decoders publish onto one events bus.">
  <line x1="30" y1="70" x2="650" y2="70" stroke="var(--fg-muted)"/>
  <text x="340" y="18" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one wideband capture — centre 852.0 MHz, 2.4 MS/s (±1.08 MHz usable)</text>
  <path d="M60 70 Q85 30 110 70" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M200 70 Q225 26 250 70" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M370 70 Q395 34 420 70" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M520 70 Q545 38 570 70" fill="none" stroke="var(--accent)" stroke-width="2" stroke-dasharray="5 3"/>
  <text x="85" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site 1 CC</text>
  <text x="225" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site 2 CC</text>
  <text x="395" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DMR carrier</text>
  <text x="545" y="84" text-anchor="middle" fill="var(--accent)" font-size="9">voice grant</text>
  <line x1="85" y1="90" x2="85" y2="130" stroke="currentColor"/><polygon points="81,128 85,136 89,128" fill="currentColor"/>
  <line x1="225" y1="90" x2="225" y2="130" stroke="currentColor"/><polygon points="221,128 225,136 229,128" fill="currentColor"/>
  <line x1="395" y1="90" x2="395" y2="130" stroke="currentColor"/><polygon points="391,128 395,136 399,128" fill="currentColor"/>
  <line x1="545" y1="90" x2="545" y2="130" stroke="var(--accent)" stroke-dasharray="5 3"/><polygon points="541,128 545,136 549,128" fill="var(--accent)"/>
  <text x="340" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tuner.Bank — one 48 kHz narrowband stream per tap (DDCBank or ChannelizerBank)</text>
  <rect x="30" y="136" width="110" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="153" text-anchor="middle" fill="currentColor" font-size="9">P25 P1 receiver</text>
  <text x="85" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TSBK chain, site 1</text>
  <rect x="170" y="136" width="110" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="225" y="153" text-anchor="middle" fill="currentColor" font-size="9">P25 P1 receiver</text>
  <text x="225" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TSBK chain, site 2</text>
  <rect x="340" y="136" width="110" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="395" y="153" text-anchor="middle" fill="currentColor" font-size="9">DMR receiver</text>
  <text x="395" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Tier II/III chain</text>
  <rect x="490" y="136" width="110" height="42" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="5 3"/>
  <text x="545" y="153" text-anchor="middle" fill="var(--accent)" font-size="9">wbvoice tap</text>
  <text x="545" y="167" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per-call, → composer</text>
  <line x1="85" y1="178" x2="300" y2="210" stroke="var(--fg-muted)"/>
  <line x1="225" y1="178" x2="320" y2="210" stroke="var(--fg-muted)"/>
  <line x1="395" y1="178" x2="360" y2="210" stroke="var(--fg-muted)"/>
  <line x1="545" y1="178" x2="380" y2="210" stroke="var(--fg-muted)"/>
  <rect x="240" y="210" width="200" height="28" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="228" text-anchor="middle" fill="var(--accent)" font-size="10">events bus → engine, grants, calls</text>
</svg>
<figcaption>Every carrier in the IQ window gets its own 48 kHz stream and its own state machine — control channels permanently, voice channels per call.</figcaption>
</figure>

## The bank

The channelizer contract is small and both implementations honour it:

```go
// internal/dsp/tuner/tuner.go (shape)
type Bank interface {
    AddTap(offsetHz float64, sink SinkFunc) error
    Process(src []complex64) // one wideband chunk in, every sink fed
    InputRateHz() float64
    // The rate the bank ACTUALLY emits — may differ a fraction of a
    // percent from the nominal target (issue #550). Symbol clocks must
    // be built from this value, not the nominal target.
    OutputRateHz() float64
    Reset()
}
```

`DDCBank` runs one NCO mixer plus rational polyphase resampler per tap —
O(taps × samples), simple and exact. `ChannelizerBank` amortises the
anti-alias work across taps with a shared polyphase filter bank plus a
fine-tune DDC per channel, which wins when tap counts grow;
`tuner_strategy: auto` picks `ddc` up to six channels and `polyphase`
above ([SDR Internals Part 5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})
covers the DSP). That `OutputRateHz` comment in the contract is a scar: a
non-standard input rate can reduce to a pathological L/M ratio, and the old
fallback silently shifted the achieved rate — a 3.019 MS/s capture landed a
144 kHz tap at 143762 Hz, −0.165% of symbol rate, the "baud drift"
signature. The bounded `bestRatioUnderCaps` search (issue #550) lands
143998 Hz instead.

`DDCBank` also carries the CPU half of the #764 story. At 10 MS/s the old
bank ran a 208:1 single-stage decimation *per tap*, which starved the live
wideband pump goroutine until samples dropped at the hardware layer. The
fix is a **shared** integer pre-decimation stage that runs once over the
wideband stream, bringing it down to the ~2.5 MS/s regime the per-tap
resamplers were tuned for — and below that floor the path is deliberately
byte-for-byte identical to the pre-#764 behaviour, so the fix could not
silently change working configs.

## Voice without retuning

A control-channel tap never moves; voice grants land wherever the band plan
puts them. `internal/sdr/wbvoice` closes that gap: a `VirtualTuner`
implements the same interfaces as a physical voice SDR
(`SetCenterFreq` for the
[voice pool]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }}),
`StreamIQ` for the composer) but its `SetCenterFreq` allocates a per-call
DDC tap on the wideband stream instead of touching hardware. It emits
exactly 48 kHz (`wbvoice.NarrowbandRateHz`), so the composer's decimator
collapses to a no-op. `voice_taps: N` sets how many concurrent calls one
dongle carries; a grant outside the IQ window (minus a 5% guard) returns
`ErrOutOfBand`, which the engine treats as "wrong tuner for this grant" and
routes to the next free device — so one wideband dongle plus one physical
voice SDR covers both the in-window common case and the out-of-window
stragglers ([wideband-voice-taps]({{ '/reference/wideband-voice-taps/' | relative_url }})
has the operator view).

## The twin pair that made a fix miss

Now the hard lesson. GopherTrunk has **two** wideband-to-narrowband
implementations, and they are not wired together:

| | `ccdecoder.Downconverter` | `tuner.DDCBank` |
|---|---|---|
| Taps | one channel | many |
| Used by | hunting daemon's CC path, `replay -tune-hz`, siglab | `role: wideband`, wbvoice, hunt sweeps |
| Rate handling | interpolates *up* too (sub-rate captures) | shared pre-decimation + per-tap resample |
| File | `internal/scanner/ccdecoder/ddc.go` | `internal/dsp/tuner/ddc.go` |

When issue #764 ("P25 decodes at 2.5 MS/s but not 10 MS/s") was first
"fixed", the fix landed in the wideband `DDCBank`. The reporter's replay
symptom lived in the single-channel `Downconverter` — a path the fix never
touched — so the issue was closed twice while the symptom was still live,
which is issue #771 and the origin of this repo's issue-closing policy
(**never close until a failing-first regression passes and the reporter
confirms**). The full postmortem chain is
[Ten Megasamples]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
and the
[Two Pipelines finale]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }});
the ending is worth restating because it changed the mental model. The
decode path is **rate-invariant**: both down-converters normalise to the
per-protocol channel rate
([Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})'s
48 kHz), and `ddc_highrate_test.go` pins that a noisy channel reaches the
receiver at the same in-channel SNR whether decoded natively at 10 MS/s or
decimated to 2.5 MS/s. The reporter's own captures then closed #764
honestly: the 10 MS/s file replayed at ≈9.5 dB demod SNR against the
2.5 MS/s file's ≈19.7 dB, an *independent* resampler reproduced the same
deficit, and neither capture clipped — the ~10 dB was **baked into the
samples** (front-end phase noise at the Airspy's native 10 MS/s clock),
not GT's DSP. The decoder can only be as good as the samples.

The twins have since been actively converged where it counts: the #550
ratio fallback was ported into the `Downconverter` explicitly to close a
"two separate DDC paths" divergence, and both files now say so in
comments. But the structural rule stands — when you fix anything in one
down-converter, go read the other one.

## What a wide capture costs

Three budgets, all shared across every tap:

- **RF budget** (issue #749): one antenna, one centre, one gain. A fixed
  gain chosen for the strongest site leaves weak co-tenants flat at the
  ADC floor; a hot site can desensitise everything. Prefer `gain: "auto"`
  on multi-site dongles — the daemon logs a startup WARN for a fixed-gain
  multi-tap config — and watch
  `gophertrunk_sdr_wideband_input_clip_ratio`. A genuinely weak distant
  site may deserve its own dongle.
- **CPU budget**: every tap is a full receiver. When decode falls behind,
  the honest signals are the `ccdecoder: decode can't keep up with real
  time` WARN and, on network SDRs, `soapyremote` overrun drops — a
  *downstream* symptom, not a driver bug. `sdr.input_sample_rate` is the
  systemwide lever: the hardware runs at its happy native rate (an Airspy
  pinned to 10 MS/s) while `decimate.Device` integer-decimates to
  `sdr.sample_rate` at the Device boundary, before the bank, the demods,
  and every recording tap. It must be an exact integer multiple, and it
  survives device reacquisition because it wraps the Device rather than
  threading a second rate through the daemon.
- **Physics budget**: the lever above is a load and recording-size fix,
  *not* an RF fix — decimating 10 → 2.5 MS/s does not recover quality lost
  to native-clock phase noise (#764 again;
  [The Analog Edge Part 6]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }})
  is the operator's guide to choosing rates).

## Where this goes next

Wideband is P25 monitoring at its most parallel — and it still inherits
the weakest link of the series:
[Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})
faces the honest gap, the default C4FM voice path's missing equalizer and
soft FEC, why the fix is gated on a real capture, and what contributing one
looks like.

## FAQ

**Can one SDR really monitor a whole P25 system?**
If the system's channels fit the dongle's usable IQ window (sample rate
minus a 5% guard per edge), yes: every site's CC as a wideband tap, voice
via `voice_taps`, with a physical voice SDR as fallback for out-of-window
grants. Systems spread across more spectrum than one dongle sees need a
second dongle — each `widebandt2.Engine` owns one SDR and they share only
the bus.

**Why does only one site decode while its siblings sit at the noise floor?**
The channelizer is gain-flat across taps, so that difference is real RF: a
weak or distant site at this shared centre and gain, or a strong site
overloading the shared ADC. Check the input clip ratio first — if it's
non-zero, *lower* the gain or add attenuation; otherwise try `gain: "auto"`
(issue #749), and give a genuinely weak site its own dongle.

**Should I raise the sample rate to cover more channels?**
Only as far as the front end stays clean. More megasamples cost CPU per
tap and, on some hardware, in-channel quality — the #764 capture pair
measured ~10 dB *worse* demod SNR at the Airspy's native 10 MS/s than at
2.5 MS/s, baked into the samples. If the hardware must run fast, set
`sdr.input_sample_rate` and decode at a lower `sdr.sample_rate`.

**What's the difference between `tuner_strategy: ddc` and `polyphase`?**
Cost shape. `ddc` (`DDCBank`) pays per tap — simple and exact, fine to ~6
channels. `polyphase` (`ChannelizerBank`) pays mostly once in a shared
filter bank, winning as tap counts grow. `auto` picks by channel count;
both honour the same `Bank` contract, so decoders can't tell them apart.

**Why did the daemon decode a capture that `replay` wouldn't (or vice versa)?**
Historically: because they channelize through different code (`DDCBank` vs
`ccdecoder.Downconverter`) and a fix had landed on one side only — the
#764/#771 story. Today the paths are pinned by rate-invariance tests and
share the ratio math, but a daemon/replay disagreement is always worth
filing: it is a twin-path drift detector by construction.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Sites, WACNs & Roaming a Multi-Site System]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
· Next →
[Part 12: The Weak-Signal Gap — P1 Voice's Missing Levers]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})
