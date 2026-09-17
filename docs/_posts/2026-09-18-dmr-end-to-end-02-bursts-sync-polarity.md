---
title: "DMR End to End, Part 2: Bursts, Sync Words & Polarity"
description: How GopherTrunk turns a raw DMR dibit stream into bursts — the 132-dibit layout with its central sync, the nine 48-bit sync words and the polarity flip that maps every data sync onto a voice sync, the Golay-protected slot type that routes each burst, and the one FEC-valid burst that fixes a stream's polarity.
category: deep-dives
keywords: dmr burst structure, dmr sync patterns, dmr sync word polarity flip, dmr spectrum inversion, dmr slot type golay 20 8, dmr color code data type, dmr sync detector, dmr bs voice bs data sync, gophertrunk dmr
tags: [dmr-end-to-end, dmr, framing, sync, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 2
---

*Part 2 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 1]({{ '/blog/deep-dives/dmr-end-to-end-01-4fsk-carrier/' | relative_url }})
turned IQ into dibits at 4800 a second and planted the thread: one carrier,
two of everything. This part gives the firehose its punctuation — burst,
sync word, slot type — and meets the first concrete twin: DMR's data and
voice sync words are each other's image under a polarity flip, so an
inverted front end makes every data burst look like voice and vice versa,
and the sync detector alone cannot tell which stream it is hearing.*

> **TL;DR:** A DMR burst is **132 dibits**: 49 of payload, a 5-dibit
> slot-type field, the 24-dibit **sync**, another slot-type field, 49 more
> of payload (`internal/radio/dmr/burst.go`). Nine 48-bit sync words
> (`dmr.AllSyncs`, `sync.go`) mark burst boundaries; a `SyncDetector` reports
> the best match within tolerance (2 in production). The words are **closed
> under `PolarityFlip`** — adding 2 mod 4 to every dibit turns BS-Data into
> BS-Voice bit for bit (`TestSyncPairsClosedUnderPolarityFlip`) — so an
> inverted stream syncs happily and reports the wrong twin. The **slot type**
> (`slottype.go`) is colour code + data type under a (20,8,7) code
> (`framing.HammingDecode20_8`, corrects 3), read only from data-sync
> bursts; polarity is resolved by decoding at both `CandidatePolarities`
> and letting FEC arbitrate. Tier II then **locks** the polarity on the
> first FEC-valid burst (`c.polarity`, `tier2/process.go`) and never drops
> it.

**Key takeaways**

- **The burst is a fixed frame around a central landmark.** Sync and slot
  type sit mid-burst; a match at index *i* means a burst from *i* − 77 to
  *i* + 54.
- **A sync match tells you where, not which.** The nine words are closed
  under the flip, so a BS-Voice match is either a clean voice burst or an
  inverted data burst; only FEC can say.
- **The slot type routes before anyone parses.** Colour code plus data type
  under a distance-7 code lets `IngestBurst` dispatch a burst without
  touching its 196-bit payload.
- **Polarity is learned once and kept.** Tier II tries both polarities until
  the first FEC-valid burst, then only the learned one — which makes "slot
  types only from data syncs" exact and closes the door on Part 4's forged
  terminator.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Burst layout | 49 + 5 + 24 + 5 + 49 dibits | `internal/radio/dmr/burst.go` (`Burst`, `PayloadBits`, `SlotTypeBitsAll`) |
| Nine sync words | 48-bit patterns as 24 dibits; sliding detector | `internal/radio/dmr/sync.go` (`AllSyncs`, `SyncDetector`) |
| Polarity flip | `(dibit + 2) & 3`; candidates {0, 2} | `burst.go` (`PolarityFlip`, `RotateBurstDibits`, `CandidatePolarities`) |
| Data sync per polarity | which words carry a slot type after rotation | `sync.go` (`IsDataSync`, `SyncIsDataAtPolarity`) |
| Slot type | colour code + data type, (20,8,7) t = 3 | `slottype.go` (`ParseSlotType`); `framing/hamming20.go` |
| Polarity lock | first FEC-valid burst, kept across resets | `tier2/process.go`, `conventional.go` (`polarity`, `burstValid`) |

## In this post

- **The burst, with offsets** — 132 dibits, and how a match becomes a burst.
- **Nine words, twenty-four dibits** — the sync alphabet and the detector.
- **The polarity twin** — why a match can't tell data from inverted voice.
- **The slot type** — colour code, data type, twelve parity bits.
- **Locking the polarity** — Tier II's first FEC-valid burst.

## The burst, with offsets

[Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
drew the burst; this part needs its coordinates, because everything that
follows is expressed in dibit offsets from the sync:

```go
// internal/radio/dmr/burst.go (shape)
//   dibits   0..48   payload first half
//   dibits  49..53   slot type before sync
//   dibits  54..77   sync / embedded signalling
//   dibits  78..82   slot type after sync
//   dibits  83..131  payload second half
const (
    BurstDibits       = 132
    HalfPayloadDibits = 49
    SlotTypeDibits    = 5
    SyncDibits        = 24
)
```

`Burst.PayloadBits` concatenates the halves into the 196-bit
[BPTC(196,96)]({{ '/reference/bptc/' | relative_url }}) codeword a data
burst carries; `SlotTypeBitsAll` concatenates the two 5-dibit fields into
the 20-bit slot-type codeword. A voice burst uses the same envelope
differently — 54 + 24 + 54 dibits, three 72-bit AMBE+2 frames and **no
slot-type fields** (`voice.VoiceBits`) — the fact Part 4's forged
terminator hinges on.

The framers slice a burst from a match with three shared constants — the
match index is the *last* sync dibit, so `burstLookback` is 77,
`burstLookahead` 54 and `bufKeep` 163 — and the
[burst reference]({{ '/reference/dmr-burst/' | relative_url }}) explains
why sync and slot type sit in the *middle*.

## Nine words, twenty-four dibits

ETSI TS 102 361-1 §9.1.1 defines nine 48-bit sync words, split by source and
traffic type:

```go
// internal/radio/dmr/sync.go (shape)
var (
    BSVoice  = mkSync("BS-Voice", 0x755FD7DF75F7)
    BSData   = mkSync("BS-Data",  0xDFF57D75DF5D)
    MSVoice  = mkSync("MS-Voice", 0x7F7D5DD57DFD)
    MSData   = mkSync("MS-Data",  0xD5D7F77FD757)
    MSRC     = mkSync("MS-RC",    0x77D55F7DFD77)
    DMVoice1 = mkSync("DM-Voice-TS1", 0x5D577F7757FF)
    DMVoice2 = mkSync("DM-Voice-TS2", 0x7DFFD5F55D5F)
    DMData1  = mkSync("DM-Data-TS1",  0xF7FDD5DDFD55)
    DMData2  = mkSync("DM-Data-TS2",  0xD7557F5FF7F5)
)
```

Read the hex digits: every nibble is 5, 7, D or F — in dibits, only ever
±3. Sync words are built from **outer symbols exclusively**, which is why a
receiver whose AGC has collapsed the eye still matches syncs while every
payload fails, and why Part 9's gap noise (outer symbols sliced as inner)
kills sync detection outright.

`SyncDetector.Process` slides a 24-dibit ring over the stream, counts
mismatches against each pattern with early exit, and appends a
`Match{Index, Pattern}` when the best is within tolerance — 4 of 24 by
default, **2** in every production adapter. Tier I restricts the detector to the four direct-mode words; Tier II
and III take all nine
([sync-pattern reference]({{ '/reference/dmr-sync-patterns/' | relative_url }})).

## The polarity twin

A spectrum-inverted or I/Q-swapped front end (issue #264 — the RTL-SDR
Blog V4 / R828D path) negates the FM discriminator output, mapping +3 ↔ −3
and +1 ↔ −1. In Part 1's dibit map — high bit is the sign — that is exactly **add 2 mod 4** to every dibit: `PolarityFlip =
2` in `burst.go`, self-inverse (2 + 2 ≡ 0), applied by `RotateBurstDibits`
and enumerated with identity as `CandidatePolarities = {0, 2}`.

And the sync alphabet is *closed* under that flip. Apply it to BS-Voice
(`0x755FD7DF75F7`) and the result is `0xDFF57D75DF5D` — BS-Data, bit for
bit. MS-Voice ↔ MS-Data and both DM pairs behave the same way; MS-RC is the
lone word without a twin. `TestSyncPairsClosedUnderPolarityFlip` pins all
four pairs in both directions.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two rows of DMR bursts. On a clean stream a burst framed by BS-Data carries a slot type and a burst framed by BS-Voice carries AMBE bits in the same positions. On an inverted stream every dibit gains two modulo four, so the data burst now matches BS-Voice and the voice burst matches BS-Data. A gate at the right, SyncIsDataAtPolarity, hands a burst to the slot-type parser only when its sync is a data word at the polarity being tried, and the first FEC-valid decode locks that polarity.">
  <text x="12" y="28" fill="currentColor" font-size="10" font-weight="bold">clean stream (polarity 0)</text>
  <rect x="12" y="38" width="200" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="112" y="59" text-anchor="middle" fill="currentColor" font-size="9">data burst · sync BS-Data</text>
  <rect x="228" y="38" width="200" height="34" rx="4" fill="none" stroke="var(--fg-muted)"/>
  <text x="328" y="59" text-anchor="middle" fill="currentColor" font-size="9">voice burst A · sync BS-Voice</text>
  <text x="12" y="118" fill="var(--accent)" font-size="10" font-weight="bold">inverted stream: every dibit + 2 mod 4</text>
  <rect x="12" y="128" width="200" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="112" y="149" text-anchor="middle" fill="var(--accent)" font-size="9">same data burst reads BS-Voice</text>
  <rect x="228" y="128" width="200" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="328" y="149" text-anchor="middle" fill="var(--accent)" font-size="9">same voice burst reads BS-Data</text>
  <line x1="112" y1="72" x2="112" y2="128" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="328" y1="72" x2="328" y2="128" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="220" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the detector fires on both rows and cannot tell them apart</text>
  <rect x="456" y="60" width="212" height="102" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="562" y="80" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">SyncIsDataAtPolarity(p, k)</text>
  <text x="562" y="98" text-anchor="middle" fill="currentColor" font-size="9">k = 0: IsDataSync(p) · k = 2: IsVoiceSync(p)</text>
  <text x="562" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">only then rotate, parse slot type, IngestBurst</text>
  <text x="562" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="8">first burstValid ⇒ c.polarity = k, kept</text>
  <line x1="428" y1="55" x2="456" y2="80" stroke="var(--fg-muted)"/>
  <line x1="428" y1="145" x2="456" y2="120" stroke="var(--fg-muted)"/>
  <text x="340" y="204" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">a sync match says where, never which polarity</text>
</svg>
<figcaption>The polarity twin: an inverted data burst is byte-identical to a clean voice burst at the sync, so the slot-type parser is gated per polarity and the first FEC-valid burst decides which world the stream lives in.</figcaption>
</figure>

So **the detector already fires on an inverted stream — it just reports the
flipped twin.** Nothing at the sync layer can tell the rows apart, and
GopherTrunk does not try: both adapters hand `IngestBurst` a candidate at
each polarity and let the FEC below — slot type, then BPTC(196,96), then
RS(12,9) or the CSBK CRC — drop the wrong one with no state change. It is the rule of
[P25 Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
and
[TETRA Part 2]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})'s
rotation searches: **the sync layer finds candidates; FEC decides.** (An
operator can also fix an inverted device once: `iq_invert: true` sets
`ccdecoder.Options.Conjugate`, negating Q on every raw sample — the
correction TETRA *requires*.)

## The slot type

Each data burst carries its slot type twice — 10 bits before the sync, 10
after — concatenated into one 20-bit codeword:

```go
// internal/radio/dmr/slottype.go (shape)
// bits 0..3 colour code · bits 4..7 data type · bits 8..19 parity
func ParseSlotType(bits []byte) (SlotType, int, error) {
    var cw uint32 /* … pack the 20 bits MSB-first … */
    data, errs := framing.HammingDecode20_8(cw) // errs < 0 ⇒ uncorrectable
    if errs < 0 { return SlotType{}, -1, ErrSlotTypeUncorrectable }
    return SlotType{ColorCode: (data >> 4) & 0x0F, DataType: DataType(data & 0x0F)}, errs, nil
}
```

The code is a (20,8,7) shortened Hamming/Golay with minimum distance 7, so
it corrects up to three bit errors. `framing.HammingDecode20_8` decodes by brute-force minimum
distance over all 256 valid codewords rather than a syndrome table: 256
entries fit in cache and the twelve parity masks stay auditable against the
spec. Hold on to **256 of 2²⁰**: it is the seed of Part 4's story.

The eight information bits are what a decoder needs before it can interpret
a payload. The **colour code** is DMR's system discriminator
([reference]({{ '/reference/color-code/' | relative_url }})); on an IPSC
profile the operator can pin one via `dmr_color_code`, which `IngestBurst`
enforces first. The **data type** is the router — `DTVoiceLCHeader` 0x1,
`DTTerminatorWithLC` 0x2, `DTCSBK` 0x3, `DTIdle` 0x9
([slot-type reference]({{ '/reference/dmr-slot-type/' | relative_url }})) —
and `IngestBurst` switches on it to `handleVoiceHeader`, `handleTerminator`,
`handleCSBK` or `handleIdle`, learning *what kind* of burst arrived even
when the payload is too damaged to use — the ISCH's role in
[P25 Phase 2]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }}).

## Locking the polarity

Tier III offers every burst at both candidates and lets FEC sort it out.
Tier II learned to do better, because of a bug Part 4 tells in full: until
the polarity is known, a **voice** burst A at the untried polarity looks
like a **data** burst, so its AMBE bits get parsed as a slot type. The fix
is two gates:

```go
// internal/radio/dmr/tier2/process.go (shape)
for _, k := range dmr.CandidatePolarities {
    if c.polarity >= 0 && uint8(c.polarity) != k { continue } // locked
    if !dmr.SyncIsDataAtPolarity(m.Pattern, k) { continue }  // data syncs only
    var b dmr.Burst /* … copy 132 dibits from the buffer … */
    dmr.RotateBurstDibits(&b, k)
    slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
    if err != nil { continue }
    c.burstValid = false
    c.IngestBurst(&b, slot)
    if c.burstValid && c.polarity < 0 {
        c.polarity = int(k)
        c.log.Debug("dmr/tier2: discriminator polarity fixed by first FEC-valid burst",
            "polarity", k, "sync", m.Pattern.Name)
    }
}
```

`SyncIsDataAtPolarity(p, k)` is the per-polarity form of "is this a data
sync": at identity it is `IsDataSync(p)`; at the flip it is `IsVoiceSync(p)`,
because a voice-sync match on an inverted stream *is* a data burst. So a
BS-Voice match is offered only at k = 2 and a BS-Data match only at k = 0 —
the parser never sees AMBE bits as a slot type. `burstValid` is set by the
FEC-validated paths (header BPTC + RS, CSBK CRC, terminator LC, the Idle
pattern), and the first time it comes back true the polarity is fixed:

```
DBG dmr/tier2: discriminator polarity fixed by first FEC-valid burst polarity=2 sync=BS-Voice
```

That line means the dongle is inverted — the cue to set `iq_invert`. The
lock is **never dropped**: `ResyncReset`, the reset
Part 10's deaf-heal drives, clears the dibit buffer and pending matches but
keeps the polarity, because inversion is a property of the hardware, not of
the stream position. `TestProcess_DecodesPolarityFlippedVoiceLCHeader` is the #264
regression: a Voice LC Header stream rotated by +2 must still yield
`cc.locked` and a grant.

### How the polarity twin shaped the Go code

- **The flip is one constant, used in both directions.** `PolarityFlip` is
  self-inverse, so `RotateBurstDibits` both models an inverted front end in
  tests and undoes one in production.
- **Data-ness is a function of (sync, polarity), never of sync alone.**
  `SyncIsDataAtPolarity` replaced a bare `IsDataSync` check once the forged
  terminator showed why.
- **Evidence flows back up through one flag.** `burstValid` is the only
  channel from the FEC-validated handlers to the polarity decision; a new
  validated burst type (Idle, Part 7) joins by setting one boolean.
- **State that describes hardware survives resets.** `polarity` is excluded
  from `ResyncReset`, and its doc comment says why — the
  [issue-tracker lesson]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  that a fix needs its reasoning next to it.

## Where this goes next

Bursts are punctuation; the next question is *rhythm*.
[Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})
follows a voice superframe across six bursts and meets the second twin: a
repeater interleaves two slots so a call's bursts sit 264 or 288 dibits
apart, a simplex handheld leaves the other slot empty at the same cadence —
and a decoder that assumed 132 sliced the gaps as voice.

## FAQ

**What does a DMR sync word do?**
It marks where a burst sits in the dibit stream and which role it plays —
base station or mobile, data or voice, direct-mode slot 1 or 2. GopherTrunk's
`SyncDetector` reports the best of the nine ETSI words within two dibit
errors; the match anchors a 132-dibit burst 77 dibits back and 54 forward.

**Why can't the sync word tell GopherTrunk the stream's polarity?**
Because the nine words are closed under the discriminator-polarity flip:
adding 2 mod 4 to every dibit turns each data sync into its voice twin and
back. An inverted front end therefore produces perfectly valid sync matches
— of the wrong type. The decoder tries both polarities and lets the
slot-type, BPTC and CRC codes reject the wrong one.

**What is the DMR slot type?**
A 20-bit field split around the sync of every data burst: 4 bits of colour
code, 4 bits of data type, and 12 parity bits of a (20,8,7) code that
corrects three errors. It tells the decoder which system a burst belongs to
and whether it is a voice header, terminator, CSBK, idle or data block —
before the payload is touched.

**How does GopherTrunk handle an RTL-SDR Blog V4 that inverts the spectrum?**
Automatically, per burst: the Tier II and III adapters decode each burst at
both polarities until FEC validates one, and Tier II then locks that polarity
for the life of the stream. An operator can also set `iq_invert: true` on the
device to conjugate the raw IQ up front.

**Do voice bursts have a slot type?**
No. Only data bursts carry the two 10-bit slot-type fields; a voice burst
uses those positions for AMBE+2 speech bits, and bursts B–F carry embedded
signalling where burst A carries its sync. That is why GopherTrunk parses a
slot type only from data-sync bursts at the stream's polarity.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The 4FSK Carrier & the Tier Family]({{ '/blog/deep-dives/dmr-end-to-end-01-4fsk-carrier/' | relative_url }})
· Next →
[Part 3: Two Slots, One Carrier — Repeater vs Simplex Cadence]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})
