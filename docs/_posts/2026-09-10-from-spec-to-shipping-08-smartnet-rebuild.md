---
title: "From Spec to Shipping, Part 8: Case Study — Rebuilding SmartNet From Proven Decoders"
description: "The full #1143 arc as a method case study: a Motorola SmartNet decoder green on every synthetic test yet unable to lock any real system, diagnosed as fabricated framing and rebuilt from OP25 and trunk-recorder — 8-bit sync 0xAC, stride-19 deinterleave, inverted data, CRC-10."
category: deep-dives
keywords: motorola smartnet decoder, smartnet osw format, smartnet control channel 3600 baud, op25 rx_smartnet, trunk-recorder smartnetparser, smartnet deinterleave crc, smartnet band plan 800 mhz, rebuilding a broken decoder, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, smartnet, motorola, trunking, case-study, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 8
---

*Part 8 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
laid out the rules for tests that keep the power to call your code
wrong. This part is the campaign where every one of those rules — and
most of the series' others — got exercised at once: the Motorola
Type II / SmartNet decoder, shipped on framing that matched no real
reference, caught by an operator who could never lock, and rebuilt
constant-by-constant from decoders proven on air. It is the villain's
biggest scene and the method's clearest demonstration.*

> **TL;DR:** GopherTrunk's original SmartNet decoder framed on a
> **24-bit sync `0xA4D7AA`**, a 32-bit OSW and **BCH(64,16,11)** —
> a format matching **no real reference**, so every operator hunt ended
> in `cchunt: hunt failed` while every synthetic test stayed green
> ([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)).
> The rebuild (`internal/radio/motorola/`) ports the real format from
> OP25 `rx_smartnet` and trunk-recorder's `SmartnetParser`: **8-bit sync
> `0xAC`**, 84-bit frames trusted only when the *next* sync lands 76
> bits later, stride-19 deinterleave, `parity[i]=info[i]^info[i-1]`
> ECC, 27 data + 10 CRC bits with the data **inverted on the wire**
> (masks `0xCC38`/`0x0D5`, CRC complemented) — and a physical layer
> re-laid as **3600-baud 2-FSK at ±1.2 kHz** into an 18 kHz DDC with a
> slow DC tracker. Pinned by reference-literal tests plus a
> failing-first real-air regression; still awaiting the reporter's
> capture for on-air verification, and the post says so.

**Key takeaways**

- **A fabricated format is worse than a missing one.** Placeholder
  framing plus round-trip tests produced a decoder that *looked*
  finished for months — the most expensive state a feature can be in,
  because nothing flags it but an operator's silence.
- **Rebuild from implementations proven on air, not from prose.** OP25
  and trunk-recorder have decoded live SmartNet systems for years; every
  constant in the new decoder cites one of them, per
  [Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})'s
  first rule.
- **The physical layer was fabricated too.** Not just the framing:
  the modem was wrong (MSK-flavoured, wide DDC) — a reminder that a
  fabricated stack is fabricated all the way down, and the audit has to
  start at the antenna.
- **Green synthetics still prove nothing.** The rebuild's own tests are
  reference-pinned and failing-first — and the decoder is still marked
  unverified until a real capture decodes, because that gate
  ([Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }}))
  is the whole lesson.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame codec | sync + deinterleave + ECC + CRC-10 → 27-bit OSW | `internal/radio/motorola/frame.go` (`DecodeOSWPayload`) |
| Bracket framer | trusts a frame only when the next sync lands 76 bits later | `internal/radio/motorola/process.go` (`ControlChannel.Process`) |
| OSW semantics | 16-bit address + group bit + 10-bit command; 1–3-OSW sequences | `internal/radio/motorola/osw.go`, `control.go` |
| Band plans | channel number → frequency; 800 standard/rebanded/splinter, 900 | `internal/radio/motorola/bandplan.go` (`ParseBandPlan`) |
| Physical layer | 3600-baud 2-FSK ±1.2 kHz, DC tracker, M&M timing | `internal/radio/motorola/receiver/receiver.go` |
| Channel rate | 18 kHz DDC target (5 samples/symbol) | `internal/scanner/ccdecoder/ddc.go` (`motorolaDDCTargetRateHz`) |
| Reference pins | sync bits, interleave permutation, XOR masks vs OP25 literals | `frame_test.go`; real-air regression in `process_test.go` |

## In this post

- **The symptom** — green CI, and a hunt that never locks.
- **The diagnosis** — framing that matched no reference anywhere.
- **The real format** — the OSW wire chain, constant by constant.
- **The physical layer, re-laid** — 2-FSK, 18 kHz, and a DC tracker
  that earns its keep.
- **Pinning it down** — the tests that can catch this class of bug, and
  the gate still open.

## The symptom

