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

## How it works

When a user keys up, their radio requests a call on the control channel, which issues a
[channel grant](/reference/channel-grant/) pointing the [talkgroup](/reference/talkgroup/)
to a free [voice channel](/reference/voice-channel/). Because real traffic is bursty, a
few channels can serve many groups.

## Relevance to SDR

To monitor a trunked system you decode the control channel first, then follow grants —
exactly what GopherTrunk does for [P25](/reference/project-25/),
[DMR Tier III](/reference/dmr-tier-3/), and others.
