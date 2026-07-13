---
slug: nvis
title: Near-vertical incidence skywave (NVIS)
entry_type: term
category: propagation
description: NVIS is an HF technique that fires signals almost straight up so the ionosphere reflects them down over a wide area, giving reliable regional coverage with no skip zone.
keywords: NVIS, near vertical incidence skywave, regional HF, cloud warmer, high angle radiation, skip zone fill, tactical HF, emergency communications
aka: [NVIS, near-vertical incidence skywave]
autolink: true
infobox:
  - { label: Type, value: High-angle HF skywave mode }
  - { label: Mechanism, value: Near-vertical ionospheric reflection }
  - { label: Coverage, value: ~0–400 km, no skip zone }
see_also: [sky-wave, ionospheric-propagation, ground-wave, radio-propagation, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Near_vertical_incidence_skywave
  - https://en.wikipedia.org/wiki/Skywave
---

**Near-vertical incidence skywave** (**NVIS**) is an
[HF](/reference/frequency-bands/) technique that radiates a
[radio wave](/reference/radio-wave/) almost straight up so the ionosphere reflects it
back down over a wide surrounding area, giving dependable regional coverage with no dead
zone underneath.[^wiki] It is the deliberate, high-angle special case of
[sky-wave](/reference/sky-wave/) propagation, chosen precisely to fill the
[skip zone](/reference/sky-wave/) that ordinary long-haul HF leaves empty.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A central transmitter firing rays nearly straight up to the ionosphere, which reflects them back down in an umbrella that blankets receivers on all sides within a few hundred kilometres." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="20" y1="40" x2="440" y2="40" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="6 4"/><text x="20" y="33" font-size="9" fill="currentColor">ionosphere (F layer)</text>
  <line x1="230" y1="138" x2="230" y2="118" stroke="currentColor" stroke-width="2"/><text x="230" y="153" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <path d="M230 118 L200 44 L120 130" fill="none" stroke="currentColor" stroke-width="1.4" marker-end="url(#nvar)"/>
  <path d="M230 118 L230 44 L230 132" fill="none" stroke="currentColor" stroke-width="1.4" marker-end="url(#nvar)"/>
  <path d="M230 118 L262 44 L345 130" fill="none" stroke="currentColor" stroke-width="1.4" marker-end="url(#nvar)"/>
  <text x="120" y="145" text-anchor="middle" font-size="8" fill="currentColor">RX</text><text x="345" y="145" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <text x="300" y="90" font-size="8" fill="currentColor" fill-opacity="0.7">umbrella of coverage, no skip zone</text>
  <defs><marker id="nvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Firing energy nearly vertically, NVIS blankets everything within a few hundred kilometres — including the terrain shadows that VHF line-of-sight cannot reach.</figcaption>
</figure>

## How it works

The ionosphere returns a wave only up to a maximum frequency that depends on the launch
angle. For a straight-up shot that ceiling is the **critical frequency** — typically a
few megahertz — so NVIS must operate on low HF bands (roughly 2–10 MHz), dropping in
frequency at night as ionisation falls. A signal launched at a steep angle refracts off
the F layer and comes back down spread over a circle a few hundred kilometres across,
centred near the transmitter.

The technique hinges on the antenna's radiation pattern:

- **Low horizontal antennas.** A dipole mounted low — often just a fraction of a
  wavelength above ground — throws most of its energy upward instead of toward the
  horizon, exactly the pattern NVIS wants. This is why NVIS dipoles are nicknamed
  "cloud warmers."
- **Right frequency.** The band must sit below the critical frequency or the signal
  punches through to space; operators track the ionosphere and shift bands day to night.
- **No skip zone.** Because the coverage circle starts at the transmitter and the
  ground wave fills the immediate vicinity, there is no ring of silence — the key
  advantage over long-haul HF.

Coverage is also terrain-independent: signals arrive from overhead, so hills, valleys,
and buildings that block [line-of-sight](/reference/radio-horizon/) VHF do not create
shadows.

## In practice

NVIS is the workhorse of regional HF where VHF/UHF cannot reach: military and tactical
field communications over rough terrain, disaster and emergency nets after
infrastructure fails, and any application needing reliable voice or data across a
province-sized area without repeaters or satellites. Its main limits are bandwidth and
noise — the low HF bands are congested and electrically noisy — so NVIS carries voice,
CW, and narrow digital modes rather than high-rate traffic.

## Relevance to SDR

NVIS listening needs an HF-capable receiver: an [upconverter](/reference/upconverter/)
ahead of an [RTL-SDR](/reference/rtl-sdr/) or a native HF SDR such as the
[Airspy HF+](/reference/airspy-hf-plus/). For a VHF/UHF trunking scanner like
**GopherTrunk**, NVIS is out of band and out of scope — the land-mobile trunked systems
GopherTrunk decodes live far above HF and rely on
[line-of-sight](/reference/radio-horizon/) and repeater infrastructure rather than
ionospheric coverage. It is worth knowing as the counterexample that shows how a
completely different coverage philosophy — overhead reflection instead of horizon-bound
line of sight — solves the terrain-shadow problem that shapes VHF/UHF system design.

## Sources

[^wiki]: [Near vertical incidence skywave](https://en.wikipedia.org/wiki/Near_vertical_incidence_skywave) — Wikipedia, on high-angle HF radiation, critical frequency, low-dipole patterns, and skip-zone-free regional coverage.
