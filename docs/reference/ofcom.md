---
slug: ofcom
title: Ofcom
entry_type: organization
category: organizations
description: Ofcom is the United Kingdom's communications regulator, allocating and licensing radio spectrum — the UK counterpart to the FCC.
keywords: Ofcom, UK regulator, spectrum, licensing, United Kingdom, communications, Office of Communications, WT Act
aka: [Ofcom]
autolink: true
infobox:
  - { label: Type, value: UK government regulator }
  - { label: Founded, value: "2003" }
  - { label: Role, value: UK spectrum allocation and licensing }
see_also: [fcc, itu, cept, ntia, frequency-bands, rsgb]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
cite_urls:
  - https://www.ofcom.org.uk/
  - https://en.wikipedia.org/wiki/Ofcom
---

**Ofcom** (the Office of Communications) is the **United Kingdom's communications
regulator**.[^wiki] It allocates and licenses radio spectrum and sets the rules for its use —
the UK counterpart to the United States' [FCC](/reference/fcc/), working within the
global framework set by the [ITU](/reference/itu/).[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The ITU global framework within which Ofcom regulates UK spectrum and licenses users, coordinated regionally through CEPT." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="55">ITU (global)</text><text x="75" y="67" font-size="7.5">+ CEPT (Europe)</text>
    <rect x="175" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="55">Ofcom (UK)</text><text x="230" y="67" font-size="7.5">allocate + license</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">licensed users</text><text x="385" y="67" font-size="7.5">+ licence-exempt</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="130" y1="58" x2="174" y2="58" marker-end="url(#or_ofc)"/><line x1="285" y1="58" x2="329" y2="58" marker-end="url(#or_ofc)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">international allocations → UK band plan → licences &amp; rules</text>
  </g>
  <defs><marker id="or_ofc" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Ofcom regulates UK spectrum within the ITU's international framework, coordinated regionally through CEPT.</figcaption>
</figure>

## Overview

Ofcom was created by the Office of Communications Act 2002 and began operating in 2003,
merging five earlier regulators (including the Radiocommunications Agency) into a single body
responsible for the UK's broadcasting, telecommunications, postal, and — most relevant here —
radio-spectrum affairs. Its spectrum work rests on the Wireless Telegraphy Act, under which it
manages the UK Frequency Allocation Table, issues (and in some bands auctions) licences, sets
technical conditions and power limits, defines licence-exempt bands, and enforces the rules by
investigating interference and unlicensed transmitters. Like every national regulator, Ofcom
does not invent its band plan from scratch: it implements the international allocations of the
[ITU](/reference/itu/) and the harmonised European arrangements coordinated through
[CEPT](/reference/cept/), the body whose recommendations align spectrum use across European
countries. This is the same relationship the [FCC](/reference/fcc/) and
[NTIA](/reference/ntia/) have with the ITU in the United States.

For a hobbyist, Ofcom's most visible outputs are the licences and licence-exempt permissions
that govern day-to-day radio use in Britain: the amateur-radio licence (whose interests are
represented to Ofcom by the [RSGB](/reference/rsgb/)), business radio, PMR446 licence-exempt
handhelds, and the conditions on consumer and short-range devices. It also publishes the UK
Frequency Allocation Table itself, which is the authoritative answer to "what is this band for
in the UK". Crucially for listeners, Ofcom — not any international body — determines what may
lawfully be *received* in the UK, and UK law here differs notably from the US: it is generally
an offence to receive messages you are not authorised to receive, a stricter position than the
American one.

## Relevance to SDR

Ofcom matters to a UK SDR user in two distinct ways. First, its band plan explains what you are
tuning to: where marine, aeronautical, business-radio, amateur, and licence-exempt services sit
in the UK, and why some allocations differ from those in North America (a consequence of the
ITU's three-region split and CEPT harmonisation). Second, and more importantly than for many
regulators, Ofcom defines the *legality* of reception. Because UK rules on receiving are
stricter than the US model, the question of what you may lawfully listen to and decode is a real
one, and Ofcom is the authority that answers it. This is exactly why guidance to "check the
regulator for your own jurisdiction" is not boilerplate.

GopherTrunk is a receiver and decoder and implements no regulator-specific logic — it will
demodulate whatever RF you feed it — so the responsibility to stay within Ofcom's rules rests
with the operator. See the [legal &amp; ethical monitoring](/learn/rf-sdr/legal-ethical/) lesson,
and always confirm the position for your own country before decoding traffic.

## Sources

[^home]: [Ofcom](https://www.ofcom.org.uk/) — the UK communications regulator's official site, for spectrum allocation, licensing, and the UK Frequency Allocation Table.
[^wiki]: [Ofcom](https://en.wikipedia.org/wiki/Ofcom) — Wikipedia, for Ofcom's creation, remit, and role as the UK communications regulator.
