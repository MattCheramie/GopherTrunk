---
title: "From Spec to Shipping, Part 10: The On-Air Gate — Green Synthetics Prove Nothing"
description: Why every decoder claim in GopherTrunk carries a verification status — synthetic-green, capture-verified, or on-air-verified — and how the TETRA DMO "encrypted" misdiagnosis was overturned three times by three successive operator captures before the real causes surfaced.
category: deep-dives
keywords: on-air verification, synthetic test vs real capture, tetra dmo encrypted misdiagnosis, dm colour code recovery, iq replay test workflow, crc chance floor, decoder verification levels, issue closing discipline, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, tetra, dmo, verification, testing, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 10
---

*Part 10 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 9]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
built four nets around a schemaless RPC wire — and ended on the admission
that passing all four proves nothing about a real system. This part makes
that boundary a formal gate: three levels of evidence, three kinds of
claims each level licenses, and the cautionary tale of a DMO decoder that
was "encrypted," then "clear but colour 3," then "no dominant colour" —
each verdict overturned by the **next** operator capture.*

> **TL;DR:** Every decoder claim in GopherTrunk carries one of three
> statuses: **synthetic-green** (our tests pass), **capture-verified**
> (a real recording decodes), **on-air-verified** (the operator confirms
> on the live system). The gate exists because
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) was
> closed twice on a synthetic-green "fix" while the symptom stayed live
> ([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)). The
> TETRA DMO saga
> ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) shows
> why: a chance-floor TCH/S CRC was read as encryption, the radios were
> TEA0-clear, and the real causes — a colour-0 descramble skip, a wrong
> colour code, an MNI-0 search blind spot — surfaced one capture at a
> time, through env-gated replay tests (`TestTETRADMOReplay`,
> `GT_TETRA_DMO_*`) whose **verdict lines are written for the person
> holding the antenna**.

**Key takeaways**

- **"The tests pass" and "it works" are different claims with different
  evidence.** A synthetic fixture shares assumptions with the code it
  tests; a capture carries only the transmitter's assumptions; the air
  carries nobody's. Each rung falsifies things the rung below cannot.
- **A chance floor is a measurement, not a diagnosis.** TCH/S CRC at
  ~1/256 is consistent with encryption, a wrong descramble seed, wrong
  burst geometry, *and* a weak signal. The DMO investigation read it as
  encryption and was wrong — the discriminating fact (TEA0-clear radios)
  had to come from outside the code.
- **Build the operator loop as a product.** When the person with RF
  access isn't the person with the debugger, the test harness *is* the
  interface: env-var replay tests, verdict lines that state the
  conclusion and the next step, and a flag (`GT_TETRA_DMO_CLEAR`) that
  folds the operator's ground truth into the verdict.
- **A gate that refuses to answer is working.** `RecoverDMColourCode`
  demands its winner beat the runner-up 3× — and on the one capture where
  no colour dominated, that refusal was the correct output. The temptation
  to hardcode the 33%-yield "winner" was the self-consistent trap again.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The gate's origin | issue closed twice on an unverified fix, symptom live | [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) / [#771](https://github.com/MattCheramie/GopherTrunk/issues/771); `CLAUDE.md` issue-closing policy |
| DMO replay harness | operator-run capture replay with verdict lines | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`TestTETRADMOReplay`) |
| Ground-truth flag | operator asserts the call is clear → verdict flips | `GT_TETRA_DMO_CLEAR=1` |
| Colour diagnostics | 64-colour CRC-yield sweep, prints the full map | `TestTETRADMOColourScan` (`GT_TETRA_DMO_SCAN=1`) |
| Empirical colour recovery | pick the colour that maximizes CRC yield, with a dominance gate | `internal/radio/tetra/dmo_decode.go` (`RecoverDMColourCode`) |
| Signal-vs-DSP referee | independent resampler proves a deficit lives in the samples | `internal/scanner/ccdecoder/ddc_highrate_test.go`, [Weak-Signal Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}) |

## In this post

- **The three-rung ladder** — what each level of evidence can and cannot
  falsify.
- **The "encrypted" verdict that wasn't** — the DMO saga, capture by
  capture.
- **The operator loop** — env-gated replay tests as a user interface.
- **The gate that refused** — why a non-answer was the right answer.
- **Writing the status down** — institutional notes, `Refs` vs `Closes`,
  and the close that needs a human.

## The three-rung ladder

The workflow this series has been building implies a hierarchy of claims,
and it is worth making explicit, because each rung is defined by what it
**cannot** rule out:

