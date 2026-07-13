---
slug: noise-floor
title: Noise floor
entry_type: term
category: rf-fundamentals
description: The noise floor is the ever-present background level of thermal and environmental noise in a receiver; a signal must rise above it to be usable.
keywords: noise floor, thermal noise, background noise, sensitivity, dBm, kTB, noise figure
aka: [noise floor]
autolink: true
infobox:
  - { label: Type, value: Background noise level }
  - { label: Unit, value: dBm }
  - { label: Sources, value: Thermal noise, environment, receiver }
see_also: [signal-to-noise-ratio, dbm, low-noise-amplifier, attenuation, thermal-noise, noise-figure, receiver-sensitivity]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Noise_floor
  - https://en.wikipedia.org/wiki/Johnson%E2%80%93Nyquist_noise
---

The **noise floor** is the constant background level of random energy present in any
receiver — [thermal noise](/reference/thermal-noise/) in the electronics plus
environmental RF picked up by the antenna.[^wiki] It is measured in
[dBm](/reference/dbm/) and sets the bar every signal must clear to be usable.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A wandering noisy baseline across frequency marked as the noise floor, with the level set by thermal noise, receiver noise figure, and environmental RF, and no signal present." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 95 L60 90 L90 99 L120 88 L150 96 L180 86 L210 98 L240 90 L270 100 L300 88 L330 95 L360 87 L390 97 L420 91" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="30" y1="93" x2="430" y2="93" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="30" y="34" font-size="11" fill="currentColor">noise floor — the level a signal must rise above</text>
  <text x="30" y="52" font-size="9" fill="currentColor">= thermal (kTB) + noise figure + environment</text>
  <text x="230" y="128" text-anchor="middle" font-size="10" fill="currentColor">frequency →</text>
</svg>
<figcaption>The noise floor is the ever-present background from thermal, receiver, and environmental noise; its height sets the usable sensitivity.</figcaption>
</figure>

## How it works

The floor has a hard physical bottom set by **thermal noise**. Any resistor — or antenna
radiation resistance — at temperature T generates noise power *kTB*, where k is
Boltzmann's constant, T is temperature in kelvin, and B is bandwidth.[^jn] At room
temperature this works out to about **−174 dBm per hertz**. Multiply by bandwidth (add
10·log₁₀(B) in dB) and you get the floor for a perfect receiver: roughly −121 dBm in a
12.5 kHz channel, −114 dBm in a 100 kHz span. No receiver can hear below this.

Two things push the real floor above that ideal. First, the receiver adds its own noise,
quantified by [noise figure](/reference/noise-figure/): a 5 dB noise figure raises the
floor 5 dB above kTB. This is why a [low-noise amplifier](/reference/low-noise-amplifier/)
placed first in the chain matters — it sets the noise figure of everything after it.
Second, the **environment** contributes external noise: atmospheric, galactic, and
especially man-made interference from switching power supplies, USB buses, LED lighting,
Ethernet, and solar inverters. In a noisy urban install this environmental noise, not the
receiver, dominates the floor.

Because the thermal contribution scales with bandwidth, **the floor moves with the
measurement**. Halving the receive bandwidth lowers the noise power by 3 dB while keeping
a narrowband signal intact — which is exactly why matched filtering and narrow channel
filters improve [SNR](/reference/signal-to-noise-ratio/). Conversely, a wideband capture
shows a higher floor simply because it integrates noise over more spectrum.

## In practice

The margin between a signal and the floor is the [SNR](/reference/signal-to-noise-ratio/),
and the floor's height directly sets [receiver sensitivity](/reference/receiver-sensitivity/)
— the weakest signal that still decodes. Lowering the effective floor is therefore the
cheapest way to hear more:

- Put a low-noise preamp at the antenna, before cable
  [attenuation](/reference/attenuation/) degrades noise figure.
- Reduce environmental pickup: ferrite chokes, quality shielded cable, and moving the SDR
  away from computers and chargers can drop the floor 10 dB or more.
- Filter out strong out-of-band signals that would otherwise raise the floor through
  front-end [intermodulation](/reference/intermodulation/).
- Match the receive bandwidth to the signal so noise is not admitted needlessly.

A common diagnostic: if the displayed floor drops several dB when you disconnect the
antenna, external environmental noise is dominating and the fix is at the install, not in
the software.

## Relevance to SDR

On an SDR, the floor you see on the waterfall is the sum of thermal noise, the front-end
and ADC [noise figure](/reference/noise-figure/), and whatever the antenna is collecting.
GopherTrunk cannot decode a signal that sits at or below the floor, so getting the floor
as low as possible — quiet install, LNA at the antenna, right bandwidth — is the highest-
leverage thing an operator can do before touching any DSP setting.

## Sources

[^wiki]: [Noise floor](https://en.wikipedia.org/wiki/Noise_floor) — Wikipedia, the background noise level in a measurement system and its constituents.
[^jn]: [Johnson–Nyquist noise](https://en.wikipedia.org/wiki/Johnson%E2%80%93Nyquist_noise) — Wikipedia, the kTB thermal-noise law that sets the physical floor.
