---
title: "P25 End to End, Part 4: Multi-Block Trunking — The PDU That Isn't Noise"
description: Why a PDU on a P25 control channel is Multi-Block Trunking, not garbage — the AMBT header and data blocks with their two different CRCs, the neighbour sites and WACN that only arrive this way, and why a "MBT data CRC failed" line next to a clean decode is two frames, not a contradiction.
category: deep-dives
keywords: p25 multi-block trunking, ambt decode, p25 pdu control channel, p25 network status broadcast wacn, adjacent site status broadcast, p25 crc-32 pdu, op25 process_pdu, p25 neighbor sites missing, gophertrunk p25
tags: [p25-end-to-end, p25, mbt, ambt, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 4
---

*Part 4 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice.
[Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
opened the TSBK, the single-block message that runs a trunked system. This
part is about the messages that don't fit in a single block — and about a
decoder that spent months logging its richest incoming data as noise. If your
P25 site table shows one neighbour where another tool shows twelve, or a
blank WACN on a system that certainly has one, this is almost certainly the
layer you're missing.*

> **TL;DR:** A PDU (DUID 0xC) on a P25 control channel is usually
> **Multi-Block Trunking** — and GopherTrunk used to log every one as
> `non-control DUID duid=PDU` and drop it. The cost, straight from the
> operator report that exposed it: SDRTrunk listed **12 neighbour sites with
> uplinks in 15 s** while GT showed **one** and "No Network Status Broadcast
> yet". `mbt.go` now decodes the Alternate MBT (format 0x17, SAP 61): blocks
> reuse the **TSBK 98-dibit trellis** exactly; the 12-octet header ends in
> the same **augmented CRC-CCITT16** as a TSBK trailer, the concatenated data
> blocks in a **CRC-32** — both validated against OP25's `process_PDU`. A
> `p25: MBT data CRC failed` line whose identity fields match a decoded
> broadcast is **two different frames** (the header carries its own CRC), and
> an AMBT's explicit uplink channel resolves as plain base + spacing with
> **no transmit offset**. An MBT decodes everything — but never locks the
> channel.

**Key takeaways**

- **"Non-control DUID" was hiding the network map.** Many systems — notably
  Motorola — broadcast Network Status (the sole WACN carrier), RFSS Status
  and most of their neighbour list *only* in AMBT form. Dropping DUID 0xC
  didn't lose exotic data; it lost the system's self-description.
- **MBT blocks are TSBK blocks.** Same 196-bit, 1/2-rate trellis, same
  interleave, same 98-dibit geometry — `DecodeMBTBlockChannel` reuses the
  Part 3 pipeline wholesale. Only the CRC conventions differ.
- **A failed-CRC line beside a clean decode is not a contradiction.** Every
  identity field on the failure line comes from the *header* block, which
  carries its own CCITT-16 and decoded clean; only a data block failed its
  CRC-32. The broadcast repeats, so a clean copy usually logs nearby.
- **Explicit uplink channels already encode the uplink.** An AMBT that names
  an uplink (id, number) pair resolves via plain base + number·spacing — the
  band plan's transmit offset must NOT be applied on top.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Header parse + CRC-16 | format/SAP/MFID/blocks/opcode, augmented CCITT | `internal/radio/p25/phase1/mbt.go` (`ParseMBTHeader`) |
| Trunking-control gate | SAP 61, formats 0x17 (AMBT) / 0x15 | `mbt.go` (`MBTSAPTrunkingControl`, `IsTrunkingControl`) |
| Block channel decode | reuses TSBK deinterleave + Viterbi | `mbt.go` (`DecodeMBTBlockChannel`) |
| Data CRC-32 | poly 0x04C11DB7 over the block train | `mbt.go` (`ValidateMBTData`, `mbtCRC32`) |
| AMBT broadcasts | Network/RFSS/Adjacent Status with uplinks | `mbt.go` (`ParseMBTNetworkStatusBroadcast` et al.) |
| Dispatch + labelling | decode, census, never mislabel | `control.go` (`dispatchMBT`, `ambtOpcodeLabel`) |
| Neighbour merge | TSBK + AMBT halves of one site | `network.go` (`NetworkModel.upsertNeighbor`) |

## In this post

- **The frame that got thrown away** — the operator's twelve-to-one gap.
- **Anatomy of an MBT** — header, data blocks, and the reused trellis.
- **Two CRCs, two verdicts** — and the log line that looks like a paradox.
- **What only the AMBT carries** — WACN, uplinks, and the neighbour merge.
- **The uplink rule** — base + spacing, no transmit offset.

## The frame that got thrown away

Part 2 established the lock rule: only a TSDU proves a control channel, and
every other DUID gets a `non-control DUID` debug line. For voice DUIDs that
line is correct and boring. For DUID 0xC it was a quiet catastrophe: on the
reporting operator's system, the frames being logged as
`non-control DUID duid=PDU` spam *were* the network data — SDRTrunk sitting
on the same signal listed 12 neighbour sites with uplink frequencies inside
15 seconds, while GopherTrunk showed one neighbour and "No Network Status
Broadcast yet", with no WACN at all.

The explanation is a spec corner with real-world teeth. P25 defines the PDU
for packet data, but it also defines a **trunking-control SAP (61)** under
which a PDU carries trunking signalling that needs more room than one TSBK's
eight payload bytes: Multi-Block Trunking. In its Alternate form (AMBT,
format 0x17), it is how many systems — notably Motorola — transmit their
*richest* broadcasts: the Network Status Broadcast that is the sole carrier
of the WACN, and Adjacent Status Broadcasts with explicit downlink *and*
uplink channels. A decoder that drops DUID 0xC never surfaces most neighbour
sites, and on some systems never learns the WACN. The site-topology
machinery that consumes all this is
[Trunking Engine Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})'s
story, and Part 10 of this series returns to it; here we decode the frames.

## Anatomy of an MBT

An MBT rides the same physical layer as everything else in this series: each
block is 196 bits, 1/2-rate trellis coded and interleaved **exactly like a
TSBK**, so `DecodeMBTBlockChannel` simply reuses Part 3's deinterleave →
Viterbi → repack pipeline. What differs is the content: first a 12-octet PDU
header, then `BlocksToFollow` 12-octet data blocks.

```go
// internal/radio/p25/phase1/mbt.go (shape) — ParseMBTHeader
//	octet 0     : bits 4..0 = format (0x17 AMBT / 0x15 unconfirmed)
//	octet 1     : bits 5..0 = SAP (61 = trunking control)
//	octet 2     : MFID
//	octets 3-5  : AMBT: message-specific (LRA / System ID)
//	octet 6     : bits 6..0 = blocks to follow
//	octet 7     : bits 5..0 = opcode (AMBT only)
//	octets 10-11: header CRC (CCITT-16)
h.Format = info[0] & 0x1F
h.SAP = info[1] & 0x3F
h.MFID = info[2]
h.BlocksToFollow = info[6] & 0x7F
h.Opcode = Opcode(info[7] & 0x3F)
if framing.CRCCCITTAugmented(info) != 0 {
    return h, ErrMBTHeaderCRC
}
```

The layouts were cross-checked against **two independent decoders** — OP25's
`process_PDU` for the header extraction and both CRC algorithms, SDRTrunk's
`PDUHeader`/`AMBTC*` classes for per-field bit offsets — precisely so this
would not be another working model of the kind
[Part 2 of From Spec to Shipping]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
warns about. The AMBT carries its opcode in header octet 7 and message
fields split between header and first data block; the Unconfirmed form
(0x15) buries opcode and fields in the data blocks with different layouts,
and GopherTrunk deliberately **censuses it without decoding it** — counting
occurrences until a capture justifies a parser is the #764/#771 discipline,
the same instinct behind
[census everything]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Structure of a P25 Alternate Multi-Block Trunking message: a 12-octet header block with format, SAP, MFID, blocks-to-follow and opcode fields ending in its own CCITT-16 CRC, followed by one or two 12-octet data blocks whose concatenation ends in a CRC-32; every block rides the same 98-dibit TSBK trellis coding">
  <text x="20" y="26" fill="currentColor" font-size="11" font-weight="bold">one AMBT = header block + BlocksToFollow data blocks</text>
  <!-- header block -->
  <rect x="20" y="44" width="260" height="70" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="150" y="62" text-anchor="middle" fill="var(--accent)" font-size="10">HEADER — 12 octets</text>
  <text x="150" y="78" text-anchor="middle" fill="currentColor" font-size="9">format 0x17 · SAP 61 · MFID · blocks · opcode</text>
  <rect x="196" y="88" width="76" height="18" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="234" y="101" text-anchor="middle" fill="var(--accent)" font-size="9">CRC-CCITT16</text>
  <text x="106" y="101" fill="var(--fg-muted)" font-size="9">System ID / LRA / msg fields</text>
  <!-- data blocks -->
  <rect x="300" y="44" width="170" height="70" rx="6" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="385" y="62" text-anchor="middle" fill="currentColor" font-size="10">DATA BLOCK 0 — 12 octets</text>
  <text x="385" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WACN · downlink ch · uplink ch …</text>
  <rect x="490" y="44" width="170" height="70" rx="6" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="575" y="62" text-anchor="middle" fill="currentColor" font-size="10">DATA BLOCK 1 (optional)</text>
  <rect x="584" y="88" width="68" height="18" rx="3" fill="none" stroke="currentColor"/>
  <text x="618" y="101" text-anchor="middle" fill="currentColor" font-size="9">CRC-32</text>
  <line x1="304" y1="97" x2="582" y2="97" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="420" y="92" fill="var(--fg-muted)" font-size="9">CRC-32 spans ALL data octets</text>
  <!-- trellis note -->
  <line x1="20" y1="132" x2="660" y2="132" stroke="var(--fg-muted)"/>
  <text x="340" y="150" text-anchor="middle" fill="currentColor" font-size="10">every block: 98 channel dibits through the SAME 1/2-rate trellis + interleave as a TSBK (Part 3)</text>
  <text x="340" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="10">header CRC clean + data CRC failed ⇒ the log line's opcode/blocks/nac are REAL (they're the header's) —</text>
  <text x="340" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="10">only a data block is corrupt, and the repeating broadcast usually logs a clean copy nearby</text>
  <text x="340" y="216" text-anchor="middle" fill="var(--accent)" font-size="10">decodes fully — but never locks the channel: only a TSDU proves a control channel (Part 2)</text>
</svg>
<figcaption>An AMBT: one self-checked header block plus a CRC-32-terminated data train, all riding the TSBK trellis — two CRCs with separate verdicts, which is why one message can half-fail honestly.</figcaption>
</figure>

## Two CRCs, two verdicts

The header block ends in the same augmented CRC-CCITT16 a TSBK trailer uses;
the concatenated data blocks end in a **CRC-32** — polynomial 0x04C11DB7,
init 0, final XOR 0xFFFFFFFF, MSB-first over all data octets except the last
four (`mbtCRC32`, translated from OP25's `p25craft.py` derivation and
validated against `process_PDU`). Two codes, two independently checkable
units — and one log line that has repeatedly been misread as a parser bug:

```
p25: MBT data CRC failed opcode=NET_STS_BCST blocks=2 metric=… nac=…
     cause="corrupt data block (header decoded clean; identity fields above are the header's)"
```

Seeing that line with identity fields matching a *successfully decoded*
broadcast moments earlier looks like one frame both failing and passing.
It is **two different PDU frames**. Everything the failure line prints —
opcode, block count, NAC — comes from the header block, which carries its own
CCITT-16 and decoded clean; only a data block failed its CRC-32, from
ordinary residual RF errors. Broadcasts repeat many times a minute, so a
clean copy of the same message usually logs nearby. The line now carries the
worst data-block Viterbi metric and that explanatory `cause` precisely so
nobody chases a phantom parser bug from the pairing again — the CRC span was
re-verified byte-for-byte against OP25 before that conclusion was allowed to
stand.

The dispatch behind it (`dispatchMBT`) counts everything: `MBTDecoded`,
`MBTHeaderFailed`, `MBTDataCRCFailed`, and per-broadcast counters
(`NetStatusSeen`, `RFSSStatusSeen`, `AdjacentSeen`) that make "this system
only sends its WACN in AMBT form" a measurable statement instead of a hunch.
Unhandled AMBT opcodes log through `ambtOpcodeLabel` — numerically, per
Part 3's naming rule, because an AMBT opcode GopherTrunk doesn't decode must
never borrow a TSBK mnemonic.

## What only the AMBT carries

Three broadcasts get full AMBT parsers, each keyed to what the TSBK form
lacks:

| Broadcast | AMBT opcode | What the AMBT form adds |
|---|---|---|
| Network Status | 0x3B | **WACN** + explicit downlink *and* uplink control channels |
| RFSS Status | 0x3A | site identity with explicit uplink |
| Adjacent Status | 0x3C | neighbour's explicit uplink channel — often the *only* form sent |

The neighbour table merge is where the two worlds meet.
`NetworkModel.upsertNeighbor` keys on (RFSS, Site) and treats the TSBK and
AMBT forms as **complementary halves of one site**: the TSBK form carries
CFVA flags and service class, the AMBT form the explicit uplink — so a merge
keeps whichever half the new observation lacks instead of clobbering it with
zeros. That design fell out of the field data: systems that broadcast most
of their neighbour list only in AMBT form, and the rest only as TSBKs, are
both real.

## The uplink rule

An explicit uplink channel pair resolves with deliberate simplicity:

```go
// internal/trunking/network_report.go (shape)
// An explicit uplink (the P25 AMBT adjacent-status form) wins; a
// pair-only explicit uplink resolves via plain base + number*spacing
// (no transmit offset — the uplink channel number already encodes the
// uplink frequency).
if hz := b.BaseHz + uint64(n.UplinkChannelNumber)*uint64(b.SpacingHz); /* … */
```

The temptation is to treat every uplink as downlink + `TxOffsetHz`, because
that is how a *derived* uplink works when only the downlink channel is known.
But an explicit uplink channel number already encodes the uplink frequency
through the band plan's base + spacing arithmetic — applying the transmit
offset on top would shift it a second time. Two resolution paths, chosen by
what the broadcast actually said: derived (downlink + offset) when the
uplink is absent, plain arithmetic when it's explicit. Part 5 unpacks the
identifier tables this arithmetic runs on — and what breaks when they're
wrong.

### How the thrown-away frame shaped the Go code

- **Decode ≠ lock.** `parseFrame` routes DUID 0xC through the full MBT
  pipeline but never sets the locked flag — only a TSDU proves a control
  channel, so richer decoding never weakened Part 2's rule.
- **Reuse the proven coding layer.** MBT blocks ride the TSBK trellis and
  interleaver unchanged; the new code is parsing and CRC conventions only,
  which kept the blast radius of a new frame type small.
- **Census before decode.** The Unconfirmed MBT is counted, not parsed —
  no spec-guessing without a capture to verify against.
- **Log lines carry their own epistemology.** The failure line says *which
  block* its fields came from, so the two-frames reading is in the log
  itself rather than in tribal memory.

## Where this goes next

Every grant in Part 3 and every AMBT channel here leaned on the same quiet
machinery: the identifier tables that turn a 4-bit ID plus 12-bit channel
number into hertz.
[Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
gives the band plan its own episode — IDEN_UP in its three flavours, the
base + spacing arithmetic, transmit offsets, and the failure mode where every
voice grant tunes to nothing.

## FAQ

**Is every PDU on a control channel Multi-Block Trunking?**
No — that's why the header gate exists. `IsTrunkingControl` requires SAP 61
(trunking control) and a known format (0x17 AMBT or 0x15 unconfirmed);
anything else is genuine packet data (SNDCP, IP) and takes the data path.
"Usually MBT" is the right prior on a control channel, but the SAP decides.

**Why doesn't an MBT lock the channel?**
Lock is a claim that *this frequency is a control channel*, and Part 2's
rule keys that claim to a validated TSDU. MBTs also appear in contexts a
scanner shouldn't camp on, and the TSDU test is cheap and constantly
refreshed — so MBT decoding feeds the network model without touching the
lock state machine.

**My site table shows one neighbour but the system has many. Is that this
bug?**
On current GopherTrunk, no — the AMBT path has decoded Adjacent Status
broadcasts since `mbt.go` landed. Check the decode-status counters:
`AdjacentSeen` climbing with `MBTDecoded` means the frames are arriving;
a neighbour list that still looks thin then usually reflects what the site
actually broadcasts, which Part 10's topology tooling can show you directly.

**What does `p25: unhandled AMBT opcode opcode=AMBT(0x05)` mean?**
The header decoded cleanly (SAP 61, valid CRC) but its opcode isn't one of
the three broadcasts GopherTrunk parses, so it was counted and skipped. The
numeric label is deliberate — per the naming rule, an undecoded AMBT opcode
never borrows a TSBK mnemonic that doesn't apply to it.

**What is a WACN and why does it only arrive this way on some systems?**
The Wide Area Communications Network ID — the top of P25's identity
hierarchy (WACN → System → RFSS → Site), and the key that makes a system
globally unambiguous. The TSBK Network Status Broadcast carries it too, but
some systems transmit that broadcast only in AMBT form — which is exactly
why dropping DUID 0xC left the WACN blank. Part 10 climbs the full
hierarchy.

## Series navigation

**Part 4 of 14** · ←
[Part 3: TSBKs — The Control Channel's Workhorse]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
· Next →
[Part 5: Channel Identifiers & Band Plans — From Channel Number to Hertz]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
