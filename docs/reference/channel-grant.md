---
slug: channel-grant
title: Channel grant
entry_type: term
category: trunked-radio
description: A channel grant is the control-channel message that assigns a talkgroup's call to a specific voice channel (and timeslot), telling radios and monitors where to tune.
keywords: channel grant, grant, control channel message, voice channel assignment, trunking, group grant, unit grant, update
aka: [channel grant]
autolink: true
infobox:
  - { label: Type, value: Control-channel message }
  - { label: Contains, value: Talkgroup, voice channel, (slot) }
  - { label: Triggers, value: Radios retune to the call }
see_also: [control-channel, voice-channel, talkgroup, trunked-radio, tsbk, csbk, group-call, private-call, late-entry]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

A **channel grant** is the [control-channel](/reference/control-channel/) message that
assigns a [talkgroup](/reference/talkgroup/)'s call to a specific
[voice channel](/reference/voice-channel/) (and timeslot on
[TDMA](/reference/tdma/) systems).[^wiki] It is the pivotal event in trunking: the instant
a grant is broadcast, a call is starting and the message says exactly where.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The control channel issuing a grant message that causes affiliated radios to retune to a voice channel." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="40" width="120" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="90" y="61" text-anchor="middle" font-size="9" fill="currentColor">control channel</text>
  <line x1="152" y1="57" x2="290" y2="57" stroke="currentColor" stroke-width="1.1" marker-end="url(#cgar)"/><text x="221" y="50" text-anchor="middle" font-size="8.5" fill="currentColor">grant: TG 101 → ch 3</text>
  <rect x="300" y="40" width="130" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="365" y="56" text-anchor="middle" font-size="9" fill="currentColor">radios retune</text><text x="365" y="68" text-anchor="middle" font-size="8" fill="currentColor">to voice channel 3</text>
  <defs><marker id="cgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A channel grant is the control-channel message assigning a talkgroup to a specific voice channel.</figcaption>
</figure>

## How it works

When the controller accepts a call request, it packs a grant into a
[TSBK](/reference/tsbk/) (P25) or [CSBK](/reference/csbk/) (DMR) and broadcasts it on the
control channel. The grant carries the addressing needed to join the call: the target
identity (a [talkgroup](/reference/talkgroup/) for a [group call](/reference/group-call/),
or a source/destination pair for a [private call](/reference/private-call/)), the physical
voice channel — usually as a *channel number* the radio maps to a frequency through the
system's channel plan — and, on TDMA systems, the timeslot. Affiliated radios and monitors
read it and retune together.

Because a radio might miss the initial grant — it was out of range, or just powered on —
systems repeat a *group update* or *grant update* message while a call is in progress. This
periodic re-announcement of active calls is what lets a late arrival find and join a call
already underway, the mechanism behind [late entry](/reference/late-entry/).

## Variants

- **Group voice grant** — the common case, directing a talkgroup to a voice channel.
- **Unit-to-unit (private) grant** — assigns a channel for a one-to-one
  [private call](/reference/private-call/) between two radio IDs.
- **Grant / group update** — an in-progress re-announcement of active calls for late entry
  and for radios that missed the original.
- **Data / packet grant** — assigns a channel for a short data transfer rather than voice.

## In practice

Grants are the backbone of any trunk-following scanner. During a busy incident the control
channel may issue several grants per second across many voice channels, and a monitor's job
is to task receivers to the ones a listener cares about. If all voice channels are busy, a
call request is queued rather than granted, and the eventual grant may arrive seconds
later.

## Relevance to SDR

GopherTrunk reads grants in real time to task a receiver to the right channel and slot,
which is how it follows conversations as they scatter across the channel pool. It maps the
grant's channel number to an actual frequency via the decoded channel plan, then
down-converts and demodulates that voice channel — repeating the process for every grant
so a single wideband capture reconstructs the whole system's traffic.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on the grant message that assigns calls to voice channels.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 grant and update messages carried in TSBKs.
