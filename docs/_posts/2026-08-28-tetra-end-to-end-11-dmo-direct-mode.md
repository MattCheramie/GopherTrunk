---
title: "TETRA End to End, Part 11: DMO I — Direct Mode & the DSB/DNB Geometry"
description: "TETRA without a network: how Direct Mode reuses the trunked physical layer with different burst layouts, how GopherTrunk derived the DSB and DNB block geometry from the spec's bit tables, and why one pair of offsets was confirmed by a sharp CRC optimum rather than trusted from another project."
category: deep-dives
keywords: tetra dmo, direct mode operation, dsb dnb burst geometry, en 300 396-2, dm-sync pdu, sch/s decode, osmo-tetra-dmo, tetra burst slicing, gophertrunk tetra
tags: [tetra-end-to-end, tetra, dmo, protocol, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 11
---

*Part 11 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
hardened the trunked-mode control channel and closed the TMO arc. Now the
detour the series has been building toward: an operator pointed GopherTrunk at
438.9 MHz, where two handheld radios were talking **directly to each other** —
no repeater, no site, no control channel. That is TETRA DMO, Direct Mode
Operation, and almost nothing in the trunked ingestion path applies to it. Over
the next three parts we take DMO from "an undocumented-in-practice mode read
out of the spec" to a production daemon pipeline — including the wrong verdict
we published along the way and had to retract.*

> **TL;DR:** DMO (ETSI **EN 300 396-2**) reuses the TMO physical layer this
> series already built — same π/4-DQPSK at 18 ksym/s, same 25 kHz channels and
> 255-symbol slots, same training sequences, same scrambler polynomial — but
> with its **own burst field layouts**: the **DSB** (sync burst: frequency
> correction + SCH/S + sync train + BKN2) and **DNB** (normal burst: BKN1 +
> normal train + BKN2). `internal/radio/tetra/dmo.go` slices them by geometry
> derived from the spec's Tables 15/16, and the DNB offsets **−108/+11**
> (`dmDNBBKN1Start`/`dmDNBBKN2Start`) were *confirmed by a sharp TCH/S-CRC
> optimum* on a real capture — osmo-tetra-dmo's TMO-copied −115/+19 measures
> worse. The DM-SYNC PDU reuses the TMO SYNC-PDU layout, so **`ParseSyncPDU`**
> decodes it unchanged; with the receiver's blind CMA equalizer, CRC-valid
> SCH/S on the first capture went **6 → 64**.

**Key takeaways**

- **DMO is the same radio with different framing.** Modulation, slot length,
  training sequences, scrambler polynomial — all shared with TMO. Only the
  burst field layout and the protocol on top differ, so the receiver, the
  correlators, and the channel-coding chains are all reused.
- **Derive geometry from the spec, then confirm it on air.** The block offsets
  come from halving EN 300 396-2's bit tables into dibits — and the −108/+11
  DNB answer was validated by scanning offsets against real CRC yield, where
  the correct geometry shows a sharp optimum.
- **The sync path is TMO in a trench coat.** A DSB's SCH/S is scrambled with
  colour 0 and coded exactly like TMO's BSCH, and the DM-SYNC PDU matches the
  TMO SYNC-PDU field layout — `DecodeBSCH` and `ParseSyncPDU` both just work.
- **Honest scope, stated in the source.** `dmo.go`'s package comment says
  plainly what is and isn't validated — the #764/#771 discipline applied to a
  brand-new mode.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Burst geometry | DSB/DNB block offsets in dibits, from Tables 15/16 | `internal/radio/tetra/dmo.go` (`dmDNBBKN1Start` = −108, `dmDNBBKN2Start` = +11) |
| Burst extraction | correlate training sequences, slice blocks | `dmo.go` (`ExtractDMBursts`, `ExtractDMBurstsSoft`) |
| Burst record | kind, lead index, rotation, hard + soft blocks | `dmo.go` (`DMBurst`) |
| SCH/S decode | DSB block 1 → 60 type-1 bits, colour-0 scramble | `internal/radio/tetra/dmo_decode.go` (`DecodeDMSCHS` → `DecodeBSCH`) |
| SCH/H decode | DSB block 2 → 124 type-1 bits | `dmo_decode.go` (`DecodeDMSCHH` → `DecodeSCHHD`) |
| SYNC PDU | frame/slot numbering, MNI, colour — TMO layout | `internal/radio/tetra/sync_pdu.go` (`ParseSyncPDU`) |
| Trained equalizer | per-burst LMS on rotation-0 DNBs (opt-in) | `internal/radio/tetra/dmo_equalizer.go` (`ExtractDMBurstsEqualized`) |
| Replay harness | skip-guarded capture A/B | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`TestTETRADMOReplay`) |

