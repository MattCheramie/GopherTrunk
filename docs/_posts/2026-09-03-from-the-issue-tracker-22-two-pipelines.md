---
title: "From the Issue Tracker, Part 22: Two Pipelines, One Symptom — When Parallel Code Paths Drift"
description: GopherTrunk grew parallel implementations of the same contract — daemon vs replay, single-channel vs wideband, C4FM vs CQPSK — and five issues trace to one path getting a fix, a warning, or a config knob the other never received. The series finale, with the rules that keep twins honest.
category: solution-postmortem
keywords: parallel code paths, daemon vs replay, wideband pipeline, ccdecoder, widebandt2, ddc bank, downconverter, config drift, trellis=0, demod mode, broken instrument, shared tests
tags: [from-the-issue-tracker, architecture, p25, dsp, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 22
---

*Part 22 — the last — of **From the Issue Tracker**, postmortems of GopherTrunk
bugs that fought back. [Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
argued for counting everything; this part is about *where* to count, because a
count taken on one code path says nothing about its twin. GopherTrunk, like any
system that grows, has pairs of pipelines that implement the same contract:
live daemon and offline replay, single-channel decoder and wideband
channelizer, C4FM and CQPSK demodulators. Five issues in the tracker are, at
bottom, the same bug — the pair drifted, and everyone reasoned about one
member while the symptom lived in the other.*

> **TL;DR:** When two code paths implement one contract, every fix, warning,
> and config knob must land in both — and nothing enforces that by default. A
> DDC fix landed in the wideband bank while replay ran a separate
> down-converter ([#771](https://github.com/MattCheramie/GopherTrunk/issues/771));
> a carrier-offset warning lived only in the daemon
> ([#815](https://github.com/MattCheramie/GopherTrunk/issues/815)); Phase 2
> FEC options reached one control-channel pipeline and defaulted to zero in
> the other ([#882](https://github.com/MattCheramie/GopherTrunk/issues/882));
> the wideband path never stamped a demod mode at all
> ([#935](https://github.com/MattCheramie/GopherTrunk/issues/935)); and a
> diagnostic printed one demodulator's gauges while the other was running
> ([#492](https://github.com/MattCheramie/GopherTrunk/issues/492)). The rules
> that fall out: share the code or share the tests, make each pipeline
> announce its own configuration, and read "the fix didn't work" as "which
> path did you fix?"

## Cheat sheet

| Issue | The pair | What only one side had |
| --- | --- | --- |
| [#771](https://github.com/MattCheramie/GopherTrunk/issues/771) | live wideband DDC vs replay DDC | the #768 decimation fix |
| [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) | daemon CC decoder vs replay | the carrier-offset warning |
| [#882](https://github.com/MattCheramie/GopherTrunk/issues/882) | single-channel vs wideband CC pipeline | P25 Phase 2 FEC options |
| [#935](https://github.com/MattCheramie/GopherTrunk/issues/935) | single-channel vs wideband grants | any demod-mode stamp at all |
| [#492](https://github.com/MattCheramie/GopherTrunk/issues/492) | C4FM vs CQPSK receiver | gauges the diag line actually read |

## In this post

- **How twins are born** — each pair was justified; the drift came one commit at a time.
- **The fix that landed on the other path: #771** — a report structurally incapable of being about the fix.
- **The warning only the daemon can say: #815** — a diagnostic bound to one pipeline protects one pipeline.
- **Config that reads back correctly and never arrives: #882** — the knob that validates, displays, and is ignored.
- **The knob one path never had: #935** — the sibling audit, paid out one issue later.
- **The instrument from the wrong pipeline: #492** — gauges wired to the other engine.
- **The rules for keeping twins honest** — four rules that fall out.
- **Closing the series** — what twenty-two parts add up to.

## How twins are born

None of these pairs was a mistake when it was created. Replay exists so a bug
can be reproduced offline from a capture — it wants to be a thin harness, not
the whole daemon. The wideband engine exists because one dongle can watch a
whole site — but the single-channel decoder predates it and still serves the
one-antenna case. CQPSK exists because some simulcast sites won't lock any
other way. Each twin was justified. The drift came later, one commit at a
time, because a change made where the bug was observed has no natural pressure
to visit the sibling nobody was looking at.

## The fix that landed on the other path: #771

The definitive instance. [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)'s
high-sample-rate fix rebuilt the shared decimation in `tuner.DDCBank`
(`internal/dsp/tuner/ddc.go`) — the bank that serves the live wideband path
and the spectrum view. The follow-up report, [#771](https://github.com/MattCheramie/GopherTrunk/issues/771),
said the fix didn't work: the same failure reproduced in *offline replay*. But
`gophertrunk replay -tune-hz` doesn't touch `DDCBank` at all — it runs the
single-channel `ccdecoder.Downconverter` (`internal/scanner/ccdecoder/ddc.go`),
a separate implementation the fix never modified. The report was structurally
incapable of being about that fix.

Once that was recognized, the elimination could run clean — and it is worth
recording how systematic it was. The report's headline metric died first: "AGC
stuck at 10× above target" turned out to be the *normal operating point* — the
working 2.5 MS/s capture showed the identical `agc_level≈1.47`, because a
symbol-domain AGC's gain scales with the matched filter's samples-per-symbol.
Then the replay path's own suspects fell one by one: band-limiting the capture
before decode changed nothing (no aliasing from out-of-band neighbours), a
float64 rerun was bit-identical (no precision loss in the NCO or the
polyphase), and squeezing the channel filter from 24 kHz to 6.25 kHz moved
post-discriminator SNR by less than half a dB. The decisive move was a
gain-independence test: captures at gain 600 versus gain 300 — about 6 dB less
power — decoded to essentially the same EVM (22.5% vs 22.7%), which rules out
compression and intermodulation and leaves sampling-clock phase noise. The
deficit was baked into the captured samples by the SDR's own front end at its
native 10 MS/s clock: the same file decimated 4:1 by an *independent*
resampler and decoded through the proven 2.5 MS/s path kept the same ≈9.5 dB
SNR, while a native 2.5 MS/s capture of the same carrier measured ≈19.7 dB.
Both down-converters were fine. The two-pipeline confusion didn't cause the RF
problem, but it burned the first round of the investigation and framed the
report as a regression that never was.

The durable output was a sentence now pinned in the repo's own notes: the
replay path and the live wideband path are separate code — *a fix to one does
not touch the other.*

## The warning only the daemon can say: #815

[#815](https://github.com/MattCheramie/GopherTrunk/issues/815) found GopherTrunk
confidently decoding the *wrong site*: no carrier at the configured frequency,
a neighbor 12.5 kHz up, and the DDC's ~±24 kHz passband happy to lock it —
while reporting the configured frequency as truth. Off-pipeline `capture` +
`spectrum` made it vivid: literally nothing at 0 Hz offset, the dominant
carrier at +12.2 kHz, −76.6 dBFS, SNR 26.6 dB. The receiver had been measuring
the truth the whole time — its AFC offset read 12,436–12,699 Hz live, almost
exactly the channel spacing — but nothing inspected that value unless autotune
was on. And nothing in the decode disagreed with itself: the reporter only
caught the wrong-site lock by cross-referencing the decoded RFSS/site identity
against RadioReference and the national spectrum-licensing records and noticing
the identity didn't belong on that frequency. The fix was a warning keyed on
the receiver's measured AFC offset (`sdr.carrier_offset_warn_hz`, default
4 kHz — a ~12.5 kHz reading is the fingerprint of an adjacent-carrier lock);
telling the user, not out-smarting the DSP, because autotune could never
legitimately chase a 12.5 kHz error anyway (its plausibility bound sits near
1.5 kHz at 420 MHz).

The two-pipeline catch: that warning lives in the daemon's
`ccdecoder.Decoder`. Replay drives the receiver directly and never constructs
a `ccdecoder.Decoder` — so replaying the *very capture that demonstrated the
bug* would never emit the warning built to catch it. A diagnostic bound to one
pipeline protects one pipeline. The offline harness that exists precisely for
verifying fixes couldn't show this fix working, which is its own small
instance of the pattern: the verification tool and the production path had
drifted apart on the exact axis under test.

The pattern even reproduced *inside* the fix. The published API field was
wired to the residual AFC offset only, while the WARN computed
applied-autotune-plus-residual — so with autotune on, the API field would hide
the very adjacent-carrier lock the WARN flags. Two consumers of one
measurement, two different formulas: a micro-scale pipeline pair, caught and
reconciled in a follow-up ([#866](https://github.com/MattCheramie/GopherTrunk/issues/866)).

## Config that reads back correctly and never arrives: #882

The subtlest failure mode is a knob that exists, validates, and displays —
on a code path that ignores it. [#882](https://github.com/MattCheramie/GopherTrunk/issues/882):
the single-channel CC pipeline (`internal/scanner/ccdecoder/pipelines.go`)
passed the P25 Phase 2 FEC options — trellis, Reed-Solomon, interleave,
scrambler — into the decoder it built. The wideband engine
(`internal/scanner/widebandt2/engine.go`) built the same decoder and passed
none of them; they defaulted to zero. A helper that applied the Phase 2 modes
existed, but only ran for Phase 2 control-channel taps — and the reporter's
topology, a Phase 1 control channel granting Phase 2 voice, fell exactly in
the gap.

From the operator's chair this is maddening: the config file has the options,
the API reads them back correctly, and the decoder never sees them. Nothing
between the YAML and the constructor validates the *hand-off*. The observable
tell was a single field in a startup line — `composer: p25p2 voice chain
started trellis=0` where a correctly-plumbed system says `trellis=1` — which
is [Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})'s
lesson applied here: the pipeline *announced its own effective configuration*,
and the announcement disagreed with the config file. That one log field
converted "mysterious Phase 2 decode failure" into "the wideband path drops
the options," and the fix verified on air the same day (`trellis=1 rs=0
interleave=0 scrambler=1`, MAC PDUs flowing 7/10).

## The knob one path never had: #935 and the #882 remainder

While fixing #882, an adjacent gap was found and deferred: the wideband path
also never forwarded the Phase 1 demodulator mode. [#935](https://github.com/MattCheramie/GopherTrunk/issues/935)
paid that deferral out. The wideband decode path stamped *no* demod mode onto
its grants, so wideband P25 voice always ran the C4FM chain regardless of any
setting anywhere — the CQPSK option was, on that path, decorative.

The same issue overturned a piece of folklore worth its own paragraph, because
the project's own docs, config comments, and config-builder labels
("CQPSK / LSM (simulcast)") had it backwards — steering operators on simulcast
sites into CQPSK. LSM is a transmitter *coordination* technique — timing and
phase alignment across towers — not a baseband modulation, and the reporter's
genuinely three-tower simulcast site decodes reliably in C4FM (in GopherTrunk
and SDRTrunk alike) while forcing CQPSK kills the decode entirely. Licensing
metadata can't decide either: the site's emission designator (`10K1D7W`)
covers both modulations. The corrected rule shipped across three doc surfaces:
choose `cqpsk` *empirically* — a strong, clean signal that won't lock in
C4FM — never by inferring from "the site is simulcast." (The site's actual
problem was neither modulation nor plumbing but a gain mis-translation: the
working SDRTrunk gain of ~36 dB is `gain: "363"` in GopherTrunk's
tenths-of-a-dB form. Set correctly on a known-good antenna port: locked, ~2,000
grants in five minutes.) The per-channel override that shipped alongside is
keyed by frequency rather than RFSS/site for a chicken-and-egg reason: the
demod mode must be chosen *before* a control channel can lock — it's what lets
it lock — but RFSS/site identity is only known *after* decoding that control
channel, so the wideband `channels:` entry is the only place site identity
exists at config time.

Twice in two issues, the same seam: the single-channel pipeline carried a
property the wideband pipeline silently dropped. When a gap is found on one
side of a known pair, the sibling deserves an immediate audit for the *whole
class* — the second gap was already visible in the first issue's margins.

## The instrument from the wrong pipeline: #492

Drift doesn't only eat fixes and config — it eats diagnostics. In
[#492](https://github.com/MattCheramie/GopherTrunk/issues/492), the CQPSK path
wouldn't lock, and the reporter's smoking gun was `mm_sps=0.00` — the symbol
clock apparently stopped. But the diag line printed the *C4FM* path's
Mueller-Müller accessors, and the CQPSK path uses a Gardner loop; on CQPSK
those accessors legitimately return zero. The "timing slip" theory was built
on a gauge wired to the other engine.

Recognizing the broken instrument took nothing away from how hard the real
repair was — and the repair itself opened on another twin-gap: the CQPSK path
had **no carrier-frequency recovery at all**, because `CoarseAFC` existed only
on its C4FM sibling. Four sequential fixes — carrier recovery, a
multipath-gated acquisition seed with anti-windup, a T/2 fractionally-spaced
equalizer, and a fast BCH decoder — took real-capture CQPSK locks from 0/8 to
3/8 to 8/8. All of it happened *after* the broken instrument was recognized as
broken. A shared diagnostic surface over divergent pipelines is worse than
none: it produces readings that are precise, plausible, and about the wrong
machine.

## The rules for keeping twins honest

1. **Share the code, or share the tests.** The best fix is to have one
   implementation (the #764 repair made the per-tap DDCs consume one shared
   decimation stage). Where the twins must stay separate — replay *should* be
   thinner than the daemon — run the same conformance fixtures through both.
   A capture that must decode identically through `replay` and through the
   daemon's pipeline is a drift alarm that fires in CI instead of in a
   reporter's field setup.
2. **Make each pipeline announce its own configuration.** Not what the config
   file says — what the constructed pipeline actually holds. `trellis=0` in a
   startup line solved #882; the demod/rotation/build line added back in
   [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) exists for
   the same reason. Announcements make hand-off bugs one `grep` deep, and
   they keep diagnostics honest about which engine they describe (#492's
   gauges would have been caught by a line that named its demodulator).
3. **Treat "the fix didn't work" as "which path did you fix?"** Before
   re-opening the debugging of the fix itself, route the report: what binary,
   what subcommand, what role, what topology? #771's answer was "replay —
   which the fix never touched." The question costs one message and can save
   a week.
4. **When you find a gap on one side of a pair, audit the sibling for the
   class.** #882's deferred finding became #935's bug. Pairs drift together;
   gaps come in litters.

## FAQ

**Why not merge each pair into one implementation and be done?**
Because most of the pairs earn their separation: replay is deliberately a thin
offline harness, the single-channel and wideband paths serve different
hardware topologies, and C4FM/CQPSK are different demodulation problems. The
sin is not having twins; it is letting them drift unmanaged. Where merging is
possible — the shared decimation stage from #764 — it is the strongest fix;
everywhere else, shared conformance fixtures are the substitute.

**How do you find out which pipeline a report is actually about?**
Route it before debugging it: which binary, which subcommand (`daemon` vs
`replay`), which SDR role (`control`, `voice`, `wideband`), which topology
(Phase 1 CC granting Phase 2 voice was #882's gap). One question in the thread
— #771's "this reproduces in replay" — can reclassify an entire investigation.

**Can the drift happen inside a single feature?**
Yes, and #815 proved it: the WARN and the published API field computed
different offset formulas from the same measurement, so with autotune on the
API hid the lock the WARN flagged. Any time one measurement has two consumers,
the pair-drift rules apply at miniature scale.

**What's the minimum CI guard against pipeline drift?**
Two cheap things: one shared capture decoded through both members of every
pair with the outputs asserted identical, and a startup announcement of each
pipeline's effective configuration so a dropped knob is one `grep` away. The
first catches behavioral drift; the second catches hand-off drift.

## Closing the series

Twenty-two parts ago this series set out to write down what GopherTrunk's
issue tracker actually taught — not the changelog, but the reasoning: the
plausible theories that were wrong, the diagnostics that cracked each case,
and the habits that would have shortened the hunt. The single-bug stories
(Parts 1–19) each ended with lessons; these last three parts collected the
lessons that kept recurring. Round-trip tests validate their own bugs — break
the symmetry with real captures and independent references
([Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})).
Success-only logging is silence when you need it most — census everything,
with denominators ([Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})).
And parallel pipelines drift — share code or share tests, and make every
pipeline say what it's running (this part).

The condensed, evergreen versions of all of it live in the Field Guide's
[Field Notes domain]({{ '/reference/#domain-field-notes' | relative_url }}) —
the [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}),
[signal signatures]({{ '/reference/signal-signatures/' | relative_url }}),
[wideband voice taps]({{ '/reference/wideband-voice-taps/' | relative_url }}),
and their neighbors — each entry tracing back to the issues that earned it.
When the next bug fights back, start there. And when it teaches something new,
it goes in the tracker first and the Field Notes after — that loop is the
whole method.

Thanks for reading along. The tracker is still open; the bugs are still
coming; the lessons keep compounding.

## Series navigation

**Part 22 of 22** · ←
[Part 21: Census Everything — The Silence of a Success-Only Log Line Carries No Information]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
