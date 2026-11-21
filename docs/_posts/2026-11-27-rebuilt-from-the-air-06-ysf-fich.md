---
title: "Rebuilt From the Air, Part 6: System Fusion — A FICH of 100 Dibits, Not 100 Bits"
description: "GopherTrunk's System Fusion control channel locked on the frame sync and never granted a call: its FICH codec modelled 100 on-air bits where the air carries 100 dibits behind four Golay words, and nothing called it. The rebuild, pinned to MMDVMHost's own encoder output, also added V/D mode 2 voice; still unverified on air."
category: solution-postmortem
keywords: ysf fich decoder, system fusion fich golay, ysf fich interleave 20x5, yaesu c4fm decoder sdr, ysf v/d mode 2 voice, ysf dg-id grant, mmdvmhost ysf reference, ysf frame layout dibits, c4fm carrier offset afc, gophertrunk ysf
tags: [rebuilt-from-the-air, system-fusion, ysf, self-consistent-trap, reference-implementations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 6
---

*Part 6 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }})
replaced six invented layers under a correct LoRa chirp. This part is Yaesu
System Fusion, where the frame sync was right, the frame was the right
length, and the 100-dibit header in between was decoded by a codec built
for a field half that size.*

> **TL;DR:** A `ysf` control channel logged `ysf cc locked` and nothing else
> ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)). The
> FICH codec in `fich.go` / `fich_trellis.go` turned 48 bits into 100 on-air
> *bits* through a punctured K=5 trellis and a 10×10 interleave; the air
> carries 100 *dibits*: four Golay(24,12) words behind an unpunctured rate-½
> K=5 code and a 20×5 interleave. Its encoder shared the layout, so its tests
> passed, and nothing on the live path called it anyway. Both files are
> deleted. `ysf/airframe.go` (`DecodeAirFICH`, `DecodeVD2Voice`, `AirFramer`)
> follows MMDVMHost's layout, pinned by five on-air FICH vectors from that
> project's own encoder and by its YSF2DMR silence pair, which decodes to the
> same 49 bits through GopherTrunk's YSF and DMR decoders.
> `ControlChannel.Process` now grants on the DG-ID. V/D mode 2 only;
> reference-pinned and synthetic-verified; unverified on air.

**Key takeaways**

- **"One wiring step away" was wrong twice.** The codec had no caller, and
  the codec it would have called could not decode a real FICH.
- **The units were the tell.** A codec that emits 100 bits cannot fill a
  field that `frame.go` itself sized at 100 dibits.
- **A code's name is not its generator.** MMDVMHost's Golay(24,12) and the
  shared `framing.GolayEncode24_12` differ on 4,094 of 4,096 codewords.
- **A reference's test data can cross-check two of your decoders.** One
  AMBE+2 silence frame, carried as YSF and as DMR, has to decode to the same
  49 bits.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — digital conventional modes in the scan list; System Fusion was the last one added |
| Symptom | a System Fusion control channel logged `ysf cc locked` and never published a grant, a call or a recording |
| Wrong theories | the FICH codec is built and one wiring step from live; if a real FICH fails its CRC, the puncture schedule is the two lines to change; any C4FM modulator makes a valid YSF fixture |
| Real cause | fabricated framing — a FICH codec for 100 on-air bits where the air carries 100 dibits behind four Golay words — and no caller on the live path |
| Fix | `internal/radio/ysf/airframe.go`; grants from decoded FICHs in `control.go`; `fich.go` and `fich_trellis.go` deleted; carrier correction and a channel filter on the voice receiver |
| Diagnostic | building the voice chain against MMDVMHost: five on-air FICH vectors from its encoder, and the YSF2DMR silence pair through two decoders |
| Status | reference-pinned and synthetic-verified; V/D mode 2 voice only; not capture-verified; unverified on air |

## In this post

- **Locked, and then nothing** — a symptom the code guaranteed.
- **One wiring step, two puncture lines** — the theories the old code invited.
- **The diagnostic: a voice chain forces you to read the FICH** — and the unit that did not fit.
- **The FICH on the air** — Golay, convolutional code, 20×5 interleave, CRC.
- **Voice, grants and a receiver for real radios** — the silence pair, the DG-ID, AFC.
- **What this overturns** — Beyond Voice 13's System Fusion section.

## Locked, and then nothing

