---
slug: trunking-site
title: Trunking site
entry_type: term
category: trunked-radio
description: "A trunking site is one repeater location in a trunked radio system: a tower whose transmitters broadcast a control channel plus a pool of voice channels."
keywords: trunking site, repeater site, RF site, tower, cell, control channel, voice channel, multisite, P25 site, DMR site
aka: [trunking site, RF site, repeater site]
autolink: true
infobox:
  - { label: Type, value: Physical repeater location }
  - { label: Broadcasts, value: One control channel + voice pool }
  - { label: Identified by, value: Site number (within an RFSS/system) }
see_also: [control-channel, multisite-trunking, simulcast, neighbor-site, rfss, voice-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

A **trunking site** is a single repeater location — a tower with its transmitters
and receivers — that radiates one [control channel](/reference/control-channel/)
and a pool of [voice channels](/reference/voice-channel/) covering the area around
it.[^wiki] It is the physical building block of a trunked system: the antennas, the
combiner, and the channel electronics that together define one coverage cell. A radio
in that cell registers with the site's control channel and receives all its
[channel grants](/reference/channel-grant/) from it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A single tower radiating one control channel and several voice channels over a coverage area, with radios registered to it." xmlns="http://www.w3.org/2000/svg">
  <line x1="70" y1="30" x2="70" y2="120" stroke="currentColor" stroke-width="2"/>
  <path d="M70 30 L52 120 M70 30 L88 120" stroke="currentColor" stroke-width="1.1" fill="none"/>
  <g stroke="currentColor" stroke-width="1" fill="none" stroke-opacity="0.6">
    <path d="M70 40 A70 70 0 0 1 70 40"/>
    <circle cx="70" cy="55" r="55" stroke-dasharray="4 4"/>
  </g>
  <text x="70" y="150" text-anchor="middle" font-size="9" fill="currentColor">Site 3</text>
  <g font-size="9" fill="currentColor">
    <rect x="200" y="26" width="230" height="22" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="315" y="41" text-anchor="middle">control channel (data, always on)</text>
    <rect x="200" y="58" width="72" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="236" y="72" text-anchor="middle" font-size="8">voice 1</text>
    <rect x="279" y="58" width="72" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="315" y="72" text-anchor="middle" font-size="8">voice 2</text>
    <rect x="358" y="58" width="72" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="394" y="72" text-anchor="middle" font-size="8">voice 3</text>
  </g>
  <text x="315" y="105" text-anchor="middle" font-size="9" fill="currentColor">one control channel + a voice pool = one site</text>
</svg>
<figcaption>A trunking site is one physical repeater cell: a single control channel plus the voice channels it hands out.</figcaption>
</figure>

## How it works

Each site runs one dedicated control channel that never carries voice — it streams
signalling continuously so idle radios can lock to it, affiliate, and wait. The rest
of the site's licensed frequencies form the voice pool. When a member keys up, the
site's controller picks a free voice channel, transmits a grant on the control
channel, and every affiliated radio in the cell retunes to hear the call. A site is
identified by a **site number** that is unique within its parent
[RF subsystem](/reference/rfss/); larger networks stitch many sites together (see
[multisite trunking](/reference/multisite-trunking/)).

A site can be **standalone** (its own controller makes all decisions locally) or
**networked** (a central controller coordinates it with peers so a talkgroup can be
active on several sites at once). Two nearby transmitters carrying the same audio in
lock-step is a special case called [simulcast](/reference/simulcast/), which improves
coverage but complicates reception.

## In practice

- A site advertises its **neighbours** on the control channel so radios know where to
  roam; a monitor reads the same list to map the system (see
  [neighbor site](/reference/neighbor-site/)).
- Site parameters — colour/network codes, site number, and the control-channel
  frequency — are broadcast periodically, letting a scanner identify exactly which
  cell it is hearing.
- Coverage is a trade-off: high towers reach far but overlap with neighbours, which is
  why boundary areas hand traffic between sites.

## Relevance to SDR

A software receiver sees a trunking site as one control-channel carrier surrounded by
a cluster of voice-channel frequencies. **GopherTrunk** locks a site's control
channel, decodes its identity and neighbour list, and tunes voice grants as they are
issued — so from GopherTrunk's point of view a site is the unit it tracks: one control
channel, one set of grants, one coverage cell. Systems such as
[P25](/reference/project-25/) and [DMR](/reference/dmr/) Tier III are built from these
sites, and a wide-area network is simply many of them linked together.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on repeater sites and trunked-system architecture.
