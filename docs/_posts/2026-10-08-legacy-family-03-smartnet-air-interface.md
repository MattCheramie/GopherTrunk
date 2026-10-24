---
title: "The Legacy Family End to End, Part 3: Motorola Type II — 3600-Baud OSWs, Sync 0xAC and the Band Plans"
description: "The Motorola Type II / SmartNet control channel bit by bit in GopherTrunk's motorola package — the 84-bit sync-bracketed frame, the stride-19 deinterleave and parity ECC, the CRC-10 registers, the inverted 27-bit OSW, the multi-OSW sequencer that turns two words into a grant, the four band plans, the 18 kHz front end, and the capture that would move it off the reference-pinned rung."
category: deep-dives
keywords: motorola type ii decoder, smartnet osw format, smartnet 3600 baud 2-fsk, smartnet sync 0xac, smartnet deinterleave crc-10, smartnet band plan 800 rebanded splinter 900, smartzone control channel, motorola_band_plan, op25 rx_smartnet port, gophertrunk legacy family
tags: [legacy-family-end-to-end, smartnet, motorola, trunking, band-plan, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 3
---

*Part 3 of **The Legacy Family End to End**, a 14-part deep dive into the
FM-era trunking generation and the AMBE-era narrowband and amateur modes
through GopherTrunk.
[Part 2]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
laid out the skeleton every legacy package shares. This part reads the
first protocol's flesh: the Motorola Type II / SmartNet control channel
in `internal/radio/motorola`, from the 8-bit sync to the band plan that
makes a 10-bit command into 854.5625 MHz. The story of how this package
came to be rebuilt from OP25 and trunk-recorder — issue
[#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143) — is told
as a method case study in
[From Spec to Shipping Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }});
this part does not retell it. It reads the bits.*

> **TL;DR:** A SmartNet control channel is **84-bit frames at 3600 baud
> over 2-FSK at ±1.2 kHz**, back to back forever: 8-bit sync `0xAC`
> (`OutboundSyncHex`), then 76 coded bits (`PayloadBits`). The framer
> trusts a frame only when the *next* sync lands 76 bits later. The
> payload deinterleaves at stride 19 into 38 (info, parity) pairs with
> `parity[i] = info[i] ^ info[i-1]`; two adjacent flipped syndromes
> correct one info bit. The first 37 corrected bits are 27 data + a
> CRC-10 (`crcInit 0x0393`, `crcOp 0x036E`, `crcPoly 0x0225`), all
> inverted on the wire — address `^ 0xCC38`, command `^ 0x0D5`, CRC
> complemented. The 27-bit **OSW** has no opcodes: a 10-bit command that
> `plan.IsChannel` accepts *is* a channel; `0x308`/`0x30B` open two-word
> sequences, `0x2F8` is idle. `motorola_band_plan` selects
> `800_standard` (851.0125 MHz + 25 kHz × ch), `800_rebanded`,
> `800_splinter` or `900`. The DDC runs at **18 kHz** (5 samples/symbol)
> with a 0.1 s DC tracker. Every constant is pinned to an OP25 or
> trunk-recorder literal; no real capture has decoded yet. Rung:
> **reference-pinned**.

**Key takeaways**

- **The bracket is the sync.** An 8-bit sync matches random bits every
  256 positions; it is safe because frames are back-to-back and the
  framer demands the next sync exactly 76 bits on.
- **No opcodes, only a band plan.** `BandPlan.IsChannel` is the
  discriminator of the whole OSW state machine; a wrong plan does not
  mis-tune a grant, it fails to recognise one.
- **A grant is two words.** The source radio rides the `0x308` opener,
  the talkgroup and channel the follower; single-word updates carry no
  source, so `sourceMemoryTTL` backfills it for 30 s.
