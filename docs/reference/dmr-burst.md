---
slug: dmr-burst
title: DMR burst
entry_type: term
category: trunked-radio
description: A DMR burst is the 264-bit (132-dibit) unit one timeslot transmits every 30 ms — two 98-bit payload halves surrounding a central 48-bit field that carries either a sync word or embedded signalling, with slot-type fields on data bursts.
keywords: DMR burst, 264 bits, 132 dibits, DMR burst structure, payload halves, sync field, embedded signalling, TDMA timeslot, ETSI TS 102 361-1
aka: ["DMR burst", "TDMA burst", "DMR timeslot burst"]
autolink: true
infobox:
  - { label: Size, value: 264 bits (132 dibits) }
  - { label: Duration, value: 27.5 ms of a 30 ms slot }
  - { label: Layout, value: 98 + 48 + 98 bits }
  - { label: Spec, value: ETSI TS 102 361-1 §6.2 / §6.4.2 }
see_also: [dmr-sync-patterns, dmr-slot-type, dmr-emb, dmr-voice-superframe, bptc, tdma, dmr, dmr-cach, csbk]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

A **DMR burst** is the fundamental transmission unit of [DMR](/reference/dmr/): the
**264 bits** (132 [dibits](/reference/dibit/)) one [TDMA](/reference/tdma/) timeslot sends in
its 27.5 ms of a 30 ms frame.[^wiki] Every burst has the same skeleton — two 98-bit payload
halves wrapped around a central 48-bit field — and that central field is what distinguishes a
data/control burst (which puts a [sync word](/reference/dmr-sync-patterns/) or
[slot type](/reference/dmr-slot-type/) there) from a voice burst (which puts
[embedded signalling](/reference/dmr-emb/) there).[^tdma]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A 132-dibit DMR data burst laid out left to right: a 49-dibit first payload half, a 5-dibit slot-type field, the 24-dibit sync or embedded field, a second 5-dibit slot-type field, and a 49-dibit second payload half, with the two 98-bit halves joining into a 196-bit BPTC codeword." xmlns="http://www.w3.org/2000/svg">
  <rect x="12" y="34" width="110" height="28" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="67" y="51" text-anchor="middle" font-size="8" fill="currentColor">info · 98 b</text>
  <rect x="122" y="34" width="40" height="28" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="142" y="51" text-anchor="middle" font-size="7" fill="currentColor">ST 10</text>
  <rect x="162" y="34" width="110" height="28" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/>
  <text x="217" y="51" text-anchor="middle" font-size="8" fill="currentColor">SYNC / EMB · 48 b</text>
  <rect x="272" y="34" width="40" height="28" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="292" y="51" text-anchor="middle" font-size="7" fill="currentColor">ST 10</text>
  <rect x="312" y="34" width="110" height="28" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="367" y="51" text-anchor="middle" font-size="8" fill="currentColor">info · 98 b</text>
  <path d="M12 74 L122 74" stroke="currentColor" stroke-width="1"/><path d="M312 74 L422 74" stroke="currentColor" stroke-width="1"/>
  <path d="M67 74 L67 82 L367 82 L367 74" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="217" y="98" text-anchor="middle" font-size="8" fill="currentColor">two halves → 196-bit BPTC(196,96) codeword</text>
</svg>
<figcaption>A data/control burst: 49 payload dibits, a 5-dibit slot-type field, the 24-dibit sync field, a second slot-type field, and 49 more payload dibits; the two 98-bit halves concatenate into one BPTC codeword.</figcaption>
</figure>

## Burst layout

GopherTrunk models the burst in `internal/radio/dmr/burst.go` as 132 dibits. For a
data/control burst the ETSI TS 102 361-1 §6.2 / §6.4.2 layout is:

| Dibits | Bits | Field |
|---|---|---|
| 0–48 | 98 | `info[0]` — first payload half |
| 49–53 | 10 | slot type, before sync |
| 54–77 | 48 | sync word or embedded signalling |
| 78–82 | 10 | slot type, after sync |
| 83–131 | 98 | `info[1]` — second payload half |

The two 98-bit info halves are read out and concatenated (`Burst.PayloadBits`) into the
196-bit [BPTC(196,96)](/reference/bptc/) codeword that carries a
[CSBK](/reference/csbk/), voice link-control header, or data block. The two 10-bit slot-type
fields around the sync concatenate into a 20-bit [slot-type](/reference/dmr-slot-type/)
codeword (`Burst.SlotTypeBitsAll`). Splitting the payload in half and placing the sync and
slot type *between* the halves is deliberate: it puts the most reliably-recovered fields — the
sync landmark and the FEC-heavy slot type — at the burst's centre, where a receiver that has
locked the sync is best synchronised, and it spreads the 196 payload bits symmetrically around
that anchor so a timing slip at either edge damages only one half.

## Voice bursts differ

A **voice** burst carries no slot-type fields. Its split is 108 + 48 + 108 bits: three 72-bit
[AMBE+2](/reference/ambe-plus-2/) voice frames plus the central 48-bit field. The voice
information reclaims the 20 bits the data burst spent on slot type, so the same 264-bit
envelope holds more speech. In burst A of a [voice superframe](/reference/dmr-voice-superframe/)
the central field is a voice sync word; in bursts B–F it holds
[embedded signalling](/reference/dmr-emb/) instead — a 16-bit EMB framing a 32-bit fragment —
which is why only burst A produces a sync match and the rest are located by TDMA cadence. A
burst is thus self-describing only in part: the sync word and slot type identify a data burst
outright, but a voice burst past the first is recognised by its position in the cadence, not by
anything in the burst itself.

## Relevance to SDR

`Burst` and its accessors are the seam between GopherTrunk's dibit-level demodulator and its
protocol decoders. `FirstHalf`/`SecondHalf`, `Sync`, `SlotTypeBefore`/`SlotTypeAfter`, and the
`PayloadBits`/`SlotTypeBitsAll` bit-packers give every downstream layer — sync detection, slot
type, BPTC, embedded LC — a consistent view of the same 132 dibits. The same file also carries
the polarity machinery (`RotateBurstDibits`, `CandidatePolarities`) that lets a burst be
re-decoded at the flipped polarity a spectrum-inverted front end imprints, so the burst is the
natural place both the clean and inverted decode paths converge.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its burst/timeslot structure.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on the two-slot TDMA framing DMR bursts occupy.
