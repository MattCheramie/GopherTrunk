---
slug: e-reader
title: E-reader
entry_type: hardware
category: hw-mobile
description: An e-reader is a portable device optimized for reading digital books, typically using a low-power electronic-paper display that mimics ink on paper and runs for weeks on a single charge.
keywords: e-reader, e-ink, electronic paper, e-paper, Kindle, Kobo, ereader, reflective display, e-book, microcapsule
aka: [eReader, e-book reader]
infobox:
  - { label: Type, value: Portable reading device }
  - { label: Display, value: Electronic paper (E Ink) }
  - { label: Battery life, value: Weeks per charge }
  - { label: Examples, value: Kindle, Kobo }
see_also: [tablet, touchscreen, battery-technology, smartphone, mobile-operating-system, system-on-a-chip]
related_lessons:
  - { title: "Tablets", url: /learn/intro-hardware/tablets/ }
cite_urls:
  - https://en.wikipedia.org/wiki/E-reader
---

An **e-reader** is a portable device built for reading digital books, using a low-power electronic-paper display that looks like ink on paper.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A cross-section of an E Ink electronic-paper display. Microcapsules sit between two electrode layers. Each capsule holds white and black pigment particles in a clear fluid. An applied voltage pulls white particles to the top to show white, or black particles up to show black, and the image stays put with no power until the next page change." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.1" font-family="ui-sans-serif, sans-serif">
    <line x1="40" y1="34" x2="420" y2="34"/>
    <line x1="40" y1="120" x2="420" y2="120"/>
    <g>
      <circle cx="95" cy="77" r="26"/>
      <circle cx="160" cy="77" r="26"/>
      <circle cx="225" cy="77" r="26"/>
      <circle cx="290" cy="77" r="26"/>
      <circle cx="355" cy="77" r="26"/>
    </g>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <circle cx="88" cy="62" r="4" fill="none" stroke-width="1"/>
    <circle cx="102" cy="60" r="4" fill="none" stroke-width="1"/>
    <circle cx="95" cy="90" r="4" fill-opacity="1"/>
    <circle cx="153" cy="92" r="4" fill="none" stroke-width="1"/>
    <circle cx="167" cy="90" r="4" fill="none" stroke-width="1"/>
    <circle cx="160" cy="63" r="4" fill-opacity="1"/>
    <circle cx="220" cy="65" r="4" fill="none" stroke-width="1"/>
    <circle cx="225" cy="90" r="4" fill-opacity="1"/>
    <circle cx="285" cy="90" r="4" fill="none" stroke-width="1"/>
    <circle cx="290" cy="63" r="4" fill-opacity="1"/>
    <circle cx="350" cy="63" r="4" fill="none" stroke-width="1"/>
    <circle cx="355" cy="92" r="4" fill-opacity="1"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="44" y="28">top electrode &#183; viewer side</text>
    <text x="44" y="136">bottom electrode &#183; drives pixels</text>
    <text x="95" y="150" text-anchor="middle" font-size="8">white up</text>
    <text x="160" y="150" text-anchor="middle" font-size="8">black up</text>
    <text x="290" y="150" text-anchor="middle" font-size="8">holds without power</text>
  </g>
</svg>
<figcaption>Each E Ink microcapsule holds white and black pigment in clear fluid; a voltage pulses the chosen particles to the top and the image then persists with zero power until the next page turn — the root of the e-reader's weeks-long battery life.</figcaption>
</figure>

## Overview

The defining part is the *electronic-paper* (E Ink) display: a reflective screen of microcapsules that hold their image with no power once set, so the device only draws energy when the page changes. Because it reflects ambient light rather than emitting its own, the page stays readable in direct sunlight and is gentle on the eyes, at the cost of slow refresh and (on most models) grayscale only.

The result is paper-like readability and [battery](/reference/battery-technology/) life measured in weeks rather than hours. E-readers run a stripped-down [mobile operating system](/reference/mobile-operating-system/) on a modest [SoC](/reference/system-on-a-chip/), often with a [touchscreen](/reference/touchscreen/) and a front light for reading in the dark. Familiar examples include Amazon's Kindle and Rakuten's Kobo.

## E Ink vs LCD/OLED

The display choice is the whole trade-off between an e-reader and a [tablet](/reference/tablet/):

| Property | E Ink (e-reader) | LCD / OLED (tablet) |
|----------|------------------|---------------------|
| Light | Reflective (ambient) | Emissive (backlit) |
| Power when static | ~Zero | Continuous |
| Refresh rate | Slow (page turns) | Fast (video) |
| Color | Mostly grayscale | Full color |
| Sunlight | Excellent | Washes out |
| Best for | Long-form reading | Media, apps |

## Where it fits

An e-reader trades the bright, fast color of a tablet for outstanding battery life and eye comfort at one task — reading. The same E Ink technology shows up on shelf labels and low-power status panels, where a display that needs power only to update is exactly the right tool for a device that mostly sits idle — the opposite end of the power spectrum from a continuously running SDR capture node.

## Sources

[^wiki]: [E-reader](https://en.wikipedia.org/wiki/E-reader) — Wikipedia, on e-readers and electronic-paper displays.
