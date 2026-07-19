---
title: "Protocol Decoders, Part 4: P25 Phase 2 TDMA — The MAC Layer"
description: How GopherTrunk decodes P25 Phase 2's two-slot TDMA — the ISCH slot-type field, the 360 ms superframe, the MAC PDU layer, and the compressed source-less grant that forces a source-RID backfill downstream.
category: deep-dives
keywords: p25 phase 2 tdma, p25 mac pdu, isch slot type, p25 superframe, h-dqpsk, compressed grant, source rid backfill, group voice channel user, gophertrunk p25 phase 2
tags: [protocol-decoders, p25, tdma, mac, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 4
---

*Part 4 of **Protocol Decoders**. Phase 1 (Parts 2–3) was FDMA — one call per
carrier. Phase 2 packs two calls onto one carrier with TDMA, and the signalling
moves from standalone TSBKs to a MAC layer riding inside the traffic channel. We
walk the ISCH, the superframe, and the MAC PDU — then hit the problem this design
creates: grants that arrive with no source radio.*

> **TL;DR:** P25 Phase 2 is two-slot TDMA at 6000 sym/s, H-DQPSK. A 360 ms
> **superframe** holds 12 sub-frames; each sub-frame is prefixed by an **ISCH**
> (Golay-protected) that names it as voice or MAC. MAC sub-frames carry **MAC
> PDUs** — the Phase 2 analogue of TSBKs — whose opcode selects a message. Grants
> come in two shapes, and the compressed in-call form (`GroupVoiceChannelUser`
> Abbreviated) carries **no source RID**: the call is granted source-less, and the
> source must be backfilled later — the exact seam the trunking engine's
> source-RID recovery closes.

**Key takeaways**

- The **ISCH** is how a receiver tells a voice sub-frame from a MAC sub-frame
  *without* decoding the payload — a 12-bit field under Golay(24,12,8).
- The **superframe grid** (12 sub-frames, anchored on a sync match) is the whole
  slice geometry, derived from one anchor constant.
- **MAC PDUs** mirror TSBKs: opcode-first, MFID for vendor PDUs, opcode-specific
  payload — and the same grant, status, and identifier messages reappear.
- Phase 2's descrambler needs a **PN44 seed** derived from WACN/System/Color-Code,
  which the Network Status Broadcast supplies.
- **The problem we hit:** compressed grants are source-less, so the source RID is
  recovered from a later in-call MAC PDU downstream in the engine.

## Cheat sheet

| Concept | Value / shape | GopherTrunk file |
|---|---|---|
| Symbol rate | 6000 sym/s, H-DQPSK | `phase2/receiver` |
| Superframe | 360 ms, 12 sub-frames, 2160 dibits | `phase2/superframe.go` |
| Sub-frame | 30 ms, 180 dibits | `phase2/superframe.go` |
| ISCH | 12 data bits, Golay(24,12,8) | `phase2/isch.go` |
| MAC PDU | opcode + [MFID] + payload | `phase2/mac.go` |
| Grant (full) | opcode 0x44 — src present | `phase2/mac.go` |
| Grant (compressed) | opcode 0x01 — **src = 0** | `phase2/mac.go` |

## In this post

- **Why TDMA changes the decode** — signalling moves inside the traffic channel.
- **The ISCH** — the one field that routes a sub-frame.
- **The superframe** — 12 slots, one anchor.
- **The MAC PDU layer** — opcodes, grants, status, encryption sync.
- **The source-less grant** — the compression that costs a field.

## Why TDMA changes the shape

In Phase 1, a control channel is a dedicated carrier streaming TSBKs. In Phase 2,
the two timeslots of a carrier are shared between voice and signalling: the same
360 ms **superframe** that carries digital voice also carries **MAC PDUs** in the
sub-frames that aren't voice. So the decoder can't just "read the control channel"
— it has to lock the superframe grid, decode a per-sub-frame slot-type field, and
route voice sub-frames one way and MAC sub-frames another. The `SuperframeDecoder`
does exactly this before the control-channel state machine ever sees a PDU.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A P25 Phase 2 superframe: an outbound sync at sub-frame zero anchors a grid of twelve 30-millisecond sub-frames, each prefixed by an ISCH slot-type field, alternating voice and MAC content across two timeslots">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="10">360 ms superframe · 12 sub-frames</text>
  <rect x="14" y="34" width="40" height="40" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="34" y="58" text-anchor="middle" fill="var(--accent)" font-size="9">SYNC</text>
  <g font-size="9">
    <rect x="56" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="81" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="81" y="66" text-anchor="middle" fill="currentColor">voice</text>
    <rect x="108" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="133" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="133" y="66" text-anchor="middle" fill="var(--accent)">MAC</text>
    <rect x="160" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="185" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="185" y="66" text-anchor="middle" fill="currentColor">voice</text>
    <rect x="212" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="237" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="237" y="66" text-anchor="middle" fill="currentColor">voice</text>
    <rect x="264" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="289" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="289" y="66" text-anchor="middle" fill="var(--accent)">MAC</text>
    <rect x="316" y="34" width="50" height="40" rx="3" fill="none" stroke="currentColor"/><text x="341" y="52" text-anchor="middle" fill="var(--fg-muted)">ISCH</text><text x="341" y="66" text-anchor="middle" fill="currentColor">voice</text>
  </g>
  <text x="392" y="60" fill="var(--fg-muted)" font-size="16">…</text>
  <line x1="14" y1="88" x2="366" y2="88" stroke="var(--fg-muted)"/>
  <line x1="14" y1="84" x2="14" y2="92" stroke="var(--fg-muted)"/><line x1="66" y1="84" x2="66" y2="92" stroke="var(--fg-muted)"/>
  <text x="40" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">30 ms / 180 dibits</text>
  <text x="340" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the ISCH prefix routes each sub-frame; voice → composer, MAC → control-channel state machine</text>
</svg>
<figcaption>The Phase 2 superframe. A sync at sub-frame 0 anchors the 12-slot grid; each sub-frame's ISCH says whether the payload is voice or a MAC PDU.</figcaption>
</figure>

## The ISCH: one field that routes a sub-frame

The Inter-Slot Signalling Channel prefixes every sub-frame and answers one
question — what does this sub-frame carry? — cheaply, so the decoder never has to
speculatively parse a payload. It's a 12-bit field (a 4-bit `SlotType`, a 4-bit
sub-frame counter, 4 reserved) protected by the extended Golay(24,12,8) code:

