---
slug: doppler-shift
title: Doppler shift
entry_type: term
category: propagation
description: Doppler shift is the change in received frequency caused by relative motion between transmitter and receiver, prominent on satellite and fast-mobile links.
keywords: Doppler shift, Doppler effect, frequency shift, relative motion, Doppler spread, satellite tracking, LEO Doppler, mobile fading rate
aka: [Doppler shift, Doppler effect]
autolink: true
infobox:
  - { label: Type, value: Motion-induced frequency error }
  - { label: Formula, value: "Δf ≈ (v/c)·f" }
  - { label: Largest for, value: LEO satellites, fast mobiles }
see_also: [iridium, gps-gnss, frequency, automatic-frequency-control, rayleigh-fading]
cite_urls:
  - https://en.wikipedia.org/wiki/Doppler_effect
  - https://en.wikipedia.org/wiki/Doppler_effect_(radio)
---

**Doppler shift** is the change in the frequency of a received radio wave caused by
relative motion between the transmitter and receiver.[^wiki] Motion toward the receiver
compresses the wavefronts and raises the observed [frequency](/reference/frequency/);
motion away stretches them and lowers it. The magnitude is proportional to the radial
velocity divided by the speed of light, so it is negligible for slow terrestrial links
but dominant for [low-Earth-orbit satellites](/reference/iridium/) and fast aircraft.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A moving transmitter emits wavefronts that bunch up ahead of it, raising the received frequency, and spread out behind it, lowering it, illustrating Doppler shift." xmlns="http://www.w3.org/2000/svg">
  <circle cx="230" cy="80" r="0" />
  <circle cx="215" cy="80" r="18" fill="none" stroke="currentColor" stroke-width="1"/>
  <circle cx="205" cy="80" r="38" fill="none" stroke="currentColor" stroke-width="1"/>
  <circle cx="192" cy="80" r="60" fill="none" stroke="currentColor" stroke-width="1"/>
  <circle cx="178" cy="80" r="84" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="222" y="72" width="18" height="16" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1"/>
  <line x1="240" y1="80" x2="290" y2="80" stroke="currentColor" stroke-width="1.4" marker-end="url(#dopar)"/>
  <text x="265" y="72" font-size="9" fill="currentColor">motion</text>
  <text x="360" y="60" text-anchor="middle" font-size="9" fill="currentColor">ahead:</text>
  <text x="360" y="72" text-anchor="middle" font-size="9" fill="currentColor">higher f</text>
  <text x="70" y="60" text-anchor="middle" font-size="9" fill="currentColor">behind:</text>
  <text x="70" y="72" text-anchor="middle" font-size="9" fill="currentColor">lower f</text>
  <defs><marker id="dopar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Wavefronts bunch ahead of a moving source (higher frequency) and spread behind it (lower frequency).</figcaption>
</figure>

## How it works

For velocities small compared with light, the frequency shift is

- `Δf ≈ (v_radial / c) · f`,

where `v_radial` is the component of relative velocity along the line of sight, `c` is
the speed of light, and `f` is the carrier frequency. Only the radial component matters:
a source moving directly across the field of view has zero instantaneous Doppler even at
high speed, while one heading straight at the receiver shows the maximum.

Two consequences matter for radio:

- **A frequency offset.** The whole carrier lands off its nominal channel by Δf, which a
  narrow demodulator must track or it loses lock.
- **Doppler spread.** In a [multipath](/reference/multipath-propagation/) channel the
  scattered rays arrive from many directions, each with its own Doppler, smearing a pure
  tone into a small band. The width of that band sets how fast the envelope fades and is
  the physical driver of the fade rate in [Rayleigh fading](/reference/rayleigh-fading/).

## Relevance to SDR

Doppler is central to satellite reception. A [LEO](/reference/iridium/) bird crossing
overhead sweeps its carrier by several kilohertz to tens of kilohertz across a pass; at
[Iridium's](/reference/iridium/) 1.6 GHz L-band the shift reaches roughly ±35 kHz, and
NOAA/Meteor weather-satellite and cubesat downlinks show similar sweeps. Receiving them
requires either a wide capture bandwidth plus software tracking, or predicting the shift
from orbital elements and retuning as the pass progresses.

The opposite use is positioning: [GPS and other GNSS](/reference/gps-gnss/) receivers
measure the Doppler on each satellite's carrier to recover velocity, and must search a
Doppler dimension during signal acquisition. For terrestrial land-mobile radio the raw
offset is tiny — a vehicle at highway speed shifts a 450 MHz P25 carrier by only tens of
hertz — but the associated Doppler spread still governs how quickly the mobile channel
fades, which is what stresses interleaving and [FEC](/reference/forward-error-correction/).

[GopherTrunk](/reference/software-defined-radio/) targets terrestrial trunking, where the
carrier offset from motion is well within the pull-in range of its
[automatic frequency control](/reference/automatic-frequency-control/); it does not
implement satellite Doppler tracking, which belongs to dedicated sat-tracking receivers.

## In practice

Doppler is why satellite ground stations continuously retune, and why any narrowband
demodulator carries an AFC or [frequency-locked loop](/reference/frequency-locked-loop/)
to absorb residual carrier offset. It also sets a subtle limit on how long a receiver can
coherently integrate a signal before the accumulating phase rotation from an uncorrected
shift washes out the gain.

## Sources

[^wiki]: [Doppler effect](https://en.wikipedia.org/wiki/Doppler_effect) — Wikipedia, on the frequency change from relative motion and its radio applications.
