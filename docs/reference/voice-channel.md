---
slug: voice-channel
title: Voice channel
entry_type: term
category: trunked-radio
description: A voice channel (traffic channel) is a frequency temporarily assigned by a trunked system to carry an individual call, released back to the pool when the call ends.
keywords: voice channel, traffic channel, channel grant, trunking, call audio, timeslot, hang time
aka: [voice channel, traffic channel]
autolink: true
infobox:
  - { label: Type, value: Assigned call-carrying channel }
  - { label: Assigned by, value: Control channel (grant) }
  - { label: Lifetime, value: Duration of one call }
see_also: [control-channel, trunked-radio, channel-grant, talkgroup, tdma, fdma, group-call, late-entry]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

A **voice channel** (or traffic channel) is a frequency that a
[trunked radio](/reference/trunked-radio/) system **temporarily assigns** to carry one
call.[^wiki] When the call ends, the channel returns to the pool for reuse. It is the
counterpart of the [control channel](/reference/control-channel/): control carries data
and coordinates, while a voice channel carries the actual conversation for the brief life
of a single transmission or call.

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

The control channel issues a [grant](/reference/channel-grant/) directing a
[talkgroup](/reference/talkgroup/) to a specific voice channel. On a
[TDMA](/reference/tdma/) system the grant also names a timeslot, because two calls share
one RF channel; on an [FDMA](/reference/fdma/) system the channel is the whole frequency.
Radios retune, the talker's audio flows for the duration of the transmission, and a short
*hang time* keeps the channel reserved so replies in the same conversation stay on it
rather than being reassigned. When the hang timer expires with no more traffic, the
controller marks the channel idle and it rejoins the pool. The next call from the same
group may land on a completely different voice channel.

Beyond audio, a voice channel carries embedded low-speed signalling — the talkgroup and
source [radio ID](/reference/radio-id/), encryption status, and periodic frame
synchronisation — so a receiver that tunes in mid-call can identify the participants and
achieve [late entry](/reference/late-entry/) without having heard the original grant.

## Variants

- **Analog voice channel** — FM audio (older systems, or analog talkgroups on hybrid
  systems), with unit IDs carried by in-band signalling.
- **Digital voice channel** — a [vocoder](/reference/vocoder/) bitstream (AMBE+2 on P25/DMR,
  ACELP on TETRA) wrapped in error-correction and framing.
- **Data / packet channel** — some systems grant channels for short data bursts rather
  than voice, using the same assignment machinery.

## In practice

Voice channels are only busy in short bursts, which is exactly why trunking works: a pool
far smaller than the number of talkgroups suffices. During a busy incident, though, all
voice channels can be in use at once, at which point new call requests are queued on the
control channel until one frees up. Encrypted calls still occupy a granted voice channel;
a monitor sees the grant and the encryption flag but cannot recover the audio.

## Relevance to SDR

GopherTrunk tunes a receiver to the granted voice channel to capture the audio, then
returns to await the next assignment — following many calls from one wideband capture.
On TDMA systems it decodes the specific granted slot of the assigned channel. Because the
voice channel carries the source ID inline, each captured call is tagged with who was
talking, not just which [group](/reference/group-call/) — the basis of GopherTrunk's
per-call logging.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on per-call voice (traffic) channel assignment.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on how one traffic frequency carries multiple calls via timeslots.
