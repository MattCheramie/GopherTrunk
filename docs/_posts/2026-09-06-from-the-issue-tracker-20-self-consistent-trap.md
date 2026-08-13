---
title: "From the Issue Tracker, Part 20: The Self-Consistent Trap — Round-Trip Tests That Validate Their Own Bugs"
description: Seven GopherTrunk bugs shipped behind green test suites because the encoder and decoder shared the same wrong constant, table, or convention — a wrong sync word, a wrong BCH polynomial, a skipped descramble — and a round-trip test cannot see a mistake it makes twice.
category: solution-postmortem
keywords: round-trip test, self-consistent test, synthetic fixtures, ground truth iq, sync word constant, bch generator polynomial, ldu interleave, tetra training sequence, dibit rotation, mp3 frame sync, colour code descramble, test independence
tags: [from-the-issue-tracker, testing, p25, tetra, dmr, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 20
---

*Part 20 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 19]({{ '/blog/solution-postmortem/from-the-issue-tracker-19-one-render-loop/' | relative_url }})
closed out the single-bug stories. The last three parts are different: they read
across the whole tracker and pull out the patterns that produced bug after bug.
This one is about the most expensive pattern of all — the test suite that stays
green because the encoder and the decoder agree with each other about something
that is wrong.*

> **TL;DR:** A round-trip test — encode with your encoder, decode with your
> decoder, assert the payload survives — proves only that the two halves are
> *consistent*, not that either is *correct*. Any constant, table, or convention
> they share is invisible to it: a wrong sync word, a wrong generator polynomial,
> a wrong bit-offset table, a skipped descramble step. GopherTrunk shipped at
> least seven decoder bugs this way, each behind a fully green suite. The escape
> is independence: decode real captured air, cross-check against an independent
> implementation, and build fixtures from authoritative constants rather than
> from your own encoder.

## The shape of the trap

Every protocol decoder in GopherTrunk has a natural first test: synthesize a
frame, push it through the decode chain, assert the bits come back. It is a good
test — it catches slicing errors, state-machine bugs, off-by-ones in the decode
path itself. What it structurally cannot catch is anything the synthesis and the
decode *share*. If both sides read the sync word from the same constant, the
constant can be garbage and the test passes. If both sides use the same
interleave table, the table can be fiction and the test passes. The test is a
mirror: it shows the decoder agreeing with its own reflection.

The tracker holds at least seven distinct instances. None of them were exotic.
Every one was a bread-and-butter decode bug that a single frame of real
captured signal would have exposed on day one.

