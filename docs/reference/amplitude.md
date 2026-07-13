---
slug: amplitude
title: Amplitude
entry_type: term
category: rf-fundamentals
description: Amplitude is the magnitude or strength of a wave; in radio it corresponds to signal power and is the quantity varied by amplitude modulation.
keywords: amplitude, signal strength, magnitude, power, AM, envelope, peak, RMS
infobox:
  - { label: Type, value: Wave property }
  - { label: Represents, value: Strength / power }
  - { label: Reported as, value: Power level (dBm / dBFS) }
see_also: [phase, decibel, dbm, amplitude-modulation, signal-to-noise-ratio, path-loss]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Amplitude
  - https://en.wikipedia.org/wiki/Decibel
---

**Amplitude** is the magnitude — the "height" — of a wave, the peak departure of its
oscillation from the resting level.[^wiki] For a [radio wave](/reference/radio-wave/) it
corresponds to signal strength, which a receiver reports as a power level in
[dBm](/reference/dbm/) or, inside an SDR, in [dBFS](/reference/dbfs/) relative to the
converter's full scale. Amplitude is one of the three properties (with
[frequency](/reference/frequency/) and [phase](/reference/phase/)) that fully describe a
wave and that a transmitter can vary to carry information.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A larger-amplitude sine wave and a smaller-amplitude sine wave sharing a centre line, with the larger one labelled as the stronger signal." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 70 q35 -48 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M20 70 q35 -16 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.6" stroke-opacity="0.6"/>
  <line x1="90" y1="70" x2="90" y2="22" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="98" y="40" font-size="11" fill="currentColor">larger amplitude = stronger signal</text>
</svg>
<figcaption>Amplitude is the height of the wave; greater amplitude delivers more power to the receiver.</figcaption>
</figure>

## How it works

Amplitude and power are tightly linked: the power a wave carries is proportional to the
square of its amplitude. So doubling the amplitude quadruples the power, which is why
power is the more natural currency once a signal reaches a receiver, and why levels are
usually quoted logarithmically in [decibels](/reference/decibel/) — a scale that turns
those wide multiplicative ranges into manageable additions. A single sinusoid can be
summarised by its peak amplitude, but for complex or noisy signals engineers more often
use the RMS (root-mean-square) amplitude, which relates directly to average power, and
the peak-to-average ratio ([crest factor](/reference/crest-factor-papr/)), which matters
for amplifier headroom.

As a wave spreads out from a transmitter, its amplitude falls. In free space the field
strength drops with distance ([free-space path loss](/reference/free-space-path-loss/)),
and obstacles, absorption, and destructive [multipath](/reference/multipath-propagation/)
reduce it further ([path loss](/reference/path-loss/)). That steady weakening is why a
distant station is faint and why link budgets exist. At the receiver the wanted
amplitude competes with an ever-present [noise floor](/reference/noise-floor/); their
ratio is the [signal-to-noise ratio](/reference/signal-to-noise-ratio/) that ultimately
bounds whether the message can be recovered.

## In practice

- **The envelope.** The slowly varying outline traced by a modulated carrier's
  amplitude is its envelope. [Amplitude modulation](/reference/amplitude-modulation/)
  writes the message directly into that envelope, which a simple diode detector can
  recover.
- **Gain staging.** Front-end amplifiers and attenuators set where a signal's amplitude
  lands relative to the converter's range. Too little and it sinks into quantisation
  noise; too much and it clips, generating [intermodulation](/reference/intermodulation/)
  products. Automatic [gain control](/reference/automatic-gain-control/) keeps it in the
  sweet spot.
- **Fading.** Amplitude is not static — multipath and motion make it fluctuate over
  time ([Rayleigh](/reference/rayleigh-fading/) / [Rician](/reference/rician-fading/)
  fading), which receivers must ride out.

## Relevance to SDR

In an SDR the wave arrives as [IQ samples](/reference/iq-data/), and a sample's amplitude
is its distance from the origin of the IQ plane — the magnitude √(I² + Q²). GopherTrunk
reads this to judge whether a channel is active, to drive automatic gain control, and to
compute the SNR estimates it uses to gate weak decodes. Varying amplitude is also the
basis of amplitude modulation and one axis of
[QAM](/reference/quadrature-amplitude-modulation/), so tracking it accurately is part of
demodulating those schemes. Because the four-level C4FM and π/4-DQPSK signals GopherTrunk
decodes are largely constant-envelope, amplitude there serves mainly as a health and
gain-staging indicator rather than as the information itself.

## Sources

[^wiki]: [Amplitude](https://en.wikipedia.org/wiki/Amplitude) — Wikipedia, on the magnitude of a wave's oscillation and its peak/RMS measures.
[^db]: [Decibel](https://en.wikipedia.org/wiki/Decibel) — Wikipedia, on the logarithmic scale used to express amplitude and power ratios in radio.
