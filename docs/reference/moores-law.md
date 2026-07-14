---
slug: moores-law
title: Moore's law
entry_type: concept
category: hw-foundations
description: Moore's law is the observation that the number of transistors on an integrated circuit roughly doubles about every two years, driving decades of exponential gains in computing.
keywords: Moore's law, transistor density, Gordon Moore, exponential, scaling, integrated circuit, semiconductor
aka: [Moore's Law]
autolink: true
infobox:
  - { label: Type, value: Empirical observation }
  - { label: Stated by, value: "Gordon Moore, 1965" }
  - { label: Claim, value: Transistor count doubles ~2 yrs }
  - { label: Status, value: Slowing }
see_also: [transistor, integrated-circuit, semiconductor, central-processing-unit, clock-speed, graphics-processing-unit]
cite_urls:
  - https://en.wikipedia.org/wiki/Moore%27s_law
---

**Moore's law** is the observation, made by Intel co-founder Gordon Moore in 1965, that the number of [transistors](/reference/transistor/) on an [integrated circuit](/reference/integrated-circuit/) roughly doubles about every two years.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 214" role="img" aria-label="A chart on a logarithmic vertical axis. Transistor count rises as a straight line, meaning exponential doubling roughly every two years. Clock speed rises alongside it until the mid-2000s, then flattens as Dennard scaling ends, pushing designers toward multiple cores." xmlns="http://www.w3.org/2000/svg">
  <line x1="52" y1="28" x2="52" y2="176" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6"/>
  <line x1="52" y1="176" x2="432" y2="176" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6"/>
  <g stroke="currentColor" stroke-width="0.6" stroke-opacity="0.2">
    <line x1="52" y1="140" x2="432" y2="140"/><line x1="52" y1="104" x2="432" y2="104"/><line x1="52" y1="68" x2="432" y2="68"/><line x1="52" y1="32" x2="432" y2="32"/>
  </g>
  <text x="24" y="105" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(-90 24 105)">count (log)</text>
  <g font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle">
    <text x="80" y="190">1970</text><text x="200" y="190">1990</text><text x="320" y="190">2010</text>
  </g>
  <line x1="66" y1="168" x2="420" y2="40" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="52" font-size="8" fill="currentColor" font-weight="600">transistors · 2× / 2 yr</text>
  <path d="M66 158 L150 128 L240 92 L300 78 Q340 74 420 74" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="392" y="88" font-size="8" fill="currentColor" fill-opacity="0.9">clock speed</text>
  <line x1="300" y1="28" x2="300" y2="176" stroke="currentColor" stroke-width="0.9" stroke-opacity="0.45" stroke-dasharray="3 3"/>
  <text x="300" y="24" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">~2005: Dennard scaling ends</text>
  <text x="242" y="208" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">clocks stopped rising → progress shifted to more cores &amp; accelerators</text>
</svg>
<figcaption>On a log axis, steady doubling is a straight line — that's Moore's law, an empirical trend, not a physical one. Clock speed rode along until the mid-2000s, when Dennard scaling broke and frequencies plateaued. Transistors kept growing, so the industry turned to multiple cores, specialized accelerators, and advanced packaging.</figcaption>
</figure>

## Overview

It is not a law of physics but an empirical trend that held for decades as [semiconductor](/reference/semiconductor/) makers steadily shrank feature sizes, packing ever more switches onto a chip at falling cost per transistor. That exponential scaling is what turned room-sized computers into pocket-sized ones and made each generation of [CPUs](/reference/central-processing-unit/) and [GPUs](/reference/graphics-processing-unit/) dramatically more capable. A related trend, Dennard scaling, once let [clock speeds](/reference/clock-speed/) rise as transistors shrank — but that broke down in the mid-2000s, pushing designers toward multiple cores.

## Where it fits

Moore's law explains why computing got cheap and abundant enough that a hobbyist can run real DSP — like GopherTrunk decoding multiple trunked channels — on a board costing a few tens of dollars. The doubling has slowed as features approach atomic scale, so progress now leans more on parallelism, specialized accelerators, and packaging than on raw shrinking.

## Sources

[^wiki]: [Moore's law](https://en.wikipedia.org/wiki/Moore%27s_law) — Wikipedia, on the transistor-doubling trend and its slowdown.
