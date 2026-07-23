---
slug: the-decibel-in-dsp
title: Decibels & dynamic range
description: Why signal levels are measured in decibels, how dB math turns multiplication into addition, and what dynamic range means for a sampled digital signal.
keywords: decibel dsp, dB math, dBFS, dynamic range, bit depth dynamic range, 6 dB per bit, power ratio, amplitude ratio, log scale
level: beginner
status: full
prereq:
  - sampling-and-quantization
faq:
  - q: Why use decibels instead of plain ratios?
    a: "Signal strengths in radio span an enormous range — the strongest signal can be a trillion times more powerful than the weakest you can still decode. Decibels compress that range onto a manageable scale and turn multiplication into addition, so a gain of 100x becomes simply adding 20 dB. Both make signal-level bookkeeping far easier."
  - q: What is dBFS?
    a: "dBFS means decibels relative to full scale — the loudest level a sampled signal can represent before it clips. Full scale is 0 dBFS, and every real sample sits below it at a negative dBFS value. It is the natural yardstick for digital signals because it is referenced to the number format itself, not to any physical power."
---

# Decibels & dynamic range

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **decibel (dB)** is a logarithmic ratio: **10·log₁₀** of a *power* ratio, or
**20·log₁₀** of an *amplitude* ratio. Logs turn multiplication into **addition**, so
chained gains just add. For sampled signals we measure level in **dBFS** (relative to
full scale), and the **dynamic range** a format can hold is set by its **bit depth** —
roughly **6 dB per bit**.
</div>

Radio spans a colossal range of signal strengths, and DSP code juggles gains at every
stage. Decibels are how we keep that sane. This lesson builds on
[sampling & quantization](/learn/dsp/sampling-and-quantization/) and mirrors the RF
path's [decibels](/learn/rf-sdr/decibels/) lesson from the DSP side.

## Why a logarithmic scale

The weakest signal you can still decode and the strongest one the front end can handle
differ by factors of millions to trillions. Writing those as plain numbers is unwieldy,
and gains *multiply* as they chain through a receiver. Taking a logarithm fixes both:
it compresses a huge range into small numbers, and it turns each multiply into an **add**.

```text
gain of 100x  -> +20 dB
gain of 2x    -> +3 dB (approx)
loss of 1/2   -> -3 dB
chain them:   100x then 2x  ->  20 dB + 3 dB = 23 dB
```

## Power vs amplitude: the 10 and the 20

There are two forms of the dB formula, and the difference trips people up:

| Quantity | Formula | Because… |
|----------|---------|----------|
| **Power** ratio | dB = **10·log₁₀**(P₁/P₀) | power is the base definition |
| **Amplitude** ratio | dB = **20·log₁₀**(A₁/A₀) | power ∝ amplitude², and log of a square doubles the factor |

So doubling the *amplitude* of a signal is **+6 dB**, while doubling its *power* is
**+3 dB**. Both describe the same physical change viewed through different quantities.
A few values worth memorising:

```text
+3 dB  ~ 2x power        +6 dB  ~ 2x amplitude
+10 dB = 10x power       +20 dB = 10x amplitude / 100x power
 0 dB  = no change (ratio of 1)
```

## dBFS: the digital yardstick

For a signal that lives as numbers, the natural reference is the **largest value the
format can hold** — full scale. Level measured against that is **dBFS** (dB relative to
full scale). Full scale is **0 dBFS**; every real sample sits below it, at a negative
value. A signal peaking at −6 dBFS is using half the available amplitude; one that
reaches 0 dBFS is on the edge of **clipping**, where samples slam into the limit and the
waveform is destroyed — a failure mode covered in
[front-end overload](/learn/rf-sdr/front-end-and-overload/).

## Dynamic range and bit depth

**Dynamic range** is the span between the loudest signal a format can represent and the
quantization-noise floor beneath it. It is set directly by **bit depth**: each extra bit
doubles the number of levels, which is **~6 dB** more range.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A vertical bar from 0 dBFS at top down to a noise floor, with the span between labelled dynamic range and marked at roughly 6 dB per bit." xmlns="http://www.w3.org/2000/svg">
  <line x1="120" y1="20" x2="120" y2="130" stroke="currentColor" stroke-width="1.5"/>
  <line x1="112" y1="20" x2="128" y2="20" stroke="currentColor" stroke-width="2"/>
  <text x="140" y="24" font-size="11" fill="currentColor">0 dBFS (full scale — clipping above)</text>
  <line x1="112" y1="130" x2="128" y2="130" stroke="currentColor" stroke-width="2"/>
  <text x="140" y="133" font-size="11" fill="currentColor">quantization noise floor</text>
  <line x1="120" y1="24" x2="120" y2="126" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="60" y="78" font-size="10" fill="currentColor" transform="rotate(-90 60 78)">dynamic range</text>
  <text x="300" y="78" font-size="10" fill="currentColor">~6 dB per bit</text>
  <text x="300" y="94" font-size="9" fill="currentColor">8-bit ≈ 48 dB · 12-bit ≈ 72 dB · 16-bit ≈ 96 dB</text>
</svg>
<figcaption>Dynamic range is the span from full scale down to the noise floor — about 6 dB for every bit of depth.</figcaption>
</figure>

That is why an 8-bit SDR (about 48 dB) can be swamped by a strong nearby signal that a
12-bit unit (about 72 dB) handles, and why keeping the input well below 0 dBFS — but not
so low it sinks into the noise floor — is the goal of [gain staging](/learn/dsp/gain-and-agc/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a decibel is 10·log₁₀ of a power ratio (or 20·log₁₀ of an amplitude ratio)." markdown="0">
  <p class="knowledge-check__q">Quick check: how much does adding one bit of depth increase dynamic range?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">About 3 dB</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">About 6 dB</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">About 20 dB</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **decibel** is a log ratio: **10·log₁₀** of power, **20·log₁₀** of amplitude.
- Logs turn multiplied gains into **added** dB — the reason receiver budgets are done in dB.
- **dBFS** measures a sampled signal against full scale; 0 dBFS is the clipping edge.
- **Dynamic range** ≈ **6 dB per bit** of depth, from full scale down to the noise floor.

Next up: the single most useful way to look at a signal — its frequency content.