```go
// internal/radio/p25/phase2/isch.go (shape)
func DecodeISCH(dibits []uint8) (ISCH, int, error) {
    var cw uint32
    for _, d := range dibits { cw = cw<<2 | uint32(d&3) }
    data, errs := framing.GolayDecode24_12(cw) // errs < 0 ⇒ uncorrectable
    if errs < 0 { return ISCH{}, -1, ErrISCHUncorrectable }
    return ISCH{
        SlotType: SlotType(data & 0x0F),
        Counter:  uint8((data >> 4) & 0x0F),
    }, errs, nil
}
```

The `SlotType` enum splits cleanly into voice (`Voice4V`, `Voice2V`) and MAC
(`MAC_PTT`, `MAC_ACTIVE`, `MAC_IDLE`, `MAC_HANGTIME`, `MAC_END`…), with `IsMAC()`
and `IsVoice()` helpers the decoder branches on. All ISCH knowledge is deliberately
confined to this one file — the exact bit packing is a working model against
TIA-102.BBAB — so a spec correction is a local change and the rest of the pipeline
only ever sees a decoded `SlotType`. That containment discipline is the same one
Part 1 described for the whole registry, applied at field scale.

## The superframe grid

The superframe geometry is almost entirely constants: 12 sub-frames of 180 dibits,
2160 dibits total (× 2 bits = 4320, matching the PN44 scrambler period). The one
piece of real machinery is *where the grid starts*, and that's anchored on a single
constant — the sub-frame whose head carries the 20-dibit outbound sync word:

```go
// internal/radio/p25/phase2/superframe.go (shape)
const (
    SubframesPerSuperframe = 12
    DibitsPerSubframe      = 180
    DibitsPerSuperframe    = SubframesPerSuperframe * DibitsPerSubframe // 2160
    SyncSubframeIndex      = 0 // anchor the 0..11 grid on a sync match here
)
```

`SuperframeDecoder.Process` slides a sync detector over the H-DQPSK dibit stream,
anchors the grid on each match, decodes one ISCH per sub-frame, and hands the
MAC-bearing sub-frames to `IngestSuperframe`. If a real capture ever shows the sync
recurring at a different cadence, `SyncSubframeIndex` is the single fix — the slice
geometry derives entirely from it. This is the tidy end of the design; the messy
end is the descrambler.

