---
slug: rician-fading
title: Rician fading
entry_type: term
category: propagation
description: Rician fading models a multipath channel that keeps a dominant line-of-sight component; the envelope follows a Rice distribution set by the K-factor.
keywords: Rician fading, Ricean fading, Rice distribution, K-factor, line-of-sight, LOS multipath, dominant component, fading
aka: [Rician fading, Ricean fading, Rice fading]
autolink: true
infobox:
  - { label: Type, value: Small-scale fading model }
  - { label: Condition, value: Dominant line-of-sight + scatter }
  - { label: Key parameter, value: Rice K-factor (dB) }
see_also: [rayleigh-fading, multipath-propagation, radio-propagation, fade-margin, fresnel-zone]
cite_urls:
  - https://en.wikipedia.org/wiki/Rician_fading
  - https://en.wikipedia.org/wiki/Fading
---

**Rician fading** (also spelled Ricean) models a [multipath](/reference/multipath-propagation/)
channel in which a strong, steady **line-of-sight (LOS)** component arrives alongside
the many weaker scattered rays.[^wiki] The dominant term stabilises the envelope, so
fades are shallower and less frequent than in the no-LOS case. Rician fading is
parameterised by the **K-factor**, the power ratio of the dominant component to the
scattered power; it bridges the calm and the harsh ends of the mobile channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="Two probability density curves: a Rician distribution with a strong peak for high K-factor, and a broader Rayleigh-like curve for K equal to zero, showing that a line-of-sight component narrows the fading spread." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="20" x2="40" y2="140" stroke="currentColor" stroke-width="1.2"/>
  <line x1="40" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="1.2" marker-end="url(#ricar)"/>
  <text x="240" y="160" text-anchor="middle" font-size="9" fill="currentColor">envelope amplitude</text>
  <text x="28" y="30" font-size="9" fill="currentColor" transform="rotate(-90 28 82)">probability</text>
  <path d="M40 140 Q80 138 105 96 Q125 62 150 74 Q182 92 220 130 Q235 138 260 140" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="120" y="55" font-size="8" fill="currentColor">K = 0 (Rayleigh)</text>
  <path d="M40 140 L250 140 Q280 139 300 100 Q315 40 330 45 Q345 50 360 108 Q378 139 400 140" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="330" y="34" font-size="8" fill="currentColor" text-anchor="middle">large K (strong LOS)</text>
  <defs><marker id="ricar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>As the K-factor rises, the envelope clusters near the line-of-sight level; at K = 0 the distribution collapses to Rayleigh.</figcaption>
</figure>

## How it works

As with [Rayleigh fading](/reference/rayleigh-fading/), the scattered rays make the I
and Q components Gaussian — but here they have a **non-zero mean** contributed by the
LOS ray. The envelope then follows the **Rice distribution**, whose shape is governed by

- `K = A² / 2σ²`, the ratio of dominant power `A²` to scattered power `2σ²`, usually
  quoted in dB.

The two limits are instructive:

- **K → 0** (no dominant path): the mean vanishes and the Rice distribution becomes the
  [Rayleigh](/reference/rayleigh-fading/) distribution — the harshest case.
- **K → ∞** (pure LOS, no scatter): the envelope becomes constant, an ideal
  non-fading channel.

Real links sit between these. A rooftop-to-rooftop path or an open rural road might show
K of 6–15 dB (mostly LOS, small ripple), while a receiver just inside building clutter
might fall to 0–3 dB, approaching Rayleigh. Keeping the first
[Fresnel zone](/reference/fresnel-zone/) clear is what preserves a high K-factor.

## Relevance to SDR

Whether a channel is Rician or Rayleigh decides how reliably a digital signal decodes.
A scanner with a clear view of a [trunking site](/reference/trunking-site/) antenna sees
a high-K Rician channel: the envelope barely moves, EVM stays low, and P25 or DMR frames
decode cleanly. Move indoors or behind a hill and the K-factor drops toward zero, deep
fades appear, and error rates climb. This is the quantitative reason that antenna siting
and height matter so much for scanner reception — height buys line-of-sight, which buys
K-factor, which buys reliability.

Rician statistics also describe satellite and aeronautical links, where a strong direct
ray usually dominates weak ground reflections. [GopherTrunk](/reference/software-defined-radio/)
does not model the channel explicitly, but its reported per-frame
[SNR](/reference/signal-to-noise-ratio/) and [EVM](/reference/error-vector-magnitude/)
effectively reveal the fading regime: a steady low-EVM stream indicates a high-K path,
while bursty errors with a strong average signal indicate the channel has slipped toward
Rayleigh.

## In practice

Because a Rician channel fades less deeply than a Rayleigh one, it needs a smaller
[fade margin](/reference/fade-margin/) to hit the same outage target — every decibel of
K-factor directly buys back link headroom. Link planners therefore estimate K from the
geometry (clearance, terrain, foliage) and choose margin accordingly, budgeting near the
Rayleigh worst case only when no LOS can be guaranteed.

## Sources

[^wiki]: [Rician fading](https://en.wikipedia.org/wiki/Rician_fading) — Wikipedia, on the line-of-sight-plus-scatter model, the Rice distribution, and the K-factor.
