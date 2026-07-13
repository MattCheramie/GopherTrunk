---
slug: control-channel
title: Control channel
entry_type: term
category: trunked-radio
description: The control channel is the data-only frequency of a trunked radio system that coordinates it, announcing channel grants, registrations, and system parameters.
keywords: control channel, trunking signalling, channel grant, affiliation, data channel, TSBK, CSBK, neighbor list
aka: [control channel]
autolink: true
infobox:
  - { label: Type, value: Trunking signalling channel }
  - { label: Carries, value: Data only (no voice) }
  - { label: Announces, value: Grants, affiliations, system info }
see_also: [trunked-radio, voice-channel, channel-grant, affiliation, talkgroup, tsbk, csbk, rest-channel, neighbor-site, busy-idle, late-entry]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

The **control channel** is the **data-only** frequency that coordinates a
[trunked radio](/reference/trunked-radio/) system.[^wiki] It carries a continuous stream of
signalling — never voice — managing registrations, call requests, and
[channel grants](/reference/channel-grant/). Because it announces where every call goes,
the control channel is the single most valuable thing to decode on a trunked system:
lock onto it and the entire system's activity becomes visible.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A steady always-on control channel stripe issuing messages that point radios to voice channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="400" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="48" text-anchor="middle" font-size="10" fill="currentColor">control channel — continuous data, never voice</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="100" y="86">grant</text><text x="100" y="98" font-size="7.5">→ ch 3</text>
    <text x="230" y="86">affiliation</text><text x="230" y="98" font-size="7.5">radio 4567</text>
    <text x="360" y="86">grant</text><text x="360" y="98" font-size="7.5">→ ch 7</text>
  </g>
  <g stroke="currentColor" stroke-width="1"><line x1="100" y1="58" x2="100" y2="74"/><line x1="230" y1="58" x2="230" y2="74"/><line x1="360" y1="58" x2="360" y2="74"/></g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">decode this first — it's the map to every call</text>
</svg>
<figcaption>The control channel carries the system's running commentary — affiliations, requests, and channel grants.</figcaption>
</figure>

## How it works

The control channel runs a fixed-rate downlink from the site controller, packed
back-to-back with short signalling messages — [TSBKs](/reference/tsbk/) on P25,
[CSBKs](/reference/csbk/) on DMR, and their equivalents elsewhere. When a radio requests a
call, the controller broadcasts a grant naming the [talkgroup](/reference/talkgroup/) and
the assigned [voice channel](/reference/voice-channel/); affiliated radios and monitors
retune to listen. Between grants, the channel also carries the system identity
([system ID](/reference/system-id/), [WACN](/reference/wacn/)), a
[neighbor-site](/reference/neighbor-site/) list so radios know where to
[roam](/reference/roaming/), and periodic status so idle radios stay locked.

On the reverse path, subscribers contend for a small [inbound signalling channel](/reference/control-channel/)
to send their requests, and the outbound stream can carry a [busy/idle](/reference/busy-idle/)
indication so radios know whether the reverse path is free. A radio arriving mid-call can
use the ongoing grant announcements to join a call already in progress — the mechanism
behind [late entry](/reference/late-entry/).

## Variants

- **Dedicated control channel** — one frequency is permanently the control channel (P25
  systems, DMR Tier III, TETRA). This is the easiest case to monitor.
- **Rotating / distributed control** — lighter DMR modes move the control function onto a
  [rest channel](/reference/rest-channel/) that hops around the pool as calls are assigned.
- **Composite control** — some systems interleave control signalling with voice on the
  same physical channel rather than dedicating a separate downlink.

## In practice

Finding the control channel is the first step in identifying an unknown system. Its
signature is a strong, continuous data carrier that never carries audio; scanners and
databases publish known control-channel frequencies for documented systems. On
[multisite](/reference/multisite-trunking/) networks each site has its own control
channel, and a monitor typically camps on the strongest one, using the broadcast neighbor
list to understand the wider topology.

## Relevance to SDR

Decoding the control channel is the key to monitoring a trunked system — it is the map
that tells GopherTrunk where every call goes, so it can task a receiver to each granted
voice channel in turn and follow many conversations from one capture. GopherTrunk parses
the grant, affiliation, and system-parameter messages of the control channels it supports
to drive its trunk-following engine and populate its activity and radio-ID views.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on the control channel and trunking signalling.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 control channel and its TSBK signalling structure.
