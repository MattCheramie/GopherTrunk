---
slug: tetra-sync-pdu
title: TETRA SYNC PDU
entry_type: term
category: trunked-radio
description: "The TETRA SYNC PDU is the MAC broadcast carried by the always-decodable BSCH: colour code, timeslot, frame and multiframe number, MCC and MNC — everything a cold receiver needs to build the cell's scrambling code and slot grid."
keywords: TETRA SYNC PDU, DM-SYNC, BSCH, colour code, frame number, multiframe number, timeslot number, MCC, MNC, extended colour code, EN 300 392-2, cold start
aka: [SYNC PDU, DM-SYNC PDU, TETRA synchronisation PDU]
autolink: true
infobox:
  - { label: Carried by, value: BSCH (block 1 of the sync burst) }
  - { label: Scrambled with, value: Colour code 0 — always decodable }
  - { label: Fields, value: "Colour, TN, FN, MN, MCC, MNC" }
  - { label: Spec, value: ETSI EN 300 392-2 §21.4.4.2 }
see_also: [tetra, tetra-extended-colour-code, tetra-mobile-network-identity, tetra-logical-channels, tetra-burst-formats, tetra-dmo, tetra-scrambler]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://www.etsi.org/deliver/etsi_en/300300_300399/30039202/03.04.01_60/en_30039202v030401p.pdf
---

**The TETRA SYNC PDU** is the small MAC broadcast that bootstraps every
[TETRA](/reference/tetra/) receiver: carried on the BSCH (block 1 of the synchronisation
downlink burst) and always scrambled with colour code 0, it is the one message a cold
receiver — with no configuration at all — can decode, and it carries exactly the fields
needed to unlock everything else.[^etsi] Once the SYNC PDU is in hand, the receiver knows
the cell's scrambling identity and its position in time, and every other
[logical channel](/reference/tetra-logical-channels/) becomes decodable.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The 60-bit SYNC PDU laid out as labelled fields: colour code, timeslot number, frame number, multiframe number, a reserved gap, mobile country code, and mobile network code." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="20" y="40" width="18" height="30"/><rect x="38" y="40" width="52" height="30"/><rect x="90" y="40" width="24" height="30"/><rect x="114" y="40" width="42" height="30"/><rect x="156" y="40" width="50" height="30"/><rect x="206" y="40" width="52" height="30"/><rect x="258" y="40" width="80" height="30"/><rect x="338" y="40" width="102" height="30"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="64" y="59">colour (6)</text><text x="102" y="59">TN</text><text x="135" y="59">FN (5)</text><text x="181" y="59">MN (6)</text><text x="232" y="59">…</text><text x="298" y="59">MCC (10)</text><text x="389" y="59">MNC (14)</text>
  </g>
  <text x="20" y="32" font-size="8" fill="currentColor">bit 0</text>
  <text x="440" y="32" font-size="8" fill="currentColor" text-anchor="end">bit 59</text>
  <text x="230" y="92" font-size="8.5" fill="currentColor" text-anchor="middle">MCC | MNC | colour concatenate into the 30-bit extended colour code that seeds the scrambler</text>
</svg>
<figcaption>Sixty type-1 bits, MSB-first: identity on the right, timing in the middle — the two halves of a cold start.</figcaption>
</figure>

## What the fields buy

- **Colour code (6 bits), MCC (10), MNC (14)** — concatenated MSB-first these form the
  30-bit [extended colour code](/reference/tetra-extended-colour-code/) that seeds the
  [scrambler](/reference/tetra-scrambler/) for every channel *except* the BSCH itself.
  This is the deliberate bootstrap: the BSCH stays at colour 0 so anyone can read it, and
  hands over the key to the rest. (Colour 0 is still real scrambling — the seed is
  non-zero — a fact that has bitten more than one implementation.)
- **Timeslot number (2 bits), frame number (5), multiframe number (6)** — the cell's
  position in the [TDMA](/reference/tdma/) hierarchy: 4 timeslots per frame, 18 frames per
  multiframe, with frame 18 reserved for control. A receiver that decodes two SYNC PDUs and
  sees FN advancing at the right cadence has proof of a genuine time lock, which makes FN
  advance a useful *liveness* signal long before payload decodes.

## One layout, two modes

The direct-mode variant — the **DM-SYNC PDU**, broadcast in the DSB of a
[TETRA DMO](/reference/tetra-dmo/) transmission (EN 300 396-3) — reuses the same field
layout, so a single parser serves both trunked and direct mode. GopherTrunk's
`ParseSyncPDU` (`internal/radio/tetra/sync_pdu.go`) decodes the 60 type-1 bits for either;
its bit offsets are byte-for-byte identical to the osmo-tetra reference parser. Two DMO
caveats are worth knowing: on air the DMO SCH/S carries MNI 0 (MCC = MNC = 0), so a
direct-mode network's real [mobile network identity](/reference/tetra-mobile-network-identity/)
cannot be learned from this PDU; and the DM colour code that scrambles DMO *traffic* is not
recoverable from the SCH/S either — both must come from configuration or empirical
recovery.

## Relevance to SDR

The SYNC PDU is the hinge of GopherTrunk's TETRA cold start: hunt a carrier, decode BSCH at
colour 0, parse the SYNC PDU, form the extended colour code, and only then do
BNCH/SCH/F/SCH/HD decode — no operator-supplied colour needed. Two hard-won operational
rules attach to it. First, *confirm identity changes*: a single mis-corrected BSCH can
decode with a wrong-but-CRC-valid MCC/MNC, so GopherTrunk requires two consecutive agreeing
SYNC PDUs before rewriting a locked cell identity (the alternative was bogus "cc locked
mcc=996" flaps). Second, *FN liveness is the lock heartbeat* for camped channels — a DMO
channel between transmissions stays "locked" on the strength of advancing frame numbers
rather than decode rate, so inter-PTT silence does not trigger a re-hunt. The bring-up traps
around this PDU's neighbours are collected in
[TETRA lock facts](/reference/tetra-lock-facts/).

## Sources

[^etsi]: [ETSI EN 300 392-2 (TETRA Voice plus Data, Air Interface)](https://www.etsi.org/deliver/etsi_en/300300_300399/30039202/03.04.01_60/en_30039202v030401p.pdf) — ETSI, §21.4.4.2 SYNC PDU field layout and §8.2.5.2 scrambling-code formation.
