---
title: "Rebuilt From the Air, Part 12: P25 Phase 2 II — Mirrored Taps, a Sign Error, and the Carrier Already in the Repo"
description: "What the P25 Phase 2 rebuild exposed around the frame format: a PN44 scrambler that fed back the polynomial's exponents, a Gardner timing loop with its error sign reversed, a real Phase 2 carrier sitting unnoticed in samples/mmr — and a first live call whose cleanly decoded frames played as noise through the wrong vocoder table set."
category: solution-postmortem
keywords: p25 phase 2 pn44 scrambler, lfsr tap positions, gardner timing recovery sign error, symbol timing loop test, constant modulus timing estimate, p25 phase 2 sacch decode, tdma framer iq hole, real air regression test, p25 phase 2 garbled audio ambe+2 3600x2450, gophertrunk p25 phase 2
tags: [rebuilt-from-the-air, p25, phase2, dsp, testing, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 12
---

*Part 12 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 11]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
replaced the P25 Phase 2 framing with the real air format. This part is what
that rebuild found underneath the framing — two primitives every Phase 2 path
shares, both wrong, and a real Phase 2 carrier that had been sitting in the
repository the whole time — and what the first live call found above it.*

> **TL;DR:** The Phase 2 rebuild exposed defects on both sides of the frame
> format. `framing.PN44Scrambler` fed back state bits 40/35/29/24/10/0 — the
> polynomial's *exponents* — instead of 43/33/19/14/8/3, and diverged from
> OP25 and SDRTrunk at bit 49; a literal 4320-bit sequence pins it. The
> shared `sync.Gardner` loop applied its error with the wrong sign, so every
> call's dibits decayed to ~50 % after ~16k symbols. A real Phase 2 carrier
> already in `samples/mmr` (#1364) pinned the MAC path and found two
> receive-side defects. Then on 18 Nov a reporter's live call framed and
> FEC-decoded cleanly — 175 of 175 MAC PDUs — and played as noise: Phase 2
> was mapped to D-STAR's AMBE 3600x2400 tables, not AMBE+2 3600x2450.
> Phase 2 voice is **on-air verified at the frame level**; the audio fix is
> pinned on that call, and the reporter's run on the fixed build is open.

**Key takeaways**

- **A property test cannot pin a sequence.** Round trip, determinism, seed
  sensitivity and a period bound all pass on the mirrored register; 4320
  literal bits from a decoder that works on air do not.
- **Test a timing loop's convergence point, not its symbol count.** The old
  Gardner tests used a triangle pulse and a quadrant check; both survive a
  half-symbol error.
- **Real air finds what no reference can.** An idle carrier decoded six of
  six SACCH bursts with the site's seed and none with the next one — and
  exposed a one-second timing pull-in and a framer lost for 720 ms.
- **Clean FEC counters are not decoded audio.** A live call with ten
  corrected bits in 115 voice bursts played as noise; no test had judged
  what the vocoder made of a real frame.

## Cheat sheet

| | |
|---|---|
| Issue | [#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273) (the Phase 2 rebuild), [#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364) (the carrier in `samples/mmr`) and the 18 Nov live-call report |
| Symptom | bodies descrambled with the wrong mask; dibits decaying to ~50 % about 3 s into a call; a first sync up to a second late; on a live call, clean counters and a WAV of noise |
| Wrong theories | weak air; a wrong seed; the descramble order (#915 — real, not sufficient); "the primitives have tests"; "it sounds encrypted" |
| Real cause | PN44 taps mirrored; Gardner error sign reversed; pull-in from the loop's unstable point; `AirFramer` flywheeling on a stale grid; Phase 2 frames sent to the AMBE 3600x2400 vocoder |
| Fix | taps at bits 43/33/19/14/8/3; `g.mu += g.sps - g.gain*err`; `Receiver.ArmTimingAcquisition` → `sync.EstimateSymbolPhaseCM`; re-anchor after one missed ISCH; `p25-phase2` → `ambe2-dmr` |
| Diagnostic | a literal reference sequence; a convergence-point test on RRC data; a wrong-seed control on a real carrier; b0 read from the `.raw` sidecar |
| Status | PN44 and framing **reference-pinned**; SACCH MAC path **capture-verified** on an idle carrier; Phase 2 voice **on-air verified at the frame level** (18 Nov); vocoder fix pinned on the committed call; the reporter's run on the fixed build still open |

## In this post

- **The symptom, and the theories that fit it** — a correct frame over two wrong primitives.
- **Defect one: PN44 tapped the exponents** — the mirrored register and the 4320 bits that pin it.
- **Defect two: the Gardner loop left the eye** — one sign, and tests that could not see it.
- **The carrier already in the repo, and what it found** — #1364, the wrong-seed control, a slow pull-in, a stale grid.
- **The first live call** — clean frames, noise for audio, and the vocoder table set.
- **What this overturns, and what we keep** — the corrections, the rules, the open list.

## The symptom, and the theories that fit it

After Part 11 the timeslot grid, the
[ISCH]({{ '/reference/p25-isch/' | relative_url }}), the DUID positions and
the MAC coding matched OP25 and SDRTrunk constant for constant. Two things
under that format were still GopherTrunk's own, and every Phase 2 path
shares them: the scrambler mask XORed over each burst body, and the timing
loop that turns matched-filter samples into dibits. With the mask wrong,
every voice frame and MAC PDU reaches its FEC as noise. With the loop
wrong, every call's dibits decayed to about 50 % after ~16k symbols —
roughly three seconds.

Neither symptom says "scrambler" or "timing loop". The theories on record
were all reasonable:

- **Weak air.** The notes had filed the Phase 2 MAC yield on the
  [#915](https://github.com/MattCheramie/GopherTrunk/issues/915) capture —
  0 of 17 RIDs at `mac_rs_valid=0` — under weak signal.
- **A wrong or missing seed.**
  [P25 End to End, Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
  warned that a bad seed "looks like a dead traffic channel". There *was* a
  seed bug — the Network Status Broadcast parser read the colour code from
  the channel bytes — but a right seed in a wrong register is the same dead
  channel.
- **The descramble order.** #915 moved the descramble ahead of the FEC, onto
  channel bits. Correct, and not enough.
- **"They have tests."** Both primitives had green suites for months.

One caveat: the notes record no re-run of the #915 capture after these
fixes, so this post does not claim that capture's yield is explained.

## Defect one: PN44 tapped the exponents

The [PN44 scrambler]({{ '/reference/pn44-scrambler/' | relative_url }}) is a
44-bit [LFSR]({{ '/reference/linear-feedback-shift-register/' | relative_url }})
with generator `x^44 + x^40 + x^35 + x^29 + x^24 + x^10 + 1`, seeded from
WACN, System ID and colour code. GopherTrunk's register shifts left, reads
its output from bit 43 and clocks the feedback into bit 0. In that form the
polynomial's `x^k` term taps bit `43 − k`: `x^40` is bit 3, and the constant
term is bit 43. The old code tapped bits 40, 35, 29, 24, 10 and 0 — the
exponents, read as bit positions.

```go
// internal/radio/framing/pn44_p25_phase2.go
func (s *PN44Scrambler) Next() byte {
	out := byte((s.state >> 43) & 1)
	fb := byte((s.state>>43 ^
		s.state>>33 ^
		s.state>>19 ^
		s.state>>14 ^
		s.state>>8 ^
		s.state>>3) & 1)
	s.state = ((s.state << 1) | uint64(fb)) & pn44Mask
	return out
}
```

Mirroring the taps does not break the register. It still shifts, still
depends on its seed, still emits a pseudo-random stream — just not the one a
P25 transmitter uses. So every existing test stayed green.
`TestPN44ScramblerRoundTrip` scrambles twice and gets its input back;
`TestPN44ScramblerSequenceIsDeterministic` and
`TestPN44ScramblerOutputDiffersAcrossSeeds` compare the register with
itself; `TestPN44LFSRPeriodLowerBound` checks the state does not recur
early. Each is a property the wrong register has.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The 44-bit PN44 shift register drawn from bit 43 on the left, where the output is read, to bit 0 on the right, where the feedback bit enters. Above it are the six taps the old code used, at bits 40, 35, 29, 24, 10 and 0; below it are the six correct taps, at bits 43, 33, 19, 14, 8 and 3.">
  <text x="340" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="9">before: taps at the exponents — bits 40, 35, 29, 24, 10, 0 (the mirrored sequence)</text>
  <g stroke="var(--fg-muted)" fill="none">
    <line x1="104.5" y1="42" x2="104.5" y2="80"/><line x1="168.2" y1="42" x2="168.2" y2="80"/><line x1="244.5" y1="42" x2="244.5" y2="80"/>
    <line x1="308.2" y1="42" x2="308.2" y2="80"/><line x1="486.4" y1="42" x2="486.4" y2="80"/><line x1="613.6" y1="42" x2="613.6" y2="80"/>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--fg-muted)">
    <text x="104.5" y="36">40</text><text x="168.2" y="36">35</text><text x="244.5" y="36">29</text>
    <text x="308.2" y="36">24</text><text x="486.4" y="36">10</text><text x="613.6" y="36">0</text>
  </g>
  <rect x="60" y="80" width="560" height="28" fill="none" stroke="currentColor"/>
  <text x="340" y="98" text-anchor="middle" fill="currentColor" font-size="9">44-bit state, shifting left one bit per clock</text>
  <text x="54" y="98" text-anchor="end" fill="currentColor" font-size="8">out ← 43</text>
  <text x="626" y="98" text-anchor="start" fill="currentColor" font-size="8">0 ← fb</text>
  <g stroke="var(--accent)" fill="none">
    <line x1="66.4" y1="108" x2="66.4" y2="146"/><line x1="193.6" y1="108" x2="193.6" y2="146"/><line x1="371.8" y1="108" x2="371.8" y2="146"/>
    <line x1="435.5" y1="108" x2="435.5" y2="146"/><line x1="511.8" y1="108" x2="511.8" y2="146"/><line x1="575.5" y1="108" x2="575.5" y2="146"/>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--accent)">
    <text x="66.4" y="158">43</text><text x="193.6" y="158">33</text><text x="371.8" y="158">19</text>
    <text x="435.5" y="158">14</text><text x="511.8" y="158">8</text><text x="575.5" y="158">3</text>
  </g>
  <text x="340" y="186" text-anchor="middle" fill="var(--accent)" font-size="9">after: the x^k term taps bit 43 − k — bits 43, 33, 19, 14, 8, 3 (OP25 and SDRTrunk, all 4320 bits)</text>
</svg>
<figcaption>Same register, same polynomial on paper: reading the exponents as bit positions mirrors every tap about the register's centre, and the mirrored register generates a different sequence.</figcaption>
</figure>

It also hid on a casual comparison. The first 44 bits out of this register
are the seed itself, whatever the taps — the reference vector opens
`00001018293`: WACN `0x00001`, System `0x018`, NAC `0x293` — so on that
identity the first wrong bit is bit 49. `AirScrambleMask` takes timeslot
`t`'s body mask from superframe bits `[20+360t, 340+360t)`, so everything
but the opening dibits of timeslot 0 was descrambled with the wrong mask.

The pin is a
[literal vector]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}),
not another property. `TestPN44MatchesOP25AndSDRTrunk` holds the whole
4320-bit outbound superframe sequence for that identity as hex in the test
file, as OP25's `apps/tdma/lfsr.py` generates it; SDRTrunk's
`LinearFeedbackShiftRegister` produces the identical 4320 bits. Both
decoders descramble live Phase 2 air with it — the property no round trip
has. The sequence is a protocol fact; no reference code was ported.

## Defect two: the Gardner loop left the eye

[Gardner timing recovery]({{ '/reference/gardner-timing-recovery/' | relative_url }})
takes one sample at the symbol instant and one at the midpoint, and forms
`e = Re{ conj(s[n] − s[n−1]) · m[n] }`. The error is positive when the
samples are *late* — the midpoint has passed the transition's zero crossing
— so a positive error must shorten the next interval:

```go
// internal/dsp/sync/gardner.go (shape)
diff := complex64(sym - g.prevSym)
err := float64(real(diff)*real(midSym) + imag(diff)*imag(midSym))
g.mu += g.sps - g.gain*err // the old code added the error
```

Flip that sign and the equilibria swap: the eye centre becomes the unstable
point and the transition zero crossing the stable one. From any start the
loop walks half a symbol off and stays. At the Phase 2 chain's gain of
0.005 the walk took about 16,000 symbols — a call began decodable, and
three seconds later its dibits were noise.

The old tests could not see it, and both reasons are about the fixture.
`TestGardnerRecoversAlignedQPSK` and `TestGardnerLocksOnPhaseOffset` build
each symbol as its constellation point times a taper confined to its own
period, so no intersymbol interference punishes a wrong phase. And they
score by symbol *count* and *quadrant* match: a triangle pulse has the
right sign almost anywhere inside its symbol, and a loop emits one symbol
per period wherever it sits.

`TestGardnerConvergesOnEyeCentre` tests what the loop is for. It runs
40,000 random QPSK symbols through a
[root-raised-cosine]({{ '/reference/root-raised-cosine-filter/' | relative_url }})
pair — the matched-filter output every caller feeds it — from four start
phases, at gain 0.005, and measures the last 5,000 symbols. At the eye
centre `|re|` and `|im|` cluster about their mean; at the zero crossing
they spread across the whole range. The relative spread must stay under 0.1.

The loop is shared — the TETRA receiver's `ClockGardner` mode and the P25
Phase 1 CQPSK demodulator construct it too — but the notes record the
damage on Phase 2 only, and this post claims no more.

## The carrier already in the repo

Both fixes were **reference-pinned** and nothing more. Then
[#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364) noticed
the capture had been in the tree all along. `samples/mmr` holds nine
`mmr-s9-*.cfile` grabs, 2.4 MS/s centred on 420.0125 MHz, used until then
for the Phase 1 control channel at +37.5 kHz — WACN `BEE00`, system `164`,
NAC `164`. At +637.5 kHz in every one of them sits the network's idle voice
channel, 420.650 MHz: a continuous Phase 2 TDMA carrier.

An idle carrier has no voice, but it does carry signalling: the
[SACCH]({{ '/reference/p25-sacch-facch/' | relative_url }}) on timeslots 10
and 11, fill elsewhere. `TestRealAirMMRPhase2MACDecodes` runs the production
receiver's dibits through the `AirFramer` and `DecodeAirMAC` twice — with
the seed built from the control channel's identity, and with the colour
code off by one:

```text
realair_mmr_test.go:86: 33 bursts, 6 SACCH-S, 6 CRC-valid with the site's seed, 0 with colour code 0x165
```

Six of six with `PN44SeedFromIdentity(0xBEE00, 0x164, 0x164)`, none with
`0x165`. That control is the point: it shows the descramble is what makes
the bursts valid. It was the first real-air pin of the rebuilt path —
air-format framing, RS(63,35), CRC-12 and PN44 together.

The content cross-checks too. The Phase 1 control channel in the same
capture announces a Motorola
[patch]({{ '/reference/patch-multigroup/' | relative_url }}): supergroup
3202 carrying talkgroup 201. The Phase 2 SACCH bursts decode as `MAC_IDLE`
PDUs holding `81 90 0b 0c 82 00 c9 …` — `0x0C82` is 3202, `0x00C9` is 201.
Two channels, two FEC chains, one regroup.

## What real air found

**A pull-in of up to a second.** A voice chain starts at the grant, at an
arbitrary sub-symbol phase. Started at the eight sub-symbol phases of each
of the nine captures, the corrected loop found its first S-ISCH 8–100 ms in
at most phases, 0.15–1.0 s in at about one in four, and not within the
1.09 s capture at two: near a half-symbol offset a feedback loop has almost
no error to pull on. MAC_PTT, the burst that names a call's source radio,
is sent at key-up, so a slow start costs exactly that.

The fix is the feed-forward acquisition
[DMR direct mode]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
needed in #836, re-derived for a PSK signal. `sync.EstimateSymbolPhaseCM`
is a constant-modulus eye search: at the decision instant every sample of
an RRC-matched PSK signal has the same magnitude, so `E|y|⁴ / E|y|²²`
measures about 1.01 there and at least 1.67 on noise, with no carrier lock
needed. `Receiver.ArmTimingAcquisition` seeds the loop with
`Gardner.SetPhase` once two consecutive 96-symbol windows agree within one
sample. It is armed in the voice chain and the signalling follower only.
On the committed slice the two slow start phases go from 0.992 s and
0.272 s to 0.062 s.

**A framer on a stale grid.** `mmr-s9-3.cfile` has a hole about 0.26 s in:
the on-air timeslot grid moves by 155 dibits, consistent with 25 dropped
16 KiB USB buffers. The `AirFramer` kept cutting on the old grid until
`airLockLossDibits` — two superframes, 720 ms — passed without a recognised
ISCH, and lost 24 of 33 bursts, handing out packets with random DUIDs. It
now re-anchors after **one** missed ISCH (`airReanchorMisses`), searching
for an off-grid S-ISCH from *inside* the packet that missed, where the new
grid's first S-ISCH sat. `TestAirFramerReanchorsAfterIQHole` reports 34
bursts, 2 off the idle pattern, 6 SACCH CRC-valid.

The nine files are separate `rtl_sdr` grabs, not one recording, and five of
them contain a hole. Never concatenate them, and check S-ISCH spacing before
blaming the receiver — the subject of
[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }}).

