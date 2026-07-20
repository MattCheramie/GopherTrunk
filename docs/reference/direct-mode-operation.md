---
slug: direct-mode-operation
title: Direct mode operation (DMO)
entry_type: concept
category: land-mobile-trunking
description: Direct mode operation lets radios talk peer-to-peer without any repeater or base station, trading trunking features for line-of-sight simplex range.
keywords: DMO, direct mode operation, talkaround, simplex, TMO, TETRA direct mode, DMR direct mode, peer to peer radio, off network
aka: [DMO, Direct mode operation, talkaround, simplex operation]
autolink: true
infobox:
  - { label: Type, value: Peer-to-peer radio mode }
  - { label: Also called, value: Talkaround, simplex }
  - { label: Opposite of, value: Trunked mode (TMO) }
see_also: [tetra, dmr, trunked-radio, talkgroup, conventional-radio, failsoft, channel-grant]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Talkaround
---

**Direct mode operation** (**DMO**) is a mode in which radios talk **peer-to-peer**,
handset to handset, bypassing the infrastructure entirely — no repeater and no base
station in the path.[^wiki] It is the counterpart to normal
[trunked](/reference/trunked-radio/) operation, where every call is routed through a
network of sites; in DMO the radios simply transmit and receive on a shared channel the
way a walkie-talkie does. The same idea is called **talkaround** or **simplex** in
conventional land-mobile parlance.[^talk]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="On the left, two handsets talk directly to each other in direct mode. On the right, two handsets each link through a central tower in trunked mode." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="115" y="20" font-size="10">Direct mode (DMO)</text>
    <text x="345" y="20" font-size="10">Trunked mode (TMO)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <rect x="40" y="90" width="20" height="40" rx="3"/>
    <rect x="170" y="90" width="20" height="40" rx="3"/>
    <line x1="62" y1="100" x2="168" y2="100" stroke-dasharray="4 3"/>
    <line x1="168" y1="120" x2="62" y2="120" stroke-dasharray="4 3"/>
  </g>
  <text x="115" y="150" font-size="8" fill="currentColor" text-anchor="middle">peer-to-peer, line of sight</text>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <rect x="280" y="95" width="20" height="40" rx="3"/>
    <rect x="390" y="95" width="20" height="40" rx="3"/>
    <line x1="345" y1="45" x2="345" y2="80"/>
    <path d="M336 80 L345 45 L354 80 Z" fill="currentColor" fill-opacity="0.2"/>
    <line x1="302" y1="105" x2="340" y2="70"/>
    <line x1="388" y1="105" x2="350" y2="70"/>
  </g>
  <text x="345" y="150" font-size="8" fill="currentColor" text-anchor="middle">every call through the site</text>
</svg>
<figcaption>DMO keeps traffic between the two radios; trunked mode routes every call through the network's tower.</figcaption>
</figure>

## How it works

In DMO the two (or more) radios agree on a single simplex frequency and share it: one
transmits while the others listen, keyed by push-to-talk, with no repeater retransmitting
the signal on a second frequency. Because nothing amplifies or re-radiates the
transmission from a tall site, range is strictly **radio-to-radio line of sight** —
typically a few kilometres on the ground, far less than the wide-area coverage a trunked
site provides. The trade is deliberate: DMO keeps working when the network cannot.

Different standards give the mode different names but the concept is identical.
[TETRA](/reference/tetra/) draws the sharpest line, contrasting **DMO** with **TMO**
(trunked mode operation) as two formally specified modes a radio switches between.
[DMR](/reference/dmr/) calls it **direct mode**, where two radios share a channel using
the same two-slot TDMA structure without a repeater. P25 and analog conventional systems
call it **direct** or **talkaround** — a button that drops the repeater and transmits on
the output frequency directly.

Switching to DMO means giving up the trunking machinery. There is no control channel, so
there are no [channel grants](/reference/channel-grant/) assigning a traffic frequency per
call, no wide-area roaming between sites, and no centralised affiliation of a
[talkgroup](/reference/talkgroup/). Users pick a channel manually and stay on it. This
makes DMO closer in spirit to [conventional radio](/reference/conventional-radio/) than to
trunking, even on a radio that is otherwise a trunked subscriber unit.

## Where it is used

DMO is the fallback and the local-work mode. It is used when radios are **out of network
coverage** — inside a building, down in a valley, or beyond the last site — and for
**on-scene traffic** where a crew working together does not need to burden the network,
such as firefighters on a fireground talking among themselves. It is distinct from
[failsoft](/reference/failsoft/), the degraded mode a trunked *site* falls back to when it
loses its controller; DMO removes the infrastructure from the path entirely rather than
running a crippled version of it.

Many systems allow a **DMO gateway** or **DMO repeater**: a radio or fixed unit that
bridges direct-mode traffic back into the trunked network, so a crew working simplex on
scene can still reach dispatch. The gateway listens on the direct channel and relays onto
the infrastructure, extending DMO's short range without every handset needing a network
link.

## Relevance to SDR

For an SDR listener, DMO traffic looks different from trunked traffic on the waterfall:
there is no control channel to follow and no grant to chase, so calls appear as
intermittent simplex transmissions on fixed frequencies rather than as a managed hop
across a site's channel pool. GopherTrunk's trunking follower keys on the control channel
and channel grants, so purely direct-mode conversations fall outside that flow and are
monitored like conventional channels — parked on the known simplex frequency. Recognising
that a system has dropped to DMO explains why expected grant activity on the control
channel goes quiet while voice is still on the air nearby.

## Sources

[^wiki]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, for TETRA's DMO versus TMO distinction and the loss of trunking features off-network.
[^talk]: [Talkaround](https://en.wikipedia.org/wiki/Talkaround) — Wikipedia, for the conventional/simplex "talkaround" equivalent of direct mode.
