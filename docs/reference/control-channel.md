---
slug: control-channel
title: Control channel
entry_type: term
category: trunked-radio
description: The control channel is the data-only frequency of a trunked radio system that coordinates it, announcing channel grants, registrations, and system parameters.
keywords: control channel, trunking signalling, channel grant, affiliation, data channel
aka: [control channel]
autolink: true
infobox:
  - { label: Type, value: Trunking signalling channel }
  - { label: Carries, value: Data only (no voice) }
  - { label: Announces, value: Grants, affiliations, system info }
see_also: [trunked-radio, voice-channel, channel-grant, affiliation, talkgroup]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

The **control channel** is the **data-only** frequency that coordinates a
[trunked radio](/reference/trunked-radio/) system.[^wiki] It carries a continuous stream of
signalling — never voice — managing registrations, call requests, and
[channel grants](/reference/channel-grant/).

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

When a call starts, the control channel broadcasts a grant naming the
[talkgroup](/reference/talkgroup/) and the assigned [voice channel](/reference/voice-channel/);
affiliated radios retune to listen. It also conveys the system identity and parameters.

## Relevance to SDR

Decoding the control channel is the key to monitoring a trunked system — it is the map
that tells GopherTrunk where every call goes, so it can follow them all.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on the control channel and trunking signalling.