**Synthetic-green.** Every unit test, round-trip, literal vector and
conformance harness passes. This falsifies internal inconsistency and —
with [Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
literal vectors — disagreement with a reference. It cannot falsify a
wrong model of the transmitter: the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
lives entirely inside this rung, and GopherTrunk's fabricated SmartNet
framing ([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143))
sat synthetic-green for its whole wrong life.

**Capture-verified.** A recording of a real transmitter decodes. This
falsifies the wrong-model class — the bits in the file were arranged by
somebody else's radio. It cannot falsify problems the capture doesn't
contain: the wrong RF conditions, the wrong network parameters, a
different vendor's option. As the DMO saga below shows, *each capture
verifies exactly one point in configuration space*, and generalizing from
it is a fresh assumption.

**On-air-verified.** The operator runs the shipped path on the live
system and confirms the original symptom is gone. Only this rung closes
an issue, because only this rung tests the claim the issue actually made.
The policy has a scar behind it: **#764 was closed twice** on a fix green
in every test while the reported symptom was still live (#771). The
issue-closing policy now requires a failing-first regression *plus*
reporter confirmation — or a reproduced symptom shown resolved — before
any close-as-completed.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Three gates in a row from left to right. First box: synthetic-green, licensed claim is the code agrees with itself and with references; below it, cannot see a wrong model of the transmitter. Arrow through a gate labelled real capture decodes into the second box: capture-verified, licensed claim is a real transmitter's bits decode; below it, cannot see conditions the capture lacks. Arrow through a gate labelled operator confirms on the live system into the third box: on-air-verified, licensed claim is the reported symptom is gone, and only this rung closes an issue. Beneath the row, a note that the DMO saga crossed the middle gate three times, each capture overturning the previous conclusion.">
  <rect x="16" y="40" width="180" height="76" rx="6" fill="none" stroke="currentColor"/>
  <text x="106" y="60" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">synthetic-green</text>
  <text x="106" y="78" text-anchor="middle" fill="currentColor" font-size="9">claim: agrees with itself</text>
  <text x="106" y="91" text-anchor="middle" fill="currentColor" font-size="9">and with references</text>
  <text x="106" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">blind to: a wrong model</text>
  <rect x="250" y="40" width="180" height="76" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="60" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">capture-verified</text>
  <text x="340" y="78" text-anchor="middle" fill="currentColor" font-size="9">claim: a real transmitter's</text>
  <text x="340" y="91" text-anchor="middle" fill="currentColor" font-size="9">bits decode</text>
  <text x="340" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">blind to: what the capture lacks</text>
  <rect x="484" y="40" width="180" height="76" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="574" y="60" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">on-air-verified</text>
  <text x="574" y="78" text-anchor="middle" fill="var(--accent)" font-size="9">claim: the reported</text>
  <text x="574" y="91" text-anchor="middle" fill="var(--accent)" font-size="9">symptom is gone</text>
  <text x="574" y="108" text-anchor="middle" fill="var(--accent)" font-size="9">only this rung closes an issue</text>
  <line x1="196" y1="78" x2="250" y2="78" stroke="var(--fg-muted)"/><polygon points="246,74 254,78 246,82" fill="var(--fg-muted)"/>
  <line x1="223" y1="58" x2="223" y2="98" stroke="var(--accent)"/>
  <text x="223" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">gate: a real capture decodes</text>
  <line x1="430" y1="78" x2="484" y2="78" stroke="var(--fg-muted)"/><polygon points="480,74 488,78 480,82" fill="var(--fg-muted)"/>
  <line x1="457" y1="58" x2="457" y2="98" stroke="var(--accent)"/>
  <text x="457" y="158" text-anchor="middle" fill="var(--fg-muted)" font-size="9">gate: operator confirms live</text>
  <text x="340" y="196" text-anchor="middle" fill="currentColor" font-size="10">the DMO saga crossed the middle gate three times —</text>
  <text x="340" y="212" text-anchor="middle" fill="currentColor" font-size="10">each new capture overturned the previous capture's conclusion</text>
</svg>
<figcaption>Three rungs of evidence, each defined by what it cannot falsify — and an issue closes only at the last one.</figcaption>
</figure>

## The "encrypted" verdict that wasn't

The DMO direct-mode investigation
([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003), told
at protocol depth in
[TETRA End to End Parts 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
[and 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}))
is the cleanest demonstration this project owns of why the middle rung
must be crossed repeatedly. Compressed to its verdicts:

| Round | Evidence | Conclusion drawn | Overturned by |
|---|---|---|---|
| 1 | first capture: signalling decodes, TCH/S at the ~1/256 CRC chance floor | "air-interface encrypted voice" | the reporter's codeplug: radios are **TEA0, clear**, colour 0 |
| 2 | code audit with the clear fact in hand | the voice path **skipped descrambling at colour 0** — TETRA scrambling is non-identity even at colour 0 | — (a real fix, but not yet the whole story) |
| 3 | 10aug capture: signalling 44/45, TCH/S still floored at colour 0 | colour sweep: **colour 3** → 35/269 CRC-valid, 70 speech frames, 2.1 s of PCM | nothing — but only for *this* capture |
| 4 | 15aug capture: ~90% signalling, **no colour dominates** (best 140 vs runner-up 74 of 831) | "marginal signal, partial keystream artifacts" | the radios' identity: Motorola MTP8500Ex running **MCC 250 / MNC 1** |
| 5 | the MNI fact | the colour search only ever tried MNI 0 — the true seed `ExtendedColourCode(250, 1, colour)` was **unreachable** by construction | still open: the MNI-folded fix is synthetic-pinned, awaiting the operator's A/B |

Three things to take from that table. First, **every round's conclusion
was reasonable on its evidence** — a chance-floor CRC really is what
encryption looks like; the failure was treating one capture's evidence as
more general than it was. Second, the facts that broke each deadlock —
TEA0-clear, the colour, the MNI — all came **from the operator's side of
the gate**: a codeplug screen, a radio model number. Third, round 2's fix
was real and necessary and *still* didn't make voice decode; rounds 3–5
each peeled another layer. A gate you cross once is a gate you'll cross
again.

The empirical answer that came out of round 3 is worth showing, because
it encodes the discipline in two constants:

```go
// internal/radio/tetra/dmo_decode.go (shape)
// Confidence gate for RecoverDMColourCode: the winning colour must clear
// this many CRC-valid TCH/S bursts AND beat the runner-up by this factor.
// The correct colour descrambles the class-2 protected speech while every
// wrong colour sits at the ~1/256 chance floor (measured ≈35 vs ≤3 on the
// #1003 10aug capture) — while an encrypted or unreceivable call clears
// neither.
const (
    dmColourMinCRC    = 6
    dmColourDominance = 3
)
```

The DM colour code is *not recoverable from the signalling* (the SCH/S is
always colour-0 scrambled), and pinning its exact bit offset in the
DM-SYNC SYSINFO from one capture — where colour 3 lights only two bits —
would have been the self-consistent trap in a new spot. So GopherTrunk
recovers it the way
[Part 6]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }})
resolved the DNB geometry: **design a measurement where the right answer
wins by a wide margin, and refuse to answer when it doesn't.** On the
15aug capture the gate refused — 140 is not 3 × 74 — and that refusal,
initially read as "the colour guesser is broken," was the harness
correctly reporting that its hypothesis space didn't contain the truth.
The truth (a non-zero MNI) enlarged the space; the gate stands unchanged.

