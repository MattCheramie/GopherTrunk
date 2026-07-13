---
slug: gsma
title: GSMA
entry_type: organization
category: organizations
description: "The GSMA is the industry association representing the world's mobile network operators, coordinating standards, spectrum policy, and interoperability across the cellular ecosystem."
keywords: GSMA, GSM Association, mobile operators, MWC, cellular industry, IMEI, roaming, spectrum policy
aka: [GSMA, GSM Association]
autolink: true
infobox:
  - { label: Type, value: Industry trade association }
  - { label: Founded, value: "1995" }
  - { label: Members, value: Mobile operators worldwide }
see_also: [gsm, 3gpp, lte, 5g-nr, imsi-imei]
cite_urls:
  - https://www.gsma.com/
  - https://en.wikipedia.org/wiki/GSMA
---

The **GSMA** (originally the **GSM Association**) is the **global industry body that
represents the world's mobile network operators**.[^wiki] It brings together several hundred
operators, along with device makers, chipset vendors, and software companies, to coordinate
the commercial and technical machinery that lets cellular networks interoperate — roaming
agreements, numbering and identity registries, spectrum-policy advocacy, and the annual
Mobile World Congress trade event.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 118" role="img" aria-label="The GSMA coordinating mobile operators worldwide while 3GPP writes the underlying air-interface standards." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="170" y="12" width="120" height="28" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="30">GSMA</text>
    <rect x="24" y="80" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="69" y="97">operators</text>
    <rect x="130" y="80" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="175" y="97">roaming</text>
    <rect x="236" y="80" width="90" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="281" y="97">IMEI registry</text>
    <rect x="342" y="80" width="96" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="390" y="93" font-size="8">3GPP standards</text><text x="390" y="103" font-size="7.5">(referenced)</text>
    <g stroke="currentColor" stroke-width="1.1">
      <line x1="215" y1="40" x2="80" y2="79" marker-end="url(#ar_gsma)"/>
      <line x1="225" y1="40" x2="178" y2="79" marker-end="url(#ar_gsma)"/>
      <line x1="240" y1="40" x2="278" y2="79" marker-end="url(#ar_gsma)"/>
      <line x1="255" y1="40" x2="382" y2="79" stroke-dasharray="3 3" marker-end="url(#ar_gsma)"/>
    </g>
  </g>
  <defs><marker id="ar_gsma" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The GSMA coordinates operators and industry-wide services; 3GPP writes the radio standards the GSMA relies on.</figcaption>
</figure>

## Overview

The GSMA grew out of the group of European operators who deployed the original
[GSM](/reference/gsm/) digital cellular system in the early 1990s, and it took its current
name and worldwide scope in 1995. Its work is organised around keeping a global, competitive
mobile industry technically compatible. It administers the identity systems that make that
possible — allocating IMEI ranges for devices and coordinating the numbering that underpins
international roaming — and it maintains registries and settlement mechanisms so that a phone
from one network can work on another abroad.

Beyond the plumbing, the GSMA acts as the industry's collective voice on policy: it lobbies at
spectrum forums for harmonised mobile allocations, publishes guidelines on network security
and fraud, and drives industry initiatives such as eSIM provisioning and RCS messaging. It is
important to distinguish its role from that of [3GPP](/reference/3gpp/): 3GPP is the
partnership that actually writes the air-interface specifications — GSM, UMTS,
[LTE](/reference/lte/), and [5G NR](/reference/5g-nr/) — while the GSMA represents the
operators who deploy those standards commercially and coordinates everything around them. The
GSMA's most public face is Mobile World Congress, the large annual gathering it hosts in
Barcelona.

## Relevance to SDR

Cellular downlinks are among the strongest and most structured signals an SDR encounters, and
the GSMA sits behind the ecosystem that produces them. The identifiers a mobile analyst reads
about — the [IMSI and IMEI](/reference/imsi-imei/) — live within numbering schemes the GSMA
administers, and the roaming and eSIM machinery it runs is why a single band plan carries
traffic from many networks. Understanding that the GSMA coordinates operators while
[3GPP](/reference/3gpp/) defines the radio interface helps place any cellular signal in
context.

GopherTrunk is a land-mobile trunking scanner and does not decode cellular traffic — modern
cellular is encrypted and out of scope — so the GSMA has no direct role in its decode chain.
It is included here as background for the broader RF world, where cellular occupies a large,
tightly coordinated slice of the spectrum that SDR users routinely see on the waterfall even
if they cannot demodulate it.

## Sources

[^home]: [GSMA](https://www.gsma.com/) — the association's official site, for its membership, industry programmes, and identity registries.
[^wiki]: [GSMA](https://en.wikipedia.org/wiki/GSMA) — Wikipedia, for the organisation's history and its role representing mobile operators.
