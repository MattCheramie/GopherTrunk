---
slug: numerically-controlled-oscillator
title: Numerically controlled oscillator (NCO)
entry_type: term
category: sdr-dsp
description: A numerically controlled oscillator (NCO) generates a digital sine of programmable frequency from a phase accumulator — the tunable mixer inside an SDR's down-converter.
keywords: NCO, numerically controlled oscillator, phase accumulator, DDS, digital mixing, frequency synthesis, phase truncation, tuning word
aka: [NCO, "numerically controlled oscillator", DDS]
autolink: true
infobox:
  - { label: Type, value: Digital oscillator / DSP block }
  - { label: Generates, value: Programmable-frequency sine & cosine }
  - { label: Core, value: Phase accumulator + sine lookup }
see_also: [digital-down-converter, local-oscillator, decimation, iq-data, cordic, dds-synthesizer, phase-noise]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Numerically-controlled_oscillator
  - https://en.wikipedia.org/wiki/Direct_digital_synthesis
---

A **numerically controlled oscillator** (**NCO**) generates a digital sine and cosine
of any programmable frequency from a **phase accumulator** — a counter that adds a fixed
step each sample and looks up the corresponding sine value.[^wiki] It is the software
equivalent of a [local oscillator](/reference/local-oscillator/) and the tunable mixer
inside a [digital down-converter](/reference/digital-down-converter/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A phase accumulator adding a step each sample, feeding a sine lookup that outputs a digital tone." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="44" width="120" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="90" y="58">phase accumulator</text><text x="90" y="70" font-size="7.5">+ step each sample</text>
    <rect x="190" y="44" width="90" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="235" y="64">sine LUT</text>
    <line x1="150" y1="60" x2="189" y2="60" stroke="currentColor" marker-end="url(#ncoar)"/>
    <line x1="280" y1="60" x2="320" y2="60" stroke="currentColor" marker-end="url(#ncoar)"/>
  </g>
  <path d="M325 60 q8 -16 16 0 t16 0 t16 0 t16 0 t16 0 t16 0 t8 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <defs><marker id="ncoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An NCO steps a phase accumulator each sample and looks up the sine — a precisely tunable digital oscillator.</figcaption>
</figure>

## How it works

The heart of an NCO is an *N*-bit **phase accumulator**. Each sample clock, a constant **tuning
word** (the phase increment) is added to it; the accumulator wraps around modulo 2^N, and its
value represents the current phase of the oscillator as a fraction of a full turn. The high bits
of the accumulator index a **sine lookup table** (or drive a [CORDIC](/reference/cordic/)
rotation) to produce the instantaneous amplitude. Taking both sine and cosine gives the complex
exponential needed to mix a signal.

The output frequency follows a simple relation: *f_out = (tuning word / 2^N) × f_sample*. Two
properties fall straight out of this:

- **Fine resolution.** With a wide accumulator (32 or 48 bits), the smallest frequency step is
  tiny — sub-hertz at any practical sample rate — so an NCO can be tuned essentially
  continuously.
- **Instant, phase-continuous retuning.** Changing the tuning word takes effect on the next
  sample without resetting the accumulator, so the phase never jumps. This clean retune is why
  NCOs are used for hopping and channelised tuning.

## In practice

Real NCOs use a lookup table indexed by only the *top* bits of the accumulator, discarding the
rest. This **phase truncation** introduces small periodic spurs in the spectrum — the digital
analogue of [phase noise](/reference/phase-noise/) — whose level is set by the number of address
bits and the table's amplitude quantisation. Wider tables and techniques like phase dithering or
Taylor-series correction push these spurs down. A pure-software NCO (as in GopherTrunk) can
compute the sine directly in floating point and largely sidesteps the truncation-spur problem
that constrains fixed-point hardware [DDS](/reference/dds-synthesizer/) chips.

## Relevance to SDR

Changing the accumulator's step instantly retunes the NCO, which is how an SDR **digitally
tunes** to a channel: multiply the [IQ](/reference/iq-data/) stream by the NCO's complex output
to shift the wanted channel down to [baseband](/reference/baseband/) before low-pass filtering
and [decimation](/reference/decimation/). In GopherTrunk's down-converters an NCO is exactly this
tuning mixer — one per channel in the wideband path — translating each control- or voice-channel
frequency to zero IF so the demodulator can work on it. The same accumulator idea also underlies
hardware [DDS synthesizers](/reference/dds-synthesizer/) used as signal sources.

## Sources

[^wiki]: [Numerically-controlled oscillator](https://en.wikipedia.org/wiki/Numerically-controlled_oscillator) — Wikipedia, on phase-accumulator-based digital frequency synthesis.
[^dds]: [Direct digital synthesis](https://en.wikipedia.org/wiki/Direct_digital_synthesis) — Wikipedia, on the tuning-word/accumulator/lookup architecture and phase-truncation spurs.
