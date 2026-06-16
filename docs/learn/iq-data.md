---
slug: iq-data
title: IQ data & complex signals
description: IQ data explained for SDR beginners — what In-phase (I) and Quadrature (Q) samples are, why radios output pairs of numbers, how IQ captures both amplitude and phase, and how the complex plane connects directly to the constellation diagram.
keywords: IQ data, IQ samples, in-phase quadrature, complex signal, IQ explained, amplitude and phase, complex plane, negative frequency, SDR baseband
level: intermediate
status: full
prereq:
  - what-is-sdr
  - digital-modulation
faq:
  - q: What is IQ data in an SDR?
    a: IQ data is the stream of paired numbers an SDR produces, where I is the "in-phase" component and Q is the "quadrature" component (90 degrees out of phase). Together each I/Q pair captures both the amplitude and the phase of the signal at that instant, which is everything needed to represent any modulation. Plotting I horizontally and Q vertically gives the complex plane used by constellation diagrams.
  - q: Why does an SDR output two numbers per sample instead of one?
    a: A single amplitude reading can't tell you phase — whether the wave is rising or falling, or which way it's rotating. By sampling two components 90 degrees apart (I and Q), the receiver captures amplitude and phase together, and can even distinguish frequencies above and below the tuned centre (positive vs negative frequencies). One number is ambiguous; the pair is complete.
  - q: What do I and Q stand for?
    a: I stands for In-phase and Q for Quadrature. They are two versions of the signal sampled 90 degrees apart in phase. Mathematically they're the real and imaginary parts of a complex number, which is why IQ samples are often called complex samples.
  - q: How does IQ relate to the constellation diagram?
    a: A constellation diagram is literally a plot of IQ samples — I on the horizontal axis, Q on the vertical. Each symbol's amplitude is its distance from the centre and its phase is its angle. So the constellation you see in GopherTrunk is a direct picture of the IQ data coming off your SDR.
gophertrunk_links:
  - title: Constellation
    url: /constellation.html
    note: a live plot of the IQ samples themselves.
  - title: Architecture
    url: /architecture.html
    note: how GopherTrunk moves IQ through its pipeline.
---

# IQ data & complex signals

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
SDRs output **IQ data** — pairs of numbers, **I** (in-phase) and **Q** (quadrature,
90° apart). Together each pair captures the signal's **amplitude** *and* **phase** at
that instant, which is everything needed to represent any modulation. Plot **I
horizontally and Q vertically** and you get the **complex plane** — exactly the
[constellation diagram](/learn/digital-modulation/). IQ also lets the receiver tell
frequencies *above* the tuned centre from those *below* (positive vs. negative
frequencies). One number is ambiguous; the **pair is complete**.
</div>

IQ is the language an SDR speaks, and it's the concept that ties
[modulation](/learn/digital-modulation/), the [constellation](/constellation.html), and
[demodulation](/learn/demodulation-pipeline/) together. It looks abstract for about
five minutes, then it clicks.

## Why isn't one number per sample enough?

Imagine sampling a wave's height and getting "0.5." Is the wave rising or falling? Is
it a 1 kHz tone above your tuned frequency or 1 kHz below? A single amplitude reading
can't say — it's **ambiguous**. To fully describe a wave at an instant you need two
things: how *strong* it is (amplitude) and *where in its cycle* it is (phase). One
number gives you only one of those.

## I and Q, defined

The fix is to sample the signal **twice, 90° apart in phase**:

- **I — In-phase:** the signal measured against a reference cosine.
- **Q — Quadrature:** the same signal measured against a reference sine (90° shifted).

The [quadrature-sampling receiver](/learn/sdr-receiver/) produces these two channels
naturally when it mixes the signal down to baseband. Mathematically, I and Q are the
**real and imaginary parts of a complex number**, which is why IQ samples are also
called *complex samples* — but you don't need the math to use them.

## How I and Q encode amplitude and phase

Here's the payoff. Treat each sample as a point with coordinates (I, Q):

- Its **distance from the origin** is the **amplitude** (how strong).
- Its **angle** around the origin is the **phase** (where in the cycle).