## The operator loop as a user interface

Crossing the middle gate repeatedly only works if crossing is cheap for
the person who has the RF. GopherTrunk's answer is a family of env-gated
replay tests: they **skip cleanly** with a one-line instruction when their
capture is absent, and become field instruments when it's present. The
reproduce line for the DMO harness is a single command:

```
GT_TETRA_DMO_IQ=<capture.raw> GT_TETRA_DMO_RATE=144000 \
GT_TETRA_DMO_MCC=250 GT_TETRA_DMO_MNC=1 \
go test ./cmd/gophertrunk -run TestTETRADMOReplay -v
```

The part that makes it a *loop* rather than a log dump is that the output
speaks in verdicts, and the operator's ground truth changes what the
verdict says:

```go
// cmd/gophertrunk/tetra_dmo_replay_test.go (shape)
// GT_TETRA_DMO_CLEAR=1 asserts the capture is clear and flips the verdict.
if os.Getenv("GT_TETRA_DMO_CLEAR") == "1" {
    t.Logf("VERDICT: DMO SIGNALLING decodes (dsb_schs_crc=%d, distinct FN=%d) "+
        "but TCH/S is at the chance floor (tch_crc=%d/%d) on a capture asserted "+
        "CLEAR (TEA0) — this is a clear-voice DECODE defect to keep chasing "+
        "(geometry/interleave/colour), NOT encryption. …", /* … */)
} else {
    t.Logf("VERDICT: … either air-interface ENCRYPTED voice or a remaining "+
        "clear-voice decode defect. If the call is known CLEAR (TEA0), set "+
        "GT_TETRA_DMO_CLEAR=1; otherwise validate against a known-CLEAR DMO call.")
}
```

The same measurement, two conclusions — because the operator holds a
fact the harness cannot measure. Round 1's mistake, engineered away: the
harness will never again launder "chance floor" into "encrypted" when
somebody on the other end *knows* the call is clear. And when the verdict
says "keep chasing," the next instrument is one env var away:
`GT_TETRA_DMO_SCAN=1` runs `TestTETRADMOColourScan`, the 64-colour
CRC-yield sweep whose full colour→yield map is what made the 15aug "no
dominant colour" signature — and therefore the MNI blind spot — visible
at all. The design rule: **the test harness is an interface between two
people who each hold half the evidence**.