## In this post

- **What direct mode is** — and why the trunked ingestion path can't touch it.
- **Shared physical layer, different bursts** — what carries over from TMO.
- **Deriving the geometry** — bit tables to dibit offsets, and the −108/+11 proof.
- **Locking onto a DSB** — SCH/S, the DM-SYNC PDU, and the first capture's numbers.
- **What was honestly not yet known** — the scope statement that paid off.

## What direct mode is

Everything in this series so far assumed infrastructure: a base station
transmitting a continuous downlink, a control channel to camp, grants to
follow. DMO deletes all of it. Two (or more) radios share one 25 kHz channel
directly: a transmitting radio keys up, sends a **Direct Mode Synchronisation
Burst (DSB)** so listeners can acquire, then a train of **Direct Mode Normal
Bursts (DNB)** carrying the call — and when the PTT releases, the channel is
*silent*. No carrier, no heartbeat, nothing to stay locked to.

That inverts the receiver's job. The TMO `ControlChannel` state machine
expects a persistent carrier and slices the TMO burst geometry; pointed at a
DMO channel it produces the operator-visible symptom we'll dissect in Part 13
(`sch_pdus=0`, endless `sync gap` / `dsp resync`). So DMO gets its own layer:
stateless extractors over the demodulated dibit stream (`ExtractDMBursts` and
its soft/equalized variants), designed offline-first so every decision could be
validated against captures before any daemon wiring existed.

## Shared physical layer, different bursts

The good news, verified in `dmo_test.go` against EN 300 396-2 §9.4.3.3
equations 63/64/66: DMO reuses the TMO physical layer nearly wholesale. Same
π/4-DQPSK at 18 ksym/s ([Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})),
same 255-symbol / 14.167 ms timeslots and 4-slot frame / 18-frame multiframe
([Part 2]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})),
the **same** normal and synchronisation training sequences, and the same 32-tap
scrambler polynomial with colour 0 for the DSB's signalling — exactly like
TMO's BSCH ([Part 4]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})).
Even the channel coding turned out to be no new code family at all: SCH/S is
bit-for-bit the TMO BSCH chain, SCH/H is TMO's SCH/HD, SCH/F and TCH/S likewise
([Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})'s
machinery, reused whole).

What differs is the **field layout inside the burst** — where the blocks sit
relative to the training sequence — and that is the entire content of
`dmo.go`'s geometry section.

## Deriving the geometry

EN 300 396-2 specifies burst contents as bit-number tables. GopherTrunk works
in dibits (one per π/4-DQPSK symbol), so the derivation is mechanical — halve
the bit numbers — and recorded verbatim in the source so the next reader can
re-check it against the spec:

```go
// internal/radio/tetra/dmo.go (shape) — geometry, relative to the training
// sequence lead dibit L, derived by halving EN 300 396-2 Tables 15/16:
//   DSB: freq-corr dibits 7..46 (40) · SCH/S 47..106 (60) ·
//        sync train 107..125 (19, lead L=107) · BKN2 126..233 (108)
//   DNB: BKN1 7..114 (108) · normal train 115..125 (11, lead L=115) ·
//        BKN2 126..233 (108)
const (
    dmDSBFreqCorrStart = -100 // L-100 .. L-60
    dmDSBBKN1Start     = -60  // SCH/S, 60 dibits
    dmDSBBKN2Start     = 19
    dmDNBBKN1Start     = -108 // L-108 .. L
    dmDNBBKN2Start     = 11   // L+11 .. L+119
)
```