The report was as undramatic as protocol bugs get: an operator
configured a SmartNet system, and GopherTrunk's control-channel hunt
cycled candidates forever — `cchunt: hunt failed — no control-channel
lock`, every pass. No decode errors, no CRC storms, no partial locks.
Meanwhile the repository's SmartNet tests were green, had always been
green, and covered encode, decode, corruption handling and grant
sequencing.

That combination — *zero* on-air function with *total* synthetic
success — is the signature
[From the Issue Tracker Part 17]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
taught us to dread, and it should now trigger a specific reflex: audit
the constants against an independent reference before touching the DSP.
A weak-signal problem degrades; a wrong-constant problem is binary. This
was binary.

## The diagnosis

The audit was short and brutal. The original package framed on a 24-bit
sync word `0xA4D7AA`, wrapped a 32-bit OSW in two BCH(64,16,11)
codewords, and none of it — not the sync, not the width, not the code —
appears in OP25, trunk-recorder, gr-smartnet, or the venerable
`mottrunk.txt` notes that all of them descend from. The framing had not
been mis-transcribed from a reference; it had been **fabricated**, and
the round-trip tests enshrined it: the encoder produced the invented
format, the decoder consumed it, and the two agreed perfectly about a
protocol that exists nowhere —
[Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})'s
trap at maximum severity. The new `frame.go` header states the history
in the code itself, so the next reader inherits the scar tissue along
with the constants.

