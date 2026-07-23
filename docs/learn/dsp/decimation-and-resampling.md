---
slug: decimation-and-resampling
title: Decimation & resampling
description: Changing a signal's sample rate without introducing aliasing — decimation, interpolation, and rational resampling — how an SDR narrows a wideband capture down to one channel's rate.
keywords: decimation, resampling, interpolation, sample rate conversion, downsampling, polyphase filter, anti-aliasing, dsp resampling
level: intermediate
status: full
prereq:
  - fir-filters
faq:
  - q: What is decimation?
    a: Decimation lowers a signal's sample rate by keeping only every Nth sample — but you must low-pass filter first, or high frequencies alias down and corrupt the result. So decimation is really filter-then-drop-samples. Reducing the rate early, once you've isolated a narrow channel, makes all downstream processing far cheaper.
  - q: Why not just capture at the low rate you need?
    a: The radio has to sample fast enough to cover the whole band you're tuned across (Nyquist), which is much wider than any single channel. Once DSP isolates one narrow channel, that wide rate is wasteful, so you decimate down to just what the channel needs — often from megahertz to tens of kilohertz — before demodulating.
---

# Decimation & resampling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Decimation** lowers the sample rate by keeping every Nth sample — but you must
**low-pass filter first** or high frequencies **alias** in. **Interpolation** raises
the rate. Combining them gives **rational resampling** (by a fraction). An SDR
decimates a wide capture down to one channel's rate — from megahertz to tens of
kilohertz — so the rest of the pipeline is cheap.
</div>

An SDR captures far more bandwidth than any single channel needs. This lesson is about
shedding that excess safely, so downstream DSP runs on a manageable stream.

## Why change the rate at all

The radio must sample fast enough to cover the whole band you're watching — say
2.4 MS/s across 2.4 MHz, as [Nyquist](/learn/dsp/sampling-and-quantization/) demands.
But once you've [mixed](/learn/dsp/mixing-and-downconversion/) and
[filtered](/learn/dsp/fir-filters/) down to one 12.5 kHz channel, carrying millions of
samples a second is pure waste. Everything after — demodulation, symbol recovery — gets
cheaper the lower the rate. So you **decimate**: reduce the rate to just what the
channel needs.

## Decimation = filter, then drop

The naive idea is "keep every Nth sample, throw the rest away." Done alone, that's a
bug: any energy above the *new* Nyquist limit folds down and **aliases** into your
channel, exactly the corruption from the sampling lesson. So decimation is always two
steps:

```text
1. low-pass filter  -> remove everything above the new Nyquist limit
2. keep every Nth sample -> the rate is now 1/N
```

The filter is the **anti-aliasing** guard, and it's usually the same FIR that isolated
the channel — one filter doing two jobs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="A fast sample stream enters an anti-alias low-pass filter, then a downsample-by-N block, and exits as a slower stream." xmlns="http://www.w3.org/2000/svg">
  <text x="55" y="30" text-anchor="middle" font-size="9" fill="currentColor">2.4 MS/s</text>
  <line x1="10" y1="55" x2="90" y2="55" stroke="currentColor" stroke-width="1.5"/>
  <rect x="90" y="38" width="110" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="145" y="59" text-anchor="middle" font-size="10" fill="currentColor">low-pass (FIR)</text>
  <line x1="200" y1="55" x2="240" y2="55" stroke="currentColor" stroke-width="1.5"/>
  <rect x="240" y="38" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="285" y="59" text-anchor="middle" font-size="11" fill="currentColor">&#8595; by N</text>
  <line x1="330" y1="55" x2="410" y2="55" stroke="currentColor" stroke-width="1.5"/>
  <text x="460" y="30" text-anchor="middle" font-size="9" fill="currentColor">48 kS/s</text>
  <text x="440" y="59" text-anchor="middle" font-size="10" fill="currentColor">to demod</text>
</svg>
<figcaption>Decimation is filter-then-drop: the anti-alias filter must run <em>before</em> samples are discarded.</figcaption>
</figure>

## Interpolation and rational resampling

The reverse — **interpolation** — raises the rate by inserting samples between existing
ones (then filtering to smooth them in). Combine the two and you can resample by any
**rational** factor L/M: interpolate by L, decimate by M. That's how you convert an
awkward capture rate to a clean channel rate — for example turning 2.4 MS/s into
exactly 48 kS/s for a 4800-baud C4FM channel.

Done efficiently, the filtering and rate change are folded together into a
**polyphase** filter, which skips computing samples it's about to throw away.

## In a real receiver

This is precisely what a **digital downconverter (DDC)** does: mix the channel to
zero, low-pass filter it, and resample to the channel rate — all in one stage.
GopherTrunk normalizes every protocol to its channel rate this way (48 kHz for the
4800-baud C4FM family, 144 kHz for TETRA), which is why its decode path is
*rate-invariant*: no matter the capture rate, the demodulator always sees the same
channel rate. You'll see that end to end in
[DSP in GopherTrunk](/learn/dsp/dsp-in-gophertrunk/); the RF path frames the same idea
in [filtering & decimation](/learn/rf-sdr/filtering-decimation/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — you must low-pass filter before dropping samples, or aliasing corrupts the result." markdown="0">
  <p class="knowledge-check__q">Quick check: what must you do <em>before</em> keeping every Nth sample when decimating?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Raise the bit depth</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Low-pass filter to remove energy above the new Nyquist limit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Apply an FFT</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Decimation** lowers the rate by keeping every Nth sample — but **filter first** or
  suffer **aliasing**.
- **Interpolation** raises the rate; together they give **rational resampling** (L/M).
- Efficient implementations fold both into a **polyphase** filter.
- A **DDC** mixes, filters, and resamples a channel to its rate — making a decoder
  **rate-invariant**.

Next up: Unit 4 — tuning a channel to zero with a mixer.
