---
slug: cordic
title: CORDIC
entry_type: algorithm
category: algorithms
description: CORDIC computes sine, cosine, atan2, and magnitude using only shifts and adds, giving multiplier-free NCOs and FM discriminators in FPGA and hardware SDR front ends.
keywords: CORDIC, coordinate rotation digital computer, shift-add algorithm, sine cosine, atan2, magnitude phase, vectoring mode, rotation mode, NCO, FM discriminator, FPGA DSP, Jack Volder
aka: [CORDIC, coordinate rotation digital computer, Volder's algorithm]
autolink: true
infobox:
  - { label: Type, value: Iterative shift-add rotator }
  - { label: Computes, value: sin/cos, atan2, magnitude/phase }
  - { label: Cost, value: No multipliers — shifts and adds only }
see_also: [numerically-controlled-oscillator, quadrature-demodulation, field-programmable-gate-array, demodulation, iq-data]
cite_urls:
  - https://en.wikipedia.org/wiki/CORDIC
  - https://dl.acm.org/doi/10.1109/TEC.1959.5222693
---

**CORDIC** (COordinate Rotation DIgital Computer) computes trigonometric and related
functions — sine, cosine, `atan2`, vector magnitude and phase — using only **bit-shifts,
additions, and a small lookup table**, with no hardware multiplier.[^wiki][^volder] It works
by rotating a 2-D vector toward a target angle (or toward the x-axis) in a fixed sequence of
ever-smaller steps whose tangents are exact powers of two, so each rotation collapses to a
shift-and-add. That makes it the natural choice for [FPGA](/reference/field-programmable-gate-array/)
and fixed-point hardware where multipliers are scarce but shifts are free.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A vector is rotated toward a target angle in a sequence of successively smaller shift-and-add micro-rotations, converging on the desired sine and cosine coordinates." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="40" y1="130" x2="40" y2="20" stroke="currentColor" stroke-width="1"/>
    <line x1="40" y1="130" x2="250" y2="130" stroke="currentColor" stroke-width="1"/>
    <line x1="40" y1="130" x2="230" y2="40" stroke="currentColor" stroke-width="1.6" marker-end="url(#cdar)"/>
    <line x1="40" y1="130" x2="235" y2="95" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <line x1="40" y1="130" x2="215" y2="55" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <text x="150" y="30">target angle</text>
    <text x="120" y="112">&#177;atan(2^-i) steps</text>
    <text x="290" y="70">each step:</text>
    <text x="330" y="88">shift + add</text>
    <text x="330" y="104">(no multiply)</text>
    <text x="255" y="43">(cos, sin)</text>
  </g>
  <defs><marker id="cdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>CORDIC converges on a target angle through successively smaller micro-rotations, each of which is just a shift and an add — the final coordinates are cosine and sine.</figcaption>
</figure>

## How it works

A rotation by angle θ can be decomposed into a series of micro-rotations by fixed angles
`αᵢ = atan(2⁻ⁱ)`. Rotating by such an angle requires multiplying by `tan(αᵢ) = 2⁻ⁱ`, which is
simply a right-shift by *i* bits. At each iteration CORDIC decides the *sign* of the next
micro-rotation and updates the vector:

`xᵢ₊₁ = xᵢ ∓ (yᵢ >> i)`, `yᵢ₊₁ = yᵢ ± (xᵢ >> i)`,

while accumulating the angle from a tiny stored table of the `αᵢ`. Each step gains roughly one
bit of accuracy, so *n* iterations give ~*n* bits of precision. A fixed gain of about 1.647
(the product of the per-step scale factors) is corrected once by a constant. CORDIC runs in
two dual modes:

- **Rotation mode** drives the residual *angle* to zero: feed in θ and a unit vector, and the
  final `(x, y)` are `cos θ` and `sin θ`. This is how you synthesise a sinusoid.
- **Vectoring mode** drives the residual *y* to zero: feed in a vector `(I, Q)`, and the
  accumulated angle is `atan2(Q, I)` while the final `x` is the magnitude `√(I²+Q²)`. This is
  how you get phase and amplitude from a complex sample.

Because every step is a shift and an add, CORDIC pipelines beautifully in hardware — one stage
per iteration — delivering a new result every clock cycle. Hyperbolic and linear variants
extend the same idea to `sinh`/`cosh`, exponentials, logarithms, multiply, and divide.

## In practice

CORDIC trades multipliers for latency: it needs one iteration per bit rather than a single
multiply, so on a modern CPU with fast hardware multipliers a table or polynomial is usually
quicker. Its home turf is fixed-point hardware — FPGAs, ASICs, and small DSP cores — where a
multiplier is expensive silicon and a shift is nearly free, and where the fully-pipelined,
constant-throughput structure is exactly what a streaming signal path wants.

## Relevance to SDR

CORDIC is pervasive in radio hardware. A [numerically-controlled oscillator](/reference/numerically-controlled-oscillator/)
in an FPGA commonly generates its quadrature sine/cosine with a rotation-mode CORDIC instead
of a large sine ROM, and the digital down-converters and up-converters inside SDR chips and
transceivers use it to mix signals to and from baseband. In vectoring mode it is the standard
way to build an **FM discriminator** — the instantaneous phase of successive
[I/Q](/reference/iq-data/) samples, differenced, is the frequency — and to compute the
magnitude/phase needed for AM detection,
[quadrature demodulation](/reference/quadrature-demodulation/), and
[constellation](/reference/constellation-diagram/) angle measurement. Its multiplier-free
nature is why it shows up in the front ends of RTL-SDR-class silicon, Airspy/HackRF-adjacent
FPGA designs, and countless embedded radios.

GopherTrunk runs on general-purpose CPUs, so its own down-conversion and demodulation use
ordinary floating-point trig and complex arithmetic rather than a CORDIC core; the algorithm's
importance to GT's world is upstream, in the device hardware and FPGA front ends that deliver
clean I/Q for GT to decode. Knowing CORDIC explains how those multiplier-light front ends
generate their oscillators and discriminators.

## Sources

[^wiki]: [CORDIC](https://en.wikipedia.org/wiki/CORDIC) — Wikipedia, on the shift-add rotation algorithm, its rotation and vectoring modes, and hardware use.
[^volder]: [The CORDIC Trigonometric Computing Technique](https://dl.acm.org/doi/10.1109/TEC.1959.5222693) — J. E. Volder, IRE Transactions on Electronic Computers (1959), the original description of the algorithm.
