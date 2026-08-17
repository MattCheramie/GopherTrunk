---
title: "Weak-Signal Engineering, Part 13: The Odd Path Out — P25 Phase 1 C4FM"
description: "A diagnosis, not a fix — why P25 Phase 1 C4FM voice is the one decode path in GopherTrunk with neither an equalizer nor soft-decision FEC, what the Astro Spectra report showed, and the baseline harness waiting on the one thing that lets a fix land: a real weak-signal capture."
category: deep-dives
keywords: p25 phase 1 c4fm, weak signal voice decode, fm discriminator receiver, imbe hard fec, missing equalizer, cqpsk fse equalizer, capture request p25, baseline metrics harness, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, p25, c4fm, imbe, equalizer, capture, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 13
---

*Part 13 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
showed how to prove a deficit lives in the samples before touching the code.
This part is the mirror case: a deficit that almost certainly lives in the
code — structurally, visibly — and still goes unfixed, on purpose. P25
Phase 1 C4FM voice is the one decode path in GopherTrunk that has **neither**
of the two levers that roughly doubled TETRA's marginal yield: no channel
equalizer, no soft-decision FEC. An operator's hardware radio decodes a
marginal call cleanly; GopherTrunk recovers a handful of frames on the same
antenna. We know why. We know the fix. And per the discipline this series
has enforced since Part 1, no change lands without a capture to measure it
against. This post is honest about being a roadmap — that's the point.*

> **TL;DR:** The default P25 Phase 1 **C4FM** voice receiver
> (`internal/radio/p25/phase1/receiver/receiver.go`) is FM discriminator →
> fixed matched filter (`demod.P25C4FMRxTaps`) → `CoarseAFC` →
> Mueller-Müller timing → AGC → 4-level slicer → **hard** Golay/Hamming IMBE
> FEC — no equalizer, no soft decisions. The opt-in CQPSK/LSM path has a T/2
> fractionally-spaced blind equalizer (the `fse` field in `cqpsk.go`), and
> P25 Phase 2 wires one too; C4FM Phase 1 is the odd path out. An operator
> whose Astro Spectra decodes a marginal Phase 1 call cleanly gets ~4–5 IMBE
> frames from GopherTrunk on the same antenna — consistent with the missing
> levers, and also too short to even qualify for the autotune fold
> (`minAutotuneLDUs = 5` in `composer/p25p1_voice.go`). The fix is
> unverifiable without a real weak C4FM **voice** capture, so the tree holds
> a baseline harness instead: drop a `.cfile` + metadata into `samples/p25/`
> and `TestReplayP25RealCaptureMetrics` measures pre-FEC EVM / SNR /
> FSW-margin as the number to improve.

**Key takeaways**

- **The gap is structural, not a bug.** Every stage of the C4FM chain does
  its job; the chain is simply missing the two stages — equalization and
  soft FEC — that the marginal regime demands. Nothing to patch; something
  to build.
- **The evidence is a differential diagnosis, not a stack trace.** A hardware
  radio decoding what GopherTrunk cannot, on the same antenna, bounds the
  problem: the information is in the RF; the software discards it somewhere.
  The chain inventory says where.
- **The levers are proven — next door.** A blind CMA/FSE equalizer already
  runs on the opt-in CQPSK/LSM path, and equalizer + soft decisions
  approximately doubled TETRA's marginal yield (Parts 4–8). The port is
  scoped; the measurement isn't.
- **Refusing to fix without a capture is the discipline, not an excuse.**
  A green synthetic is not proof of an on-air fix — the project has been
  burned by exactly that. The harness, the README ask, and the metadata
  format all exist so the day the capture arrives, the A/B starts
  immediately.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| C4FM voice chain | discriminator → MF → AFC → M&M → slicer, all hard | `internal/radio/p25/phase1/receiver/receiver.go` (`DemodC4FM` branch) |
| The equalizer next door | T/2 fractionally-spaced blind CMA on LSM | `internal/radio/p25/phase1/receiver/cqpsk.go` (`fse *equalizer.FSE`) |
| Hard IMBE FEC | Golay/Hamming with no soft inputs | `internal/radio/p25/phase1` voice frame path; see [IMBE channel coding]({{ '/reference/imbe-channel-coding/' | relative_url }}) |
| Short-call brush | <5 LDUs ⇒ autotune fold skipped | `internal/voice/composer/p25p1_voice.go` (`minAutotuneLDUs`) |
| Baseline harness | EVM / SNR / FSW-margin / NID / TSBK from a real capture | `cmd/gophertrunk/p25_realcapture_metrics_test.go` (`TestReplayP25RealCaptureMetrics`) |
| The ask | what to capture and how to annotate it | `samples/p25/README.md` (weak-signal voice section) |

