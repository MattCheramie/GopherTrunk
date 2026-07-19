---
title: "Protocol Decoders, Part 5: DMR Tier 2/3 — Bursts, EMB & FLC"
description: How GopherTrunk decodes DMR's two-slot TDMA — the 132-dibit burst, the sync words closed under polarity flip, the slot-type Hamming code, embedded signalling and Full Link Control, and the reverse channel.
category: deep-dives
keywords: dmr decoder, dmr tier 2 tier 3, dmr burst structure, dmr sync patterns, dmr emb, full link control flc, bptc 196 96, dmr reverse channel, gophertrunk dmr
tags: [protocol-decoders, dmr, tdma, fec, framing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 5
---

*Part 5 of **Protocol Decoders**. We leave P25 for DMR — ETSI's two-slot TDMA
family, running Tier II (conventional) and Tier III (trunked) over the same wire
format. The burst layout, the sync words, the embedded signalling, and the Full
Link Control are all shared; the tier only changes which state machine reads them.*

> **TL;DR:** A DMR burst is 132 dibits (264 bits, 27.5 ms): two 98-bit payload
> halves around a central 48-bit sync/embedded field, with two 10-bit **slot-type**
> fields flanking the sync. Control/data bursts concatenate the halves into one
> **BPTC(196,96)** codeword; voice bursts carry AMBE and **embedded signalling**
> instead. The slot-type is a (20,8) Hamming code carrying color code + data type;
> voice bursts B–E reassemble a 128-bit embedded **Full Link Control**. GopherTrunk
> shares this whole layer across Tier II and Tier III — the tier is just a different
> reader on the same bursts.

**Key takeaways**

- The **9 DMR sync words** are closed under discriminator-polarity flip, so a sync
  match alone can't resolve spectral inversion — the decoder tries both polarities
  and lets FEC arbitrate.
- The **slot-type** field is the DMR analogue of P25's ISCH: color code + data
  type under a (20,8) Hamming code, read twice per burst (before + after sync).
- **Full Link Control** carries the call's group/source, reachable two ways — a
  Voice LC Header burst, or reassembled across the embedded fragments of voice
  bursts B–E.
- The **reverse channel** rides the embedded field of ordinary bursts; GopherTrunk
  decodes the downlink-observable carrier and is honest about the unverified FEC.
- Tier II vs Tier III is a **state-machine choice**, not a wire-format one — the
  same receiver feeds both.

## Cheat sheet

| Field | Size | Coding | GopherTrunk file |
|---|---|---|---|
| Burst | 132 dibits / 264 bits | — | `dmr/burst.go` |
| Sync | 48 bits (24 dibits) | 9 patterns, tol=4 | `dmr/sync.go` |
| Slot type | 20 bits (2×10) | Hamming(20,8), t=3 | `dmr/slottype.go` |
| Control payload | 196 bits | BPTC(196,96) | `dmr/burst.go` |
| EMB | 16 bits | QR(16,7) | `dmr/emb.go` |
| Embedded LC | 128 → 72 bits | BPTC(128,72) + CRC | `dmr/emb.go` |
| FLC | 72 bits (9 octets) | + RS(12,9) trailer | `dmr/flc.go` |
| Reverse channel | 11 info + 21 parity | (32,11) | `dmr/rc.go` |

## In this post

- **The burst** — one picture of the 132-dibit layout.
- **Sync** — nine words and the polarity twin problem.
- **Slot type** — color code + data type, the routing field.
- **EMB & Full Link Control** — call identity, two carriages.
- **The reverse channel** — in-call signalling, honestly incomplete.

## The burst

Everything in DMR hangs off one structure. A burst is 132 dibits split into two
49-dibit payload halves around a central sync region, with 5-dibit slot-type fields
tucked on either side of the sync:

```go
// internal/radio/dmr/burst.go (shape)
//   dibits   0..48   info half 1  (98 bits)
//   dibits  49..53   slot type before sync  (10 bits)
//   dibits  54..77   sync / embedded signalling  (48 bits)
//   dibits  78..82   slot type after sync  (10 bits)
//   dibits  83..131  info half 2  (98 bits)
const (
    BurstDibits     = 132
    HalfPayloadBits = 98
    BPTCPayloadBits = 196
)
```

For a control/data burst, the two 98-bit halves concatenate into a single 196-bit
**BPTC(196,96)** codeword — the product code that protects CSBKs and Link Control.
`PayloadBits()` does exactly that concatenation, handing the result straight to
`framing.DecodeBPTC196_96`. Voice bursts use a different split (108 + 48 + 108,
three 72-bit AMBE frames, no slot-type fields) and put embedded signalling in the
sync region instead. One structure, two readings.

<figure class="lab-figure">
<svg viewBox="0 0 680 132" width="680" height="132" role="img" aria-label="A DMR burst: a 98-bit first payload half, a 10-bit slot-type field, a 48-bit sync or embedded-signalling region, a 10-bit slot-type field, and a 98-bit second payload half, spanning 132 dibits">
  <rect x="10" y="40" width="180" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="100" y="60" text-anchor="middle" fill="var(--accent)" font-size="11">info half 1</text>
  <text x="100" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">98 bits</text>
  <rect x="192" y="40" width="52" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="218" y="58" text-anchor="middle" fill="currentColor" font-size="9">slot</text>
  <text x="218" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">type</text>
  <rect x="246" y="40" width="150" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="321" y="58" text-anchor="middle" fill="currentColor" font-size="11">sync / EMB</text>
  <text x="321" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">48 bits</text>
  <rect x="398" y="40" width="52" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="424" y="58" text-anchor="middle" fill="currentColor" font-size="9">slot</text>
  <text x="424" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">type</text>
  <rect x="452" y="40" width="180" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="542" y="60" text-anchor="middle" fill="var(--accent)" font-size="11">info half 2</text>
  <text x="542" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">98 bits</text>
  <text x="321" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="10">control bursts: halves 1+2 → one BPTC(196,96) · voice bursts: embedded LC in the sync region</text>
</svg>
<figcaption>The DMR burst. The two info halves reunite into a BPTC codeword for control traffic; on voice bursts the sync region carries embedded signalling instead.</figcaption>
</figure>

## Sync: nine words, and the polarity twin

DMR defines nine 48-bit sync patterns — base-station and mobile voice/data, the
reverse-channel sync, and the direct-mode variants. The detector slides a 24-dibit
window and picks the best-matching pattern within tolerance:

```go
// internal/radio/dmr/sync.go (shape)
var (
    BSVoice = mkSync("BS-Voice", 0x755FD7DF75F7)
    BSData  = mkSync("BS-Data",  0xDFF57D75DF5D)
    MSVoice = mkSync("MS-Voice", 0x7F7D5DD57DFD)
    // …MSData, MSRC, DM-Voice/Data TS1/TS2 — nine total
)
```

Here's the subtlety that shows up across every C4FM protocol in this series. The
nine sync words are **closed under the discriminator-polarity flip** — adding 2 mod
4 to every dibit (what a spectrum-inverted or I/Q-swapped front end does) turns a
clean data sync into a byte-identical *voice* sync, and vice versa. So the sync
detector fires happily on an inverted stream; it just reports the flipped twin. The
sync match cannot tell you the polarity. The decoder resolves it downstream by
attempting the burst at *both* polarities (`CandidatePolarities = {0, 2}`) and
letting the slot-type Hamming + BPTC + CSBK CRC drop the wrong one with no state
change. FEC as the arbiter, again — the same principle Part 2 named for P25's
rotation search, here applied to a whole burst.

## Slot type: DMR's routing field

Each control/data burst carries its slot type twice — 10 bits before the sync, 10
after — concatenated into a 20-bit codeword. Eight information bits (4-bit color
code + 4-bit data type) sit under a (20,8) shortened Hamming code correcting up to
three errors:

```go
// internal/radio/dmr/slottype.go (shape)
func ParseSlotType(bits []byte) (SlotType, int, error) {
    var cw uint32
    for i := 0; i < 20; i++ {
        if bits[i]&1 != 0 { cw |= uint32(1) << uint(19-i) }
    }
    data, errs := framing.HammingDecode20_8(cw) // errs < 0 ⇒ uncorrectable
    if errs < 0 { return SlotType{}, -1, ErrSlotTypeUncorrectable }
    return SlotType{ColorCode: (data >> 4) & 0x0F, DataType: DataType(data & 0x0F)}, errs, nil
}
```

The `DataType` is the branch point: `VoiceLCHeader` (call setup), `TerminatorWithLC`
(call teardown), `CSBK` (control signalling block — the trunking messages), the
data-header and rate types, and idle. It plays the exact role P25 Phase 2's
`SlotType` played in Part 4: a small, FEC-protected field that routes the burst
before anyone parses the payload. The color code is DMR's system discriminator, the
analogue of the P25 NAC.

## EMB and Full Link Control: call identity, two ways

The 72-bit **Full Link Control** word is where a DMR call names itself — group or
subscriber destination, source subscriber, service options:

```go
// internal/radio/dmr/flc.go (shape)
type FLC struct {
    PF             bool
    FLCO           FLCO   // GroupVoiceUser 0x00, UnitToUnit 0x03, TalkerAlias 0x04…
    FID            uint8
    ServiceOptions uint8
    DstAddr        uint32 // 24-bit group or subscriber
    SrcAddr        uint32 // 24-bit subscriber
}
```

That FLC reaches the receiver by two independent routes, and DMR provides both for
redundancy. The direct route is a **Voice LC Header** burst (data type 0x1) whose
BPTC(196,96) payload is the FLC plus an RS(12,9) trailer. The clever route is
**embedded signalling**: voice bursts B through E each carry a 32-bit fragment in
their sync region, framed by a 16-bit **EMB** header (color code, privacy
indicator, and a 2-bit LCSS marking the fragment's position). The four fragments
reassemble into a 128-bit embedded LC:

```go
// internal/radio/dmr/emb.go (shape)
func ReassembleEmbeddedLC(frags [4][]byte) (FLC, bool) {
    channel := concat(frags)                       // 4 × 32 = 128 bits
    lcBits, corrected := framing.DecodeEmbeddedLC(channel) // BPTC(128,72) + CRC
    if corrected < 0 { return FLC{}, false }
    return ParseFLC(pack(lcBits))
}
```

The LCSS values (`First`, `Cont`, `Cont`, `Last`) tell the reassembler how to
order the fragments; a bad fragment fails the embedded-LC BPTC + CRC and the whole
word is rejected rather than surfaced as a garbled call. Note `FLCOTalkerAlias`
(0x04) in the opcode list — DMR carries a talker alias too, and while it's a
different wire format from the Motorola P25 alias, it's the same *kind* of field:
the display name of the transmitting radio, riding link control. Our cross-series
mystery has cousins everywhere.

### A note on honest FEC — the reverse channel

The DMR **reverse channel** is in-call signalling distinct from the control-channel
CSBKs. It rides the 32-bit embedded fragment when the EMB marks a single, non-LC
fragment (`LCSS == LCSSSingle`), as 11 information bits under 21 parity bits:

```go
// internal/radio/dmr/rc.go (shape)
type ReverseChannel struct {
    Info   uint16 // 11-bit RC information field
    Parity uint32 // 21-bit FEC parity — preserved but not yet applied
}
```

This is worth calling out because the code is candid about its own limits.
GopherTrunk has no real off-air RC capture, and no open reference decoder
implements the (32,11) RC FEC, so **the parity is preserved but not used to correct
or validate the information bits**. The integrity gate is the caller's EMB framing
check plus a null/idle rejection. That's the same "capture-pending" posture the EMB's
own QR(16,7) and the MBC last-block CRC carry — a deliberate choice to decode what's
verifiable and flag what isn't, rather than fake a correction. When a real RC
capture lands, the parity gets promoted to a true FEC gate without changing a single
caller. It's the discipline that keeps the decoder trustworthy: **don't claim
correction you can't test.** (The same discipline is exactly why the Motorola alias
cipher stays flagged unreliable — a theme Parts 10–11 pay off.)

### How the tier is just a reader

Everything above is shared. Tier II (conventional, per-repeater) and Tier III
(trunked) run the *same* receiver and the *same* burst/slot-type/BPTC decode — the
difference is the state machine on top. Tier III reads CSBKs off a dedicated control
channel and publishes grants; Tier II has no control channel, so its
`ConventionalChannel` emits a synthetic grant on every Voice LC Header burst
(deduped per call, cleared on the terminator). Both hand the engine and recorder the
same `trunking.Grant` shape, so nothing downstream needs to know whether the system
was trunked or conventional. The wire format is one; the tier is a policy.

## Where this goes next

[Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
moves to the narrowband FDMA family — NXDN and dPMR at 6.25 kHz — and contrasts
their per-channel structure with the TDMA modes we've now seen in P25 Phase 2 and
DMR. The grants DMR produces feed the
[trunking engine]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }}),
and the AMBE voice frames feed the
[Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }}) series.

