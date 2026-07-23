---
slug: dsp-in-gophertrunk
title: DSP in GopherTrunk
description: A walkthrough of GopherTrunk's real signal path — capture, downconvert, filter, demodulate, recover symbols — mapping each stage to the DSP concepts you just learned.
keywords: gophertrunk dsp, sdr pipeline, ddc, downconverter, c4fm decode, real dsp example, sdr signal chain, dsp in practice
level: advanced
status: full
prereq:
  - decimation-and-resampling
  - demodulation
  - clock-and-symbol-recovery
faq:
  - q: Does GopherTrunk use fixed-point or floating-point DSP?
    a: GopherTrunk is pure Go and carries I/Q samples as complex64 — a pair of 32-bit floats — throughout its pipeline. Floating point keeps the DSP code simple and accurate, and modern CPUs run 32-bit float math fast enough for the channel rates involved. The next lesson explains the fixed-versus-float tradeoff in general.
  - q: Why does GopherTrunk have two different downconverters?
    a: The single-channel Downconverter is used by the replay/tune path to decode one channel from a capture, while the wideband DDCBank extracts many channels at once from a live stream. They are separate code paths that implement the same conceptual mix-filter-decimate DDC, so a fix to one does not automatically apply to the other — a distinction the project documents carefully.
---

# DSP in GopherTrunk

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
GopherTrunk's decode path is every concept in this module, in order: **capture**
I/Q, **downconvert** (mix + filter + resample) to the channel rate, **demodulate**,
then **recover symbols** into bits. Because the DDC normalizes to a fixed **channel
rate**, the decoder is **rate-invariant** — it behaves the same whatever the capture
rate. Samples flow as Go **`complex64`** over channels between concurrent stages.
</div>

This is where it all comes together. We'll walk GopherTrunk's real signal chain and
label each stage with the lesson that explains it — so the pipeline stops being
abstract and becomes something you can read in the source.

## The chain, end to end

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Pipeline: capture, downconvert (mix filter decimate), demodulate, symbol and clock recovery, framing, decoded output." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
  <rect x="4" y="35" width="66" height="30" rx="4" fill="none" stroke="currentColor"/><text x="37" y="53">capture</text>
  <rect x="78" y="35" width="96" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.8"/><text x="126" y="49">downconvert</text><text x="126" y="60">mix·filter·decim</text>
  <rect x="182" y="35" width="80" height="30" rx="4" fill="none" stroke="currentColor"/><text x="222" y="53">demodulate</text>
  <rect x="270" y="35" width="96" height="30" rx="4" fill="none" stroke="currentColor"/><text x="318" y="49">symbol/clock</text><text x="318" y="60">recovery</text>
  <rect x="374" y="35" width="60" height="30" rx="4" fill="none" stroke="currentColor"/><text x="404" y="53">framing</text>
  <rect x="442" y="35" width="74" height="30" rx="4" fill="none" stroke="currentColor"/><text x="479" y="49">decoded</text><text x="479" y="60">voice/data</text>
  </g>
  <g stroke="currentColor"><line x1="70" y1="50" x2="78" y2="50"/><line x1="174" y1="50" x2="182" y2="50"/><line x1="262" y1="50" x2="270" y2="50"/><line x1="366" y1="50" x2="374" y2="50"/><line x1="434" y1="50" x2="442" y2="50"/></g>
</svg>
<figcaption>GopherTrunk's decode path — each box is a lesson from this module.</figcaption>
</figure>

## Stage by stage

**1. Capture — I/Q in.** The SDR front-end (the `sdr` package) reads samples off the
radio and pushes them downstream as a stream of `complex64` — the
[I/Q samples](/learn/dsp/complex-signals-and-iq/) from Unit 1. Concurrent stages hand
this stream along over Go channels, exactly the pattern from the
[Go concurrency lessons](/learn/programming-go/channels/).

**2. Downconvert — mix, filter, decimate.** A **digital downconverter** centres the
target channel at zero with an [NCO mix](/learn/dsp/mixing-and-downconversion/), runs a
[FIR low-pass](/learn/dsp/fir-filters/) (a Kaiser-windowed design with a stopband more
than 60 dB down — the [windowing](/learn/dsp/windows-and-leakage/) from Unit 2), and
[resamples](/learn/dsp/decimation-and-resampling/) to the per-protocol channel rate —
**48 kHz** for the 4800-baud C4FM family, **144 kHz** for TETRA. GopherTrunk has two
DDC implementations: a single-channel `Downconverter` (used by the replay/tune path)
and a multi-tap wideband `DDCBank` (many channels from one live capture). They're
**separate code paths** doing the same conceptual job — a distinction the project
documents deliberately, because a fix to one does not touch the other.

**3. Demodulate.** The channel, now at baseband and at the channel rate, is
[FM/C4FM demodulated](/learn/dsp/demodulation/) — the phase-change discriminator that
turns I/Q into a signal stepping between symbol levels.

**4. Symbol & clock recovery.** A [matched filter and timing loop](/learn/dsp/clock-and-symbol-recovery/)
lock onto the 4800-baud symbol clock and read each symbol at its centre, with an
[AGC](/learn/dsp/gain-and-agc/) holding the level steady. Out come bits.

**5. Framing and decode.** The `scanner`/`radio` packages assemble bits into frames,
follow the control channel, and hand voice frames to the vocoder — the boundary where
DSP ends and the [digital trunking](/learn/digital-trunking/) module picks up.

## The rate-invariance payoff

Notice what the downconverter guarantees: **whatever** the capture rate — 2.4 MS/s,
10 MS/s — everything after step 2 sees the *same* channel rate. The receiver, matched
filter, and AGC are all sized from that channel rate, so the decoder is
**rate-invariant**. This isn't a detail; it's a load-bearing design fact. It means a
signal that decodes at one capture rate but not another points at the *captured data*
(front-end overload, phase noise) rather than the steady-state DSP — a diagnosis the
project has used to run down real field bugs.

## Reading it in the source

With this map, the code is navigable. Following the
[reading-real-Go](/learn/programming-go/reading-real-go/) habits: start where samples
enter, follow the `complex64` channel from `sdr` into the downconverter, then into the
demodulator and symbol recovery. Each package is one box in the diagram above.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the DDC normalizes to a fixed channel rate, making the decoder rate-invariant." markdown="0">
  <p class="knowledge-check__q">Quick check: why does GopherTrunk's decoder behave the same regardless of capture rate?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It always captures at exactly one rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The downconverter resamples every channel to a fixed channel rate before decoding</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses fixed-point math</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk's chain is this module in order: **capture → downconvert → demodulate →
  recover symbols → frame**.
- The **DDC** (mix + FIR + resample) normalizes to a fixed **channel rate** (48 kHz
  C4FM, 144 kHz TETRA).
- That makes the decoder **rate-invariant** — a powerful diagnostic property.
- Two DDCs exist — single-channel and wideband — as **separate code paths**; samples
  flow as **`complex64`** over Go channels.

Next up: the last lesson — how numbers are stored, and why it matters for performance.