The gate cuts the other way too, and honestly. Sometimes the capture
proves the *decoder* innocent: the #764 endgame decimated the operator's
10 MS/s capture with an **independent resampler**, replayed it through
the proven 2.5 MS/s path, and reproduced the identical ~9.5 dB deficit —
the problem was baked into the samples, front-end phase noise, not DSP
([the full method]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})).
An on-air gate that can only ever indict your code is a ritual; one that
can acquit it is a measurement.

## Writing the status down

A ladder nobody records is a ladder that gets skipped under deadline. The
project's standing engineering notes mark every claim with its rung — the
phrase **"still on-air-gated"** appears verbatim next to the DMO MNI fix,
the conventional-DMR two-slot rebuild, and the SmartNet reconstruction,
each with the exact command an operator runs to move it up a rung. Three
concrete habits fall out:

- **PRs say `Refs #N`, not `Closes #N`, until the fix is verified** — so
  a merge cannot auto-close an issue whose symptom nobody has re-checked.
- **A close-as-completed requires a human to confirm the verification**,
  mechanically — the repo's tooling interposes the question, because
  #764 proved good intentions don't.
- **When you can't verify, you leave the issue open and say what's
  blocking** — "needs the reporter's capture" is a status, not a failure,
  and [Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})
  builds the whole definition of "done" on it.

### How that principle shaped the Go code

- **Verdict lines are code-reviewed prose.** The `t.Logf` strings in
  `tetra_dmo_replay_test.go` state the conclusion, the alternative, and
  the next command — they are written for the operator, and changing one
  is a semantic change.
- **Ground truth enters as input, not assumption.** `GT_TETRA_DMO_CLEAR`,
  `GT_TETRA_DMO_MCC/MNC` and `GT_TETRA_DMO_COLOUR` let the operator
  assert what they know; unset, the harness recovers what it can
  (`RecoverDMColourCode`) and says so.
- **Confidence gates return three values, not two.** `(colour, count,
  confident)` — and every caller of a non-confident result keeps its
  default rather than trusting a chance-floor winner.
- **The rung is greppable.** "on-air-gated," "capture-verified," and the
  reproduce commands live in the notes and test comments, so the next
  investigation starts knowing exactly which claims are load-bearing.

## Where this goes next

Everything in this part assumes a capture exists to replay — and getting
the *right* capture is its own discipline.
[Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
treats captures as first-class test fixtures: the `samples/` conventions
and metadata sidecars, the env-gated harnesses, how to ask a reporter for
the capture that can actually answer the question, and why you baseline
the metrics *before* touching the code.

## FAQ

**Why isn't a passing test suite enough to close a bug?**
Because the suite and the code can share the same wrong assumption — this
series' recurring villain. A test the fix's author wrote checks the
author's model; the reported symptom lives on a system that doesn't care
about the model. #764 was closed twice by exactly this overconfidence.

**What does a "CRC chance floor" mean and why does it matter?**
With random input, a CRC-gated decode still passes occasionally — for the
TCH/S class-2 check, at roughly 1/256. Yield *at* that floor means the
decoder is recovering nothing — but encryption, a wrong descramble seed,
wrong geometry, and severe RF damage all look identical there. The floor
says something is wrong; it cannot say what.

**How do I run GopherTrunk's DMO replay against my own capture?**
Record cs16 IQ on the DMO carrier, then
`GT_TETRA_DMO_IQ=<file> GT_TETRA_DMO_RATE=<hz> go test ./cmd/gophertrunk
-run TestTETRADMOReplay -v`. Add `GT_TETRA_DMO_CLEAR=1` if you know the
call is unencrypted, `GT_TETRA_DMO_MCC`/`GT_TETRA_DMO_MNC` if your
network runs a non-zero MNI, and `GT_TETRA_DMO_SCAN=1` (with
`TestTETRADMOColourScan`) to see the full colour-yield map.

**Isn't demanding operator confirmation slow?**
Slower than closing on green, faster than closing wrong. Every round of
the DMO saga that shipped a premature conclusion cost a full
capture-request cycle to unwind — and #764's two wrong closes cost more
credibility than any delay would have. The gate converts "we think" into
"we measured," and the loop is engineered to make measuring cheap.

**Does the gate apply to features, or only bug fixes?**
Both, with the same ladder. A new decoder path (the DMO production
pipeline, the SmartNet rebuild) ships synthetic-green and
capture-verified, is *documented* as on-air-gated, and its issue stays
open until the live confirmation lands. Shipping staged-but-honest beats
shipping "done."

## Series navigation

**Part 10 of 14** · ←
[Part 9: Wire Protocols Without Schemas]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})
· Next →
[Part 11: Capture-Driven Development]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
