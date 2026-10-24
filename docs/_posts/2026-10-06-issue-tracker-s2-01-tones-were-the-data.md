---
title: "From the Issue Tracker, Season 2, Part 1: The Tones Were the Data — MDC1200's XOR-Precoded Line Code"
description: "How GopherTrunk's MDC1200 decoder went four months without decoding a single real Motorola radio — the receiver sliced the 1200/1800 Hz tones as data when the data is the running XOR of the tones — and the three independent pins (reference-encoder bytes, reference-encoder audio, a real-air slice) that finally made the 40-bit sync word match on 447.100 MHz."
category: solution-postmortem
keywords: mdc1200 never decodes, mdc1200 xor precoded msk, mdc1200 line code 1200 1800 hz, mdc1200 sync word 0x07092a446f, ptt id ani decode sdr, self-consistent test trap, reference encoder vectors, mdc1200 fleetsync same dsp chain, issue 1220, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, mdc1200, afsk, line-code, testing, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 1
---

*Part 1 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems — one bug per part, told the way it happened: symptom, the
explanations that looked right, the instrument that settled it, the fix,
the failing-first regression, what is still open. It picks up where
[Season 1]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
left off on 3 September and leans on the lessons that series ended with:
the [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
[census everything]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
and [two pipelines]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}).
The opening story is the purest case of the first lesson yet: a decoder
whose tests stayed green for its whole life because decoder and tests
agreed on a line code no radio uses.*

> **TL;DR:** MDC1200 is **XOR-precoded MSK**: a Motorola radio sends one
> cycle of 1200 Hz when a data bit equals the previous one and 1.5 cycles of
> 1800 Hz when it changed, so the data is the **running XOR of the tone
> decisions**, and the 40-bit sync word `0x07092A446F` lives in the data
> domain. GopherTrunk's `afsk.feedSymbol` sliced each tone against a tracked
> mean and pushed the result into the sync hunt as plain NRZ; its tests
> encoded bursts the same way — green for four months, zero decodes on air
> ([#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220)). The
> reporter's Kenwood lab exposed it: FleetSync decoded 8/8 through the
> *identical* DSP chain while MDC1200 "never locked". Interleave, CRC and bit
> order were already correct. The fix is a zero-threshold tone decision, a
> running XOR (`r.data ^= 1` on 1800 Hz) and the framer's existing
> complemented-sync accept to absorb the XOR's unknown start state. Pinned
> three ways that share nothing with the decoder: `synth_test.go` holds
> literal 26-byte bursts and 208-bit tone strings from the reference encoder,
> `airformat_test.go` decodes the reference encoder's own audio, and
> `TestMDC1200RealAirSlice` decodes unit 0x1777's PTT ID from a real
> 447.100 MHz capture. On-air verified 28 September; #1220 is closed.

**Key takeaways**

- **A line code is a convention the encoder and decoder must not share.**
  Both sides of the round-trip believed the tones were the bits; the test
  could only measure their agreement.
- **The control experiment was already running.** FleetSync — same
  discriminator, resampler, `demod.FFSK` and timing loop — decoded live.
  The only stage that differed was the one after the slicer.
- **Pin against things you did not write.** Reference-encoder bytes,
  reference-encoder audio and a real radio's slice fail independently.
- **A silent decoder is harder to see than a wrong one.** With the data
  XOR-scrambled no burst came within five errors of the sync word, so there
  were no CRC failures to read — just nothing.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Tone decision + XOR | `s <= 0` is 1800 Hz (changed) → `r.data ^= 1`; zero threshold | `internal/radio/mdc1200/afsk/receiver.go` (`feedSymbol`, `Reset`) |
| Start-state ambiguity | sync accepted at ≤ `gdThresh` (5) errors or ≥ 35 (complement) | `internal/radio/mdc1200/receiver/receiver.go` (`Push`) |
| Transmit model | `BurstBits` → `Precode`; the 0x55 leader is one continuous 1800 Hz tone | `internal/radio/mdc1200/synth.go` |
| Reference-literal pins | 26-byte bursts, 208-bit tone strings, CRC vectors | `internal/radio/mdc1200/synth_test.go` |
| Reference-audio pin | `mdc1200_ref_01_80_1234_48k.s16` through `ProcessAudio` | `afsk/airformat_test.go` (`TestReceiverDecodesReferenceEncoderAudio`) |
| Real-air pin | unit 0x1777, PTT ID (end), 447.100 MHz, 0.7 s cs16 slice | `afsk/realair_test.go` (`TestMDC1200RealAirSlice`), `TestMDC1200Replay` |

## In this post

- **The report** — one lab, two decoders, one DSP chain, one of them silent.
- **What the decoder believed** — tones as NRZ bits, and why nothing downstream could tell.
- **What the radio sends** — XOR precoding, and where the sync word really lives.
- **The fix** — four lines in `feedSymbol` and an ambiguity the framer already handled.
- **Three pins that share nothing with the decoder** — bytes, audio, air.
- **On air, 28 September** — eleven bursts, a clipped capture, a closed issue.

## The report

[Beyond Voice Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }})
called MDC1200 the template decoder — the framer, de-interleaver, CRC and
opcode table FleetSync was later cloned from — and ended on an honest gap:
no on-air fixture was committed, so MDC1200 was synthetic-verified only.
That gap was the whole story.

