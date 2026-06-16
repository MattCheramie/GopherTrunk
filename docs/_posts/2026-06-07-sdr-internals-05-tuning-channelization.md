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