The DNB pair deserves the emphasis it gets, because there was a competing
answer in the wild: osmo-tetra-dmo slices DNBs at **−115/+19** — the TMO NCDB
offsets from [Part 5]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }}),
copied across. The DMO tables say −108/+11 (a DNB has no AACH halves flanking
its midamble). Rather than trust either reading, the offsets were confirmed
empirically: slicing a real capture's DNBs across candidate offsets, the
TCH/S-CRC yield shows a **sharp optimum exactly at −108/+11**, and the
TMO-copied geometry is measurably worse. A CRC-protected payload is a wonderful
instrument — the correct geometry is not marginally better, it is a spike.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Field layouts of the two DMO bursts drawn as horizontal bars in dibits relative to the training-sequence lead L. The DSB has a 40-dibit frequency-correction field, a 60-dibit SCH/S block, a 19-dibit synchronisation training sequence, and a 108-dibit BKN2 block. The DNB has a 108-dibit BKN1 block starting at L minus 108, an 11-dibit normal training sequence, and a 108-dibit BKN2 block starting at L plus 11. The training sequences are highlighted as the anchors everything is measured from.">
  <text x="10" y="40" fill="currentColor" font-size="11">DSB</text>
  <rect x="60" y="26" width="88" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="104" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="9">freq corr · 40</text>
  <rect x="148" y="26" width="132" height="26" fill="none" stroke="currentColor"/>
  <text x="214" y="43" text-anchor="middle" fill="currentColor" font-size="9">SCH/S (BKN1) · 60</text>
  <rect x="280" y="26" width="44" height="26" fill="var(--accent)" opacity="0.85"/>
  <text x="302" y="43" text-anchor="middle" fill="currentColor" font-size="8">sync·19</text>
  <rect x="324" y="26" width="238" height="26" fill="none" stroke="currentColor"/>
  <text x="443" y="43" text-anchor="middle" fill="currentColor" font-size="9">BKN2 · 108</text>
  <line x1="280" y1="58" x2="280" y2="66" stroke="var(--accent)"/>
  <text x="280" y="78" text-anchor="middle" fill="var(--accent)" font-size="9">L (sync train lead)</text>
  <text x="10" y="130" fill="currentColor" font-size="11">DNB</text>
  <rect x="60" y="116" width="238" height="26" fill="none" stroke="currentColor"/>
  <text x="179" y="133" text-anchor="middle" fill="currentColor" font-size="9">BKN1 · 108 — starts at L−108</text>
  <rect x="298" y="116" width="26" height="26" fill="var(--accent)" opacity="0.85"/>
  <text x="311" y="133" text-anchor="middle" fill="currentColor" font-size="8">11</text>
  <rect x="324" y="116" width="238" height="26" fill="none" stroke="currentColor"/>
  <text x="443" y="133" text-anchor="middle" fill="currentColor" font-size="9">BKN2 · 108 — starts at L+11</text>
  <line x1="298" y1="148" x2="298" y2="156" stroke="var(--accent)"/>
  <text x="298" y="168" text-anchor="middle" fill="var(--accent)" font-size="9">L (normal train lead)</text>
  <text x="340" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">−108/+11, from Tables 15/16 — confirmed by a sharp TCH/S-CRC optimum; the TMO-copied −115/+19 measures worse</text>
</svg>
<figcaption>The two DMO bursts in dibits relative to the training-sequence lead: the DSB carries frequency correction and SCH/S ahead of its sync train; the DNB is two 108-dibit blocks around an 11-dibit midamble at −108/+11.</figcaption>
</figure>

## Locking onto a DSB

Detection reuses the TMO trick: correlate each training sequence under all four
residual constellation rotations, record which rotation matched in
`DMBurst.Rotation`, and de-rotate the sliced blocks by `(4−Rotation)&3` before
decoding. From there the DSB's signalling path is TMO in a trench coat:

```go
// internal/radio/tetra/dmo_decode.go (shape) — DecodeDMSCHS
// Per EN 300 396-2 §8.2.5.2 a DSB's SCH/S is scrambled with colour code 0,
// so this is the DMO analog of the TMO BSCH decode.
func DecodeDMSCHS(b DMBurst) (type1 []byte, crcOK bool) {
    if b.Kind != DMBurstSync || len(b.SCHS) != dmSCHSDibits {
        return nil, false
    }
    return DecodeBSCH(dmDerotatedBits(b.SCHS, b.Rotation))
}
```

