---
slug: capacity-plus
title: Capacity Plus
entry_type: protocol
category: protocols
description: Capacity Plus is Motorola's proprietary DMR trunking mode that pools channels and rotates the control signalling, adding trunked capacity to conventional DMR.
keywords: Capacity Plus, Cap Plus, Motorola DMR trunking, MOTOTRBO, rest channel, single-site trunking
aka: [Capacity Plus, "Cap Plus", "Capacity Max"]
autolink: true
see_also: [dmr, dmr-tier-3, rest-channel, trunked-radio, csbk]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

**Capacity Plus** is Motorola's proprietary **DMR trunking** mode (part of the MOTOTRBO
family) that pools several channels and **rotates the control signalling** among them
(see [rest channel](/reference/rest-channel/)), giving conventional [DMR](/reference/dmr/)
trunked capacity without a separate dedicated control channel. (Capacity Max is the
larger multi-site successor.)

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A pool of DMR channels with control rotating among them, several carrying two TDMA voice slots." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="40" y="40" width="90" height="40" fill="currentColor" fill-opacity="0.2"/><line x1="85" y1="40" x2="85" y2="80"/><text x="85" y="95">rest/control</text>
    <rect x="150" y="40" width="90" height="40" fill="none"/><line x1="195" y1="40" x2="195" y2="80"/><text x="195" y="95">voice</text>
    <rect x="260" y="40" width="90" height="40" fill="none"/><line x1="305" y1="40" x2="305" y2="80"/><text x="305" y="95">voice</text>
    <rect x="370" y="40" width="70" height="40" fill="none"/><line x1="405" y1="40" x2="405" y2="80"/><text x="405" y="95">voice</text>
  </g>
  <text x="230" y="22" text-anchor="middle" font-size="8.5" fill="currentColor">pooled two-slot DMR channels; control rotates</text>
</svg>
<figcaption>Capacity Plus pools two-slot DMR channels and moves control among them rather than dedicating one.</figcaption>
</figure>

## Overview

Following Capacity Plus means tracking the rotating control as it hops, then decoding the
granted [timeslot](/reference/tdma/) — GopherTrunk's DMR support is vendor-aware of these
Motorola grant formats.