The reporter behind
[#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220) runs a
Kenwood lab and had just confirmed FleetSync on air
([Beyond Voice Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})):
FS-I and FS-II both decoding, 8/8 bursts. The same rig, pointed at a
Motorola radio keying on 447.100 MHz, produced nothing — MDC1200 "never
locked". No CRC failures, no partial frames, no `(CRC?)` rows on the panel.
Silence.

That silence is the diagnostic. Both front ends are one pipeline:
`demod.FM` → `dsp.RealResampler` to 9600 Hz (`BaudHz` 1200 × `Oversample`
8) → `demod.FFSK` with mark 1200 Hz and space 1800 Hz → `sync.MuellerMuller`
at eight samples per bit → a slicer
([Beyond Voice Part 2]({{ '/blog/deep-dives/beyond-voice-02-afsk-ffsk-fundamentals/' | relative_url }})).
FleetSync's bursts came out of that chain as clean bits, so every stage up
to the slicer was proven on air. What remained was one function —
`feedSymbol` — and a framer whose only anchor is a 40-bit sync word hunted
with five errors of tolerance. A decoder that never reaches a CRC failure
is one whose sync word never matches; a sync word that never matches on a
bit stream the FleetSync framer finds usable means the *bits are not what
the framer thinks they are*.

The plausible wrong explanations all sat one layer away. An inverted
discriminator? The framer already accepts the complemented sync (≥ 35 of
40 bits differing). Too little deviation? FleetSync's 1200/1800 Hz bursts
from the same room sliced fine. A wrong interleave or CRC? Those produce
framed-but-failing bursts, not silence. The sync constant? `0x07092A446F`
is the published value. Every one of these had a test, and every test was
green.

## What the decoder believed

The pre-fix `feedSymbol` did the obvious thing: track the discriminator's
bias, slice against it, call the result the bit.

```go
// internal/radio/mdc1200/afsk/receiver.go — as it shipped before #1220
// feedSymbol slices one recovered symbol to an NRZ bit (tracking DC
// bias with a slow EMA) and pushes it into the framer. Unlike APRS
// there is no NRZI decode — MDC1200 is plain NRZ.
func (r *Receiver) feedSymbol(s float32) {
    if !r.meanReady {
        r.meanEMA = s
        r.meanReady = true
    } else {
        r.meanEMA += (s - r.meanEMA) * (1.0 / 64.0)
    }
    var bit byte
    if s > r.meanEMA {
        bit = 1
    }
    r.inner.Push(bit)
    r.bitsEmitted.Add(1)
}
```

The `mdc1200` package doc says the same thing in so many words — *"Unlike
APRS the line code is plain NRZ, not NRZI"* — and the tests agreed, because
the test synthesiser FFSK-modulated the data bits directly, mark for 1 and
space for 0, so the decoder read back exactly what the encoder wrote.
Interleave correct, CRC correct (reflected CRC-16/CCITT, polynomial
`0x8408`, init 0, final XOR `0xFFFF`), bit order correct, opcode table
correct — all tested against a transmitter that does not exist. That is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its cleanest form: Season 1 catalogued seven cases where a round-trip
proved consistency and nothing else, and here the shared convention was
not a constant or a table but the mapping from tones to bits — the one
thing an FFSK round-trip exercises on *both* sides by construction.

