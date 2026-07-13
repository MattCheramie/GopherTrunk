---
slug: starlink
title: Starlink
entry_type: technology
category: satellite-gnss
description: "Starlink is SpaceX's low-earth-orbit broadband network using thousands of satellites and an electronically steered Ku/Ka-band phased-array user terminal that tracks satellites without moving."
keywords: Starlink, SpaceX, LEO broadband, phased array, beamforming, Ku-band, Ka-band, user terminal, Dishy, satellite internet, electronically steered antenna
aka: [Starlink]
autolink: true
infobox:
  - { label: Type, value: LEO broadband constellation }
  - { label: Idea, value: Electronically steered Ku/Ka phased-array terminal }
  - { label: Examples, value: SpaceX Starlink user terminal ("Dishy") }
see_also: [phased-array-antenna, beamforming, ofdm, frequency-bands, iridium, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Starlink
  - https://en.wikipedia.org/wiki/Phased_array
---

**Starlink** is SpaceX's satellite-internet system: a mega-constellation of thousands of
[low-earth-orbit](/reference/frequency-bands/) satellites at around 550 km, paired with a
flat user terminal that connects homes and vehicles to broadband. Its defining piece of
RF engineering is that terminal — an electronically steered
[phased-array antenna](/reference/phased-array-antenna/) that tracks fast-moving
satellites across the sky with no moving parts, forming and re-pointing its beam purely
by adjusting the phase of hundreds of antenna elements.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A flat Starlink phased-array terminal steers a beam electronically from one low-earth-orbit satellite to the next as they cross the sky." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="slink" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="150" y="150" width="160" height="12" rx="2" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <text x="230" y="176" text-anchor="middle" font-size="9" fill="currentColor">flat phased-array terminal (no moving parts)</text>
  <circle cx="120" cy="35" r="7" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <circle cx="340" cy="35" r="7" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="230" y="22" text-anchor="middle" font-size="9" fill="currentColor">LEO satellites crossing overhead (~550 km)</text>
  <line x1="215" y1="150" x2="125" y2="44" stroke="currentColor" stroke-dasharray="3 3"/>
  <line x1="245" y1="150" x2="335" y2="44" stroke="currentColor" marker-end="url(#slink)"/>
</svg>
<figcaption>The Starlink terminal is a phased array: it steers and hands off its beam electronically as satellites sweep across the sky in minutes.</figcaption>
</figure>

## How it works

A satellite at 550 km is only in view for a few minutes and moves fast across the sky, so
a fixed dish would need constant mechanical aiming. Starlink instead uses a
**[phased array](/reference/phased-array-antenna/)**: many small radiating elements whose
signals are combined with controlled phase offsets. By stepping those phases, the array
synthesises a beam pointed in any direction and slews it electronically — the essence of
**[beamforming](/reference/beamforming/)** — then hands off from one satellite to the next
without ever moving.[^array] The links run in the microwave **Ku** and **Ka** bands
(roughly 10.7–12.7 GHz user downlink, ~14 GHz and higher for uplink), using wide channels
and a high-order [OFDM](/reference/ofdm/)-style waveform to pack in gigabit-class capacity.
Later satellites add laser inter-satellite links so traffic can hop between spacecraft
without a ground station under every satellite.

The user terminal also has to cope with a large, continuously changing
[Doppler shift](/reference/doppler-shift/) as the satellite races overhead, and with the
free-space path loss of a microwave link over hundreds of kilometres — hence the array's
high gain and an integrated [low-noise amplifier](/reference/low-noise-amplifier/) front
end.

## Relevance to SDR

Starlink is not a hobbyist decode target: the user data is encrypted and the waveform is
proprietary, so there is no open payload to demodulate the way there is for
[Orbcomm](/reference/orbcomm/) or the [NOAA APT](/reference/noaa-apt/) weather birds. What
Starlink *is* useful for is as the highest-profile real-world example of an electronically
steered phased array in consumer hands — the same beamforming principle behind military
radar, 5G base stations, and direction-finding receivers. Researchers have also exploited
Starlink's synchronisation and beacon sequences as an opportunistic positioning signal,
demonstrating GPS-independent navigation from the constellation's downlink structure.

GopherTrunk is a terrestrial land-mobile trunking receiver and does nothing with Starlink;
the entry sits in this guide to connect the phased-array and beamforming articles to a
system readers will recognise. GopherTrunk is a pure receiver with no phased-array
hardware, so it does not beamform — it relates to these concepts only as background RF
theory.

## Sources

[^wiki]: [Starlink](https://en.wikipedia.org/wiki/Starlink) — Wikipedia, for the constellation, orbit, Ku/Ka-band links, and the phased-array user terminal.
[^array]: [Phased array](https://en.wikipedia.org/wiki/Phased_array) — Wikipedia, for how phase control across elements steers a beam electronically.