And the 60 type-1 bits it yields are a **DM-SYNC PDU** that reuses the TMO
SYNC-PDU field layout — so `ParseSyncPDU`, written for trunked mode, decodes it
unchanged: colour 0, timeslot and frame numbers, MNI with MCC/MNC = 0 for a
radio-to-radio call. On the operator's first capture
(`tetra_dmo_test2_20sec_cs16_144k`, 438.9 MHz, cs16 at 144 kHz), replayed
through `TestTETRADMOReplay`, the frame counter advances monotonically across
bursts — a genuine lock, not a correlator coincidence. The yield numbers also
re-ran Part 9's lesson in miniature: with the receiver's blind CMA equalizer
(the harness default), CRC-valid SCH/S went **6 → 64** — `DecodeDMSCHS` at
64/68 and `DecodeDMSCHH` at 62/68 — the same ISI-inversion lever, working on a
mode it had never seen. A per-burst trained LMS variant also exists for DNBs
(`ExtractDMBurstsEqualized`, rotation-0 bursts only, since a frozen filter
cannot invert a residual per-symbol phase ramp), pinned at 12% → 0% synthetic
payload error in `dmo_equalizer_test.go` and staged behind `GT_TETRA_DMO_LMS`.

## What was honestly not yet known

`dmo.go`'s package comment, at the point this part describes, said the quiet
part in writing: burst detection and block slicing only; the DM call-control
protocol (EN 300 396-3 — who is calling whom) not wired; and *nothing yet
validated against a real DMO capture* for the voice path. The signalling
decoded beautifully. The TCH/S speech in those same captures sat stubbornly at
the CRC chance floor — roughly 1 in 256 bursts passing, exactly what random
bits produce. Every synthetic round-trip passed. And the first conclusion drawn
from that split was wrong in a way this series has warned about since
[Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }}):
a green synthetic is not proof, and a self-consistent bug hides on both sides
of a round-trip.

## Where this goes next

The chance floor got read as air-interface encryption — a plausible verdict for
a professional radio system, published in the issue, and **wrong**.
[Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
tells the #1003 arc honestly: the reporter's codeplug proving the radios were
clear, the one-line descramble skip that Part 4 foreshadowed, the failing-first
regression that finally caught it — and the second capture where the voice
descrambles with a colour code the signalling never advertised.

## FAQ

**Is DMO just TETRA's version of talk-around/simplex?**
Functionally yes — radios talking directly on a channel without
infrastructure, common for fireground and coverage-hole use. But it is a fully
specified TETRA mode (EN 300 396) with its own burst types, sync procedure,
and call-control protocol, not merely a carrier with no repeater.

**Why couldn't the existing TMO decoder be adapted instead of writing new extractors?**
The TMO `ControlChannel` assumes a continuous downlink and slices TMO burst
geometry at fixed positions in a persistent stream. DMO is bursty and silent
between transmissions, with different block offsets. Stateless per-burst
extractors fit that shape and — critically — could be validated offline against
captures before any live wiring existed.

**How was −108/+11 confirmed if another open-source project uses −115/+19?**
By measurement, not authority. Slicing a real capture's DNBs across candidate
offsets and counting CRC-valid TCH/S shows a sharp optimum at −108/+11 —
which also matches a fresh reading of Tables 15/16. The lesson generalizes:
when two sources disagree, a CRC-protected payload is the tiebreaker.

**Why does the DSB use colour code 0 for its signalling?**
By spec (§8.2.5.2): the sync burst must be decodable by a radio that knows
nothing yet, so its SCH/S and SCH/H are scrambled with the fixed colour-0 seed
— the same bootstrap logic as TMO's BSCH. Remember from Part 4 that colour 0 is
*not* a no-op — a fact about to become the whole plot of Part 12.

**Can I try this on my own DMO radios?**
Yes — record cs16 IQ of a transmission and run
`GT_TETRA_DMO_IQ=<file> GT_TETRA_DMO_RATE=144000 go test ./cmd/gophertrunk
-run TestTETRADMOReplay -v`. Part 14 catalogs the whole harness, including the
colour and equalizer knobs the next two parts introduce.

## Series navigation

**Part 11 of 14** · ←
[Part 10: The Control Channel Under Stress — Sync Loss & the CC Equalizer]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
· Next →
[Part 12: DMO II — The 'Encrypted' Verdict That Was a Descramble Skip]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
