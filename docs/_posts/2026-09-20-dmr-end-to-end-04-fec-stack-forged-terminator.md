---
title: "DMR End to End, Part 4: The FEC Stack & the Forged Terminator"
description: The error-correcting codes a DMR burst passes through in GopherTrunk — BPTC(196,96) with its stride-181 interleave, the (20,8,7) slot-type code, RS(12,9) with three context seeds, the two CRC masks whose convention hid a bug — and how a voice burst's AMBE bits once Golay-decoded to "Terminator with LC" and ended a live call mid-sentence.
category: deep-dives
keywords: dmr bptc 196 96, dmr rs 12 9 seeds, dmr csbk crc mask 5a5a, dmr golay 20 8 slot type, dmr terminator with lc, dmr false call ended, dmr fec decoder, dmr embedded lc bptc 128 72, gophertrunk dmr
tags: [dmr-end-to-end, dmr, fec, bptc, reed-solomon, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 4
---

*Part 4 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})
found the bursts by cadence; this part decides whether to believe them.
DMR stacks four codes and two CRCs between the dibits and a decision, and
the series thread surfaces in the stack's one structural weakness: the slot
type is a strong code on eight bits, so about a third of *any* twenty bits
decode to a valid type — and a voice burst carries AMBE bits exactly where a
data burst carries its slot type. That is how a live IPSC call ended
mid-sentence on 9 September.*

> **TL;DR:** A DMR data burst's 196 payload bits are one **BPTC(196,96)**
> codeword — a 13 × 15 product matrix of Hamming(15,11) rows and
> Hamming(13,9) columns behind a stride-181 interleave
> (`framing.DecodeBPTC196_96`). The 96 recovered bits are then verified by
> **RS(12,9)** over GF(2⁸) with a per-context parity seed — `0x96` header,
> `0x99` terminator, `0x6A` embedded LC (`framing.VerifyRS12_9`) — or by a
> **CRC-CCITT** with mask `0x5A5A` (CSBK) or `0x9696` (PI header), ETSI's
> `0xA5A5`/`0x6969` in the inverted convention. Above sits the **(20,8,7)**
> slot type; inside voice frames, **Golay(23,12)**. The **forged
> terminator**: a voice burst A has AMBE bits where a data burst has its
> slot type, and the (20,8) code decodes ~1/3 of arbitrary words to
> *something* — so about one voice burst A in twelve read as
> TerminatorWithLC and the single-call fallback ended the live call. Fix
> (`tier2/process.go`, `conventional.go`): slot types only from data-sync
> bursts, and an undecodable terminator must at least be a BPTC codeword.

**Key takeaways**

- **Every DMR decision has two gates.** BPTC corrects; RS or CRC *verifies*,
  because a product code can converge on the wrong codeword and report
  success.
- **The seeds are the routing check.** RS(12,9) parity is XORed with a
  context seed, so a header decoded as a terminator fails RS even when BPTC
  is clean — the code says which kind of block it is.
- **A CRC convention is a wire fact, not a style choice.** Masks
  `0x5A5A`/`0x9696` are ETSI's `0xA5A5`/`0x6969` un-inverted; the earlier
  reading rejected every real CSBK while passing every synthetic one.
- **A strong code on few bits is a weak filter.** 256 radius-3 spheres cover
  a third of 2²⁰, so parsing a slot type from a burst that has none forges a
  valid one often.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| BPTC(196,96) | 13 × 15 product code, stride-181 interleave, ≤ 5 passes | `internal/radio/framing/bptc.go` (`DecodeBPTC196_96`) |
