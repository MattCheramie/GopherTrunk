---
slug: rest-channel
title: Rest channel
entry_type: term
category: trunked-radio
description: In trunked systems with a rotating control channel, the rest channel is the one currently carrying control signalling — it moves as calls are assigned.
keywords: rest channel, distributed control channel, DMR Capacity Plus, rotating control channel, trunking
aka: ["rest channel"]
autolink: true
see_also: [control-channel, channel-grant, trunked-radio, capacity-plus, dmr]
related_lessons:
  - { title: "Finding & identifying systems", url: /learn/finding-systems/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

A **rest channel** is the channel currently carrying control signalling in trunked
systems that **rotate the control function around the pool** rather than dedicate one
frequency to it. When a call is assigned to the current rest channel, control moves to
another idle channel — the new rest channel.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A pool of channels where one is marked as the rest channel carrying control; an arrow shows the rest role moving to another channel when a call starts." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="40" y="40" width="80" height="32" fill="currentColor" fill-opacity="0.22"/><text x="80" y="56">ch 1</text><text x="80" y="66" font-size="7">rest (control)</text>
    <rect x="140" y="40" width="80" height="32" fill="none"/><text x="180" y="60">ch 2</text>
    <rect x="240" y="40" width="80" height="32" fill="none"/><text x="280" y="60">ch 3</text>
    <rect x="340" y="40" width="80" height="32" fill="none"/><text x="380" y="60">ch 4</text>
  </g>
  <path d="M80 74 q60 30 120 0" fill="none" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#rcar)"/>
  <text x="150" y="108" text-anchor="middle" font-size="8" fill="currentColor">control moves when this channel takes a call</text>
  <defs><marker id="rcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>With a rotating control channel, the "rest channel" is wherever control currently lives, and it moves as calls are assigned.</figcaption>
</figure>

## Overview

Rotating control (used by Motorola [Capacity Plus](/reference/capacity-plus/) and some
DMR systems) complicates monitoring: a scanner must follow the rest channel as it hops,
rather than camping on one fixed [control channel](/reference/control-channel/).
