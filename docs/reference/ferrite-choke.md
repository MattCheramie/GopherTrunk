---
slug: ferrite-choke
title: Ferrite choke
entry_type: hardware
category: rf-front-end
description: "A ferrite choke is a ferrite core clamped over a cable that adds impedance to common-mode RF currents, suppressing interference and cable-radiated noise."
keywords: ferrite choke, ferrite bead, ferrite core, common-mode choke, RFI suppression, EMI filter, snap-on ferrite, clip-on ferrite, cable choke, feedline choke, current balun
aka: [ferrite choke, "ferrite bead", "clip-on ferrite", "common-mode choke"]
autolink: true
infobox:
  - { label: Type, value: "Common-mode suppression component" }
  - { label: Material, value: "Ferrite core (NiZn or MnZn mix)" }
  - { label: Key spec, value: "Impedance vs frequency, mix number" }
  - { label: TX, value: "Yes (as feedline choke)" }
  - { label: Typical price, value: "$0.50–$10 each" }
see_also: [balun, coaxial-cable, feedpoint-impedance, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Ferrite_bead
  - https://en.wikipedia.org/wiki/Ferrite_core
---

A **ferrite choke** is a ring or clamp of ferrite material placed around a cable so
that it adds impedance to *common-mode* currents — the unwanted RF that flows on the
outside of a coax shield or equally on all conductors of a cable — while leaving the
wanted *differential* signal inside almost untouched.[^bead] By turning that
common-mode current into a lossy, high-impedance path, the choke suppresses cable-
radiated interference and stops noise from riding a feedline into a receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A ferrite core clamped around a coaxial cable presents high impedance to common-mode shield current while the differential signal passes through unaffected." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="65" x2="430" y2="65" stroke="currentColor" stroke-width="3"/>
  <line x1="30" y1="65" x2="430" y2="65" stroke="currentColor" stroke-opacity="0.35" stroke-width="9"/>
  <rect x="200" y="35" width="60" height="60" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.8"/>
  <rect x="218" y="53" width="24" height="24" rx="3" fill="none" stroke="currentColor"/>
  <text x="230" y="28" text-anchor="middle" font-size="9" fill="currentColor">ferrite core</text>
  <text x="110" y="55" text-anchor="middle" font-size="8" fill="currentColor">common-mode noise</text>
  <text x="350" y="55" text-anchor="middle" font-size="8" fill="currentColor">suppressed</text>
  <text x="230" y="118" text-anchor="middle" font-size="8" fill="currentColor">wanted signal passes inside</text>
</svg>
<figcaption>Clamped over a cable, a ferrite core chokes common-mode current while the differential signal passes through.</figcaption>
</figure>

## Overview

Ferrite is a ceramic of iron oxide mixed with other metals; its high magnetic
permeability makes a few turns of cable through the core act as an inductor, and at
RF the ferrite's loss turns that inductance into a resistive impedance that
dissipates common-mode energy as heat. The effect is frequency-dependent and set by
the ferrite **mix**: manganese-zinc mixes (e.g. type 31, 43) work best from a few
MHz through VHF, while nickel-zinc mixes (type 61) peak higher into UHF. Winding the
cable through the core several times multiplies the impedance (it rises with the
square of the turns) but lowers the frequency of peak effect.

## Variants

- **Snap-on / clip-on ferrites** — hinged cores that clamp over an existing cable
  without disconnecting it; the familiar lump near the end of USB and monitor
  cords.
- **Ferrite beads** — small cylinders threaded on a single wire, used on circuit
  boards and power leads for high-frequency decoupling.
- **Toroid chokes** — a cable wound several times through a ring for a stronger,
  lower-frequency [feedline](/reference/coaxial-cable/) choke.
- **Current (choke) balun** — many turns of coax on a ferrite core form a
  [current balun](/reference/balun/): a purpose-built common-mode choke at an
  antenna [feedpoint](/reference/feedpoint-impedance/).

## Relevance to SDR

Common-mode current on a feedline is one of the most common — and most misdiagnosed
— sources of noise for SDR listeners. When the coax shield carries RF, the feedline
itself radiates and receives: it picks up switching-supply hash, Ethernet and USB
noise, and household [RFI](/reference/electromagnetic-spectrum/), then delivers it
straight to the receiver as a raised [noise floor](/reference/noise-floor/). A
ferrite choke of the right mix at the antenna feedpoint (and at the SDR end of the
USB cable) breaks that path, often dropping the noise floor by several dB with no
change to the wanted signal.

For a [dipole](/reference/dipole-antenna/) or other balanced antenna fed with coax,
a choke also restores balance, keeping the pattern clean and preventing the shield
from becoming an unintended part of the antenna.

GopherTrunk is a software decoder and includes no hardware, so a ferrite choke is
purely part of the analog install ahead of the SDR. Its payoff for GopherTrunk is
practical: lowering feedline-borne noise directly improves the signal-to-noise ratio
at the demodulator, which is often the cheapest way to turn a marginal
control-channel lock into a solid one.

## Sources

[^bead]: [Ferrite bead](https://en.wikipedia.org/wiki/Ferrite_bead) — Wikipedia, on ferrite cores and beads used to suppress common-mode and high-frequency currents on cables.
