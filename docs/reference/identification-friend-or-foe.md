---
slug: identification-friend-or-foe
title: Identification friend or foe (IFF)
entry_type: technology
category: aviation-marine
description: IFF is a military challenge-and-reply transponder system from WWII, using 1030 MHz interrogations and 1090 MHz replies; civil secondary surveillance radar descends from it.
keywords: IFF, identification friend or foe, secondary surveillance radar, SSR, transponder, interrogator, 1030 MHz, 1090 MHz, Mode 5, challenge reply
aka: [IFF, Identification friend or foe, secondary surveillance radar]
autolink: true
infobox:
  - { label: Type, value: Military transponder system }
  - { label: Frequencies, value: "1030 MHz challenge / 1090 MHz reply" }
  - { label: Origin, value: World War II }
see_also: [mode-ac, mode-s, ads-b, tcas, encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Identification_friend_or_foe
  - https://en.wikipedia.org/wiki/Secondary_surveillance_radar
---

**Identification friend or foe** (**IFF**) is a military transponder-and-interrogator
system, born in World War II, that answers a single question: is this radar contact a
friend?[^wiki] A ground or airborne **interrogator** transmits a coded challenge on
**1030 MHz**, and a friendly **transponder** aboard the target replies on **1090 MHz**
with a coded answer. Civil **secondary surveillance radar** (SSR) — the
[Mode A/C](/reference/mode-ac/) and [Mode S](/reference/mode-s/) systems that track
airliners — descends directly from IFF and shares the very same frequency pair.[^ssr]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A ground interrogator transmits a challenge at 1030 megahertz to an aircraft transponder, which replies at 1090 megahertz." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <path d="M40 150 L60 110 L80 150 Z" fill="currentColor" fill-opacity="0.15"/>
    <line x1="60" y1="110" x2="60" y2="95"/>
    <path d="M380 55 L420 45 L400 65 Z" fill="currentColor" fill-opacity="0.15"/>
    <line x1="360" y1="60" x2="400" y2="52"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <path d="M75 100 Q220 40 375 55" marker-end="url(#iff_arrow)"/>
    <path d="M375 80 Q220 120 80 130" marker-end="url(#iff_arrow)"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="55" y="168" text-anchor="middle">interrogator</text>
    <text x="400" y="38" text-anchor="middle">transponder</text>
    <text x="200" y="62" text-anchor="middle">challenge → 1030 MHz</text>
    <text x="210" y="140" text-anchor="middle">← reply 1090 MHz</text>
  </g>
  <defs><marker id="iff_arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The interrogator challenges on 1030 MHz; a friendly transponder replies on 1090 MHz. No reply is not proof of a foe.</figcaption>
</figure>

## How it works

IFF is a **cooperative** system: it works only because the friendly aircraft carries a
transponder that recognises the challenge and answers correctly. The interrogator sends a
pulse-coded challenge in one of several **modes**, and a valid transponder replies with a
coded response after a fixed delay. Matching the reply to the interrogation confirms the
contact is equipped and responding as a friend. This is fundamentally different from
primary radar, which merely bounces energy off a target and reports a blip with no
identity attached.

The wartime lineage runs straight into today's civil airspace. The military "modes" of
early IFF were opened for air-traffic use as **SSR Modes A and C** — Mode A returns a
four-digit squawk code, Mode C adds pressure altitude — and later
[Mode S](/reference/mode-s/) added selective, addressed interrogation and the data link
that [ADS-B](/reference/ads-b/) rides on. All of them still use the 1030/1090 MHz pair
inherited from IFF, which is why a single antenna and receiver band covers both military
and civil secondary surveillance.

Modern military IFF has moved well beyond those open modes. **Mode 5** uses
[encryption](/reference/encryption/) and spread-spectrum techniques so that challenges and
replies cannot be spoofed or exploited by an adversary, replacing the older Mode 4. The
civil and military systems coexist on the same frequencies but the secure military modes
are cryptographically protected.

## What it can and cannot do

The name is precise and slightly misleading. IFF identifies **friends** — it cannot mark
**foes**. A correct reply positively confirms a friend; the *absence* of a reply means
only that the contact is not a responding friend. It could be an enemy, or it could be a
friendly aircraft with a broken, switched-off, or wrongly keyed transponder. For this
reason IFF is one input to identification, never a standalone weapons-release
authority. The same cooperative-only limitation applies to civil SSR: an aircraft with no
working transponder is invisible to secondary radar even though it is plainly a friend,
which is part of why [TCAS](/reference/tcas/) collision avoidance and primary radar exist
alongside it.

## Relevance to SDR

For the SDR hobbyist, IFF's civil descendants are the accessible part: the 1090 MHz replies
that carry Mode A/C squawks, Mode S addresses, and ADS-B position reports are unencrypted
and decodable with a cheap receiver and software like `dump1090`. The military secure modes
are not. Understanding IFF explains *why* the aviation surveillance band is arranged the way
it is — a challenge-and-reply architecture from the 1940s, still built on 1030 MHz up and
1090 MHz down. Aircraft surveillance sits outside GopherTrunk's land-mobile trunking focus,
so it does not decode 1090 MHz itself; the reference stands as context for the wider RF
landscape.

## Sources

[^wiki]: [Identification friend or foe](https://en.wikipedia.org/wiki/Identification_friend_or_foe) — Wikipedia, for IFF's WWII origin, the 1030/1090 MHz challenge-reply pair, modes, and Mode 5.
[^ssr]: [Secondary surveillance radar](https://en.wikipedia.org/wiki/Secondary_surveillance_radar) — Wikipedia, for how civil SSR Modes A/C and S descend from IFF and share its frequencies.