A [System Fusion]({{ '/reference/system-fusion-ysf/' | relative_url }})
frame is 480 dibits, 100 ms of 4800-baud
[C4FM]({{ '/reference/c4fm/' | relative_url }}): a 20-dibit frame sync word
`0xD471C9634D`, 100 dibits of FICH — the frame information channel that says
what the frame is — and 360 dibits of payload. The whole live path of
GopherTrunk's `ysf` control channel was: find the sync word, publish
`cc.locked`, log `ysf cc locked`, stop. No grant, no call, no recording —
on any signal, by construction.

This was documented behaviour, not a surprise.
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
described it in September: the live pipeline "locks on the FSW and stops",
while a complete FICH codec, `DecodeFICHOnAir`, and the grant logic,
`ProcessFICH`, existed with callers only in tests. When the scan list grew
digital modes in October, System Fusion was the one left out: the composer
had no YSF voice chain at all.

## One wiring step, two puncture lines

Two theories came with that state, both written down.

**The codec is built; only the join is missing.** Beyond Voice 13 put it as
a key takeaway — "YSF is one wiring step from a real decoder" — and the
evidence looked good: `TestFICHOnAirRecoversFromSingleBitFlip` corrected a
flip at every one of the 100 on-air positions. Slice the FICH region, call
the decoder, call `ProcessFICH`, done.

**If a real FICH fails its CRC, adjust the puncture schedule.** The codec
punctured positions 0, 1, 102 and 103 of a 104-bit trellis output and called
that MMDVMHost's schedule; the capture notes named a fallback, an alternative
spread puncture attributed to another decoder. That is the calibration theory
from
[Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }})
again: believe the structure, keep two lines ready to edit.

A third assumption sat underneath and surfaced later: that the generic C4FM
test modulator and the P25 one were interchangeable for building a YSF
signal.

## The diagnostic: a voice chain forces you to read the FICH

Nobody set out to audit the FICH. The task was `mode: ysf` on a scan-list
channel, which needs voice, and voice needs the FICH: it is the field that
says whether a frame is a header, voice or a terminator, and which of four
payload layouts follows. So the voice chain was written from MMDVMHost's
System Fusion code — its FICH, payload and convolution sources, read for
protocol facts only, nothing ported — and the first thing to compare was the
header.

The unit did not fit. The package's own `frame.go` sizes the field at
`FICHDibits = 100`. The codec in the same package produced
`FICHOnAirBits = 100` — a hundred *bits*, fifty dibits. The glossary entry
written from that code carried both numbers a few lines apart: "100 channel
bits (50 dibits)" in its summary box and "the next 100 dibits carry the
FICH" in its text. A codec cannot be right about a field it only half
fills, and that check needed no reference at all.

MMDVMHost then supplied what the other half of the field is for. The fix
did not start from the old codec: `airframe.go` is a new file, and the old
pair stayed in the tree a little longer — still feeding nothing — until the
control channel moved over and they were deleted.

## The FICH on the air

Both models begin the same way and end at the same field. Everything between
is different.

| Stage | Old model (deleted) | On the air |
|---|---|---|
| Information | 32 bits | 32 bits, MMDVMHost's fields — FI, CS, CM, BN, BT, FN, FT, Dev, MR, DT, DG-ID |
| Check | CRC-16, polynomial 0x1021, init 0 | the same CRC with the output inverted (`crc16GSM`) |
| Block code | none | four extended Golay(24,12) words, 12 bits each (`golay24Encode`) — 96 bits |
| Convolutional code | K=5 rate ½ over 48 bits plus 4 tail bits — 104 bits, 4 punctured | K=5 rate ½ over 96 bits plus 4 tail bits — 200 bits, nothing punctured (`viterbiK5`) |
| Interleave | 10×10 over 100 bits | 20×5 over 100 symbol pairs (`airFICHInterleave`) |
| On air | 100 bits | 100 dibits |

The number 100 appears in both columns, which is probably how it got
misplaced: on air it is the count of bits going *into* the convolutional
coder — 96 Golay bits and 4 tail bits — and the count of dibits coming out.

