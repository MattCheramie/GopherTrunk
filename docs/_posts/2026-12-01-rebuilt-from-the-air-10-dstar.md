---
title: "Rebuilt From the Air, Part 10: D-STAR — Wrong Sync, Wrong FEC, Wrong Golay"
description: "GopherTrunk's D-STAR decoder was fabricated end to end: sync words no radio sends, an invented header FEC, the wrong CRC, a placeholder voice schedule and a Golay code that corrupted even clean AMBE frames. It was rebuilt to the MMDVM firmware's air format and pinned by that firmware's own output; no real D-STAR capture has decoded yet."
category: solution-postmortem
keywords: d-star decoder sdr, d-star header fec viterbi, d-star frame sync word, d-star crc x-25, ambe golay 23 12 mbelib, d-star slow data sync, mmdvm d-star format, d-star late entry, dpmr ambe golay, gophertrunk d-star
tags: [rebuilt-from-the-air, d-star, dpmr, ambe, self-consistent-trap, reference-implementations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 10
---

*Part 10 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
finished NXDN with two known-key captures. This part has no capture at all.
D-STAR was rebuilt from the firmware that the mode's hotspots and repeaters
run, and what the comparison found was not a layer or two: every constant
between the demodulated bit and the vocoder was wrong.*

> **TL;DR:** `internal/radio/dstar` was fabricated end to end
> ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)). The
> frame sync `0xEAA060` matched nothing a radio sends; the slow-data sync
> `0x552D16` was the right bytes unpacked in the wrong bit order; the header
> FEC (K=5, PN15, 22×30) was invented; the CRC was the unreflected CCITT one;
> the voice bit schedule was a sequential placeholder; and the voice FEC used
> `framing.GolayDecode23_12`, a *different* (23,12) Golay code from the one
> AMBE uses, so even a clean frame decoded wrong. The rebuild follows the
> MMDVM firmware: K=3 code, a 24-column interleave, the x⁷ + x⁴ + 1 scrambler
> (`framing.DStarPN`), CRC-16/X-25, both polarities. It is pinned by four
> header bursts from that firmware's own encoder and by its null AMBE frame,
> which now decodes with zero Golay corrections. dPMR had the same Golay
> defect. Status: reference-pinned; a real D-STAR capture is still the gate.

**Key takeaways**

- **A decoder can be wrong in every layer and still have one right number.**
  The old header chain also produced 660 bits. Nothing else in it matched.
- **Two codes with the same parameters are not the same code.** Both Golay
  decoders are valid (23,12,7) codes; a clean codeword of one looks like bit
  errors to the other, and gets "corrected".
- **Firmware output is an independent encoder.** Four header bursts and one
  null voice frame, produced outside the repository, replaced every round
  trip as the definition of correct.
- **A decoder's error count is not always evidence.** mbelib's convention
  counts corrected *data* bits, which a third of random frames pass.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — D-STAR scan-list channels; dPMR's share is [#1374](https://github.com/MattCheramie/GopherTrunk/issues/1374) |
| Symptom | none from a D-STAR radio — no capture exists; a `dstar` scan-list channel opened a call on any carrier and recorded nothing |
| Wrong theories | the scrambler and interleaver tables are a "follow-up calibration step"; the voice interleave is the only placeholder; the AMBE FEC is shared with DMR, so it is already right |
| Real cause | fabricated framing in every layer: sync words, header FEC, CRC, voice schedule — and a Golay(23,12) from the wrong family |
| Fix | MMDVM's receive patterns in `sync.go`; `framing/dstar_header.go`; CRC-16/X-25 in `header.go`; DSD's schedule and mbelib's Golay in `voice_ambe.go` |
| Diagnostic | the firmware's own encoder output and null AMBE frame as literal vectors; the old chain fails all eight header checks |
| Status | reference-pinned and synthetic-verified; not capture-verified; D-STAR and dPMR audio are experimental until a capture decodes |

## In this post

- **A decoder with nothing to decode** — an audit that started without a capture.
- **The wrong theory: calibration** — what the earlier post believed was left.
- **The diagnostic: the firmware's own bits** — vectors made outside the repository.
- **Sync, header and CRC as they are sent** — the real chain.
- **The Golay nobody suspected** — a clean frame, three corrections, and dPMR.
- **Joining a transmission** — two receivers, late entry, the wrong error count.

## A decoder with nothing to decode

