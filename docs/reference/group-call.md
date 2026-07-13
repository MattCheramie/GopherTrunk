---
slug: group-call
title: Group call
entry_type: term
category: trunked-radio
description: "A group call is a one-to-many voice transmission to a talkgroup on a trunked system; the control channel grants a voice channel that every affiliated member retunes to."
keywords: group call, group voice call, talkgroup call, one-to-many, PTT, channel grant, dispatch call, trunking voice, half-duplex
aka: [group call, group voice call]
autolink: true
infobox:
  - { label: Type, value: One-to-many voice call }
  - { label: Addressed to, value: A talkgroup }
  - { label: Set up by, value: A channel grant }
see_also: [talkgroup, channel-grant, private-call, control-channel, voice-channel, radio-id]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

A **group call** is a one-to-many voice transmission addressed to a
[talkgroup](/reference/talkgroup/): when a user presses to talk, every affiliated member
of that group hears it, no matter which physical frequency the system assigns.[^wiki] It
is the everyday mode of trunked land-mobile radio — a dispatcher calling a fleet, a
supervisor addressing a shift — and it is set up by a
[channel grant](/reference/channel-grant/) on the
[control channel](/reference/control-channel/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="One radio keys up to a talkgroup; the control channel grants a voice channel and multiple listening radios retune to hear the same call." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="60" width="80" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/>
  <text x="60" y="80" text-anchor="middle" font-size="8.5" fill="currentColor">talker</text>
  <text x="60" y="90" text-anchor="middle" font-size="7.5" fill="currentColor">radio 4567</text>
  <line x1="102" y1="77" x2="160" y2="77" stroke="currentColor" stroke-width="1.1" marker-end="url(#gcar)"/>
  <rect x="162" y="52" width="120" height="50" rx="6" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
  <text x="222" y="72" text-anchor="middle" font-size="8.5" fill="currentColor">grant: TG 101</text>
  <text x="222" y="86" text-anchor="middle" font-size="8.5" fill="currentColor">→ voice ch 3</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <line x1="284" y1="60" x2="340" y2="34" stroke="currentColor" stroke-width="1" marker-end="url(#gcar)"/>
    <line x1="284" y1="77" x2="340" y2="77" stroke="currentColor" stroke-width="1" marker-end="url(#gcar)"/>
    <line x1="284" y1="94" x2="340" y2="120" stroke="currentColor" stroke-width="1" marker-end="url(#gcar)"/>
    <rect x="342" y="22" width="96" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="38">listener A</text>
    <rect x="342" y="65" width="96" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="81">listener B</text>
    <rect x="342" y="108" width="96" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="124">listener C</text>
  </g>
  <defs><marker id="gcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A group call: one talker keys up, the control channel grants a voice channel, and every affiliated member of the talkgroup retunes to listen.</figcaption>
</figure>

## How it works

A radio requests a group call by sending its talkgroup and its own
[radio ID](/reference/radio-id/) to the control channel. The system finds a free
[voice channel](/reference/voice-channel/), issues a grant naming the talkgroup and that
channel, and every radio affiliated to the talkgroup retunes to it — including radios on
other [sites](/reference/trunking-site/) where members are present. The call is
half-duplex and push-to-talk: only one person transmits at a time, and when the talker
releases, the channel is freed. Successive transmissions in the same conversation may
land on *different* voice channels, because the system re-grants each time.

A group call contrasts with a [private call](/reference/private-call/), which is a
one-to-one transmission addressed to a single radio ID rather than a talkgroup.
Emergency and priority calls are group calls flagged for preferential handling, so the
system grants them a channel ahead of routine traffic.

## Relevance to SDR

Group calls are the bulk of what a trunking monitor hears, and following them is the
whole point of decoding the control channel. **GopherTrunk** reads each grant, tunes a
receiver to the assigned voice channel, and presents the call tagged with its talkgroup
and the transmitting radio ID — then follows the conversation as the system re-grants it
across the channel pool. GopherTrunk decodes clear and de-scramblable group-call audio;
it cannot recover calls protected by keyed encryption, though it still logs the call
metadata (talkgroup, radio ID, channel) even when the voice is encrypted.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on group (one-to-many) calls in trunked systems.
