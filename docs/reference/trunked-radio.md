---
slug: trunked-radio
title: Trunked radio
entry_type: term
category: trunked-radio
description: Trunked radio is a system that shares a small pool of frequencies among many user groups by assigning a channel to each call on demand, coordinated by a control channel.
keywords: trunked radio, trunking, control channel, talkgroup, channel pool, public safety
aka: [trunked radio, trunking]
autolink: true
infobox:
  - { label: Type, value: Radio-system architecture }
  - { label: Coordinated by, value: Control channel }
  - { label: User identity, value: Talkgroup }
  - { label: Examples, value: P25, DMR Tier III, TETRA }
see_also: [conventional-radio, control-channel, voice-channel, talkgroup, channel-grant, fdma, tdma]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

**Trunked radio** is a system architecture in which many user groups share a small pool
of frequencies, with a computer assigning a free channel to each call for its duration
and reclaiming it afterward. A [control channel](/reference/control-channel/)
coordinates the whole system.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A control channel at the top issuing an assignment, and a pool of voice channels below with one assigned to a talkgroup." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="20" width="400" height="34" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="35" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">control channel (data)</text>
  <text x="230" y="48" text-anchor="middle" font-size="9" fill="currentColor">"TG 101 → channel 3"</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="73" y="120">voice 1</text>
    <rect x="135" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="178" y="120">voice 2</text>
    <rect x="240" y="100" width="86" height="32" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="283" y="116">voice 3</text><text x="283" y="128" font-size="8">TG 101</text>
    <rect x="345" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="388" y="120">voice 4</text>
  </g>
  <line x1="230" y1="54" x2="283" y2="98" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#trar)"/>
  <text x="230" y="155" text-anchor="middle" font-size="9" fill="currentColor">a free channel is assigned per call, then released</text>
  <defs><marker id="trar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A trunked system shares a pool of channels; the control channel assigns a free one to each call.</figcaption>
</figure>

## How it works

When a user keys up, their radio requests a call on the control channel, which issues a
[channel grant](/reference/channel-grant/) pointing the [talkgroup](/reference/talkgroup/)
to a free [voice channel](/reference/voice-channel/). Because real traffic is bursty, a
few channels can serve many groups.

## Relevance to SDR

To monitor a trunked system you decode the control channel first, then follow grants —
exactly what GopherTrunk does for [P25](/reference/project-25/),
[DMR Tier III](/reference/dmr-tier-3/), and others.
