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
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

The **control channel** is the **data-only** frequency that coordinates a
[trunked radio](/reference/trunked-radio/) system. It carries a continuous stream of
signalling — never voice — managing registrations, call requests, and
[channel grants](/reference/channel-grant/).

## How it works

When a call starts, the control channel broadcasts a grant naming the
[talkgroup](/reference/talkgroup/) and the assigned [voice channel](/reference/voice-channel/);
affiliated radios retune to listen. It also conveys the system identity and parameters.

## Relevance to SDR

Decoding the control channel is the key to monitoring a trunked system — it is the map
that tells GopherTrunk where every call goes, so it can follow them all.