Phase 2 scrambles the traffic channel with a PN44 sequence seeded from the system's
identity — WACN, System ID, and the Color Code (which equals the Phase 1 NAC per
spec). The seed is computed from the **Network Status Broadcast**, so the same MAC
PDU that tells you *which* system you're on also unlocks the descrambler. Until an
NSB arrives, a hand-configured seed stands in; when one arrives, the decoder
recomputes and swaps the seed if it differs.

## The MAC PDU layer

A MAC PDU is the Phase 2 counterpart of a TSBK, and its parse is deliberately the
same shape: opcode first, an MFID octet only for manufacturer-specific opcodes,
then opcode-specific payload:

```go
// internal/radio/p25/phase2/mac.go (shape)
func ParseMACPDU(info []byte) (MACPDU, error) {
    pdu := MACPDU{Opcode: Opcode(info[0])}
    rest := info[1:]
    if pdu.Opcode.IsManufacturerSpecific() && len(rest) >= 1 { // 0x80..0xBF
        pdu.MFID = rest[0]
        rest = rest[1:]
    }
    pdu.Payload = append([]byte(nil), rest...)
    return pdu, nil
}
```

The familiar messages reappear under new opcode numbers: `GroupVoiceChannelGrant`
(0x44) and its update (0x40), the unit-to-unit grant (0x48), the RFSS and Network
Status Broadcast updates (0xFA/0xFB), the Identifier Update band plan (0x7D), and
the Encryption Sync (0x70) that surfaces algorithm and key IDs for a protected call
(GopherTrunk identifies encryption but does not decrypt). Each has an `As*`
accessor that returns `false` on an opcode mismatch, so the state machine probes a
PDU against several in turn. A full grant publishes exactly the `trunking.Grant` the
[trunking engine consumes]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }}) —
same event, same downstream — carrying the channel, talkgroup, source, encryption
flag, and the FEC-config snapshot the voice composer needs.

## The problem we hit: source-less grants

Phase 2 has *two* ways to tell you a group call is up, and they cost different
amounts of airtime. The full `GroupVoiceChannelGrant` (0x44) carries service
options, channel, talkgroup, **and** the 24-bit source radio ID. But the in-call
`GroupVoiceChannelUser` broadcast comes in a compressed **Abbreviated** form
(opcode 0x01) that, in the field, arrives with the source **compressed away** —
`SourceID` decodes to 0. The Extended form (0x21) carries the full SUID, but the
abbreviated form is what a system emits most of the time to save slot space:

```go
// internal/radio/p25/phase2/mac.go (shape)
type GroupVoiceChannelUser struct {
    ServiceOptions uint8
    GroupAddress   uint16
    SourceID       uint32 // 24-bit — but 0 in the compressed abbreviated form
}
// opcodes: OpGroupVoiceChannelUserAbbreviated (0x01) · …Extended (0x21)
```

So the decoder faithfully reports a call on a talkgroup with **no source radio**.
That's not a bug in the parse — the source genuinely isn't on the wire in that PDU.
The talkgroup is enough to *start* and file the call, but "who was talking" has to
come from somewhere else: a subsequent extended in-call PDU, or the full grant, seen
later in the transmission. Resolving it is a stateful, cross-PDU backfill, and it
belongs in the layer that owns call state, not the stateless decoder.

<figure class="lab-figure">
<svg viewBox="0 0 680 158" width="680" height="158" role="img" aria-label="Two MAC PDUs describe the same call: the full grant 0x44 carries the 24-bit source radio ID, while the compressed abbreviated user 0x01 decodes its source to zero, and the trunking engine backfills the missing source downstream where call state lives">
  <rect x="6" y="16" width="300" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="156" y="36" text-anchor="middle" fill="currentColor" font-size="12">Full grant 0x44</text>
  <text x="156" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="10">TG=1234 · Source=5001 ✓</text>
  <text x="156" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="10">costs slot space</text>
  <rect x="6" y="86" width="300" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="156" y="106" text-anchor="middle" fill="var(--accent)" font-size="12">Abbreviated user 0x01</text>
  <text x="156" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="10">TG=1234 · Source=0 ✗ (compressed)</text>
  <text x="156" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the common case in the field</text>
  <line x1="306" y1="77" x2="372" y2="77" stroke="currentColor"/><polygon points="372,73 382,77 372,81" fill="currentColor"/>
  <rect x="382" y="40" width="150" height="74" rx="6" fill="none" stroke="currentColor"/>
  <text x="457" y="70" text-anchor="middle" fill="currentColor" font-size="11">stateless decoder</text>
  <text x="457" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="10">publishes what</text>
  <text x="457" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the PDU says</text>
  <line x1="532" y1="77" x2="596" y2="77" stroke="var(--accent)"/><polygon points="596,73 606,77 596,81" fill="var(--accent)"/>
  <rect x="606" y="46" width="68" height="62" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="640" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">engine</text>
  <text x="640" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">backfills</text>
  <text x="640" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">source</text>
