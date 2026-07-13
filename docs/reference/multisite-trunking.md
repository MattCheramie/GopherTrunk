---
slug: multisite-trunking
title: Multisite trunking
entry_type: term
category: trunked-radio
description: "Multisite trunking links many repeater sites into one wide-area system, so a talkgroup's call is repeated on every site where its members are affiliated."
keywords: multisite trunking, wide-area trunking, networked trunking, simulcast, roaming, affiliation, P25 WACN, DMR Tier III, connected sites
aka: [multisite trunking, wide-area trunking, networked trunking]
autolink: true
infobox:
  - { label: Type, value: Wide-area system topology }
  - { label: Built from, value: Many linked trunking sites }
  - { label: Enables, value: Roaming + system-wide calls }
see_also: [trunking-site, roaming, neighbor-site, simulcast, control-channel, wacn]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

**Multisite trunking** is the topology in which many
[trunking sites](/reference/trunking-site/) are linked into a single wide-area system
so that users stay on one logical network as they move across a region.[^wiki] Each
site still runs its own [control channel](/reference/control-channel/) and voice pool,
but a central network controller ties them together: when a call is placed, the system
repeats it on every site where members of that talkgroup are currently affiliated, so
a conversation can span an entire county or state.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Three towers connected to a central network controller; a call on one talkgroup is repeated on the sites where its members are registered." xmlns="http://www.w3.org/2000/svg">
  <rect x="185" y="76" width="90" height="30" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="95" text-anchor="middle" font-size="9" fill="currentColor">network controller</text>
  <g stroke="currentColor" stroke-width="2">
    <line x1="60" y1="30" x2="60" y2="80"/><line x1="230" y1="150" x2="230" y2="150"/><line x1="400" y1="30" x2="400" y2="80"/><line x1="230" y1="140" x2="230" y2="140"/>
    <line x1="60" y1="140" x2="60" y2="140"/>
  </g>
  <g stroke="currentColor" stroke-width="2">
    <line x1="60" y1="120" x2="60" y2="160"/><line x1="400" y1="120" x2="400" y2="160"/>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-dasharray="4 3">
    <line x1="90" y1="150" x2="182" y2="95"/><line x1="278" y1="95" x2="372" y2="150"/>
  </g>
  <text x="60" y="178" text-anchor="middle" font-size="9" fill="currentColor">Site 1</text>
  <text x="400" y="178" text-anchor="middle" font-size="9" fill="currentColor">Site 2</text>
  <text x="230" y="35" text-anchor="middle" font-size="9" fill="currentColor">one call → repeated on every site with affiliated members</text>
</svg>
<figcaption>A wide-area system links sites through a controller; a talkgroup's call is keyed up on each site where its members are registered.</figcaption>
</figure>

## How it works

Radios [affiliate](/reference/affiliation/) with the strongest site's control channel
and register their [talkgroup](/reference/talkgroup/) membership. The network tracks,
per talkgroup, which sites have listeners. On a call the controller grants a voice
channel on each of those sites and links their audio, while quiet sites are spared —
saving spectrum. As a user drives out of one cell and into another, their radio reads
the current site's advertised [neighbour](/reference/neighbor-site/) list and hands off
automatically, a process called [roaming](/reference/roaming/).

There are two ways sites can overlap. In **multisite** proper, adjacent sites use
*different* frequencies and radios roam between them. In
[simulcast](/reference/simulcast/), a group of transmitters share *one* frequency in
tight sync to act as a single large cell. Large networks mix both: several simulcast
cells, each treated as one site, wired into a multisite whole.

## In practice

- Each site is numbered within its subsystem; P25 groups sites into an
  [RF subsystem (RFSS)](/reference/rfss/), and RFSSs into a network identified by a
  [WACN](/reference/wacn/).
- Only sites with affiliated members carry a given call, so what you can hear depends
  on where the talkgroup's users physically are.
- A monitor near a system edge often receives several sites at once and must choose
  which control channel to follow.

## Relevance to SDR

A single [software-defined radio](/reference/software-defined-radio/) tuned to one
site hears only the calls that site is carrying. To follow a whole wide-area system you
either move between control channels or run several receivers. **GopherTrunk** decodes
one site's control channel at a time — reading its identity, neighbour list, and grants
— which is exactly the slice of a multisite network a scanner can observe from a fixed
location. Modern [P25](/reference/project-25/) and [DMR](/reference/dmr/) Tier III
public-safety networks are almost always multisite.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on wide-area, networked multi-site trunking.
