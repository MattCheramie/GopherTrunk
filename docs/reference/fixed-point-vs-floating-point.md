---
slug: fixed-point-vs-floating-point
title: Fixed-point vs floating-point DSP
entry_type: concept
category: sdr-programming
description: "Fixed-point vs floating-point DSP is the choice between integer-scaled and floating arithmetic for signal processing, trading dynamic range against cost and power."
keywords: fixed-point DSP, floating-point DSP, integer DSP, Q format, dynamic range, quantization, MCU DSP, FPGA DSP, float32, scaling, overflow, headroom
aka: [fixed point, floating point, integer vs float DSP]
autolink: true
infobox:
  - { label: Type, value: DSP numeric-representation choice }
  - { label: Fixed-point, value: Integer + implied scale; cheap, small range }
  - { label: Floating-point, value: Mantissa + exponent; wide range, costlier }
see_also: [numerical-precision-dsp, quantization, embedded-sdr, field-programmable-gate-array, microcontroller]
cite_urls:
  - https://en.wikipedia.org/wiki/Fixed-point_arithmetic
  - https://en.wikipedia.org/wiki/Floating-point_arithmetic
---

**Fixed-point vs floating-point DSP** is the decision of how to represent sample values
numerically inside a signal-processing chain: as scaled integers with an implied binary point
(fixed-point), or as sign-mantissa-exponent numbers that track their own scale
(floating-point).[^fx][^fl] The choice sets the achievable [dynamic range](/reference/dynamic-range/),
the risk of overflow, and the silicon, power, and effort cost — which is why it splits along
hardware lines, with tiny embedded targets favouring fixed-point and PC-class
[SDR](/reference/software-defined-radio/) favouring float.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Fixed-point uses evenly spaced integer levels giving a fixed window of dynamic range, while floating-point spaces levels proportionally to magnitude giving a far wider range at the cost of more complex hardware." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="18" font-size="9.5" fill="currentColor">fixed-point: evenly spaced levels, fixed window</text>
  <line x1="20" y1="40" x2="300" y2="40" stroke="currentColor"/>
  <g stroke="currentColor"><line x1="40" y1="35" x2="40" y2="45"/><line x1="72" y1="35" x2="72" y2="45"/><line x1="104" y1="35" x2="104" y2="45"/><line x1="136" y1="35" x2="136" y2="45"/><line x1="168" y1="35" x2="168" y2="45"/><line x1="200" y1="35" x2="200" y2="45"/><line x1="232" y1="35" x2="232" y2="45"/><line x1="264" y1="35" x2="264" y2="45"/></g>
  <text x="330" y="43" font-size="8" fill="currentColor">integer + scale</text>
  <text x="20" y="92" font-size="9.5" fill="currentColor">floating-point: proportional spacing, wide range</text>
  <line x1="20" y1="114" x2="300" y2="114" stroke="currentColor"/>
  <g stroke="currentColor"><line x1="40" y1="109" x2="40" y2="119"/><line x1="52" y1="109" x2="52" y2="119"/><line x1="70" y1="109" x2="70" y2="119"/><line x1="96" y1="109" x2="96" y2="119"/><line x1="132" y1="109" x2="132" y2="119"/><line x1="180" y1="109" x2="180" y2="119"/><line x1="244" y1="109" x2="244" y2="119"/></g>
  <text x="330" y="117" font-size="8" fill="currentColor">mantissa × 2^exp</text>
  <text x="20" y="150" font-size="8.5" fill="currentColor" fill-opacity="0.75">float keeps relative precision across many decades; fixed-point keeps absolute step size within one window.</text>
</svg>
<figcaption>Fixed-point places quantization levels at a uniform absolute spacing over a bounded window; floating-point spaces them proportionally to magnitude, covering a vastly larger range with roughly constant relative precision.</figcaption>
</figure>

## How it works

A **fixed-point** value is an integer paired with an agreed scale — a "Q15" number, for
example, treats a 16-bit signed integer as a fraction between −1 and just under +1 with 15
bits after the binary point. Arithmetic is plain integer arithmetic, which is why it runs on
the cheapest hardware. The catch is that the programmer manages scale by hand: multiplying two
Q15 numbers yields a Q30 result that must be shifted back, sums can overflow the window and
must be saturated or pre-scaled, and every filter needs headroom analysis so intermediate
values never wrap. Get it wrong and the signal clips or the low bits vanish into
[quantization](/reference/quantization/) noise.

A **floating-point** value carries its own exponent, so it automatically rescales: a float32
(IEEE 754 single precision) spans roughly ±10^38 with about 24 bits (~7 digits) of relative
precision anywhere in that range. The programmer stops worrying about overflow and headroom in
normal use and just writes the maths. The cost is that floating-point units are larger, hotter,
and historically slower than integer units, and float samples take more memory bandwidth than
tightly packed integers.

The essential trade:

- **Dynamic range.** Float wins enormously — it holds tiny and huge values in the same buffer
  without manual rescaling. Fixed-point offers a fixed window (~6 dB per bit) that the designer
  must position correctly.
- **Cost / power / speed.** Fixed-point wins on small or massively parallel hardware where
  integer units are cheap and float units expensive or absent.
- **Developer effort.** Float is far easier and less bug-prone; fixed-point demands explicit
  scaling, saturation, and overflow reasoning throughout the chain.

## In practice

The split follows the hardware. Low-power [microcontrollers](/reference/microcontroller/) and
DSP chips, and [FPGAs](/reference/field-programmable-gate-array/) where every multiplier is
silicon you pay for, are overwhelmingly fixed-point — an [embedded SDR](/reference/embedded-sdr/)
decimator or filter is typically built in Q-format integers with carefully budgeted headroom.
PC- and phone-class processors have fast floating-point units, so desktop SDR software is
almost always written in float32, which removes a whole category of scaling bugs. A common
pattern is hybrid: fixed-point in the FPGA close to the ADC where rates are highest, converting
to float once samples reach the host CPU. The [numerical precision](/reference/numerical-precision-dsp/)
question — how much error accumulates through a long chain — sits directly on top of this
choice, since fixed-point rounding at every stage compounds differently from float rounding.

## Relevance to SDR

Real SDR hardware straddles the boundary: the RTL-SDR, Airspy, and HackRF deliver 8- to
14-bit integer samples straight off the ADC, and any FPGA or on-chip decimation runs
fixed-point, but virtually every host-side SDR framework converts to float32 the moment
samples arrive because CPUs handle float effortlessly and the code stays clean. **GopherTrunk**
follows exactly that convention: it ingests integer sample formats from the device or capture
file, converts to floating-point early, and does its down-conversion, filtering, and
demodulation in float — so it never has to reason about Q-format headroom or saturation in its
decode chain. Understanding the trade still matters when writing SDR software, because it tells
you why an FPGA design and a Go or C++ host program make opposite representation choices, and
where the cost lives when you push DSP down toward the sensor.

## Sources

[^fx]: [Fixed-point arithmetic](https://en.wikipedia.org/wiki/Fixed-point_arithmetic) — Wikipedia, on integer-scaled representation, Q formats, and overflow/scaling management.
[^fl]: [Floating-point arithmetic](https://en.wikipedia.org/wiki/Floating-point_arithmetic) — Wikipedia, on IEEE 754 representation, dynamic range, and relative precision.
