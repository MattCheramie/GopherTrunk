---
title: "From Spec to Shipping, Part 7: Tests That Can Disagree With You"
description: "Four ways GopherTrunk engineers the self-consistent trap out of its test suites: fixture transmitters that behave like real radios, independent-path controls, fake servers that enforce the real protocol's strictness, and synthetic streams laid out on the real slot grid."
category: deep-dives
keywords: self consistent test trap, round trip test weakness, faithful test fixtures, fake server protocol strictness, independent resampler control, synthetic transmitter slot grid, failing first regression test, sdr decoder testing, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, testing, methodology, tetra, soapyremote, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 7
---

*Part 7 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 6]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
put a real capture in the referee's chair when references disagreed.
This part turns to the tests themselves — because the series villain,
**the test that passes because both sides share the same assumption**,
is not bad luck. It is a design defect with known fixes. The
[postmortem version]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
catalogued the wreckage; this is the constructive mirror: four worked
examples of building a test that retains the power to call your code
wrong.*

> **TL;DR:** A test can only catch a bug it doesn't share. Four
> structural fixes from the GopherTrunk tree: **make the encode side
> unconditional where the air is unconditional** — `dmo_decode_test.go`
> now scrambles at colour 0 like a real transmitter, turning a
> passing-either-way round-trip into a failing-first regression for the
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)
> descramble skip; **build controls from independent parts** — the
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) verdict
> stood on decimating a 10 MS/s capture with a resampler the DDC under
> test doesn't use; **give fakes the real protocol's strictness** —
> `newFakeSoapyServer` asserts every request body is fully consumed,
> mirroring the real `~SoapyRPCUnpacker`, and promptly caught two more
> drifted calls; **make synthetic transmitters structurally faithful** —
> `buildDMODibitStream` lays bursts on a true 255-dibit slot grid,
> because the noise-grant bug lived in exactly the structure the old
> arbitrary-filler fixture didn't model.

**Key takeaways**

- **A round-trip test proves consistency, not correctness.** Encoder and
  decoder sharing one wrong assumption pass forever; the fix is to pin
  one side to something external — real-air behaviour, an upstream
  literal, an independent implementation.
- **Fixtures must copy the transmitter, not the decoder.** Every "shortcut"
  in a synthetic stream — skipping a descramble the air performs, filling
  gaps arbitrarily instead of on the slot grid — is a place your test
  quietly adopts your code's worldview.
- **A fake service inherits none of the real service's strictness unless
  you give it some.** The real SoapyRemote server complains about
  unconsumed payload bytes; a fake that doesn't is blind to
  argument-shape drift.
- **Know which drift class each net catches.** Full-consumption asserts
  catch argument-shape drift but structurally *cannot* catch opcode
  drift — both sides move together — which is why upstream-literal pins
  exist as a separate net.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Unconditional encode side | fixture scrambles at colour 0, as real air does | `internal/radio/tetra/dmo_decode_test.go` (`TestDMTCHSpeechRoundTrip`) |
| Independent-path control | 4:1 decimation via a separate resampler before replay | `internal/scanner/ccdecoder/ddc_highrate_test.go` (`TestDownconverterSNRInvariantAcrossRate`) |
| Strict fake server | fails any test leaving unconsumed request bytes | `internal/sdr/soapyremote/driver_test.go` (`newFakeSoapyServer`, `assertCleanProtocol`) |
| Opcode pins | numeric RPC ids checked against upstream literals | `driver_test.go` (`TestOpenSetAntennaUsesUpstreamOpcode`) |
| Faithful synthetic transmitter | bursts on a true 255-dibit slot grid | `internal/scanner/ccdecoder/pipelines_dmo_test.go` (`buildDMODibitStream`, `dmoSlot`) |
| The bug the grid caught | noise-driven grants on an idle channel | `pipelines_dmo_test.go` (`TestTETRADMOPipelineIgnoresIdleChannel`) |

## In this post

- **The trap, stated precisely** — what makes a test unable to disagree.
- **Rule 1: encode like the air, not like the decoder** — the colour-0
  scramble.
- **Rule 2: build the control from parts you're not testing** — the
  #764 resampler.
