---
slug: rest-channel
title: Rest channel
entry_type: term
category: trunked-radio
description: In trunked systems with a rotating control channel, the rest channel is the one currently carrying control signalling — it moves as calls are assigned.
keywords: rest channel, distributed control channel, DMR Capacity Plus, rotating control channel, trunking, Connect Plus, LCN
aka: ["rest channel"]
autolink: true
see_also: [control-channel, channel-grant, trunked-radio, capacity-plus, dmr, csbk, connect-plus, voice-channel]
related_lessons:
  - { title: "Finding & identifying systems", url: /learn/rf-sdr/finding-systems/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

A **rest channel** is the channel currently carrying control signalling in trunked
systems that **rotate the control function around the pool** rather than dedicate one
frequency to it.[^wiki] When a call is assigned to the current rest channel, control moves
to another idle channel — the new rest channel — so a radio that wants to place a call
must first find where control now lives.

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

## How it works

Instead of burning one frequency permanently as a [control channel](/reference/control-channel/),
a rest-channel system lets the idle channel that is *not* carrying a call act as control.
It sends the [CSBKs](/reference/csbk/) — grants, requests, status — until a call needs a
channel. If the controller assigns the current rest channel to that call, it first tells
listening radios which idle channel becomes the new rest channel, then hands the old one
over to voice. Radios (and monitors) follow the pointer to the new rest channel and keep
listening. The scheme trades a dedicated control frequency for one more usable voice
channel — attractive on small systems with only a handful of frequencies.

The cost is monitoring complexity. There is no fixed frequency to camp on: a scanner must
track the rest channel as it hops around the pool, using the "next rest channel" pointers
embedded in the signalling, or it loses the thread of the system entirely.

## Variants

- **Motorola Capacity Plus** — [Capacity Plus](/reference/capacity-plus/) single-site uses
  a rotating rest channel across its channel pool.
- **Motorola Connect Plus** — [Connect Plus](/reference/connect-plus/) is multisite and
  uses a dedicated control channel per site instead, a useful contrast to the rest-channel
  model.
- **Other light-trunking modes** — several small DMR and LTR-style systems rotate control
  similarly to conserve channels.

## In practice

Rest-channel systems are common in commercial and business DMR where spectrum is scarce and
every channel counts. Identifying one starts with recognising that the "control" carrier
moves: a searcher sees signalling jump between frequencies as calls come and go. Correctly
following the rest channel is essential, because a missed hop means missing every
subsequent grant.

## Relevance to SDR

Rotating control complicates monitoring: GopherTrunk must follow the rest channel as it
hops rather than camping on one fixed frequency. It does this by reading the next-rest
pointers in the [CSBK](/reference/csbk/) stream and retasking its control decoder to the
new frequency, keeping continuity of the system's activity as control migrates around the
pool.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR trunking modes that rotate the control (rest) channel.
[^trs]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on dedicated vs. distributed control-channel trunking.
