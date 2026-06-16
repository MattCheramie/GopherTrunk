---
slug: voice-channel
title: Voice channel
entry_type: term
category: trunked-radio
description: A voice channel (traffic channel) is a frequency temporarily assigned by a trunked system to carry an individual call, released back to the pool when the call ends.
keywords: voice channel, traffic channel, channel grant, trunking, call audio
aka: [voice channel, traffic channel]
autolink: true
infobox:
  - { label: Type, value: Assigned call-carrying channel }
  - { label: Assigned by, value: Control channel (grant) }
  - { label: Lifetime, value: Duration of one call }
see_also: [control-channel, trunked-radio, channel-grant, talkgroup]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

A **voice channel** (or traffic channel) is a frequency that a
[trunked radio](/reference/trunked-radio/) system **temporarily assigns** to carry one
call. When the call ends, the channel returns to the pool for reuse.

## How it works

The [control channel](/reference/control-channel/) issues a
[grant](/reference/channel-grant/) directing a [talkgroup](/reference/talkgroup/) to a
specific voice channel (and, on TDMA systems, a timeslot). The next call from that group
may land on a different voice channel entirely.

## Relevance to SDR

GopherTrunk tunes a receiver to the granted voice channel to capture the audio, then
returns to await the next assignment — following many calls from one capture.