- **Reference-pinned, not on-air verified.** Three literal tests catch
  constant drift; the #1143 reporter's 854.5625 MHz capture is the open
  gate.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame constants | `SyncBits` 8, `OutboundSyncHex` 0xAC, `PayloadBits` 76, `FrameBits` 84 | `internal/radio/motorola/frame.go` |
| Payload codec | `deinterleave76` → `eccDecode76` → `crc10` → fields, masks `idXORMask`/`cmdXORMask` | `frame.go` (`DecodeOSWPayloadDetail`) |
| Bracket framer | 84-bit ring, `inSync`, `count`; decode only when the next sync arrives | `process.go` (`ControlChannel.Process`) |
| OSW model | `Address` 16, `Group` 1, `Command` 10; `Talkgroup()`, `Encrypted()`, `Emergency()` | `osw.go` |
| Sequencer | `oswQueueDepth` 3, `stepLocked`, `CmdFirstNormal` 0x308 / `CmdFirstAlternate` 0x30B | `control.go` |
| Source backfill | `sourceMemoryTTL` 30 s, `resolveSource` (#1143) | `control.go`, `TestGrantSourceBackfilledOntoUpdates` |
| Band plans | `plan800` (standard / rebanded / splinter), `plan900`; `ParseBandPlan` | `bandplan.go`, `bandplan_test.go` |
| Front end | 18 kHz DDC, `dcTrackSeconds` 0.1, boxcar, M&M gain 0.05 | `receiver/receiver.go`, `ccdecoder/ddc.go` |

## In this post

- **The frame on the wire** — 84 bits, and why the bracket makes an 8-bit sync safe.
- **From 76 bits to 27** — deinterleave, parity ECC, CRC-10, inversion.
- **The OSW and its sequencer** — a channel number as the only opcode.
- **Four band plans** — the formulas, with the #1143 channel as the example.
- **The 18 kHz front end** — the DDC target and the DC tracker.
- **The rung and the gate** — pinned, unwired, missing.

## The frame on the wire

The package doc states the air interface in one sentence: "The control
channel transmits 84-bit frames (8-bit sync + 76 coded bits) at 3600
baud over ~±1.2 kHz binary FSK; each frame's payload deinterleaves and
error-corrects to one 27-bit Outbound Status Word." The constants in
`frame.go` are the whole framing:

```go
// internal/radio/motorola/frame.go
const (
    SyncBits        = 8
    OutboundSyncHex uint32 = 0xAC // 10101100
    PayloadBits     = 76
    FrameBits       = SyncBits + PayloadBits
    oswDataBits     = 27
    oswCRCBits      = 10
    eccBits         = oswDataBits + oswCRCBits // 37; the 38th pair is spare
)
```

An 8-bit sync on its own is unusable — random data matches `10101100`
once every 256 bit positions. The reference design, which `process.go`
ports from OP25's `rx_smartnet::rx_sym`, makes it safe structurally:
frames run back to back, so a frame is accepted only when *another* sync
arrives exactly `PayloadBits` after the previous one ended. The adapter
keeps a `syncReg uint8`, an 84-bit `ring`, and a `count` of bits since
the last accepted sync; when `count` reaches `FrameBits` the current
byte must be a sync again or framing drops. Any early sync inside a
frame is treated as payload, as the reference does. On a bracket match
the ring holds `[payload(76) | next sync(8)]`, and the 76-bit window
goes to `DecodeOSWPayload`. `TestProcessRequiresBracketSync` corrupts
the *second* frame's sync and asserts the first does not decode;
`TestProcessSurvivesChunkBoundaries` feeds one bit per `Process` call.

## From 76 bits to 27

Inside the bracket, `DecodeOSWPayloadDetail` runs three transforms, each
a port of a named OP25 function. The deinterleave reads wire position
`k + l*19` out to sequence position `k*4 + l` for `k` in 0..18 and `l`
in 0..3 — `TestDeinterleavePermutationMatchesReference` probes it one
set bit at a time against OP25's documented shuffle
`{1,20,39,58,2,21,40,59,…}`. The result is 38 pairs of `(info, parity)`
bits in which, on the wire, `parity[i] = info[i] ^ info[i-1]`:

```go
// internal/radio/motorola/frame.go (shape) — eccDecode76
syndrome[1] = raw[1] ^ raw[0]
for k := 2; k < PayloadBits; k += 2 {
    syndrome[k+1] = raw[k+1] ^ (raw[k]^raw[k-2])&1
}
for k := 0; k < eccBits; k++ {
    if syndrome[2*k+1]&1 != 0 && syndrome[2*k+3]&1 != 0 {
        dst[k] = ^raw[2*k] & 1 // two adjacent bad parities bracket one bad info bit
        flips++
    } else {
        dst[k] = raw[2*k] & 1
    }
}
```

A flipped info bit breaks the parity on both sides of it, so two
consecutive non-zero syndromes pinpoint it — a single-error-correcting
convolutional parity check. `TestECCCorrectsSingleInfoBitError` flips
wire bit 40 (sequence position 10, pair 5's info) and expects `flips == 1`;
`TestCRCRejectsCorruptPayload` flips a parity bit and a neighbouring info
bit, a pattern the pairwise syndrome miscorrects, and expects the CRC to
refuse rather than a wrong OSW.

The first 37 corrected bits are 27 data bits followed by a CRC-10, a
register port of `rx_smartnet.cc crc_check` — `crcInit 0x0393`, `crcOp
0x036E`, `crcPoly 0x0225`, stepped once per data bit — computed over the
data *as it appears on the wire, still inverted*. The received CRC field
is complemented before comparison, and the fields come out through XOR
masks that are the complements of OP25's `ID_XOR 0x33C7` and `CMD_XOR
0x32A`:

```go
Address: addr ^ idXORMask,       // idXORMask  = ^0x33C7 = 0xCC38
Group:   corrected[16]&1 == 0,   // group bit is 0 for a group address
Command: cmd ^ cmdXORMask,       // cmdXORMask = ^0x32A & 0x3FF = 0x0D5
```

Bits 0..15 are the address, bit 16 the group flag — zero means group —
and bits 17..26 the command. `TestXORMasksMatchReference` pins the two
masks as literals, the only kind of test that can catch a constant
drifting, since a round-trip through `EncodeOSWFrame` would drift with
it.

<figure class="lab-figure">
<svg viewBox="0 0 680 206" width="680" height="206" role="img" aria-label="An 84-bit SmartNet frame (sync 0xAC, 76-bit payload, next sync as bracket) decoded in three rows: stride-19 deinterleave into 38 info-parity pairs, then the 37 corrected bits split into a 16-bit address, group flag, 10-bit command and complemented CRC-10 with their XOR masks.">
  <rect x="20" y="18" width="50" height="28" rx="3" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="45" y="36" text-anchor="middle" fill="var(--accent)" font-size="9">sync 0xAC</text>
  <rect x="70" y="18" width="470" height="28" rx="3" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="305" y="36" text-anchor="middle" fill="currentColor" font-size="9">76 coded payload bits (PayloadBits)</text>
  <rect x="540" y="18" width="50" height="28" rx="3" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="565" y="36" text-anchor="middle" fill="var(--accent)" font-size="9">next sync</text>
  <text x="600" y="36" fill="var(--fg-muted)" font-size="8">= bracket</text>
  <line x1="305" y1="46" x2="305" y2="68" stroke="currentColor"/>
  <text x="330" y="62" fill="var(--fg-muted)" font-size="8">deinterleave76: seq[k*4+l] = wire[k + l*19]</text>
  <rect x="20" y="70" width="570" height="28" rx="3" fill="none" stroke="currentColor"/>
  <text x="305" y="88" text-anchor="middle" fill="currentColor" font-size="9">38 (info, parity) pairs · parity[i] = info[i] ^ info[i−1] · adjacent bad syndromes ⇒ flip info</text>
  <line x1="305" y1="98" x2="305" y2="120" stroke="currentColor"/>
  <text x="330" y="114" fill="var(--fg-muted)" font-size="8">eccDecode76 → first 37 corrected bits (pair 38 spare)</text>
  <rect x="20" y="122" width="240" height="28" rx="3" fill="none" stroke="currentColor"/>
  <text x="140" y="140" text-anchor="middle" fill="currentColor" font-size="9">Address · 16 bits · ^ 0xCC38</text>
  <rect x="260" y="122" width="40" height="28" rx="3" fill="none" stroke="currentColor"/>
  <text x="280" y="135" text-anchor="middle" fill="currentColor" font-size="8">Group</text>
  <text x="280" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="7">0 = group</text>
  <rect x="300" y="122" width="150" height="28" rx="3" fill="none" stroke="currentColor"/>
  <text x="375" y="140" text-anchor="middle" fill="currentColor" font-size="9">Command · 10 bits · ^ 0x0D5</text>
  <rect x="450" y="122" width="140" height="28" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="520" y="140" text-anchor="middle" fill="var(--accent)" font-size="9">CRC-10 · complemented</text>
  <text x="305" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="8">crc10: init 0x0393 · op 0x036E · poly 0x0225, over the 27 still-inverted data bits</text>
  <text x="305" y="194" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">27-bit OSW — no opcode field; a command inside the band plan IS a channel</text>
</svg>
<figcaption>One SmartNet frame from sync to OSW; every constant is pinned by a literal test, not a round-trip.</figcaption>
</figure>

## The OSW and its sequencer

An `OSW` is three fields — `Address uint16`, `Group bool`, `Command
uint16` — and a command is not an opcode. `osw.go` lists only five
control values, all from trunk-recorder's `SmartnetParser`: `CmdIdle
0x2F8`, `CmdGroupBusy 0x300`, `CmdEmergencyBusy 0x303`, `CmdFirstNormal
0x308` and `CmdFirstAlternate 0x30B`. Any command the band plan accepts
is a channel number. A single OSW is therefore not self-describing, and
`control.go`'s sequencer buffers `oswQueueDepth = 3` words before
interpreting the oldest. `stepLocked` mirrors trunk-recorder's
`process_osws` with the oldest word as `o2`:

- `o2` a group-addressed channel → a **voice update**: a grant with no
  source that keeps a call alive and lets a scanner join late.
- `o2` an individual channel with address `0x1Fxx` → a CC broadcast
  beacon; consumed.
- `o2 == 0x308`, `o1` a channel with address `0x1Fxx` → **system ID +
  CC broadcast**, the lock signal: `observeSystemID(o2.Address)`.
- `o2 == 0x308`, `o1` a group channel, both addresses non-zero →
  **analog group voice grant**: `o2.Address` is the source radio, `o1`
  the talkgroup and channel.
- `o2 == 0x308`, `o1` an individual channel → a private or interconnect
  call, logged at debug, not recorded.
- `o2 == 0x30B`, `o1.Address & 0xFC00 == 0x2800` → the alternate lock
  form; `0x6000` → an alternate or adjacent CC whose low 10 address bits
  are its channel, folded into the topology as `NeighborSite{LCN,
  Adjacent: o1.Group}`.

`publishGrant` strips the group address's low nibble — `Talkgroup()` is
`Address & 0xFFF0` — and reads the status flags from it: `Encrypted()`
is bit `0x8`, `Emergency()` is option 2, 4 or 5 in the low three bits.
The grant carries `Protocol: "motorola"`, `ChannelNum` as the raw
command and `FrequencyHz` from the plan.

One subtlety came from the #1143 reporter's screen: SmartNet sends the
calling radio only on the two-word grant, and every single-word update
omits it, so the Active Calls source blanked after one frame.
`resolveSource` keeps a per-talkgroup `srcMemo` for `sourceMemoryTTL` =
30 s, refreshed on every hit and aged out so a stale talker is never
attached to a later source-less call — pinned by
`TestGrantSourceBackfilledOntoUpdates`, `TestGrantSourceMemoryAgesOut`
and `TestGrantSourceMemoryFollowsNewTalker`.

## Four band plans

`bandplan.go` ports trunk-recorder's `SmartnetParser::get_freq` and
`is_chan`, in integer hertz. `ParseBandPlan` accepts `""`, `800`,
`800_standard` or `800_domestic` for the default; `800_reband` or
`800_rebanded`; `800_splinter`; and `900`. An unknown name returns the
standard plan with `ok=false`, which the factory warn-logs.

The 800 MHz plans share their upper segments and differ below:

```go
// internal/radio/motorola/bandplan.go (shape) — plan800.Frequency, spacing 25 kHz
case ch >= 0x2D0 && ch <= 0x2F7: return 866_000_000 + spacing*(ch-0x2D0)
case ch >= 0x32F && ch <= 0x33F: return 867_000_000 + spacing*(ch-0x32F)
case ch >= 0x3C1 && ch <= 0x3FE: return 867_425_000 + spacing*(ch-0x3C1)
case ch == 0x3BE:                return 868_975_000
// per variant:
rebanded: ch <= 0x1B7 → 851_012_500 + spacing*ch; 0x1B8..0x22F → 851_025_000 + spacing*(ch-0x1B8)
splinter: ch <= 0x257 → 851_000_000 + spacing*ch; 0x258..0x2CF → 866_012_500 + spacing*(ch-0x258)
standard: ch <= 0x2CF → 851_012_500 + spacing*ch
```

The 900 MHz plan is one line: `ch <= 0x1DE` → 935.0125 MHz + 12.5 kHz × ch.
`bandplan_test.go` pins the values against trunk-recorder: channel
`0x08E` is **854.5625 MHz** — the #1143 reporter's control channel —
`0x1A5` is 861.5375 MHz, and `0x2F8`, `0x308`, `0x30B`, `0x320` and
`0x3FF` must *not* read as channels. That last assertion is the
load-bearing one: with no opcode, `IsChannel` is what separates "go to
854.5625 MHz" from "idle".

## The 18 kHz front end

The receiver mirrors trunk-recorder's `smartnet_fsk2_demod`: FM
discriminator, slow DC tracker, one-symbol boxcar, Mueller-Müller,
zero-threshold slicer. `SymbolRate` is 3600 and `DeviationHz` 1200; the
production DDC delivers 18 kHz, so `boxTaps` is 5 and the clock loop
runs at 5 samples per symbol with the default gain of 0.05.

Two choices carry the design. The first is the channel rate:
`motorolaDDCTargetRateHz` = 18000 in `ccdecoder/ddc.go` exists because
at the 48 kHz C4FM-family target the DDC's ±24 kHz passband "admits the
adjacent 25 kHz-spaced channels into the FM discriminator; at 18 kHz the
±9 kHz passband rejects them." The second is the DC tracker. At ±1.2 kHz
deviation a few hundred hertz of residual carrier offset is a large
slicer bias — `TestReceiverToleratesCarrierOffset` puts a 250 Hz offset
at roughly 21 % — so `dcTrackSeconds` = 0.1 subtracts a one-pole mean of
the discriminator: long against the ~23 ms frame so NRZ content is
untouched, short enough to pull in drift.
`TestReceiverDecodesModulatedControlChannel` modulates the real format
at 18 kHz and expects at least 15 CRC-clean OSWs; the offset test expects
the same with 250 Hz applied.

`TestMotorolaPipelineDecodesThroughProductionDDC` closes the loop at the
daemon's rate: a 90 kHz stream through `NewDownconverter(90000, 18000)`
in 4096-sample chunks, expecting a lock on system `0x4567` and a grant
for talkgroup `0xB010` at 861.5375 MHz — the configuration the pre-#1143
tests never exercised, since they synthesised at a bespoke 97.2 kHz.
`TestDaemonCCDecodesMotorola` repeats it through the whole daemon.

## The rung and the gate

Four literal tests catch constant drift — sync bits
(`TestOutboundSyncBitsMatchReference`), the interleave permutation
(`TestDeinterleavePermutationMatchesReference`), the XOR masks
(`TestXORMasksMatchReference`) and the band-plan frequencies
(`TestBandPlan800Standard` and siblings) — and
`TestProcessDecodesRealAirFormat` is the failing-first regression: a
real-format stream locks and grants here, and decoded nothing on the
pre-#1143 decoder.

What is not wired, per the package doc: Type I fleet/subfleet decoding,
OBT custom VHF/UHF band plans, ASTRO digital voice channels, and the
long tail of extended-function OSWs — the `0x30B` sequencer branch
consumes those pairs and does nothing. `motorola_bch_mode` is accepted
and ignored with an INFO line, because it gated a BCH(64,16,11) layer
the real air interface never had.

And the rung. Every constant cites a decoder proven on live systems and
the regression suite fails against the old code — **reference-pinned**,
one rung below capture-pinned. The capture that would move it is the
#1143 reporter's 854.5625 MHz, Airspy R2, 3 MS/s cfile, unreachable from
the development environment's network policy when the rebuild landed and
still the open gate. Until it decodes, the decoder is what its own method
case study called it: a strong hypothesis with excellent provenance.

## Where this goes next

SmartNet's control channel is austere by design — one word, no opcodes,
a band plan as the parser. EDACS went the other way: a 9600-baud GFSK
channel of 40-bit Control Channel Words with a real command nibble, a
site ID, adjacent-site pointers and a shortened BCH per word.
[Part 4]({{ '/blog/deep-dives/legacy-family-04-edacs/' | relative_url }})
reads `internal/radio/edacs` — the two readings of the same 40 bits, the
BCH(40,28,2) generator and its 780-pair double-error search, the LCN map
that is never wired to a frequency in production, and the ProVoice grant
the composer declines.

## FAQ

**How is a SmartNet OSW frame structured in GopherTrunk?**
As 84 bits: an 8-bit sync `0xAC` and 76 coded payload bits
(`motorola.FrameBits`). The payload deinterleaves at stride 19 into 38
info/parity pairs, a convolutional parity check corrects single info-bit
errors, and the first 37 corrected bits are a 27-bit OSW plus a CRC-10,
all inverted on the wire. `DecodeOSWPayload` runs the chain.

**Why does the decoder need the next frame's sync before it trusts a frame?**
Because the sync is only 8 bits and random data matches it every 256
positions. `ControlChannel.Process` keeps an 84-bit ring and decodes a
frame only when another `0xAC` arrives exactly 76 bits after the last
one, pinned by `TestProcessRequiresBracketSync`.

**What does motorola_band_plan select, and what happens with a wrong plan?**
`ParseBandPlan` maps it to `800_standard` (851.0125 MHz + 25 kHz × ch),
`800_rebanded`, `800_splinter` or `900` (935.0125 MHz + 12.5 kHz × ch).
With no grant opcode, `IsChannel` is what recognises a grant at all; an
unknown name falls back to `800_standard` with a WARN.

**Is GopherTrunk's SmartNet decoder verified on a real system?**
No. It is reference-pinned: sync bits, interleave permutation, XOR masks
and band-plan frequencies are tested against OP25 and trunk-recorder
literals, and `TestProcessDecodesRealAirFormat` fails against the
pre-#1143 decoder. The reporter's 854.5625 MHz capture is the open gate.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Shared Skeleton]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
· Next →
[Part 4: EDACS — 9600-Baud Control Channel Words, BCH, and the LCN Map]({{ '/blog/deep-dives/legacy-family-04-edacs/' | relative_url }})
