---
title: "From Spec to Shipping, Part 12: Failing First — The Regression Rule"
description: Why every GopherTrunk bug fix is one narrow commit plus a regression test that fails without the fix — and three worked examples where writing the failing test WAS the diagnosis, from the DMO colour-0 descramble to the SmartNet rebuild to the noise-grant trio.
category: deep-dives
keywords: failing first regression test, regression test that fails without the fix, reproduce a bug before fixing it, dmo colour code descramble regression, smartnet decoder rebuild test, protocol decoder bug fixing, go testing discipline, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, testing, regressions, methodology, tetra, smartnet]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 12
---

*Part 12 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
made captures first-class test fixtures — the evidence that grades a decoder.
This part is the rule that governs what you do with that evidence: **a bug fix
is one narrow commit plus a regression test that fails without the fix and
passes with it** — and if you can't write the failing test, you haven't
reproduced the bug, so you keep digging instead of guessing. Three worked
examples show the failing test doing the diagnosis itself.*

> **TL;DR:** `CONTRIBUTING.md` states the rule in one line — *"regression test
> that fails without the fix and passes with it"* — and the tree is full of
> tests built to that shape. `TestDMTCHSpeechRoundTrip`
> (`internal/radio/tetra/dmo_decode_test.go`) scrambles its encode side
> **unconditionally**, like a real transmitter: against the old colour-0
> descramble skip it gets **0 speech frames**; with the
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) fix, **2
> CRC-valid frames**. `TestProcessDecodesRealAirFormat`
> (`internal/radio/motorola/process_test.go`) feeds the real SmartNet air
> format to the decoder: the pre-[#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)
> fabricated framing decodes **nothing** from it. And three tests in
> `internal/scanner/ccdecoder/pipelines_dmo_test.go` all fail against the old
> DMO grant path that opened recordings on noise. The pattern in every case:
> **the failing test is the reproduction, and the reproduction is where the
> root cause surfaces.**

**Key takeaways**

- **A failing test is a reproduction you can re-run.** Until something fails
  on demand, you have a theory about the bug, not the bug — and a fix built on
  a theory is a guess wearing a commit message.
- **The hard part of failing-first is making the test *able* to fail.** All
  three examples needed the test's world made more faithful first: an
  unconditional scrambler, a real-air bit stream, a true 255-dibit slot grid.
  That work is Part 7's lesson paying rent.
- **Failing-first inverts the workflow, and the inversion is the point.**
  Reproduce → fail → fix → pass front-loads the diagnosis; guess → fix →
  close defers it to the reporter, who becomes your regression suite.
- **When no failing test is possible, the fix does not land.** The P25 C4FM
  weak-signal levers are diagnosed, designed — and parked until a capture can
  make a test fail. That discipline is
  [Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})'s
  whole subject.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The rule | bug fix = one narrow commit + a test that fails without it | `CONTRIBUTING.md` ("How changes are scoped") |
| Colour-0 regression | encode side scrambles unconditionally, like real air | `internal/radio/tetra/dmo_decode_test.go` (`TestDMTCHSpeechRoundTrip`) |
| SmartNet regression | real-air OSW stream → old decoder = zero decodes | `internal/radio/motorola/process_test.go` (`TestProcessDecodesRealAirFormat`) |
| Noise-grant trio | idle channel, lock ordering, re-arm — three failure modes | `internal/scanner/ccdecoder/pipelines_dmo_test.go` |
| Faithful synthetic transmitter | every burst on a true 255-dibit timeslot | `pipelines_dmo_test.go` (`buildDMODibitStream`) |
| When you can't fail first | the issue stays open, gated on evidence | `CLAUDE.md` issue-closing policy → Part 14 |

## In this post

- **The rule, and what it is actually for** — a failing test is a
  reproduction, not paperwork.
- **Zero frames, then two** — the DMO colour-0 descramble regression.
- **A decoder that decodes nothing** — the SmartNet real-air-format test.
- **Three tests against one pipeline** — the noise-grant trio.
- **Failing first as a diagnostic method** — why root causes surface while
  building the reproduction.
