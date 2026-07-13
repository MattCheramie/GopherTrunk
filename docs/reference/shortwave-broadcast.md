---
slug: shortwave-broadcast
title: Shortwave broadcasting
entry_type: technology
category: broadcast
description: "Shortwave broadcasting is long-distance HF radio using sky-wave ionospheric propagation, carried mostly in AM with some digital DRM."
keywords: shortwave broadcasting, shortwave radio, HF broadcasting, sky wave, ionospheric propagation, international broadcasting, DRM, AM shortwave, 3-30 MHz, broadcast bands
aka: [shortwave radio, SW broadcasting, HF broadcasting, international broadcasting]
autolink: true
infobox:
  - { label: Type, value: HF broadcast service }
  - { label: Band, value: "~3–30 MHz (HF), in designated broadcast segments" }
  - { label: Idea, value: Sky-wave propagation for intercontinental reach }
  - { label: Modes, value: Mostly double-sideband AM; some DRM }
see_also: [sky-wave, amplitude-modulation, drm-broadcast, ionospheric-propagation, ground-wave, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/Shortwave_radio
  - https://en.wikipedia.org/wiki/Shortwave_bands
---

**Shortwave broadcasting** is international and long-distance radio broadcasting in the
high-frequency (HF) range of roughly 3–30 MHz, using designated broadcast segments
scattered across the band.[^wiki] Its defining feature is not its modulation — which is
mostly ordinary double-sideband [AM](/reference/amplitude-modulation/) — but its reach:
HF signals refract off the ionosphere as a [sky wave](/reference/sky-wave/), returning to
earth hundreds or thousands of kilometres from the transmitter and enabling a single
station to cover continents.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A shortwave signal leaving a transmitter, refracting off the ionospheric layer, and returning to a distant receiver beyond the horizon in a single sky-wave hop." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 165 Q230 175 440 165" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="230" y="150" text-anchor="middle" font-size="8" fill="currentColor" opacity="0.7">earth</text>
  <line x1="30" y1="55" x2="430" y2="55" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="5 4"/>
  <text x="60" y="48" font-size="8" fill="currentColor">ionosphere</text>
  <path d="M40 160 L225 60" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <path d="M225 60 L410 160" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <rect x="30" y="155" width="12" height="12" fill="currentColor"/>
  <text x="36" y="180" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <rect x="404" y="155" width="12" height="12" fill="none" stroke="currentColor"/>
  <text x="410" y="180" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <text x="225" y="52" text-anchor="middle" font-size="8" fill="currentColor">refraction</text>
</svg>
<figcaption>One sky-wave hop: an HF signal refracts off the ionosphere and returns far beyond the transmitter's horizon.</figcaption>
</figure>

## How it works

Below the maximum usable frequency, HF waves striking the ionosphere are bent back
toward the ground rather than escaping to space, so a wave launched at a shallow angle
returns as a "hop" far downrange; multiple hops can circle much of the globe. Because
ionospheric density varies with the sun, the usable bands shift dramatically between day
and night: lower shortwave bands (e.g. 49 m, 41 m) carry best after dark, while higher
bands (19 m, 16 m, 13 m) open during daylight. Broadcasters therefore change frequency by
time of day and season, and coordinate schedules through the HFCC. Signals also fade
(QSB) as multiple paths interfere, and long multi-hop paths add characteristic distortion.

The dominant mode is double-sideband AM, chosen for receiver simplicity across a huge and
cheap installed base, typically with a few kHz of audio bandwidth. A minority of
transmissions use single-sideband or the digital [DRM](/reference/drm-broadcast/) system,
which packs a COFDM waveform into the same ~10 kHz channel to deliver near-FM audio
quality where propagation allows.

Close to the transmitter, a [ground wave](/reference/ground-wave/) provides reliable local
coverage, but the long-haul reach that defines shortwave comes entirely from the sky wave.
Because a given path only supports a limited range of frequencies at any moment — bounded
above by the maximum usable frequency and below by absorption in the lower ionosphere —
broadcasters simulcast the same programme on several bands so listeners can find whichever
one is propagating. This constant band-hopping, and the audible flutter of multi-hop
paths, are the sonic signature of HF broadcasting.

## Relevance to SDR

Shortwave is a rewarding SDR band because a wideband HF SDR — a direct-sampling receiver
or an upconverter feeding a VHF dongle — can display the whole 3–30 MHz range at once,
letting an operator watch broadcast bands open and close with the ionosphere. AM
demodulation is trivial (envelope or synchronous detection), and the same receiver reveals
utility stations, amateur traffic, and [DRM](/reference/drm-broadcast/) blocks that
decoding software can turn back into audio. It is a live demonstration of
[ionospheric propagation](/reference/ionospheric-propagation/) that VHF/UHF listening never
shows.

**GopherTrunk** is a VHF/UHF trunked land-mobile decoder and does nothing with HF
broadcasting; the two occupy different worlds of the spectrum. Shortwave is included here
as essential propagation context — the clearest everyday example of the
[sky-wave](/reference/sky-wave/) mechanism that governs why some signals travel far beyond
the horizon and others do not.

## Sources

[^wiki]: [Shortwave radio](https://en.wikipedia.org/wiki/Shortwave_radio) — Wikipedia, for the HF broadcast bands, sky-wave ionospheric propagation, day/night band selection, and the use of AM with some DRM.
