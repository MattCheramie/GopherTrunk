---
slug: complex-signals-and-iq
title: Complex signals & I/Q
description: Why SDRs represent signals as complex I/Q pairs, what the imaginary part really means, and how negative frequency becomes a useful, everyday idea.
keywords: iq data, i/q samples, complex signal, in-phase quadrature, negative frequency, complex baseband, analytic signal, dsp iq
level: intermediate
status: full
prereq:
  - sampling-and-quantization
faq:
  - q: Why does an SDR give me two numbers per sample instead of one?
    a: The two numbers are the in-phase (I) and quadrature (Q) components — together they form one complex sample. Two numbers capture both the amplitude and the phase of the signal at that instant, which a single real number can't. This lets the radio tell a frequency above the tuned centre from one below it, something a single stream cannot do.
  - q: What is negative frequency, physically?
    a: With complex I/Q samples, frequency is measured relative to the tuned centre frequency. A component below the centre shows up as a negative frequency, one above as positive. It's not mystical — it just means "which side of centre, and how the phase rotates." The imaginary axis is what lets the two sides be told apart.
---

# Complex signals & I/Q

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An SDR delivers **two numbers per sample** — **I** (in-phase) and **Q**
(quadrature) — which together form one **complex** sample carrying both **amplitude
and phase**. This is what lets a receiver distinguish a signal **above** the tuned
frequency from one **below** it (positive vs **negative frequency**). I/Q is the
native language of software radio.
</div>

Open a raw SDR capture and you'll find pairs of numbers, not single values. This
lesson explains why — and why "complex" here means *useful*, not *complicated*.

## Two numbers, one sample

A single real number can tell you a wave's height right now, but not which way its
phase is turning. SDRs solve this by recording two measurements 90° apart:

- **I — in-phase**: the signal multiplied by a cosine.
- **Q — quadrature**: the signal multiplied by a sine (a quarter-cycle offset).

Treat the pair as a point on a plane — I across, Q up — and each sample becomes a
little arrow (a **complex number**). Its **length** is the signal's amplitude; the
**angle** is its phase; how fast the angle rotates from sample to sample is its
frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 260 160" role="img" aria-label="An I/Q plane with horizontal I axis and vertical Q axis, and an arrow from the origin whose length is amplitude and angle is phase." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="80" x2="240" y2="80" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="130" y1="10" x2="130" y2="150" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="232" y="74" font-size="11" fill="currentColor">I</text>
  <text x="136" y="20" font-size="11" fill="currentColor">Q</text>
  <line x1="130" y1="80" x2="200" y2="35" stroke="currentColor" stroke-width="2"/>
  <path d="M170 80 A 40 40 0 0 0 183 57" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/>
  <text x="176" y="72" font-size="10" fill="currentColor">phase</text>
  <text x="150" y="50" font-size="10" fill="currentColor">amplitude</text>
</svg>
<figcaption>An I/Q sample is an arrow on the complex plane: length = amplitude, angle = phase. A rotating arrow is a frequency.</figcaption>
</figure>

## Why single numbers aren't enough

Imagine tuning your radio to 851.000 MHz. A signal at 851.010 MHz and one at
850.990 MHz are both 10 kHz away from centre. With a single real number stream, they
look *identical* — you can't tell which side of centre a signal is on. With I/Q, the
first makes the arrow rotate one way and the second the other way. That difference is
what we call **positive** versus **negative frequency**: not a strange idea, just
"which side of the tuned centre."

## Complex baseband

Because I/Q measures everything relative to the tuned frequency, the signal is said
to be at **complex baseband** — centred on zero, with real signals spread on both
the positive and negative sides. Every operation later in this path —
[mixing](/learn/dsp/mixing-and-downconversion/) to retune,
[filtering](/learn/dsp/fir-filters/) to isolate a channel, the
[FFT](/learn/dsp/the-fft/) to see the spectrum — is arithmetic on these complex
samples. GopherTrunk carries them as Go `complex64` values, a pair of 32-bit floats,
all the way down its pipeline.

The RF path's [I/Q data](/learn/rf-sdr/iq-data/) lesson introduces the same idea from
the radio side; here we care that each sample is one complex number the math treats
as a whole.

## Reading a raw capture

A raw SDR file is just these pairs, interleaved: I, Q, I, Q, … If it's 8-bit, that's
two bytes per sample. Knowing this, you can see why a few seconds of capture at
2.4 MS/s is a large file — and why [decimating](/learn/dsp/decimation-and-resampling/)
to a narrow channel rate as early as possible keeps the rest of the pipeline fast.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the I/Q pair encodes phase, so the receiver can tell the two sides of centre apart." markdown="0">
  <p class="knowledge-check__q">Quick check: what can a complex I/Q stream do that a single real stream cannot?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Sample twice as fast</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Distinguish a frequency above the tuned centre from one below it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Eliminate quantization noise</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- SDRs give **two numbers per sample** — **I** and **Q** — forming one **complex**
  sample.
- The pair encodes **amplitude (length)** and **phase (angle)**; a rotating arrow is
  a frequency.
- I/Q lets the receiver tell **positive** from **negative** frequency — which side of
  centre a signal sits.
- Everything downstream is arithmetic on these **complex baseband** samples
  (`complex64` in GopherTrunk).

Next up: the single most useful way to look at a signal — the frequency domain.
