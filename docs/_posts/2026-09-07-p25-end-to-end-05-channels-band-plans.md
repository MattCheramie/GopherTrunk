---
title: "P25 End to End, Part 5: Channel Identifiers & Band Plans — From Channel Number to Hertz"
description: How a P25 grant's 16-bit channel field becomes a frequency — the IDEN_UP band-plan broadcasts in their 0x3D, 0x34 and 0x33 dialects, the base-plus-spacing arithmetic in BandPlan.Frequency, and what happens when a grant arrives before the table that decodes it.
category: deep-dives
keywords: p25 iden_up, p25 band plan, p25 channel identifier, p25 channel number to frequency, iden_up_tdma 0x33, p25 grant frequency calculation, p25 tx offset uplink, trunking band plan decoding, gophertrunk p25
tags: [p25-end-to-end, p25, band-plan, trunking, control-channel, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 5
---

*Part 5 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})
decoded Multi-Block Trunking and recovered the site broadcasts some
systems only send in AMBT form. But every grant and neighbor announcement
names its frequency the same indirect way: a 4-bit channel identifier plus
a 12-bit channel number. This part is the dictionary turning those sixteen
bits into hertz — the IDEN_UP broadcasts, the arithmetic, and the failure
modes when the dictionary is late, missing, or wrong.*

> **TL;DR:** P25 never broadcasts frequencies with grants. A grant carries
> a 16-bit channel field — **4-bit channel ID + 12-bit channel number** —
> and the site separately broadcasts **IDEN_UP** messages defining each ID
> slot's base frequency, spacing and transmit offset. GopherTrunk
> accumulates them in `phase1.BandPlan`
> (`internal/radio/p25/phase1/identifier.go`) and resolves
> `freq = BaseHz + ChannelNumber × SpacingHz`. Three on-air dialects
> (opcodes **0x3D**, **0x34** VHF/UHF, **0x33** TDMA) parse into one
> struct; bases scale ×5 Hz, spacing ×125 Hz. A grant arriving before its
> IDEN_UP is queued 5 s (`pendingGrants`) and surfaced as a `decode.error`
> with `stage="no-bandplan"` — and an explicit uplink channel from Part 4's
> AMBT broadcasts resolves as plain base+spacing with **no** tx offset.

**Key takeaways**

- **A grant is not tunable by itself.** Its 16 bits index a table the site
  broadcasts separately, on its own schedule. Every P25 receiver is
  therefore stateful: no IDEN_UP yet, no frequency — which is why
  GopherTrunk queues early grants instead of dropping them.
- **Three IDEN_UP dialects, and their offset fields disagree.** Opcode
  0x3D scales its 9-bit signed offset by a fixed 250 kHz; 0x34 and 0x33
  carry a sign bit plus a 13-bit magnitude scaled by the *channel
  spacing*. Parse one with the other's rules and every uplink you compute
  is fiction.
- **An explicit uplink channel number already encodes the uplink
  frequency.** When an AMBT neighbor broadcast names an uplink channel,
  GopherTrunk resolves it base+spacing with no transmit offset — matching
  SDRTrunk, and the opposite of what the offset field tempts you to do.
