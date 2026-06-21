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
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

A **voice channel** (or traffic channel) is a frequency that a
[trunked radio](/reference/trunked-radio/) system **temporarily assigns** to carry one
call.[^wiki] When the call ends, the channel returns to the pool for reuse.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A pool of voice channels over time, each lighting up briefly for a call then going idle." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor"><text x="20" y="35">ch 1</text><text x="20" y="65">ch 2</text><text x="20" y="95">ch 3</text></g>
  <g stroke="currentColor" stroke-opacity="0.3"><line x1="55" y1="30" x2="440" y2="30"/><line x1="55" y1="60" x2="440" y2="60"/><line x1="55" y1="90" x2="440" y2="90"/></g>
  <g fill="currentColor" fill-opacity="0.3"><rect x="70" y="22" width="80" height="16"/><rect x="250" y="22" width="60" height="16"/><rect x="120" y="52" width="100" height="16"/><rect x="180" y="82" width="70" height="16"/><rect x="330" y="82" width="90" height="16"/></g>
  <text x="240" y="120" text-anchor="middle" font-size="9" fill="currentColor">time → (each call borrows a channel, then frees it)</text>
</svg>
<figcaption>Voice channels are assigned for the duration of a call, then returned to the shared pool.</figcaption>
</figure>

## How it works

The [control channel](/reference/control-channel/) issues a
[grant](/reference/channel-grant/) directing a [talkgroup](/reference/talkgroup/) to a
specific voice channel (and, on TDMA systems, a timeslot). The next call from that group
may land on a different voice channel entirely.

## Relevance to SDR

GopherTrunk tunes a receiver to the granted voice channel to capture the audio, then
returns to await the next assignment — following many calls from one capture.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on per-call voice (traffic) channel assignment.