- **When you cannot fail first** — the honest ending, and the Part 14 tie-in.

## The rule, and what it is actually for

`CONTRIBUTING.md` scopes a bug fix in one sentence: *one commit, narrow diff,
regression test that fails without the fix and passes with it.* The project's
standing guidance sharpens the contrapositive: **if you can't write a test
that fails first, you have not yet reproduced the bug — keep digging or ask
for a reproduction, don't guess at a fix.**

Read as process, that sounds like ceremony. Read as epistemology, it's the
cheapest verification gate in the series. A fix without a failing test makes
two unproven claims at once: that you found the real cause, and that your
change removes it. The failing test converts both into observations — it
fails against the old code, so the reproduction is real; it passes against
the new, so the change addresses *that* reproduction. This project learned
the price of skipping them in public: issue
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) was closed
twice on fixes nobody had watched fail, while the symptom stayed live
([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)) — the story
[the on-air gate]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
told and Part 14 turns into policy.

One caveat the series has already earned: a failing test proves you
reproduced *a* bug; only outside evidence proves it's *the* bug. The villain
of [Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
— the test that shares its assumption with the code — fabricates a
convincing failure as easily as a convincing pass, and all three examples
below had to defeat it before the rule could work.

## Zero frames, then two: the colour-0 descramble

[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
told the DMO story from the verdict side: clear TETRA direct-mode voice sat
at a ~1/256 chance-floor CRC rate and got misread as encryption, when the
real cause was that `DMBurstTCHSpeech` skipped descrambling at colour code 0
— inherited from a TMO shortcut that was safe there only because a TMO
extended colour code is never 0. TETRA scrambling is **not** an identity at
colour 0: the LFSR seeds to `0xC0000000` (EN 300 392-2 §8.2.5.2), which is
exactly why the signalling paths always descramble and always decoded.

The regression test is the interesting artifact, because the *old* test suite
had checked this and passed. Its round-trips scrambled and descrambled under
the same `colour != 0` condition — self-consistent on both sides, green
either way, blind to the asymmetry. The fix's test changes one thing: the
encode side now behaves like the air, unconditionally.

```go
// internal/radio/tetra/dmo_decode_test.go (shape) — TestDMTCHSpeechRoundTrip
// The colour-0 iteration is the failing-first regression for issue #1003:
// a real DMO transmitter scrambles TCH/S with the DM colour code even at
// colour 0 (seed 0xC0000000, §8.2.5.2), so a decoder that skips descramble
// at colour 0 leaves the block scrambled and the class-2 CRC fails.
for _, colour := range []uint32{0, 0x0AB1F} {
    /* … encode two speech frames through EN 300 395-2 TCH/S coding … */
    onair := framing.ScrambleTetra(type4, colour) // scrambled like real air, incl. colour 0
    /* … frame as a DNB, run the real extractor … */
    frames := DMBurstTCHSpeech(*dnb, colour)
    if len(frames) != 2 {
        t.Fatalf("colour %#x: got %d frames, want 2 (CRC failed?)", colour, len(frames))
    }
}
```

Verified failing-first, with the numbers on record: the old code returns **0
frames** at colour 0; the fixed code returns the **2 CRC-valid frames** that
went in. That asymmetric shape — pin one side to reality, let the other side
prove itself — is the whole difference between this test and its green
predecessor, and it's the constructive mirror of the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).
The full saga, colour sweep and all, is in
[TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}).

## A decoder that decodes nothing: the SmartNet regression

[Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})'s
case study ended with a rebuilt Motorola SmartNet decoder — the original's
24-bit sync word and BCH(64,16,11) framing matched no real reference, so no
real system could ever lock, while every synthetic test glowed green. The
rebuild's headline regression states the indictment as a runnable fact:

```go
// internal/radio/motorola/process_test.go (shape) — TestProcessDecodesRealAirFormat
// The issue #1143 regression: a control-channel stream in the REAL SmartNet
// air format (OP25 rx_smartnet framing) must lock and publish the grant.
// The pre-#1143 decoder (24-bit sync 0xA4D7AA + BCH(64,16,11), a framing no
// real system transmits) decodes NOTHING from this stream — verified
// failing-first against the old code.
cc := New(Options{Bus: bus, SystemName: "Real", FrequencyHz: 854_562_500})
cc.Process(realAirStream(200, seq...), 0)

locked, grants := drainEvents(sub)
if len(locked) == 0 {
    t.Fatal("no cc.locked from a real-air-format stream")
}
/* … grant must carry tg 0xB010, src 0x2E9A, 861.5375 MHz (channel 0x1A5) … */
```

Notice what the test's input is: not the old encoder's output, but a stream
built to the framing **two independent proven-on-air decoders** transmit and
consume — OP25's `rx_smartnet` and trunk-recorder's parser, cross-checked as
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
prescribes. That provenance is what makes the failure meaningful. Feed this
stream to the old decoder and it produces zero locks and zero grants — the
reporter's `cchunt: hunt failed` symptom, reproduced on the desk, no antenna
required. Fourteen months of green tests never said that; one failing-first
test did. (And per the on-air gate, even this rebuild stays
capture-verification-pending until the reporter's 854.5625 MHz capture
replays through it — synthetic-green has fooled this exact decoder before.)

## Three tests against one pipeline: the noise-grant trio

The richest example is the DMO grant path, because one field report yielded
**three distinct failing tests** — each pinning a different way the old code
was wrong. The report: the daemon granted and opened a recording ~230 ms
after startup on a *silent* channel, then never granted again — the
operator's real 10-second PTT, which genuinely locked (`dsb_schs_crc=46/54`),
produced nothing. The mechanism (told in full in
[TETRA End to End Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})):
the DNB burst correlator is loose enough to fire **~18 times per second on
pure noise**, and the grant logic took raw detections as traffic.

| Test (`pipelines_dmo_test.go`) | The failure it pins | Old code's behaviour |
|---|---|---|
| `TestTETRADMOPipelineIgnoresIdleChannel` | 10 s of noise must yield no lock, no grant, zero *qualified* DNBs — while raw detections keep raining | grants in ~230 ms on nothing |
| `TestTETRADMOPipelineGrantsOnlyAfterLock` | no grant may precede `cc.locked` | `dnbSinceLock` was named for a lock the guard never consulted |
| `TestTETRADMOPipelineRearmsBetweenTransmissions` | after a silent gap, a second PTT must grant again | re-arm needed a DNB drought the 18/s noise made impossible — and was only evaluated when a burst arrived |