## What the radio sends

The reference modem — Matthew Kaufman's `mdc-encode-decode`, the decoder
most MDC1200 tooling in the field is built on; GPL, read for protocol facts
only, nothing ported — describes the line code precisely. Each data bit is
compared with the previous one. If it is the **same**, the radio sends one
cycle of 1200 Hz. If it **changed**, 1.5 cycles of 1800 Hz. The tones at
1200 baud are MSK with modulation index 0.5, and the comparison is an XOR
precoder: the air carries the *transition* sequence, and the data is
recovered as the running XOR of the tone decisions.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Three rows over seven bit periods. The top row is the data bits 0 1 1 0 0 0 1. The middle row shows the tone the radio sends for each bit, decided by comparing it with the previous bit starting from zero: 1200 hertz when the bit is the same, 1800 hertz when it changed, giving 1200, 1800, 1200, 1800, 1200, 1200, 1800. The bottom rows show what the old receiver pushed into the framer, the tone decisions themselves, 1 0 1 0 1 1 0, which differ from the data, and the correct decode, the running XOR of the tone decisions, which reproduces the data row.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">XOR precoding: the air carries transitions, not bits</text>
  <text x="70" y="52" text-anchor="end" fill="currentColor" font-size="9">data bit</text>
  <text x="70" y="104" text-anchor="end" fill="currentColor" font-size="9">tone sent</text>
  <text x="70" y="156" text-anchor="end" fill="var(--fg-muted)" font-size="9">old slicer's "bit"</text>
  <text x="70" y="200" text-anchor="end" fill="var(--accent)" font-size="9">running XOR</text>
  <g font-size="10" text-anchor="middle" fill="currentColor">
    <text x="120" y="52">0</text><text x="200" y="52">1</text><text x="280" y="52">1</text><text x="360" y="52">0</text><text x="440" y="52">0</text><text x="520" y="52">0</text><text x="600" y="52">1</text>
  </g>
  <text x="360" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1200 Hz = same as the previous bit (start state 0) · 1800 Hz = changed</text>
  <g fill="none" stroke="currentColor" stroke-width="1.2">
    <path d="M90 104 q15 -14 30 0 q15 14 30 0"/>
    <path d="M170 104 q10 -14 20 0 q10 14 20 0 q10 -14 20 0"/>
    <path d="M250 104 q15 -14 30 0 q15 14 30 0"/>
    <path d="M330 104 q10 -14 20 0 q10 14 20 0 q10 -14 20 0"/>
    <path d="M410 104 q15 -14 30 0 q15 14 30 0"/>
    <path d="M490 104 q15 -14 30 0 q15 14 30 0"/>
    <path d="M570 104 q10 -14 20 0 q10 14 20 0 q10 -14 20 0"/>
  </g>
  <g font-size="8" text-anchor="middle" fill="currentColor">
    <text x="120" y="126">1200 Hz</text><text x="200" y="126">1800 Hz</text><text x="280" y="126">1200 Hz</text><text x="360" y="126">1800 Hz</text><text x="440" y="126">1200 Hz</text><text x="520" y="126">1200 Hz</text><text x="600" y="126">1800 Hz</text>
  </g>
  <g font-size="10" text-anchor="middle" fill="var(--fg-muted)">
    <text x="120" y="156">1</text><text x="200" y="156">0</text><text x="280" y="156">1</text><text x="360" y="156">0</text><text x="440" y="156">1</text><text x="520" y="156">1</text><text x="600" y="156">0</text>
  </g>
  <text x="640" y="156" fill="var(--fg-muted)" font-size="8">≠ data</text>
  <g font-size="10" text-anchor="middle" fill="var(--accent)" font-weight="bold">
    <text x="120" y="200">0</text><text x="200" y="200">1</text><text x="280" y="200">1</text><text x="360" y="200">0</text><text x="440" y="200">0</text><text x="520" y="200">0</text><text x="600" y="200">1</text>
  </g>
  <text x="640" y="200" fill="var(--accent)" font-size="8">= data</text>
</svg>
<figcaption>The tone decision says whether the bit changed, not what it is. Reading the tones as bits produced a stream that agreed with the data only by accident — and the 40-bit sync word, defined in the data domain, never matched.</figcaption>
</figure>

