---
slug: fade-margin
title: Fade margin
entry_type: term
category: propagation
description: Fade margin is the extra decibels a link is designed with above its minimum usable level, so brief fades do not push the signal below threshold.
keywords: fade margin, link margin, system operating margin, fading headroom, availability, outage probability, link budget headroom
aka: [fade margin, link margin, system operating margin]
autolink: true
infobox:
  - { label: Type, value: Link-budget headroom }
  - { label: Unit, value: Decibels (dB) }
  - { label: Buys, value: Availability against fading }
see_also: [link-budget, rayleigh-fading, rician-fading, rain-fade, free-space-path-loss]
cite_urls:
  - https://en.wikipedia.org/wiki/Fade_margin
  - https://en.wikipedia.org/wiki/Link_budget
---

**Fade margin** is the number of decibels by which a link's normal received signal level
exceeds the minimum level the receiver needs to work — the headroom set aside so that
short-term [fading](/reference/rayleigh-fading/) does not drop the signal below
threshold.[^wiki] It falls straight out of the [link budget](/reference/link-budget/): once
you know the expected received power and the receiver's required
[SNR](/reference/signal-to-noise-ratio/) or sensitivity, the difference is your margin. A
link with too little margin works in fine weather but drops out on every deep fade.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A horizontal bar diagram: received signal level near the top, a receiver threshold lower down, and the gap between them labelled fade margin; a fading trace dips toward the threshold but stays above it." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="20" x2="40" y2="140" stroke="currentColor" stroke-width="1.1"/>
  <text x="28" y="30" font-size="9" fill="currentColor" transform="rotate(-90 28 82)">level (dB)</text>
  <line x1="40" y1="45" x2="440" y2="45" stroke="currentColor" stroke-width="1.4"/>
  <text x="330" y="40" font-size="9" fill="currentColor">mean received level</text>
  <line x1="40" y1="110" x2="440" y2="110" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/>
  <text x="330" y="124" font-size="9" fill="currentColor">receiver threshold</text>
  <line x1="120" y1="46" x2="120" y2="109" stroke="currentColor" stroke-width="1" marker-start="url(#fmar)" marker-end="url(#fmar)"/>
  <text x="128" y="82" font-size="9" fill="currentColor">fade margin</text>
  <path d="M160 50 Q185 70 210 55 Q235 100 260 62 Q285 105 310 58 Q335 95 360 52 Q385 88 410 56" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <defs><marker id="fmar" markerWidth="8" markerHeight="8" refX="4" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Fade margin is the gap between the mean received level and the receiver threshold; deeper margin survives deeper fades.</figcaption>
</figure>

## How it works

Fade margin is simply

- `M = P_received − P_threshold`  (all in dB),

where `P_received` is the level predicted by the [link budget](/reference/link-budget/)
after [free-space path loss](/reference/free-space-path-loss/), antenna gains, and fixed
losses, and `P_threshold` is the receiver's minimum usable level (its
[sensitivity](/reference/receiver-sensitivity/), or the level giving the required SNR).

How much margin you need is dictated by the **fading statistics** of the channel and the
**availability** you are targeting:

- On a deeply fading [Rayleigh](/reference/rayleigh-fading/) channel, deep fades are common,
  so hitting even 99% availability demands roughly 20 dB of margin, and 99.9% demands about
  30 dB — each extra "nine" costs about 10 dB.
- On a gentler [Rician](/reference/rician-fading/) channel with a strong line-of-sight
  component, the same availability needs far less margin, because deep fades are rarer.
- On microwave links the driver is often [rain fade](/reference/rain-fade/), and margin is
  sized from rain-rate statistics for the target percentage of the year.

The relationship is non-linear: because fade depth grows slowly with rarity, buying the last
fraction of a percent of availability costs disproportionately many decibels.

## In practice

Because margin is expensive, engineers spend it wisely and combine it with techniques that
lower the *required* margin rather than just cranking up power:

- **Antenna gain and height** raise the mean received level directly and, by improving
  line-of-sight, push the channel toward [Rician](/reference/rician-fading/) statistics.
- **Diversity** (space, frequency, or polarisation) makes a deep fade on one branch unlikely
  to coincide with one on another, sharply cutting the margin needed for a given outage.
- **Forward error correction and interleaving** let the receiver tolerate a lower
  instantaneous SNR, effectively lowering the threshold and so freeing up margin.
- **Adaptive coding and modulation** trades data rate for robustness on the fly, keeping the
  link alive through a fade instead of budgeting fixed margin for it.

## Relevance to SDR

Fade margin is why a scanner's reception is reliable in some spots and marginal in others.
A strong average signal from a nearby [trunking site](/reference/trunking-site/) carries
plenty of margin, so occasional deep fades never cross threshold and the audio stays clean.
Near the fringe of coverage the margin shrinks to a few dB, and every fade produces a burst
of errors or a dropout — the audible signature of running out of margin. Raising the
antenna, adding gain, or reducing feedline loss all buy margin back.

[GopherTrunk](/reference/software-defined-radio/) does not compute a link budget, but its
per-frame [SNR](/reference/signal-to-noise-ratio/) and
[EVM](/reference/error-vector-magnitude/) readouts are effectively a live margin meter: a
comfortable SNR above the demodulator's floor means healthy margin, while values hovering
near the decode threshold warn that the link has little headroom left against the next fade.

## Sources

[^wiki]: [Fade margin](https://en.wikipedia.org/wiki/Fade_margin) — Wikipedia, on the difference between received level and threshold and its role in link availability.