| Slot type (20,8,7) | colour code + data type, t = 3, brute-force 256 | `framing/hamming20.go` (`HammingDecode20_8`) |
| RS(12,9,4) verify | GF(2⁸) 0x11D, g(x) roots α¹..α³, seeded parity | `framing/rs12_9.go` (`VerifyRS12_9`, `RS129Seed*`) |
| CSBK CRC | CRC-CCITT init 0 over 80 bits, mask 0x5A5A | `internal/radio/dmr/tier3/csbk.go` (`csbkCRCMask`) |
| PI header CRC | same CRC, mask 0x9696 (ETSI 0x6969 inverted) | `internal/radio/dmr/pi.go` (`ParsePIHeader`) |
| Embedded LC | BPTC(128,72): Hamming(16,11) rows, column parity, 5-bit checksum | `framing/embedded_bptc.go` (`DecodeEmbeddedLC`) |
| AMBE+2 FEC | Golay(23,12) on C0/C1, C0-seeded descramble | `internal/radio/dmr/voice/ambefec.go` (`DecodeAMBEFrame`) |
| Forged-terminator gates | data-sync-only slot types; BPTC-valid fallback | `tier2/process.go`, `conventional.go` (`handleTerminator`) |

## In this post

- **The stack, layer by layer** — which code guards which bits.
- **BPTC(196,96)** — a product code with a stride, decoded iteratively.
- **RS(12,9) and its three seeds** — verification that also routes.
- **Two CRC masks and a convention** — the bug a convention hid.
- **The forged terminator** — one voice burst A in twelve, ending a call.

## The stack, layer by layer

[SDR Internals Part 9]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }})
surveyed framing and FEC in general; DMR's stack is worth laying out
concretely, because each layer has a distinct job and a distinct failure:

| Layer | Code | Protects | Failure mode |
|---|---|---|---|
| slot type | (20,8,7) Hamming/Golay | colour code + data type | forges a valid type on non-data bits |
| payload | BPTC(196,96) | 96 info bits of a data burst | converges on a wrong codeword |
| LC integrity | RS(12,9,4) + seed | header / terminator / embedded LC | catches BPTC misdecode, names the context |
| CSBK / PI integrity | CRC-CCITT + mask | control blocks, privacy header | wrong mask convention rejects everything real |
| voice | Golay(23,12) ×2 | 24 of 49 AMBE payload bits | random words look "corrected" |

The
[Golay]({{ '/reference/golay-code/' | relative_url }}),
[Hamming]({{ '/reference/hamming-code/' | relative_url }}) and
[Reed–Solomon]({{ '/reference/reed-solomon-code/' | relative_url }})
reference pages cover the mathematics; the point here is the *pairing* —
every block that matters is guarded twice, a corrector below and a verifier
above.

## BPTC(196,96)

The two 98-bit payload halves of a data burst concatenate
(`Burst.PayloadBits`) into one Block Product Turbo Code word
([BPTC reference]({{ '/reference/bptc/' | relative_url }})) — a deinterleave
then a two-dimensional Hamming pass:

```go
// internal/radio/framing/bptc.go (shape)
// Deinterleave: deInter[a] = channel[(a*181) mod 196]; deInter[0] is the
// reserved R bit, cell (r,c) sits at deInter[r*15+c+1]. Rows 0..8 are
// Hamming(15,11,3); every column is Hamming(13,9,3).
func DecodeBPTC196_96(channel []byte) ([]byte, int) {
    deInter := InterleaveBPTC(channel)
    /* … fill the 13×15 matrix … */
    for pass := 0; pass < 5; pass++ {
        /* column pass: HammingDecode13_9 · row pass: HammingDecode15_11 */
        if !anyChanged { break }
    }
    return info, totalCorrected // -1 if any row/column stayed uncorrectable
}
```

Each Hamming component corrects one bit; iterating unlocks the other
dimension. `-1` means a row or column never converged, and every caller
reads that as "not a codeword" — the `bptcOK` distinction the
forged-terminator fix turns on. The layout is
pinned by `TestBPTCCanonicalLayoutGolden`, whose bit positions come from
MMDVM / OP25 / DSD, not the encoder in the same file — the
[literal-vector discipline]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
applied to a product code.

## RS(12,9) and its three seeds