<figure class="lab-figure">
<svg viewBox="0 0 680 260" width="680" height="260" role="img" aria-label="A System Fusion frame of 480 dibits: a 20-dibit frame sync word, a 100-dibit FICH and a 360-dibit payload. Below, two encoding chains that both start from 48 bits. The deleted model adds 4 tail bits, runs a K equals 5 half-rate code to 104 bits, punctures 4 and interleaves 10 by 10, ending at 100 bits, half the field. The on-air chain forms four Golay 24 12 words of 96 bits, adds 4 tail bits to make 100, runs the K equals 5 half-rate code to 200 bits and interleaves 20 by 5, ending at 100 dibits.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one YSF frame: 480 dibits, 100 ms</text>
  <rect x="10" y="24" width="70" height="34" fill="none" stroke="currentColor"/>
  <text x="45" y="39" text-anchor="middle" fill="currentColor" font-size="9">FSW</text>
  <text x="45" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">20 dibits</text>
  <rect x="80" y="24" width="150" height="34" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="155" y="39" text-anchor="middle" fill="var(--accent)" font-size="9">FICH</text>
  <text x="155" y="52" text-anchor="middle" fill="var(--accent)" font-size="8">100 dibits = 200 bits</text>
  <rect x="230" y="24" width="440" height="34" fill="none" stroke="currentColor"/>
  <text x="450" y="39" text-anchor="middle" fill="currentColor" font-size="9">payload · 360 dibits</text>
  <text x="450" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="8">V/D mode 2: five sections of a 40-bit DCH and a 104-bit VCH</text>
  <text x="10" y="92" fill="var(--fg-muted)" font-size="9" font-weight="bold">deleted model — the codec stops at 100 bits, half the field</text>
  <rect x="10" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="60" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">32 info + CRC = 48</text>
  <rect x="122" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="172" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+ 4 tail = 52 bits</text>
  <rect x="234" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="284" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">K=5 rate ½ → 104</text>
  <rect x="346" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="396" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">puncture 4 → 100</text>
  <rect x="458" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="508" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">10×10 interleave</text>
  <rect x="570" y="100" width="100" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="620" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">100 bits · 50 dibits</text>
  <line x1="110" y1="114" x2="122" y2="114" stroke="var(--fg-muted)"/>
  <line x1="222" y1="114" x2="234" y2="114" stroke="var(--fg-muted)"/>
  <line x1="334" y1="114" x2="346" y2="114" stroke="var(--fg-muted)"/>
  <line x1="446" y1="114" x2="458" y2="114" stroke="var(--fg-muted)"/>
  <line x1="558" y1="114" x2="570" y2="114" stroke="var(--fg-muted)"/>
  <text x="10" y="168" fill="currentColor" font-size="9" font-weight="bold">on the air — 100 bits go into the coder, 100 dibits come out</text>
  <rect x="10" y="176" width="100" height="28" fill="none" stroke="currentColor"/>
  <text x="60" y="194" text-anchor="middle" fill="currentColor" font-size="8">32 info + CRC = 48</text>
  <rect x="122" y="176" width="100" height="28" fill="none" stroke="var(--accent)"/>
  <text x="172" y="194" text-anchor="middle" fill="var(--accent)" font-size="8">4 × Golay(24,12) → 96</text>
  <rect x="234" y="176" width="100" height="28" fill="none" stroke="currentColor"/>
  <text x="284" y="194" text-anchor="middle" fill="currentColor" font-size="8">+ 4 tail = 100 bits</text>
  <rect x="346" y="176" width="100" height="28" fill="none" stroke="currentColor"/>
  <text x="396" y="194" text-anchor="middle" fill="currentColor" font-size="8">K=5 rate ½ → 200</text>
  <rect x="458" y="176" width="100" height="28" fill="none" stroke="var(--accent)"/>
  <text x="508" y="194" text-anchor="middle" fill="var(--accent)" font-size="8">20×5 interleave</text>
  <rect x="570" y="176" width="100" height="28" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="620" y="194" text-anchor="middle" fill="var(--accent)" font-size="8">100 dibits</text>
  <line x1="110" y1="190" x2="122" y2="190" stroke="currentColor"/>
  <line x1="222" y1="190" x2="234" y2="190" stroke="currentColor"/>
  <line x1="334" y1="190" x2="346" y2="190" stroke="currentColor"/>
  <line x1="446" y1="190" x2="458" y2="190" stroke="currentColor"/>
  <line x1="558" y1="190" x2="570" y2="190" stroke="currentColor"/>
  <text x="340" y="236" text-anchor="middle" fill="var(--fg-muted)" font-size="8">both chains start from the same 48 bits; the deleted one had no block code and ended one factor of two short</text>
</svg>
<figcaption>The frame sync and the frame length were right; the header between them was encoded and decoded by a model that produced half as many channel bits as the field holds.</figcaption>
</figure>

The decoder reads the chain backwards:

```go
// internal/radio/ysf/airframe.go (shape)
func airFICHInterleave(i int) int { return (i%5)*40 + (i/5)*2 }

for i := range pairs { // 100 symbol pairs out of 200 bits
	n := airFICHInterleave(i)
	pairs[i] = [2]uint8{bit(n), bit(n + 1)}
}
dec := viterbiK5(pairs[:]) // 100 bits: four 24-bit words, then the tail
// each word: data, good := golay24Decode(cw); !good → reject the frame
if crc16GSM(info[:4]) != uint16(info[4])<<8|uint16(info[5]) {
	return AirFICH{}, false
}
```