## In this post

- **The report** — hardware decodes, we get 4–5 frames.
- **The chain, stage by stage** — where confidence goes to die.
- **The levers exist next door** — CQPSK's FSE, Phase 2, and the TETRA 2×.
- **Why no fix has landed** — the capture discipline, applied to ourselves.
- **The harness is waiting** — what to record and what happens next.

## The report: hardware decodes, we get 4–5 frames

The report that opened this investigation is the most useful kind: a
controlled comparison an operator ran without meaning to. A marginal P25
Phase 1 voice call — fringe site, real terrain — decodes cleanly on a
hardware Astro Spectra. The same call, same antenna, through GopherTrunk:
**about 4–5 IMBE frames** recovered, out of a call's worth. Not silence — a
lock that produces almost nothing, which is the marginal regime's signature
from [Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
in its purest form.

The hardware radio is the bound that matters. It proves the information
survives the antenna; whatever is lost is lost inside the software. (A
small aside the numbers brush against: at 9 frames per LDU, 4–5 frames is
well under the `minAutotuneLDUs = 5` threshold in `composer/p25p1_voice.go`,
so such a call is also treated as too short to trust for the per-dongle
autotune fold — a correct guard, but a reminder of how little the chain is
extracting from these calls.)

Commercial subscriber radios earn their weak-signal behaviour with exactly
the machinery this series has been building: adaptive equalization and
soft-decision error correction. So the first question is an inventory: what
does GopherTrunk's C4FM voice path actually run?

## The chain, stage by stage

The receiver's own doc comment lays it out:

```go
// internal/radio/p25/phase1/receiver/receiver.go (shape) — the C4FM branch
//	DemodC4FM (default):
//	    → FM discriminator (internal/dsp/demod.FM)
//	    → spec P25 C4FM matched filter (internal/dsp/demod.C4FM
//	      with demod.P25C4FMRxTaps — raised-cosine, not RRC)
//	    → coarse AFC: residual carrier-offset removal
//	      (internal/dsp/demod.CoarseAFC)
//	    → Mueller-Müller symbol clock recovery (sync.MuellerMuller)
//	    → 4-level slicer → C4FM symbol → 0..3 dibit
//	      (internal/dsp/demod.C4FM, phase1.SymbolToDibit)
```

Downstream of the dibits, the IMBE voice frames go through
[Golay]({{ '/reference/golay-code/' | relative_url }}) and
[Hamming]({{ '/reference/hamming-code/' | relative_url }}) decoding — hard
bits in, hard corrections out (the frame anatomy is
[Voice Coding, Part 5]({{ '/blog/deep-dives/voice-coding-05-imbe-fec-deinterleave/' | relative_url }})'s
territory). Now audit the chain against this series:

- **No equalizer, anywhere.** Between timing recovery and the slicer there is
  no stage that inverts ISI. Multipath smear on the 4-level eye —
  [Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})'s
  invertible impairment — goes uninverted, straight into symbol decisions.
- **A hard slicer feeding hard FEC.** The 4-level slicer collapses each
  symbol to a dibit and discards its distance from the thresholds — exactly
  the confidence the FEC needed
  ([Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})).
  The Golay/Hamming decoders then correct a bounded number of
  equally-weighted errors, defending coin-flip bits as firmly as certain
  ones.

