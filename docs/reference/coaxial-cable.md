---
slug: coaxial-cable
title: Coaxial cable
entry_type: hardware
category: hardware
description: Coaxial cable carries RF from antenna to receiver on a shielded centre conductor. Every metre and connector adds loss, more at higher frequencies — keep runs short.
keywords: coaxial cable, coax, feedline, RG-58, RG-6, LMR-400, cable loss, shield, impedance
aka: [coax, "coaxial cable", feedline]
autolink: true
see_also: [attenuation, path-loss, antenna, standing-wave-ratio, low-noise-amplifier]
related_lessons:
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Coaxial cable (Wikipedia)", url: https://en.wikipedia.org/wiki/Coaxial_cable }
---

**Coaxial cable** ("coax") carries RF between the antenna and receiver. A centre
conductor runs inside a tubular **shield**, separated by a dielectric, which keeps the
signal contained and the impedance constant (commonly 50 Ω). Every metre and every
connector adds [loss](/reference/attenuation/) — and the loss grows with frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 130" role="img" aria-label="A cutaway of coaxial cable showing the centre conductor, dielectric, shield, and outer jacket as concentric layers." xmlns="http://www.w3.org/2000/svg">
  <ellipse cx="160" cy="65" rx="120" ry="48" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <ellipse cx="160" cy="65" rx="92" ry="36" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <ellipse cx="160" cy="65" rx="58" ry="22" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-opacity="0.6"/>
  <circle cx="160" cy="65" r="6" fill="currentColor"/>
  <g font-size="8" fill="currentColor"><text x="160" y="58" text-anchor="middle">core</text><text x="160" y="95" text-anchor="middle">dielectric</text><text x="160" y="118" text-anchor="middle">shield + jacket</text></g>
</svg>
<figcaption>Coax carries RF on a centre conductor inside a shield; keep runs short and quality high to limit loss.</figcaption>
</figure>

## Overview

A long or low-grade cable can quietly undo a good antenna, so operators keep feedline
short or mount a [low-noise amplifier](/reference/low-noise-amplifier/) at the antenna. A
poor match also raises [SWR](/reference/standing-wave-ratio/).
