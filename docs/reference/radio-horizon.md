---
slug: radio-horizon
title: Radio horizon
entry_type: term
category: propagation
description: The radio horizon is the farthest distance a line-of-sight signal reaches before the Earth's curvature blocks it, slightly beyond the visual horizon due to atmospheric refraction.
keywords: radio horizon, line of sight, Earth curvature, antenna height, coverage range, 4/3 earth, tropospheric refraction
aka: [radio horizon]
autolink: true
infobox:
  - { label: Type, value: Propagation limit }
  - { label: Extended by, value: Antenna height }
  - { label: Vs visual horizon, value: Slightly farther }
see_also: [radio-propagation, ground-wave, free-space-path-loss, frequency-bands, antenna]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Line-of-sight_propagation
  - https://en.wikipedia.org/wiki/Horizon
---

The **radio horizon** is the farthest point a line-of-sight signal reaches before the
Earth's curvature gets in the way.[^wiki] It lies slightly beyond the visual horizon because
the atmosphere refracts radio waves a little, bending them gently around the curve rather
than letting them fly straight off into space.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A curved earth with a tall antenna whose line of sight reaches further around the curve than a short antenna." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 140 Q230 80 450 140" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <line x1="120" y1="118" x2="120" y2="40" stroke="currentColor" stroke-width="2"/><text x="120" y="34" text-anchor="middle" font-size="8" fill="currentColor">tall</text>
  <line x1="120" y1="40" x2="370" y2="106" stroke="currentColor" stroke-width="1.3" stroke-dasharray="5 3"/>
  <circle cx="370" cy="106" r="3" fill="currentColor"/><text x="385" y="104" font-size="8" fill="currentColor">horizon</text>
  <text x="230" y="135" text-anchor="middle" font-size="9" fill="currentColor">height extends the radio horizon</text>
</svg>
<figcaption>The radio horizon is the farthest line-of-sight point before the Earth's curve blocks it; raising the antenna extends it.</figcaption>
</figure>

## How it works

Because the Earth curves away, an antenna at height *h* can "see" only out to where its
straight sight line grazes the surface. Geometry alone puts that distance at roughly
3.57·√h kilometres for *h* in metres. But the lower atmosphere is denser near the ground and
thins with altitude, so it refracts radio waves downward slightly, letting them follow the
curve a bit farther than light does. Engineers model this with the **4/3 Earth radius**
approximation, which stretches the constant to about **4.12·√h km**.[^horizon] The distance
between two stations is the sum of each one's horizon distance:

- d(km) ≈ 4.12·(√h_tx + √h_rx), with heights in metres.
- A 30 m tower reaches about 22.5 km to the horizon on its own.
- Two 30 m towers can therefore work each other out to roughly 45 km.

The rule is nearly frequency-independent across VHF and UHF — a UHF and a VHF signal from
the same tower reach about the same horizon — so **antenna height, not band, sets the range**
for line-of-sight work.

## In practice

This is why repeaters and trunked [trunking sites](/reference/trunking-site/) sit on towers,
rooftops, and hilltops, and why raising a receive antenna even a few metres can pull in
systems that were invisible at ground level. Beyond the horizon, signals fall off fast, but
not to nothing: diffraction fills in some of the shadow, and occasional
**tropospheric ducting** can carry VHF/UHF far past the normal limit when atmospheric layers
form a waveguide. At lower frequencies, [ground wave](/reference/ground-wave/) and
[sky wave](/reference/sky-wave/) reach well beyond the horizon by entirely different
mechanisms, so the radio-horizon limit chiefly governs VHF and above.

## Relevance to SDR

For line-of-sight bands, antenna height is often the most effective way to reach more distant
systems — usually more so than a higher-gain antenna or a better radio. Estimating the radio
horizon from your antenna height and a target site's tower height tells you quickly whether a
system is even reachable before you spend time chasing it. GopherTrunk cannot recover a
transmitter that sits below your horizon; the practical response is to get the antenna up and
clear, which extends the horizon and improves the [link budget](/reference/link-budget/) at
the same time.

## Sources

[^wiki]: [Line-of-sight propagation](https://en.wikipedia.org/wiki/Line-of-sight_propagation) — Wikipedia, on the radio horizon, Earth curvature, and atmospheric refraction.
[^horizon]: [Horizon](https://en.wikipedia.org/wiki/Horizon) — Wikipedia, for the geometric horizon distance and the 4/3-Earth radio-horizon correction.
