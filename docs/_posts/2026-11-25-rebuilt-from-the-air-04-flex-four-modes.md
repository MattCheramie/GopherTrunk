---
title: "Rebuilt From the Air, Part 4: FLEX — The Mode Code Comes Before the Marker"
description: "GopherTrunk's FLEX pager decoder hunted its mode code after the sync marker, read the frame information word with no BCH and reversed the BCH information bits — and its test encoder did the same. The rebuild follows all four modes, pins 1600 bps on five frames off the air, and says what 3200 and 6400 bps still lack."
category: solution-postmortem
keywords: flex pager decoder, flex paging 6400 bps, flex frame sync 0xa6c6aaaa, flex mode code 0x870c, 4-level fsk paging, bch 31 21 syndrome decoder, p2000 flex decode, multimon-ng flex reference, self-consistent test trap, gophertrunk flex
tags: [rebuilt-from-the-air, flex, paging, bch, four-fsk, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 4
---

*Part 4 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference or a reporter's capture, rebuilt against
something independent, and pinned to real air where a capture exists.
[Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
followed the tone gates through two rounds on a reporter's bench. This part
has no reporter at all: FLEX was caught by reading a reference before
extending the decoder, and the reading showed the decoder had never matched
a frame.*

> **TL;DR:** A request for FLEX at 3200 and 6400 bps began with reading
> multimon-ng's frame handling, and that showed the shipped 1600 bps decoder
> matched no real frame. It hunted `0xA6C6AAAA` and read a 16-bit mode code
> *after* it; on air the 64-bit sync is **code : marker : ^code**, then 16
> more sync bits, then a frame information word the old code read raw, with
> no BCH. Its page-type table was invented and `FLEXBCHEncode21` left the 21
> information bits reversed; the test encoder shared all of it. The rebuild
> (`internal/radio/pager/flex`) reads the mode — `0x870C`, `0x7B18`,
> `0xB068`, `0xDEA0`, `0x4C7C` — from every frame, with a 4-level slicer and
> a syndrome-table BCH. 1600 bps is **capture-verified** on five frames off
> the Dutch P2000 network; 3200 and 6400 bps are reference-derived and
> **synthetic-verified** only.

**Key takeaways**

- **The sync is 64 bits and carries the mode code twice.** Code, marker,
  complemented code. A decoder that looks for the code after the marker
  reads the complement and resets.
- **Every frame opens the same way.** 1600 symbols/s and two levels through
  the frame information word; the sync names the rate and level count of
  the rest, so nothing is configured.
- **A capture pins only the path it exercises.** Five 1600 bps frames hold
  the sync, the BCH layout, the page layout and reassembly; they say nothing
  about four-level slicing or the phase demultiplexer.
- **Time a multi-level slicer on every transition.** An idle 6400 bps tail
  has no centre crossing for thousands of symbols, and 60 ppm free-running
  that long is a third of a symbol.

## Cheat sheet

| | |
|---|---|
| Issue | none — a request for the 3200 and 6400 bps modes; the 1600 bps decoder had shipped as "1600/2" in an earlier release |
| Symptom | no report at all. No FLEX recording had been through the decoder, and its tests were green |
| Wrong theories | that 1600 bps worked and the faster modes were an extension; that reversing 31 bits turns POCSAG's BCH into FLEX's |
| Real cause | fabricated framing — mode code hunted after the marker, an unprotected frame information word, an invented page-type table, reversed BCH information bits — shared with the test encoder |
| Fix | the frame layer rebuilt on the reference's facts; a per-frame 2- and 4-level demodulator; BCH as a syndrome lookup |
| Diagnostic | a diff against multimon-ng's frame handling, then five frames sliced from the P2000 recording in its test suite |
| Status | 1600 bps capture-verified; 3200 / 6400 bps synthetic-verified; a capture of a 929–932 MHz multi-rate channel is the on-air gate |

## In this post

- **The symptom** — there wasn't one, which is the point.
- **Two wrong theories** — "1600 works, extend it", and a true fact about bit reversal.
- **The old frame and the real one** — where the mode code sits, and what follows it.
- **The BCH** — the same code, the other way round, forty thousand times faster.
- **Five frames off the air** — what P2000 pins and what it cannot.
- **The receiver** — three lessons from a frame that changes speed.

## The symptom: there wasn't one

GopherTrunk has listed [FLEX]({{ '/reference/flex/' | relative_url }})
paging beside POCSAG since the decoder landed as a "1600/2" first slice —
1600 bps, two-level FSK, one phase.
[Beyond Voice, Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
walked it in September and was careful about its status: one mode,
synthetic-verified, real-capture calibration pending.

Nobody reported it broken. Nobody could: the fast US nationwide channels
run at higher rates the decoder skipped, and no FLEX recording had ever been
through it. What arrived instead, on the project's Discord, was a request
for the 3200 and 6400 bps modes.

The first step of that work was to read how a decoder that is known to work
handles a frame —
[multimon-ng]({{ '/reference/multimon-ng/' | relative_url }})'s
`demod_flex.c` and `demod_flex_next.c`, whose constants cite ARIB STD-43A.
They are GPL and were read for protocol facts; nothing from them is ported.
The commit that came out of the reading says what it found: the existing
1600 bps decoder's framing "matched no real frame."

## Two wrong theories

**"1600 bps works; the fast modes are an extension."** The old decoder was a
four-state machine fed one bit at a time — hunt the marker, read the mode
code, read the frame information word, de-interleave the data — and its
mode check ended in a branch that reset on anything but `0x870C`, described
as recognising the other modes "enough to be skipped cleanly." Extending it
looked like filling in that branch. It could not be extended, because the
state machine's first two transitions were in the wrong order.

**"Reversing 31 bits turns POCSAG's BCH into FLEX's."** Both protocols use
the same BCH(31,21) code, and the old `bch_flex.go` reasoned that the two
layouts "are a bit-reversal of the 31-bit codeword apart, and bit-reversal
preserves Hamming distance, so we reuse the tested POCSAG primitive." The
premise is true. Reversal maps codewords to codewords, so two errors still
get corrected. But it also hands back the 21 information bits in reverse
order, and every field the page layer reads is a bit range of that word.
A property of the code was mistaken for a property of the layout.

Both files said, in their own comments, that the on-wire bit order was left
for real-capture calibration. That honesty did not make the tests any less
green: the test encoder pushed the marker, then the mode code, then a raw
frame information word, then words from the same `FLEXBCHEncode21` — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
across four layers at once.

## The old frame and the real one

<figure class="lab-figure">
<svg viewBox="0 0 680 226" width="680" height="226" role="img" aria-label="Two FLEX frame layouts. The old model: bit sync, the marker, a mode code, a raw frame information word, then data. On the air: bit sync, mode code, marker, complemented code, sixteen more sync bits, a BCH-protected frame information word, a second sync and the data. The old mode-code box sits above the real complemented code.">
  <text x="10" y="26" fill="var(--fg-muted)" font-size="9">old model (the decoder and its test encoder)</text>
  <rect x="10" y="34" width="126" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="73" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bit sync</text>
  <rect x="136" y="34" width="104" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="188" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">marker A6C6AAAA</text>
  <rect x="240" y="34" width="56" height="26" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="268" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">mode code</text>
  <rect x="296" y="34" width="144" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="368" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FIW, 32 bits read raw</text>
  <rect x="440" y="34" width="230" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="555" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">2816 data bits, one phase</text>
  <line x1="268" y1="60" x2="268" y2="76" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="268" y="88" text-anchor="middle" fill="var(--accent)" font-size="8">compared with 0x870C — but on air these 16 bits are its complement, 0x78F3</text>
  <line x1="268" y1="94" x2="268" y2="112" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="10" y="106" fill="currentColor" font-size="9" font-weight="bold">on the air</text>
  <rect x="10" y="112" width="70" height="26" fill="none" stroke="currentColor"/>
  <text x="45" y="129" text-anchor="middle" fill="currentColor" font-size="8">bit sync 32</text>
  <rect x="80" y="112" width="56" height="26" fill="none" stroke="var(--accent)"/>
  <text x="108" y="129" text-anchor="middle" fill="var(--accent)" font-size="8">code 16</text>
  <rect x="136" y="112" width="104" height="26" fill="none" stroke="currentColor"/>
  <text x="188" y="129" text-anchor="middle" fill="currentColor" font-size="8">marker A6C6AAAA</text>
  <rect x="240" y="112" width="56" height="26" fill="none" stroke="var(--accent)"/>
  <text x="268" y="129" text-anchor="middle" fill="var(--accent)" font-size="8">^code 16</text>
  <rect x="296" y="112" width="52" height="26" fill="none" stroke="currentColor"/>
  <text x="322" y="129" text-anchor="middle" fill="currentColor" font-size="8">sync 16</text>
  <rect x="348" y="112" width="92" height="26" fill="none" stroke="currentColor"/>
  <text x="394" y="129" text-anchor="middle" fill="currentColor" font-size="8">FIW 32, BCH coded</text>
  <rect x="440" y="112" width="70" height="26" fill="none" stroke="currentColor"/>
  <text x="475" y="129" text-anchor="middle" fill="currentColor" font-size="8">sync 2, 25 ms</text>
  <rect x="510" y="112" width="160" height="26" fill="none" stroke="currentColor"/>
  <text x="590" y="129" text-anchor="middle" fill="currentColor" font-size="8">data 1760 ms, 1 to 4 phases</text>
  <path d="M 10 146 L 10 152 L 440 152 L 440 146" fill="none" stroke="var(--fg-muted)"/>
  <text x="225" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">144 symbols at 1600 symbols/s, two levels: 90 ms</text>
  <path d="M 442 146 L 442 152 L 670 152 L 670 146" fill="none" stroke="var(--accent)"/>
  <text x="556" y="166" text-anchor="middle" fill="var(--accent)" font-size="8">the rate and levels the code named</text>
  <text x="340" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sync bits: MSB first, 1 = low deviation · frame information word and data: LSB first, 1 = high deviation</text>
  <text x="340" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">a whole frame is 1.875 s whatever its mode</text>
</svg>
<figcaption>The old decoder's mode-code field sat exactly where a real frame carries the complement of the code, so its one comparison could never succeed.</figcaption>
</figure>

On the air a frame opens with 32 bits of bit sync, then a 64-bit frame sync
that carries the mode code *twice* around the marker: the 16-bit code, the
32-bit marker `0xA6C6AAAA`, and the code again complemented. Sixteen more
sync bits follow, then the 32-bit frame information word. All 144 of those
symbols go out at 1600 symbols/s on two levels, whatever the frame's mode.
Only then does the frame switch to the rate the code named: 25 ms of second
sync and 1760 ms of data, for a frame of exactly 1.875 s.

That is why the old decoder could not have locked on anything real. After
the marker it read sixteen bits and compared them with `0x870C`; on air
those sixteen bits are `0x78F3`, the complement. It reset. Had it passed,
the next 32 bits it took for the frame information word are the sync tail
and half of the real one — which is BCH-protected, carries a nibble
checksum and goes out LSB first.

The rebuilt hunt matches all 64 bits at once:

```go
// internal/radio/pager/flex/decoder.go
func MatchSync(reg uint64) (Mode, bool) {
	marker := uint32(reg >> 16)
	codeHigh := uint16(reg >> 48)
	codeLow := ^uint16(reg)
	if bits.OnesCount32(marker^SyncMarker) >= 4 || bits.OnesCount16(codeHigh^codeLow) >= 4 {
		return Mode{}, false
	}
	for _, m := range Modes {
		if bits.OnesCount16(m.Code^codeHigh) < 4 {
			return m, true
		}
	}
	return Mode{}, false
}
```

The code it returns is the frame's whole configuration:

| Code | Symbols/s | Levels | Bit rate | Phases |
|---|---|---|---|---|
| `0x870C` | 1600 | 2 | 1600 bps | A |
| `0x7B18` | 3200 | 2 | 3200 bps | A, C |
| `0xB068` | 1600 | 4 | 3200 bps | A, B |
| `0xDEA0` | 3200 | 4 | 6400 bps | A, B, C, D |
| `0x4C7C` (ReFLEX) | 3200 | 4 | 6400 bps | A, B, C, D |

Each phase is an independent 1600 bps stream of 88 interleaved words with
its own block information word, addresses, vectors and messages. Three
facts decide how symbols become phases. The bit senses differ by section:
in the sync a 1 is the *low* deviation, sent MSB first; in the frame
information word and the data a 1 is the *high* deviation, sent LSB first.
A four-level symbol carries two
[Gray-coded]({{ '/reference/gray-code/' | relative_url }}) bits — the first
is the sign of the deviation, the second is set on the two inner levels.
And at 3200 symbols/s alternate symbols belong to phases A/B and C/D.
`placeSymbol` does all three, and puts bit counter `c` in word
`((c>>5)&^7)|(c&7)` to undo the block interleave.

The page layer moved too. The frame information word, the block
information word and every vector word carry a nibble checksum — five
nibbles plus bit 20 summing to `0xF` — which the rebuilt decoder enforces.
The old page-type table agreed with the real vector types in two of eight
positions, tone and alphanumeric; a numeric page is type 3, not 0 or 1.
Long two-word addresses, the numeric bit stream, short messages and
alphanumeric messages split across frames are all new.

## The BCH: the same code, the other way round

The real layout of a FLEX word, from `framing/bch_flex.go`: information in
bits 0–20, [BCH]({{ '/reference/bch-code/' | relative_url }}) parity in
21–30, even parity in bit 31 — and, as a polynomial, codeword bit *b* is
the coefficient of x^(30−b). Against POCSAG's layout that is a mirror of the
information *and* of the codeword. `TestFLEXBCHMatchesPOCSAGCode` states the
relation in one line: reverse the 21 information bits, encode with the
independent POCSAG encoder, reverse the 31-bit result. The old wrapper did
the second reversal and not the first.

Three things pin the new layout. `TestFLEXBCHLayout` is a vector worked by
hand from the generator: information bit 20 alone must set bits 21, 22, 24,
25, 27 and 30, plus the parity bit. The POCSAG encoder is a second
implementation of the same code. And the air pins it from the other side —
a random 31-bit word has a zero syndrome once in 1024, so
`TestRealAirWordsAreCodewords` requires that at least 80 % of the on-air
words are error-free codewords and none is uncorrectable.

The decoder changed as well as the layout. The old one found the nearest
codeword by searching all 2^21 of them — milliseconds per word, with 352
words in a 6400 bps frame. A (31,21) code that corrects two errors has
1 + 31 + 465 correctable patterns, each with its own 10-bit syndrome, so
decoding is one 1024-entry table lookup: about 45 ns.

## Five frames off the air

multimon-ng's own test suite contains a FLEX recording of the Dutch P2000
network at 1600 bps. Five frames were sliced from it and committed as
`testdata/p2000_1600x2_frames.hex` — one line per frame, the 2904 symbols
after the 64-bit sync, one bit per symbol. The recording itself is the
reference project's GPL content and is not in the repository.

`TestRealAirFrames` pushes each frame through the decoder and expects the
ten pages multimon-ng's suite expects from the same audio: capcode 1120103
at cycle 14, frame 116, reading "test"; 1180000 reading "TESTOPROEP MOB";
and a test alarm addressed to six capcodes across frames 12 and 13 — one of
which, 1400141, starts in frame 12 and finishes in frame 13, so fragment
reassembly is pinned too. The counters must read five frames synced, no
frame-information-word errors, no failed words. The test's comment records
what the previous decoder made of the same frames: "it decoded none."
(Paging is unencrypted and often personal; monitor it only where you are
permitted to. The pages quoted here are the network's own test traffic.)

`TestRealAirFrameSections` then turns the fixture on the synthesiser:
`FrameSymbols` must reproduce the sixteen sync-tail symbols and the second
sync exactly as the air carries them, and re-encoding the on-air frame
information word — cycle 0, frame 12 — must give back the transmitted word
bit for bit.

Now the limit, which is the reason this post exists. Five 1600 bps frames
exercise phase A, two levels, one rate. They do not touch four-level
slicing, the Gray mapping, the A/B–C/D alternation or the second sync at
3200 symbols/s — and the reference's own generator is 1600/2 only, so there
is no independent *encoder* for the fast modes either. Those paths follow
the reference's facts and decode GopherTrunk's own synthetic frames in every
mode (`TestEveryModeCarriesItsPhases`, `TestDemodDecodesEveryMode`,
`TestDemodFollowsMixedRates`). That is reference-derived and
synthetic-verified, and nothing more.

## The receiver: three lessons from a frame that changes speed

The old front end was POCSAG's — an integrator and a mean-tracking sign
slicer at a fixed 1600 baud — and a sign slicer cannot read four levels.
`receiver/demod.go` resamples the discriminator to 25.6 kHz, sixteen samples
per symbol while hunting and eight in a 3200 symbols/s body, and does three
things differently.

**Hunt at every phase, and keep no loop while hunting.** The sync is
searched at all sixteen sample phases of the 1600 symbols/s grid at once,
and the middle of the run of phases that match fixes the timing. A timing
loop would have to pull in — and one hunting at 1600 symbols/s through the
3200 symbols/s body of a frame it missed is dragged off.

**Measure the frame's levels from its own sync.** The 64 sync symbols are
all sent at full deviation in a near-balanced pattern, so their means give
the centre and the outer deviation; the inner decision levels sit at ±2/3
of it. Each frame is calibrated on its own.

**Time on every transition.** The first loop used only crossings through
the centre. A frame's idle tail, with all four phases idle, is the pattern
0, 0, 2, 2 — no centre crossing for thousands of symbols — and a 60 ppm
clock error left free-running that long slides the slicer a third of a
symbol off, after which mis-sliced symbols feed the loop and it runs away.
The loop now uses the midpoint of whichever two levels a transition joins.
`TestDemodFollowsMixedRates` runs five frames of mixed modes back to back at
±60 ppm.

A recording of your own replays through the same demodulator:

```bash
GT_FLEX_AUDIO=<wav or flac> \
  go test ./internal/radio/pager/flex/receiver -run TestFLEXAudioReplay -v
```

Live, there is no rate to set — `paging.flex` takes a serial and a
frequency — and the first frame on a channel logs what the network runs:

```text
flex: frame sync source=pager-sdr bps=6400 symbol_rate=3200 levels=4 inverted=false
```

The [paging page]({{ '/pocsag.html' | relative_url }}) has the operator
view and
[Cookbook Volume 2, Part 2]({{ '/blog/tutorials/cookbook-volume-2-02-paging-rig/' | relative_url }})
the recipe.

## What this overturns

[Beyond Voice, Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
described the FLEX decoder as it then was, and its FLEX half is superseded:
the four-state machine that read a mode code after the marker, the single
supported mode `0x870C`, `FLEXBCHDecode32` as a 31-bit reversal around the
POCSAG primitive, short addresses only, no reassembly, and a receiver that
was "the same chain at a fixed 1600 baud." Its POCSAG half stands, as does
everything downstream of the decoder — the `pager.message` event,
`pager_log`, the REST route, the panel and `paging.wideband`. Its status
line needs splitting in two: 1600 bps is now capture-verified; the modes it
listed as follow-ups exist and are synthetic-verified.

## What we keep

- **Read the reference before extending, not after.** The feature request
  was for two new modes; the finding was that the old one never worked.
- **Separate what a transform preserves from what it rearranges.** Bit
  reversal keeps Hamming distance and scrambles field positions.
- **Say which path a capture pins.** "Real-air pinned" is true of 1600 bps
  and silent about phases B, C and D; the status has to be per mode.
- **An independent encoder is a resource, and it can run out.** Where the
  reference cannot generate a mode, the only remaining pin is a recording.
- **When a frame changes speed, acquire it per frame.** No state from one
  frame's timing is worth carrying into the next.

## Where this goes next

FLEX had a right marker in a wrong frame. The next decoder had a right
*physical layer* under six wrong ones.
[Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }})
is LoRa, where the chirps, the preamble sync and the SFD were real all
along and everything from the symbol up was invented — and where two
independent references, transcribed separately, had to agree on 2,916 cases
before the new bit layer was allowed to replace the old.

## FAQ

**What does a FLEX frame sync look like?**
Sixty-four bits at 1600 symbols/s: a 16-bit mode code, the 32-bit marker
`0xA6C6AAAA`, and the same code complemented. It follows 32 bits of bit
sync and is followed by sixteen more sync bits and a BCH-protected frame
information word. The code names the symbol rate and level count of the
rest of the frame.

**Which FLEX modes does GopherTrunk decode?**
All of them, read per frame from the sync: 1600 bps (code `0x870C`), 3200
bps two-level (`0x7B18`), 3200 bps four-level (`0xB068`) and 6400 bps
(`0xDEA0`, and `0x4C7C` for ReFLEX). There is no rate to configure, and a
channel that mixes rates is followed frame by frame.

**Is FLEX decoding verified on real signals?**
At 1600 bps, yes: five frames sliced from an off-air P2000 recording decode
to the pages the reference decoder's own test suite expects. The 3200 and
6400 bps paths follow the same reference and decode synthetic signals in
every mode, but no recording of a multi-rate channel has been through them
yet.

**Why did the old FLEX decoder never decode a real frame?**
It looked for the mode code after the sync marker, where a real frame
carries the complement of the code, so its mode check always failed. Past
that, it read the frame information word without BCH and its BCH wrapper
returned the information bits reversed. Its test encoder shared every one
of those choices.

**How do I check a FLEX recording of my own?**
Record discriminator audio as WAV or FLAC — for example with `rtl_fm` — and
run `TestFLEXAudioReplay` with `GT_FLEX_AUDIO` set to the file. It prints
every page and the frame counters by bit rate. A capture of a 3200 or
6400 bps channel is exactly what the open gate needs.

## Series navigation

**Part 4 of 14** · ←
[Part 3: CTCSS & DCS — A Rate Trap, a Rounded Bin and an Invented Codeword]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
· Next →
[Part 5: LoRa — Six Invented Layers Under a Correct Chirp]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }})
