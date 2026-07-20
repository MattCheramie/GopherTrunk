---
title: "Protocol Decoders, Part 6: NXDN & dPMR — The Narrowband FDMA Family"
description: How GopherTrunk decodes NXDN and dPMR — the 6.25 kHz FDMA narrowband modes — their 80 ms frames, LICH and CAC signalling, CSBK trunking blocks, and convolutional FEC, and how they contrast with the TDMA modes.
category: deep-dives
keywords: nxdn decoder, dpmr decoder, 6.25 khz fdma, nxdn lich cac, nxdn viterbi fec, dpmr csbk mode 3, rcch vcall, narrowband digital voice, gophertrunk nxdn dpmr
tags: [protocol-decoders, nxdn, dpmr, fdma, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 6
---

*Part 6 of **Protocol Decoders**. After two TDMA families (P25 Phase 2, DMR) we
come to the narrowband FDMA pair — NXDN and dPMR — squeezed into 6.25 kHz. No
timeslots, one call per carrier, and a framing philosophy that trades DMR's
per-burst product codes for repetition and convolutional coding. Same pipeline
shape from Part 1; a different set of dials.*

> **TL;DR:** NXDN and dPMR are 4-level FSK narrowband modes at 6.25 kHz. NXDN runs
> 80 ms frames — Frame Sync, an 8-bit **LICH** (repetition-coded), a **SACCH**, and
> a 144-dibit Info field carrying the **CAC** control channel under a K=5 ½-rate
> convolutional code. dPMR Mode 3 uses three frame-sync words and an 80-bit **CSBK**
> signalling block on its control channel. Both are FDMA, so there's no slot-type
> to route — the whole frame is one channel. GopherTrunk decodes their structured
> surface today; the heaviest FEC/interleave stages are staged, honest deferrals.

**Key takeaways**

- **FDMA, not TDMA**: one call per 6.25 kHz carrier, so there's no ISCH/slot-type
  routing field — the frame *is* the channel.
- NXDN's **LICH** is 8 information bits transmitted twice (1-to-2 repetition) —
  decode by majority vote per bit pair, then parity.
- NXDN's **CAC** carries the RCCH trunking opcodes (VCALL, VCALL_ASSGN…) under a
  selectable Viterbi mode; the spec-correct path is the full deinterleave +
  depuncture + K=5 Viterbi + CRC chain.
- dPMR Mode 3 packs source, destination, and channel into an **80-bit CSBK**; only
  Mode 3 has a control channel to hunt.
- Both packages **ship the structured surface and defer the analogue/FEC pieces**
  as named follow-ups — the same honesty posture as DMR's reverse channel.

## Cheat sheet

| Aspect | NXDN | dPMR (Mode 3) |
|---|---|---|
| Channel | 6.25 kHz FDMA | 6.25 kHz FDMA |
| Symbols | BFSK 4800 / 4-FSK 9600 | 4-FSK, 2400 sym/s |
| Frame | 80 ms, 192 dibits | superframe + CSBK bursts |
| Sync | FSW out `0xC55A` / in `0x3AA5` | FS1 / FS2 / FS3 |
| Control | CAC → RCCH opcodes | 80-bit CSBK |
| Grant coding | K=5 ½-rate Viterbi + CRC | short cyclic + 3/4 conv (deferred) |
| File | `internal/radio/nxdn/` | `internal/radio/dpmr/` |

## In this post

- **FDMA vs TDMA** — what changes when the frame is the channel.
- **The NXDN frame** — FSW, LICH, SACCH, Info.
- **LICH & CAC** — routing and the RCCH control channel.
- **dPMR Mode 3** — three syncs and the 80-bit CSBK.
- **Honest deferrals** — what's decoded and what's staged.

## FDMA changes the routing

The TDMA modes needed a small FEC-protected field — P25 Phase 2's ISCH, DMR's
slot-type — to say *which* of the interleaved things a sub-frame carried, because
voice and signalling shared a carrier. NXDN and dPMR don't have that problem: each
6.25 kHz channel is one thing at a time. A control channel is a control channel; a
traffic channel is a traffic channel. So the routing question shrinks to a single
per-frame flag (NXDN's LICH RF-channel-type bit) rather than a per-sub-frame decode.
The pipeline shape from Part 1 is unchanged — symbols → sync → FEC → PDU → opcode →
event — but the framing is flatter.

<figure class="lab-figure">
<svg viewBox="0 0 680 128" width="680" height="128" role="img" aria-label="An NXDN 80-millisecond frame: an 8-dibit frame sync word, an 8-dibit LICH, a 32-dibit SACCH, and a 144-dibit information field carrying CAC, VCH, UDCH or FACCH">
  <rect x="10" y="38" width="70" height="46" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="45" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">FSW</text>
  <text x="45" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">8 db</text>
  <rect x="82" y="38" width="70" height="46" rx="4" fill="none" stroke="currentColor"/>
  <text x="117" y="58" text-anchor="middle" fill="currentColor" font-size="10">LICH</text>
  <text x="117" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">8 db · 1→2</text>
  <rect x="154" y="38" width="120" height="46" rx="4" fill="none" stroke="currentColor"/>
  <text x="214" y="58" text-anchor="middle" fill="currentColor" font-size="10">SACCH</text>
  <text x="214" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">32 db</text>
  <rect x="276" y="38" width="394" height="46" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="473" y="58" text-anchor="middle" fill="var(--accent)" font-size="10">Info field — 144 dibits</text>
  <text x="473" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CAC (control) / VCH / UDCH / FACCH</text>
  <text x="340" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="10">80 ms · same layout at 4800 (BFSK) and 9600 (4-FSK); only the symbol mapping differs</text>
</svg>
<figcaption>The NXDN frame is one flat structure. The LICH's RF-channel-type bit says whether the Info field is control (CAC) or traffic — the FDMA analogue of a slot-type.</figcaption>
</figure>

## The NXDN frame

NXDN's structural layout is identical at both channel rates — the only difference
between 4800 (BFSK, 1 bit/symbol) and 9600 (4-FSK, 1 dibit/symbol) is the symbol
mapping, not the framing:

```go
// internal/radio/nxdn/frame.go (shape)
const (
    FSWDibits       = 8   // 16 bits
    LICHWireDibits  = 8   // 16 wire bits carrying 8 doubled info bits
    SACCHDibits     = 32  // 64 bits
    InfoFieldDibits = 144 // 288 bits — CAC / VCH / UDCH / FACCH
    FrameDibits     = FSWDibits + LICHWireDibits + SACCHDibits + InfoFieldDibits // 192
    FrameDurationMs = 80
)
```

The frame sync words are direction-specific — `0xC55A` outbound (base→mobile),
`0x3AA5` inbound — so the detector can lock the downlink and reject uplink bursts.
The `SyncDetector` follows the same shape as every other protocol in this series,
sliding an 8-dibit window with a low tolerance.

## LICH and CAC

The **LICH** (Link Information Channel) is 8 information bits, but it's the first
place NXDN's coding philosophy shows: instead of a block code, each bit is simply
transmitted *twice*. Decode is a majority vote per bit-pair (with disagreements
flagged soft), then an even-parity check:

```go
// internal/radio/nxdn/lich.go (shape)
type LICH struct {
    RFCh   RFChannelType       // bit 0: RCCH (control) vs RDCH (traffic)
    FCT    FunctionChannelType // NSACCH / NUDCH / FrameStep
    Option uint8
    // …Direction (outbound/inbound), Parity
}
```

That `RFCh` bit is the routing decision: a control frame's Info field is a **CAC**
(Common Access Channel) carrying trunking signalling. The CAC message is 88
information bits — an 8-bit RCCH opcode, 64-bit payload, 16-bit CRC:

```go
// internal/radio/nxdn/cac.go (shape)
type CACMessage struct {
    Type    RCCHType // VCALL 0x01, VCALL_ASSGN 0x04, DCALL 0x09, SITE_INFO 0x3C…
    Payload [8]byte  // 64 bits
}
```

The RCCH opcodes are NXDN's grant vocabulary — `VCALL`/`VCALL_ASSGN` for voice
calls, `DCALL` variants for data, `SITE_INFO` and `SRV_INFO` for topology. On the
air, those 88 bits are wrapped in a K=5 ½-rate convolutional code, punctured, and
interleaved across the 144-dibit Info field. GopherTrunk exposes this as a
selectable **Viterbi mode**, and the spec-correct path (`ViterbiSpec`) runs the full
NXDN-TS chain — 25×12 deinterleave, 50/350 depuncture, K=5 R=½ Viterbi, 16-bit CRC
verify, tail strip — recovering the 155-bit info block. A simpler mode reads CAC
bits straight off the wire for fixtures where they aren't channel-coded; on real
signals that path fails the CRC and drops the frame, which is the correct failure.

<figure class="lab-figure">
<svg viewBox="0 0 660 150" width="660" height="150" role="img" aria-label="The NXDN CAC receive chain: 300 channel bits pass through a 25 by 12 deinterleaver, a depuncture stage, a constraint-length-5 half-rate Viterbi decoder, and a 16-bit CRC check to recover the 88-bit CAC message">
  <rect x="6" y="54" width="96" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="54" y="74" text-anchor="middle" fill="currentColor" font-size="10">CAC bits</text>
  <text x="54" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">300 ch</text>
  <line x1="102" y1="76" x2="120" y2="76" stroke="currentColor"/><polygon points="120,72 130,76 120,80" fill="currentColor"/>
  <rect x="130" y="54" width="100" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="180" y="74" text-anchor="middle" fill="currentColor" font-size="10">deinterleave</text>
  <text x="180" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">25×12</text>
  <line x1="230" y1="76" x2="248" y2="76" stroke="currentColor"/><polygon points="248,72 258,76 248,80" fill="currentColor"/>
  <rect x="258" y="54" width="96" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="306" y="79" text-anchor="middle" fill="currentColor" font-size="10">depuncture</text>
  <line x1="354" y1="76" x2="372" y2="76" stroke="currentColor"/><polygon points="372,72 382,76 372,80" fill="currentColor"/>
  <rect x="382" y="54" width="110" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="437" y="74" text-anchor="middle" fill="var(--accent)" font-size="10">Viterbi</text>
  <text x="437" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">K=5 · R=½</text>
  <line x1="492" y1="76" x2="510" y2="76" stroke="currentColor"/><polygon points="510,72 520,76 510,80" fill="currentColor"/>
  <rect x="520" y="54" width="132" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="586" y="74" text-anchor="middle" fill="var(--accent)" font-size="10">CRC-16 verify</text>
  <text x="586" y="88" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ 88-bit CAC</text>
  <text x="330" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the ViterbiSpec path; a simpler mode reads pre-coded CAC bits for fixtures</text>
</svg>
<figcaption>NXDN's CAC receive chain. The convolutional decode is the lock gate here, the way the trellis was for a P25 TSBK — the CRC is the final accept.</figcaption>
</figure>

## dPMR Mode 3

dPMR is NXDN's European sibling — the digital successor to analogue PMR446, also
6.25 kHz, also 4-FSK. Of its three operating modes, only **Mode 3** (centralised
trunking) has a dedicated control channel for the engine to hunt; Modes 1 and 2 are
peer-to-peer or managed-direct with nothing to grant. Mode 3 uses three frame-sync
words — FS1 (voice/data superframe start), FS2 (mid-superframe resync), and FS3
(the CSBK signalling burst on the control channel):

```go
// internal/radio/dpmr/sync.go (shape)
const (
    FS1Hex uint64 = 0x57FF5F75F575 // superframe start
    FS2Hex uint64 = 0x5F7F77FD7DFD // mid-superframe sync
    FS3Hex uint64 = 0x7DDFFD5F55D5 // CSBK burst
    SyncDibits = 24
)
```

The control channel's unit is the **CSBK** (Common Signalling Block) — 80 bits
carrying a 5-bit message type, source and destination IDs, service info, and an
opcode-specific 16-bit field (typically the granted channel number):

```go
// internal/radio/dpmr/csbk.go (shape)
type CSBK struct {
    Type        MessageType // VoiceServiceAllocation, IndividualCall, …
    Flags       uint8       // emergency + group flags
    SourceID    uint32      // 24-bit calling subscriber
    DestID      uint32      // 24-bit group or subscriber
    ServiceInfo uint8
    Extra       uint16      // opcode-specific (e.g. channel number)
}
```

`CSBKFromBits` packs 80 wire bits into that struct, the state machine dispatches on
`MessageType`, and a voice-service allocation becomes a `trunking.Grant` with
`Protocol = "dpmr"` — the same event every other decoder in this series emits.

## Honest deferrals — the recurring posture

Both packages ship a clean *structured* surface and are explicit about what's not
yet wired. The dPMR package header lists them plainly: the 4-FSK demodulator +
symbol-clock recovery for the 2400 sym/s air interface, and the interleaver + FEC
over CSBK bits (a short-block cyclic code with rate-3/4 convolutional outer coding)
are named follow-ups; the current parse assumes an upstream caller has corrected
errors. NXDN's CAC channel-coding is present but gated behind the selectable Viterbi
mode precisely because the simplest path only works on non-coded fixtures.

This is the same discipline you saw with DMR's reverse channel in Part 5 and the
Motorola alias cipher back in Part 1: **decode what you can verify, flag what you
can't, and leave the seam obvious.** Shipping the structured surface first lets the
trunking engine consume these protocols end-to-end against fixtures while the
analogue and FEC layers are proven out capture by capture. It's not a shortcut — it's
the same reason GopherTrunk refuses to report the talker alias as reliable until its
cipher is confirmed. A decoder that lies about what it corrected is worse than one
that admits the gap.

## Where this goes next

That closes the first six parts of the series — the shared pipeline, P25 Phase 1 and
2, DMR, and the NXDN/dPMR narrowband pair. The remaining parts (written by other
authors) take on
[TETRA]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }}),
the [EDACS/LTR/MPT-1327]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
legacy families, and
[conventional & wideband]({{ '/blog/deep-dives/protocol-decoders-09-conventional-wideband/' | relative_url }})
decoding — before the two-part alias hunt
([Part 10]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }}),
[Part 11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }}))
finally cracks — or fails to crack — the Motorola field we planted in Part 1, the
same *Mercury* emitter the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series chases. Grants
from every one of these feed the
[trunking engine]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }}).

