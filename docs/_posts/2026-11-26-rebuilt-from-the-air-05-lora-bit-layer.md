---
title: "Rebuilt From the Air, Part 5: LoRa — Six Invented Layers Under a Correct Chirp"
description: "GopherTrunk's LoRa receiver dechirped and synced correctly while its whitening, payload CRC, header, Hamming layout, interleaver and chip mapping were invented and shared with its own modulator. Two independent references, transcribed separately and identical on 2,916 cases, replaced the bit layer; no real LoRa frame has been decoded yet."
category: solution-postmortem
keywords: lora decoder bit layer, lora whitening sequence, lora explicit header checksum, lora diagonal interleaver formula, lora hamming codeword layout, lora reduced rate header block, gr-lora_sdr reference implementation, lora payload crc, self-consistent test trap, gophertrunk lora
tags: [rebuilt-from-the-air, lora, self-consistent-trap, reference-implementations, testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 5
---

*Part 5 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 4]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }})
rebuilt FLEX around a mode code that comes before its marker. This part is
the one rebuild in the series that had no capture and no bug report at all:
LoRa, where the chirp physics was right from the first commit and every
layer between a demodulated bin and a payload byte was made up.*

> **TL;DR:** `internal/radio/lora` shipped in v0.3.8 with a correct chirp
> layer — dechirp, fold, preamble and SFD sync — over an invented bit layer:
> whitening, payload CRC, explicit header, Hamming layout, diagonal
> interleaver and chip mapping, all shared with the package's own modulator,
> so every round trip passed
> ([#586](https://github.com/MattCheramie/GopherTrunk/issues/586)). For four
> months "bit-exact Semtech interop" sat on the follow-up list as a
> calibration step waiting for a golden capture that never came. Transcribing
> two independent references separately — gr-lora_sdr and jkadbear/LoRaPHY,
> identical on 2,916 random cases — showed the old code decoded **0 of 6**
> reference frames; its header checksum failed every time. The bit layer was
> replaced in v1.3.0 and is pinned by literal chip vectors in
> `bitexact_test.go`. Status: reference-pinned and synthetic-verified. No real
> LoRa IQ has been decoded.

**Key takeaways**

- **"Calibration" was the wrong word for the gap.** The follow-up list named
  three constants to tune; the diff found six layers with the wrong shape,
  including a header one nibble short and a first block that did not exist.
- **Two references beat one capture you do not have.** gr-lora_sdr and
  LoRaPHY were transcribed separately and agreed chip for chip on 2,916
  cases, which is what made a disagreement with GopherTrunk mean something.
- **A fabricated framing can sit under a verified physical layer.** The
  dechirp tests at 0 dB SNR were real evidence about chirps and no evidence
  about bits.
- **Reference-pinned is not on-air verified.** The chip vectors prove
  agreement with two decoders that work on Semtech radios; a capture is
  still the gate.

## Cheat sheet

| | |
|---|---|
| Issue | [#586](https://github.com/MattCheramie/GopherTrunk/issues/586) — LoRa decoding; "bit-exact Semtech interop" open as a follow-up from June to October |
| Symptom | none reported — every test green, the `/lora` panel wired, and no off-air frame ever fed to the decoder |
| Wrong theories | interop is a calibration step (whitening table, CRC seed, interleaver rotation sign); a golden packet will tune it; the reduced-rate header is a separate, later feature |
| Real cause | fabricated framing: whitening clocking, CRC construction, header layout, Hamming bit order, interleaver formula and chip mapping were all invented, and `LoRaModulate` encoded the same inventions |
| Fix | the Semtech air format in `whitening.go`, `header.go`, `fec.go`, `frame.go`; shipped in v1.3.0 |
| Diagnostic | two reference encoders transcribed separately, compared on 2,916 random cases, then compared with GopherTrunk: 0 of 6 frames, header checksum failing 100 % |
| Status | reference-pinned and synthetic-verified; not capture-verified; on-air gate open |

## In this post

- **Four months of green** — a decoder with no symptom, and what its tests proved.
- **The wrong theory: calibration** — three constants on a follow-up list.
- **The diagnostic: two references, transcribed separately** — and where they disagree.
- **The Semtech format, layer by layer** — six rows, one worked frame.
- **The pin, and the gate that is still open** — `bitexact_test.go` and what it cannot say.
- **What this overturns** — Beyond Voice 11's bit chain.

## Four months of green

Most parts of this series start with someone's radio not decoding. This
one starts with nothing. The [LoRa]({{ '/reference/lora/' | relative_url }})
receiver landed in v0.3.8 on 10 June 2026 as a complete feature: chirp
synthesis, a dechirp-and-fold FFT kernel, preamble and SFD acquisition, a
bit chain, a channelized streaming receiver, LoRaWAN MIC verification, a
`lora_log` table and a `/lora` panel. Low Data Rate Optimize followed in
v0.7.6. The tests were thorough in the way round-trip tests are thorough:
`TestModDemodClean` decodes every spreading factor 7–12 at every coding
rate, `TestModDemodNoise` decodes at 0 dB SNR, `TestModDemodCFO` survives
carrier offsets of several bins.

All of those drive `LoRaModulate` into the package's own `Demodulator`, and
the project said what that meant. The LDRO release note ended "the
reduced-rate SF-2 *header* region and bit-exact Semtech interop remain gated
on captured golden vectors";
[Beyond Voice Part 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
called the header "a compact, self-consistent header layout". A request for
a capture went out on the issue. None arrived.

That is the uncomfortable property of the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
when nobody owns the hardware: there is no symptom to chase.

## The wrong theory: calibration

The follow-up list named the gap precisely, and the precision was the
mistake. It read as three knobs — the whitening table, the CRC seed, the
interleaver's rotation direction — plus the reduced-rate header as a
separate feature. The glossary entry written from the same code called
bit-exact interop "a calibration step gated behind captured golden vectors".

Calibration is what you do to a structure you believe, and the belief was
reasonable. Each stage is a textbook object — a
[Gray code]({{ '/reference/gray-code/' | relative_url }}), a
[Hamming code]({{ '/reference/hamming-code/' | relative_url }}), a diagonal
interleaver, an LFSR whitener, a CRC-16 — and descriptions of LoRa's PHY
list those stages in that order. If the stages and the order are right,
what is left is sign conventions and seeds, and one golden packet settles
those in an afternoon.

What that skips is that a stage's *name* fixes almost nothing. A
Hamming(8,4) code has many bit layouts; a diagonal interleaver is a family;
"CRC-16 with polynomial 0x1021" does not say which bytes it covers. A
capture would have shown the header checksum failing on frame one, and
nothing about why.

## The diagnostic: two references, transcribed separately

The capture was never coming, so the question changed from "what does the
air carry?" to "what do decoders known to work on Semtech radios agree the
air carries?" Two open implementations qualify, independent of each other:

- **gr-lora_sdr**, the GNU Radio implementation validated against an SX1276
  — whitening table, header checksum, CRC, Hamming encoder, interleaver and
  Gray demapping.
- **jkadbear/LoRaPHY**, validated against an SX1278 — its `encode()` path.

Both were read for the format only; nothing was ported
([clean-room rules]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})).
Each was transcribed *separately* — only the resulting chip vectors are
committed — and the two were run against each other on 2,916 random
combinations of spreading factor, coding rate, CRC, LDRO and payload. They
produced identical chip sequences on all of them. That gives the next step
its weight: when two independent codebases agree 2,916 times, a third
implementation that disagrees is the one that is wrong
([choosing references]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})).