Neither absence is a defect in what exists — the chain is a clean,
well-tested implementation of the classical C4FM receiver. It is simply a
fair-weather design, and the marginal regime is not fair weather.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="The P25 Phase 1 C4FM voice chain drawn as solid blocks: FM discriminator, matched filter, coarse AFC, Mueller-Muller timing, four-level slicer, and hard Golay and Hamming IMBE FEC. Two dashed accent blocks show where the missing stages would insert: an equalizer between timing recovery and the slicer, and soft LLRs replacing the hard bits into the FEC. A caption row notes the CQPSK path already has the equalizer.">
  <rect x="6" y="60" width="92" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="52" y="79" text-anchor="middle" fill="currentColor" font-size="9">FM</text>
  <text x="52" y="92" text-anchor="middle" fill="currentColor" font-size="9">discriminator</text>
  <line x1="98" y1="82" x2="112" y2="82" stroke="currentColor"/>
  <rect x="112" y="60" width="86" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="155" y="79" text-anchor="middle" fill="currentColor" font-size="9">matched filter</text>
  <text x="155" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">P25C4FMRxTaps</text>
  <line x1="198" y1="82" x2="212" y2="82" stroke="currentColor"/>
  <rect x="212" y="60" width="76" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="250" y="79" text-anchor="middle" fill="currentColor" font-size="9">CoarseAFC</text>
  <line x1="288" y1="82" x2="302" y2="82" stroke="currentColor"/>
  <rect x="302" y="60" width="90" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="347" y="79" text-anchor="middle" fill="currentColor" font-size="9">Mueller-Müller</text>
  <text x="347" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">timing + AGC</text>
  <rect x="404" y="40" width="104" height="36" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <text x="456" y="55" text-anchor="middle" fill="var(--accent)" font-size="9">equalizer</text>
  <text x="456" y="68" text-anchor="middle" fill="var(--accent)" font-size="8">MISSING (CMA/FSE)</text>
  <line x1="392" y1="82" x2="520" y2="82" stroke="currentColor"/>
  <rect x="520" y="60" width="72" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="556" y="79" text-anchor="middle" fill="currentColor" font-size="9">4-level</text>
  <text x="556" y="92" text-anchor="middle" fill="currentColor" font-size="9">slicer</text>
  <line x1="592" y1="82" x2="606" y2="82" stroke="currentColor"/>
  <rect x="606" y="60" width="68" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="640" y="79" text-anchor="middle" fill="currentColor" font-size="9">hard IMBE</text>
  <text x="640" y="92" text-anchor="middle" fill="currentColor" font-size="9">FEC</text>
  <rect x="540" y="128" width="134" height="36" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <text x="607" y="143" text-anchor="middle" fill="var(--accent)" font-size="9">soft decisions</text>
  <text x="607" y="156" text-anchor="middle" fill="var(--accent)" font-size="8">MISSING (LLRs → FEC)</text>
  <line x1="607" y1="128" x2="618" y2="106" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="456" y1="76" x2="456" y2="82" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="340" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the opt-in CQPSK/LSM path already runs a T/2 fractionally-spaced CMA equalizer (cqpsk.go's fse field);</text>
  <text x="340" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the default C4FM voice path has neither lever — the two that ~2×'d TETRA's marginal yield</text>
</svg>
<figcaption>Every solid block works; the dashed blocks are the diagnosis. C4FM Phase 1 voice is the one path missing both marginal-regime levers.</figcaption>
</figure>

## The levers exist next door

What makes this a roadmap rather than a research question is that both
missing stages already exist in the same codebase, proven on neighbouring
paths.

The equalizer: the opt-in `DemodCQPSK`/LSM receiver carries a **T/2
fractionally-spaced blind equalizer** — `fse *equalizer.FSE` in `cqpsk.go`,
built as `equalizer.NewFSE(cqpskFSESymbolSpan, cqpskFSEStep, 1.0,
cqpskFSELeak)` with a 6-symbol span, CMA step 0.025, and a leak of 5e-4
pulling the fractional taps back toward a delta when the signal is already
clean. Its constants were tuned to open the constellation within a few
hundred symbols of lead-in and its comments record precisely why
fractional spacing matters: a T/2 equalizer corrects channel shape a
symbol-spaced one cannot (the story of how that path was proven is
[From the Issue Tracker, Part 6]({{ '/blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/' | relative_url }})).
P25 **Phase 2** wires an equalizer as well. C4FM Phase 1 voice is,
literally, the odd path out.

The yield claim: on TETRA, the same two levers — a blind snapshot equalizer
([Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }}))
plus soft TCH/S
([Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})) —
roughly **doubled** CRC-valid yield on ISI-smeared and weak captures. The
physics does not care about the trunking protocol. There is every structural
reason to expect the C4FM port to move the number, and no measured reason
yet — which is exactly the distinction the next section is about.

## Why no fix has landed

The temptation is obvious: port the FSE onto the C4FM branch, add an LLR
track to the Golay/Hamming decoders, run the synthetic sweep, merge. The
project has a rule against it, written in scar tissue. Issue #764 was closed
twice on fixes that looked right and tested green against synthetic
fixtures; the symptom was still live both times
([Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
tells the eventual resolution). A synthetic channel exercises the code you
wrote through the impairments you *imagined*; the operator's hill, trees,
and fading exercise the ones you didn't. A green synthetic is necessary.
It is not proof.

So the standard here is explicit: baseline a real weak C4FM voice capture,
apply one lever, and A/B LDU/IMBE yield against that capture — decode yield
as the verdict, per
[Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }}).
Until such a capture exists, the port would be unverifiable, and an
unverifiable fix is how a project accumulates confident-looking code that
solves imagined problems. Diagnosed, scoped, and deliberately parked is the
honest state — and saying so publicly is part of the method.

