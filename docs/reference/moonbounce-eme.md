---
slug: moonbounce-eme
title: Moonbounce (EME)
entry_type: term
category: propagation
description: Moonbounce (Earth-Moon-Earth) is a communication mode that uses the Moon as a passive reflector for VHF/UHF/microwave signals, overcoming ~250 dB of path loss.
keywords: moonbounce, EME, earth moon earth, lunar reflection, VHF DX, UHF DX, microwave DX, libration fading, faraday rotation, path loss
aka: [moonbounce, EME, earth-moon-earth]
autolink: true
infobox:
  - { label: Type, value: Passive-reflector VHF/UHF/µW mode }
  - { label: Reflector, value: The Moon (~384,000 km) }
  - { label: Path loss, value: ~250 dB round trip }
see_also: [free-space-path-loss, doppler-shift, parabolic-antenna, antenna-gain, radio-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Earth%E2%80%93Moon%E2%80%93Earth_communication
  - https://en.wikipedia.org/wiki/Moonbounce
---

**Moonbounce**, or **Earth-Moon-Earth (EME)**, is a propagation mode that uses the Moon
as a passive reflector: a station aims a [radio wave](/reference/radio-wave/) at the
Moon, and the faint echo scattered back reaches any other station that can also see it,
linking points anywhere on the Moon-facing hemisphere.[^wiki] Because the round trip is
about 770,000 km, EME must overcome roughly 250 dB of
[free-space path loss](/reference/free-space-path-loss/), making it the most demanding
"natural reflector" mode in amateur and experimental radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Two dish antennas on a curved earth both pointing at the Moon; one transmits a beam that reflects diffusely off the lunar surface and returns as a much weaker echo to the other station." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 160 Q140 138 260 160" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <circle cx="400" cy="45" r="26" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-opacity="0.6"/><text x="400" y="88" text-anchor="middle" font-size="9" fill="currentColor">Moon</text>
  <path d="M70 145 l-10 -6 l10 -6 z" fill="currentColor"/><text x="70" y="160" text-anchor="middle" font-size="8" fill="currentColor">station A</text>
  <path d="M210 148 l-10 -6 l10 -6 z" fill="currentColor"/><text x="210" y="163" text-anchor="middle" font-size="8" fill="currentColor">station B</text>
  <path d="M74 140 L378 50" fill="none" stroke="currentColor" stroke-width="1.6" marker-end="url(#emear)"/><text x="180" y="86" font-size="8" fill="currentColor">uplink</text>
  <path d="M378 58 L214 142" fill="none" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.5" stroke-dasharray="4 3" marker-end="url(#emear)"/><text x="300" y="118" font-size="8" fill="currentColor" fill-opacity="0.7">weak echo</text>
  <defs><marker id="emear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The Moon scatters a tiny fraction of the incident energy back toward earth; the returning echo is roughly 250 dB weaker than the transmitted signal.</figcaption>
</figure>

## How it works

The Moon is a poor, rough reflector — only a few percent of the energy that hits it
scatters back, and it scatters diffusely rather than mirror-like. The dominant challenge
is distance: [free-space path loss](/reference/free-space-path-loss/) grows with the
square of range and frequency, and over the Earth-Moon-Earth round trip it totals about
250 dB. Closing that link is a brute-force exercise in the
[link budget](/reference/link-budget/):

- **Big antennas.** Stations use high-gain arrays or
  [parabolic dishes](/reference/parabolic-antenna/) with as much
  [antenna gain](/reference/antenna-gain/) as they can steer and track across the sky.
- **Power and low noise.** Legal-limit power amplifiers on transmit, and very low
  [noise figure](/reference/noise-figure/) preamplifiers on receive, since the echo can
  be at or below the noise floor.
- **Weak-signal coding.** Modern EME leans on error-correcting digital modes that dig
  signals out of the noise, replacing the extreme operator skill once needed for CW.

The path also imposes its own distortions. **Doppler shift** from the relative motion of
earth and Moon offsets the echo frequency by tens to hundreds of hertz and must be
tracked. **Libration fading** — a slow flutter caused by the Moon's apparent rocking,
which makes different parts of its rough surface add and cancel — modulates the signal
over seconds. And **Faraday rotation** in the ionosphere twists the
[polarization](/reference/polarization/), so a signal that left vertical can arrive
horizontal and momentarily vanish on a fixed antenna.

## In practice

EME is primarily an amateur-radio and experimental pursuit on the VHF, UHF, and
microwave bands, prized as the ultimate long-haul path: a single Moonbounce contact can
span half the planet with no infrastructure between the two stations. Digital
weak-signal software has widened access enormously — modes designed for sub-noise
decoding let modest stations make contacts that once required enormous arrays. The Moon
was also an early passive relay in military and research communications before active
satellites existed.

## Relevance to SDR

EME is a natural fit for [software-defined radio](/reference/software-defined-radio/):
narrow, weak echoes buried in noise, with a Doppler offset that shifts over a pass, are
exactly what SDR-based weak-signal decoders handle well, and a
[waterfall display](/reference/waterfall-display/) makes the faint trace visible. It has
no connection to trunked land-mobile radio, so **GopherTrunk** neither targets nor
encounters it. EME earns a place in an SDR reference as the extreme case of the same
[path-loss](/reference/path-loss/) and link-budget arithmetic that governs every RF
link — the demonstration that with enough antenna, power, and coding, even a 250 dB path
can be closed.

## Sources

[^wiki]: [Earth–Moon–Earth communication](https://en.wikipedia.org/wiki/Earth%E2%80%93Moon%E2%80%93Earth_communication) — Wikipedia, on lunar reflection, ~250 dB path loss, libration fading, Doppler, and Faraday rotation.
