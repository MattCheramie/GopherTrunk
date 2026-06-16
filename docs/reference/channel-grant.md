---
slug: channel-grant
title: Channel grant
entry_type: term
category: trunked-radio
description: A channel grant is the control-channel message that assigns a talkgroup's call to a specific voice channel (and timeslot), telling radios and monitors where to tune.
keywords: channel grant, grant, control channel message, voice channel assignment, trunking
aka: [channel grant]
autolink: true
infobox:
  - { label: Type, value: Control-channel message }
  - { label: Contains, value: Talkgroup, voice channel, (slot) }
  - { label: Triggers, value: Radios retune to the call }
see_also: [control-channel, voice-channel, talkgroup, trunked-radio]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

A **channel grant** is the [control-channel](/reference/control-channel/) message that
assigns a [talkgroup](/reference/talkgroup/)'s call to a specific
[voice channel](/reference/voice-channel/) (and timeslot on
[TDMA](/reference/tdma/) systems).

## How it works

The grant names the talkgroup and the channel; affiliated radios and monitors retune to
follow the call. It is the moment a monitor learns that a call is starting and exactly
where.

## Relevance to SDR

GopherTrunk reads grants in real time to task a receiver to the right channel/slot,
which is how it follows conversations as they scatter across the channel pool.
