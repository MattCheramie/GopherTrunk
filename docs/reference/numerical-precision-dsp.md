---
slug: numerical-precision-dsp
title: Numerical precision in DSP
entry_type: concept
category: sdr-programming
description: "Numerical precision in DSP is how many bits represent each sample and how rounding error accumulates through a chain, deciding when float32 suffices over float64."
keywords: numerical precision DSP, float32 vs float64, single precision, double precision, rounding error, error accumulation, catastrophic cancellation, IIR precision, FFT precision
aka: [numerical precision, floating-point precision, DSP precision]
autolink: true
infobox:
  - { label: Type, value: DSP accuracy consideration }
  - { label: Idea, value: Bit width vs accumulated rounding error }
  - { label: Trade, value: "float32 speed/memory vs float64 accuracy" }
see_also: [fixed-point-vs-floating-point, quantization, fast-fourier-transform, dynamic-range, iir-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Floating-point_arithmetic
  - https://en.wikipedia.org/wiki/Round-off_error
---

**Numerical precision in DSP** is the question of how many bits represent each sample value
and how the tiny rounding errors of finite-precision arithmetic build up as a signal passes
through a processing chain.[^round] Even in floating-point, every multiply and add rounds its
result slightly, and across the millions of operations in a [SDR](/reference/software-defined-radio/)
receiver those errors can accumulate — so choosing between **float32** (single precision) and
**float64** (double precision) is a real engineering decision, not a formality.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A true signal value and its float32 representation diverge slowly as rounding error accumulates over successive operations, while float64's much smaller error stays negligible for far longer." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="30" y1="20" x2="30" y2="120" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="30" y="14" font-size="8" fill="currentColor">error</text>
  <text x="400" y="134" font-size="8" fill="currentColor">operations →</text>
  <path d="M30 118 Q 200 112 440 60" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="360" y="52" font-size="8" fill="currentColor">float32</text>
  <path d="M30 119 Q 240 118 440 112" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="360" y="108" font-size="8" fill="currentColor">float64</text>
  <text x="120" y="40" font-size="8" fill="currentColor" fill-opacity="0.75">error grows with operation count and conditioning</text>
</svg>
<figcaption>Rounding error accumulates as a signal traverses more operations; float32 diverges from the true value far sooner than float64, though for many DSP chains it never diverges enough to matter.</figcaption>
</figure>

## How it works

A float32 carries about 24 bits of mantissa — roughly 7 significant decimal digits and around
140 dB of relative precision. A float64 carries 53 bits — about 16 digits. For a single
operation both are wildly more accurate than the underlying samples, which arrive from an ADC
with only 8 to 16 real bits ([quantization](/reference/quantization/) already dominates the
error there). Precision loss becomes visible only when errors *accumulate* or *cancel*:

- **Long summations.** Adding millions of terms — an FFT butterfly network, a long
  correlation, an integrator — lets rounding errors pile up. Naive accumulation in float32 can
  drift; techniques like Kahan summation or accumulating in float64 recover the lost bits.
- **Recursive filters.** An [IIR filter](/reference/iir-filter/) feeds its own output back, so
  rounding error recirculates and, near an unstable pole, can grow into limit cycles or
  outright instability. High-Q recursive structures are the classic case where float32 is not
  enough and float64 (or careful biquad cascades) is required.
- **Catastrophic cancellation.** Subtracting two nearly equal large numbers annihilates the
  significant digits, leaving mostly rounding noise — a hazard in poorly conditioned formulas.

The countervailing pressure is cost. Float32 uses half the memory and half the memory
bandwidth of float64, and on SIMD hardware a vector register holds twice as many float32
lanes, so single precision is roughly twice as fast in bandwidth-bound DSP. The craft is
using float32 wherever the error budget allows and reaching for float64 only at the few stages
that need it.

## In practice

The default in SDR software is float32, and it is almost always right: sample data is already
low-precision, and the streaming, mostly-FIR arithmetic of a receive chain does not accumulate
error dangerously. Engineers selectively upgrade the sensitive spots — accumulate long FFTs or
correlations in double, implement high-Q [IIR](/reference/iir-filter/) sections carefully or in
float64, and watch any recursive loop or large summation. This is a different axis from the
[fixed-point-versus-floating-point](/reference/fixed-point-vs-floating-point/) choice: that
decides integer versus float; this decides *how wide* the float. On a large
[FFT](/reference/fast-fourier-transform/), for example, both questions appear at once — whether
to compute in fixed or float, and if float, whether single precision holds enough dynamic range
across the transform.

## Relevance to SDR

Practically every host-side SDR framework computes in float32, from GNU Radio's default sample
type to the kernels in liquid-dsp and VOLK, precisely because it is fast enough and accurate
enough for streaming radio DSP while halving memory traffic. **GopherTrunk** follows the same
convention: it converts incoming integer samples to float32 and runs its down-conversion,
filtering, and demodulation in single precision, which is entirely adequate because the input
already carries far fewer real bits than a float32 mantissa and the FIR-dominated chain does
not accumulate error the way a high-Q recursive filter would. Understanding numerical precision
still matters when writing SDR software: it tells you why float32 is the sane default, and it
flags the handful of places — long integrators, tight feedback loops, big transforms — where
you must check the error budget or reach for double precision rather than assume the numbers
are exact.

## Sources

[^round]: [Round-off error](https://en.wikipedia.org/wiki/Round-off_error) — Wikipedia, on finite-precision rounding, error accumulation, and catastrophic cancellation.
