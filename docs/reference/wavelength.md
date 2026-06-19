---
slug: wavelength
title: Wavelength
entry_type: term
category: rf-fundamentals
description: Wavelength is the physical distance a wave travels in one cycle; for radio it is inversely proportional to frequency and sets antenna dimensions.
keywords: wavelength, lambda, antenna length, frequency relation
aka: [wavelength]
autolink: true
infobox:
  - { label: Symbol, value: λ (lambda) }
  - { label: Unit, value: Metres }
  - { label: Rule of thumb, value: "λ (m) ≈ 300 / f (MHz)" }
see_also: [frequency, radio-wave, antenna, dipole-antenna]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Wavelength
---

**Wavelength** (λ) is the physical distance a wave covers in one complete cycle.[^wiki] For a
[radio wave](/reference/radio-wave/) it is inversely proportional to
[frequency](/reference/frequency/): higher frequency means shorter wavelength.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A sine wave with the distance of one full cycle marked as the wavelength." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="75" x2="440" y2="75" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 75 C 60 15, 120 15, 160 75 S 260 135, 300 75 S 400 15, 440 75" fill="none" stroke="currentColor" stroke-width="2.2"/>
  <line x1="40" y1="30" x2="180" y2="30" stroke="currentColor" marker-start="url(#wls)" marker-end="url(#wle)"/>
  <text x="110" y="24" text-anchor="middle" font-size="12" fill="currentColor">λ = one cycle</text>
  <text x="20" y="118" font-size="11" fill="currentColor">λ (m) ≈ 300 ÷ frequency (MHz)</text>
  <defs>
    <marker id="wls" markerWidth="8" markerHeight="8" refX="2" refY="3" orient="auto"><path d="M6 0 L0 3 L6 6 z" fill="currentColor"/></marker>
    <marker id="wle" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
</svg>
<figcaption>Wavelength is the physical length of one cycle — inversely proportional to frequency.</figcaption>
</figure>

## How it works

Since radio travels at the speed of light, *λ = c / f*. A handy approximation is
**λ (metres) ≈ 300 ÷ frequency (MHz)** — so 150 MHz is about 2 m and 460 MHz about
0.65 m.

## Relevance to SDR

Wavelength sets [antenna](/reference/antenna/) size (a quarter-wave whip is λ/4) and
influences how signals bend around obstacles, making it central to antenna choice and
[propagation](/reference/radio-propagation/).

## Sources

[^wiki]: [Wavelength](https://en.wikipedia.org/wiki/Wavelength) — Wikipedia, on the spatial period of a wave and its inverse relation to frequency.
