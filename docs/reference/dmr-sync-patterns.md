---
slug: dmr-sync-patterns
title: DMR sync patterns
entry_type: term
category: synchronization
description: The nine 48-bit DMR sync words defined in ETSI TS 102 361-1 §9.1.1 mark the start and role of every burst, and they are closed under a discriminator-polarity flip so an inverted data sync is byte-identical to a clean voice sync.
keywords: DMR sync patterns, DMR sync words, BS-Voice sync, 0x755FD7DF75F7, MS-RC sync, DMR spectral inversion, polarity flip, dibit plus 2
aka: ["DMR sync words", "DMR sync patterns", "BS-Voice sync"]
autolink: true
infobox:
  - { label: Count, value: 9 sync words }
  - { label: Length, value: 48 bits (24 dibits) }
  - { label: Spec, value: ETSI TS 102 361-1 §9.1.1 }
  - { label: Closure, value: "XOR 0xAAAA… (dibit + 2)" }
see_also: [dmr-burst, dmr-slot-type, frame-synchronization, correlate-access-code, scrambling, dmr, dmr-reverse-channel, direct-mode-operation, tdma]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

The **DMR sync patterns** are the nine fixed 48-bit words that open the sync field of every
[DMR](/reference/dmr/) burst, telling the decoder both *where* a burst begins and *what role*
it plays.[^wiki] Before a receiver can read a [slot type](/reference/dmr-slot-type/) or a
payload it must lock onto one of these words in the demodulated dibit stream — exactly the job
of [frame synchronization](/reference/frame-synchronization/).[^fsync] The nine words are
defined in ETSI TS 102 361-1 §9.1.1 and split by source (Base Station, Mobile Station, Direct
Mode) and traffic type (voice, data, reverse channel).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DMR burst with its central 48-bit sync field highlighted; a sliding 24-dibit correlator locks onto one of the nine sync words, and an inverted data sync maps onto a clean voice sync under the dibit-plus-two polarity flip." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="130" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="85" y="47" text-anchor="middle" font-size="8" fill="currentColor">payload half · 98 b</text>
  <rect x="150" y="30" width="140" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="220" y="47" text-anchor="middle" font-size="8" fill="currentColor">SYNC · 48 bits</text>
  <rect x="290" y="30" width="130" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="355" y="47" text-anchor="middle" font-size="8" fill="currentColor">payload half · 98 b</text>
  <text x="150" y="80" font-size="8" font-family="monospace" fill="currentColor">BS-Voice  0x755FD7DF75F7</text>
  <text x="150" y="94" font-size="8" font-family="monospace" fill="currentColor">XOR 0xAAAAAAAAAAAA  ↓</text>
  <text x="150" y="108" font-size="8" font-family="monospace" fill="currentColor">BS-Data   0xDFF57D75DF5D</text>
  <path d="M60 122 L140 122 L140 106" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="90" y="138" text-anchor="middle" font-size="7.5" fill="currentColor">polarity flip = dibit + 2 mod 4</text>
</svg>
<figcaption>Each burst is framed by a central 48-bit sync word; the nine words are closed under the discriminator-polarity flip, so a spectrum-inverted data sync arrives byte-identical to a clean voice sync.</figcaption>
</figure>

## The nine sync words

Each word is 48 bits, expressed as 24 dibits MSB-first. GopherTrunk stores them in
`internal/radio/dmr/sync.go` as the package-level `AllSyncs` slice, in the stable order below.

| Name | Hex (low 48 bits) | Role |
|---|---|---|
| BS-Voice | `0x755FD7DF75F7` | Base station outbound, voice |
| BS-Data | `0xDFF57D75DF5D` | Base station outbound, data / control |
| MS-Voice | `0x7F7D5DD57DFD` | Mobile station, voice |
| MS-Data | `0xD5D7F77FD757` | Mobile station, data |
| MS-RC | `0x77D55F7DFD77` | Mobile station, [reverse channel](/reference/dmr-reverse-channel/) |
| DM-Voice-TS1 | `0x5D577F7757FF` | [Direct mode](/reference/direct-mode-operation/) voice, slot 1 |
| DM-Voice-TS2 | `0x7DFFD5F55D5F` | Direct mode voice, slot 2 |
| DM-Data-TS1 | `0xF7FDD5DDFD55` | Direct mode data, slot 1 |
| DM-Data-TS2 | `0xD7557F5FF7F5` | Direct mode data, slot 2 |

A `SyncDetector` slides a 24-dibit window across the stream and, at each position, counts dibit
mismatches against every pattern; a hit is declared where the best match falls within the
configured tolerance (default 4 of 24). The words are chosen for sharp autocorrelation, so a
genuine alignment stands clear of noise even a few dB into the channel.

## The polarity-flip closure

The most consequential property of the set is that it is *closed* under a whole-alphabet
polarity flip. A conjugated-I/Q or spectrum-inverted front end negates the FM-discriminator
output, mapping `+3 ↔ −3` and `+1 ↔ −1` — in dibit space, adding 2 mod 4 to every dibit, which
is exactly XOR-ing the 48-bit word with `0xAAAAAAAAAAAA`. Apply that to BS-Voice
(`0x755FD7DF75F7`) and the result is `0xDFF57D75DF5D` — BS-Data, bit-for-bit. Every voice word
maps to its data twin and back, the flip being its own inverse (`2 + 2 ≡ 0 mod 4`).

The practical consequence, recorded as GopherTrunk issue #264 (the RTL-SDR Blog V4 / R828D
path), is that a sync match *alone cannot resolve polarity*: an inverted data burst fires the
detector as a clean voice sync and vice versa. GopherTrunk therefore treats sync detection as
polarity-agnostic and resolves the ambiguity downstream — the Tier II / III `Process` adapters
hand each burst to the decoder at both candidate polarities (`CandidatePolarities = {0, 2}`,
`RotateBurstDibits`), and the slot-type Hamming(20,8), [BPTC(196,96)](/reference/bptc/), and
CSBK CRC together act as the arbiter, dropping the wrong polarity with no state change.

## Relevance to SDR

The sync patterns are the front door of GopherTrunk's DMR decoder. `internal/radio/dmr/sync.go`
holds the canonical dibit forms, the `AllSyncs` ordering callers index against, and the
`SyncDetector` that reports each hit's stream position and matched pattern. Voice decoding uses
only the four voice syncs to anchor burst A of a [superframe](/reference/dmr-voice-superframe/),
then locates bursts B–F by [TDMA](/reference/tdma/) cadence; control decoding slices a burst on
every BS-Data / MS-Data hit. Getting both the 24-dibit patterns and the polarity closure right
is what lets everything downstream decode a real, possibly inverted, off-air DMR signal.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard whose bursts these sync words frame.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on locating frame boundaries with a fixed sync sequence.