Then six reference frames went into GopherTrunk. The old decoder decoded
none. The header checksum failed on every one.

The two references do differ, in two places, and a pin that hid them would
be a second trap:

- **Padding.** When the last interleaver block is short, gr-lora_sdr sends
  zero codewords and LoRaPHY pads the byte stream with `0xFF`. No receiver
  reads them. GopherTrunk pads with zero codewords, and every committed
  vector is identical under both.
- **Receive rounding.** In a reduced-rate block gr-lora_sdr floors the
  divide-by-four; GopherTrunk rounds, for a reason the next section gets to.

## The Semtech format, layer by layer

Six layers were wrong. The table sets what the old code did — as Beyond
Voice 11 and the glossary documented it — against what both references send.

| Layer | What the old code did | What the air carries |
|---|---|---|
| Whitening | 8-bit LFSR from seed 0xFF, clocked eight times per payload byte | the same seed advanced **one bit per byte**, over the payload only — not the header, not the CRC (`Whiten`) |
| Payload CRC | a plain CRC-16 (polynomial 0x1021, init 0) over the payload | that CRC over all but the last two bytes, XORed with those two as a big-endian word; sent un-whitened, low nibble first (`PayloadCRC`) |
| Header | four nibbles — length, coding rate with the CRC flag in bit 3, an XOR checksum — in a block of its own | five nibbles — length high, length low, `CR<<1` plus the CRC flag, a 5-bit checksum — opening a reduced-rate 4/8 block of SF−2 rows that also carries the first SF−7 payload nibbles |
| Hamming(4+CR, 4) | a bit layout of its own | the nibble bit-reversed in the top four bits, parity below; 4/5 is one overall parity bit (`HammingEncode4`) |
| Interleaver | a diagonal with a different rotation and bit order | `symbol[i]` bit `j` = `codeword[(i−j−1) mod rows]` bit `i`, counted MSB first |
| Chip mapping | Gray-encode on transmit; reduced-rate bins at multiples of four | Gray-**decode** on transmit, ×4 in a reduced-rate block, then **+1**: a reduced-rate chip is always 4k+1 (`symbolToChip`) |

