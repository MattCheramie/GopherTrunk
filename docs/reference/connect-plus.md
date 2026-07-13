---
slug: connect-plus
title: Connect Plus
entry_type: protocol
category: land-mobile-trunking
description: Connect Plus is Motorola's proprietary multi-site DMR trunking system that dedicates one control channel per site to manage roaming across a wide-area network.
keywords: Connect Plus, Motorola Connect Plus, MOTOTRBO, multi-site DMR trunking, XRC controller, DMR Tier III, wide-area trunking, rest channel
aka: [Connect Plus, "Con Plus", "MOTOTRBO Connect Plus"]
autolink: true
infobox:
  - { label: Type, value: Proprietary multi-site DMR trunking }
  - { label: Standards body, value: "Motorola (proprietary, on ETSI DMR)" }
  - { label: Access, value: TDMA (2 slots) }
  - { label: Channel spacing, value: 12.5 kHz }
  - { label: Modulation, value: 4FSK (4800 baud, 9600 bps) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: Tiers, value: "Motorola trunking atop DMR" }
  - { label: GopherTrunk support, value: See Status }
see_also: [dmr, capacity-plus, rest-channel, multisite-trunking, control-channel, csbk]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/MOTOTRBO
---

**Connect Plus** is Motorola's proprietary **multi-site [DMR](/reference/dmr/) trunking**
system, part of the MOTOTRBO family, that ties several sites into one wide-area network
and lets subscribers roam between them. Unlike single-site [Capacity Plus](/reference/capacity-plus/),
each Connect Plus site **dedicates one timeslot as a continuous
[control channel](/reference/control-channel/)** managed by an XRC network controller,
so it behaves much like a classic Tier III trunked system.[^wiki][^trbo]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Three Connect Plus sites, each with a dedicated control channel, linked by an XRC controller into one roaming network." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cp_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="200" y="20" width="60" height="26" fill="currentColor" fill-opacity="0.18"/><text x="230" y="37">XRC</text>
    <rect x="30" y="110" width="90" height="40" fill="none"/><text x="75" y="103">Site A</text>
    <rect x="185" y="110" width="90" height="40" fill="none"/><text x="230" y="103">Site B</text>
    <rect x="340" y="110" width="90" height="40" fill="none"/><text x="385" y="103">Site C</text>
  </g>
  <g stroke="currentColor" stroke-width="1"><line x1="230" y1="46" x2="75" y2="110" marker-end="url(#cp_ar)"/><line x1="230" y1="46" x2="230" y2="110" marker-end="url(#cp_ar)"/><line x1="230" y1="46" x2="385" y2="110" marker-end="url(#cp_ar)"/></g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle"><text x="75" y="132">ctrl+voice</text><text x="230" y="132">ctrl+voice</text><text x="385" y="132">ctrl+voice</text></g>
  <text x="230" y="165" text-anchor="middle" font-size="8" fill="currentColor">each site runs its own dedicated control channel</text>
</svg>
<figcaption>Connect Plus links sites through an XRC controller; every site carries its own dedicated DMR control channel for wide-area roaming.</figcaption>
</figure>

## Overview

Connect Plus was Motorola's answer to wide-area DMR trunking before the
ETSI [Tier III](/reference/dmr-tier-3/) standard and the later Capacity Max product
matured. It layers a proprietary signalling and networking scheme on top of the standard
DMR air interface, so the radio-frequency modulation is ordinary DMR while the trunking
logic — grants, registration, and roaming — follows Motorola's own formats. A network of
sites is coordinated by one or more XRC 9000 controllers, giving subscribers seamless
handoff as they move between coverage areas.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | Two-slot [TDMA](/reference/tdma/) |
| Channel | 12.5 kHz |
| Modulation | 4FSK, 4800 [baud](/reference/symbol-rate/) (9600 bps) |
| Vocoder | AMBE+2 |
| Control | Dedicated control-channel slot per site |
| Networking | XRC controller, proprietary site linking |
| Signalling | Motorola control-signalling blocks (variant of DMR [CSBK](/reference/csbk/)) |

The dedicated control channel is the key contrast with Capacity Plus, which instead
rotates control among pooled channels using a [rest channel](/reference/rest-channel/).
Because the control slot runs continuously, a scanner can lock the outbound signalling
and follow channel grants much as it would on a conventional trunked system.

## History

Motorola introduced Connect Plus in the late 2000s as a multi-site extension to its
MOTOTRBO line, filling the gap between single-site Capacity Plus and true standardised
Tier III. It saw wide deployment in commercial fleet, utility, and campus systems that
needed regional coverage. Motorola later positioned **Capacity Max** as the strategic
successor, offering larger capacity and closer alignment with the DMR standard, and has
steered new deployments toward it.

## Deployment

Connect Plus is common in transportation, utilities, security, and industrial fleets
across North America and elsewhere, wherever an operator needed multi-site DMR coverage
under a single subscriber fleet. Many of these networks remain in service even as
Motorola migrates customers to Capacity Max, so Connect Plus traffic is still frequently
seen on the air in the UHF and VHF land-mobile bands.

## Decoding it with GopherTrunk

Connect Plus rides on the same DMR physical layer GopherTrunk already demodulates —
4FSK at 4800 baud with AMBE+2 voice — so the raw bursts and voice frames are recoverable.
The trunk-following logic, however, depends on Motorola's proprietary control-channel
formats, which differ from standard Tier III signalling; support for parsing those grants
and following roaming is best confirmed against the current [Status](/status.html) page
rather than assumed. Encrypted talkgroups remain out of scope, as GopherTrunk decodes
clear and known-key traffic only.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, for the DMR air interface, its two-slot TDMA structure, and Motorola's proprietary trunking modes built on it.
[^trbo]: [MOTOTRBO](https://en.wikipedia.org/wiki/MOTOTRBO) — Wikipedia, for the MOTOTRBO product family and the Connect Plus multi-site trunking option with dedicated control channels.
