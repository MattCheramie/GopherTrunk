---
title: "Protocol Decoders, Part 3: P25 Phase 1 TSBKs & Link Control"
description: How GopherTrunk parses P25 Phase 1 Trunking Signaling Blocks — the CRC-checked 12-byte TSBK, its OSP opcode taxonomy, vendor Motorola extensions, TDULC and Link Control — and turns a grant PDU into the Grant the trunking engine records.
category: deep-dives
keywords: p25 tsbk, p25 phase 1 opcodes, group voice channel grant, p25 link control, tdulc, motorola vendor tsbk, p25 crc, trunking grant, gophertrunk p25 decoder
tags: [protocol-decoders, p25, tsbk, link-control, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 3
---

*Part 3 of **Protocol Decoders**. Part 2 delivered 12 clean bytes out of the
trellis; this post is what those bytes *say*. We parse the TSBK, walk the opcode
taxonomy, follow a grant PDU all the way to the `Grant` the trunking engine
records, and open the Link Control / TDULC path — which is exactly where the
talker-alias mystery from Part 1 rides.*

> **TL;DR:** A P25 Phase 1 TSBK is a 12-byte block: a header byte (last-block flag,
> protected flag, 6-bit opcode), an MFID, 8 payload bytes, and a 2-byte CRC. The
> opcode plus MFID pick a parser — `GRP_V_CH_GRANT` (opcode 0x00) becomes the
> grant the engine turns into a recorded call; `RFSS_STS_BCST` and friends backfill
> site identity; Motorola/Harris MFIDs remap the same opcode space to vendor
> messages like patches and the talker alias. Link Control words (in LDU1 and the
> TDULC terminator) carry per-call talkgroup, source, and alias fragments behind an
> RS-over-Hamming FEC stack.

**Key takeaways**

- The TSBK is uniform: same 12-byte frame, same CRC, every opcode. Only the 8
  payload bytes change meaning.
- The **CRC is augmented-codeword** style (init 0, final XOR 0xFFFF, expect 0 over
  all 12 bytes) — the "obvious" CRC-CCITT/FALSE variant is wrong on-air (issue #275).
- **MFID switches namespaces**: the same 6-bit opcode decodes differently under
  Motorola (0x90) or Harris (0xA4).
- A **group voice channel grant** carries channel + talkgroup + source; that tuple
  is the seed of a `Grant`.
- **Link Control** carries the per-call talkgroup/source under RS(24,12,13); the
  TDULC path is where the Motorola talker alias hides.

## Cheat sheet

| Opcode | Mnemonic | Carries | Becomes |
|---|---|---|---|
| `0x00` | GRP_V_CH_GRANT | channel, talkgroup, source | a **Grant** |
| `0x02` | GRP_V_CH_GRANT_UPDT | two (channel, group) pairs | grant refresh |
| `0x04` | UU_V_CH_GRANT | channel, target, source | private-call grant |
| `0x3B` | NET_STS_BCST | WACN, System ID | network identity |
| `0x3A` | RFSS_STS_BCST | RFSS, Site | camped-site identity |
| `0x3C` | ADJ_STS_BCST | neighbour site + CC | roaming target |
| `0x90/0x00` | Motorola patch add | super-group + members | a regroup/patch |
| `0x90/0x15` | Motorola talker alias | alias fragment | **the mystery** |

## In this post

- **The TSBK frame** — header, MFID, payload, and the CRC that gates it.
- **The opcode taxonomy** — grants, updates, status broadcasts.
- **From grant to Grant** — the field that seeds a recorded call.
- **Vendor namespaces** — how MFID remaps the opcode space.
- **Link Control & TDULC** — per-call identity, and the alias carriage.

## The TSBK frame

A TSDU (DUID 0x7) carries one to three TSBKs, each a 12-byte info block recovered
by the trellis + interleaver from Part 2. The header byte is fixed-shape; the rest
is opcode-dependent:

```go
// internal/radio/p25/phase1/tsbk.go (shape)
type TSBK struct {
    LB      bool   // Last Block in the TSDU sequence
    P       bool   // Protected
    Opcode  Opcode // 6-bit opcode
    MFID    uint8  // 0x00 = standard, 0x90 = Motorola, 0xA4 = Harris
    Payload [8]byte
}
```

`ParseTSBK` peels the header, copies the 8 payload octets, and verifies the trailer
CRC. That CRC is worth a full stop, because getting it wrong looked like a decode
failure for a long time:

```go
// internal/radio/p25/phase1/tsbk.go (shape)
if framing.CRCCCITTAugmented(info) != 0 {
    return t, CRCError   // still returns the partial block for diagnostics
}
```

The prior implementation used CRC-CCITT/FALSE (init 0xFFFF, no final XOR) with an
inverted trailer — an algorithm documented in many P25 references but *not* what the
air uses. On the Mt Anakie capture it produced 195 of 197 TSBK CRC failures even
when the trellis reported a clean path (metric 0). The fix (issue #275): the
"augmented codeword" variant — init 0, final XOR 0xFFFF, run over all 12 bytes,
expect 0. A clean trellis decode plus a failing CRC is the fingerprint of a wrong
CRC algorithm, not a wrong signal.

<figure class="lab-figure">
<svg viewBox="0 0 680 128" width="680" height="128" role="img" aria-label="A 12-byte TSBK: header byte with last-block flag, protected flag and 6-bit opcode, then a manufacturer ID byte, eight payload bytes, and a two-byte CRC trailer">
  <rect x="12" y="40" width="70" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="47" y="60" text-anchor="middle" fill="var(--accent)" font-size="11">header</text>
  <text x="47" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">LB·P·op6</text>
  <rect x="86" y="40" width="56" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="114" y="60" text-anchor="middle" fill="currentColor" font-size="11">MFID</text>
  <text x="114" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">1 byte</text>
  <rect x="146" y="40" width="400" height="44" rx="5" fill="none" stroke="currentColor"/>
  <text x="346" y="60" text-anchor="middle" fill="currentColor" font-size="11">payload — 8 opcode-specific bytes</text>
  <text x="346" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">channel · talkgroup · source · …</text>
  <rect x="550" y="40" width="118" height="44" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="609" y="60" text-anchor="middle" fill="var(--accent)" font-size="11">CRC-16</text>
  <text x="609" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">augmented</text>
  <text x="340" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="10">CRCCCITTAugmented over all 12 bytes must return 0 — the last gate from Part 2's staircase</text>
</svg>
<figcaption>The TSBK is a fixed 12-byte envelope. Header and MFID select a parser for the 8 payload bytes; the CRC is the final accept/reject.</figcaption>
</figure>

## The opcode taxonomy

The 6-bit opcode is a TIA-102.AABC-D Outbound Signalling Packet (OSP) type.
GopherTrunk names the full table with idiomatic Go identifiers while `String()`
carries the canonical spec mnemonic, so logs read in spec terms:

```go
// internal/radio/p25/phase1/opcodes.go (shape)
const (
    OpGroupVoiceChannelGrant      Opcode = 0x00 // GRP_V_CH_GRANT
    OpGroupVoiceChannelUpdate     Opcode = 0x02 // GRP_V_CH_GRANT_UPDT
    OpUnitToUnitVoiceChannelGrant Opcode = 0x04 // UU_V_CH_GRANT
    OpNetworkStatusBroadcast      Opcode = 0x3B // NET_STS_BCST
    OpRFSSStatusBroadcast         Opcode = 0x3A // RFSS_STS_BCST
    OpAdjacentSiteStatusBroadcast Opcode = 0x3C // ADJ_STS_BCST
    OpIdentifierUpdate            Opcode = 0x3D // IDEN_UP
    // …~40 opcodes across grants, registration, status, and identifiers
)
```

They fall into a few families. **Grants** (0x00, 0x02, 0x03, 0x04, 0x06…) announce
voice channels — the events that become calls. **Status broadcasts** (0x3A–0x3D)
carry the system's identity and topology: the Network Status Broadcast is the
authoritative WACN + System ID; the RFSS Status Broadcast names the camped site;
the Adjacent Site Status Broadcast advertises neighbours to roam to. **Identifier
updates** (0x33/0x34/0x3D) carry the band plan that turns a grant's abstract
`(channel ID, channel number)` into a real frequency. **Registration/affiliation**
responses (0x28, 0x2B, 0x2C…) track which radios are on which talkgroups —
the raw material for the affiliation features in the trunking engine.

## From a grant to a `Grant`

The grant is the reason the decoder exists. Opcode 0x00's 8 payload bytes decode to
a channel reference, a talkgroup, and the source radio:

```go
// internal/radio/p25/phase1/opcodes.go (shape)
type GroupVoiceChannelGrant struct {
    ServiceOptions uint8
    ChannelID      uint8   // 4-bit band-plan identifier
    ChannelNumber  uint16  // 12-bit channel within that band
    GroupAddress   uint16  // the talkgroup
    SourceID       uint32  // 24-bit transmitting radio
}
```

The `ServiceOptions` byte is small but load-bearing: bit 7 is Emergency, bit 6 is
**Protected** (the encryption indicator the recorder consults before it wastes disk
on a call it can't play), and the low three bits are call priority. The
`(ChannelID, ChannelNumber)` pair is resolved to a frequency through the band plan
the IDEN_UP opcodes maintain, and the resulting tuple — system, talkgroup,
frequency, source, encrypted flag, priority — is what the control-channel state
machine publishes as `events.KindGrant`. That event is the hand-off: the
[trunking engine]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
picks it up, allocates a voice SDR, and starts a call. Everything in Part 2 and this
post exists to fill in that struct correctly.

A subtlety the decoder gets right: `GRP_V_CH_GRANT_UPDT` (0x02) carries *two*
(channel, group) pairs — it's a compact refresh of currently-active calls, not a new
setup — while the explicit update (0x03) reuses the full grant layout including
source. Parsing them into the right shape is what keeps a single ongoing
transmission from fragmenting into phantom calls.

## Vendor namespaces: MFID changes the language

The same 6-bit opcode means different things under a manufacturer's MFID. This is
handled with `As*` accessors that check the MFID before decoding, so a spec
correction stays local to one file:

```go
// internal/radio/p25/phase1/tsbk_vendor.go (shape)
const (
    MFIDMotorola uint8 = 0x90
    MFIDHarris   uint8 = 0xA4
    OpMotorolaPatchGroupAdd          Opcode = 0x00 // regroup/"patch" add
    OpMotorolaPatchGroupChannelGrant Opcode = 0x02 // super-group voice grant
    OpVendorTalkerAlias              Opcode = 0x15 // alias fragment (MFID-agnostic)
)

func (t TSBK) AsMotorolaPatchGroup() (MotorolaPatchGroup, bool) {
    if t.MFID != MFIDMotorola || t.Opcode != OpMotorolaPatchGroupAdd {
        return MotorolaPatchGroup{}, false
    }
    // …super-group address + up to 3 member talkgroups
}
```

Motorola **patches** (group-regroup) aggregate member talkgroups under a super-group
address, and there's a real-world wrinkle the accessor absorbs: for a
fewer-than-three-member patch, Motorola repeats the first member's ID in the unused
slots rather than zeroing them (Mt Anakie showed `0x7EF5 0x7EF5 0x7EF5` for a
one-member patch), so `AsMotorolaPatchGroup` deduplicates. Those patches feed the
engine's patch registry. And `AsTalkerAliasFragment` — MFID-agnostic, opcode 0x15 —
is one of the two doors the talker alias comes through.

## Link Control and the TDULC — where the alias rides

Grants come from the *control* channel. But once a call is up on a *voice* channel,
each LDU1 frame carries a 72-bit **Link Control** word that restates the call's
talkgroup and source, so a receiver that tuned in mid-call still knows who it's
hearing. That word is wrapped in a two-layer FEC stack:

```go
// internal/radio/p25/phase1/link_control.go (shape)
// 24 shortened Hamming(10,6,3) codewords → 144 bits → 24 six-bit RS symbols;
// outer RS(24,12,13) corrects up to t=6 symbol errors the Hamming layer missed.
type LinkControl struct {
    LCFormat       uint8  // the LCO opcode
    MFID           uint8
    ServiceOptions uint8
    TalkgroupID    uint16 // octets 4-5
    SourceID       uint32 // octets 6-8
}
```

The octet layout here is exact for a reason: an earlier working model placed the
talkgroup at octets 2-3, which made it decode to the constant service-options byte
(0x0400 = 1024) while the real talkgroup landed inside a misread source field. The
on-air talkgroup then never matched the granted talkgroup, the voice composer's
foreign-talkgroup gate fired, and single transmissions shattered into dozens of
2-LDU1 fragments. One wrong byte offset, one very visible field bug.

Certain LCOs don't carry a group-voice-user record at all — they carry alias
fragments. `LCOTalkerAliasHeader` (0x15) and the data blocks (0x16/0x17) assemble a
radio's display name across the call. And here is the carriage twist that matters
for our mystery: on real Motorola systems those alias LCs ride primarily in the
**TDULC** (the terminator-with-link-control, DUID 0xF), not LDU1. The TDULC wraps
the same 72-bit LC word behind an even heavier FEC — 12 Golay(24,12,8) inner
codewords under an RS(24,12,13) outer layer:

```go
// internal/radio/p25/phase1/tdulc.go (shape)
func ExtractTDULCLinkControl(ldu []byte) (lcf uint8, content [9]byte, errs int, err error) {
    // …12 Golay codewords → 24 RS symbols → outer RS(24,12,13) → 9 LC octets
}
```

So the pipeline can recover those 9 alias octets cleanly, error-corrected twice
over. The bytes arrive intact. What happens *next* — running them through the
Motorola per-byte substitution cipher — is the wall we hit.

### The mystery, lightly

> **↩ The alias, revisited.** Feed those recovered LC octets into
> `MotorolaTalkerAliasBuf`, let the header's declared block count arrive, and you
> get a decoded string — and a `reliable` flag that is **always false**. The decode
> runs through `motorola.DecodeAlias`, but GopherTrunk gates the result on
> `motorola.CipherVerified`, which is `false`: the substitution table is a
> clean-room reverse-engineering guess, unconfirmed against a known-plaintext frame
> (issue #773). We can frame it, FEC-correct it, and reassemble it perfectly — and
> it still comes out as noise. That's the same *Mercury* emitter the
> [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series chases, and
> [Part 11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
> is where we try to verify the table.

## Where this goes next

[Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})
moves from FDMA to TDMA: P25 Phase 2's two-slot MAC layer, the ISCH, superframes,
and a nastier grant problem — compressed grants that arrive *source-less*. Grants
decoded here feed the
[trunking engine's grant handling]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }}),
and the voice frames the Link Control rides on feed the
[Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }}) series.

## FAQ

**What is a TSBK?**
A Trunking Signaling Block — the atomic unit of the P25 Phase 1 control channel. It's
a 12-byte block (header, MFID, 8 payload bytes, CRC) whose opcode selects a message
type: grants, updates, status broadcasts, registrations.

**Which TSBK becomes a call?**
`GRP_V_CH_GRANT` (opcode 0x00) and its updates. They carry the channel, talkgroup,
and source; GopherTrunk resolves the channel to a frequency via the band plan and
publishes a `KindGrant` event the trunking engine turns into a recorded call.

**Why does MFID matter for opcode parsing?**
Because vendors reuse the 6-bit opcode space. Under MFID 0x90 (Motorola) opcode 0x00
is a patch-group add, not a standard grant. GopherTrunk dispatches on MFID first,
via `As*` accessors that return `false` for a mismatched manufacturer.

**Where does the per-call talkgroup come from during a call?**
From the Link Control word in each LDU1 (and the TDULC terminator), decoded through a
shortened-Hamming inner layer and an RS(24,12,13) outer layer. It restates the
talkgroup and source so a mid-call tune-in still identifies the traffic.

**Why can't GopherTrunk read the Motorola talker alias?**
The alias bytes reassemble and FEC-correct fine, but the final per-byte substitution
cipher is unverified (`motorola.CipherVerified == false`, issue #773), so the decoder
refuses to report the result as reliable. Parts 10–11 take that on.

## Series navigation

**Part 3 of 12** · ← [Part 2: P25 Phase 1 Physical Layer]({{ '/blog/deep-dives/protocol-decoders-02-p25-phase-1-physical-layer/' | relative_url }}) · Next →
[Part 4: P25 Phase 2 TDMA — The MAC Layer]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})
