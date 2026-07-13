---
slug: antenna-diversity
title: Antenna diversity
entry_type: term
category: sdr-dsp
description: "Antenna diversity uses two or more spaced antennas and combines or selects among them to counter multipath fading, improving reliability without more transmit power."
keywords: antenna diversity, spatial diversity, selection combining, maximal ratio combining, MRC, equal gain combining, switched diversity, receive diversity, fading mitigation
aka: [spatial diversity, receive diversity, diversity reception]
autolink: true
infobox:
  - { label: Type, value: Multi-antenna technique }
  - { label: Combats, value: Multipath / fading dropouts }
  - { label: Methods, value: Selection, switched, EGC, MRC }
see_also: [mimo, multipath-propagation, rayleigh-fading, rician-fading, beamforming, antenna-gain]
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_diversity
  - https://en.wikipedia.org/wiki/Diversity_combining
---

**Antenna diversity** uses two or more antennas that receive independently faded copies of the
same signal and combines or selects among them to counter
[multipath fading](/reference/multipath-propagation/), improving link reliability without any
increase in transmit power.[^wiki] The premise is statistical: fades are localized, so when one
antenna sits in a deep null another — spaced far enough apart to fade independently — is likely
in a strong spot. Taking the better copy, or a weighted sum, makes a simultaneous deep fade on
*all* branches rare, which is what raises reliability.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="Two spaced antennas receiving the same signal over different multipath, feeding a combiner that outputs a single improved signal by selection or maximal-ratio combining." xmlns="http://www.w3.org/2000/svg">
  <line x1="55" y1="40" x2="55" y2="80" stroke="currentColor" stroke-width="1.4"/>
  <path d="M48 34 L55 42 L62 34" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="55" y1="120" x2="55" y2="160" stroke="currentColor" stroke-width="1.4"/>
  <path d="M48 114 L55 122 L62 114" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="55" y="98" font-size="8" fill="currentColor" text-anchor="middle">branch 1</text>
  <text x="55" y="178" font-size="8" fill="currentColor" text-anchor="middle">branch 2</text>
  <path d="M70 60 C 150 60 150 60 210 80" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#adar)"/>
  <path d="M70 140 C 150 140 150 120 210 100" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#adar)"/>
  <rect x="212" y="70" width="90" height="42" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="257" y="87" font-size="9" fill="currentColor" text-anchor="middle">combiner</text>
  <text x="257" y="101" font-size="7.5" fill="currentColor" text-anchor="middle">select / MRC</text>
  <line x1="302" y1="91" x2="360" y2="91" stroke="currentColor" stroke-width="1.3" marker-end="url(#adar)"/>
  <text x="395" y="94" font-size="9" fill="currentColor" text-anchor="middle">best output</text>
  <defs><marker id="adar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Two independently faded branches feed a combiner; selecting the stronger branch or summing them in phase makes a simultaneous deep fade on both unlikely.</figcaption>
</figure>

## How it works

The branches must be **decorrelated** so their fades are independent. That is achieved by
spacing antennas apart (spatial diversity — roughly half a wavelength or more indoors, farther
outdoors), by using orthogonal [polarizations](/reference/polarization/), or by different
radiation patterns (pattern diversity). Given decorrelated branches, several combining rules
trade complexity against gain:

- **Selection combining** picks the single branch with the highest instantaneous signal (or
  SNR). One receiver chain suffices if a switch samples both; cheap and effective against
  dropouts.
- **Switched diversity** stays on the current branch until it fades below a threshold, then
  switches — even cheaper, with a small penalty.
- **Equal-gain combining (EGC)** co-phases the branches and adds them with equal weight, needing
  a receiver per branch but no amplitude weighting.
- **Maximal-ratio combining (MRC)** co-phases the branches and weights each by its own SNR before
  summing. It is optimal for additive noise: the output SNR is the *sum* of the branch SNRs, so
  two equal branches gain 3 dB even with no fade, and much more when one branch is deep in a
  fade.

The theoretical benefit is measured as **diversity order** — with N independent branches the
probability of a deep fade drops roughly as the Nth power of a single branch's fade probability,
which is why even two antennas dramatically cut dropout rates in a
[Rayleigh-fading](/reference/rayleigh-fading/) channel.

## Relevance to SDR

Antenna diversity is everywhere in modern wireless: Wi-Fi access points, cellular handsets and
base stations, DECT phones, and vehicular receivers all use it, and it is the receive-side
foundation that [MIMO](/reference/mimo/) generalizes into spatial multiplexing. In land-mobile
radio, base-station **voting** receivers are a form of selection diversity across geographically
separate sites. For an SDR listener, diversity requires multiple coherent (or at least
independently sampled) front ends; a phase-coherent array such as KrakenSDR can implement MRC in
software. GopherTrunk is a single-stream decoder — it processes one [I/Q](/reference/iq-data/)
capture at a time and does not combine multiple antennas — so diversity is relevant to the
broader RF context and to hardware choices rather than something GT itself performs.

## Sources

[^wiki]: [Antenna diversity](https://en.wikipedia.org/wiki/Antenna_diversity) — Wikipedia, on spatial diversity and its role against multipath fading.
