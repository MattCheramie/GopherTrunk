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
  - { title: "What is a radio wave?", url: /learn/radio-waves/ }
external:
  - { title: "Wavelength (Wikipedia)", url: https://en.wikipedia.org/wiki/Wavelength }
---

**Wavelength** (λ) is the physical distance a wave covers in one complete cycle. For a
[radio wave](/reference/radio-wave/) it is inversely proportional to
[frequency](/reference/frequency/): higher frequency means shorter wavelength.

## How it works

Since radio travels at the speed of light, *λ = c / f*. A handy approximation is
**λ (metres) ≈ 300 ÷ frequency (MHz)** — so 150 MHz is about 2 m and 460 MHz about
0.65 m.

## Relevance to SDR

Wavelength sets [antenna](/reference/antenna/) size (a quarter-wave whip is λ/4) and
influences how signals bend around obstacles, making it central to antenna choice and
[propagation](/reference/radio-propagation/).
