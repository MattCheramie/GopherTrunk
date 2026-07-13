---
slug: 3gpp
title: 3GPP (3rd Generation Partnership Project)
entry_type: organization
category: organizations
description: "3GPP is the partnership of regional standards bodies that authors the cellular specifications — GSM, UMTS, LTE, and 5G NR — used worldwide."
keywords: 3GPP, 3rd Generation Partnership Project, cellular standards, GSM, UMTS, LTE, 5G NR, mobile standards, ETSI, releases
aka: [3GPP, 3rd Generation Partnership Project]
autolink: true
infobox:
  - { label: Type, value: Standards partnership }
  - { label: Founded, value: "1998" }
  - { label: Standards, value: GSM, UMTS, LTE, 5G NR }
see_also: [gsm, lte, 5g-nr, umts-wcdma, gprs, etsi]
cite_urls:
  - https://www.3gpp.org/
  - https://en.wikipedia.org/wiki/3GPP
---

**3GPP** (the **3rd Generation Partnership Project**) is a collaboration of regional
telecommunications standards bodies that develops and maintains the specifications behind
the world's cellular networks, from [GSM](/reference/gsm/) through [LTE](/reference/lte/)
to [5G NR](/reference/5g-nr/).[^home] Rather than a single national institute, it is an
umbrella under which seven "Organizational Partners" — including [ETSI](/reference/etsi/)
in Europe, ATIS in North America, and counterparts in Japan, China, Korea, and India —
pool their work into one globally shared body of specifications.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="3GPP unifies regional standards bodies into one specification set spanning GSM, UMTS, LTE, and 5G NR." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tgpp_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="14" width="90" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="65" y="27">ETSI · ATIS</text>
    <rect x="20" y="44" width="90" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="65" y="57">ARIB · TTC</text>
    <rect x="20" y="74" width="90" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="65" y="87">CCSA · TTA · TSDSI</text>
    <rect x="170" y="44" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="215" y="63">3GPP</text>
    <rect x="320" y="14" width="120" height="18" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="380" y="26">GSM · GPRS</text>
    <rect x="320" y="40" width="120" height="18" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="380" y="52">UMTS · LTE</text>
    <rect x="320" y="66" width="120" height="18" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="380" y="78">5G NR</text>
    <g stroke="currentColor" stroke-width="1">
      <line x1="110" y1="24" x2="168" y2="52" marker-end="url(#tgpp_ar)"/>
      <line x1="110" y1="54" x2="168" y2="59" marker-end="url(#tgpp_ar)"/>
      <line x1="110" y1="84" x2="168" y2="66" marker-end="url(#tgpp_ar)"/>
      <line x1="260" y1="55" x2="318" y2="24" marker-end="url(#tgpp_ar)"/>
      <line x1="260" y1="59" x2="318" y2="49" marker-end="url(#tgpp_ar)"/>
      <line x1="260" y1="63" x2="318" y2="74" marker-end="url(#tgpp_ar)"/>
    </g>
  </g>
</svg>
<figcaption>Seven regional partners feed one 3GPP process that publishes the cellular standards.</figcaption>
</figure>

## Overview

3GPP was formed in December 1998, originally to produce a single third-generation mobile
standard building on GSM. Its remit quickly broadened to cover the entire evolution of
cellular systems, and the "3G" in its name is now historical — the same organization
authors 4G [LTE](/reference/lte/) and 5G [NR](/reference/5g-nr/), and continues toward 6G.
A sister project, 3GPP2, once maintained the competing CDMA2000 family, but that lineage
has faded and 3GPP's specifications now dominate globally.

Work is organized into numbered **Releases** (Release 99, Release 8, Release 15, and so
on), each a stable snapshot that vendors and operators build to. Release 8 introduced LTE;
Release 15 introduced the first 5G NR specifications. Within each release the technical
work is split across Technical Specification Groups — notably RAN (radio access network),
SA (service and system aspects), and CT (core network and terminals) — whose working
groups draft the thousands of documents that define everything from the air-interface
modulation and channel coding to the signalling protocols and security architecture.
Crucially, 3GPP itself does not publish or ratify the final legal standards; it hands the
specifications to its Organizational Partners, so a 3GPP specification reaches the market
as, for example, an ETSI TS document.

## Relevance to SDR

For anyone working with software-defined radio, 3GPP specifications are the authoritative
description of what a cellular signal actually looks like on the air. They define the frame
structures, reference signals, and modulation of [GSM](/reference/gsm/),
[GPRS](/reference/gprs/), [UMTS/WCDMA](/reference/umts-wcdma/), [LTE](/reference/lte/), and
[5G NR](/reference/5g-nr/) — the OFDM numerologies, the physical broadcast channel, the
synchronization sequences that an SDR must find to decode a cell. Open-source projects such
as srsRAN and OpenAirInterface implement these standards directly, and SDR hardware is the
usual front end for experimenting with them.

GopherTrunk is a trunked land-mobile scanner and does **not** decode cellular traffic; the
protocols it targets — P25, DMR, NXDN, TETRA — come from other bodies such as
[ETSI](/reference/etsi/), the TIA, and APCO. Cellular systems are out of scope for its
decode chain, both technically (they use wideband OFDM and complex scheduling) and
legally. 3GPP nonetheless matters as context: land-mobile radio and cellular share a great
deal of underlying DSP — matched filtering, forward error correction, TDMA framing — and
3GPP's public documents are among the best-written references for how a modern digital
radio link is engineered end to end.

## Sources

[^home]: [3GPP](https://www.3gpp.org/) — the partnership's official site, where its cellular specifications and Releases are published.
[^wiki]: [3GPP](https://en.wikipedia.org/wiki/3GPP) — Wikipedia, for the organization's history, its Organizational Partners, and the standards it maintains.