The [Golay]({{ '/reference/golay-code/' | relative_url }}) stage holds the
trap that would have caught a careless rebuild. The repository already had
an extended Golay(24,12), `framing.GolayEncode24_12`, serving the P25 and M17
paths. MMDVMHost's code for the FICH is also an extended Golay(24,12) —
data, eleven parity bits from the generator x¹¹+x¹⁰+x⁶+x⁵+x⁴+x²+1, one
overall parity bit — and it is a different code. Data word `0x001` encodes
to `0x0018EB` in one and `0x001C75` in the other; across all 4,096 data
words the two agree on two codewords. Reusing the shared function would have
produced a FICH decoder that round-trips and rejects every real frame, which
is why `airframe.go` carries its own `golay24Encode` and
`TestGolay24MatchesMMDVMHost` pins seven codewords as literals.

`TestDecodeAirFICHReferenceVectors` is the main pin, and it is
[literal vectors]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
from an independent encoder: five 25-byte on-air FICHs produced by
MMDVMHost's own FICH encoder, compiled outside the repository — a header,
V/D mode 2 and V/D mode 1 communications frames (the latter with DG-ID 11),
a voice full-rate frame and a terminator. None was produced by GopherTrunk.
`TestDecodeAirFICHCorrectsAndRejects` adds three symbol errors to one and
still recovers it, and confirms that pseudo-random dibits fail rather than
decode as a plausible header.

## Voice, grants and a receiver for real radios

**V/D mode 2 voice.** In the mode Yaesu radios use by default, the 360
payload dibits are five 144-bit sections, each a 40-bit data channel then a
104-bit voice channel. `DecodeVD2Voice` deinterleaves each voice channel
(26 columns of 4), removes a 13-byte whitening sequence, majority-votes the
first 27 bits from three copies each and takes 22 more as sent: 49 bits, an
[AMBE+2]({{ '/reference/ambe-plus-2/' | relative_url }}) payload.

A layout that intricate needs a pin that is not GopherTrunk's own encoder,
and MMDVMHost's YSF2DMR converter supplies an elegant one. It carries the
same AMBE+2 silence frame twice — as the 13 on-air bytes of a YSF voice
channel and as the 9 on-air bytes of a DMR voice frame.
`TestDecodeVD2VoiceMatchesDMRSilence` decodes the first with the new YSF
code and the second with `dmrvoice.DecodeAMBEFrame`, the
[DMR decoder]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
that is already verified on air, and requires the same 49 bits. That one
equality checks the deinterleave, the whitening, the triplet order and the
bit order together, and it is why the recorder can map `ysf` straight onto
the `ambe2-dmr` vocoder. V/D mode 1 and Voice Full Rate are recognised from the
FICH and logged, not decoded.

**Grants.** `ControlChannel.Process` now runs an `AirFramer` beside the sync
detector:

```go
// internal/radio/ysf/control.go (shape)
for _, fr := range c.framer.Process(dibits, baseIdx) {
	if fr.FICHOK {
		c.ProcessAirFICH(fr.FICH) // header/comms → grant on DG-ID; terminator → clear
	}
}
```

A header or communications frame of a group call grants on the FICH's
DG-ID; a communications frame is late entry when the header was missed;
individual calls and data full-rate frames are skipped; a terminator clears
the dedup. The lock still follows the bare sync word.
`TestControlChannelGrantsFromOnAirFrames` is the failing-first pin — two
transmissions of real FICHs through `Process`, two grants; the old `Process`
publishes none.

**The fixture trap.** The first modulated fixture would not decode and
looked like a receiver bug. It was the fixture: YSF is root-raised-cosine
shaped, α 0.20, like DMR, so its test signal comes from
`demod.ModulateC4FM`. `demod.ModulateP25C4FM` shapes with the P25 transmit
filter instead, and through the YSF receiver it gives about 45 % dibit
agreement. Two C4FM modes at one symbol rate are not one waveform.