- **The channel ID carries more than arithmetic.** A slot advertised via
  opcode 0x33 is flagged `AccessTDMA`, and that one bit routes the grant
  into the Phase 2 voice chain — wiring hybrid Phase 1 CC / Phase 2 voice
  systems (issue #376) silently lacked.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Standard IDEN_UP (0x3D) | 700/800/900 form: 9-bit BW, 9-bit offset ×250 kHz | `internal/radio/p25/phase1/identifier.go` (`ParseIdentifierUpdate`) |
| VHF/UHF IDEN_UP (0x34) | 4-bit BW code, sign+13-bit offset ×spacing | `ParseIdentifierUpdateVUHF` |
| TDMA IDEN_UP (0x33) | channel-type nibble, flags the slot `AccessTDMA` | `ParseIdentifierUpdateTDMA` |
| The table | 16 slots of (base, spacing, offset), one per channel ID | `identifier.go` (`BandPlan.Apply` / `Frequency`) |
| Grant resolution | (ID, number) → Hz, or `stage="no-bandplan"` error | `control.go` (`publishVoiceGrant`) |
| Early-grant queue | 5 s TTL, 4-deep ring per channel ID, drained on Apply | `pending_grants.go` (`pendingGrants`) |
| Phase 2 routing | `IsTDMA(id)` flips the grant to `"p25-phase2"` | `identifier.go` + `control.go` |

## In this post

- **Frequencies are never broadcast with grants** — the 4+12-bit indirection.
- **Three dialects, one struct** — 0x3D vs 0x34 vs 0x33, where they disagree.
- **The arithmetic, worked** — bitfield to megahertz in one figure.
- **When the table is late, missing, or wrong** — pending grants, `no-bandplan`, tuning to nothing.
- **Uplinks, offsets, and the AMBT rule** — the Part 4 callback, in code.

## Frequencies are never broadcast with grants

Here is the whole payload of a Group Voice Channel Grant (TSBK opcode
0x00), the message
[Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
decoded hundreds of times a minute:

```go
// internal/radio/p25/phase1/opcodes.go (shape) — ParseGroupVoiceChannelGrant
func ParseGroupVoiceChannelGrant(p [8]byte) GroupVoiceChannelGrant {
    chanField := binary.BigEndian.Uint16(p[1:3])
    return GroupVoiceChannelGrant{
        ServiceOptions: p[0],
        ChannelID:      uint8(chanField >> 12),  // 4 bits: which band plan
        ChannelNumber:  chanField & 0x0FFF,      // 12 bits: which channel in it
        GroupAddress:   binary.BigEndian.Uint16(p[3:5]),
        SourceID:       uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
    }
}
```

Sixteen bits of channel, none of them a frequency. The reason is economy:
a TSBK payload is eight bytes, a frequency takes four, and a site grants
calls constantly while its frequency plan changes never. So P25
factors the plan out — the site periodically broadcasts up to sixteen
**channel identifier** definitions, each pairing a 4-bit ID with a base
frequency, spacing, nominal bandwidth and transmit offset, and every grant,
neighbor announcement and secondary-CC broadcast thereafter spends only 16
bits per channel reference. The same `ID<<12 | number` packing appears in
every channel-bearing TSBK in `opcodes.go` and in Part 4's AMBT forms alike.

The cost of the economy is state: a receiver tuning in mid-stream knows
the grants before the dictionary — a race GopherTrunk built real machinery
for, below.

## Three dialects, one struct

The dictionary entries arrive as **IDEN_UP** TSBKs, in three on-air
variants. All three parse into one `IdentifierUpdate` struct — downstream
code only wants base, spacing, and offset — but the byte-0 low nibble and
the offset field mean different things in each:

| | 0x3D `IDEN_UP` | 0x34 `IDEN_UP_VUHF` | 0x33 `IDEN_UP_TDMA` |
|---|---|---|---|
| Typical bands | 700/800/900 MHz | VHF / UHF | Phase 2 TDMA channels |
| Byte-0 low nibble | top of a 9-bit BW field (×125 Hz) | 4-bit BW lookup code | 4-bit channel-type code (slots + access mode) |
| Tx offset | 9-bit two's complement × **250 kHz** | sign + 13-bit magnitude × **SpacingHz** | sign + 13-bit magnitude × **SpacingHz** |
| Extra semantics | — | — | marks the slot `AccessTDMA` |

Two fields are identical everywhere, and their scaling explains every
suspicious number in a band-plan log: the 32-bit base frequency is in
units of **5 Hz** (`BaseHz = freq32 × 5`), the 10-bit channel step in
units of **125 Hz** (`SpacingHz = step × 125`). The standard form's parser
is compact enough to quote:

```go
// internal/radio/p25/phase1/identifier.go (shape) — ParseIdentifierUpdate
// Bit layout, MSB first: [ ChanID(4) | BW(9) | OFF(9) | STEP(10) | FREQ(32) ]
func ParseIdentifierUpdate(p [8]byte) IdentifierUpdate {
    chanID := p[0] >> 4
    /* … extract bw, offRaw, step, freq5Hz from the packed bytes … */
    off := int16(offRaw)
    if off&0x100 != 0 {
        off -= 0x200 // sign-extend the 9-bit two's-complement offset
    }
    return IdentifierUpdate{
        ChannelID:   chanID,
        BandwidthHz: uint32(bw) * 125,
        SpacingHz:   uint32(step) * 125,
        TxOffsetHz:  int64(off) * 250_000, // 0x3D only: fixed 250 kHz units
        BaseHz:      uint64(freq5Hz) * 5,
    }
}
```

The VUHF and TDMA parsers were cross-checked against two independent
decoders — OP25's `trunking.py` and SDRTrunk's `FrequencyBandUpdateVUHF` /
`FrequencyBandUpdateTDMA` — rather than against GopherTrunk's own
assemblers. Deliberate policy: Part 3's SCCB bug read a channel one byte
early and its round-trip test passed *because the assembler encoded the
same wrong layout*. A parser pinned only by its own inverse is pinned to
nothing.

One more thing rides on the 0x33 form. Its channel-type nibble says the
granted channels are Phase 2 H-DQPSK TDMA carriers, so the parser stamps
`AccessTDMA: true` on the slot. When a grant later resolves through that
ID, `BandPlan.IsTDMA` flips the published protocol to `"p25-phase2"` so
the composer runs the Phase 2 voice chain — wiring that was missing on
hybrid Phase 1-CC / Phase 2-voice systems until issue #376, where every
grant landed in the Phase 1 chain and the Phase 2 MAC dispatch was dead
code. [Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
picks up the other side of that routing decision.

## The arithmetic, worked

Resolution is one line plus guard rails:

```go
// internal/radio/p25/phase1/identifier.go (shape) — BandPlan.Frequency
func (b *BandPlan) Frequency(channelID uint8, channelNumber uint16) (uint32, error) {
    if int(channelID) >= len(b.slots) || !b.known[channelID] {
        return 0, fmt.Errorf("%w: id=%d", ErrUnknownChannelID, channelID)
    }
    u := b.slots[channelID]
    hz := u.BaseHz + uint64(channelNumber)*uint64(u.SpacingHz)
    if hz > 0xFFFFFFFF { // malformed IDEN_UP must not silently wrap
        return 0, fmt.Errorf("p25/phase1: resolved frequency %d Hz overflows uint32", hz)
    }
    return uint32(hz), nil
}
```

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A 16-bit channel field splits into a 4-bit identifier selecting a band-plan slot and a 12-bit channel number; base plus number times spacing yields 853.88125 megahertz, and a missing slot queues the grant instead">
  <!-- channel field -->
  <rect x="30" y="28" width="60" height="30" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <rect x="90" y="28" width="180" height="30" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="60" y="47" text-anchor="middle" fill="var(--accent)" font-size="11">ID = 1</text>
  <text x="180" y="47" text-anchor="middle" fill="currentColor" font-size="11">number = 460</text>
  <text x="150" y="18" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant channel field (16 bits): 0x11CC</text>
  <text x="60" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">4 bits</text>
  <text x="180" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">12 bits</text>
  <!-- band plan table -->
  <rect x="330" y="24" width="320" height="76" fill="none" stroke="var(--fg-muted)"/>
  <text x="340" y="40" fill="var(--fg-muted)" font-size="10">band plan (from IDEN_UP broadcasts)</text>
  <text x="340" y="58" fill="currentColor" font-size="10">id 0: base 851.00625 MHz · spacing 6.25 kHz</text>
  <text x="340" y="76" fill="var(--accent)" font-size="10" font-weight="bold">id 1: base 851.00625 MHz · spacing 6.25 kHz</text>
  <text x="340" y="94" fill="var(--fg-muted)" font-size="10">id 2..15: unknown until broadcast</text>
  <path d="M 92 60 C 140 110 260 90 334 74" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <polygon points="334,74 322,72 328,82" fill="var(--accent)"/>
  <!-- arithmetic -->
  <text x="60" y="140" fill="currentColor" font-size="11">851.00625 MHz</text>
  <text x="168" y="140" fill="var(--fg-muted)" font-size="11">+</text>
  <text x="185" y="140" fill="currentColor" font-size="11">460 × 6.25 kHz</text>
  <text x="300" y="140" fill="var(--fg-muted)" font-size="11">=</text>
  <text x="320" y="140" fill="var(--accent)" font-size="12" font-weight="bold">853.88125 MHz</text>
  <text x="60" y="162" fill="var(--fg-muted)" font-size="10">BaseHz (freq32 × 5 Hz)   number × SpacingHz (step × 125 Hz)</text>
  <!-- failure branch -->
  <line x1="30" y1="182" x2="650" y2="182" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="30" y="202" fill="currentColor" font-size="10" font-weight="bold">no IDEN_UP for that ID yet?</text>
  <text x="230" y="202" fill="var(--fg-muted)" font-size="10">→ queue 5 s (pendingGrants) + publish decode.error stage="no-bandplan"</text>
  <text x="230" y="218" fill="var(--fg-muted)" font-size="10">→ drained when BandPlan.Apply lands the slot</text>
</svg>
<figcaption>Sixteen bits of grant become a frequency only through the band-plan table the site broadcasts separately — and GopherTrunk queues the grant when the table hasn't arrived yet.</figcaption>
</figure>

Run the example backwards to see the scaling: base 851.00625 MHz is
`freq32 = 170201250` in 5 Hz units; spacing 6.25 kHz is `step = 50` in
125 Hz units. The overflow guard exists in the same spirit: a malformed
broadcast must produce an error, never a silently wrapped frequency that
looks plausible.

## When the table is late, missing, or wrong

The race is real and routine: tune to a busy control channel and the first
grant usually beats the first IDEN_UP. GopherTrunk's answer has two halves.
First, visibility: `publishVoiceGrant` can't act on a grant whose channel
ID has no slot, so it publishes a `decode.error` with
`stage="no-bandplan"` — a counter operators can watch, not a silent drop.
Second, patience: the grant enters `pendingGrants`
(`internal/radio/p25/phase1/pending_grants.go`), a bounded ring of four
entries per channel ID, drained the moment `BandPlan.Apply` lands the
matching slot, with a **5-second TTL** — a voice grant addresses a live
window measured in seconds, so resolving one later buys nothing, and the
bound caps memory against a site that never defines some ID. This is the
difference between "missed the first half-second of the first call after
tune-in" and "missed the call."

*Wrong* is worse than *missing*, because wrong resolves. A stale or
corrupted band plan tunes every voice grant to a confidently computed
frequency where nothing is transmitting — recordings of hiss, or none at
all, while the control channel decodes perfectly. The 800 MHz **rebanding**
era made this a live concern: NPSPAC-region systems moved their voice
channels years ago, so any receiver carrying a hardcoded 800 MHz table
decodes grants against history. GopherTrunk's position is structural: there
is no built-in table. The band plan is *always* learned from the site's own
broadcasts (the wideband path can pre-seed it from a config snapshot), so a
rebanded, splintered, or just-plain-odd system is decoded on its own
declared arithmetic. The learned plan is also exported —
`BandPlan.Snapshot` feeds the discovery and
[hunting]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
layers so a discovered system is documented with the plan it actually
advertised.

## Uplinks, offsets, and the AMBT rule

Every IDEN_UP carries a transmit offset — uplink = downlink + offset —
mostly documentation for a receive-only scanner. But
[Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})'s
AMBT broadcasts reintroduce it with a trap: the Adjacent Status and RFSS
Status AMBT forms carry **explicit uplink channels** — a second
(ID, number) pair naming the neighbor's uplink directly. The tempting bug
is to resolve that pair and then also apply the offset. The rule
GopherTrunk encodes, matching SDRTrunk's AMBT resolution, is that an
explicit uplink channel number *already encodes the uplink frequency*:
the topology snapshot in `control.go` runs the neighbor's
`(UplinkID, UplinkNumber)` through the very same `bandPlan.Frequency`
call as any downlink — plain base+spacing, with the comment "no transmit
offset is applied" doing the standing-rule duty.

Same arithmetic, same table, no offset. The offset exists so a *derived*
uplink can be computed when only the downlink is named; an explicit uplink
channel is not derived. Getting this wrong doesn't break receive — it
corrupts the exported site topology
[Part 10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
builds roaming on: exactly the kind of error that survives for months
because nothing tunes to it.

### How the indirection shaped the Go code

- **The table is a type, not a map in a handler.** `BandPlan` owns the
  sixteen slots and the arithmetic; the control channel and the
  [trunking engine]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
  only ever see a resolved `trunking.Grant.FrequencyHz` or an error.
- **Three parsers, one output shape.** The dialects differ exactly where
  confusing them is expensive, so each opcode gets its own parser and its
  own literal-vector tests — all converging on `IdentifierUpdate`.
- **Absence is an event.** `ErrUnknownChannelID` becomes a published
  `stage="no-bandplan"` decode error — a gap you can graph gets fixed; a
  gap that logs at debug does not.
- **Grants wait, bounded.** `pendingGrants` turns the broadcast race into
  a short TTL'd queue instead of dropping the first call after tune-in or
  growing memory on a broken site.

## Where this goes next

Everything so far assumed Part 1's FM discriminator — the C4FM physics
nearly every P25 site transmits.
[Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})
walks the twin path: the linear CQPSK/LSM demodulator for the minority of
sites where the discriminator produces near-random dibits, the equalizer
that path carries, and the hard-won rule that "simulcast" does not mean
what the project's own docs once said.

## FAQ

**Why doesn't P25 just broadcast the frequency in the grant?**
Payload economy on a channel granting calls many times per second: a
16-bit index is half a 32-bit frequency and the plan it indexes almost
never changes. The trade is that every receiver becomes stateful — no
grant is actionable before the site's IDEN_UP table arrives, which is why
GopherTrunk queues the ones that beat it.

**How long does it take to learn a site's band plan?**
Sites repeat IDEN_UP continuously alongside their status broadcasts — the
slots that matter usually land within seconds of locking. The 5-second
pending-grant TTL is sized to that cadence: long enough to bridge a normal
broadcast gap, short enough that a stale grant isn't resolved pointlessly
late.

**What does `decode.error stage="no-bandplan"` mean in my metrics?**
A voice grant referenced a channel ID with no IDEN_UP seen yet. Occasional
counts right after lock are the normal race; a steady rate means the site
uses an ID it rarely defines on this control channel, or your capture is
dropping the TSBKs that define it.

**Can I hardcode my system's band plan instead of learning it?**
The wideband path can pre-seed the table from configured `P25BandPlan`
entries, which helps replay and very short captures. But the on-air
broadcasts still apply and win — a site's own IDEN_UP is the ground truth,
and rebanding history is the argument against static tables.

**Does the tx offset matter for a receive-only scanner?**
For tuning, no — GopherTrunk only tunes downlinks. It matters for the
exported topology, and it matters that you *don't* apply it to an explicit
AMBT uplink channel, which already encodes the uplink frequency in
base+spacing terms.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Multi-Block Trunking — The PDU That Isn't Noise]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})
· Next →
[Part 6: CQPSK & LSM — The Linear Twin Path]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})