The whitening row shows how close a wrong layer can sit to the right one.
Both registers start at `0xFF`, so the first mask byte agrees. Run the old
routine as the glossary entry printed it and the sequence continues
`23 82 70 65`; the air continues `FE FC F8 F0 E1 C2 85`. A one-byte payload
would have whitened correctly. The old interleaver rule, for the record, was
`symbol[k]` bit `(r+k) mod ppm` = `codeword[r]` bit `k`.

The header row is the structural one. The old code gave the header its own
block; on air the first block is always coding rate 4/8 *and* reduced rate,
whatever the payload uses, and it is SF−2 codewords deep, so from SF8 up it
carries payload too. The "SF-2 header region" on the follow-up list was
never a separate feature. It was the reason no real header could parse.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A LoRa frame drawn left to right: preamble upchirps, two sync symbols and a 2.25-symbol SFD, marked as the chirp layer that already matched the references; then a first block of eight symbols at coding rate 4/8 with SF minus 2 rows, and payload blocks of 4 plus CR symbols, marked as the rebuilt bit layer. Below, the first block's nibbles: length high, length low, coding rate and CRC flag, two checksum nibbles, then SF minus 7 payload nibbles. At the bottom, the transmit chain from nibbles through Hamming, diagonal interleave, Gray decode, times four if reduced, plus one, to a chip value.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one LoRa frame: the chirp layer that was right, the bit layer that was rebuilt</text>
  <rect x="10" y="26" width="110" height="36" fill="none" stroke="var(--fg-muted)"/>
  <text x="65" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="9">preamble</text>
  <text x="65" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="8">base upchirps</text>
  <rect x="120" y="26" width="60" height="36" fill="none" stroke="var(--fg-muted)"/>
  <text x="150" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sync</text>
  <text x="150" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2 symbols</text>
  <rect x="180" y="26" width="70" height="36" fill="none" stroke="var(--fg-muted)"/>
  <text x="215" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SFD</text>
  <text x="215" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2.25 down</text>
  <rect x="250" y="26" width="180" height="36" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="42" text-anchor="middle" fill="var(--accent)" font-size="9">first block · 8 symbols</text>
  <text x="340" y="55" text-anchor="middle" fill="var(--accent)" font-size="8">always 4/8 · SF−2 rows · chips 4k+1</text>
  <rect x="430" y="26" width="240" height="36" fill="none" stroke="currentColor"/>
  <text x="550" y="42" text-anchor="middle" fill="currentColor" font-size="9">payload blocks · 4+CR symbols each</text>
  <text x="550" y="55" text-anchor="middle" fill="currentColor" font-size="8">SF rows · SF−2 rows with LDRO</text>
  <text x="130" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="8">chirp layer — matched both references all along</text>
  <text x="460" y="78" text-anchor="middle" fill="currentColor" font-size="8">bit layer — six invented stages, replaced</text>
  <text x="10" y="106" fill="currentColor" font-size="9" font-weight="bold">first block, SF−2 nibbles:</text>
  <rect x="150" y="92" width="64" height="24" fill="none" stroke="var(--accent)"/>
  <text x="182" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">len high</text>
  <rect x="214" y="92" width="64" height="24" fill="none" stroke="var(--accent)"/>
  <text x="246" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">len low</text>
  <rect x="278" y="92" width="74" height="24" fill="none" stroke="var(--accent)"/>
  <text x="315" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">CR·2 + CRC</text>
  <rect x="352" y="92" width="64" height="24" fill="none" stroke="var(--accent)"/>
  <text x="384" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">chk bit 4</text>
  <rect x="416" y="92" width="74" height="24" fill="none" stroke="var(--accent)"/>
  <text x="453" y="108" text-anchor="middle" fill="var(--accent)" font-size="8">chk bits 3–0</text>
  <rect x="490" y="92" width="180" height="24" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="580" y="108" text-anchor="middle" fill="currentColor" font-size="8">first SF−7 payload nibbles (whitened)</text>
  <text x="10" y="150" fill="currentColor" font-size="9" font-weight="bold">transmit chain, per block:</text>
  <rect x="10" y="162" width="70" height="30" fill="none" stroke="currentColor"/>
  <text x="45" y="181" text-anchor="middle" fill="currentColor" font-size="8">nibbles</text>
  <line x1="80" y1="177" x2="98" y2="177" stroke="currentColor"/>
  <rect x="98" y="162" width="110" height="30" fill="none" stroke="currentColor"/>
  <text x="153" y="181" text-anchor="middle" fill="currentColor" font-size="8">Hamming(4+CR, 4)</text>
  <line x1="208" y1="177" x2="226" y2="177" stroke="currentColor"/>
  <rect x="226" y="162" width="120" height="30" fill="none" stroke="currentColor"/>
  <text x="286" y="181" text-anchor="middle" fill="currentColor" font-size="8">diagonal interleave</text>
  <line x1="346" y1="177" x2="364" y2="177" stroke="currentColor"/>
  <rect x="364" y="162" width="90" height="30" fill="none" stroke="var(--accent)"/>
  <text x="409" y="181" text-anchor="middle" fill="var(--accent)" font-size="8">Gray decode</text>
  <line x1="454" y1="177" x2="472" y2="177" stroke="currentColor"/>
  <rect x="472" y="162" width="100" height="30" fill="none" stroke="var(--accent)"/>
  <text x="522" y="181" text-anchor="middle" fill="var(--accent)" font-size="8">×4 if reduced</text>
  <line x1="572" y1="177" x2="590" y2="177" stroke="currentColor"/>
  <rect x="590" y="162" width="80" height="30" fill="none" stroke="var(--accent)"/>
  <text x="630" y="181" text-anchor="middle" fill="var(--accent)" font-size="8">+1 → chip</text>
  <text x="340" y="218" text-anchor="middle" fill="var(--fg-muted)" font-size="8">receive: chip − 1 → divide by 4, rounded, in a reduced block → Gray encode → deinterleave → Hamming decode</text>
  <text x="340" y="236" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the nibble stream is header, whitened payload, then the un-whitened CRC, low nibble first</text>