Two consequences follow, and both are tests. First, the 0x55 bit-sync
leader — alternating 0101… in the data domain — changes on every bit, so
under precoding it is **one continuous 1800 Hz tone** right up to the sync
word (`TestPrecodeLeaderIsContinuousSpaceTone`); a receiver reading tones
as data sees `0111…` where it expects `0101…`. Second, the start state: the
precoder starts from a previous bit of 0, but a receiver joining mid-stream
does not know the running XOR's state, and getting it wrong complements the
entire data stream. The tone *sense* is never ambiguous — an inverted FM
discriminator negates the audio, and a negated 1800 Hz tone is still
1800 Hz — so the start state is the only unknown, and it is a whole-stream
complement. The framer already handles exactly that: `Push` in
`mdc1200/receiver` accepts a sync match at Hamming distance ≤ `gdThresh`
(5) **or** ≥ 35 — the complemented sync word — and inverts the payload
back. Written for an inverted discriminator, a case FFSK cannot produce, it
turned out to be the mechanism the precoder needed.

## The fix

The receiver-side fix is the decode step:

```go
// internal/radio/mdc1200/afsk/receiver.go
func (r *Receiver) feedSymbol(s float32) {
    if s <= 0 {      // space, 1800 Hz: the data bit CHANGED
        r.data ^= 1
    }
    r.inner.Push(r.data)
    r.bitsEmitted.Add(1)
}
```

Zero threshold, not the old 1/64 EMA — the lesson measured on FleetSync,
where a bias tracker drifted toward a long single-tone run and flipped the
ISI-weakened bits after it. An MDC1200 leader *is* a long single-tone run,
so a tracked threshold drifts into every burst's preamble. `Reset` clears
`r.data` with every DSP stage, so a burst right after a retune starts from
the same state as a first one (`TestReceiverResetClearsPrecodingState`).

The synthesiser gained the transmit side: `BurstBits` lays out leader, sync
word and interleaved block as data bits, and `Precode` turns them into the
tone sequence. `fecParity` now generates the seven redundancy bytes after
the header — a rate-1/2 convolutional parity over the first seven bytes'
bits, LSB first, taps at register stages 0, 2, 5 and 6 — so synthetic
bursts match a radio's byte for byte (the decoder still does not use them
for correction), and the leader is `LeaderBytes` = 7, the reference
encoder's default.

## Three pins that share nothing with the decoder

A fix to a self-consistent bug needs tests that are not self-consistent.
Three landed, chosen so that no single wrong layout could satisfy all of
them.

**Reference-encoder bytes.** `synth_test.go` holds `referenceVectors`: for a
PTT ID (op `0x01`, arg `0x80`, unit `0x1234`) and a radio check (op `0x63`,
unit `0xABCD`), the exact 26 bytes the reference encoder queues — seven
leader bytes, five sync bytes, fourteen interleaved block bytes, MSB first —
and the 208-character tone string its sampler emits.
`TestBurstBitsMatchReferenceEncoderBytes` holds interleave, CRC, parity
bytes and bit order to those literals; `TestPrecodeMatchesReferenceToneSequence`
holds the precoding — the
[literal-vector]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
discipline, the only kind of test that catches drift from the air interface.

**Reference-encoder audio.** `afsk/testdata/mdc1200_ref_01_80_1234_48k.s16`
is 0.37 s of discriminator audio the reference modem itself produced at
48 kHz — 17 921 mono int16 samples, default leader, 68 % amplitude: data
the program emitted, not its code. `TestReceiverDecodesReferenceEncoderAudio`
pushes it through the new `Receiver.ProcessAudio` at three chunk sizes
(4096, 1000 and 1 sample) and requires exactly one CRC-valid PTT ID. The
reverse was done once, offline: the reference decoder decoded GopherTrunk's
synthesised audio.

**Real air.** The reporter's capture closed the loop — next section.

Every test in `airformat_test.go` — eight of them, including
`TestReceiverToneDecisionIsPolarityInvariant`, which conjugates the IQ and
inserts a stray change-tone between two bursts so the second arrives with
every data bit complemented — fails against the old slicer: 448 bits in,
zero sync locks. The failing-first bar was trivially met, because the old
code could not decode *any* correctly modulated burst.

