---
slug: antenna-gain
title: Antenna gain
entry_type: term
category: antennas
description: Antenna gain is the degree to which an antenna concentrates radiated or received energy in a preferred direction, expressed in dBi or dBd.
keywords: antenna gain, dBi, dBd, directivity, radiation pattern, Yagi, beamwidth, efficiency, EIRP
aka: [antenna gain]
autolink: true
infobox:
  - { label: Type, value: Antenna property }
  - { label: Units, value: dBi (vs isotropic), dBd (vs dipole) }
  - { label: Trade-off, value: More gain = narrower pattern }
see_also: [antenna, dipole-antenna, yagi-uda-antenna, radiation-pattern, beamwidth, decibel, radio-propagation]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_gain
  - https://en.wikipedia.org/wiki/Directivity
---

**Antenna gain** measures how strongly an [antenna](/reference/antenna/) concentrates
energy in a preferred direction compared with a reference. It is given in
[decibels](/reference/decibel/): **dBi** relative to an isotropic radiator, or **dBd**
relative to a [dipole](/reference/dipole-antenna/).[^wiki] The two scales differ by a
constant: because a half-wave dipole already has 2.15 dBi of gain, any figure in dBd is
2.15 dB lower than the same antenna's dBi number.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An omnidirectional circular pattern on the left and a focused directional lobe on the right." xmlns="http://www.w3.org/2000/svg">
  <circle cx="110" cy="75" r="45" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="110" cy="75" r="2.5" fill="currentColor"/>
  <text x="110" y="140" text-anchor="middle" font-size="9" fill="currentColor">omnidirectional</text>
  <path d="M330 75 C 330 35, 430 45, 440 75 C 430 105, 330 115, 330 75 Z" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="330" cy="75" r="2.5" fill="currentColor"/>
  <text x="360" y="140" text-anchor="middle" font-size="9" fill="currentColor">directional (gain)</text>
</svg>
<figcaption>Antenna gain doesn't create energy — it focuses the pattern, trading all-round coverage for reach.</figcaption>
</figure>

## How it works

Gain does not create energy; it redistributes it. An antenna's **directivity** is a purely
geometric quantity: how peaked its [radiation pattern](/reference/radiation-pattern/) is
compared with an isotropic source that radiates equally in every direction.[^dir]
Concentrating the same total power into a narrower solid angle raises the intensity in the
favoured direction, and the number of decibels by which it exceeds isotropic is the gain.
**Gain equals directivity times [antenna efficiency](/reference/antenna-efficiency/)** —
so ohmic and mismatch losses make a real antenna's gain a little lower than its directivity
suggests.

Two facts follow directly. First, **higher gain means a narrower
[beamwidth](/reference/beamwidth/)**: you buy reach in one direction by giving up coverage
in the others. An omnidirectional vertical hears every bearing at modest strength; a
[Yagi](/reference/yagi-uda-antenna/) with 12 dBi hears far more in the direction it points
and much less to the sides and rear. Second, gain is **reciprocal** — an antenna that
transmits 6 dB harder toward a spot also receives 6 dB better from that spot, so receive-only
users benefit from gain exactly as transmitters do.

## In practice

- **dBi vs dBd** — read the reference carefully when comparing products; a "9 dBd" antenna
  is 11.15 dBi, so vendors quoting dBi look better on paper for the same hardware.
- **Vertical collinear** gain — stacking half-waves flattens the doughnut toward the
  horizon, adding gain without becoming directional; popular for base scanners.
- **EIRP** — on transmit, gain adds to power to give effective isotropic radiated power;
  on receive the analogous benefit is a better [link budget](/reference/link-budget/).
- **Height usually beats gain** — for line-of-sight bands, raising the antenna to clear
  obstructions often helps more than a few dB of pattern focusing.

## Relevance to SDR

For general scanning, an omnidirectional antenna is usually best, because a wideband SDR
watches many sites on many bearings at once and a narrow beam would miss most of them. A
directional, high-gain antenna such as a [Yagi](/reference/yagi-uda-antenna/) earns its
keep when the goal is to pull in one specific distant system, or to null out a strong local
interferer by pointing the antenna's low-response direction at it. GopherTrunk has no
beamforming hardware and does not steer patterns electronically — gain here is entirely a
property of the physical antenna the operator chooses, and it shows up in the decode chain
as extra [SNR](/reference/signal-to-noise-ratio/) at the [ADC](/reference/analog-to-digital-converter/).

## Sources

[^wiki]: [Antenna gain](https://en.wikipedia.org/wiki/Antenna_gain) — Wikipedia, for the definition of gain and the dBi/dBd reference units.
[^dir]: [Directivity](https://en.wikipedia.org/wiki/Directivity) — Wikipedia, on directivity, its relation to gain via efficiency, and the beamwidth trade-off.