</svg>
<figcaption>Everything left of the first block was correct from the first release; everything from the first block on was a layout only GopherTrunk's own modulator spoke.</figcaption>
</figure>

One frame, worked through with the functions now in the package. Payload
`de ad be ef`, SF7, coding rate 4/5, CRC on:

- `encodeHeader` gives the nibbles `0 4 3 0 6`: length `0x04`, then
  `CR<<1` plus the CRC bit = 3, then the checksum `0x06` across two nibbles.
  At SF7 the first block holds SF−2 = 5 nibbles, so it is exactly the
  header.
- `HammingEncode4` at 4/8 turns them into the codewords `00 2D C5 00 63`.
- `Interleave` over five rows and eight columns gives the symbols
  `4 10 20 0 4 6 8 21`.
- `symbolToChip` Gray-decodes, multiplies by four and adds one:
  `29 49 97 1 29 17 61 101` — every one 4k+1, and exactly the first eight
  chips of the committed vector `deadbeef_sf7_cr1`.
- The payload whitens to `21 53 42 17`; `PayloadCRC` is `0xEC80`, appended as
  the nibbles `0 8 C E`. Twelve more nibbles in two 5-symbol blocks make
  eighteen chips, which is what `FrameSymbolCount` returns.

The +1 and the rounding belong together. A reduced-rate chip is 4k+1, so a
dechirp peak one bin low or one bin high is still nearest to the same k — if
the receiver rounds:

