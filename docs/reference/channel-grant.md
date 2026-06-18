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
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

A **channel grant** is the [control-channel](/reference/control-channel/) message that
assigns a [talkgroup](/reference/talkgroup/)'s call to a specific
[voice channel](/reference/voice-channel/) (and timeslot on
[TDMA](/reference/tdma/) systems).

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

The grant names the talkgroup and the channel; affiliated radios and monitors retune to
follow the call. It is the moment a monitor learns that a call is starting and exactly
where.

## Relevance to SDR

GopherTrunk reads grants in real time to task a receiver to the right channel/slot,
which is how it follows conversations as they scatter across the channel pool.
