---
slug: busy-idle
title: Busy/idle status
entry_type: term
category: trunked-radio
description: Busy/idle status is the control-channel signalling that tells radios which trunked channels are free and which are in use, coordinating who may transmit.
keywords: busy idle, busy/idle, channel status, busy bit, idle bit, status word, control channel, trunking, inbound signalling, LTR status
aka: [busy/idle status, busy idle, channel status]
autolink: true
infobox:
  - { label: Type, value: Channel-status signalling }
  - { label: Carried on, value: Control channel }
  - { label: Purpose, value: Coordinate channel access }
see_also: [control-channel, channel-grant, voice-channel, late-entry, trunked-radio, ltr]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Logic_trunked_radio
---

**Busy/idle status** is the [control-channel](/reference/control-channel/) signalling that
continuously tells radios which channels of a trunked system are currently in use (busy)
and which are free (idle).[^wiki] It is the traffic-control layer of trunking: by
publishing the state of every channel, the system lets each radio know whether it may
request a channel and where the controller is likely to place its call.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A row of channel status indicators, some marked busy and some idle, published by the control channel so radios know which channels are free." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="46" width="110" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="75" y="65" text-anchor="middle" font-size="9" fill="currentColor">control channel</text>
  <line x1="132" y1="61" x2="168" y2="61" stroke="currentColor" stroke-width="1.1" marker-end="url(#biar)"/>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="176" y="44" width="42" height="34" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/><text x="197" y="58">ch1</text><text x="197" y="70">BUSY</text>
    <rect x="224" y="44" width="42" height="34" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="245" y="58">ch2</text><text x="245" y="70">idle</text>
    <rect x="272" y="44" width="42" height="34" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/><text x="293" y="58">ch3</text><text x="293" y="70">BUSY</text>
    <rect x="320" y="44" width="42" height="34" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="341" y="58">ch4</text><text x="341" y="70">idle</text>
    <rect x="368" y="44" width="42" height="34" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="389" y="58">ch5</text><text x="389" y="70">idle</text>
  </g>
  <defs><marker id="biar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The control channel publishes each channel's busy or idle state so radios know what is free.</figcaption>
</figure>

## How it works

The exact mechanism depends on the system's architecture:

- On systems with a **dedicated control channel** (P25, Motorola Type II, DMR Tier III),
  the controller broadcasts the state of the channel pool as part of its normal
  signalling. Each [channel grant](/reference/channel-grant/) marks a channel busy; a
  channel-release or the simple absence of further grants returns it to idle. Radios read
  this to know a call is underway and, together with grant updates, to support
  [late entry](/reference/late-entry/).
- On **sub-audible / distributed** systems such as [LTR](/reference/ltr/), there is no
  separate control channel: a low-speed data word rides beneath the voice on every
  repeater, and its "busy" indication tells radios on other channels that a given logical
  channel is in use. A radio inspects these status words to pick a free channel.
- On **inbound** signalling, a busy/idle bit on the outbound control channel also tells
  radios whether the *inbound* (radio-to-system) request slot is currently clear, so two
  radios do not collide when requesting service — an ALOHA-style access control.

In all cases the essential content is the same: a compact, constantly refreshed statement
of which channels are occupied and which are available.

## In practice

Busy/idle signalling is what makes trunking's efficiency possible, and its behaviour under
load is worth understanding. When every channel is busy, new requests are *queued* rather
than dropped, and the busy indication is what tells waiting radios there is nothing free
yet; a radio holds its request until the status shows a channel has gone idle or the
controller grants it one. On the inbound path the same idea prevents collisions: because
many radios share one request slot, a busy indication on that slot tells a radio to defer
and back off (an ALOHA-style random-access scheme), so two units keying up at once do not
step on each other indefinitely.

The distributed [LTR](/reference/ltr/) case is instructive because it shows busy/idle
signalling stripped to its essentials. LTR has no dedicated control channel at all; each
repeater continuously sends a sub-audible data word that includes a busy/idle bit and the
identity of the "home" repeater for the current traffic. A radio wanting to transmit reads
these words across the repeaters, finds one advertising idle, and uses it. The entire
trunking logic rides in that low-speed status stream — proof that busy/idle status, not a
fancy control channel, is the irreducible core of channel coordination.

## Relevance to SDR

Busy/idle information is operationally useful to a monitor for two reasons. First, it is
how a scanner knows *without decoding audio* that a [voice channel](/reference/voice-channel/)
is active, so it can schedule its receivers and follow calls efficiently — the busy state
is effectively a to-do list of channels worth listening to. Second, on distributed systems
like LTR, the sub-audible status word is the *only* signalling available, so parsing it is
mandatory just to know where the traffic is.

GopherTrunk reads channel-status signalling from the control channel (or the sub-audible
data on LTR-style systems) to maintain a live model of the channel pool — which channels
are busy, by which [talkgroup](/reference/talkgroup/), and which are free. This model
drives how it tasks a limited number of tuners across a system that may have far more
channels than the receiver can watch at once. As with all trunking control data, this is
read passively; GopherTrunk observes the busy/idle picture, it does not participate in
channel arbitration.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on channel-status signalling and access control in trunked systems.