```go
// internal/radio/lora/fec.go (shape)
func chipToSymbol(chip uint16, sf int, reduced bool) uint16 {
	n := 1 << uint(sf)
	v := modN(int(chip)-1, n)
	if reduced {
		v = ((v + 2) >> 2) % (n >> 2) // round to the nearest multiple of four
	}
	return grayEncode(uint16(v))
}
```

Chip 29 is k = 7. A peak at bin 28 gives `v` = 27; rounding reads 7,
flooring would read 6. That one-bin tolerance is the point of reduced-rate
mode — why the header block always uses it, and why LDRO extends it to the
payload at long symbol times. On an exact chip the two choices are
identical, so GopherTrunk can round and still match both references.

## The pin, and the gate that is still open

`bitexact_test.go` is built so that a shared mistake cannot pass it — the
[literal vectors]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
discipline:

- `TestWhiteningSequenceMatchesReference` — the first 48 whitening bytes.
- `TestHammingCodewordsMatchReference` — all sixteen codewords at each of
  the four coding rates.
- `TestPayloadCRCMatchesReference` — four payloads, including `"Hello"` →
  `0x8196`. A plain CRC-16 over those five bytes is `0xCBD6`.
- `TestEncodeMatchesReferenceChips` — six complete frames in `refFrames`,
  SF7 to SF12, all four coding rates, CRC and LDRO on and off. This is the
  failing-first pin: the old layer produced different chips for every one.
- `TestDecodeReferenceChips` and `TestDecodeCorrectsReferenceChipErrors` —
  the same chips decode with zero FEC activity, and every chip of the SF8
  4/8 frame moved one bin either way still yields `"Hello"`.
- `TestModulatedIQDemodulatesToReferenceChips` — the modulator's IQ
  demodulates to the reference chip values. This is the evidence for the
  title: the chirp, sync word and SFD needed no change.

In the project's vocabulary that is **reference-pinned** against two
independent implementations, each validated on Semtech hardware, and
**synthetic-verified** end to end. It is not capture-verified and not on-air
verified: nothing in the tree has decoded a LoRa frame a radio transmitted.
The receive side has parts no chip vector touches — per-SF probing in the
streaming receiver, real carrier offsets, LDRO at SF12 on a drifting clock —
and the
[on-air gate]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
exists for exactly those.