All three fail against the old pipeline. But the load-bearing work happened
*before* the assertions could mean anything: the synthetic transmitter had to
become faithful. The old test fixture laid bursts into the stream with
arbitrary filler between them — and against that layout, the eventual fix
(`tetra.DMSlotGrid`, which qualifies bursts by voting their positions onto
one learned residue of the 255-dibit timeslot grid) is untestable, because
the "transmitter" itself didn't keep slot time. `buildDMODibitStream` now
emits every burst on its own 255-dibit timeslot, *as a real radio transmits
them* — at which point the idle-channel test could demand `dnbQualified == 0`
against ~18 raw detections a second and watch the old code fail. A test can
only disagree with you if its world is real enough to disagree in.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Two bug-fix workflows compared. Top row, guess-fix-close: symptom, guessed cause, fix, existing tests green, issue closed — with a dashed arrow looping from the closed issue back to the symptom, labelled reporter reopens. Bottom row, reproduce-fail-fix-pass: symptom, faithful reproduction, a test that fails against the old code (marked as where the root cause surfaces), the fix, the same test passing, verified close.">
  <text x="20" y="24" fill="currentColor" font-size="11" font-weight="bold">guess → fix → close</text>
  <rect x="20" y="36" width="92" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="66" y="54" text-anchor="middle" fill="currentColor" font-size="10">symptom</text>
  <line x1="112" y1="50" x2="142" y2="50" stroke="currentColor"/><polygon points="140,46 148,50 140,54" fill="currentColor"/>
  <rect x="150" y="36" width="104" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="202" y="54" text-anchor="middle" fill="currentColor" font-size="10">guessed cause</text>
  <line x1="254" y1="50" x2="284" y2="50" stroke="currentColor"/><polygon points="282,46 290,50 282,54" fill="currentColor"/>
  <rect x="292" y="36" width="56" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="320" y="54" text-anchor="middle" fill="currentColor" font-size="10">fix</text>
  <line x1="348" y1="50" x2="378" y2="50" stroke="currentColor"/><polygon points="376,46 384,50 376,54" fill="currentColor"/>
  <rect x="386" y="36" width="130" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="451" y="54" text-anchor="middle" fill="currentColor" font-size="9">existing tests green</text>
  <line x1="516" y1="50" x2="546" y2="50" stroke="currentColor"/><polygon points="544,46 552,50 544,54" fill="currentColor"/>
  <rect x="554" y="36" width="104" height="28" rx="5" fill="none" stroke="currentColor"/>
  <text x="606" y="54" text-anchor="middle" fill="currentColor" font-size="10">issue closed</text>
  <path d="M 606 64 C 606 96 66 96 66 68" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <polygon points="62,70 66,62 70,70" fill="var(--fg-muted)"/>
  <text x="340" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">reporter reopens: the symptom was still live (#764 → #771)</text>
  <text x="20" y="132" fill="var(--accent)" font-size="11" font-weight="bold">reproduce → fail → fix → pass</text>
  <rect x="20" y="144" width="92" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="66" y="165" text-anchor="middle" fill="var(--accent)" font-size="10">symptom</text>
  <line x1="112" y1="161" x2="140" y2="161" stroke="var(--accent)"/><polygon points="138,157 146,161 138,165" fill="var(--accent)"/>
  <rect x="148" y="144" width="122" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="209" y="159" text-anchor="middle" fill="var(--accent)" font-size="9">faithful reproduction</text>
  <text x="209" y="171" text-anchor="middle" fill="var(--fg-muted)" font-size="8">real-air stream / slot grid</text>
  <line x1="270" y1="161" x2="298" y2="161" stroke="var(--accent)"/><polygon points="296,157 304,161 296,165" fill="var(--accent)"/>
  <rect x="306" y="144" width="126" height="34" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="369" y="159" text-anchor="middle" fill="var(--accent)" font-size="9">test FAILS vs old code</text>
  <text x="369" y="171" text-anchor="middle" fill="var(--fg-muted)" font-size="8">root cause surfaces here</text>
  <line x1="432" y1="161" x2="460" y2="161" stroke="var(--accent)"/><polygon points="458,157 466,161 458,165" fill="var(--accent)"/>
  <rect x="468" y="144" width="52" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="494" y="165" text-anchor="middle" fill="var(--accent)" font-size="10">fix</text>
  <line x1="520" y1="161" x2="548" y2="161" stroke="var(--accent)"/><polygon points="546,157 554,161 546,165" fill="var(--accent)"/>
  <rect x="556" y="144" width="104" height="34" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="608" y="159" text-anchor="middle" fill="var(--accent)" font-size="9">same test passes</text>
  <text x="608" y="171" text-anchor="middle" fill="var(--fg-muted)" font-size="8">→ verified close (Part 14)</text>
</svg>
<figcaption>Both workflows write the same fix — but only the bottom one ever observes the bug happening, and the observation is where the real cause surfaces.</figcaption>
</figure>

## Failing first is a diagnostic method, not a formality

The rule's reputation is bureaucratic: proof-of-work attached to a fix. In
practice it's where most of the diagnosis happens, because **building a
reproduction forces you to state, mechanically, what you believe the bug is**
— and mechanical statements get checked in ways prose theories don't.

