---
slug: nasa
title: NASA (National Aeronautics and Space Administration)
entry_type: organization
category: organizations
description: "NASA is the US civilian space agency whose deep-space communications work drove practical error-correcting codes, including convolutional and Reed-Muller coding."
keywords: NASA, National Aeronautics and Space Administration, deep space network, error correction, coding, Reed-Muller, Viterbi, telemetry, DSN
aka: [NASA, National Aeronautics and Space Administration]
autolink: true
infobox:
  - { label: Type, value: US government space agency }
  - { label: Founded, value: "1958" }
  - { label: Region, value: United States }
see_also: [reed-muller-code, viterbi-algorithm, forward-error-correction, convolutional-code, gnss]
cite_urls:
  - https://www.nasa.gov/
  - https://en.wikipedia.org/wiki/NASA
---

**NASA** (the **National Aeronautics and Space Administration**) is the civilian space
agency of the United States, established in 1958.[^home] Beyond spaceflight, NASA's need to
recover faint telemetry from distant spacecraft made it a driving force behind practical
[forward error correction](/reference/forward-error-correction/) — the deep-space link is
where codes such as [Reed-Muller](/reference/reed-muller-code/) and convolutional codes
decoded by the [Viterbi algorithm](/reference/viterbi-algorithm/) proved themselves.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 112" role="img" aria-label="A distant spacecraft transmits a weak signal that error-correcting codes recover at NASA's Deep Space Network ground stations." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nasa_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <circle cx="55" cy="40" r="14" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="55" y="70">spacecraft</text>
    <path d="M40 84 Q230 96 405 40" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
    <text x="230" y="80" font-size="7.5">weak signal + FEC (Reed-Muller, convolutional)</text>
    <path d="M395 78 L405 40 L415 78 Z" fill="none" stroke="currentColor" stroke-width="1.2"/><line x1="385" y1="78" x2="425" y2="78" stroke="currentColor" stroke-width="1.2"/><text x="405" y="94">DSN antenna</text>
    <line x1="70" y1="42" x2="392" y2="46" stroke="currentColor" stroke-width="1" marker-end="url(#nasa_ar)"/>
  </g>
</svg>
<figcaption>Deep-space links carry very weak signals, so NASA leaned on strong error-correcting codes to recover the data.</figcaption>
</figure>

## Overview

NASA was created in 1958, absorbing the earlier NACA aeronautics committee, to lead US
civilian efforts in aeronautics and space. Its programs — from the Apollo lunar landings to
the Space Shuttle, the great observatories, the Mars rovers, and the Voyager probes now in
interstellar space — are its public face. Underlying all of them is a communications problem
of extraordinary difficulty: a transmitter the size of a light bulb, billions of kilometers
away, whose signal arrives at Earth vanishingly weak.

Solving that problem made NASA and its Jet Propulsion Laboratory a proving ground for
channel coding. The [Deep Space Network](/reference/gnss/) of large ground antennas, paired
with increasingly powerful error-correcting codes, let missions trade a little bandwidth for
enormous gains in reliable data return. The Mariner missions carried [Reed-Muller
codes](/reference/reed-muller-code/) to protect imagery; later spacecraft adopted
[convolutional codes](/reference/convolutional-code/) decoded with the [Viterbi
algorithm](/reference/viterbi-algorithm/), then concatenated Reed-Solomon-plus-convolutional
schemes, and eventually turbo and LDPC codes, several of which were later standardized for
space through CCSDS with heavy NASA involvement.

## Relevance to SDR

NASA's coding heritage is woven through everyday SDR practice. The same
[forward-error-correction](/reference/forward-error-correction/) families that first earned
their keep on deep-space links — convolutional codes with Viterbi decoding, Reed-Muller and
Reed-Solomon codes — now protect terrestrial digital voice, satellite broadcast, and data
links that SDRs routinely decode. Amateur reception of NASA and other spacecraft telemetry,
weather-satellite imagery, and beacon signals is a popular SDR pursuit, and understanding
the coding NASA helped pioneer is often what separates a locked, decoded frame from noise.

GopherTrunk does not communicate with spacecraft, but it decodes several of the very codes
NASA helped mature: the land-mobile protocols it targets lean on convolutional and
block-coding techniques with the same underlying mathematics. NASA appears in this guide as
the organization whose demanding links pushed error-correcting codes from theory into
routine engineering, benefiting every digital radio that came after.

## Sources

[^home]: [NASA](https://www.nasa.gov/) — the agency's official site, for its missions, the Deep Space Network, and communications work.
[^wiki]: [NASA](https://en.wikipedia.org/wiki/NASA) — Wikipedia, for the agency's history and its role in deep-space communications and coding.