The issue was closed for inactivity on 14 November, with a note that closing
was not a claim that the problem is fixed. The gate has not moved: about ten
seconds of IQ from `gophertrunk capture` on a LoRa channel with the spreading
factor, bandwidth, coding rate and sync word written down, or a live
`lora.channels` run that puts frames on the `/lora` panel. An SF12 frame at
125 kHz would exercise LDRO too.

## What this overturns

[Beyond Voice Part 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }})
described the old bit chain accurately and flagged it as unverified. Its
section "The bit chain: Gray, interleave, Hamming, whitening, CRC" is
superseded by this post:

- The stage table's interleaver rule, the "LFSR 0xB8" whitening and the
  plain `PayloadCRC` are the old model. So is `grayDecode` as the receive
  direction: on air the transmitter Gray-decodes and the receiver encodes.
- The four-nibble header with an XOR checksum is gone; `HeaderNibbles` is 5
  and `ParseHeader` checks a 5-bit checksum.
- LDRO's "bins that are multiples of four" should read 4k+1.
- "Synthetic-verified only" is now "reference-pinned; still not verified on
  a capture".

The glossary entries for
[LoRa whitening]({{ '/reference/lora-whitening/' | relative_url }}) and the
[LoRa diagonal interleaver]({{ '/reference/lora-diagonal-interleaver/' | relative_url }})
were written from the same model. What stands: the chirp and dechirp
sections, `syncFrame`, the streaming receiver, and the
[LoRaWAN]({{ '/reference/lorawan/' | relative_url }}) layer, whose AES-CMAC
was always pinned to RFC 4493's vectors rather than to itself.

## What we keep

- **Name the stage, then distrust the name.** "Hamming, interleave, whiten,
  CRC" was true of both formats. A list of stages is a table of contents,
  not a layout.
- **With no capture, get two references — and transcribe them apart.**
  Their agreement is the measurement; their disagreements (padding,
  rounding) go in the test file's header comment.
- **Pin the physical layer and the bit layer separately.** The
  [SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
  replaced a whole stack; here one half was right, and reading reference
  chips back out of modulated IQ proved which half.
- **A closed issue is not a passed gate.** Verification status is kept
  per decoder, not per ticket; the ledger for this whole series is in
  [Part 14]({{ '/blog/solution-postmortem/rebuilt-from-the-air-14-audit-checklist-errata/' | relative_url }}).

## Where this goes next

LoRa's bit layer was wrong in six places and nobody noticed because nobody
transmitted at it. The next decoder had the opposite problem: a System
Fusion control channel that locked on its frame sync and then did nothing,
because its header codec modelled a field half the size of the one on the
air.
[Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})
takes the FICH apart.

## FAQ

**Does GopherTrunk decode real LoRa packets now?**
Its bit layer matches two independent implementations, each validated
against Semtech radios, on literal chip vectors covering SF7–SF12, all four
coding rates, CRC and LDRO. No frame transmitted by a real radio has been
decoded by GopherTrunk yet, so the status is reference-pinned, not on-air
verified.

**Why did the LoRa tests pass if the format was wrong?**
Every test modulated a frame with the package's own `LoRaModulate` and
decoded it with the package's own demodulator. Both sides used the same
invented layers, and a round trip cannot see a mistake that is made twice.

**What was actually wrong with the LoRa header?**
The old header was four nibbles in a block of its own. The air format is
five nibbles — length high, length low, coding rate and CRC flag, a 5-bit
checksum — opening a first block that is always 4/8 in reduced-rate mode
and also carries the first SF−7 payload nibbles.

**How do I help close the LoRa on-air gate?**
Record about ten seconds of IQ with `gophertrunk capture` while a LoRa
device with known settings sends a few packets, and note the spreading
factor, bandwidth, coding rate and sync word. A capture that decodes to the
payload you expect moves the status to capture-verified.

## Series navigation

**Part 5 of 14** · ←
[Part 4: FLEX — The Mode Code Comes Before the Marker]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }})
· Next →
[Part 6: System Fusion — A FICH of 100 Dibits, Not 100 Bits]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})