## The first live call: clean frames, noise for audio

An idle carrier cannot test voice. On 18 November another operator's rig
did: a wideband RTL-SDR on an 867 MHz site — Phase 1 control channel,
Phase 2 voice, clear in DSD+ — whose recordings were "garbled noise, as if
it is encrypted". The call's counters said the air path was fine:
`air_bursts=579`, `voice_bursts=115`, `corrected_bit_errs=10`,
`mac_bursts=175`, `mac_pdus=175`. That is the frame, the scrambler and the
timing loop working on a live call — Phase 2 voice **on-air verified at the
frame level**. The WAV was still noise.

The fourth defect sat above the FEC. `DefaultVocoderForProtocol` sent
`p25-phase2` to `ambe2`, the AMBE 3600x2400 table set only D-STAR uses.
Phase 2 carries AMBE+2 3600x2450, the half-rate vocoder DMR and NXDN
already decode through `ambe2-dmr`. The two layouts share the top four
pitch bits and nothing else, so the output is speech-shaped noise. Every
Phase 2 test used synthetic frames and none judged the audio; before
v1.3.1 no real frame had reached the vocoder.

The proof came from the reporter's `.raw` sidecar, before any DSP theory.
Read with the 2450 layout, the pitch field b0 glides like a voice and reads
124 on silence frames; mbelib's 3600x2450 decode matches `unpackParams2450`
on all 118 frames; and the reporter's WAV tracked the 2400 decode
(correlation 0.86). That call is now `testdata/p25p2-voice.raw`, and
`TestP25Phase2DefaultVocoderDecodesRealAirSpeech` holds the recorder's
Phase 2 vocoder to mbelib's level track on it: 0.22 before, 0.92 now.

