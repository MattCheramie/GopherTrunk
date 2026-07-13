---
slug: polarization
title: Polarization
entry_type: term
category: antennas
description: Polarization is the orientation of a radio wave's electric field, set by the transmitting antenna; matching receive and transmit polarization avoids signal loss.
keywords: polarization, vertical, horizontal, circular, cross-polarization, electric field, Faraday rotation, polarisation
aka: [polarization, polarisation]
autolink: true
infobox:
  - { label: Type, value: Wave/antenna property }
  - { label: Common kinds, value: Vertical, horizontal, circular }
  - { label: Mismatch loss, value: Up to ~20 dB }
see_also: [antenna, dipole-antenna, radiation-pattern, mimo, radio-propagation, antenna-gain]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Polarization_(waves)
  - https://en.wikipedia.org/wiki/Circular_polarization
---

**Polarization** is the orientation of a [radio wave](/reference/radio-wave/)'s
electric field, determined by how the transmitting [antenna](/reference/antenna/) is
mounted — vertical, horizontal, or circular.[^wiki] It is a property of the wave itself,
carried alongside the signal, and matching it between transmitter and receiver is one of
the simplest ways to avoid throwing away signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A vertically oriented wave on the left and a horizontally oriented wave on the right, showing polarization." xmlns="http://www.w3.org/2000/svg">
  <line x1="110" y1="20" x2="110" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M110 20 q -28 25 0 50 t0 50" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="110" y="135" text-anchor="middle" font-size="9" fill="currentColor">vertical</text>
  <line x1="300" y1="70" x2="420" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M300 70 q 25 -28 50 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="125" text-anchor="middle" font-size="9" fill="currentColor">horizontal</text>
  <text x="230" y="70" text-anchor="middle" font-size="9" fill="currentColor">match the</text><text x="230" y="84" text-anchor="middle" font-size="9" fill="currentColor">transmitter</text>
</svg>
<figcaption>Polarization is the orientation of the wave's electric field; match it to the transmitter to avoid loss.</figcaption>
</figure>

## How it works

An electromagnetic wave has an electric field and a magnetic field at right angles to each
other and to the direction of travel. Polarization names the direction the **electric**
field points. A vertical antenna element launches a vertically polarized wave; a horizontal
element, a horizontal one. When the receiving antenna's element lies along the arriving
field, it develops maximum voltage; when it lies across the field, the induced voltage
approaches zero. The loss from a partial mismatch follows the cosine-squared of the angle
between them, so a 45° tilt costs about 3 dB and a full 90° cross-polarization can cost
**20 dB or more** in practice.[^cir]

## Variants

- **Linear (vertical / horizontal)** — a single straight element. Most terrestrial
  two-way radio is vertical; over-the-air FM and TV broadcast is often horizontal or mixed.
- **Circular (RHCP / LHCP)** — the field rotates as the wave advances, produced by feeding
  crossed elements 90° out of phase or by a helix. A circularly polarized receiver loses
  only about 3 dB to *any* linear signal regardless of tilt, which makes it forgiving of
  tumbling or unknown orientation.[^cir]
- **Slant / elliptical** — intermediate cases; real signals are rarely perfectly pure.

Two related effects matter in the field. Reflections off buildings and terrain scramble
polarization, so a signal that started vertical can arrive partly horizontal after
[multipath](/reference/multipath-propagation/). And over long HF paths, **Faraday rotation**
in the ionosphere slowly turns the plane of a linear wave, one reason satellite and
long-haul links often use circular polarization instead.

## In practice

Match polarization at both ends when you can. Satellites — GPS/GNSS, weather birds, many
comms payloads — transmit circular, so a circular or turnstile antenna outperforms a
straight whip for them. Cross-polarization is also used deliberately: it lets two signals
share one frequency on orthogonal polarizations, and [MIMO](/reference/mimo/) systems
exploit polarization (and spatial) diversity to separate multiple streams. For a scanner,
the practical rule is to align the antenna with the dominant traffic's polarization and
accept that reflections will blur the picture.

## Relevance to SDR

A vertical antenna is the safe default for scanning land-mobile and trunked systems,
matching their vertical polarization; using a horizontal antenna on the same traffic can
bury a signal in the noise floor for no other reason. When a strong transmitter still won't
decode, a quiet polarization mismatch is worth ruling out before blaming the receiver.
GopherTrunk itself does no polarization processing — it decodes the single stream its front
end delivers — so polarization is entirely an antenna-siting decision the operator makes
ahead of the [ADC](/reference/analog-to-digital-converter/).

## Sources

[^wiki]: [Polarization (waves)](https://en.wikipedia.org/wiki/Polarization_(waves)) — Wikipedia, on the orientation of a wave's electric field and polarization types.
[^cir]: [Circular polarization](https://en.wikipedia.org/wiki/Circular_polarization) — Wikipedia, on circular/elliptical polarization and the ~3 dB and cross-pol loss figures.
