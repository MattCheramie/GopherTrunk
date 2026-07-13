---
slug: carrier-wave
title: Carrier wave
entry_type: term
category: rf-fundamentals
description: A carrier wave is a steady radio-frequency signal that carries no information by itself until it is modulated, varying its amplitude, frequency, or phase to convey a message.
keywords: carrier wave, carrier, modulation, unmodulated, RF, sidebands, DC spike
aka: [carrier wave, carrier]
autolink: true
infobox:
  - { label: Type, value: Reference signal }
  - { label: Carries info via, value: Modulation }
  - { label: Appears as, value: Single spectral spike (unmodulated) }
see_also: [modulation, radio-wave, frequency, amplitude-modulation, bandwidth, subcarrier]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/rf-sdr/signal-anatomy/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Carrier_wave
  - https://en.wikipedia.org/wiki/Sideband
---

A **carrier wave** is a steady [radio-frequency](/reference/radio-wave/) signal at a
single [frequency](/reference/frequency/) that conveys no information on its own.[^wiki]
It becomes useful only when [modulation](/reference/modulation/) varies one of its
properties — [amplitude](/reference/amplitude/), frequency, or [phase](/reference/phase/)
— in step with a message. The carrier is the vehicle; the message rides on it. Its
frequency is what you tune a receiver to, and it is why "the frequency of a station" and
"its carrier" mean nearly the same thing.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A steady unmodulated carrier wave on top, and the same carrier amplitude-modulated by a message below, showing how modulation shapes the carrier." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="20" font-size="10" fill="currentColor">unmodulated carrier</text>
  <path d="M20 45 q10 -18 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="20" y="95" font-size="10" fill="currentColor">modulated (carries information)</text>
  <path d="M20 120 q10 -8 20 0 t20 0 q10 -22 20 0 t20 0 q10 -22 20 0 t20 0 q10 -8 20 0 t20 0 q10 -4 20 0 t20 0 q10 -8 20 0 t20 0 q10 -22 20 0 t20 0 q10 -22 20 0 t20 0 q10 -8 20 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.6"/>
</svg>
<figcaption>A bare carrier carries no information until modulation varies its amplitude, frequency, or phase.</figcaption>
</figure>

## How it works

A pure, unmodulated carrier is a single sinusoid, so in the frequency domain it appears
as one narrow spike — all of its energy at exactly one frequency. The moment it is
modulated, that energy spreads: the message frequencies mix with the carrier and produce
**sidebands**, mirror-image bands of energy above and below the carrier frequency. The
total span the sidebands occupy is essentially the signal's
[bandwidth](/reference/bandwidth/) (more precisely its
[occupied bandwidth](/reference/occupied-bandwidth/)), and it grows with the rate and
depth of the modulation. A voice AM signal spreads a few kilohertz either side of its
carrier; a wideband FM broadcast, hundreds of kilohertz.

Why use a carrier at all? Two reasons. First, an antenna is only efficient at a size
comparable to the [wavelength](/reference/wavelength/), and baseband audio (a few
kilohertz) has wavelengths of tens of kilometres — impractical to radiate. Shifting the
message up onto a high-frequency carrier makes it radiatable from a reasonable antenna.
Second, carriers let many stations share the spectrum: each occupies its own carrier
frequency with a [guard band](/reference/guard-band/) between them, and a receiver picks
one out by tuning. Some systems suppress the carrier itself to save power
([single sideband](/reference/single-sideband/) transmits only one sideband and no
carrier), trading receiver complexity for efficiency.

## In practice

- **Spotting a carrier.** On a [waterfall](/reference/waterfall-display/) or
  [spectrum analyzer](/reference/spectrum-analyzer/), an idle transmitter shows a thin
  bright line; keying up blooms it into a modulated band.
- **Carrier squelch.** The simplest [squelch](/reference/squelch/) opens the audio when
  a carrier of sufficient strength is present, regardless of content.
- **Subcarriers.** A modulated carrier can itself carry a second, lower-frequency
  [subcarrier](/reference/subcarrier/) — how FM broadcast fits stereo and
  [RDS](/reference/rds/) data alongside the main audio.

## Relevance to SDR

After an SDR mixes a chosen slice of spectrum down toward
[baseband](/reference/baseband/), the tuned carrier ends up near zero frequency. A
residual carrier or [DC offset](/reference/dc-offset/) sitting at exactly zero produces
the familiar bright "DC spike" in the centre of an SDR's spectrum — an artifact, not a
real station. GopherTrunk locks onto each channel's carrier to establish a phase and
frequency reference, then demodulates the variations around it; the digital voice systems
it decodes keep the carrier present and modulate its phase, so accurate carrier tracking
is the foundation of the whole decode chain.

## Sources

[^wiki]: [Carrier wave](https://en.wikipedia.org/wiki/Carrier_wave) — Wikipedia, on the steady reference signal modulated to convey information.
[^sideband]: [Sideband](https://en.wikipedia.org/wiki/Sideband) — Wikipedia, on the bands of energy modulation creates around a carrier and how they set occupied bandwidth.
