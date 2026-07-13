---
slug: ionospheric-propagation
title: Ionospheric propagation
entry_type: term
category: propagation
description: Ionospheric propagation is the refraction of HF radio waves by charged layers of the upper atmosphere, enabling long-distance "skip" communication around the world.
keywords: ionosphere, skip, skywave, HF propagation, shortwave, refraction, MUF, F layer, solar cycle
aka: [ionospheric propagation, skywave]
autolink: true
infobox:
  - { label: Type, value: HF propagation mode }
  - { label: Mechanism, value: Refraction by ionosphere }
  - { label: Enables, value: Worldwide HF "skip" }
see_also: [radio-propagation, sky-wave, ground-wave, frequency-bands, airspy-hf-plus]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Skywave
  - https://en.wikipedia.org/wiki/Ionosphere
---

**Ionospheric propagation** (skywave) is the refraction of
[HF](/reference/frequency-bands/) [radio waves](/reference/radio-wave/) by ionised
layers of the upper atmosphere, allowing signals to "skip" over the horizon for
hundreds or thousands of kilometres.[^wiki] It is the mechanism behind long-distance
shortwave broadcast, amateur DX, and much HF utility traffic, and it is what makes the HF
bands feel alive and unpredictable compared with steady line-of-sight VHF.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An HF signal leaving a transmitter, refracting off the ionosphere layer, and returning to a distant receiver." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="20" y1="35" x2="440" y2="35" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="6 4"/><text x="20" y="28" font-size="9" fill="currentColor">ionosphere</text>
  <line x1="60" y1="118" x2="60" y2="100" stroke="currentColor" stroke-width="2"/><text x="60" y="135" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="400" y1="118" x2="400" y2="100" stroke="currentColor" stroke-width="2"/><text x="400" y="135" text-anchor="middle" font-size="8" fill="currentColor">RX (far)</text>
  <path d="M60 100 L230 38 L400 100" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#ioar)"/>
  <defs><marker id="ioar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>HF signals can refract off the ionosphere and "skip" thousands of kilometres beyond the horizon.</figcaption>
</figure>

## How it works

Solar ultraviolet and X-rays strip electrons from atoms high in the atmosphere, creating
ionised layers — the D, E, and F regions — between roughly 60 and 400 km up.[^iono] A radio
wave entering this ionised gas is progressively **refracted**: the free electrons slow the
wave's upper part relative to its lower part, bending its path. At HF the bend can be sharp
enough to turn the wave back toward Earth, so what looks like a reflection is really
cumulative refraction. The wave returns to the surface far away, may bounce off the ground,
and refract again — **multi-hop** paths can circle much of the planet.

Whether a given frequency skips depends on the electron density and the launch angle:

- **Maximum usable frequency (MUF)** — the highest frequency the ionosphere will bend back
  for a given path. Above the MUF the wave punches through into space instead of returning.
- **Critical frequency** — the MUF for a straight-up wave; a rough gauge of the layer's
  strength.
- **Skip zone** — a ring where the ground wave has died out but the first hop hasn't yet
  come down, so the signal is unheard.

## In practice

Because ionisation is driven by the sun, skywave conditions change with the **time of day**
(the D layer absorbs by day and fades at night, opening the low bands after dark),
**season**, and the ~11-year **solar cycle**. The same 20 m signal that reaches across an
ocean at noon may be gone by midnight, while 80 m does the opposite. Higher bands —
VHF and up — carry too much frequency for the ionosphere to bend, so they normally pass
straight through and stay line-of-sight, governed instead by the
[radio horizon](/reference/radio-horizon/). Below HF, the [ground wave](/reference/ground-wave/)
provides steady local coverage that does not depend on the ionosphere at all.

## Relevance to SDR

Receiving HF skip needs an HF-capable radio such as the
[Airspy HF+](/reference/airspy-hf-plus/) or an [upconverter](/reference/upconverter/),
since a basic [RTL-SDR](/reference/rtl-sdr/) does not tune HF directly. Ionospheric paths
add fading, [Doppler](/reference/doppler-shift/), and multi-path spreading that stress
narrowband decoders, so HF digital modes are built to be robust against them. GopherTrunk's
focus is land-mobile VHF/UHF trunking, which lives above the skip bands, so ionospheric
propagation is context for the wider spectrum rather than a path GopherTrunk itself decodes.

## Sources

[^wiki]: [Skywave](https://en.wikipedia.org/wiki/Skywave) — Wikipedia, on ionospheric refraction of HF radio waves and long-distance skip propagation.
[^iono]: [Ionosphere](https://en.wikipedia.org/wiki/Ionosphere) — Wikipedia, on the D/E/F layers, ionisation by solar radiation, and the MUF/critical frequency.
