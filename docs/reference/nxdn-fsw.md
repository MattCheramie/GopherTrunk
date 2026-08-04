---
slug: nxdn-fsw
title: NXDN Frame Sync Word
entry_type: term
category: synchronization
description: "The NXDN Frame Sync Word (FSW) is the fixed 16-bit pattern that opens every NXDN frame — 0xC55A outbound (BS→MS) and 0x3AA5 inbound (MS→BS) — letting a decoder find where a frame begins in the dibit stream."
keywords: NXDN frame sync word, NXDN FSW, 0xC55A, 0x3AA5, NXDN sync pattern, frame synchronization NXDN, RAN, 4FSK dibit sync
aka: [FSW, "NXDN sync word", "NXDN frame sync"]
autolink: true
infobox:
  - { label: Length, value: 16 bits (8 dibits) }
  - { label: Outbound, value: "0xC55A (BS→MS)" }
  - { label: Inbound, value: "0x3AA5 (MS→BS)" }
  - { label: Spec, value: NXDN TS 1-A §4.4 }
see_also: [ran-nxdn, frame-synchronization, nxdn, nxdn-frame-structure, nxdn-lich, correlate-access-code, four-fsk]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

The **NXDN Frame Sync Word** (**FSW**) is the fixed 16-bit pattern that opens every
[NXDN](/reference/nxdn/) frame, letting a decoder find where a frame begins in the incoming
dibit stream.[^wiki] Its value depends on direction: **`0xC55A`** for the outbound link
(base station to mobile) and **`0x3AA5`** for the inbound link (mobile to base station).
Before a decoder can read the [LICH](/reference/nxdn-lich/), the SACCH, or the information
field, it must know *where* the frame starts — and the FSW is the landmark it searches for,
exactly the job of [frame synchronization](/reference/frame-synchronization/).[^fsync]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="The NXDN outbound frame sync word 0xC55A shown as eight dibits at the head of a frame, followed by the LICH, SACCH, and information field; a sliding correlator locks onto the sync word to mark the frame boundary, and the inbound direction uses the complementary pattern 0x3AA5." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="28" width="110" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="75" y="45" text-anchor="middle" font-size="9" fill="currentColor">FSW · 8 dibits</text>
  <rect x="130" y="28" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="165" y="45" text-anchor="middle" font-size="8" fill="currentColor">LICH</text>
  <rect x="200" y="28" width="90" height="26" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="245" y="45" text-anchor="middle" font-size="8" fill="currentColor">SACCH</text>
  <rect x="290" y="28" width="160" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="370" y="45" text-anchor="middle" font-size="8" fill="currentColor">information field</text>
  <text x="20" y="74" font-size="8" font-family="monospace" fill="currentColor">outbound 0xC55A = 3 0 1 1 1 1 2 2   (dibits)</text>
  <text x="20" y="88" font-size="8" font-family="monospace" fill="currentColor">inbound  0x3AA5 = 0 3 2 2 2 2 1 1   (bit-complement)</text>
  <path d="M40 108 L100 108 L100 96" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="75" y="126" text-anchor="middle" font-size="8" fill="currentColor">correlator peak = frame start</text>
</svg>
<figcaption>Every NXDN frame opens with an 8-dibit sync word; a sliding correlator locks onto it to mark the boundary, then the decoder reads the LICH, SACCH, and information field that follow. The inbound word is the bit-complement of the outbound word.</figcaption>
</figure>

## How it works

An NXDN receiver produces a stream of **dibits** (two bits per 4FSK symbol) from the
demodulator. The FSW is 8 of those dibits in a fixed order, so the decoder runs a sliding
[access-code correlation](/reference/correlate-access-code/): at each new dibit it compares
the last 8 dibits against the known pattern and declares a frame boundary wherever the number
of mismatches falls under a configured tolerance. GopherTrunk's `SyncDetector` accepts one or
more FSW patterns and reports each hit's dibit index, plus whether the matched pattern was the
inbound word — a downlink monitor searches for `0xC55A`, while an uplink or direct-mode monitor
also carries `0x3AA5`.

The two directional words are exact bit-complements of one another: `0xC55A` XOR `0x3AA5`
equals `0xFFFF`. Expanded to dibits (most-significant first), the outbound word is
`3 0 1 1 1 1 2 2` and the inbound word is `0 3 2 2 2 2 1 1`. That complementary relationship
means a whole-alphabet polarity flip on the demodulator — a common consequence of a
conjugated I/Q input — turns one direction's pattern into the other's, so a detector that
carries both words is naturally robust to a swapped-sideband front end.

| Direction | Hex | Dibits (MSB-first) |
| --- | --- | --- |
| Outbound (BS→MS) | `0xC55A` | `3 0 1 1 1 1 2 2` |
| Inbound (MS→BS) | `0x3AA5` | `0 3 2 2 2 2 1 1` |

## FSW versus RAN

The FSW answers *where* a frame starts; it does not say *which system* the frame belongs to.
That second job falls to the [Radio Access Number](/reference/ran-nxdn/) (RAN), the 6-bit
per-frame identifier NXDN carries in the SACCH so a repeater or radio accepts only its own
system's traffic on a shared channel. The FSW is identical for every NXDN system on a given
link direction — it is a universal landmark — whereas the RAN differs per system. A decoder
first correlates the FSW to lock the frame grid, then reads the RAN to decide whether the
frame is worth following. The two are complementary: sync without RAN would lock onto every
co-channel system indiscriminately; RAN without sync would have no aligned bits to read the
RAN from.

## Relevance to SDR

The FSW is the front door of GopherTrunk's NXDN decoder. `internal/radio/nxdn/sync.go` holds
the canonical `FSWOutboundHex` / `FSWInboundHex` constants, materialises them into 8-dibit
patterns, and runs a `SyncDetector` that slides those patterns over the demodulated dibits and
reports each hit's absolute index and direction. Every NXDN frame — control (CAC), voice, or
data — begins at an FSW hit, so getting this search right is what lets everything downstream
decode at all. The exact bit patterns follow the values cited across public NXDN reference
implementations; GopherTrunk's source flags that they should be cross-checked against the
published NXDN technical document before being trusted on unusual captures, since the sync
search is the one place a wrong constant would silently prevent any lock.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN narrowband digital land-mobile standard and its frame structure.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on using a fixed sync sequence to locate frame boundaries in a bit stream.
