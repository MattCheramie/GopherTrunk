---
slug: channelizer
title: Channelizer
entry_type: algorithm
category: sdr-dsp
description: A channelizer splits one wideband IQ capture into many narrow channels at once — how one SDR can follow a control channel and several voice channels in parallel.
keywords: channelizer, channelization, polyphase, wideband, multi-channel, DDC, parallel decode
aka: [channelizer, channelization]
autolink: true
see_also: [digital-down-converter, decimation, digital-filter, software-defined-radio, control-channel]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Filter bank (Wikipedia)", url: https://en.wikipedia.org/wiki/Filter_bank }
---

A **channelizer** splits a single wideband [IQ](/reference/iq-data/) capture into **many
narrow channels at once**. Each output is one channel, shifted to
[baseband](/reference/baseband/), [filtered](/reference/digital-filter/), and
[decimated](/reference/decimation/) — so one SDR can feed many decoders in parallel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A wide input band fanning out into several separate narrow channel outputs." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="50" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="65" y="69" text-anchor="middle" font-size="9" fill="currentColor">wide IQ</text>
  <rect x="180" y="44" width="90" height="42" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="225" y="69" text-anchor="middle" font-size="9" fill="currentColor">channelizer</text>
  <line x1="110" y1="65" x2="179" y2="65" stroke="currentColor" marker-end="url(#char)"/>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="340" y="22" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="37">control ch</text>
    <rect x="340" y="54" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="69">voice ch 1</text>
    <rect x="340" y="86" width="100" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="101">voice ch 2</text>
  </g>
  <g stroke="currentColor" stroke-width="1"><line x1="270" y1="60" x2="339" y2="33" marker-end="url(#char)"/><line x1="270" y1="65" x2="339" y2="65" marker-end="url(#char)"/><line x1="270" y1="70" x2="339" y2="97" marker-end="url(#char)"/></g>
  <defs><marker id="char" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A channelizer fans one wide capture into many narrow channels, enabling simultaneous decodes.</figcaption>
</figure>

## Overview

Efficient channelizers use a *polyphase filter bank* to extract evenly-spaced channels
at once. In a trunking context this is what lets GopherTrunk watch the
[control channel](/reference/control-channel/) while simultaneously following the
[voice channels](/reference/voice-channel/) it assigns.
