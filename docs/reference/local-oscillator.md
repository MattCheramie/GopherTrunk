---
slug: local-oscillator
title: Local oscillator
entry_type: term
category: sdr-dsp
description: A local oscillator is a tunable reference signal mixed with the incoming signal to shift a chosen band to a lower frequency; its setting is what "tuning" actually changes.
keywords: local oscillator, LO, mixer, tuning, frequency reference, NCO, phase noise
aka: [local oscillator, LO]
autolink: true
infobox:
  - { label: Type, value: Reference signal source }
  - { label: Role, value: Sets the band shifted down by the mixer }
  - { label: Digital form, value: Numerically controlled oscillator }
see_also: [superheterodyne-receiver, digital-down-converter, numerically-controlled-oscillator, phase-noise, zero-if, image-frequency, frequency, ppm-frequency-correction]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Local_oscillator
  - https://en.wikipedia.org/wiki/Frequency_mixer
---

A **local oscillator** (**LO**) is a tunable reference signal mixed with the incoming signal
to shift a chosen band down toward baseband.[^wiki] **Tuning a receiver is just changing the
LO frequency** — everything else in the analog chain stays fixed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A mixer combining an incoming RF signal and a local-oscillator tone to produce a shifted lower frequency." xmlns="http://www.w3.org/2000/svg">
  <text x="30" y="45" font-size="9" fill="currentColor">RF in</text>
  <line x1="30" y1="55" x2="150" y2="55" stroke="currentColor" stroke-width="1.3"/>
  <circle cx="175" cy="55" r="22" fill="none" stroke="currentColor" stroke-width="1.4"/><path d="M160 40 L190 70 M190 40 L160 70" stroke="currentColor" stroke-width="1.2"/>
  <text x="175" y="100" font-size="9" fill="currentColor" text-anchor="middle">LO (tunable)</text><line x1="175" y1="92" x2="175" y2="78" stroke="currentColor"/>
  <line x1="197" y1="55" x2="320" y2="55" stroke="currentColor" stroke-width="1.3" marker-end="url(#loar)"/>
  <text x="330" y="51" font-size="9" fill="currentColor">shifted</text><text x="330" y="63" font-size="9" fill="currentColor">(IF/baseband)</text>
  <text x="120" y="30" font-size="9" fill="currentColor" text-anchor="middle">tuning = changing the LO</text>
  <defs><marker id="loar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The local oscillator sets the tuned frequency: the mixer shifts the chosen band down for digitising.</figcaption>
</figure>

## How it works

A [mixer](/reference/mixer-rf/) multiplies the incoming RF by the LO tone. Trigonometry says
the product of two sinusoids contains their sum and difference frequencies, so a station at
frequency *f* mixed with an LO at *f_LO* yields components at *f + f_LO* and *|f − f_LO|*.
The receiver keeps the difference and discards the sum. By moving *f_LO*, the operator slides
which slice of spectrum lands on the fixed
[intermediate frequency](/reference/intermediate-frequency/) or at baseband — that motion is
what a tuning knob or a frequency entry actually commands.

Two LO qualities dominate reception. **Accuracy**: if the LO sits slightly off its nominal
value the whole band lands off-centre, appearing as a
[PPM frequency error](/reference/ppm-frequency-correction/) the software must correct.
**Spectral purity**: a real oscillator is not a single clean line but a tone smeared by
[phase noise](/reference/phase-noise/). That skirt mixes nearby strong signals down onto the
wanted one (reciprocal mixing) and blurs the demodulated constellation — a carrier-clean but
modulation-degraded capture is the classic fingerprint of LO phase noise.

## Variants

- **Analog LO** — a crystal-referenced [PLL](/reference/phase-locked-loop/) synthesiser or
  [VCO](/reference/dds-synthesizer/) driving a hardware mixer, as in a superhet front-end.
- **Quadrature LO** — two copies 90° apart feeding an I and a Q mixer, producing complex
  [IQ](/reference/iq-data/) directly; the basis of [zero-IF](/reference/zero-if/)
  direct-conversion receivers.
- **[Numerically controlled oscillator](/reference/numerically-controlled-oscillator/)
  (NCO)** — the fully digital LO: software generates a rotating complex tone and multiplies
  it against the sample stream inside a
  [digital down-converter](/reference/digital-down-converter/), with none of the drift or
  phase noise of an analog part.

## In practice

An LO that is not locked to a stable reference drifts with temperature, which is why serious
receivers use a TCXO or an OCXO and why cheap dongles need periodic PPM calibration against a
known signal. The choice of LO frequency also sets where the
[image](/reference/image-frequency/) falls: high-side versus low-side injection place the
image on opposite sides of the band, one of which may be easier to filter. In a purely
digital second stage the NCO's "LO" is exact by construction, so residual tuning error is
entirely the analog front-end's.

## Relevance to SDR

The LO sets which part of the spectrum lands in the ADC's window, so its accuracy and
stability directly affect tuning and decode quality. GopherTrunk cannot fix a physically
noisy or drifting hardware LO in software, but it does perform the final, exact frequency
shift digitally with an NCO when it channelises each control and voice channel.

## Sources

[^wiki]: [Local oscillator](https://en.wikipedia.org/wiki/Local_oscillator) — Wikipedia, on the tunable reference tone a mixer uses to shift a band down.
[^mixer]: [Frequency mixer](https://en.wikipedia.org/wiki/Frequency_mixer) — Wikipedia, on how multiplying by the LO produces sum and difference frequencies.