## FAQ

**What's the difference between DMR Tier II and Tier III?**
Tier II is conventional (no control channel; each repeater is its own channel);
Tier III is trunked (a dedicated control channel grants traffic onto other
channels). They share the entire physical/framing layer in GopherTrunk — only the
control state machine differs.

**Why does DMR need to try two polarities per burst?**
Because the nine sync words are closed under the discriminator-polarity flip: a
spectrum-inverted stream produces a valid-looking sync of the flipped type. The sync
match can't disambiguate, so the decoder runs the burst at both polarities and lets
the slot-type Hamming and BPTC/CRC reject the wrong one.

**What is embedded signalling in DMR?**
Signalling carried in the sync region of voice bursts. Bursts B–E each hold a 32-bit
fragment framed by a 16-bit EMB header; the four fragments reassemble into a 128-bit
embedded Link Control word carrying the call's group and source.

**Why doesn't GopherTrunk use the reverse-channel parity?**
There's no verified off-air RC capture and no reference decoder for the (32,11) RC
FEC, so the parity is preserved but not applied — decoding only what can be
validated. A future capture promotes it to a real FEC gate transparently.

**What carries the DMR call's talkgroup and source?**
The Full Link Control word — via a Voice LC Header burst or reassembled from voice
bursts B–E. Its FLCO selects group voice, unit-to-unit, talker alias, GPS, or
terminator.

## Series navigation

**Part 5 of 12** · ← [Part 4: P25 Phase 2 TDMA — The MAC Layer]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }}) · Next →
[Part 6: NXDN & dPMR — The Narrowband FDMA Family]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
