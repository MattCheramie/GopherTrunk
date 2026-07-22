---
title: "SDR in Pure Go, Part 5: Tuning & Channelization — DDC vs. Polyphase"
description: How GopherTrunk extracts many narrowband channels from one wideband SDR capture using a digital down-converter or a polyphase channelizer, selected at runtime through a Go Strategy interface.
category: deep-dives
tags: [sdr, go, dsp, channelizer, ddc, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 5
---

*Part 5 of **SDR Internals**. One dongle can watch megahertz of spectrum at once.
This post is about pulling many narrow channels out of that wide capture — and
the two strategies GopherTrunk picks between at runtime.*

## In this post

- Why **"one dongle, many channels"** matters for a trunking scanner.
- Two approaches: a per-channel **digital down-converter (DDC)** versus a shared
  **polyphase channelizer**.
- The **Strategy pattern** in Go: a `Bank` interface with two implementations,
  chosen at runtime.

## What channelization does

A trunked radio system spreads its control channel and voice channels across a
band. With a wideband SDR you can capture the whole band once and then tune
*inside the samples* to as many channels as you want — no extra hardware. That
"tune in software" step is a
[digital down-converter]({{ '/reference/digital-down-converter/' | relative_url }}):
shift the channel to baseband with an NCO, low-pass filter it, and
[decimate]({{ '/reference/decimation/' | relative_url }}) to a manageable rate
(~48 kHz, the symbol clock for 4800-baud protocols).

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="The digital down-converter pipeline for one channel: a wideband IQ capture at 2.4 mega-samples per second is shifted so the wanted channel sits at baseband by an NCO, low-pass filtered to reject the neighbors, then decimated by M, producing a narrowband channel at about 48 kilohertz, the symbol clock.">
  <rect x="8" y="52" width="110" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="63" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="10">wideband IQ</text>
  <text x="63" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2.4 MS/s</text>
  <line x1="118" y1="74" x2="132" y2="74" stroke="currentColor"/><polygon points="132,70 140,74 132,78" fill="currentColor"/>
  <rect x="142" y="52" width="100" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="192" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">NCO shift</text>
  <text x="192" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">channel → baseband</text>
  <line x1="242" y1="74" x2="256" y2="74" stroke="currentColor"/><polygon points="256,70 264,74 256,78" fill="currentColor"/>
  <rect x="266" y="52" width="100" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="316" y="72" text-anchor="middle" fill="currentColor" font-size="10">low-pass FIR</text>
  <text x="316" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reject neighbors</text>
  <line x1="366" y1="74" x2="380" y2="74" stroke="currentColor"/><polygon points="380,70 388,74 380,78" fill="currentColor"/>
  <rect x="390" y="52" width="100" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="440" y="72" text-anchor="middle" fill="currentColor" font-size="10">decimate ↓M</text>
  <text x="440" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">drop the rate</text>
  <line x1="490" y1="74" x2="504" y2="74" stroke="currentColor"/><polygon points="504,70 512,74 504,78" fill="currentColor"/>
  <rect x="514" y="52" width="140" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="584" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">narrowband channel</text>
  <text x="584" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~48 kHz symbol clock</text>
</svg>
<figcaption>A digital down-converter tunes "inside the samples": shift the wanted channel to baseband with an NCO, low-pass filter it, and decimate down to the ~48 kHz symbol clock — no extra hardware.</figcaption>
</figure>

There are two ways to do this for *many* channels at once, and they have
opposite cost profiles:

- **DDC bank** — one NCO + resampler per channel. Cost grows linearly with the
  number of taps, but the channels can sit at *any* offset.
- **Polyphase channelizer** — a single filter bank splits the whole band into M
  evenly-spaced bins in one shared operation. Cheap per channel, but the
  channels must lie on the bin grid.
  ([reference]({{ '/reference/channelizer/' | relative_url }}))

## How GopherTrunk implements it in Go

`internal/dsp/tuner` defines the abstraction and `internal/dsp/channelizer`
provides the heavy machinery. The key is a single interface — a `Bank` — with
two interchangeable implementations:

```go
// internal/dsp/tuner — the strategy contract (shape)
type Bank interface {
    // Process consumes one wideband chunk and writes the
    // narrowband output for every configured tap.
    Process(wide []complex64) [][]complex64
    Taps() []float64 // tuned offsets, in Hz
}
```

- **`DDCBank`** instantiates one NCO + polyphase resampler per requested offset.
  Linear cost, arbitrary spacing — ideal when you only need a handful of taps at
  awkward frequencies.
- **`ChannelizerBank`** runs the wideband stream through one polyphase
  decomposition + FFT rotation, emitting one sample per channel for every M
  input samples, then fine-tunes each bin with a small DDC. Shared cost,
  grid-aligned — ideal when you want many evenly-spaced channels (for example, a
  P25 Phase 1 control channel and its voice channels on one dongle).

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="One wideband chunk of complex64 samples enters a Bank, whose interface has two interchangeable implementations — DDCBank for arbitrary offsets and ChannelizerBank for grid-aligned channels. The Bank emits several narrowband outputs at once: a P25 Phase 1 control channel and three voice channels, each around 48 kHz, all from the single capture.">
  <rect x="8" y="84" width="110" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="63" y="104" text-anchor="middle" fill="currentColor" font-size="10">wideband chunk</text>
  <text x="63" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">[]complex64</text>
  <line x1="118" y1="106" x2="160" y2="106" stroke="currentColor"/><polygon points="160,102 168,106 160,110" fill="currentColor"/>
  <rect x="170" y="40" width="180" height="132" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="260" y="56" text-anchor="middle" fill="var(--accent)" font-size="10">Bank.Process()</text>
  <rect x="182" y="66" width="156" height="42" rx="5" fill="none" stroke="currentColor"/>
  <text x="260" y="84" text-anchor="middle" fill="currentColor" font-size="10">DDCBank</text>
  <text x="260" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="8">any offset</text>
  <rect x="182" y="116" width="156" height="42" rx="5" fill="none" stroke="currentColor"/>
  <text x="260" y="134" text-anchor="middle" fill="currentColor" font-size="10">ChannelizerBank</text>
  <text x="260" y="148" text-anchor="middle" fill="var(--fg-muted)" font-size="8">grid-aligned</text>
  <line x1="350" y1="92" x2="422" y2="30" stroke="currentColor"/><polygon points="422,26 430,30 422,34" fill="currentColor"/>
  <line x1="350" y1="100" x2="422" y2="74" stroke="currentColor"/><polygon points="422,70 430,74 422,78" fill="currentColor"/>
  <line x1="350" y1="112" x2="422" y2="118" stroke="currentColor"/><polygon points="422,114 430,118 422,122" fill="currentColor"/>
  <line x1="350" y1="120" x2="422" y2="162" stroke="currentColor"/><polygon points="422,158 430,162 422,166" fill="currentColor"/>
  <rect x="430" y="14" width="210" height="32" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="535" y="30" text-anchor="middle" fill="var(--accent)" font-size="10">control channel</text>
  <text x="535" y="41" text-anchor="middle" fill="var(--fg-muted)" font-size="8">P25 Phase 1</text>
  <rect x="430" y="58" width="210" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="535" y="74" text-anchor="middle" fill="currentColor" font-size="10">voice ch A</text>
  <text x="535" y="85" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~48 kHz</text>
  <rect x="430" y="102" width="210" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="535" y="118" text-anchor="middle" fill="currentColor" font-size="10">voice ch B</text>
  <text x="535" y="129" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~48 kHz</text>
  <rect x="430" y="146" width="210" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="535" y="162" text-anchor="middle" fill="currentColor" font-size="10">voice ch C</text>
  <text x="535" y="173" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~48 kHz</text>
</svg>
<figcaption>One capture, many channels: the <code>Bank</code> interface fronts two strategies — <code>DDCBank</code> for irregular offsets, <code>ChannelizerBank</code> for a regular grid — and pulls a control channel and its voice channels out of a single wideband chunk.</figcaption>
</figure>

The caller — the wideband voice tuner in `internal/sdr/wbvoice`, or the control
decoder — picks the implementation based on how many channels it needs and how
they're spaced, then uses only the `Bank` interface.

## The design principle: the Strategy pattern

DDC-per-channel and the polyphase channelizer solve the *same* problem with
different trade-offs. That is the definition of the **Strategy pattern**:
encapsulate interchangeable algorithms behind a common interface and choose
between them at runtime.

### How that principle shaped the Go code

- **One interface, two algorithms.** Both banks satisfy `Bank`, so the
  surrounding code is written once against the interface. Swapping strategies is
  a constructor choice, not a rewrite.
- **The decision is data-driven.** Number of taps and their spacing — not a
  compile-time flag — decide which bank to build. Even channel spacing favors the
  channelizer; sparse, irregular taps favor the DDC bank.
- **Go interfaces keep it implicit.** Neither bank declares "I implement
  `Bank`"; they just have the right methods. That structural typing makes it
  trivial to add a third strategy later (say, an FFT-overlap-save filterbank)
  without touching the callers.
- **Composition over inheritance.** Each bank is *built from* the Part 4
  primitives — NCOs, resamplers, polyphase filters — rather than extending a base
  class. The strategy is assembled, not inherited.

## Where this goes next

Channelization is what makes GopherTrunk's "monitor a whole system on one SDR"
feature possible. A future deep dive can compare DDC and channelizer CPU cost as
the tap count climbs, and explain the FFT-rotation math at the channelizer's
core. Next up, the channels we just carved out get demodulated.

## FAQ

**When should I use a channelizer instead of per-channel DDCs?**
When you need many channels and they fall on a regular grid (most trunked
systems do). The channelizer amortizes one big filter across all of them. For a
few channels at irregular offsets, separate DDCs are simpler and just as fast.

**Does channelization need a wideband SDR?**
It needs the SDR's sample rate to cover all the channels you want at once. An
RTL-SDR at 2.4 MS/s spans ~2 MHz of usable bandwidth — enough for many trunked
systems' control-plus-voice spread.

**Why decimate to ~48 kHz?**
Most digital voice protocols run at 4800 symbols/s; ~48 kHz gives a clean
integer oversampling ratio for the matched filter and timing recovery stages
that follow.

## Series navigation

**Part 5 of 14** · ←
[Part 4]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }})
· Next →
[Part 6: Demodulation]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