BPTC reports its own success but cannot notice a systematic misdecode, so
the 96 recovered bits of a Voice LC Header, Terminator or embedded LC carry
a second check: 9 octets of Full Link Control plus 3 octets of Reed–Solomon
parity over GF(2⁸)
([RS(12,9) reference]({{ '/reference/dmr-rs-12-9/' | relative_url }})),
implemented verify-only with the de-facto on-air parameters:

```go
// internal/radio/framing/rs12_9.go (shape) — GF(2^8), 0x11D, α = 2
var (
    RS129SeedVoiceLCHeader = [3]byte{0x96, 0x96, 0x96}
    RS129SeedTerminatorLC  = [3]byte{0x99, 0x99, 0x99}
    RS129SeedEmbeddedLC    = [3]byte{0x6A, 0x6A, 0x6A}
)

func VerifyRS12_9(cw []byte, seed [3]byte) bool {
    /* un-XOR the parity octets with seed, then c(α^j) == 0 for j = 1, 2, 3 */
}
```

The seeds are the interesting part. ETSI XORs the parity with a
context-specific constant, so the same 9 FLC octets produce *different*
parity on a header and a terminator — RS routes as well as verifies. A burst
whose slot type claims
Terminator but whose payload is a header fails `VerifyRS12_9` with
`RS129SeedTerminatorLC` even though BPTC decoded it perfectly — the mismatch
`TestConventionalUndecodableTerminatorNeedsBPTC` uses to model a
weak-but-real terminator. The encoder is pinned to MMDVMHost's `CRS129` by
an independent reference multiply.

## Two CRC masks and a convention

CSBKs and PI headers skip RS and carry a 16-bit CRC-CCITT
([CRC-16/CCITT reference]({{ '/reference/crc-16-ccitt/' | relative_url }}))
over their leading 80 bits, XORed with a data-type mask
([CSBK CRC reference]({{ '/reference/dmr-csbk-crc/' | relative_url }})):

```go
// internal/radio/dmr/tier3/csbk.go (shape) — CRC-CCITT of the leading
// 80 bits, XOR 0x5A5A; verified against real off-air Aloha/Preamble bursts.
const csbkCRCMask uint16 = 0x5A5A
want := framing.CRCCCITTWithInit(info[:10], 0x0000) ^ csbkCRCMask
```

ETSI writes the CSBK mask `0xA5A5` and the PI-header mask `0x6969` against a
CRC whose output is *inverted*. `framing.CRCCCITTWithInit` is the plain
form, so the equivalent masks are the complements, `0x5A5A` and `0x9696`
(`piHeaderCRCMask`, `internal/radio/dmr/pi.go`). The lesson is in the code's
history: an earlier "init 0xFFFF, store the complement" reading **rejected
every real CSBK while passing every synthesised round-trip** — the assembler
encoded the same wrong convention, fixed only against off-air bursts. The
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its purest form. Part 5's embedded LC adds one more code, a BPTC(128,72)
with a 5-bit checksum (`framing.DecodeEmbeddedLC`) — same pattern.

## The forged terminator

Now the stack's structural gap, and the 9 September field report it
produced: an IPSC operator's live call was declared ended while the radios
kept talking. On their 442.3875 MHz captures (tg 11) a Terminator with LC
landed six bursts — 1728 dibits — before the next voice superframe of the
*same* over, and eleven grants covered five transmissions.