- **Rule 3: fakes must enforce, not just respond** — the strict Soapy
  server, and its honest limit.
- **Rule 4: synthetic streams need the real structure** — the 255-dibit
  slot grid.

## The trap, stated precisely

A test disagrees with your code only where the two embody *different
beliefs*. A round-trip test — encode, decode, compare — shares every
belief by construction: the layout, the scrambling policy, the framing,
all authored by the same hands, usually in the same afternoon. It
verifies the two directions are inverses, which is worth something, and
verifies nothing about whether either matches the world. GopherTrunk
shipped a SmartNet decoder for months on green round-trips over a
framing no real system transmits
([Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})).

[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
literal vectors are the first antidote: pin one side to bytes from
outside your head. But literal vectors cover parsers; whole subsystems —
pipelines, drivers, grant logic — need *dynamic* fixtures, and every
dynamic fixture is a little transmitter you wrote yourself. The
discipline below is about keeping those little transmitters honest.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A ladder of test doubles ranked by how much external reality they enforce. Bottom rung: round-trip with shared assumptions, catches nothing about the world. Second rung: fixture pinned to real-air behaviour, such as unconditional scrambling. Third rung: fake service enforcing the real protocol's strictness, full consumption of request bytes. Fourth rung: structurally faithful synthetic transmitter on the real slot grid. Top rung, highlighted: independent implementation or real capture. An arrow up the side reads: more reality enforced, more bugs catchable.">
  <line x1="70" y1="210" x2="70" y2="20" stroke="var(--fg-muted)"/>
  <polygon points="66,24 70,14 74,24" fill="var(--fg-muted)"/>
  <text x="52" y="120" fill="var(--fg-muted)" font-size="9" transform="rotate(-90 52 120)">more reality enforced</text>
  <rect x="100" y="182" width="440" height="28" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="110" y="200" fill="var(--fg-muted)" font-size="10">round-trip, shared assumptions — proves inverses, not correctness</text>
  <rect x="100" y="142" width="460" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="160" fill="currentColor" font-size="10">fixture pinned to real-air behaviour (scramble unconditionally)</text>
  <rect x="100" y="102" width="480" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="120" fill="currentColor" font-size="10">fake enforcing the real service's strictness (full consumption)</text>
  <rect x="100" y="62" width="500" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="110" y="80" fill="currentColor" font-size="10">structurally faithful synthetic transmitter (255-dibit slot grid)</text>
  <rect x="100" y="22" width="520" height="28" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="110" y="40" fill="var(--accent)" font-size="10">independent implementation / real capture (Parts 3, 6, 10)</text>
</svg>
<figcaption>Test doubles ranked by the reality they enforce: each rung up removes one class of assumption your code and its test could otherwise share.</figcaption>
</figure>

## Rule 1: encode like the air, not like the decoder

The DMO colour-0 bug is the cleanest specimen in the tree. TETRA
scrambling is non-identity at colour 0 — the LFSR seeds to `0xC0000000`
even with every colour bit zero — but the DMO voice path inherited a
`if colour != 0` descramble skip from TMO code where the guard happened
to be safe. Real colour-0 traffic therefore reached the Viterbi still
scrambled, and clear voice was misread as encrypted for weeks.

The round-trip test passed the whole time, because the fixture's encode
side skipped scrambling under *the same condition*. Encoder and decoder
agreed; the air disagreed with both. The fix to the test is one line of
philosophy: the fixture must do what a **transmitter** does, not what
the decoder expects.

```go
// internal/radio/tetra/dmo_decode_test.go (shape) — TestDMTCHSpeechRoundTrip
for _, colour := range []uint32{0, 0x0AB1F} {
    type4 := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), nBits)
    // Scrambled like real air, INCLUDING colour 0 (seed 0xC0000000,
    // §8.2.5.2) — a real DMO transmitter never skips this.
    onair := framing.ScrambleTetra(type4, colour)
    /* … frame as a DNB, extract, decode … */
    frames := DMBurstTCHSpeech(*dnb, colour)
    // colour 0: fails against the old skip (0 frames), passes fixed (2).
}
```

With the encode side unconditional, the colour-0 iteration became the
failing-first regression for the fix — verified against the old code:
zero frames before, two CRC-valid after. **Rule: every conditional in a
fixture is a shared assumption; hunt them and align each with the
transmitter's behaviour, not the decoder's.** The air has no `if`.

## Rule 2: build the control from parts you're not testing

Issue [#764]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
needed an experiment: a system decoded from a 2.5 MS/s capture but not
from a 10 MS/s capture of the same site — is the wideband DDC mangling
high-rate input, or is the deficit in the samples? The obvious test —
run GopherTrunk's own decimation and compare — is circular: the
component under suspicion sits inside the control.

The verdict experiment decimated the 10 MS/s file 4:1 with an
**independent resampler**, then replayed the result through the proven
2.5 MS/s path. Same ≈9.5 dB demod SNR as the native 10 MS/s decode — so
the deficit was baked into the captured samples (front-end phase noise
at the Airspy's native clock), not GopherTrunk's DSP.
`TestDownconverterSNRInvariantAcrossRate` in
`internal/scanner/ccdecoder/ddc_highrate_test.go` pins the invariant
permanently: a noisy channel reaches the receiver at the same in-channel
SNR whether decoded natively at 10 MS/s or decimated to 2.5 MS/s.

**Rule: when a test compares two paths, the bridge between them must not
be built from either path.** A control that shares components with the
hypothesis can only ever confirm it — the experimental twin of the
round-trip trap.

## Rule 3: fakes must enforce, not just respond

Unit tests for the SoapyRemote driver run against `newFakeSoapyServer`,
a from-scratch TCP fake. For a long time it was a *permissive* fake: it
parsed the arguments it knew about and ignored trailing bytes. The real
`SoapySDRServer` is stricter — its `~SoapyRPCUnpacker` logs
`Unconsumed payload bytes N` whenever a request carries more than the
handler consumed. That gap in strictness is exactly where the
`callSetAntenna = 600` opcode bug lived undetected: the wrong opcode
invoked a different real handler, left 9 bytes unconsumed, and only the
*real* server ever said so — in a log line on an operator's machine.

The fake now enforces what the real server enforces, and every test
gets it without asking:

```go
// internal/sdr/soapyremote/driver_test.go (shape) — newFakeSoapyServer
s := &fakeSoapyServer{t: t, ln: ln, quietDone: make(chan struct{})}
go s.acceptLoop()
/* … */
// Every test that drives the fake server gets the wire-shape check
// for free, so a new or edited call has to account for its own bytes.
t.Cleanup(func() { s.assertCleanProtocol(t) })
```

`assertCleanProtocol` fails any test whose session left an unknown call
id or unconsumed payload bytes. Retrofitting it immediately **caught two
more calls the fake had not been parsing** — drift that had been sitting
green for who knows how long.

Just as important is the fake's documented *limit*. The comment on its
error ledger says it plainly: this net catches **argument-shape** drift,
but it "cannot catch OPCODE drift on its own: this fake switches on the
same constants the client packs, so if a constant is wrong both sides
move together and agree. That is exactly how SET_ANTENNA=600 survived a
release." Opcodes are pinned by a *different* net —
`TestOpenSetAntennaUsesUpstreamOpcode`, against literals from upstream's
`SoapyRemoteDefs.hpp` — a story
[Part 9]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
tells in full. **Rule: give the fake every strictness the real service
has, and write down which drift class it still cannot see — then build
the second net for that class.**

## Rule 4: synthetic streams need the real structure

The subtlest example: the DMO pipeline once granted a voice recording
~230 ms after startup on a **silent channel**. The DNB training-sequence
correlator false-fires at ~18/s on noise — that's inherent to an
11-dibit match at tolerance 2 — and the grant logic counted raw
detections. The fix (`tetra.DMSlotGrid`) exploits a physical fact: one
radio on one clock puts every burst on one residue mod 255 dibits, while
noise detections spread uniformly over all 255 residues.

Here's the test-design point: the *old* synthetic fixture could not have
caught either the bug or verified the fix, because it laid bursts into
the stream with **arbitrary filler between them** — no slot grid at all.
A stream without the real structure can't exercise a detector built on
that structure. The rebuilt fixture is a faithful transmitter down to
the geometry:

```go
// internal/scanner/ccdecoder/pipelines_dmo_test.go (shape)
// A synthetic stream MUST be laid out this way: a real transmitter
// emits one burst per timeslot from one clock, so all its DNB leads
// share one residue mod 255 — the property tetra.DMSlotGrid uses to
// tell traffic from correlator noise.
const dmoTestSlotDibits = 255

func dmoSlot(seed int, fields ...[]uint8) []uint8 {
    slot := dmoFiller(seed, dmoTestSlotDibits)
    /* … place freq-corr / BKN1 / training / BKN2 at dibits 7..234 … */
    return slot
}
```

On that grid, three failing-first regressions became writable —
`TestTETRADMOPipelineIgnoresIdleChannel`,
`…GrantsOnlyAfterLock`, `…RearmsBetweenTransmissions` — all three
failing against the old pipeline
([TETRA End to End Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
has the operator-side story). **Rule: a synthetic transmitter must be
faithful in structure, not just content.** Detectors exploit physical
constraints — one clock, one residue, back-to-back frames — and a
fixture that doesn't model the constraint silently exempts every
component that depends on it.

### How that principle shaped the Go code

- **Fixture encoders live next to the decoder and mirror the air.**
  `EncodeTCHS`, `EncodeOSWFrame`, `buildDMODibitStream` — each is the
  transmitter's behaviour transcribed, unconditional scrambling and slot
  grids included, so a fixture "shortcut" is a visible diff.
- **Strictness is installed in the constructor.** The Soapy fake wires
  `assertCleanProtocol` into `t.Cleanup` inside `newFakeSoapyServer`, so
  no test can opt out by forgetting.
- **Every net documents its blind spot.** The fake's comment names the
  drift class it cannot catch and points at the test that can — the
  suite is a system of nets, not a pile.
- **Controls are labelled independent.** The #764 test's comments say
  *why* the resampler is separate; the independence is the load-bearing
  property, protected from a well-meaning refactor to "reuse" the DDC.

## Where this goes next

These four rules are defense; the next part is the full offensive
campaign. [Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
replays the SmartNet rebuild end to end — a decoder whose every
synthetic was green over framing no real system transmits, torn down and
rebuilt from proven decoders, with reference-literal pins and a
failing-first real-air regression standing where the round-trips used to.

## FAQ

**What is the self-consistent trap in testing?**
A test whose expected values derive from the same assumptions as the
code under test — most commonly an encode/decode round-trip where both
sides were written together. It verifies internal consistency and passes
regardless of whether either side matches the real wire format. Every
worked example in this post is a structural way to reintroduce an
outside fact.

**Are round-trip tests worthless, then?**
No — they verify inverse-ness, catch regressions in either direction,
and exercise buffer handling cheaply. They're insufficient alone: pair
them with literal vectors pinned to an independent source, and audit the
fixture's conditionals against transmitter behaviour. GopherTrunk keeps
its round-trips; it stopped letting them testify about the air.

**How faithful does a synthetic test signal need to be?**
Faithful in every property some component *exploits*. The DMO grant path
exploits slot-grid regularity, so the fixture needs a true 255-dibit
grid; the SmartNet framer exploits back-to-back frames, so its fixture
emits them back-to-back with the trailing sync. The practical method:
for each detector or gate in the path, ask "what physical constraint
does this lean on?" and check the fixture models it.

**How do I test a client against a fake server without fooling myself?**
Give the fake the real server's observable strictness — parse every
argument and fail on leftover bytes, unknown ids, out-of-order calls —
and install those asserts in the constructor so they're universal. Then
write down what the fake still can't catch (constants both sides share)
and pin those separately against upstream literals.

**What does "failing-first" add on top of all this?**
Proof that the test can disagree. Running the new test against the *old*
code and watching it fail — zero frames at colour 0, zero decodes from
the real-air SmartNet stream — is the only direct evidence a test has
discriminating power.
[Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
makes that the standing rule for every bug fix.

## Series navigation

**Part 7 of 14** · ←
[Part 6: When References Disagree, the Capture Referees]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
· Next →
[Part 8: Case Study — Rebuilding SmartNet From Proven Decoders]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
