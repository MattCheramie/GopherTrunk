---
slug: faa
title: Federal Aviation Administration (FAA)
entry_type: organization
category: organizations
description: The FAA, the Federal Aviation Administration, is the US agency that runs the National Airspace System and drove the ADS-B mandate on 1090 MHz and UAT 978.
keywords: FAA, Federal Aviation Administration, NextGen, ADS-B, UAT, TIS-B, FIS-B, air traffic control, 1090 MHz, 978 MHz
aka: [FAA, Federal Aviation Administration]
autolink: true
infobox:
  - { label: Type, value: US government agency (DOT) }
  - { label: Focus, value: Civil aviation, air traffic control }
  - { label: Founded, value: 1958 }
  - { label: Standards, value: ADS-B, UAT, TIS-B, FIS-B }
see_also: [ads-b, uat-978, tis-b, fis-b, mode-s, rtca, icao]
cite_urls:
  - https://www.faa.gov/
  - https://en.wikipedia.org/wiki/Federal_Aviation_Administration
---

**FAA** (the **Federal Aviation Administration**) is the United States Department of
Transportation agency that regulates civil aviation and operates the National Airspace
System, including air traffic control. Its NextGen modernisation drove the US
**[ADS-B](/reference/ads-b/)** mandate, carried on [Mode S](/reference/mode-s/) at
1090 MHz and on the **[UAT 978](/reference/uat-978/)** link.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The FAA's NextGen programme feeds four surveillance and data services: ADS-B on 1090 MHz, UAT on 978 MHz, TIS-B traffic, and FIS-B weather." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="55" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="65" y="70">FAA</text><text x="65" y="82" font-size="7.5">NextGen</text>
    <rect x="300" y="10" width="140" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="370" y="27">ADS-B · 1090 MHz</text>
    <rect x="300" y="44" width="140" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="370" y="61">UAT · 978 MHz</text>
    <rect x="300" y="78" width="140" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="370" y="95">TIS-B · traffic</text>
    <rect x="300" y="112" width="140" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="370" y="129">FIS-B · weather</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="110" y1="72" x2="299" y2="23" marker-end="url(#rel_faa)"/><line x1="110" y1="72" x2="299" y2="57" marker-end="url(#rel_faa)"/><line x1="110" y1="72" x2="299" y2="91" marker-end="url(#rel_faa)"/><line x1="110" y1="72" x2="299" y2="125" marker-end="url(#rel_faa)"/></g>
  </g>
  <defs><marker id="rel_faa" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>NextGen ties together the surveillance and data-link services the FAA broadcasts to and from equipped aircraft.</figcaption>
</figure>

## Overview

The FAA was created in 1958 and is responsible for the safety and regulation of US civil
aviation: it certifies aircraft and airmen, writes the Federal Aviation Regulations, and runs
the air traffic control system that keeps the National Airspace System orderly. Over the past
two decades its defining programme has been **NextGen**, a shift from ground radar to
satellite-based surveillance, of which **[ADS-B](/reference/ads-b/)** (Automatic Dependent
Surveillance – Broadcast) is the centrepiece. A rule effective January 2020 mandated ADS-B Out
in most controlled US airspace.

The FAA specifies two physical links. Air carriers and most turbine aircraft use 1090 MHz
Extended Squitter, riding on the [Mode S](/reference/mode-s/) transponder, while general
aviation may instead use the **[UAT 978](/reference/uat-978/)** (Universal Access Transceiver)
data link at 978 MHz. Because UAT is a two-way link, the FAA's ground infrastructure also
broadcasts two free services to equipped aircraft: **[TIS-B](/reference/tis-b/)**
(Traffic Information Service – Broadcast), which relays nearby traffic including non-ADS-B
targets seen by radar, and **[FIS-B](/reference/fis-b/)** (Flight Information Service –
Broadcast), which sends weather, NOTAMs, and other advisories. The FAA sets US operational
requirements but does not write the detailed signal standards alone: it works with
[RTCA](/reference/rtca/), whose DO-260 series defines the ADS-B message performance, while the
global framework above the FAA is set by [ICAO](/reference/icao/).

## Relevance to SDR

The FAA's mandates are why the airwaves over the United States are full of decodable aircraft
data. Because ADS-B, UAT, TIS-B, and FIS-B are open, unencrypted broadcasts designed for
interoperability, an inexpensive SDR and a dedicated decoder can receive them: 1090 MHz for
airliners with tools like `dump1090`, and 978 MHz to pull in the traffic and weather pictures
that TIS-B and FIS-B paint for general aviation. The FAA effectively turned a regulatory
requirement into one of the most popular SDR hobbies, and crowdsourced feeder networks now
depend on thousands of these receivers.

Aircraft surveillance sits outside GopherTrunk's land-mobile trunking focus, so GopherTrunk
does not itself decode these links; dedicated 978/1090 MHz tools handle them. The reference
stands as context for the wider RF landscape an SDR user explores, and it explains why US
aviation signals are so uniform and abundant: a single national authority mandated them.

## Sources

[^home]: [Federal Aviation Administration](https://www.faa.gov/) — the FAA's official site, for NextGen, the ADS-B rule, and the UAT/TIS-B/FIS-B services.
[^wiki]: [Federal Aviation Administration](https://en.wikipedia.org/wiki/Federal_Aviation_Administration) — Wikipedia, for the FAA's history, structure, and role in US aviation.