**Carrier offset.** The last gap was found by analogy. The reporter's NXDN
captures in
[Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
sat about 400 Hz off frequency, so the YSF and D-STAR voice receivers were
checked for the same weakness. Neither corrected the carrier and neither had
a channel filter. On synthetic streams D-STAR decoded nothing from about
150 Hz of offset and YSF failed from about 200 Hz — the four-level eye's
inner levels sit only ±600 Hz from centre. `newYSFVoiceReceiver` now turns on
`EnableAFC` and `EnableChannelFilter` (±6.25 kHz).
`TestYSFVoiceReceiverToleratesOffsetAndNeighbour` is a transmission 400 Hz
off with a same-power neighbour 12.5 kHz away: 0 of 40 voice frames before,
35 of 40 after, and either fix alone still decodes 0. The control-channel
pipeline's receiver has neither yet.

## What this overturns

The "System Fusion: a FICH codec waiting for its caller" section of
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
is superseded:

- `DecodeFICHOnAir`, `ProcessFICH`, `FICHOnAirBits`, the 10×10 interleave
  and the puncture set no longer exist. The codec was not waiting for a
  caller; it was waiting for a format.
- "The live pipeline locks on the FSW and stops" is no longer true: it
  grants on the DG-ID.
- "DCH voice — not decoded" in that post's figure is now V/D mode 2 decoded.
- Its FAQ answer on why a Fusion repeater locks but never records describes
  the old code.

The glossary entry for the
[YSF FICH]({{ '/reference/ysf-fich/' | relative_url }}) was written from the
same model. The D-STAR and dPMR halves of that post are overturned
separately, in
[Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }}).

What is verified: the FICH and V/D mode 2 layouts are **reference-pinned**
to MMDVMHost's encoder output and YSF2DMR's constants, and
**synthetic-verified** through the receiver, composer and control channel.
Nothing is capture-verified. The only System Fusion capture so far, on
11 November, was not System Fusion — an analog FM carrier with a test tone —
and what it taught is
[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }});
the call gate that came out of it is
[Squelch Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}).
The voice chain still logs itself as experimental, and the
[user guide]({{ '/user-guide-windows.html' | relative_url }}) says the same.

## What we keep

- **Check the units before the algorithm.** Bits against dibits, symbols
  against chips, bytes against nibbles: a codec's output length has to equal
  the field it claims to fill.
- **"Built but unwired" is unverified twice.** Code with no caller has not
  even met its own integration, let alone the air. The
  [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  is safest when nothing depends on the result.
- **Never reuse a code by its parameters.** (24,12) names a size. Pin the
  generator with literals from the reference, as the
  [SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
  did for its interleave and masks.
- **Look for the same payload in two formats.** A converter's constants —
  YSF2DMR's silence pair — tie a new decoder to one that is already proven.
- **Fixtures need the right pulse, and a real radio's error.** A YSF signal
  is not a P25 signal at the same baud, and a test stream on exact centre
  frequency hides a receiver that fails at 200 Hz.

## Where this goes next

System Fusion's sync word was right and its header was wrong. NXDN had the
sync word wrong too: sixteen bits where the air sends twenty, a SACCH two
dibits too long, and no descrambler — and this time a reporter's radios were
there to say so.
[Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
starts the three-part NXDN rebuild.

## FAQ

**Why did a System Fusion repeater lock in GopherTrunk but never record?**
The control channel's live path ran only the frame-sync detector. The FICH
decoder that should have named each frame existed but had no caller, and it
modelled a 100-bit FICH that no radio sends. Since v1.3.0 the control
channel decodes the real FICH and grants on its DG-ID.

**How is the YSF FICH actually encoded?**
32 information bits and an inverted CRC-16 make 48 bits, split into four
12-bit words. Each becomes an extended Golay(24,12) codeword; the 96 bits
plus 4 tail bits pass through a rate-½, K=5 convolutional code; the 200
output bits are interleaved 20×5 and sent as 100 dibits.

**Does GopherTrunk decode System Fusion voice?**
V/D mode 2 — the "DN" mode radios use by default — through the AMBE+2
vocoder, on a `mode: ysf` scan-list channel or a `ysf` system. V/D mode 1 and
Voice Full Rate are recognised and logged but not decoded.

**Is System Fusion decoding verified on air?**
No. The FICH and voice layouts are pinned to MMDVMHost's encoder output and
its YSF2DMR constants, and exercised end to end on synthetic signals. No
real System Fusion transmission has been decoded yet; a DN-mode capture is
the open gate.

**Why can't I reuse one Golay(24,12) decoder for every protocol?**
Because (24,12) fixes the code's size, not its generator or bit order.
MMDVMHost's FICH Golay and GopherTrunk's shared `framing.GolayEncode24_12`
agree on only two of 4,096 codewords, so the wrong one rejects every real
frame while passing its own round trip.

## Series navigation

**Part 6 of 14** · ←
[Part 5: LoRa — Six Invented Layers Under a Correct Chirp]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }})
· Next →
[Part 7: NXDN I — The Frame Sync Was Twenty Bits, Not Sixteen]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
