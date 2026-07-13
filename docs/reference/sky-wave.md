---
slug: sky-wave
title: Sky-wave propagation
entry_type: term
category: propagation
description: Sky-wave propagation is HF communication by ionospheric refraction, bouncing signals between the ionosphere and the ground in "hops" to reach thousands of kilometres.
keywords: sky wave, skywave, ionospheric hop, HF propagation, skip distance, MUF, maximum usable frequency, multi-hop, shortwave
aka: [sky wave, skywave]
autolink: true
infobox:
  - { label: Type, value: HF over-the-horizon mode }
  - { label: Mechanism, value: Refraction by the ionosphere }
  - { label: Reach, value: Thousands of km via multiple hops }
see_also: [ionospheric-propagation, ground-wave, radio-propagation, shortwave-broadcast, nvis]
cite_urls:
  - https://en.wikipedia.org/wiki/Skywave
  - https://en.wikipedia.org/wiki/Maximum_usable_frequency
---

**Sky-wave propagation** is long-distance
[HF](/reference/frequency-bands/) communication in which a
[radio wave](/reference/radio-wave/) is refracted back to earth by the ionosphere,
then reflected upward again by the ground, repeating in a series of "hops" that can
span oceans.[^wiki] It is the same physical mechanism described under
[ionospheric propagation](/reference/ionospheric-propagation/), viewed as a
hop-by-hop geometry that determines how far a signal reaches and where it lands.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A transmitter launching an HF ray that refracts off the ionosphere, returns to the ground, reflects, and refracts again in a second hop to a far receiver, with a skip zone marked near the transmitter." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="20" y1="40" x2="440" y2="40" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="6 4"/><text x="20" y="33" font-size="9" fill="currentColor">ionosphere</text>
  <line x1="45" y1="138" x2="45" y2="122" stroke="currentColor" stroke-width="2"/><text x="45" y="153" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="420" y1="138" x2="420" y2="122" stroke="currentColor" stroke-width="2"/><text x="420" y="153" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <path d="M45 122 L160 44 L235 138 L350 44 L420 122" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#swar)"/>
  <line x1="70" y1="128" x2="215" y2="128" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="3 3"/><text x="105" y="126" font-size="8" fill="currentColor" fill-opacity="0.8">skip zone</text>
  <text x="150" y="70" font-size="8" fill="currentColor" fill-opacity="0.7">hop 1</text><text x="330" y="70" font-size="8" fill="currentColor" fill-opacity="0.7">hop 2</text>
  <defs><marker id="swar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each hop refracts off the ionosphere and reflects off the ground; the un-illuminated ring between the ground wave and the first hop is the skip zone.</figcaption>
</figure>

## How it works

The ionosphere is a set of charged layers (D, E, and the F1/F2 layers) created when
solar ultraviolet and X-rays ionise the thin upper atmosphere. A wave entering these
layers at a shallow angle is progressively bent until it returns to earth. The steeper
the launch angle, the higher the frequency the layer can still turn back; a wave that
is too high in frequency or too steep simply punches through into space.

Two frequencies bound any given path:

- **Maximum usable frequency (MUF)** — the highest frequency the ionosphere will still
  return for that path length and time; above it the signal escapes. Longer paths use
  shallower angles and support a higher MUF.
- **Lowest usable frequency (LUF)** — set by absorption in the lower D layer; below it
  the signal is swallowed before it can refract.

Between the reach of the [ground wave](/reference/ground-wave/) and the point where the
first hop lands lies the **skip zone**, a ring of no reception. Signals arriving by
several paths of slightly different length interfere, producing the characteristic
selective fading and "flutter" of HF sky-wave reception. Because ionisation tracks the
sun, the usable frequencies swing with time of day, season, and the 11-year solar
cycle — low bands work at night, high bands in daytime.

## In practice

Sky wave is the backbone of the HF services: international
[shortwave broadcast](/reference/shortwave-broadcast/), amateur DX, marine and
aeronautical long-haul, and military over-the-horizon links. Operators pick a band near
the day's MUF for the best signal-to-noise, dropping to lower bands after dark. A
short, steep variant — [NVIS](/reference/nvis/) — deliberately aims almost straight up
to fill the skip zone for regional coverage. Multi-hop paths can circle the globe, but
each ground reflection and each ionospheric pass adds loss and distortion.

## Relevance to SDR

Hearing sky wave needs an HF-capable front end: an
[upconverter](/reference/upconverter/) ahead of an [RTL-SDR](/reference/rtl-sdr/) or a
direct-sampling HF receiver such as the [Airspy HF+](/reference/airspy-hf-plus/). For a
VHF/UHF trunking scanner like **GopherTrunk**, sky wave is out of band — land-mobile
trunked systems operate well above the HF range and rely on
[line-of-sight](/reference/radio-horizon/) coverage. The concept still matters to SDR
users as the clearest illustration of why the HF spectrum behaves so unlike the bands a
scanner monitors, and why the same dial can sound completely different by day and by
night.

## Sources

[^wiki]: [Skywave](https://en.wikipedia.org/wiki/Skywave) — Wikipedia, on ionospheric refraction, multi-hop geometry, and the skip zone.