The same log showed in-call source updates decoding live — too often. A
traffic channel repeats GROUP_VOICE_CHANNEL_USER on every FACCH and SACCH
burst, and the chain published each: 79 `call source update` lines in four
seconds over three calls. `callSourceFilter` now passes one only when the
talker or flags change.

## What this overturns

[P25 End to End, Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
lists "Gardner timing" as a stage of a working Phase 2 chain and treats the
PN44 as correctly applied; when it was written the loop's sign was wrong
and the register mirrored.
[Protocol Decoders, Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})
makes the same assumption. Both carry an update pointing at Part 11 for the
framing; this part corrects the primitives. The
[PN44 glossary entry]({{ '/reference/pn44-scrambler/' | relative_url }})'s
figure and snippet still tap bits 0, 10, 24, 29, 35 and 40. And
[Voice Coding, Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
shows the `p25-phase2` → `ambe2` mapping the live call overturned.

## What we keep

- **Pin a generator against a sequence, never against itself.** The
  [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  does not need an encoder: a primitive tested only by its own properties
  is already inside it.
- **Give a timing loop real pulses and measure where it lands.** Symbol
  count and quadrant are the two things a half-symbol error leaves intact.
- **Search your own samples.** A capture made for one channel records
  everything else in the span.
- **Real-air pins need a negative control.** "It decodes" is weak; "it
  decodes with this seed and not the next one" is evidence.
- **A hole is a property of the file.** Check sync spacing, and never join
  separate grabs into one stream.
- **Judge the output, not only the counters.** Clean FEC counters and noise
  for audio point above the FEC: read b0 from the `.raw` before any DSP
  theory.

Status: PN44 and the air framing are **reference-pinned**; the MAC path is
**capture-verified** on an idle carrier; Phase 2 voice is **on-air verified
at the frame level**, with the vocoder fix pinned on the committed call.
Open, from that log: the reporter's run on the fixed build; a second call
(+787 kHz, logical channel 0) with `corrected_bit_errs=356` over 48 voice
bursts beside 70 of 72 MAC PDUs — about two Golay corrections a voice
frame, unexplained until a `voice_iq_debug` capture exists; a tap that got
`air_bursts=0` for a timeslot-1 grant while another framed the same
carrier; the 2V burst's ESS-B, undecoded, so the encryption sync comes from
MAC_PTT only; and MAC opcodes `0x42` and `0x7C`, which print unnamed. A
talker alias from a live call is still not confirmed, and neither is the
#1273 reporter's site. Like the SmartNet capture in
[From Spec to Shipping, Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}),
each is an on-air gate, not a to-do item.

## Where this goes next

[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
turns that hole into a habit: six captures that were not what the report
said, starting with the "System Fusion" file that was an FM test tone.

## FAQ

**Why did the PN44 scrambler pass its tests with the wrong taps?**
Because every test compared the register with itself. Round trip,
determinism, seed sensitivity and a period bound are all true of the
mirrored register. Only a literal sequence from OP25 and SDRTrunk fails it.

**What does a Gardner loop with the wrong error sign look like on air?**
Not like a dead receiver. A call decodes at first; then the loop walks half
a symbol off, to the zero crossing, and the dibits decay to about 50 % —
roughly three seconds in at GopherTrunk's Phase 2 gain.

**Is GopherTrunk's P25 Phase 2 decode verified on air?**
At the frame level, yes. On 18 Nov a reporter's live call framed and
FEC-decoded cleanly: 175 of 175 MAC PDUs, ten corrected bits over 115 voice
bursts. Its audio was noise until the vocoder mapping was fixed; that fix
is pinned on the recorded call, and the reporter's run on the fixed build
is still open. A talker alias from a live call is not confirmed.

**Why does a P25 Phase 2 call sound garbled or encrypted when the decode counters are clean?**
It was the vocoder table set. Phase 2 frames went to the AMBE 3600x2400
decoder only D-STAR uses; Phase 2 is AMBE+2 3600x2450, and the layouts
share only the top four pitch bits. The recorder now uses `ambe2-dmr`; an
older `.raw` sidecar re-decodes with
`gophertrunk decode -in call.raw -out call.wav -vocoder ambe2-dmr`.

## Series navigation

**Part 12 of 14** · ←
[Part 11: P25 Phase 2 I — Twelve Timeslots and the Two That Swap]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
· Next →
[Part 13: Read the Capture First — The 'YSF' File That Was an FM Test Tone]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
