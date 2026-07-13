---
slug: arib
title: ARIB (Association of Radio Industries and Businesses)
entry_type: organization
category: organizations
description: "ARIB is Japan's standards body for radio and broadcasting, best known for the ISDB digital television and audio broadcasting standards used in Japan and Brazil."
keywords: ARIB, Association of Radio Industries and Businesses, Japan, ISDB, ISDB-T, digital broadcasting, standards, spectrum
aka: [ARIB, Association of Radio Industries and Businesses]
autolink: true
infobox:
  - { label: Type, value: Standards organization }
  - { label: Region, value: Japan }
  - { label: Standards, value: ISDB family }
see_also: [isdb-t, dab, atsc-1, dvb-t, ofdm]
cite_urls:
  - https://www.arib.or.jp/english/
  - https://en.wikipedia.org/wiki/Association_of_Radio_Industries_and_Businesses
---

**ARIB** (the **Association of Radio Industries and Businesses**) is Japan's standards-
setting organization for radio use and broadcasting, developing standards and coordinating
spectrum on behalf of the Japanese communications industry.[^home] It is best known
internationally for the [ISDB](/reference/isdb-t/) family — Integrated Services Digital
Broadcasting — the digital television and radio standards adopted across Japan and, in a
modified form, much of South America.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 108" role="img" aria-label="ARIB develops Japan's radio and broadcasting standards, most notably the ISDB terrestrial, satellite, and mobile television family." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="arib_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="100" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">ARIB</text>
    <rect x="200" y="8" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="260" y="23">ISDB-T (terrestrial)</text>
    <rect x="200" y="46" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="260" y="61">ISDB-S (satellite)</text>
    <rect x="200" y="84" width="120" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="260" y="99">1seg (mobile)</text>
    <g stroke="currentColor" stroke-width="1"><line x1="120" y1="50" x2="198" y2="20" marker-end="url(#arib_ar)"/><line x1="120" y1="57" x2="198" y2="57" marker-end="url(#arib_ar)"/><line x1="120" y1="64" x2="198" y2="94" marker-end="url(#arib_ar)"/></g>
  </g>
</svg>
<figcaption>ARIB develops Japan's ISDB broadcasting standards for terrestrial, satellite, and mobile reception.</figcaption>
</figure>

## Overview

ARIB was established in 1995 as a private-sector body designated by Japan's Ministry of
Internal Affairs and Communications to develop standards for the effective use of radio and
to promote the radio and broadcasting industries. It publishes **ARIB STD** standards and
technical reports covering broadcasting, mobile communications, and other radio services in
Japan, and it acts as a coordination point between industry and the regulator on spectrum
matters, playing a role in Japan broadly analogous to what [ETSI](/reference/etsi/) does in
Europe.

Its signature contribution is the **ISDB** family of digital broadcasting standards. ISDB-T
carries terrestrial digital television, ISDB-S carries satellite television, and the
"1seg" subset delivers television to mobile handsets by using one of the thirteen frequency
segments into which each ISDB-T channel is divided. That segmented [OFDM](/reference/ofdm/)
structure is ISDB-T's most distinctive feature and lets a single transmission serve both
fixed high-definition receivers and battery-powered mobile devices. ISDB-T was later adopted
across Brazil and much of South America as a regional variant, giving a Japanese standard a
broad international footprint.

## Relevance to SDR

ISDB broadcasts are a natural target for wideband software-defined radios: like
[DVB-T](/reference/dvb-t/) and [ATSC](/reference/atsc-1/), ISDB-T occupies a full television
channel of several megahertz, so capturing it demands a high-sample-rate front end, but the
signals are open and well documented in ARIB's published standards. The segmented OFDM
structure is visible on a wideband spectrogram, and SDR experimenters use ISDB-T reception
to study OFDM synchronization, pilot patterns, and hierarchical transmission. ARIB's role,
like that of any broadcasting standards body, is what makes such reception reproducible: the
specification defines exactly the frame layout a receiver must follow.

GopherTrunk does not decode ISDB or any broadcast television standard; it is a narrowband
trunked land-mobile scanner, and wideband digital TV is outside its scope. ARIB is included
here to complete the global picture of digital-broadcasting standards bodies — the Japanese
counterpart to Europe's DVB and North America's ATSC — for readers mapping who governs each
region's over-the-air digital services.

## Sources

[^home]: [ARIB](https://www.arib.or.jp/english/) — the association's official English site, for its standards, technical reports, and role in Japanese radio and broadcasting.
[^wiki]: [Association of Radio Industries and Businesses](https://en.wikipedia.org/wiki/Association_of_Radio_Industries_and_Businesses) — Wikipedia, for ARIB's history and its ISDB standards.
