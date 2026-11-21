---
title: "Rebuilt From the Air, Part 2: MDC1200 — The Tones Were Sliced as the Data"
description: "GopherTrunk's MDC1200 decoder never decoded a real radio: the receiver fed its 1200/1800 Hz tone decisions to the sync hunt as data bits, but MDC1200 is XOR-precoded MSK and the data is the running XOR of those decisions. The diagnosis, the fix, its three independent pins and the on-air run that closed issue 1220."
category: solution-postmortem
keywords: mdc1200 decoder, xor precoded msk, mdc1200 line code, mdc1200 ptt id sdr, ffsk 1200 1800 hz, motorola signalling decode, mdc1200 sync word 0x07092a446f, self-consistent test trap, reference encoder test vectors, gophertrunk mdc1200
tags: [rebuilt-from-the-air, mdc1200, ffsk, line-code, postmortem, testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 2
---

*Part 2 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference implementation or a reporter's capture,
rebuilt against an independent decoder, and pinned to real air where a
capture exists.
[Part 1]({{ '/blog/solution-postmortem/rebuilt-from-the-air-01-the-audit/' | relative_url }})
was the audit and the roll call. This part takes the decoder that was
closest to right: MDC1200, whose sync word, interleave and CRC were all
correct, and whose receiver still never locked on a radio.*

> **TL;DR:** For four months GopherTrunk's MDC1200 front end
> (`internal/radio/mdc1200/afsk`) decided which tone each bit period carried
> and pushed that decision into the 40-bit sync hunt as the data bit.
> MDC1200 is XOR-precoded MSK: one cycle of 1200 Hz means the data bit
> equals the previous one, 1.5 cycles of 1800 Hz means it changed, so the
> data is the **running XOR** of the tone decisions. Read as tones, the sync
> word `0x07092A446F` sits 24 bits from itself and 16 from its complement;
> the framer tolerates five. `feedSymbol` now slices at zero and toggles its
> data bit on every 1800 Hz period. Three pins hold the format: literal
> bursts and tone strings from the reference encoder, that encoder's own
> 48 kHz audio, and a real-air slice. **On-air verified** 28 Sep 2026
> ([#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220)): eleven
> PTT ID bursts from unit `0x1777`.

**Key takeaways**

- **A tone is not a bit.** Under XOR precoding 1800 Hz means "changed", so
  a receiver that reads tones as bits recovers the transitions of the data
  and never sees the sync word.
- **The control experiment came free.** FleetSync decoded eight for eight
  through the same DSP chain on the same bench, which put the fault between
  the slicer and the framer before anyone opened the code.
- **Only an encoder you did not write could fail this.** Every existing
  test passed; all eight air-format tests fail on the old slicer.
- **Clipped is not undecodable.** The committed real-air slice comes from a
  recording at both ADC rails and decodes CRC-valid.

## Cheat sheet

| | |
|---|---|
| Issue | [#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220) — where the first on-air MDC1200 run was reported (the decoder shipped for [#438](https://github.com/MattCheramie/GopherTrunk/issues/438)) |
| Symptom | on the reporter's bench FleetSync decoded through the same DSP chain; MDC1200 "never locked" — zero sync locks, not CRC failures |
| Wrong theories | the DSP chain; an inverted discriminator; a wrong sync word, interleave or CRC; the slicer threshold |
| Real cause | the line code — tone decisions pushed to the framer as data, when the data is the running XOR of the tone decisions |
| Fix | `feedSymbol` slices at zero, toggles `r.data` on an 1800 Hz period and pushes that; the synthesiser rewritten to the air format |
| Diagnostic | the reference encoder's 26-byte burst and 208-bit tone string for one packet, side by side |
| Status | on-air verified 28 Sep 2026; real-air slice committed; the on-air parity bytes are generated but not yet used for correction |

## In this post

- **The symptom** — the neighbouring protocol decoded; this one never locked.
- **Four suspects** — the chain, the polarity, the constants, the threshold.
- **The diagnostic** — an encoder we did not write, and one line of output.
- **The real line code** — XOR-precoded MSK, bit by bit.
- **The fix and three pins** — one function, and tests that cannot share it.
- **On air, and what this overturns** — eleven bursts and a corrected post.

## The symptom: the neighbour decoded, this one never locked

GopherTrunk has shipped an
[MDC1200]({{ '/reference/mdc1200/' | relative_url }}) decoder — for the data
burst a radio keys at the start and end of an analog transmission — since
the feature request in
[#438](https://github.com/MattCheramie/GopherTrunk/issues/438): an FFSK front
end, a 40-bit sync framer, a 16×7 de-interleave, a CRC, a bus event, a log
table, a web panel.
[Beyond Voice, Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
walked that chain in September and ended on its gap: every frame its tests
fed came from a helper that inverted the decoder's own interleave, and no
air capture was committed. It called the decoder synthetic-verified.

A week later the gap closed the hard way. On 26 Sep the reporter of #1220
ran both signalling decoders against their radio bench.
[FleetSync]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
— on-air verified ten days earlier — decoded eight bursts out of eight.
MDC1200 "never locked."

That phrase is a precise symptom. The framer keeps three counters — sync
words detected, bursts that failed CRC, bursts published — and they separate
two failures. The replay harness spells the triage out in its verdict
line: "sync locks with CRC failures mean the framing or field layout is the
next suspect; zero sync locks on a clean signal means the line code or
tones." This was the second kind.

## Four suspects, and what cleared each

**The DSP chain.** Both signalling decoders run the chain described in
[Beyond Voice, Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }}):
FM discriminator, resample to 9600 Hz, the
[FFSK]({{ '/reference/ffsk/' | relative_url }}) tone discriminator on the
1200/1800 Hz pair, Mueller-Müller timing at eight samples per bit. FleetSync
uses the same tones at the same baud and decoded on the same bench. A
front-end defect that hid every MDC1200 burst would not have left its
neighbour at eight for eight.

**Discriminator polarity.** The framer already matched the sync word *and*
its bitwise complement, inverting the payload on a complement lock. And the
question is moot for a tone decision: an FM discriminator with the opposite
sense negates the audio, which does not change a tone's frequency.

**The constants.** The sync word `0x07092A446F`, the 16×7 grid and the
reflected CRC-16 are public protocol facts, and the rebuild confirmed all
three had been right from the start. The framing was not fabricated.

**The slicer threshold.** MDC1200 sliced against a slow running mean.
FleetSync had already measured why that is wrong for this tone pair — a
tracked threshold drifted toward a long single-tone run and flipped 23 bits
of one frame where a fixed zero flipped none — and an MDC1200 leader turns
out to be exactly such a run. The threshold became a fixed zero in the same
fix. But a drifting threshold costs bits; it does not make a sync hunt fail
on every burst, and the arithmetic below fails on a perfectly sliced one.

## The diagnostic: an encoder we did not write

The way out of a
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
is a fact the code under test did not produce —
[SmartNet's rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
took its facts from OP25. For MDC1200 the fact is the open-source reference
modem, `mdc-encode-decode`, which the package comment describes as the
decoder every MDC1200 tool in the field is built on. It is GPL; nothing
from it is ported. It was read for the format and run to produce data.

Asked for one packet — op `0x01`, arg `0x80`, unit `0x1234`, a PTT ID — its
encoder queues 26 bytes:

```text
55555555555555 07092a446f 8e942a2e06992204001c18a22c88
leader (7)     sync (5)   interleaved block (14)
```

GopherTrunk's interleave, CRC and bit order already agreed with those
bytes. Then the reference's sampler turns them into tones, one per bit
period — `1` for 1800 Hz, `0` for 1200 Hz. The first hundred:

```text
0111111111111111111111111111111111111111111111111111111110000100100011011011111101100110010110000100
```

The leader bytes are `0x55` — the alternating pattern `01010101`. On the
air they are a `0` followed by fifty-five `1`s: after the first bit period,
**one continuous 1800 Hz tone**. A receiver that reads tones as bits sees an
unbroken run where the leader carries its bit-sync pattern. That single
line is the diagnosis.

Bit periods 56–95 of that string carry the forty sync bits; rewritten in
the receiver's own sense (1200 Hz = 1) they read `0x7B724099A7`. That is 24
bits from `0x07092A446F` and 16 from its complement. The framer's
`gdThresh` allows five. No signal quality closes that distance.

## The real line code: 1800 Hz means "changed"

The reference modem calls the format XOR-precoded MSK. At 1200 baud a
1200 Hz tone completes exactly one cycle in a bit period and an 1800 Hz tone
exactly one and a half, so the waveform ends a 1200 Hz bit where it started
and ends an 1800 Hz bit inverted. The transmitter compares each data bit
with the one before it, starting from zero:

- the bit is the **same** as the previous one → one cycle of 1200 Hz;
- the bit **changed** → one and a half cycles of 1800 Hz.

Because 1800 Hz flips the waveform and 1200 Hz does not, the *sign* of the
audio at a fixed instant in each bit period follows the data bit itself —
the property the reference decoder's one-point sampler relies on. It does
not need to identify a tone at all.

GopherTrunk's front end does identify tones — that is what an FFSK
discriminator is for — and that is a legitimate way to receive this signal,
provided the precoding is undone. The data is the running XOR of the tone
decisions:

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Twelve bit periods across the end of an MDC1200 leader and the first byte of the sync word. Four rows: the data bits; the tone sent for each, 1800 hertz when the bit changed and 1200 hertz when it did not; the old receiver reading tones as bits, which differs; and the running XOR, which reproduces the data.">
  <text x="208" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="9">end of leader (0x55…)</text>
  <text x="472" y="16" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">first byte of the sync word (0x07)</text>
  <line x1="296" y1="6" x2="296" y2="200" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="112" y="46" text-anchor="end" fill="currentColor" font-size="9">data bit</text>
  <text x="142" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="186" y="46" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="230" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="274" y="46" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="318" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="362" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="406" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="450" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="494" y="46" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="538" y="46" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="582" y="46" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="626" y="46" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="112" y="88" text-anchor="end" fill="currentColor" font-size="9">tone on air</text>
  <rect x="122" y="70" width="216" height="28" fill="none" stroke="var(--accent)"/>
  <text x="230" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">1800 Hz × 5 periods (each bit changed)</text>
  <rect x="342" y="70" width="172" height="28" fill="none" stroke="currentColor"/>
  <text x="428" y="88" text-anchor="middle" fill="currentColor" font-size="9">1200 Hz × 4 (same)</text>
  <rect x="518" y="70" width="40" height="28" fill="none" stroke="var(--accent)"/>
  <text x="538" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">1800</text>
  <rect x="562" y="70" width="84" height="28" fill="none" stroke="currentColor"/>
  <text x="604" y="88" text-anchor="middle" fill="currentColor" font-size="9">1200 × 2</text>
  <text x="112" y="132" text-anchor="end" fill="var(--fg-muted)" font-size="9">old: tone as bit</text>
  <text x="142" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="186" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="230" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="274" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="318" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="362" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="406" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="450" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="494" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="538" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0</text>
  <text x="582" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="626" y="132" text-anchor="middle" fill="var(--fg-muted)" font-size="10">1</text>
  <text x="112" y="174" text-anchor="end" fill="var(--accent)" font-size="9">new: running XOR</text>
  <text x="142" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="186" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="230" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="274" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="318" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="362" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="406" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="450" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="494" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">0</text>
  <text x="538" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="582" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="626" y="174" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">1</text>
  <text x="384" y="218" text-anchor="middle" fill="var(--fg-muted)" font-size="8">toggle on 1800 Hz, hold on 1200 Hz — the old row is the data's transitions, not the data</text>
</svg>
<figcaption>The tones carry whether each bit changed; only their running XOR is the sync word the framer is looking for.</figcaption>
</figure>

The old receiver pushed the third row into the framer. Its tests encoded
the same way — a bit became a tone — so the round trip was exact.

## The fix, and three pins that cannot share its mistake

The receiver change is one function:

```go
// internal/radio/mdc1200/afsk/receiver.go
func (r *Receiver) feedSymbol(s float32) {
	if s <= 0 {
		r.data ^= 1
	}
	r.inner.Push(r.data)
	r.bitsEmitted.Add(1)
}
```

A positive symbol is the 1200 Hz tone — hold. A non-positive one is 1800 Hz
— toggle. The threshold is a fixed zero for the FleetSync reason: the FFSK
stage's output is DC-free by construction, and a tracked mean drifts toward
exactly the kind of single-tone run the leader is.

Two consequences. The running XOR's start state is unknown, which
complements the whole data stream — so the framer's complemented-sync lock
is still needed, but it now absorbs the XOR state, not the discriminator's
sense. And one wrong tone decision inverts every later bit of the burst;
the CRC rejects such a burst as it would any other, and the reference's own
sampler has the same per-bit error structure.

The transmitter side was rewritten too (`mdc1200/synth.go`): `BurstBits`
lays out leader, sync and interleaved block, `Precode` applies the line
code, and `fecParity` generates what the earlier post called "eight
redundancy bytes" — in fact one zero byte and seven bytes of a rate-1/2
[convolutional code]({{ '/reference/convolutional-code/' | relative_url }})
over the block's first seven bytes, LSB first, from a seven-stage register
tapped at stages 0, 2, 5 and 6. The decoder still does not use them for
correction; they exist so a synthetic burst matches the reference's byte
for byte.

A rewritten encoder beside a rewritten decoder is the trap again unless
something outside pins both. Three things do:

| Pin | What it holds | Where |
|---|---|---|
| Literal vectors | three 26-byte bursts and two 208-bit tone strings exactly as the reference encoder emits them, plus two CRC values from an independent implementation | `synth_test.go`: `TestBurstBitsMatchReferenceEncoderBytes`, `TestPrecodeMatchesReferenceToneSequence`, `TestCRC16ReferenceVectors` |
| Reference audio | 17,921 samples of 48 kHz PCM the reference encoder itself produced, decoded through the production audio path in chunks of 4096, 1000 and 1 | `afsk/airformat_test.go`: `TestReceiverDecodesReferenceEncoderAudio` |
| The other direction | GopherTrunk's synthesiser, decoded by the reference decoder | done offline; the reference stays outside the repository |

The second pin "cannot share a wrong layout with anything in this
repository," as its test comment puts it. The committed file is data the
reference program produced, not its code — the distinction
[the clean-room rules]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
turn on.

And the regression is
[failing-first]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }}).
`airformat_test.go` holds eight tests — a precoded burst, the reference
audio, polarity invariance, chunk invariance, noise, carrier offset, a
double packet, reset. All eight fail against the old slicer, which the
project's notes record as 448 bits delivered and zero sync locks (448 is
the number of bit periods in the 0.37 s reference recording). Every test
that existed before the fix passed against it.

## On air: eleven bursts and a recording at both rails

On 28 Sep the same bench ran the fixed build. It decoded **eleven PTT ID
bursts** — keyup (`0x01`/`0x80`) and unkey (`0x01`/`0x00`) — from unit
`0x1777` on 447.100 MHz, on narrow and wide FM, through a scan-list channel
with `decoders: [mdc1200]`. That path was new in the same issue and has its
own front end, because a discriminator fed the scanner's raw 2.4 MS/s
stream is swamped by the whole band's noise;
[Squelch, Part 11]({{ '/blog/deep-dives/squelch-11-data-on-the-scan-list/' | relative_url }})
covers it. #1220 closed on the reporter's confirmation.

The reporter also posted the scanner's own 2.4 MS/s voice recording of one
transmission. It is ugly: ADC-clipped at both rails, RMS −0.9 dBFS, the
handheld in the same room. It replays anyway — the harness finds the
carrier at −1758 Hz and decodes a CRC-valid end-of-transmission PTT ID — the
lesson
[DMR direct mode]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
taught: clipped is not undecodable. A 0.7 s slice, channelized to 48 kHz, is
now `afsk/testdata/mdc1200_unit1777_pttid_end_48k.cs16`, and
`TestMDC1200RealAirSlice` demands exactly one CRC-valid burst from unit
`0x1777` out of it. To check another radio, record a keyup with a known
unit ID and run:

```bash
GT_MDC1200_IQ=<capture> GT_MDC1200_RATE=<Hz> GT_MDC1200_UNIT=<hex unit id> \
  go test ./cmd/gophertrunk -run 'TestMDC1200Replay$' -v
```

The [operator page]({{ '/mdc1200.html' | relative_url }}) lists the other
knobs;
[Cookbook Volume 2, Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
has the config.

## What this overturns

[Beyond Voice, Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
stands on the frame: the sync hunt and its five-error tolerance, the grid,
the CRC, the opcode table, double packets and the bus-to-panel chain are as
it describes. Four things in it are superseded:

- **What `Push` receives.** Not bits "one at a time from the slicer" but
  the running XOR of the slicer's tone decisions.
- **What the complement lock is for.** Not an inverted discriminator — a
  tone's frequency does not depend on that — but the XOR's start state.
- **The redundancy.** One zero byte and seven convolutional parity bytes,
  not eight undifferentiated ones.
- **The status.** Synthetic-verified with no committed air capture then;
  on-air verified now, with a reference recording and a real-air slice in
  `afsk/testdata`.

[Beyond Voice, Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})
carries a correction in its line-code table; its sentence that MDC1200 and
FleetSync both "carry the bit on the tone directly" is true of FleetSync
only.

## What we keep

- **"Never locked" and "bad CRC" are different diagnoses.** Zero sync locks
  on a clean signal points below the frame, at the line code or the tones.
  Keep the counters that tell the two apart.
- **A working neighbour is a control experiment.** FleetSync on the same
  chain cleared four stages of DSP in one observation.
- **The line code is part of the format.** Pinning bytes is not enough; pin
  the symbol sequence an independent encoder emits for them.
- **Commit reference data, not reference code.** Audio a GPL encoder
  produced is a fact about the protocol, and a fixture that cannot inherit
  your mistake.

## Where this goes next

MDC1200 was one wrong step between two right layers. The tone gates on the
same scan list were wrong in several unrelated ways at once.
[Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
follows CTCSS and DCS through two rounds of the same issue: a threshold
calibrated at the wrong sample rate, a Goertzel bin rounded onto the wrong
tone, and a DCS codeword that existed only in GopherTrunk.

## FAQ

**What line code does MDC1200 use?**
XOR-precoded MSK on the 1200/1800 Hz tone pair at 1200 baud. The radio
sends one cycle of 1200 Hz when a data bit equals the previous one and one
and a half cycles of 1800 Hz when it changed, so the data stream is the
running XOR of the tone decisions rather than the tones themselves.

**Why did GopherTrunk's MDC1200 decoder never lock on a real radio?**
Its front end pushed each tone decision into the sync hunt as if it were
the data bit. In that form the 40-bit sync word is 16 bits from the nearest
pattern the framer accepts, against a tolerance of five. The decoder's own
tests encoded bits as tones the same way, so they passed.

**Does an inverted FM discriminator matter for MDC1200?**
Not for the tone decision: inverting the discriminator negates the audio,
which leaves every tone at the same frequency. The only ambiguity is the
running XOR's start state, which complements the whole data stream; the
framer accepts the complemented sync word and inverts the payload back.

**Is the MDC1200 decoder verified on air?**
Yes. On 28 Sep 2026 the reporter's live run decoded eleven PTT ID bursts,
start and end, from unit `0x1777` on 447.100 MHz through a scan-list
channel, and a slice of their recording is a committed regression. The
parity bytes are not yet used to correct errors.

**How do I check an MDC1200 capture of my own?**
Record a keyup from a radio whose unit ID you know, then run
`TestMDC1200Replay` with `GT_MDC1200_IQ`, `GT_MDC1200_RATE` and
`GT_MDC1200_UNIT` set. The harness replays the file through the production
front end and passes when that unit decodes CRC-valid.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The Audit — The Decoders That Never Heard a Radio]({{ '/blog/solution-postmortem/rebuilt-from-the-air-01-the-audit/' | relative_url }})
· Next →
[Part 3: CTCSS & DCS — A Rate Trap, a Rounded Bin and an Invented Codeword]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }})
