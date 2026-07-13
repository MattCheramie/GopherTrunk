---
slug: sinad
title: SINAD (signal-to-noise-and-distortion)
entry_type: term
category: rf-metrics
description: SINAD is the ratio of total signal-plus-noise-plus-distortion power to noise-plus-distortion power, the standard metric for analog receiver sensitivity and audio quality.
keywords: SINAD, signal to noise and distortion, receiver sensitivity, 12 dB SINAD, analog receiver, audio quality, THD+N, FM sensitivity
aka: [SINAD, signal-to-noise-and-distortion ratio]
autolink: true
infobox:
  - { label: Symbol, value: SINAD }
  - { label: Unit, value: Decibels (dB) }
  - { label: Formula, value: "(S+N+D) / (N+D)" }
see_also: [signal-to-noise-ratio, receiver-sensitivity, noise-figure, carrier-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/SINAD
---

**SINAD** (**signal-to-noise-and-distortion** ratio) is the ratio, in
[decibels](/reference/decibel/), of total output power — signal plus noise plus
distortion — to the power of the noise and distortion alone.[^sinad] It is the standard
way to specify **analog receiver sensitivity**, most familiarly as the "12 dB SINAD"
figure quoted for FM radios and scanners. Unlike a plain
[signal-to-noise ratio](/reference/signal-to-noise-ratio/), SINAD folds in harmonic and
intermodulation distortion, so it reflects the quality a listener actually hears.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A bar showing total output split into a large signal portion and a small noise-plus-distortion portion, with SINAD defined as their ratio in decibels." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="300" height="30" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <rect x="40" y="40" width="250" height="30" fill="currentColor" fill-opacity="0.18"/>
  <text x="165" y="60" text-anchor="middle" font-size="10" fill="currentColor">signal (S)</text>
  <rect x="290" y="40" width="50" height="30" fill="currentColor" fill-opacity="0.35"/>
  <text x="315" y="88" text-anchor="middle" font-size="9" fill="currentColor">N + D</text>
  <line x1="315" y1="70" x2="315" y2="80" stroke="currentColor"/>
  <text x="60" y="130" font-size="11" fill="currentColor">SINAD (dB) = 10·log₁₀ ( (S+N+D) / (N+D) )</text>
  <text x="60" y="152" font-size="9" fill="currentColor" fill-opacity="0.8">12 dB SINAD ≈ usable-audio sensitivity threshold</text>
</svg>
<figcaption>SINAD compares total output to the noise-and-distortion residue; the classic 12 dB SINAD point defines an analog receiver's usable sensitivity.</figcaption>
</figure>

## How it works

SINAD is measured at the receiver's audio output while feeding a modulated test signal —
conventionally a 1 kHz tone. The total output power is measured, then a sharp notch
filter removes the 1 kHz tone, leaving only noise and distortion; the ratio of the two
measurements, in dB, is SINAD. Because the denominator includes distortion as well as
noise, SINAD is always slightly worse than the pure SNR of the same signal, and it is
closely tied to total harmonic distortion plus noise (THD+N): SINAD ≈ −(THD+N) when
expressed as a power ratio.

**Receiver sensitivity** is then defined as the RF input level required to produce a
reference SINAD at the output. For analog FM land-mobile and broadcast gear the
reference is **12 dB SINAD**, roughly the point at which speech becomes comfortably
intelligible; for some standards **20 dB** (quieting) is used instead. A radio spec of
"0.25 µV for 12 dB SINAD" means it takes just a quarter-microvolt at the antenna to
reach that quality — a lower number is a more sensitive receiver.

## In practice

- 12 dB SINAD is the near-universal benchmark for comparing analog FM
  [receiver sensitivity](/reference/receiver-sensitivity/) across radios and scanners.
- SINAD is an *analog* metric, tied to recovered audio. Digital modes are instead graded
  by [bit error rate](/reference/bit-error-rate/) at a reference RF level, because their
  output is bits, not audio — there is no gradual audio degradation to measure.
- A SINAD test exercises the whole chain — front-end [noise figure](/reference/noise-figure/),
  IF filtering, [demodulation](/reference/demodulation/), and audio stages — so it
  captures distortion that a mid-chain SNR measurement would miss.

## Relevance to SDR

SINAD remains the yardstick for the analog side of the radio world:
[broadcast FM](/reference/broadcast-fm/), [marine VHF](/reference/marine-vhf/), amateur
FM, and conventional analog land-mobile receivers are all specified in SINAD, and it is
how you compare the front-end quality of an SDR used for analog reception. It does not,
however, describe [GopherTrunk](/reference/software-defined-radio/)'s digital decode
path — [P25](/reference/p25-phase-1/), [DMR](/reference/dmr/), and the other trunked
digital modes GT targets live or die by BER and constellation error
([EVM](/reference/error-vector-magnitude/)), not by recovered-audio SINAD. Where GT does
touch analog audio — decoded [vocoder](/reference/vocoder/) output — the perceptual
quality is set by the codec and the upstream BER, so SINAD is not a meaningful measure
of it. Think of SINAD as the metric for the analog receivers GT often shares a band
with, rather than for GT's own decoders.

## Sources

[^sinad]: [SINAD](https://en.wikipedia.org/wiki/SINAD) — Wikipedia, definition, measurement method, and use in receiver sensitivity specifications.