| Issue | What was wrong | Why the round-trip passed |
| --- | --- | --- |
| [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) | RRC pulse model, BCH generator, CRC variant — three at once | Encoder used the same wrong model and constants |
| [#489](https://github.com/MattCheramie/GopherTrunk/issues/489) | LDU voice-frame bit offsets omitted the interleaved link control | Encoder and decoder shared one wrong offset table |
| [#553](https://github.com/MattCheramie/GopherTrunk/issues/553) | TETRA training sequences were placeholders; wrong dibit→bit mapping | Every fixture was built from the same placeholders |
| [#297](https://github.com/MattCheramie/GopherTrunk/issues/297) | Dibit de-rotation inverted the rotation instead of applying it | Every fixture used rotation 0, where the bug is a no-op |
| [#813](https://github.com/MattCheramie/GopherTrunk/issues/813) | Phase 2 sync constant was 48 bits crammed into a 40-bit field | `EncodeSuperframe` injected sync from the same wrong constant |
| [#874](https://github.com/MattCheramie/GopherTrunk/issues/874) | MP3 frames desynced after the first frame | The unit test checked only the first frame's sync word |
| [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) | TETRA DMO skipped descrambling at colour code 0 | Synthetic tests scrambled *and* descrambled consistently at colour 0 |

## Three bugs in one decoder: #275

The flagship P25 Phase 1 lock failure ([#275](https://github.com/MattCheramie/GopherTrunk/issues/275))
is the trap at its purest, because it happened three times inside one decoder
and every instance was masked the same way.

First, the pulse shaping. The receive chain modelled P25 C4FM as an RRC matched
pair — RRC on transmit, RRC on receive. Per TIA-102.BAAA (and OP25's
`c4fm_const.py`), the real transmitter is a raised-cosine α=0.2 filter times an
inverse-sinc, and the matched receive filter is a plain sinc. Model both ends as
RRC and the synthetic harness is perfectly self-consistent — while a real signal
arrives with ~5.75% residual ISI, enough to hold the NID's error count at a
ceiling the BCH could never correct.

Second, the BCH generator itself. The BCH(63,16,11) generator constant in the
code was simply wrong; the correct value, `0xCD930BDD3B2B`, had to be re-derived
by multiplying the minimal polynomials of α, α³, … α²¹ over GF(2⁶). Any
round-trip that encoded with the wrong generator decoded happily with the same
wrong generator.

Third, the TSBK trailer CRC. The trailer uses the "augmented codeword" variant
of CRC-CCITT — init 0, MSB-first, final XOR `0xFFFF`, evaluated over all 12
bytes with an expected result of 0. The code used CRC-CCITT/FALSE: same
`0x1021` polynomial, different everything else. Symmetric on both sides,
invisible in every synthetic test, fatal on air.

The maintainer's own summary of the issue is the thesis of this post: every one
of these bugs would have stayed silently masked by the encoder and decoder
agreeing with each other inside synthetic round-trip tests. Fifteen seconds of
real ground-truth IQ was the linchpin that broke all three loose.

## One wrong table, shared: #489

P25 Phase 1 voice decoded 1,622 LDUs and found 1,622 of them uncorrectable —
exactly 100%, deterministically ([#489](https://github.com/MattCheramie/GopherTrunk/issues/489)).
On air, an LDU interleaves a link-control block between voice subframes
(`u0, LC, u1, LC, … u6, LSD, u7, u8`); the offset table in the code placed `u0`
and `u1` back to back, so only the first subframe was ever sliced correctly.
The encoder used the same table. Round-trip tests passed, because a symmetric
error round-trips cleanly — the fix ultimately had to be checked against an
independent reference (dsd-neo's `p25p1_const.h` schedule) rather than against
GopherTrunk's own encoder. Symmetrically wrong round-trip tests prove nothing.

## Fixtures built from fiction: #553 and #297

The TETRA demodulator never locked ([#553](https://github.com/MattCheramie/GopherTrunk/issues/553))
because its training-sequence constants were placeholders that had never been
replaced. `NormalSyncHex` was a `uint64` — 64 bits — declared to hold 38 dibits,
which is 76 bits; six dibits were silently zero-filled, and the code conflated
TETRA's 38-*bit* training sequence with 38 *dibits* besides. The real values
(ETSI EN 300 392-2 §9.4.4.3: NTS1/NTS2 are 22 bits, ETS 30, STS 38) matched
nothing in the code. A second shared convention rode along: TETRA's π/4-DQPSK
dibits map to bits through a Gray code, and the code used the linear mapping
that is correct for the C4FM family. Every test fixture in the package was
built from the same placeholders under the same wrong mapping, so the suite was
green while the demodulator could not correlate a single real burst.

[#297](https://github.com/MattCheramie/GopherTrunk/issues/297) is the subtler
cousin: the code was wrong for only *part* of the input space, and the fixtures
lived entirely in the part where it was accidentally right. The sync detector
defines rotation `k` by `(received + k) mod 4 == canonical`, so recovery must
add `k`; `rotateDibits` computed `(4-rot)&3` instead — off by exactly 2 for odd
rotations. The C4FM path can only produce even rotations, which the buggy code
handled correctly by coincidence; the CQPSK/simulcast path produces odd ones.
There *was* a rotation test — `TestSyncDetectorMatchesAllRotations` — but it
covered the detector, not the recovery, and every end-to-end fixture used
rotation 0, where the bug short-circuits to a no-op. Coverage of a component is
not coverage of the space.

## The encoder injects the bug: #813

The P25 Phase 2 sync constant `OutboundSyncHex = 0x575F7DFF77FF` was a 48-bit
value declared for a 40-bit, 20-dibit field
([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)). The
conversion helper silently dropped the top byte, leaving `0x5F7DFF77FF` — not
the standard sync word, not anything transmitted on air, not anything at all.
And every round-trip test passed, because `EncodeSuperframe` injected its sync
pattern *from the same constant through the same truncation*. The test fixtures
did not merely tolerate the bug; the encoder manufactured it into every fixture.
The authoritative value — `0x575D57F7FF`, from OP25's `frame_sync_magics.h`,
SDRTrunk, and TIA-102.BBAC — differed in ways no self-referential test could
ever notice. Real Phase 2 captures decoded zero superframes for months while CI
stayed green.

## Checking only the first unit: #874

The trap is not limited to RF. The MP3 encoder's unit test verified the sync
word of the *first* frame ([#874](https://github.com/MattCheramie/GopherTrunk/issues/874)) —
and the encoder's bit-reservoir stuffing desynced the stream only once a
frame's byte budget crossed ~420 bytes, frames deep into the file. A separate
bug made every 8 kHz frame malformed at source (128 kbps is not a legal
MPEG-2.5 Layer-III bitrate; the lookup returned −1 and OR'd stray bits over the
already-written header). Both survived a test that sampled one unit of a
repeating structure and assumed the rest followed. Validation had to move to an
independent decoder — `ffmpeg` reading the whole file, and reading it again
over a non-seekable pipe, since that is how the downstream consumer reads it.

## Wrong both ways: #1003

The most recent instance is the purest demonstration that the trap can hide a
bug even when someone goes looking for it. TETRA DMO voice sat at the CRC
chance floor and was initially read as encrypted traffic
([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)). The real
cause: the DMO speech path skipped descrambling when the colour code was 0 — a
shortcut inherited from the trunked-mode code, where an extended colour code is
never 0. But TETRA scrambling is *not* the identity at colour 0; the scrambler
seeds its LFSR to a fixed non-zero state, which is exactly why the signalling
decoders always descramble and always worked.

Here is the kicker: a sweep had been run specifically comparing "with and
without a colour-0 descramble" — and it found nothing, because the synthetic
round-trips scrambled *and* descrambled consistently at colour 0. Skip both
sides or apply both sides; either way the test passes. The asymmetry only
exists against real air, where the transmitter always scrambles. The fix's
regression test encodes that lesson directly: the encode side now scrambles
unconditionally — modelling the transmitter, not the code under test — so the
colour-0 case fails against the old decoder and passes against the new one.

## The remedies

Every escape from the trap is a form of the same move: break the symmetry.
Bring in something the code under test did not produce.

1. **Decode real captured ground truth.** Fifteen seconds of IQ cracked #275's
   three masked bugs; a reporter's capture cracked #813 and #1003. A
   `replay`-able capture in `testdata/` is worth more than any quantity of
   synthetic fixtures, because nothing in it came from your encoder.
2. **Cross-check against an independent implementation.** OP25's constants
   settled #813's sync word; dsd-neo's tables settled #489's offsets; the ETSI
   reference codec validated the TETRA vocoder end to end. If two independent
   codebases agree with the spec and disagree with you, you are wrong.
3. **Build fixtures from authoritative constants, not from your own encoder.**
   A fixture whose sync word is typed in from the standard (or from OP25's
   header) detects a truncated constant; a fixture emitted by
   `EncodeSuperframe` never will.
4. **Make the synthetic side model the transmitter, unconditionally.** #1003's
   fix-proof test scrambles on encode no matter what, because real
   transmitters do. Any "optimization" shared between test encoder and decoder
   is a place for a bug to hide.
5. **Cover the parameter space, not just the happy point.** #297's fixtures all
   sat at rotation 0; #874's test sat at frame 1. Sweep rotations, colour
   codes, frame counts — the cheap dimensions where symmetric bugs go quiet.

## What we keep

- A green round-trip suite is a claim of consistency, not correctness. Ask of
  every decoder test: *what does this share with the encoder?* That shared set
  is exactly the set of bugs it cannot see.
- Real captures are the only fixtures with no author in common with the code.
  Keep one per protocol, keep it in the repo, and gate "the decoder works" on
  it — see the [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  for the replay-first workflow that grew out of these issues.
- On-air constants deserve their own provenance. Every value in the
  [P25 on-air constants]({{ '/reference/p25-onair-constants/' | relative_url }})
  entry exists because a self-consistent wrong value shipped once; the
  [TETRA lock facts]({{ '/reference/tetra-lock-facts/' | relative_url }}) entry
  records the training sequences and mappings that #553 got wrong.
- A regression test must fail first. If you cannot make it fail against the
  old code, you have not escaped the trap — you may be inside a new one.

[Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
takes on the second grand pattern: log lines that only speak on success, and
the unconditional census that replaced them.