Two facts from Part 2 combine. Only data bursts carry a slot type; a voice
burst A has AMBE bits in those twenty positions. And the (20,8,7) decoder is
a nearest-codeword search over 256 codewords with radius 3: 256 × (1 + 20 +
190 + 1140) = 345,856 of 1,048,576 words — **about one in three arbitrary
words decodes to a valid slot type.** On the captures about one voice burst
A in twelve parsed as (cc 12, TerminatorWithLC).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A voice burst A framed by BS-Voice with AMBE bits in its slot-type positions; an arrow carries those twenty bits into the (20,8,7) decoder, whose 256 radius-3 spheres cover a third of the space, and out comes Terminator with LC. The old path reached the single-call fallback and released the live call; the new path has two gates: SyncIsDataAtPolarity refuses a voice sync, and the fallback requires a BPTC-valid payload.">
  <text x="12" y="24" fill="currentColor" font-size="10" font-weight="bold">voice burst A (sync BS-Voice)</text>
  <rect x="12" y="32" width="70" height="30" fill="none" stroke="var(--fg-muted)"/><text x="47" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">AMBE</text>
  <rect x="82" y="32" width="26" height="30" fill="none" stroke="var(--accent)"/><text x="95" y="51" text-anchor="middle" fill="var(--accent)" font-size="8">AMBE</text>
  <rect x="108" y="32" width="90" height="30" fill="none" stroke="currentColor"/><text x="153" y="51" text-anchor="middle" fill="currentColor" font-size="8">BS-Voice sync</text>
  <rect x="198" y="32" width="26" height="30" fill="none" stroke="var(--accent)"/><text x="211" y="51" text-anchor="middle" fill="var(--accent)" font-size="8">AMBE</text>
  <rect x="224" y="32" width="70" height="30" fill="none" stroke="var(--fg-muted)"/><text x="259" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">AMBE</text>
  <text x="153" y="76" text-anchor="middle" fill="var(--accent)" font-size="9">the 20 dibit positions a data burst uses for slot type</text>
  <path d="M 153 82 L 153 104 L 400 104" fill="none" stroke="var(--accent)"/>
  <polygon points="400,100 410,104 400,108" fill="var(--accent)"/>
  <rect x="414" y="86" width="254" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="541" y="102" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">HammingDecode20_8: nearest of 256 codewords</text>
  <text x="541" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="8">256 × 1351 words within radius 3 ≈ 1/3 of 2²⁰ decode to something</text>
  <path d="M 541 126 L 541 146" stroke="var(--accent)"/>
  <polygon points="537,146 541,156 545,146" fill="var(--accent)"/>
  <text x="541" y="172" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">cc 12 · TerminatorWithLC (~1 voice burst A in 12)</text>
  <text x="120" y="150" fill="currentColor" font-size="10" font-weight="bold">old path</text>
  <text x="120" y="166" fill="var(--fg-muted)" font-size="9">LC undecodable ⇒ "one call active, unambiguous" ⇒ releaseCall</text>
  <text x="120" y="222" fill="currentColor" font-size="10" font-weight="bold">new path — two gates</text>
  <text x="120" y="238" fill="var(--fg-muted)" font-size="9">SyncIsDataAtPolarity refuses the voice sync · fallback needs a BPTC-valid payload</text>
  <text x="541" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a real terminator repeats 50–170 times per hang time</text>
</svg>
<figcaption>The forged terminator: AMBE bits in the slot-type positions of a voice burst fall inside one of 256 radius-3 spheres a third of the time, and the old single-call fallback trusted the label without a payload to back it.</figcaption>
</figure>

The old slicer parsed a slot type on **every** sync match, and
`handleTerminator` had a fallback for a terminator whose Full LC would not
decode: with one call active, end it. Sound for a weak real terminator;
fatal for a forged one, whose payload is speech and never decodes. Every
twelfth voice burst A ended the live call, which the voice path could only
re-grant by late entry seconds later (Part 5). Two gates fixed it:

```go
// internal/radio/dmr/tier2/conventional.go (shape) — handleTerminator
dest, ok, bptcOK := c.terminatorDest(b)
if !ok {
    // Not a BPTC codeword ⇒ a forged slot type, not a weak terminator
    // (a real one repeats for the whole hang time, 50–170 copies).
    if !bptcOK {
        c.log.Debug("dmr/tier2: terminator without a BPTC-valid payload ignored", "cc", slot.ColorCode)
        return
    }
    if len(c.calls) != 1 {
        return // ambiguous: two calls active, leave to hangtime
    }
    /* … single active call: release it … */
}
```

The first gate is Part 2's: slot types are read only from data-sync bursts,
per polarity (`SyncIsDataAtPolarity`), so after the polarity lock a voice
burst A never reaches `ParseSlotType`. The second hardens the fallback: an
undecodable terminator ends a lone call only if its BPTC(196,96) block
decoded (`bptcOK`) — a real burst with an RS or FLC mismatch, not a
non-codeword wearing a label. With two calls active it releases nothing;
each voice chain's own terminator plus hangtime ends its call (Part 6).