Every NXDN part of this series began with a reporter's recording. D-STAR
began with a missing feature. The same issue had produced call gates for
System Fusion and NXDN scan-list channels — a channel opens on a decoded
frame, not on power — and D-STAR was next. A `dstar` channel still squelched
on in-channel power alone, so any carrier on the frequency opened a "dstar"
call that recorded nothing.

A call gate is built on sync words. Laid beside the receive patterns in the
MMDVM firmware — the code D-STAR hotspots and repeaters run — neither of
GopherTrunk's two D-STAR sync constants was a pattern a radio sends, in
either polarity. A gate built on them would have shut out every real
transmission; the comment at the top of `dstar_gate.go` records that the
hole "could not be closed until the D-STAR air format was rebuilt."

That turned a feature into an audit, and it did not stop at the sync words.

## The wrong theory: calibration

[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
described the package in September, and it was candid. The header FEC chain
was "self-consistent encode/decode pairs" whose match to the reference's
tables "is a follow-up calibration step". The voice chain's deinterleave was
a sequential split marked `PLACEHOLDER`. That was the whole list.

It is the same word the LoRa bit layer hid behind in
[Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }}).
"Calibration" assumes the shape is right and a table needs tuning. Here the
shape was wrong: a different constraint length, interleaver geometry,
scrambler polynomial and CRC — and, under the voice path, a defect nobody
had listed at all, because that post called the AMBE FEC "shared with DMR
and NXDN" and so presumed it correct.

Every test was green, for the reason this series keeps returning to: the
encoder that built each fixture and the decoder that read it shared one
invented layout — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).

## The diagnostic: the firmware's own bits

With no capture, something else had to play the role of the air. The
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
used literal constants from two decoders. D-STAR got something closer to a
transmitter: the MMDVM firmware's header encoder, with the host software's
CRC routine, was compiled outside the repository and run on four headers —
a CQ call, a repeater-routed call, an urgent call and one full of arbitrary
bytes. Its output, the on-air bits in transmission order, is committed as
`framing/testdata/dstar_header_mmdvm.txt`. The firmware is GPL; it was read
for protocol facts and used to *produce* vectors. No source is ported, and
GopherTrunk's interleave and scrambler are generated in code and checked
against the firmware's tables, not copied from them.

Two checks per burst — decode it to the header bytes with a zero path
metric, and re-encode those bytes to the same bits — make eight
(`TestDStarHeaderFECMatchesMMDVMEncoder`). The old chain failed all eight.

For voice the firmware supplied one more vector: the nine-byte null AMBE
frame it transmits for silence. A real AMBE frame has structure a decoder
can test without a vocoder — its first two sub-vectors must be exact Golay
codewords. Under the old schedule and Golay the null frame needed three
corrections. Under the right ones it needs none.