<figure class="figure" markdown="0">
<svg viewBox="0 0 260 220" role="img" aria-label="The complex IQ plane: I on the horizontal axis, Q on the vertical. A point is shown with an arrow from the origin; its length is amplitude and its angle is phase." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="240" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="130" y1="20" x2="130" y2="200" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="232" y="124" font-size="11" fill="currentColor">I</text>
  <text x="116" y="30" font-size="11" fill="currentColor">Q</text>
  <line x1="130" y1="110" x2="195" y2="55" stroke="currentColor" stroke-width="2"/>
  <circle cx="195" cy="55" r="4" fill="currentColor"/>
  <path d="M160 110 A 36 36 0 0 0 150 84" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <text x="150" y="104" font-size="10" fill="currentColor">phase</text>
  <text x="150" y="70" font-size="10" fill="currentColor" transform="rotate(-40 165 80)">amplitude</text>
</svg>
<figcaption>An IQ sample as a point on the complex plane. Length = amplitude, angle = phase. A steady tone traces a circle; modulation moves the point to meaningful spots.</figcaption>
</figure>

A pure, steady tone makes this point **rotate** around the origin at a constant rate —
the rotation speed *is* the frequency. [Modulation](/learn/digital-modulation/) nudges
the point to specific places: amplitude modulation changes its distance, frequency
modulation changes its rotation rate, phase modulation jumps its angle.

To make it concrete, a few sample values and what they mean:

| I | Q | Amplitude (√(I²+Q²)) | Phase (angle) |
|---|---|----------------------|----------------|
| 1.0 | 0.0 | 1.0 | 0° (pointing right) |
| 0.0 | 1.0 | 1.0 | 90° (straight up) |
| 0.71 | 0.71 | 1.0 | 45° |
| 0.5 | 0.0 | 0.5 | 0°, but half as strong |

Notice rows 1–3 have the **same amplitude** but different **phase** — a single
real number couldn't tell them apart, yet the I/Q pair does. That's the whole reason
SDRs work in pairs.

## The constellation connection

If a point's position encodes amplitude and phase, then plotting the **symbols** of a
digital signal on the IQ plane shows exactly which state each one is — which is what a
**[constellation diagram](/learn/digital-modulation/)** is. The constellation you watch
in GopherTrunk's [Constellation panel](/constellation.html) is *literally a scatter
plot of IQ samples*. Tight clusters = clean IQ; smeared clusters = noisy IQ. Now you
know what's being plotted.

## Negative frequencies and why IQ allows them

Because IQ captures *direction of rotation*, the receiver can tell a signal **above**
the tuned centre frequency from one **below** it — "positive" vs. "negative"
frequencies relative to the centre. A single real number can't distinguish the two
(they'd look identical), but the I/Q pair can, because they rotate opposite ways. This
is why an SDR can show and process the **whole** captured band, both sides of the
centre, at once.

## How GopherTrunk uses IQ

Everything GopherTrunk does starts from the IQ stream off your SDR. It digitally
[tunes and filters](/learn/filtering-decimation/) within that IQ to isolate a channel,
[demodulates](/learn/demodulation-pipeline/) it by tracking amplitude/phase over time,
and renders the [constellation](/constellation.html) and other scopes straight from the
IQ so you can *see* signal quality. IQ isn't a detail of the hardware — it's the raw
material of the entire pipeline.

<div class="knowledge-check" data-quiz data-correct-msg="Right — distance from the origin is amplitude, angle is phase." markdown="0">
  <p class="knowledge-check__q">Quick check: on the IQ plane, what does a sample's <em>angle</em> represent?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Its amplitude</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its phase</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Its sample rate</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **IQ data** = paired **I** (in-phase) and **Q** (quadrature, 90° apart) samples.
- Together they capture **amplitude (distance)** and **phase (angle)** — the full state
  of the wave.
- The IQ plane *is* the **constellation diagram**.
- IQ distinguishes **positive vs. negative** frequencies, so the whole band is usable.
- GopherTrunk's entire pipeline runs on IQ.

Next: how much spectrum that IQ stream actually covers — sample rate, bandwidth, and Nyquist.
