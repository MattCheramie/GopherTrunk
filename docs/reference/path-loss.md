---
slug: path-loss
title: Path loss
entry_type: term
category: rf-fundamentals
description: Path loss is the attenuation a radio signal suffers travelling from transmitter to receiver, dominated by the spreading of energy over distance and obstacles in the way.
keywords: path loss, free space loss, propagation loss, distance, dB, path loss exponent
aka: [path loss, propagation loss]
autolink: true
infobox:
  - { label: Type, value: Propagation attenuation }
  - { label: Unit, value: Decibels (dB) }
  - { label: Grows with, value: Distance and frequency }
see_also: [attenuation, radio-propagation, decibel, signal-to-noise-ratio, free-space-path-loss, link-budget, fade-margin]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
  - { title: "Decibels & signal power", url: /learn/rf-sdr/decibels/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Path_loss
  - https://en.wikipedia.org/wiki/Free-space_path_loss
---

**Path loss** is the [attenuation](/reference/attenuation/) a signal experiences
travelling from transmitter to receiver.[^wiki] It is dominated by the spreading of energy
over distance, plus extra losses from terrain, buildings, and foliage — and it is the
single largest term in almost every [link budget](/reference/link-budget/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A curve of received power falling steeply with distance, illustrating that path loss in decibels grows with the logarithm of range, plus a note that free space loss goes as distance squared." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="20" x2="50" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="50" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M55 28 C 120 70, 200 100, 435 115" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="20" y="70" font-size="10" fill="currentColor" transform="rotate(-90 20 70)">power (dB)</text>
  <text x="240" y="140" text-anchor="middle" font-size="10" fill="currentColor">distance →</text>
  <text x="240" y="154" text-anchor="middle" font-size="9" fill="currentColor">free space: −20 dB per decade of distance, per decade of frequency</text>
</svg>
<figcaption>Path loss grows with distance and frequency; it can exceed 100 dB over a few kilometres, which is why budgets are done in decibels.</figcaption>
</figure>

## How it works

In empty space, a transmitter radiates power over an expanding sphere. The power crossing
a fixed receive aperture falls as the **square of distance** — the inverse-square law — so
free-space loss rises 20 dB for every tenfold increase in range and, for a fixed-size
antenna, another 20 dB for every tenfold increase in frequency. This idealised case is
[free-space path loss](/reference/free-space-path-loss/), captured by the
[Friis transmission equation](/reference/friis-transmission-equation/).

Real environments lose far more. The catch-all model writes received power as falling with
distance raised to a **path-loss exponent** n: n ≈ 2 in free space, but 2.7–4 in cluttered
urban and indoor settings, and even higher through heavy obstruction. On top of that
distance trend sit two extra effects: **shadowing**, the slow variation as terrain and
buildings block the path (often modelled as a log-normal spread of a few dB), and
**multipath fading**, the fast fluctuation from reflected copies of the signal arriving out
of phase. A [link budget](/reference/link-budget/) reserves a
[fade margin](/reference/fade-margin/) to survive these dips.

Not all of the loss is bulk absorption. Diffraction over rooftops and hills, reflection off
the ground and buildings, and blockage of the [Fresnel zone](/reference/fresnel-zone/)
around the direct line all shape how much energy reaches the receiver. This is why raising
an antenna even a few metres — clearing obstacles and opening the Fresnel zone — can buy
more than a large increase in transmit power.

## In practice

Because path loss so often totals 100 dB or more, it dwarfs the handful of dB available
from better cable or a preamp — which is why the biggest wins come from geometry: antenna
height, a clear line of sight, and picking a band that suits the range. Lower frequencies
generally carry farther for the same power (less free-space loss and better diffraction
around obstacles), one reason VHF public-safety systems reach farther per site than UHF.

Path loss also explains the *shape* of a received-power map: signal strength does not fall
off a cliff at some range but decays smoothly with the logarithm of distance, so coverage
fades gradually into the [noise floor](/reference/noise-floor/) rather than stopping
sharply. Predicting coverage means estimating path loss over terrain and comparing the
result against [receiver sensitivity](/reference/receiver-sensitivity/).

## Relevance to SDR

Path loss is why a distant or obstructed system arrives near the
[noise floor](/reference/noise-floor/) with barely enough
[SNR](/reference/signal-to-noise-ratio/) to decode, and why antenna height and a clear
path ([propagation](/reference/radio-propagation/)) matter so much for a scanner install.
GopherTrunk receives whatever the path delivers; when a known nearby system is
un-decodable, path loss (obstruction, low antenna) is usually the first suspect, ahead of
anything in the DSP chain.

## Sources

[^wiki]: [Path loss](https://en.wikipedia.org/wiki/Path_loss) — Wikipedia, definition, the path-loss exponent model, shadowing, and fading.
[^fspl]: [Free-space path loss](https://en.wikipedia.org/wiki/Free-space_path_loss) — Wikipedia, the inverse-square, distance-and-frequency-squared idealised case.
