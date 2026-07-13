---
slug: neighbor-site
title: Neighbor site
entry_type: term
category: trunked-radio
description: A neighbor site is an adjacent site of a multi-site trunked network whose control-channel frequency and identity are broadcast so roaming radios know where to go next.
keywords: neighbor site, neighbour site, adjacent site, neighbor list, adjacent status broadcast, RFSS status, multi-site trunking, roaming, control channel, P25 adjacent site
aka: [neighbor site, neighbour site, adjacent site]
autolink: true
infobox:
  - { label: Type, value: Adjacent-site advertisement }
  - { label: Carried on, value: Control channel }
  - { label: Lists, value: Nearby sites and their control channels }
see_also: [multisite-trunking, roaming, control-channel, trunking-site, registration, talkgroup]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

A **neighbor site** is an adjacent site of a
[multi-site trunked network](/reference/multisite-trunking/), and the term also names the
**broadcast** — an adjacent-site status message — by which a site advertises those
neighbors on its [control channel](/reference/control-channel/).[^wiki] Each such message
names a nearby site and, crucially, its control-channel frequency, so a
[roaming](/reference/roaming/) radio knows exactly where to look when it needs to move to
a stronger site instead of blindly searching the band.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A site broadcasting a neighbor list naming three adjacent sites and their control-channel frequencies." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="50" width="110" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="75" y="66" text-anchor="middle" font-size="9" fill="currentColor">site 3</text><text x="75" y="78" text-anchor="middle" font-size="7.5" fill="currentColor">control channel</text>
  <line x1="132" y1="67" x2="176" y2="67" stroke="currentColor" stroke-width="1.1" marker-end="url(#nsar)"/><text x="154" y="60" text-anchor="middle" font-size="7" fill="currentColor">neighbors</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="184" y="34" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="49">site 2 · 851.0125</text>
    <rect x="184" y="60" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="75">site 4 · 852.2375</text>
    <rect x="184" y="86" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="244" y="101">site 7 · 853.7625</text>
  </g>
  <text x="380" y="71" text-anchor="middle" font-size="8" fill="currentColor" opacity="0.8">roaming map</text>
  <defs><marker id="nsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A site advertises its neighbors and their control-channel frequencies so roaming radios know where to go next.</figcaption>
</figure>

## How it works

A site periodically transmits adjacent-site status messages on its control channel. Each
message identifies a neighboring [trunking site](/reference/trunking-site/) — by site
number and system identity — and gives the parameters a radio needs to acquire it,
principally the neighbor's control-channel frequency, plus flags describing the
neighbor's status (whether it is in normal operation, whether it accepts the radio's
system, and so on). Taken together, these broadcasts form a small routing table of the
local network topology that every radio on the site receives for free.

A radio uses this table when [roaming](/reference/roaming/): rather than scanning the
whole band to find another site when its current one fades, it jumps directly to a
neighbor's advertised control-channel frequency and evaluates the signal there. The
neighbor list therefore turns site selection from a blind search into a directed one,
which is what makes hand-offs across a wide-area system fast enough to be seamless.

## In practice

Neighbor broadcasts are also how the extent and shape of a network becomes discoverable.
Because every site advertises its neighbors, walking the graph — site 3 names sites 2, 4
and 7; site 7 names 3, 8 and 9; and so on — reconstructs the whole multi-site system and
all of its control-channel frequencies. This is invaluable for anyone mapping a system,
and it is exactly the data a mobile radio consumes to stay connected while moving.

The frequency of these broadcasts is a design balance. Advertise neighbors too rarely and a
fast-moving unit can reach the edge of coverage before it has learned where to go next,
producing dropouts; advertise them too often and the airtime spent on housekeeping crowds
out grants and other useful signalling. Systems therefore interleave neighbor messages with
the rest of the control-channel traffic at a modest rate, enough that a radio parking on a
site accumulates the full neighbor picture within a short window. For a monitor this means a
site's neighbor list is not learned instantly on tuning; it fills in over the first several
seconds as the individual adjacent-site messages come around.

There is a limit to how much of the network any one site reveals. A neighbor broadcast
usually lists only the *immediately adjacent* sites, not the entire system, and a large
network is advertised in fragments — each site knows its own neighborhood. Reconstructing
the whole thing therefore means monitoring several sites and stitching their neighbor lists
together, following the edges of the graph outward. The lists can also change: a site
added, removed, or taken out of service updates what its neighbors advertise, so the
topology a monitor builds is a living map rather than a fixed one. Some systems further
qualify each neighbor entry with status flags — for example indicating that a neighbor is
in a degraded mode or does not currently accept roamers — which a radio weighs before
choosing to move there.

## Relevance to SDR

For a monitor, neighbor-site broadcasts are one of the most useful control-channel
messages, because a single site reveals the frequencies of the sites around it.
GopherTrunk parses adjacent-site status messages to learn a
[multi-site system's](/reference/multisite-trunking/) topology automatically: from one
tuned control channel it can enumerate the neighboring sites and their control-channel
frequencies, building a map without the user having to enter every site by hand. That map,
combined with [registration](/reference/registration/) events, is what lets a scanner
understand [roaming](/reference/roaming/) and decide which sites to watch for a given
[talkgroup](/reference/talkgroup/).

Real systems that carry neighbor/adjacent-site broadcasts include P25 (Adjacent Status
Broadcast / RFSS Status Broadcast), Motorola SmartZone, and DMR Tier III connected
systems. GopherTrunk reads these passively from the control channel — the neighbor list is
information the system publishes for its own radios, and a receiver simply overhears it.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on multi-site systems advertising adjacent sites for roaming.