</svg>
<figcaption>The source-less-grant seam: the decoder faithfully reports <code>Source=0</code> from the compressed PDU; reconciling who was talking is the trunking engine's job, not the decoder's.</figcaption>
</figure>

### How that principle shaped the Go code

- **The decoder stays stateless about calls.** It publishes what the PDU says —
  including a source of 0 — and does not try to remember prior transmissions to
  patch the gap. Keeping the decoder memoryless is what keeps it testable one PDU at
  a time.
- **The backfill lives in the engine.** The source RID is reconciled downstream,
  where call state already exists, in the
  [trunking engine's source-RID recovery]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}).
  The seam is the `SourceID == 0` grant.
- **Both grant shapes publish one event type.** Abbreviated, extended, full grant,
  unit-to-unit — all normalise to `trunking.Grant`, so subscribers never branch on
  Phase 1 vs Phase 2 or compressed vs full.
- **FEC config travels with the grant.** The trellis/RS/interleave/scrambler state
  is snapshotted into the grant so the voice composer can run the *same* MAC dispatch
  on the voice-interleaved MAC sub-frames — which is how talker-alias fragments reach
  the receiver on Phase 2 systems that don't emit them on the control channel.

That last point is our recurring thread surfacing again: on Phase 2, the Motorola
talker alias rides FACCH-S MAC PDUs interleaved with voice, reassembled and passed
through the *same* unverified per-byte cipher as Phase 1 — and gated the same way on
`motorola.CipherVerified`. The carriage differs; the wall is identical.

## Where this goes next

[Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
leaves P25 for DMR — another two-slot TDMA family, but with its own burst structure,
EMB signalling, Full Link Control, and a reverse channel. The source-less-grant
problem introduced here is picked up in full by the
[trunking engine]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}),
and the Phase 2 voice frames feed the
[Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }}) series.

## FAQ

**How is P25 Phase 2 different from Phase 1?**
Phase 1 is FDMA (one call per 12.5 kHz carrier, C4FM). Phase 2 is two-slot TDMA
(two calls per carrier, H-DQPSK at 6000 sym/s), and its signalling rides as MAC PDUs
inside the traffic-channel superframe rather than as standalone TSBKs.

**What does the ISCH do?**
It prefixes every sub-frame with a Golay-protected slot-type field so the receiver
can route the sub-frame — voice to the composer, MAC to the control-channel state
machine — without speculatively decoding the payload.

**Why do some Phase 2 grants have no source radio?**
Because the compressed, abbreviated in-call `GroupVoiceChannelUser` PDU (opcode 0x01)
omits the source to save slot space; its `SourceID` decodes to 0. The source is
recovered later from an extended PDU or the full grant, in the trunking engine.

**What is the PN44 seed and where does it come from?**
It's the scrambler seed for the Phase 2 traffic channel, derived from the system's
WACN, System ID, and Color Code — values carried in the Network Status Broadcast MAC
PDU. Until an NSB is seen, a configured seed stands in.

**Does GopherTrunk decrypt encrypted Phase 2 calls?**
No. It parses the Encryption Sync MAC PDU to identify the algorithm and key IDs (and
flags the call protected via the grant's service options), but it does not decrypt.

## Series navigation

**Part 4 of 12** · ← [Part 3: P25 Phase 1 TSBKs & Link Control]({{ '/blog/deep-dives/protocol-decoders-03-p25-phase-1-tsbk-link-control/' | relative_url }}) · Next →
[Part 5: DMR Tier 2/3 — Bursts, EMB & FLC]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