## FAQ

**How do NXDN and dPMR differ from DMR?**
They're FDMA (one call per 6.25 kHz carrier) rather than TDMA (two slots per
carrier). There's no slot-type/ISCH routing field because voice and signalling never
share a carrier, and the FEC leans on repetition and convolutional coding rather than
DMR's per-burst product codes.

**What is the NXDN LICH?**
An 8-bit Link Information Channel that prefixes each frame. Its bits are transmitted
twice (1-to-2 repetition) and decoded by majority vote; the RF-channel-type bit says
whether the frame is control (RCCH/CAC) or traffic (RDCH).

**What carries NXDN trunking grants?**
The CAC (Common Access Channel) on a control frame, as RCCH messages — `VCALL`,
`VCALL_ASSGN`, `DCALL`, and so on — protected by a K=5 ½-rate convolutional code and
a 16-bit CRC.

**Which dPMR mode does GopherTrunk decode?**
Mode 3, the centralised trunking mode — the only one with a dedicated control channel
to hunt. Its 80-bit CSBK carries the message type, source, destination, and channel;
a voice-service allocation becomes a `trunking.Grant`.

**Why are some FEC stages deferred?**
Because GopherTrunk ships the verifiable structured surface first and leaves the
analogue demod and heavy FEC/interleave as named follow-ups, provable capture by
capture — the same "don't claim correction you can't test" rule the whole series
follows.

## Series navigation

**Part 6 of 12** · ← [Part 5: DMR Tier 2/3 — Bursts, EMB & FLC]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }}) · Next →
[Part 7: TETRA]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }})