Both gates are pinned failing-first in `conventional_lateentry_test.go`:
one weaves a burst A carrying `AssembleSlotType(cc 12, TerminatorWithLC)` in
its AMBE positions and asserts no release; the other feeds non-BPTC garbage
under a Terminator label (no release) then a BPTC-valid, RS-mismatching
terminator (release). On the operator's captures the CSBK and FEC failures
that had looked like noise — 7 in 120 s — were the same forgery, and are
gone. **Not** resolved from them: the separate cc = 7 CSBK-CRC-fail train on
443.2375 MHz, BPTC-clean and CRC-failing at a 30 ms cadence for ~4 s — a
real proprietary train Part 7 carries as an open question.

### How the forged terminator shaped the Go code

- **Every verifier returns two facts, not one.** `terminatorDest` reports
  `ok` (the LC named a destination) *and* `bptcOK` (the payload was a
  codeword at all), because "weak but real" and "not a burst" need
  different responses.
- **Reference vectors sit beside every code.** BPTC has a golden layout,
  RS(12,9) an independent-multiply encoder, the CRC masks on-air Aloha
  bursts — none from the package's own encoder.
- **Diagnostics are parked, not dropped.** A rejected terminator logs at
  Debug; the CSBK failure line is rate-limited yet carries `info_hex`, the
  instrument the next field log needs.

## Where this goes next

With the stack in place, the payloads it protects can be read.
[Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})
opens the Full Link Control — Voice LC Header, Terminator with LC, embedded
LC across bursts B–E — and the late-entry path that grants from two agreeing
embedded LCs when the header was lost, plus the ten-copy header train that
once looked like ten keyups.

## FAQ

**What error correction does DMR use?**
A stack: (20,8,7) Hamming/Golay on the slot type, BPTC(196,96) on data-burst
payloads, RS(12,9) parity on Full Link Control, CRC-CCITT with a data-type
mask on CSBKs and PI headers, BPTC(128,72) plus a 5-bit checksum on embedded
LC, and Golay(23,12) inside each AMBE+2 voice frame — all in
`internal/radio/framing`.

**Why does DMR need RS(12,9) on top of BPTC?**
Because BPTC reports its own success and cannot detect a systematic
misdecode. The RS parity — XORed with a seed of 0x96, 0x99 or 0x6A — both
verifies the recovered bits and confirms the block's context, so a header
cannot be mistaken for a terminator even when BPTC is clean.

**Why does GopherTrunk use CSBK CRC mask 0x5A5A when ETSI says 0xA5A5?**
ETSI specifies 0xA5A5 against an inverted CRC-CCITT. GopherTrunk's
`CRCCCITTWithInit` is the non-inverted form, so the equivalent mask is the
complement, 0x5A5A (0x9696 for the PI header's 0x6969). The earlier reading
rejected every real CSBK while passing every synthetic round-trip.

**How did a DMR voice burst fake a call terminator?**
Voice bursts carry AMBE bits where data bursts carry the slot type, and the
(20,8) decoder maps about a third of all 20-bit words to a codeword. Roughly
one voice burst A in twelve read as Terminator with LC and the single-call
fallback ended the live call. Slot types now come only from data-sync
bursts, and the fallback requires a BPTC-valid payload.

**Does a failed BPTC decode publish a decode error?**
For a Voice LC Header, yes — `handleVoiceHeader` publishes a
`KindDecodeError` with stage `StageVoiceHeaderBPTC` (or `StageVoiceHeaderRS`
on parity mismatch). CSBK and terminator failures on a parked channel are
Debug-only and rate-limited: between-beacon noise is expected, not a fault.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Two Slots, One Carrier — Repeater vs Simplex Cadence]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})
· Next →
[Part 5: Link Control, Embedded LC & Late Entry]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})
