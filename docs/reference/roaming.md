---
slug: roaming
title: Roaming
entry_type: term
category: trunked-radio
description: Roaming is a radio moving between sites of a multi-site trunked network, re-registering at each new site so the system can still route its calls.
keywords: roaming, roam, site roaming, site selection, multi-site trunking, re-registration, neighbor site, control channel, wide-area trunking, P25 roaming
aka: [roaming, site roaming]
autolink: true
infobox:
  - { label: Type, value: Mobility behaviour }
  - { label: Requires, value: Re-registration at each site }
  - { label: Guided by, value: Neighbor-site broadcasts }
see_also: [multisite-trunking, neighbor-site, registration, trunking-site, control-channel, affiliation]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

**Roaming** is a radio moving between the sites of a
[multi-site trunked network](/reference/multisite-trunking/) and re-attaching to whichever
site gives it the best signal, so wide-area coverage is seamless.[^wiki] As it moves, the
radio drops the old site's [control channel](/reference/control-channel/), acquires a new
one, and performs a fresh [registration](/reference/registration/); the network updates
where the unit is so it can still deliver the unit's calls.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A radio moving out of site A's coverage into site B, dropping site A's control channel and re-registering on site B." xmlns="http://www.w3.org/2000/svg">
  <circle cx="120" cy="70" r="52" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/><text x="120" y="40" text-anchor="middle" font-size="8.5" fill="currentColor">site A</text>
  <circle cx="330" cy="70" r="52" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/><text x="330" y="40" text-anchor="middle" font-size="8.5" fill="currentColor">site B</text>
  <rect x="95" y="82" width="48" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="119" y="96" text-anchor="middle" font-size="7.5" fill="currentColor">radio</text>
  <line x1="146" y1="92" x2="303" y2="92" stroke="currentColor" stroke-width="1.3" marker-end="url(#roar)"/><text x="225" y="86" text-anchor="middle" font-size="8" fill="currentColor">moves &amp; re-registers</text>
  <rect x="306" y="82" width="48" height="20" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.1"/><text x="330" y="96" text-anchor="middle" font-size="7.5" fill="currentColor">radio</text>
  <defs><marker id="roar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Roaming: as a unit leaves one site's coverage it acquires the next site's control channel and re-registers.</figcaption>
</figure>

## How it works

Each site of a wide-area system runs its own control channel. A radio continuously
monitors the signal quality of its current site, and each site broadcasts a list of its
[neighbor sites](/reference/neighbor-site/) — their control-channel frequencies and
identities. When the current site's signal degrades, the radio consults that neighbor
list, evaluates the candidates, and switches to a stronger site. It then performs a
[registration](/reference/registration/) on the new site, which propagates through the
network so the system knows the unit's new location. A call for the unit or its
[talkgroup](/reference/talkgroup/) is then delivered to the site the unit is registered
on.

Roaming is what makes a multi-site system feel like one big system to the user: a mobile
unit driving across a state can move through many sites and stay in contact, with the
hand-offs happening automatically and, ideally, imperceptibly. The neighbor-site
broadcasts are the map the radio uses to roam efficiently, sparing it from blindly
searching for a new control channel each time.

## In practice

The roaming decision is a trade-off between clinging to the current site (avoiding
needless switching) and moving early enough to prevent dropouts. Systems tune parameters
such as signal-quality thresholds and hysteresis so a unit near a coverage boundary does
not "ping-pong" between two sites. During an active call, well-designed systems can hand
a unit's call over as it roams so the conversation is not dropped mid-transmission,
though the exact capability depends on the system and its network backhaul.

Roaming and registration are two halves of the same behaviour: roaming is the *decision*
to change sites, and [registration](/reference/registration/) is the *announcement* that
carries it out. The system's picture of where every unit is stays correct only because
each roam ends in a registration, and delivering a wide-area call depends on that picture —
the controller sends the call to the site (or sites) where the target's registration says
it is. When a unit roams between two RF Subsystems of a very large network, the transfer
may involve a hand-off between controllers as well, so that the unit's "home" record is
updated network-wide rather than just at the local site.

## Relevance to SDR

For a monitor, roaming is visible as a stream of [registration](/reference/registration/)
events: the same [radio ID](/reference/radio-id/) appearing on one site's control channel,
then on another's, traces the unit's movement across the network. GopherTrunk parses
registrations and the [neighbor-site](/reference/neighbor-site/) lists a site broadcasts,
so it can map the topology of a [multi-site system](/reference/multisite-trunking/) and
follow units as they roam between [trunking sites](/reference/trunking-site/). Practically,
this also tells a scanner which sites are worth monitoring for a given unit or talkgroup.

Real systems with wide-area roaming include P25 (multi-site/networked systems), Motorola
SmartZone, and DMR Tier III connected sites. GopherTrunk observes roaming from
control-channel signalling; it does not roam itself — a receiver stays on whatever site's
control channel the user has it tuned to, though the neighbor lists tell it where the
other sites are.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on multi-site operation and units moving between sites.
