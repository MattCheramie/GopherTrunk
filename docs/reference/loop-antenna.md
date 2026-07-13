---
slug: loop-antenna
title: Loop antenna
entry_type: term
category: antennas
description: A loop antenna is a closed conductor loop; small loops respond to the magnetic field with a figure-eight pattern, while full-wave loops resonate and radiate efficiently.
keywords: loop antenna, small loop, magnetic loop, full-wave loop, resonant loop, loopstick, ferrite loop, quad loop, direction finding, magnetic field antenna
aka: [loop, magnetic loop, resonant loop]
autolink: true
infobox:
  - { label: Type, value: Closed-conductor loop }
  - { label: Small loop, value: Magnetic (H-field) pickup }
  - { label: Resonant loop, value: ~1 λ circumference }
see_also: [antenna, magnetic-loop-antenna, polarization, radiation-pattern, q-factor, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Loop_antenna
  - https://en.wikipedia.org/wiki/Magnetic_dipole
---

A **loop antenna** is simply a closed loop of wire, and its behaviour splits sharply into
two regimes depending on how big the loop is compared to the [wavelength](/reference/wavelength/).[^wiki]
A **small loop** — one whose circumference is much less than a wavelength — acts as a
magnetic dipole: it responds to the *magnetic* part of the passing radio wave and shows a
sharp figure-eight [pattern](/reference/radiation-pattern/) with deep nulls, making it a
classic direction-finding and receiving [antenna](/reference/antenna/). A **resonant
loop** — one about a full wavelength around — behaves completely differently, radiating
efficiently like a folded [dipole](/reference/dipole-antenna/) bent into a ring. The same
shape thus serves as both a tiny sensitive receiver and a full-size transmitting element.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="On the left a small wire loop with a figure-eight response pattern; on the right a full-wavelength loop that radiates efficiently." xmlns="http://www.w3.org/2000/svg">
  <circle cx="90" cy="85" r="30" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="90" y1="115" x2="82" y2="132" stroke="currentColor" stroke-width="2"/>
  <line x1="90" y1="115" x2="98" y2="132" stroke="currentColor" stroke-width="2"/>
  <text x="55" y="150" font-size="9" fill="currentColor">small loop</text>
  <path d="M170 85 C 150 55 210 55 190 85 C 210 115 150 115 170 85 Z" fill="none" stroke="currentColor" stroke-opacity="0.45" stroke-dasharray="4 3"/>
  <text x="150" y="30" font-size="9" fill="currentColor">figure-eight pattern</text>
  <circle cx="360" cy="85" r="48" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="360" y1="133" x2="352" y2="150" stroke="currentColor" stroke-width="2"/>
  <line x1="360" y1="133" x2="368" y2="150" stroke="currentColor" stroke-width="2"/>
  <text x="322" y="85" font-size="9" fill="currentColor">~1 λ</text>
  <text x="318" y="165" font-size="9" fill="currentColor">full-wave loop</text>
</svg>
<figcaption>A small loop responds to the magnetic field with a figure-eight pattern; a full-wavelength loop resonates and radiates efficiently.</figcaption>
</figure>

## How it works

**Small loop.** When the loop is electrically tiny, the current is essentially uniform
all the way around, and the loop's response is proportional to the *rate of change of
magnetic flux* threading it. Two facts follow. First, it is a **magnetic-field** sensor,
so it largely ignores the local electric-field noise (nearby power lines, dimmers, plasma
TVs) that an electric antenna picks up — a small loop can be strikingly quiet. Second,
its pattern is a figure-eight in the plane of the loop with sharp nulls broadside to the
loop face; rotating the loop to null out an interferer or to find a bearing is the basis
of radio direction finding. The penalty is efficiency: a small loop has a tiny radiation
resistance, so as a transmitter it is poor unless carefully tuned (see the
[magnetic loop antenna](/reference/magnetic-loop-antenna/)).

**Resonant loop.** Grow the loop until its circumference is about one wavelength and it
resonates. Now the current varies around the loop and the structure radiates well, with a
gain slightly higher than a dipole and its main lobe *broadside to the plane of the loop*
— the opposite of the small loop. A one-wavelength square loop is the "quad" element; two
or more make a cubical-quad beam, a compact rival to the Yagi.

## Variants

- **Ferrite loopstick.** Winding the loop on a ferrite rod concentrates magnetic flux
  through the coil, shrinking the antenna dramatically — the AM broadcast antenna inside
  nearly every portable radio.
- **Tuned small loop.** Adding a resonating capacitor turns the broadband small loop into
  a high-[Q](/reference/q-factor/), narrowband receiving (or QRP transmitting) antenna,
  covered separately as the magnetic loop.
- **Full-wave / quad loop.** A resonant transmitting element, often used in beams.

## Relevance to SDR

Loops are among the most useful receive antennas for an SDR at HF and below. Their
magnetic pickup rejects nearby electrical noise, and their nulls let you steer away from
a strong local interferer that would otherwise desensitise the receiver — valuable
because SDRs have finite [dynamic range](/reference/dynamic-range/). Small tuned and
broadband loops are the standard antenna for low-frequency, MW, and shortwave listening
on RTL-SDR and similar dongles, and loopsticks are what let a pocket radio hear AM at all.

GopherTrunk works at VHF/UHF on trunked land-mobile systems, where vertical whips
dominate and full-size resonant loops are uncommon, so a loop is not a typical GopherTrunk
antenna. It appears here as a foundational antenna type and the parent of the
[magnetic loop](/reference/magnetic-loop-antenna/).

## Sources

[^wiki]: [Loop antenna](https://en.wikipedia.org/wiki/Loop_antenna) — Wikipedia, for the small-versus-resonant-loop distinction, magnetic pickup, and pattern nulls.