## Sync, header and CRC as they are sent

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Top row: a D-STAR transmission in time order, a bit-sync preamble, the frame sync, a 660-bit header, then 96-bit DV frames of 72 voice bits and 24 slow-data bits, where frame 0 and every twenty-first frame carry the slow-data sync, ending with a 48-bit end pattern. Bottom row: the header chain as five boxes, 41 bytes sent least significant bit first plus two tail bits, a K equals 3 rate one-half code giving 660 bits, an interleave of 24-wide rows sent by column, a scrambler x to the 7 plus x to the 4 plus 1, and the bits on air.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">a D-STAR DV transmission, in time order</text>
  <rect x="10" y="26" width="66" height="40" fill="none" stroke="var(--fg-muted)"/>
  <text x="43" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bit sync</text>
  <text x="43" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1010…</text>
  <rect x="76" y="26" width="78" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="115" y="43" text-anchor="middle" fill="var(--accent)" font-size="9">frame sync</text>
  <text x="115" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0x557650</text>
  <rect x="154" y="26" width="140" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="224" y="43" text-anchor="middle" fill="currentColor" font-size="9">header</text>
  <text x="224" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">660 bits</text>
  <rect x="294" y="26" width="120" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="354" y="43" text-anchor="middle" fill="currentColor" font-size="9">DV frame 0</text>
  <text x="354" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">72 voice · 24 sync</text>
  <rect x="414" y="26" width="120" height="40" fill="none" stroke="currentColor"/>
  <text x="474" y="43" text-anchor="middle" fill="currentColor" font-size="9">DV frames 1–20</text>
  <text x="474" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">72 voice · 24 data</text>
  <rect x="534" y="26" width="70" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="569" y="43" text-anchor="middle" fill="currentColor" font-size="9">frame 21</text>
  <text x="569" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">sync again</text>
  <rect x="604" y="26" width="66" height="40" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="637" y="43" text-anchor="middle" fill="var(--fg-muted)" font-size="9">end</text>
  <text x="637" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">48 bits</text>
  <text x="340" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">a slow-data sync (0xAAB468) every 21 frames = 2016 bits = 420 ms at 4800 bit/s</text>
  <text x="340" y="116" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">the header chain, transmit order</text>
  <rect x="12" y="128" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="68" y="147" text-anchor="middle" fill="currentColor" font-size="9">41 header bytes</text>
  <text x="68" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">LSB first + 2 tail = 330</text>
  <line x1="124" y1="151" x2="140" y2="151" stroke="currentColor"/>
  <polygon points="140,147 148,151 140,155" fill="currentColor"/>
  <rect x="148" y="128" width="112" height="46" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="204" y="147" text-anchor="middle" fill="var(--accent)" font-size="9">K=3, rate ½</text>
  <text x="204" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">330 → 660 bits</text>
  <line x1="260" y1="151" x2="276" y2="151" stroke="currentColor"/>
  <polygon points="276,147 284,151 276,155" fill="currentColor"/>
  <rect x="284" y="128" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="340" y="147" text-anchor="middle" fill="currentColor" font-size="9">interleave</text>
  <text x="340" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">24-wide rows, by column</text>
  <line x1="396" y1="151" x2="412" y2="151" stroke="currentColor"/>
  <polygon points="412,147 420,151 412,155" fill="currentColor"/>
  <rect x="420" y="128" width="112" height="46" fill="none" stroke="currentColor"/>
  <text x="476" y="147" text-anchor="middle" fill="currentColor" font-size="9">scramble</text>
  <text x="476" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">x⁷ + x⁴ + 1</text>
  <line x1="532" y1="151" x2="548" y2="151" stroke="currentColor"/>
  <polygon points="548,147 556,151 548,155" fill="currentColor"/>
  <rect x="556" y="128" width="112" height="46" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="612" y="147" text-anchor="middle" fill="currentColor" font-size="9">on air</text>
  <text x="612" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">660 bits after the sync</text>
  <text x="340" y="198" text-anchor="middle" fill="currentColor" font-size="9">bytes 39–40: CRC-16/X-25 over bytes 0–38, low byte first</text>
  <text x="340" y="218" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the old model: K=5 punctured code, PN15 scrambler, 22 × 30 interleave, unreflected CRC — also 660 bits</text>
</svg>
<figcaption>The real transmission and its header chain; the invented chain agreed with it on the header's length and on nothing else.</figcaption>
</figure>

**The sync words.** `sync.go` used to carry `0xEAA060` and `0x552D16`,
unpacked most significant bit first. The first matched nothing. The second
was nearly a tragedy of bit order: `0x55 0x2D 0x16` really are the slow-data
sync *bytes*, and D-STAR sends every byte least significant bit first, so
the pattern on air is `0xAAB468`. The constants are now the firmware's
receive patterns, which it matches bit by bit as they arrive, so reading
them most significant bit first *is* time order: `FrameSyncHex` `0x557650`
(the tail of the 1010… preamble, the frame sync the transmitter sends as
`0xEA 0xA6`, and the four zero bits before the header), `SlowDataSyncHex`
`0xAAB468`, and a 48-bit `EndSyncHex`.

**Polarity.** The firmware reads a negative sample as a 1; GopherTrunk's
[GMSK]({{ '/reference/gmsk/' | relative_url }}) receiver slices the other
way, and radios differ. Neither sync is its own complement, so every
detector searches for the pattern and its inverse and learns the polarity
from which one matched. `TestProcessDecodesMMDVMHeaderBursts` feeds the
firmware's bursts both ways.

**The header FEC.** The 41 header bytes go out least significant bit first
with two zero tail bits — 330 bits — through a K=3, rate-½
[convolutional code]({{ '/reference/convolutional-code/' | relative_url }}):

```go
// internal/radio/framing/dstar_header.go (shape)
var d1, d2 byte
for i := 0; i < DStarHeaderInputBits; i++ { // 330
    var d byte
    if i < DStarHeaderInfoBits { // 328
        d = info[i/8] >> uint(i%8) & 1 // each byte LSB first
    }
    coded = append(coded, d^d1^d2, d^d2) // g1 first, then g0
    d2, d1 = d1, d
}
for k, i := range dstarInterleave {
    out[k] = coded[i] ^ dstarHeaderPN[k]
}
```