The DMO trio shows it best. "The grant fires on noise" was the theory; making
a *test* say it required numbers — how often does the correlator fire on
noise? Working that out (529 of the 4^11 possible 11-dibit sequences match
within tolerance 2, across eight matched filters, at 18,000 dibits/s — ~18
false detections a second, and the operator's own log measured 18.7/s) didn't
just justify the test; it dictated the fix's shape. A threshold can't beat an
18/s rain, but a slot-grid residue vote can: one radio on one clock puts
every burst on one residue mod 255, noise is uniform over all 255. The
reproduction and the design came out of the same arithmetic. Likewise the
colour-0 test — writing "scramble like real air" forced the question *what
does real air do at colour 0?*, whose answer (`0xC0000000`, not identity) was
the bug.

And when you *can't* make the test fail, that's data too: whatever you
reproduced isn't what was reported, and you found out at your desk instead of
in the reporter's follow-up comment.

## When you cannot fail first

The rule's honest edge case: some failures can't be reproduced from what you
have. The P25 Phase 1 C4FM weak-signal gap is the standing example — the
diagnosis is structural and confident (no equalizer, no soft-decision FEC on
that path; the same two levers that roughly doubled TETRA yield), the fix is
designed, and *nothing has landed*, because without a weak C4FM voice capture
in `samples/p25/` there is no test that fails first. Guessing a DSP fix and
shipping it green-on-synthetics is precisely the failure mode
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
exists to prevent.

So the workflow ends the only honest way it can: the issue **stays open**,
with a status comment saying what's known and what's blocking — usually a
capture request shaped by
[Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})'s
discipline. What it must *not* end with is a close. Closing is a claim that
the problem is gone, and that claim has its own definition, its own policy,
and — after this project shipped the counterexample — its own guard hook.
That is the finale's subject.

## Where this goes next

A failing test proves the fix worked once, on your machine, on your fixture.
Whether it keeps working in the field is a question only the running system
can answer — which means the running system has to be able to *say* things.
[Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
is about building diagnostics as instruments: counters on every branch,
verdict lines written for operators, and WARNs that can tell a condition from
a transient.

## FAQ

**What makes a regression test "failing-first" rather than just a test?**
Provenance: it was run against the pre-fix code and observed to fail, and
that observation is recorded (GopherTrunk writes it into the test's doc
comment — "verified failing-first against the old code"). A test written
after the fix and only ever seen green proves the code satisfies the test; it
never proves the test detects the bug.

**Isn't this just test-driven development?**
It's TDD's contrapositive, applied to defects: TDD writes a failing test to
specify behaviour that doesn't exist yet; failing-first writes one to
reproduce behaviour that shouldn't. The extra burden is fidelity — a TDD test
can define its own world, but a regression test's world must match the air,
the wire, or the protocol, or it reproduces nothing (the
[Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
problem).

**What if writing the failing test takes longer than the fix?**
It usually does, and that time is the diagnosis, not overhead. The colour-0
fix was a two-line guard removal; finding *which* two lines took a colour
sweep, a scrambler-seed check, and an encode-side rebuild of the test — and
the old "fix it and see" approach had already produced a wrong answer
("encrypted") from the same evidence.

**How do I fail first when the bug only appears on air?**
Get the air onto disk: an IQ capture replayed through an env-gated harness is
a failing test an operator can run (`GT_TETRA_DMO_IQ=… go test -run
TestTETRADMOReplay`). Part 11 covers the conventions. When even that isn't
available, the answer is the last section's: the fix waits, the issue stays
open, and the capture request *is* the work product.

**My new test fails against the old code — am I done diagnosing?**
Almost. Check what it fails *on*: a test can fail against old code for a
reason unrelated to the reported symptom, which quietly substitutes your bug
for the reporter's. Tie the assertion to the report's observable (zero
frames, a grant on silence, `hunt failed`) and, where you can, to the
reporter's own numbers — the trio's tests quote the operator's log in their
comments for exactly this reason.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Capture-Driven Development]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
· Next →
[Part 13: Instruments, Not Logs]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