## On air, 28 September

The precoding fix landed on 27 September, a day after the scanner-channel
wiring that lets `decoders: [mdc1200]` ride a `scanner.conventional` entry
instead of pinning a whole SDR. The reporter ran the fixed build the next
day and decoded **eleven PTT ID bursts** — start (op `0x01` / arg `0x80`)
and end (op `0x01` / arg `0x00`) — from unit `0x1777` on 447.100 MHz, on
narrow and wide FM, through the scanner-channel path.

They also posted the scanner's own 2.4 MS/s voice recording of a keyup
(`…_447100000_2400000_voice.cs16`, 2.18 s). The scanner opens that
recording on squelch, so the keyup PTT ID is cut off; what survives is the
end-of-transmission burst at t ≈ 1.3 s. `TestMDC1200Replay`
(`GT_MDC1200_IQ`, `GT_MDC1200_RATE`, `GT_MDC1200_UNIT`) found the carrier at
−1758 Hz and printed a CRC-valid PTT ID (end) — on a file ADC-clipped at
both rails (RMS −0.9 dBFS; the handheld was in the same room). Clipped is
not undecodable, as the
[#836 DMR captures]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
had shown a fortnight earlier. A 0.7 s channelized slice — the production
`ccdecoder.Downconverter` tuned to −1758 Hz, 48 kHz cs16, peak-normalised
to 0.9 — is committed as `afsk/testdata/mdc1200_unit1777_pttid_end_48k.cs16`,
and `TestMDC1200RealAirSlice` asserts exactly one CRC-valid `0x01`/`0x00`
burst from unit `0x1777` at two chunk sizes. With the reporter's live
confirmation and the real-air pin in the tree, #1220 was closed — the only
state in which the repository's rule allows it: a failing-first regression
passes, and the reporter has confirmed the symptom is gone.

## Where this goes next

The same reporter's lab produced the next three parts, all from one issue.
Their Kenwoods on the conventional analog scanner carried a high-pitched
tone that got worse the stronger the signal was, and the captures showed a
constellation squared by the ADC.
[Part 2]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }})
follows that whistle to a frequency exactly four times the carrier offset,
and to the reason the obvious fix — park the LO a quarter of the sample
rate away — would have made it worse.

## FAQ

**Why did GopherTrunk decode FleetSync but not MDC1200 on the same rig?**
Both share one DSP chain: `demod.FM`, resample to 9600 Hz, `demod.FFSK` at
1200/1800 Hz, Mueller-Müller timing, a slicer. FleetSync's line code is
plain NRZ, so the slicer's output is its data. MDC1200 is XOR-precoded: the
slicer's output is the transition sequence, and the data is its running
XOR. `afsk.feedSymbol` now applies that XOR.

**What is XOR precoding in MDC1200?**
The radio compares each data bit with the previous one and sends one cycle
of 1200 Hz if it is the same, 1.5 cycles of 1800 Hz if it changed. The
receiver recovers the data as the running XOR of the tone decisions.
`mdc1200.Precode` implements the transmit side; the 0x55 leader becomes one
continuous 1800 Hz tone under it.

**How does the receiver know the XOR's start state?**
It does not need to. A wrong start state complements the whole data stream,
and the framer in `mdc1200/receiver` accepts the sync word at ≤ `gdThresh`
(5) errors or ≥ 35 — the complement — then inverts the payload back.
`TestReceiverToneDecisionIsPolarityInvariant` pins both an inverted
discriminator and a stray change-tone before a burst.

**Is MDC1200 verified on air now?**
Yes. On 28 September the reporter's live run decoded eleven PTT ID bursts
from unit 0x1777 on 447.100 MHz through the scanner-channel path, and their
2.4 MS/s capture replays through `TestMDC1200Replay`. A 0.7 s slice is
committed as the real-air regression `TestMDC1200RealAirSlice`. Issue
#1220 is closed.

**What do the redundancy bytes after the header do?**
`data[7..13]` are convolutional parity over the first seven bytes — taps at
stages 0, 2, 5 and 6 over the LSB-first bits — generated by `fecParity` so
synthetic bursts match the reference encoder byte for byte. The decoder
captures them but does not yet correct with them.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: A Tone at Four Times the Offset — Zero-IF Clipping on the Analog Scanner]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }})
