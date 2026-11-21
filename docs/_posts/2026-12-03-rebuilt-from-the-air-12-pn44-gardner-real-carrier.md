---
title: "Rebuilt From the Air, Part 12: P25 Phase 2 II — Mirrored Taps, a Sign Error, and the Carrier Already in the Repo"
description: "Three shared-primitive defects the P25 Phase 2 rebuild exposed: a PN44 scrambler that fed back the polynomial's exponents instead of the right register bits, a Gardner timing loop that applied its error with the wrong sign, and a real Phase 2 carrier sitting unnoticed in samples/mmr that finally pinned the rebuilt MAC path to real air."
category: solution-postmortem
keywords: p25 phase 2 pn44 scrambler, lfsr tap positions, gardner timing recovery sign error, symbol timing loop test, constant modulus timing estimate, p25 phase 2 sacch decode, tdma framer iq hole, real air regression test, h-dqpsk receiver, gophertrunk p25 phase 2
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
that rebuild found underneath the framing: two primitives every Phase 2 path
shares, both wrong, and a real Phase 2 carrier that had been sitting in the
repository the whole time.*

> **TL;DR:** The Phase 2 rebuild exposed three defects below the frame
> format. `framing.PN44Scrambler` fed back state bits 40/35/29/24/10/0 — the
> polynomial's *exponents* — instead of 43/33/19/14/8/3, and diverged from
> OP25 and SDRTrunk at bit 49; a literal 4320-bit OP25 sequence pins it now.
> The shared `sync.Gardner` loop applied its error with the wrong sign, so it
> settled half a symbol off and every call's dibits decayed to ~50 % after
> ~16k symbols (~3 s). And a real Phase 2 carrier was already in
> `samples/mmr`: the idle MMR voice channel, +637.5 kHz in every
> `mmr-s9-*.cfile`
> ([#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364)). Its
> SACCH decodes CRC-valid only with
> `PN44SeedFromIdentity(0xBEE00,0x164,0x164)` and carries the patch the
> control channel announces — and it found two receive-side defects of its
> own. Not confirmed: a source RID or alias from a live call.

**Key takeaways**

- **A property test cannot pin a sequence.** Round trip, determinism, seed
  sensitivity and a period bound all pass on the mirrored register; 4320
  literal bits from a decoder that works on air do not.
- **Test a timing loop's convergence point, not its symbol count.** The old
  Gardner tests used a triangle pulse and a quadrant check, and both survive
  a half-symbol error.
- **A wrong-seed control is what makes a real-air pin mean something.** Six
  of six SACCH bursts decode with the site's seed and none with colour code
  `0x165`, so the descramble is what makes them valid.
- **Real air finds what no reference can.** The carrier exposed a timing
  pull-in of up to a second and a framer that cut garbage for 720 ms after
  an IQ hole — neither one a format question.

## Cheat sheet

| | |
|---|---|
| Issue | [#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273) (the Phase 2 rebuild) and [#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364) (the carrier in `samples/mmr`) |
| Symptom | every burst body descrambled with the wrong mask; a call's dibits decaying to ~50 % about 3 s in; on real air, a first sync up to a second late at one start phase in four |
| Wrong theories | weak air; a wrong or missing seed; the descramble order (#915 — real, not sufficient); "both primitives have passing tests" |
| Real cause | PN44 feedback taps mirrored; Gardner error applied with the wrong sign; Gardner pull-in from its unstable point; `AirFramer` flywheeling on a stale grid |
| Fix | taps at bits 43/33/19/14/8/3; `g.mu += g.sps - g.gain*err`; `Receiver.ArmTimingAcquisition` → `sync.EstimateSymbolPhaseCM`; re-anchor after one missed ISCH |
| Diagnostic | a literal reference sequence; a convergence-point test on RRC data; a real carrier decoded with a wrong-seed control |
| Status | PN44 and framing **reference-pinned**; the SACCH MAC path **capture-verified** on an idle carrier; no live call, source RID or alias yet |

## In this post

- **The symptom, and the theories that fit it** — a correct frame over two wrong primitives.
- **Defect one: PN44 tapped the exponents** — the mirrored register and the 4320 bits that pin it.
- **Defect two: the Gardner loop left the eye** — one sign, and tests that could not see it.
- **The carrier already in the repo** — #1364, the idle MMR voice channel, and the wrong-seed control.
- **What real air found** — a slow pull-in, a stale grid, and nine files that are not one recording.
- **What this overturns, and what we keep** — the corrections and the rules.

## The symptom, and the theories that fit it

After [Part 11]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
the timeslot grid, the [ISCH]({{ '/reference/p25-isch/' | relative_url }}),
the DUID positions and the MAC coding matched OP25 and SDRTrunk constant for
constant. Two things under that format were still GopherTrunk's own, and
every Phase 2 path shares them: the scrambler mask XORed over each burst
body, and the timing loop that turns matched-filter samples into the dibits
the framer cuts. With the mask wrong, every voice frame and MAC PDU reaches
its FEC as noise. With the loop wrong, the project's notes record the
shape exactly: every call's dibits decayed to about 50 % after ~16k symbols
— roughly three seconds at 8 samples per symbol.

Neither symptom says "scrambler" or "timing loop". The theories on record
were all reasonable:

- **Weak air.** The notes had filed the Phase 2 MAC yield on the
  [#915](https://github.com/MattCheramie/GopherTrunk/issues/915) capture —
  0 of 17 RIDs at `mac_rs_valid=0` — under weak signal.
- **A wrong or missing seed.**
  [P25 End to End, Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
  warned that a bad seed "looks like a dead traffic channel". There *was* a
  seed bug — the Network Status Broadcast parser read the colour code from
  the channel bytes and installed a wrong seed, now pinned by
  `TestAsNetworkStatusBroadcastMatchesReferenceLayout` — but a right seed in
  a wrong register is the same dead channel.
- **The descramble order.** #915 moved the descramble ahead of the FEC, onto
  channel bits. That was correct, and it was not enough.
- **"They have tests."** Both primitives had green suites for months.

One caveat belongs here rather than at the end: the notes do not record a
re-run of the #915 capture after these fixes, so this post does not claim
that capture's yield is explained. It claims three defects that would each
produce it.

## Defect one: PN44 tapped the exponents

The [PN44 scrambler]({{ '/reference/pn44-scrambler/' | relative_url }}) is a
44-bit [LFSR]({{ '/reference/linear-feedback-shift-register/' | relative_url }})
with generator `x^44 + x^40 + x^35 + x^29 + x^24 + x^10 + 1`, seeded from
WACN, System ID and colour code. GopherTrunk's register shifts left, reads
its output from bit 43 and clocks the feedback into bit 0. In that form the
polynomial's `x^k` term taps bit `43 − k`: `x^40` is bit 3, `x^35` bit 8,
`x^29` bit 14, `x^24` bit 19, `x^10` bit 33, and the constant term bit 43.
The old code tapped bits 40, 35, 29, 24, 10 and 0 — the exponents, read as
bit positions.

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
depends on its seed, still emits a long pseudo-random stream — just not the
one a P25 transmitter uses. Which is why every existing test stayed green.
`TestPN44ScramblerRoundTrip` scrambles twice and gets its input back.
`TestPN44ScramblerSequenceIsDeterministic` and
`TestPN44ScramblerOutputDiffersAcrossSeeds` compare the register with
itself. `TestPN44LFSRPeriodLowerBound` checks the state does not recur
within 10,000 clocks. Every one is a property the wrong register has.

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

It also hid well on a casual comparison. The first 44 bits out of this
register are the seed itself, most significant bit first, whatever the taps
are — the reference vector in the test file opens `00001018293`, which is
WACN `0x00001`, System `0x018` and NAC `0x293` packed end to end. So the two
registers agree for at least 44 bits by construction; on that identity the
first wrong bit is bit 49. After it the mask is simply a different one, and
`AirScrambleMask` takes timeslot `t`'s body mask from superframe bits
`[20+360t, 340+360t)` — so everything but the opening dibits of timeslot 0
was descrambled with the wrong mask.

The pin is a
[literal vector]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}),
not another property. `TestPN44MatchesOP25AndSDRTrunk` holds the whole
4320-bit outbound superframe sequence for that identity as hex in the test
file, as OP25's `apps/tdma/lfsr.py` generates it; SDRTrunk's
`LinearFeedbackShiftRegister` produces the identical 4320 bits. A second,
256-bit vector covers an MMR identity. Both references descramble live
Phase 2 air with that sequence, which is the property no round trip has.
The sequence is a fact about the protocol; no reference code was ported.

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

Flip that sign and the two equilibria swap. The eye centre becomes the
unstable point and the transition zero crossing the stable one: from any
start the loop walks to half a symbol off and stays there. At the Phase 2
chain's gain of 0.005 the walk took about 16,000 symbols. A call began
decodable, and three seconds later its dibits were noise.

The old tests could not see it, for two reasons that are both about the
fixture. `TestGardnerRecoversAlignedQPSK` and
`TestGardnerLocksOnPhaseOffset` build each symbol as its constellation
point times a taper confined to its own symbol period — a raised-cosine
ramp in one, a triangle in the other. There is no intersymbol interference
to punish a wrong sampling phase. And they score the result by recovered
symbol *count* and by *quadrant* match above 0.85 and 0.80. A triangle
pulse has the right sign almost anywhere inside its symbol, and a loop
emits one symbol per period wherever it sits.

`TestGardnerConvergesOnEyeCentre` tests the thing the loop is for. It runs
40,000 random QPSK symbols through a
[root-raised-cosine]({{ '/reference/root-raised-cosine-filter/' | relative_url }})
pair (8 samples per symbol, span 8, α 0.2) — the matched-filter output
every caller feeds it — from four start phases, in 4096-sample chunks, at
gain 0.005. Then it measures the last 5,000 symbols: at the eye centre
`|re|` and `|im|` cluster tightly about their mean, at the zero crossing
they spread across the whole range. The relative spread must be under 0.1.

`sync.Gardner` is shared: the TETRA receiver's `ClockGardner` mode and the
P25 Phase 1 CQPSK demodulator construct the same loop. The notes record the
damage on Phase 2 only, and this post claims no more than that.

## The carrier already in the repo

Both fixes were **reference-pinned** and nothing more. The rebuild's own
note said a Phase 2 capture was the on-air gate. Then
[#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364) noticed
the capture had been in the tree all along. `samples/mmr` holds nine
`mmr-s9-*.cfile` grabs, 2.4 MS/s centred on 420.0125 MHz, recorded for the
Phase 1 control channel at +37.5 kHz — WACN `BEE00`, system `164`, NAC
`164`. At +637.5 kHz in every one of them sits the network's idle voice
channel, 420.650 MHz: a continuous Phase 2 TDMA carrier.

An idle carrier carries no voice, but it does carry signalling: the
[SACCH]({{ '/reference/p25-sacch-facch/' | relative_url }}) on timeslots 10
and 11 and fill elsewhere. `TestRealAirMMRPhase2MACDecodes` runs the
production receiver's dibits through the `AirFramer` and `DecodeAirMAC`
twice — once with the seed built from the control channel's identity, once
with the colour code off by one:

```text
realair_mmr_test.go:86: 33 bursts, 6 SACCH-S, 6 CRC-valid with the site's
    seed, 0 with colour code 0x165
```

Six of six with `PN44SeedFromIdentity(0xBEE00, 0x164, 0x164)`, none with
`0x165`. That control is the point of the test: it shows the descramble —
the register from the previous section — is what makes the bursts valid.
It was the first real-air pin of the rebuilt path: air-format framing,
RS(63,35), CRC-12 and PN44 together.

The content cross-checks too. The Phase 1 control channel in the same
capture announces a Motorola
[patch]({{ '/reference/patch-multigroup/' | relative_url }}): supergroup
3202 carrying talkgroup 201. The Phase 2 SACCH bursts decode as `MAC_IDLE`
PDUs holding the bytes `81 90 0b 0c 82 00 c9 …` — MFID `0x90`, opcode
`0x81`, `0x0C82` = 3202, `0x00C9` = 201. Two channels, two modulations, two
FEC chains, one regroup
([Trunking Engine, Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
has what a supergroup is).

## What real air found

A reference tells you the format. It cannot tell you how your receiver
behaves on a carrier that is already up, or on a file with a hole in it.
The carrier found one defect of each kind.

**A pull-in of up to a second.** A voice chain starts at the grant, at an
arbitrary sub-symbol phase. Started at the eight sub-symbol phases of each
of the nine captures, the corrected Gardner loop found its first S-ISCH
8–100 ms in at most phases, 0.15–1.0 s in at about one in four, and not
within the 1.09 s capture at two. Near a half-symbol offset — its unstable
equilibrium — a feedback loop has almost no error to pull on. MAC_PTT, the
burst that names a call's source radio, is sent at key-up, so a slow start
costs exactly that.

The fix is the feed-forward acquisition
[DMR direct mode]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
needed in #836, re-derived for a PSK signal. `sync.EstimateSymbolPhaseCM`
is a constant-modulus eye search: at the decision instant every sample of
an RRC-matched PSK signal has the same magnitude, so `E|y|⁴ / E|y|²²`
measures about 1.01 there and at least 1.67 on noise. It depends only on
`|y|`, so it needs no carrier lock. `Receiver.ArmTimingAcquisition` skips
10 symbols, measures 96-symbol windows, and seeds the loop with
`Gardner.SetPhase` once two consecutive windows agree within one sample;
after four windows without agreement it gives up and the loop pulls in
alone. It is armed in the voice chain and the signalling follower only, and
an unarmed receiver is byte-identical. The committed slice shows the effect:

```text
start phase 0: first S-ISCH armed 0.062 s, loop alone 0.992 s
start phase 7: first S-ISCH armed 0.062 s, loop alone 0.272 s
```

**A framer on a stale grid.** `mmr-s9-3.cfile` has a hole about 0.26 s in:
the receiver's symbol count stays exact, but the on-air timeslot grid moves
by 155 dibits, consistent with 25 dropped 16 KiB USB buffers. The
`AirFramer` kept cutting packets on the old grid until `airLockLossDibits`
— two superframes, 720 ms — passed without a recognised ISCH. On this hole
it lost 24 of 33 bursts, handing out packets with random DUIDs, voice among
them. It now re-anchors after **one** missed ISCH (`airReanchorMisses`),
searching for an off-grid S-ISCH from *inside* the packet that missed,
because a hole shortens the stream and the new grid's first S-ISCH sat
there. `TestAirFramerReanchorsAfterIQHole` reports 34 bursts, 2 off the
idle pattern, 6 SACCH CRC-valid.

The hole taught a rule about the fixture as well. The nine files are
separate `rtl_sdr` grabs, not one recording, and five of them contain a
hole. Never concatenate them, and check S-ISCH spacing before blaming the
receiver — the subject of
[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }}).

## What this overturns

[P25 End to End, Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
lists "Gardner timing" as a stage of a working Phase 2 chain and describes
the PN44 as correctly seeded and applied; when it was written the loop's
sign was wrong and the register mirrored.
[Protocol Decoders, Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }})
makes the same assumption about the descrambler. Both posts now carry an
update pointing at Part 11 for the framing; this part is the correction for
the two primitives. One more: the
[PN44 glossary entry]({{ '/reference/pn44-scrambler/' | relative_url }})'s
figure and snippet still tap bits 0, 10, 24, 29, 35 and 40. Read it with
the register above.

## What we keep

- **Pin a generator against a sequence, never against itself.** The
  [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  does not need an encoder: a primitive tested only by its own properties
  is already inside it. And compare the whole superframe — a register that
  outputs its seed first agrees with anything for 44 bits.
- **Give a timing loop real pulses and measure where it lands.** Symbol
  count and quadrant are the two things a half-symbol error leaves intact.
- **Search your own samples.** A capture made for one channel records
  everything else in the span. The carrier that closed this gate had been
  committed for another test.
- **Real-air pins need a negative control.** "It decodes" is weak; "it
  decodes with this seed and not with the next one" is evidence.
- **A hole is a property of the file.** Check sync spacing, and never join
  separate grabs into one stream.

Status: the PN44 and the air framing are **reference-pinned**; the MAC path
is **capture-verified** on an idle carrier, and the timing seed and the
re-anchor are pinned on the same real slices. Nothing a live call carries —
voice, a source RID, a talker alias — is confirmed, because no call is in
these captures. Like the SmartNet capture in
[From Spec to Shipping, Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}),
that is an on-air gate, not a to-do item.

## Where this goes next

[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
turns the hole in `mmr-s9-3.cfile` into a general habit: six captures that
were not what the report said, and the triage instrument each one left
behind — starting with the "System Fusion" file that was an FM test tone.

## FAQ

**Why did the PN44 scrambler pass its tests with the wrong taps?**
Because every test compared the register with itself. Scrambling twice
restores the input, the same seed repeats the same sequence, different
seeds differ, and the state does not recur early — all true of the mirrored
register. Only a literal sequence from OP25 and SDRTrunk could fail it.

**What does a Gardner loop with the wrong error sign look like on air?**
Not like a dead receiver. The loop starts wherever the stream starts, so a
call can decode at first; then it walks to the transition zero crossing,
half a symbol off, and the dibits decay to about 50 %. At GopherTrunk's
Phase 2 gain that took roughly 16,000 symbols, about three seconds.

**Is GopherTrunk's P25 Phase 2 decode verified on air?**
Not yet. The framing and scrambler are reference-pinned against OP25 and
SDRTrunk, and the SACCH MAC path is capture-verified on a real idle carrier
in `samples/mmr`. A live call — voice, a source RID, a talker alias — has
not been confirmed, because no call is in those captures.

**Why can't the nine mmr-s9 files be joined into one long test capture?**
They are separate `rtl_sdr` grabs, and five of them contain a hole where
whole USB buffers were dropped. Joining them adds eight more
discontinuities to a TDMA grid that is only valid between them. Use each
file alone and check S-ISCH spacing first.

**How does the feed-forward timing estimate work without a carrier lock?**
`sync.EstimateSymbolPhaseCM` looks only at magnitude. After the matched
filter a PSK signal has constant modulus at the decision instant and a
varying one between instants, so the sampling phase with the lowest
`E|y|⁴ / E|y|²²` is the symbol instant. A frequency offset rotates the
samples and leaves their magnitudes alone.

## Series navigation

**Part 12 of 14** · ←
[Part 11: P25 Phase 2 I — Twelve Timeslots and the Two That Swap]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }})
· Next →
[Part 13: Read the Capture First — The 'YSF' File That Was an FM Test Tone]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
