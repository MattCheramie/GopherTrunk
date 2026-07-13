---
slug: cpdlc
title: Controller-Pilot Data Link Communications (CPDLC)
entry_type: protocol
category: aviation-marine
description: CPDLC (Controller-Pilot Data Link Communications) is an aviation protocol that exchanges text-based ATC clearances and requests between controllers and pilots over VHF, HF, and satcom data links.
keywords: CPDLC, Controller-Pilot Data Link Communications, ATN, FANS-1/A, air traffic control datalink, VDL Mode 2, HFDL, ACARS, satcom, ATC clearance
aka: [CPDLC]
autolink: true
infobox:
  - { label: Type, value: ATC text messaging }
  - { label: Standards body, value: ICAO / RTCA / EUROCAE }
  - { label: Introduced, value: FANS-1/A 1995, ATN B1 2010s }
  - { label: Access, value: Point-to-point over VDL/HFDL/satcom }
  - { label: Bearers, value: VDL Mode 2, HFDL, Inmarsat/Iridium }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [vdl-mode-2, hfdl, acars, ads-c, icao]
cite_urls:
  - https://en.wikipedia.org/wiki/Controller%E2%80%93pilot_data_link_communications
  - https://www.icao.int/
---

**CPDLC** (**Controller-Pilot Data Link Communications**) is an aviation protocol that
lets air traffic controllers and flight crews exchange **routine clearances, requests,
and reports as text messages** instead of voice. It rides over air-ground data links
such as [VDL Mode 2](/reference/vdl-mode-2/), [HFDL](/reference/hfdl/), and satellite
communications, and in older deployments over character-oriented
[ACARS](/reference/acars/).[^wiki] By moving structured, repetitive traffic off the
congested voice channels, CPDLC reduces frequency loading and read-back errors,
especially on oceanic and remote routes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A controller and an aircraft exchanging text clearance and request messages through a ground network and a radio or satellite data link." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cpdlcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="45" width="80" height="34" rx="3" fill="none" stroke="currentColor"/>
  <text x="60" y="66" text-anchor="middle" font-size="9" fill="currentColor">ATC ground</text>
  <rect x="200" y="45" width="70" height="34" rx="3" fill="none" stroke="currentColor"/>
  <text x="235" y="66" text-anchor="middle" font-size="8" fill="currentColor">datalink</text>
  <rect x="365" y="45" width="75" height="34" rx="3" fill="none" stroke="currentColor"/>
  <text x="402" y="66" text-anchor="middle" font-size="9" fill="currentColor">aircraft</text>
  <line x1="100" y1="56" x2="196" y2="56" stroke="currentColor" marker-end="url(#cpdlcar)"/>
  <line x1="270" y1="56" x2="361" y2="56" stroke="currentColor" marker-end="url(#cpdlcar)"/>
  <line x1="361" y1="70" x2="270" y2="70" stroke="currentColor" marker-end="url(#cpdlcar)"/>
  <line x1="196" y1="70" x2="100" y2="70" stroke="currentColor" marker-end="url(#cpdlcar)"/>
  <text x="230" y="110" text-anchor="middle" font-size="8.5" fill="currentColor">"CLIMB TO FL350" / "REQUEST DIRECT" — structured text, not voice</text>
</svg>
<figcaption>CPDLC carries structured clearances and requests as text between controllers and aircraft over VHF, HF, or satellite data links.</figcaption>
</figure>

## Overview

A CPDLC session pairs one aircraft with the controlling authority (ATC unit) that
currently owns it. Controllers send **uplink** messages drawn from a standardised set
(clearances, instructions, requests for information, negotiations), and crews reply with
**downlink** messages (requests, confirmations, WILCO/UNABLE responses). Message elements
are defined by ICAO so that the same intent renders consistently regardless of the
underlying bearer. Two message-standard families coexist: the older **FANS-1/A**, built
on ACARS and used widely over oceans, and the newer **ATN Baseline 1 (B1)**, used in
European continental airspace.

## Technical characteristics

| Property | Value |
|----------|-------|
| Message set | ICAO Doc 4444 / DO-258A element library |
| Bearers | VDL Mode 2, HFDL, Inmarsat/Iridium satcom, legacy ACARS |
| Families | FANS-1/A (ACARS-based), ATN B1 |
| Session | One controlling ATC unit at a time, with hand-off |
| Content | Clearances, requests, reports, free text |

## History

CPDLC emerged from the ICAO Future Air Navigation System (FANS) effort in the 1990s, with
FANS-1/A pioneered by Boeing and Airbus for oceanic operations where voice HF is poor. The
ATN variant, standardised by [RTCA](/reference/rtca/) and EUROCAE and coordinated through
[ICAO](/reference/icao/), followed for high-density continental airspace under programmes
such as Europe's Data Link Services mandate.

## Deployment

CPDLC is now routine on oceanic tracks (North Atlantic, Pacific) and in European en-route
sectors, typically alongside [ADS-C](/reference/ads-c/) position contracts. Airliners
carry it in their communication management units; general aviation rarely does.

## Decoding it with GopherTrunk

GopherTrunk is a land-mobile trunking scanner and does **not** decode CPDLC. The protocol
sits above bearers that GT does not currently demodulate, and much of its interest is in
higher-layer messaging rather than the RF/trunking control plane GT targets. Hobbyists
who want to see CPDLC typically decode the underlying VDL Mode 2 or ACARS link with
purpose-built aviation tools.

## Sources

[^wiki]: [Controller-pilot data link communications](https://en.wikipedia.org/wiki/Controller%E2%80%93pilot_data_link_communications) — Wikipedia, for the CPDLC message concept, FANS-1/A and ATN families, and the VHF/HF/satcom bearers it runs over.
