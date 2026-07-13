---
slug: radio-id
title: Radio ID
entry_type: term
category: trunked-radio
description: A radio ID (unit ID) is the unique identifier of an individual radio on a system, revealed in control-channel signalling and in-band signalling such as MDC1200.
keywords: radio ID, unit ID, ANI, subscriber ID, RID, source ID, MDC1200, WACN, source address
aka: [radio ID, unit ID]
autolink: true
infobox:
  - { label: Type, value: Subscriber identifier }
  - { label: Seen in, value: Control-channel data, MDC1200 }
  - { label: Identifies, value: An individual radio/unit }
see_also: [talkgroup, affiliation, control-channel, mdc1200, private-call, registration, wacn, system-id]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

A **radio ID** (unit ID) is the unique identifier of an individual radio on a system,
distinct from the [talkgroup](/reference/talkgroup/) it is using.[^wiki] It appears in
[control-channel](/reference/control-channel/) signalling and in analog in-band schemes
like [MDC1200](/reference/mdc1200/). Where a talkgroup answers *which group is talking*,
the radio ID answers *which specific unit* — a particular officer's portable, a fire
engine, a dispatch console.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A transmission burst tagged with a unique radio identifier and its talkgroup." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="40" width="340" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="60" text-anchor="middle" font-size="10" fill="currentColor">voice transmission</text>
  <text x="120" y="30" font-size="9" fill="currentColor" text-anchor="middle">radio ID 4567</text>
  <text x="330" y="30" font-size="9" fill="currentColor" text-anchor="middle">TG 101</text>
  <g stroke="currentColor" stroke-width="1"><line x1="120" y1="33" x2="120" y2="40"/><line x1="330" y1="33" x2="330" y2="40"/></g>
  <text x="230" y="92" text-anchor="middle" font-size="9" fill="currentColor">every transmission identifies the individual radio</text>
</svg>
<figcaption>A radio ID uniquely identifies the transmitting unit, so you see who is talking, not just which group.</figcaption>
</figure>

## How it works

Each transmission and [affiliation](/reference/affiliation/) carries the source radio ID,
so a monitor can see *which unit* is talking, not just which group. On a
[group call](/reference/group-call/) the voice channel's embedded signalling includes the
source ID alongside the destination talkgroup; on a [private call](/reference/private-call/)
both the source and destination radio IDs appear. Registration and
[affiliation](/reference/affiliation/) messages on the control channel expose radio IDs
even when no call is in progress.

Radio IDs are numeric and scoped to the system. On P25 a Source Unit ID is a 24-bit value
that is unique within the network defined by the [WACN](/reference/wacn/) and
[system ID](/reference/system-id/); DMR uses its own 24-bit radio IDs. Fleets often assign
ID ranges by department, so the numeric ID alone can hint at a unit's role even without an
alias table.

## Variants

- **Trunked source ID** — carried in P25/DMR control and voice signalling for every keyup.
- **Analog ANI** — [MDC1200](/reference/mdc1200/) and similar in-band schemes send a
  radio's ID as a short data burst at the start (and optionally end) of an analog FM
  transmission.
- **Special / system IDs** — some values are reserved for consoles, the system itself, or
  all-call addresses rather than a physical subscriber.

## In practice

Radio IDs let a listener follow a specific unit rather than a whole talkgroup, and they are
central to fleet mapping: correlating IDs with talkgroups and timing reveals who works with
whom. Hobbyist databases publish alias tables mapping numeric IDs to unit names for known
systems, though many agencies rotate or withhold them.

## Relevance to SDR

GopherTrunk's radio-ID view merges live radio IDs with any alias catalogue, letting you
track individual units across a system. Because the source ID is present in both control
signalling and the voice channel's embedded data, GopherTrunk can tag each logged call with
the transmitting unit, not merely the group — the basis for per-unit history and filtering.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on subscriber/unit identifiers in trunking signalling.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 source unit IDs scoped by WACN and system ID.