SmartNet has no public standards document, which is how the vacuum
formed. But it has something
[Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
rates even higher than a spec: **two independent implementations proven
on live systems for years** — OP25's `rx_smartnet.cc/.h` and
trunk-recorder's `SmartnetParser` — that agree with each other. The
rebuild ported every constant and transform from them, cited line by
line.

## The real format

The genuine control channel is austere: 84-bit frames, back-to-back,
forever.

```go
// internal/radio/motorola/frame.go (shape)
const (
    SyncBits        = 8    // frame sync length
    OutboundSyncHex uint32 = 0xAC // 10101100
    PayloadBits     = 76   // coded payload after sync
    FrameBits       = SyncBits + PayloadBits // 84

    // Data rides inverted on the wire (rx_smartnet.h:
    // ID_XOR 0x33C7, CMD_XOR 0x32A).
    idXORMask  uint16 = ^uint16(0x33C7)        // 0xCC38
    cmdXORMask uint16 = ^uint16(0x32A) & 0x3FF // 0x0D5

    // CRC-10 registers (rx_smartnet.cc crc_check).
    crcInit uint16 = 0x0393
    crcOp   uint16 = 0x036E
    crcPoly uint16 = 0x0225
)
```

An 8-bit sync word is alarmingly short — random data matches it every
256 bits — and the reference design compensates structurally: frames
run back-to-back, so the framer **trusts a frame only when the next
frame's sync arrives exactly 76 bits after the previous one ended**
(`process.go`, a port of `rx_smartnet::rx_sym`; pinned by
`TestProcessRequiresBracketSync`). Bracketing syncs plus the CRC-10
make the short sync safe.

Inside the bracket, the 76 payload bits deinterleave at **stride 19** —
wire position `k + l*19` reads out to sequence position `k*4 + l` —
into 38 `(info, parity)` pairs with `parity[i] = info[i] ^ info[i-1]`;
two consecutive flipped parity syndromes pinpoint a flipped info bit
between them, a single-error-correcting convolutional parity check. The
first 37 corrected bits are 27 data + 10 CRC (the 38th pair is spare),
and the data is **inverted on the wire**: the address and command
fields un-invert through the XOR masks above, and the CRC field is
bitwise-complemented. The surviving 27 bits form one Outbound Status
Word — 16-bit address, one group bit, 10-bit command — and the
protocol's most striking economy: **there are no opcodes.** A command
value that falls inside the system's band plan *is* a voice channel
number; grants, system-ID broadcasts and extended functions span one to
three consecutive OSWs sequenced by a state machine (`control.go`),
with `0x308`/`0x30B` opening the multi-OSW sequences and `0x2F8` as
idle. Which also explains two config keys: `motorola_band_plan` selects
`800_standard`/`800_rebanded`/`800_splinter`/`900` (channel→frequency
formulas from trunk-recorder's `get_freq`), and the old `motorola_bch_mode`
is now accepted-but-ignored — it configured a FEC layer the real
protocol never had.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Side-by-side comparison of the fabricated and real SmartNet frame structures. Top, muted and crossed out: the fabricated frame with a 24-bit sync 0xA4D7AA and two BCH 64,16,11 codewords around a 32-bit OSW — matching no real reference. Bottom, accented: the real 84-bit frame with an 8-bit sync 0xAC, a 76-bit interleaved payload, and the next frame's sync bracketing it; beneath it the decode chain: stride-19 deinterleave, doubled-parity ECC, 27 data plus 10 CRC bits, XOR un-inversion.">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="11" font-weight="bold">fabricated (pre-#1143) — matched no real reference</text>
  <rect x="20" y="34" width="90" height="30" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="65" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">24-bit sync</text>
  <rect x="110" y="34" width="180" height="30" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="200" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BCH(64,16,11) codeword</text>
  <rect x="290" y="34" width="140" height="30" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="360" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">32-bit OSW</text>
  <rect x="430" y="34" width="180" height="30" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="520" y="53" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BCH(64,16,11) codeword</text>
  <line x1="20" y1="30" x2="610" y2="68" stroke="var(--fg-muted)"/>
  <line x1="20" y1="68" x2="610" y2="30" stroke="var(--fg-muted)"/>
  <text x="20" y="106" fill="var(--accent)" font-size="11" font-weight="bold">real air format (OP25 / trunk-recorder) — 84 bits, back-to-back</text>
  <rect x="20" y="116" width="60" height="32" rx="4" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="50" y="132" text-anchor="middle" fill="var(--accent)" font-size="9">sync</text>
  <text x="50" y="143" text-anchor="middle" fill="var(--accent)" font-size="9">0xAC</text>
  <rect x="80" y="116" width="430" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="295" y="136" text-anchor="middle" fill="currentColor" font-size="10">76-bit interleaved payload (38 info/parity pairs)</text>
  <rect x="510" y="116" width="60" height="32" rx="4" fill="none" stroke="var(--accent)" stroke-width="2" stroke-dasharray="5 3"/>
  <text x="540" y="132" text-anchor="middle" fill="var(--accent)" font-size="9">next</text>
  <text x="540" y="143" text-anchor="middle" fill="var(--accent)" font-size="9">sync</text>
  <text x="596" y="136" fill="var(--fg-muted)" font-size="9">…brackets</text>
  <text x="596" y="147" fill="var(--fg-muted)" font-size="9">the frame</text>
  <line x1="295" y1="148" x2="295" y2="170" stroke="currentColor"/>
  <polygon points="291,168 295,176 299,168" fill="currentColor"/>
  <rect x="60" y="178" width="150" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="135" y="196" text-anchor="middle" fill="currentColor" font-size="9">deinterleave, stride 19</text>
  <rect x="220" y="178" width="170" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="305" y="196" text-anchor="middle" fill="currentColor" font-size="9">ECC: parity[i]=info[i]^info[i−1]</text>
  <rect x="400" y="178" width="130" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="465" y="196" text-anchor="middle" fill="currentColor" font-size="9">27 data + 10 CRC</text>
  <rect x="540" y="178" width="120" height="28" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="600" y="191" text-anchor="middle" fill="var(--accent)" font-size="9">un-invert:</text>
  <text x="600" y="202" text-anchor="middle" fill="var(--accent)" font-size="9">^0xCC38 / ^0x0D5</text>
  <text x="340" y="236" text-anchor="middle" fill="var(--fg-muted)" font-size="10">every constant cites rx_smartnet.cc/.h — none is original to this project, by design</text>
</svg>
<figcaption>The fabricated frame and the real one: nothing survives the comparison — sync length, payload width, FEC and field encoding are all different, which is why no synthetic test overlap existed with reality.</figcaption>
</figure>

## The physical layer, re-laid

The audit couldn't stop at the framing, because a fabricated stack is
rarely fabricated in only one layer. The real control channel is
**3600-baud binary FSK at roughly ±1.2 kHz deviation** — not the
MSK-flavoured ±900 Hz modem the old path assumed — and the rebuild
mirrors trunk-recorder's proven `smartnet_fsk2_demod`: FM discriminator,
a slow DC tracker, a one-symbol boxcar filter, Mueller-Müller timing,
zero-threshold slicer (`internal/radio/motorola/receiver/`).

Two physical choices carry most of the value. The channel now
downconverts to an **18 kHz target** (5 samples/symbol,
`motorolaDDCTargetRateHz` in `ccdecoder/ddc.go`) instead of riding the
48 kHz C4FM-family default — whose ±24 kHz passband also admitted
25 kHz-spaced neighbours straight into the discriminator. And the DC
tracker is load-bearing, not hygiene: at ±1.2 kHz deviation, a few
hundred hertz of residual carrier offset is a *large* slicer bias —
`dcTrackSeconds = 0.1` holds the tracker long against the ~23 ms frame
so NRZ content is untouched, short enough to pull in oscillator drift,
and `TestReceiverToleratesCarrierOffset` pins the tolerance. Protocols
with wide deviation can shrug that bias off; a 3600-baud narrow-shift
FSK cannot, and knowing *which* kind you have is a physical-layer
reading skill straight from
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }}).