The 660 coded bits fill rows of 24 — 27 full rows and a 12-bit 28th — and
go on air column by column, so the first twelve columns are 28 bits deep and
the rest 27. Then the whole section is XORed with the sequence x⁷ + x⁴ + 1
generates from the all-ones state (`framing.DStarPN`, period 127). Its first
24 bits also scramble every DV frame's slow data — the bytes
`0x70 0x4F 0x93`, which `TestDStarPN` checks.

The invented chain appended four tail bits, ran a K=5 code, punctured four
bits, applied a 15-bit PN and a 22×30 block interleave. It, too, arrived at
660 bits.

**The CRC.** The header checksum is CRC-16/X-25 — the reflected polynomial
`0x8408`, initial value and final XOR `0xFFFF` — carried low byte first.
GopherTrunk had the unreflected `0x1021` CRC with no final XOR, stored high
byte first. `TestComputeCRCKnownVector` now pins the catalogue check value
`0x906E` for `"123456789"` (the old function gives `0x29B1`) and the CRC
bytes of a header the reference host software sealed. One flag was wrong
too: FLAG1's urgent bit is `0x08`; the old code read `0x04`.

`dstar_fec_mode` now defaults to `"on"`. The earlier default, `"off"`, read
41 unencoded bytes after the sync — a format no radio sends.

## The Golay nobody suspected

After the header, a transmission is a run of 96-bit DV frames: 72 voice bits
and 24 slow-data bits, with the slow-data sync in frame 0 and every 21st
frame after. `VoiceChannel` anchors on that sync and free-runs between.

Two things map 72 voice bits to the 49-bit AMBE payload. The first is the
bit schedule — which on-air bit lands in which cell of the four sub-vectors
C0..C3. That was the admitted placeholder; it is now DSD's D-STAR schedule
(`dstarDW` / `dstarDX`, from an ISC-licensed table).

The second is the FEC on C0 and C1: a
[Golay]({{ '/reference/golay-code/' | relative_url }})(23,12) code, three
errors corrected per word. The old code called `framing.GolayDecode23_12`.
That function is a correct Golay decoder, built from the project's extended
Golay(24,12) by dropping the parity bit — and it is the wrong one. AMBE's
(23,12) code, the one mbelib implements, has a different generator. Both are
perfect codes with identical parameters, but a codeword of one is in general
not a codeword of the other, so the decoder finds the nearest word *in its
own code*, "corrects" bits that were never wrong, and returns different data
with a plausible error count. Even a clean frame decoded to the wrong
vocoder parameters.

```go
// internal/radio/dstar/voice_ambe.go (shape)
fr := dstarAMBEDeinterleave(frame) // on-air bit i → fr[dstarDW[i]][dstarDX[i]]
c0data, c0errs := dmrvoice.GolayDecode2312(c0cw) // mbelib's Golay(23,12)
ks := dstarC1Keystream(c0data)                   // C0 seeds C1's descramble
for j := 0; j <= 22; j++ {
    fr[1][j] ^= ks[23-j]
}
c1data, c1errs := dmrvoice.GolayDecode2312(c1cw)
```

The error compounds: C0's data seeds the pseudo-random sequence that
descrambles C1, so a wrong C0 scrambles C1 before its own decode.

`TestDVVoiceDecodesMMDVMNullFrameWithoutCorrection` is the pin. The
firmware's null frame — `9E 8D 32 88 26 1A 3F 61 E8`, bytes least
significant bit first — must decode with zero corrections in C0 and in the
descrambled C1, and re-encode to the same 72 bits. A wrong schedule manages
that about once in four million frames.

dPMR had the same defect in its own `voice_ambe.go`. It now uses the same
mbelib Golay, pinned by `TestTCHFrameDecodesDMRSilenceWithMbelibGolay`: the
reference host's DMR silence frame, laid out in dPMR's interleave, must
decode to the 49 bits the DMR decoder gives, with no correction. dPMR's
on-air interleave is still a placeholder, and says so.

## Joining a transmission

The gate that started all this is `conventional.DStarCallDetector`. It opens
on a header whose CRC matches, or on two slow-data syncs one or two cycles
(2016 bits, within two) apart in the same polarity — a lone sync at the
search tolerance turns up in noise about once a minute; a pair at that
spacing does not. The end pattern closes it at once.
[Squelch Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }})
has the design.