## The harness is waiting

Everything that can be built without the capture has been. The measurement
harness, `TestReplayP25RealCaptureMetrics`
(`cmd/gophertrunk/p25_realcapture_metrics_test.go`, tag `integration`),
replays any capture dropped into `samples/p25/` through the real receiver
and reports the baseline numbers — pre-FEC EVM and estimated SNR from the
demod taps (`metrics.EVMC4FM` / `metrics.SNRResidualC4FM` on the soft eye
for C4FM; constellation variants for CQPSK), the FSW sync-margin
distribution, and NID/TSBK yields — with optional bounds in the metadata
sidecar so a capture can also serve as a permanent regression:

```go
// cmd/gophertrunk/p25_realcapture_metrics_test.go (shape) — expected bounds
MaxEVMPct float64 `json:"max_evm_pct"`
MinSNRdB  float64 `json:"min_snr_db"`
/* … EVM + SNR from the demod taps, FSW sync-margin distribution,
   NID trusted/failed, TSBK decoded/CRC-failed … */
```

The pipeline is validated end to end: a live UHF C4FM control-channel
capture already graded through it (EVM ≈ 12.7%, SNR ≈ 14.5 dB, NID
trusted 31/failed 0 — see the committed metadata sidecar in `samples/p25/`).
What's missing is the *voice* capture, and `samples/p25/README.md` spells
out the ask: tune the **granted voice frequency** (not the control channel)
during a call that is genuinely **weak or multipath-degraded** — a clean,
strong call will not exercise the missing equalizer — record C4FM (and
LSM/CQPSK separately if the site runs it), note the reference radio's
result in `tool_cross_check`, set `"expected": {"demod_mode": "c4fm"}` with
the quality bounds left out, and drop the `.cfile` + `.metadata.json` pair
in. The binary stays git-ignored; the sidecar is committed. From that
moment the sequence is mechanical: baseline it, port the `cqpsk.go`
equalizer onto the C4FM branch or add soft inputs to the IMBE FEC — the
capture decides which lever first — and A/B the LDU/IMBE yield. If you have
the RF conditions the report described, this is the single highest-leverage
recording you can contribute.

## Where this goes next

That is the last lever and the last case study. What remains is to fold
fourteen parts into something you can use at the bench:
[Part 14]({{ '/blog/deep-dives/weak-signal-engineering-14-playbook/' | relative_url }})
is the playbook — the decision tree from symptom to lever, the testing
discipline stack, and the thread capture's final scorecard.

## FAQ

**Why did the C4FM path end up the under-equipped one?**
History, not judgment. C4FM is the default and oldest Phase 1 path, built
when the goal was correct decode of viable signals; the CQPSK path was built
later, against simulcast distortion that *forced* an equalizer into the
design. The marginal-voice use case arrived last, and it is the one that
indicts the original chain.

**Couldn't the operator just use the CQPSK receiver for the weak site?**
No — the modulation on air is C4FM, and the CQPSK path's linear demod
expects LSM's phase trajectory (see
[From the Issue Tracker, Part 7]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
for why the two are not interchangeable). The equalizer must come to the
C4FM chain, not the signal to the equalizer.

**Which lever would likely land first?**
Whichever the capture indicts. High pre-FEC EVM with visible eye smear says
ISI — port the equalizer. A clean-ish eye with FEC failures clustered at low
sync margins says reliability — add soft decisions. That triage being
readable from the baseline numbers is exactly why the harness reports EVM,
SNR, and FSW margin rather than a single verdict.

**Is a capture from a different weak site as good as the reporter's?**
Nearly. The essential property is a *marginal C4FM voice call* — ideally
with a hardware radio's contemporaneous decode noted as the cross-check, so
"the information was present" is anchored to that specific recording rather
than assumed.

**Why publish a diagnosis before the fix exists?**
Because the bottleneck is the capture, and captures come from operators.
A precise public statement of what is missing, why, and exactly what
recording unblocks it is the fastest path to the fix — faster than quietly
shipping an unverifiable one.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Proving It's the Signal — Rate Invariance & Independent Resamplers]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
· Next →
[Part 14: The Weak-Signal Playbook]({{ '/blog/deep-dives/weak-signal-engineering-14-playbook/' | relative_url }})
