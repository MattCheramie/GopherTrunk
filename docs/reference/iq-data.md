---
slug: iq-data
title: IQ data
entry_type: term
category: sdr-dsp
description: IQ data is the stream of paired in-phase (I) and quadrature (Q) samples an SDR produces, capturing both the amplitude and phase of a signal and enabling negative frequencies.
keywords: IQ data, in-phase quadrature, complex samples, amplitude phase, baseband, constellation, analytic signal
aka: [IQ data, IQ samples]
autolink: true
infobox:
  - { label: Type, value: Complex sample stream }
  - { label: Components, value: I (in-phase), Q (quadrature, 90°) }
  - { label: Encodes, value: Amplitude (radius) + phase (angle) }
see_also: [software-defined-radio, phase, amplitude, constellation-diagram, sample-rate, iq-modulation, iq-imbalance]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/rf-sdr/iq-data/ }
related_reading:
  - { title: "SDR Internals, Part 1: What is software-defined radio?", url: /blog/deep-dives/sdr-internals-01-what-is-software-defined-radio/ }
cite_urls:
  - https://en.wikipedia.org/wiki/In-phase_and_quadrature_components
  - https://en.wikipedia.org/wiki/Analytic_signal
---

**IQ data** is the stream of paired numbers an SDR produces — **I** (in-phase) and **Q**
(quadrature, 90° apart). Together each pair captures both the
[amplitude](/reference/amplitude/) and [phase](/reference/phase/) of the signal at that
instant, so a single complex sample fully describes the wave rather than just its height.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 260 220" role="img" aria-label="The IQ plane with I horizontal and Q vertical; an arrow to a point shows amplitude as length and phase as angle." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="240" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="130" y1="20" x2="130" y2="200" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="232" y="124" font-size="11" fill="currentColor">I</text><text x="116" y="30" font-size="11" fill="currentColor">Q</text>
  <line x1="130" y1="110" x2="195" y2="55" stroke="currentColor" stroke-width="2"/><circle cx="195" cy="55" r="4" fill="currentColor"/>
  <path d="M165 110 A 36 36 0 0 0 150 84" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="150" y="104" font-size="10" fill="currentColor">phase</text>
  <text x="150" y="70" font-size="10" fill="currentColor" transform="rotate(-40 165 80)">amplitude</text>
</svg>
<figcaption>Each IQ sample is a point on the complex plane: distance from the origin is amplitude, angle is phase.</figcaption>
</figure>

## How it works

The receiver generates two copies of a [local-oscillator](/reference/local-oscillator/)
tone 90° apart — a cosine and a sine — and multiplies the incoming signal by each. The
cosine product is **I**, the sine product is **Q**. Written as a complex number *I + jQ*,
each sample is a point on the complex plane: its distance from the origin is amplitude and
its angle is phase, which is exactly what a
[constellation diagram](/reference/constellation-diagram/) plots. This representation is a
sampled *analytic signal* — a signal with no negative-frequency mirror — which is the
mathematical reason a single real channel cannot do the same job.[^analytic]

The decisive property IQ buys you is the ability to tell apart frequencies **above** and
**below** the tuned centre. A real-only sample cannot: a tone 10 kHz above the local
oscillator and a tone 10 kHz below both beat at 10 kHz and are indistinguishable. With two
quadrature channels the *direction* of rotation on the IQ plane is preserved, so positive
and negative offsets are separated. That is why an SDR can present a symmetric band around
its tuned frequency and why the [captured bandwidth](/reference/sample-rate/) can straddle
zero.

## Variants

- **Baseband (zero-IF) IQ** — the signal is centred at 0 Hz. This is what most
  direct-conversion SDRs emit; the trade-off is a DC spike and any
  [IQ imbalance](/reference/iq-imbalance/) landing right in the middle of the band.
- **[Low-IF](/reference/low-if/) IQ** — the band is offset slightly so the DC artefact
  sits outside the channel of interest.
- **Real (interleaved) samples from [direct sampling](/reference/direct-sampling/)** — a
  single ADC stream that DSP later converts to analytic IQ with a
  [Hilbert transform](/reference/hilbert-transform/).

## In practice

On disk and on the wire, IQ is just interleaved I, Q, I, Q values — 8-bit unsigned for
RTL-SDR, 16-bit signed or 32-bit float for higher-end radios (the `.cfile` GopherTrunk
replays is complex float32). The [sample rate](/reference/sample-rate/) sets how many pairs
per second arrive; the bit depth sets the [dynamic range](/reference/dynamic-range/). A
persistent DC offset or gain mismatch between the I and Q paths shows as a fixed point or a
tilted, elliptical constellation, which downstream equalisers must correct.

## Relevance to SDR

Everything GopherTrunk does begins with the IQ stream from the radio: tuning, filtering,
demodulation, the constellation and spectrum scopes, and the offline replay path all
operate on it. The demodulator recovers symbols by tracking the phase and amplitude of
successive IQ samples.

## Sources

[^wiki]: [In-phase and quadrature components](https://en.wikipedia.org/wiki/In-phase_and_quadrature_components) — Wikipedia, on representing a signal as paired I and Q components.
[^analytic]: [Analytic signal](https://en.wikipedia.org/wiki/Analytic_signal) — Wikipedia, on the complex-valued signal with no negative-frequency content that IQ sampling approximates.