Building it exposed a receiver problem. A scanner lands on a carrier at an
arbitrary symbol phase, and from the phase half a symbol off its stable
point — one start in ten — the
[Mueller-Müller loop]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }})
hung for one or two sync cycles before pulling in. The gate therefore runs
two receivers, the second fed half a symbol later, so one always starts near
its best phase. The voice chain had the same hang and got the same answer
(`dstarVoiceLanes`): the first lane to decode owns the output, and the other
takes over only after five frames the owner missed.
`TestDStarVoiceLanesJoinFromEveryPhase` measures it on the current tree: a
lone receiver loses up to 1.42 s of a joined call (1.72 s inverted); the
pair loses 0.04 s.

One more piece of speech was being discarded. The cadence can only be
anchored on a slow-data sync, so a receiver that joins mid-cycle had already
demodulated up to 0.42 s of speech before it knew where frames began.
`VoiceChannel` now decodes those frames from history, newest first, for as
long as each is plausibly voice.

"Plausibly" needed care. The obvious test is the Golay decoder's own error
count, and it is the wrong one: mbelib's convention counts corrected *data*
bits, and a third of random frames pass it. The walk instead re-encodes each
payload and counts the on-air bits that differ; noise lands within
`retroMaxErrors` (2) of a codeword pair about once in 4000 frames
(`TestVoiceChannelLateEntryIgnoresNonVoice`).

## What this overturns

This part replaces the D-STAR and dPMR sections of
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }}).
That post gave the frame sync as `0xEAA060` and the slow-data sync as
`0x552D16`; the header FEC as a K=5 code, a PN15 scrambler and a 22×30
interleave behind a `FECOff` default; the CRC as CRC-CCITT `0x1021` pinned
to `0x29B1`; and both voice chains as correct FEC over a placeholder
interleave. All of that describes the model, not the air. The frame layout
it gave — a 41-byte header, 96-bit DV frames, a sync every 21 frames — was
right, and the D-STAR interleave is no longer a placeholder. dPMR's still
is. Its System Fusion section is corrected in
[Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }}).

## What we keep

- **"Calibration" is a claim about shape.** Before filing a gap as a table
  to tune, check that the constraint length, the geometry and the polynomial
  are the reference's.
- **An encoder you did not write is a capture you can make.** With no
  recording, run the reference transmitter's encoder and commit its output
  as literal vectors.
- **Same parameters, different code.** Pin FEC to a codeword the reference
  produced, not to a round trip.
- **Know what an error count counts.** Re-encode and compare on-air bits
  when the decoder's own number is not a distance.
- **Reference-pinned is not on-air verified.** Nothing here has met a real
  D-STAR signal; a capture is the gate.

## Where this goes next

D-STAR had no capture and one firmware. The last decoder in this series had
two reference decoders and — as it turned out — a real carrier sitting in
the repository's own samples.
[Part 11]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
starts the P25 Phase 2 rebuild: a grant that resolved to the wrong carrier,
twelve timeslots, and the two that swap.

## FAQ

**Can GopherTrunk decode D-STAR?**
The code now follows the format the MMDVM firmware transmits: the header
decodes through its real FEC and CRC, and voice frames reach the vocoder
through the right schedule and Golay code. It is pinned to that firmware's
output but has not decoded a real D-STAR capture, so audio is experimental.

**Why did a clean D-STAR voice frame decode to the wrong audio?**
The voice FEC used a Golay(23,12) decoder for a different code than the one
AMBE uses. Both correct three errors, but a valid codeword of one is
generally not a codeword of the other, so the decoder altered correct bits
and returned different data. D-STAR and dPMR now use the Golay code the DMR decoder
carries; dPMR's on-air interleave is still a placeholder.

**What are the real D-STAR sync words?**
In time order, as the MMDVM firmware matches them: `0x557650` for the frame
sync with its preamble tail and trailing zeros, `0xAAB468` for the slow-data
sync every 21 DV frames, and a 48-bit end pattern. Both polarities are
searched.

**How does GopherTrunk join a D-STAR transmission that is already under way?**
It anchors on the slow-data sync, then decodes from history the frames it
had already demodulated — up to 0.42 seconds — accepting each only if
re-encoding its payload reproduces the received bits within two. Two
receivers half a symbol apart keep a bad starting phase from costing a
second or more.

## Series navigation

**Part 10 of 14** · ←
[Part 9: NXDN III — The Scrambler, and the Key That Ranked First]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
· Next →
[Part 11: P25 Phase 2 I — Twelve Timeslots and the Two That Swap]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