## Pinning it down

The new test suite is built so this class of failure cannot re-enter
quietly, layering exactly the nets from
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
and Part 7:

| Net | What it pins | Test |
|---|---|---|
| Reference literals | sync bits vs OP25's `SMARTNET_SYNC_MAGIC` | `TestOutboundSyncBitsMatchReference` |
| Reference literals | the documented interleave permutation, probed bit-by-bit | `TestDeinterleavePermutationMatchesReference` |
| Reference literals | `0xCC38` / `0x0D5` masks vs `ID_XOR`/`CMD_XOR` | `TestXORMasksMatchReference` |
| Failing-first, real-air | a real-format stream → lock + grant; old decoder decodes **nothing** | `TestProcessDecodesRealAirFormat` |
| Structure | chunk-boundary survival, bracket-sync requirement, ECC/CRC behaviour | `process_test.go`, `frame_test.go` |

The reference-literal tests are the only ones that can catch **constant
drift** — a round-trip can't, since both sides drift together. The
real-air regression is the failing-first proof: run against the
pre-#1143 decoder, it decodes zero OSWs from a stream in the format
real systems transmit.

And then the honest line, because the method demands it: **this decoder
is synthetic-green, not on-air-verified.** The #1143 reporter's
854.5625 MHz capture (Airspy R2, 3 MS/s cfile) is the outstanding gate,
and until it decodes, the rebuild is a strong hypothesis with excellent
provenance — nothing more. (One aside from that capture's metadata: its
"≈550.3 ppm" frequency-correction warning almost certainly reflects the
capture tool's 11 ms probe window latching a different
momentarily-strong carrier in the 3 MHz span, not a genuinely
wild oscillator — sample-count math says the file really is 3 MS/s.)

### How that principle shaped the Go code

- **Provenance in the header, history included.** `frame.go` opens by
  naming its sources *and* the fabricated format it replaced, so the
  next audit starts with the full story.
- **Semantics stay with their reference.** OSW field meanings cite
  OP25/trunk-recorder in `osw.go`; band-plan formulas cite
  `SmartnetParser::get_freq` — every layer's authority is greppable.
- **Obsolete config degrades loudly but harmlessly.**
  `motorola_bch_mode` logs "obsolete and ignored" instead of failing
  configs written against the old decoder.
- **The fixture encoder is the real format's inverse.**
  `EncodeOSWFrame` exists for tests — and emits frames back-to-back
  with the trailing sync, because Part 7's faithful-transmitter rule
  applies to the fix as much as it did to the bug.

## Where this goes next

SmartNet's constants were fabricated; the next part's were merely
*wrong* — and the wire had no way to say so.
[Part 9]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
takes the same discipline to RPC protocols, where `callSetAntenna = 600`
silently invoked a different handler on a real server for a release —
and the fixes are the same nets, aimed at a `.hpp` instead of a PDF.

## FAQ

**Why did every SmartNet test pass while no real system ever decoded?**
Because encoder and decoder shared the same invented format — the
self-consistent trap. Round-trip tests verify that two functions are
inverses, not that either matches the air; when both were authored from
the same fabricated constants, green was guaranteed and meaningless.

**Where does the real SmartNet framing knowledge come from if there's no
public spec?**
From implementations proven on live systems: OP25's `rx_smartnet` and
trunk-recorder's `SmartnetParser`, which descend from gr-smartnet and
the mottrunk.txt reverse-engineering notes. Two independent, on-air
lineages that agree — which
[Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
argues beats any single document, prose specs included.

**How can an 8-bit sync word possibly be reliable?**
Alone, it isn't — random bits match it every 256 positions. The real
design trusts a frame only when it is *bracketed*: the next frame's sync
must arrive exactly 76 bits after the last one ended, which the
back-to-back control channel guarantees, and the CRC-10 gates whatever
survives. Short sync, structural redundancy — a pattern worth
recognizing in other legacy formats.

**What does "data inverted on the wire" mean for the OSW?**
The 27 information bits are transmitted complemented. The decoder
un-inverts the address and command fields by XOR with `0xCC38` and
`0x0D5` (the complements of OP25's `ID_XOR`/`CMD_XOR` masks) and
compares against a complemented CRC field — three constants that,
wrong, produce plausible-looking garbage rather than errors, which is
why each is pinned against upstream literals.

**Is the rebuilt decoder verified on a real SmartNet system yet?**
Not yet, and the code says so. The framing is reference-pinned and the
regression suite fails against the old decoder, but per the #764/#771
discipline a green synthetic is not an on-air verdict — the reporter's
854.5625 MHz capture is the open verification gate, and the issue stays
open until it decodes.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Tests That Can Disagree With You]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
· Next →
[Part 9: Wire Protocols Without Schemas]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
