---
slug: esim
title: eSIM
entry_type: hardware
category: hw-mobile
description: An eSIM is a SIM card built directly into a device as a reprogrammable chip, letting users download and switch carrier profiles over the air instead of inserting a physical card.
keywords: eSIM, embedded SIM, eUICC, SIM card, carrier profile, remote provisioning, virtual SIM, dual SIM
autolink: true
aka: [embedded SIM, eUICC]
infobox:
  - { label: Type, value: Embedded SIM chip }
  - { label: Replaces, value: Physical SIM card }
  - { label: Provisioning, value: Over the air }
  - { label: Standard, value: GSMA eUICC }
see_also: [cellular-modem, system-on-a-chip, smartphone, smartwatch, mobile-operating-system, near-field-communication]
cite_urls:
  - https://en.wikipedia.org/wiki/ESIM
---

An **eSIM** is a SIM built directly into a device as a small, reprogrammable chip, letting a user download and switch carrier profiles over the air instead of swapping a physical card.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A comparison of a physical SIM and an eSIM. On the left, a removable SIM card with a gold contact pad slides into a tray. On the right, an eSIM is a soldered chip on the board that receives one or more carrier profiles downloaded over the air from the carrier's provisioning server." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <path d="M46 44 h70 v56 h-84 v-42 z"/>
    <rect x="52" y="56" width="26" height="20" rx="2"/>
    <line x1="60" y1="56" x2="60" y2="76"/>
    <line x1="70" y1="56" x2="70" y2="76"/>
    <line x1="52" y1="66" x2="78" y2="66"/>
    <rect x="34" y="108" width="96" height="14" rx="2" stroke-dasharray="3 3"/>
    <rect x="300" y="46" width="120" height="54" rx="3"/>
    <g stroke-width="1">
      <line x1="300" y1="58" x2="290" y2="58"/>
      <line x1="300" y1="72" x2="290" y2="72"/>
      <line x1="300" y1="86" x2="290" y2="86"/>
      <line x1="420" y1="58" x2="430" y2="58"/>
      <line x1="420" y1="72" x2="430" y2="72"/>
      <line x1="420" y1="86" x2="430" y2="86"/>
    </g>
    <path d="M360 46 v-22 h-80" stroke-width="1.1"/>
    <path d="M283 24 l6 -4 v8 z" fill="currentColor" stroke="none"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="74" y="132" text-anchor="middle">physical SIM &#183; removable</text>
    <text x="360" y="116" text-anchor="middle">eSIM (eUICC) &#183; soldered</text>
    <text x="220" y="20" text-anchor="middle" font-size="8">profile downloaded over the air</text>
    <text x="360" y="130" text-anchor="middle" font-size="8">holds several profiles</text>
  </g>
</svg>
<figcaption>Where a physical SIM is a removable card in a tray, an eSIM is a soldered eUICC chip that downloads carrier profiles over the air — no tray, and room for several profiles at once.</figcaption>
</figure>

## Overview

Technically an *eUICC* (embedded Universal Integrated Circuit Card) standardized by the GSMA, an eSIM solders the SIM function onto the board and holds one or more downloadable *profiles*. Activating a plan means scanning a QR code or following a carrier flow; switching networks or running a second line no longer requires opening a tray or handling a fragile nano-card.

Freeing a device from a card slot saves internal space and improves water and dust resistance, and it lets tiny products — [smartwatches](/reference/smartwatch/), trackers, IoT sensors — carry cellular service at all. For manufacturers it also simplifies logistics: one hardware SKU ships worldwide and is provisioned to the right carrier after the sale.

## Physical SIM vs eSIM

| Aspect | Physical SIM | eSIM (eUICC) |
|--------|--------------|--------------|
| Form | Removable card | Soldered chip |
| Change carrier | Swap card | Download profile |
| Profiles held | One | Several |
| Space & sealing | Needs tray | None; better sealed |
| Remote fleet setup | Manual visit | Over the air |

## Where it fits

The eSIM is the connectivity counterpart to a device's [cellular modem](/reference/cellular-modem/): the modem provides the radio, the eSIM the identity the network authenticates. For a fleet of remote GopherTrunk capture nodes on cellular backhaul, eSIM provisioning means carriers and data plans can be assigned and changed remotely, without anyone visiting each node to insert or swap a card.

## Sources

[^wiki]: [eSIM](https://en.wikipedia.org/wiki/ESIM) — Wikipedia, on embedded SIM technology and remote provisioning.
