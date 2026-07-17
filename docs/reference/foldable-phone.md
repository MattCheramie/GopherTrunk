---
slug: foldable-phone
title: Foldable phone
entry_type: hardware
category: hw-mobile
description: A foldable phone is a smartphone with a flexible display and hinge that folds, opening from a pocketable size into a larger tablet-like screen or shrinking a full phone into a compact form.
keywords: foldable phone, folding phone, flexible display, hinge, foldable OLED, book fold, clamshell, dual screen
infobox:
  - { label: Type, value: Smartphone }
  - { label: Display, value: Flexible OLED }
  - { label: Mechanism, value: Hinge }
  - { label: Forms, value: Book-fold, clamshell }
see_also: [smartphone, tablet, touchscreen, mobile-operating-system, battery-technology, system-on-a-chip]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Foldable_smartphone
---

A **foldable phone** is a [smartphone](/reference/smartphone/) with a flexible display and a hinge, letting it fold from a pocketable size into a larger screen and back.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A cross-section of a foldable phone's hinge. The flexible OLED panel drapes over a curved teardrop gap at the fold so it bends on a radius instead of a sharp crease. Two rigid halves connect to a geared hinge mechanism at the center, which lets the panel roll into the gap as the phone closes." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3" font-family="ui-sans-serif, sans-serif">
    <path d="M40 60 h150 C210 60 214 78 230 92 C246 78 250 60 270 60 h150" stroke-width="2"/>
    <line x1="40" y1="60" x2="40" y2="80"/>
    <line x1="420" y1="60" x2="420" y2="80"/>
    <rect x="46" y="82" width="140" height="20" rx="3"/>
    <rect x="274" y="82" width="140" height="20" rx="3"/>
    <circle cx="230" cy="112" r="18"/>
    <g stroke-width="0.9">
      <line x1="230" y1="94" x2="230" y2="130"/>
      <line x1="212" y1="112" x2="248" y2="112"/>
      <line x1="217" y1="99" x2="243" y2="125"/>
      <line x1="243" y1="99" x2="217" y2="125"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="230" y="46" text-anchor="middle">flexible OLED bends on a teardrop radius</text>
    <text x="115" y="150" text-anchor="middle">rigid half</text>
    <text x="345" y="150" text-anchor="middle">rigid half</text>
    <text x="230" y="150" text-anchor="middle" font-size="8">geared hinge</text>
  </g>
</svg>
<figcaption>The flexible OLED drapes into a teardrop-shaped gap at the fold so it curves on a gentle radius rather than creasing; a geared hinge links the two rigid halves and rolls the panel into that gap as the phone closes.</figcaption>
</figure>

## Overview

The enabling part is a flexible OLED [touchscreen](/reference/touchscreen/) that bends around a precision hinge without cracking. To avoid a sharp, panel-damaging crease, the display drops into a *teardrop* gap when folded, curving on a controlled radius; the hinge itself is a small geared mechanism engineered to survive hundreds of thousands of cycles.

Two broad styles have emerged: *book-fold* designs that open from phone size into a small [tablet](/reference/tablet/), and *clamshell* (flip) designs that fold a full-size phone down to a compact square. Inside, the device is a normal phone — an [SoC](/reference/system-on-a-chip/), radios, and a [battery](/reference/battery-technology/) split across the two halves — running an ordinary [mobile OS](/reference/mobile-operating-system/) adapted to switch layouts as the screen folds.

## Book-fold vs clamshell

The two form factors optimize opposite priorities:

| Aspect | Book-fold | Clamshell (flip) |
|--------|-----------|-------------------|
| Goal | Bigger screen | Smaller footprint |
| Open size | Small tablet | Normal phone |
| Closed size | Thick phone | Compact square |
| Outer screen | Full cover display | Small window |
| Typical use | Multitasking, media | Pocketability, style |

## Where it fits

The foldable tries to collapse the phone-versus-tablet trade-off into one device: pocket portability when closed, more screen when open. The costs are mechanical complexity, a visible crease, added thickness, and a higher price — engineering challenges centered on the durability of the hinge and the flexible panel over thousands of folds. For an SDR operator, the appeal is mundane but real: a larger unfolded screen makes reviewing a decoder's live web console in the field less cramped than a standard phone.

## Sources

[^wiki]: [Foldable smartphone](https://en.wikipedia.org/wiki/Foldable_smartphone) — Wikipedia, on foldable phone designs and flexible displays.
