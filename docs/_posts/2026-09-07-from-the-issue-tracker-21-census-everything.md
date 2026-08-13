---
title: "From the Issue Tracker, Part 21: Census Everything — The Silence of a Success-Only Log Line Carries No Information"
description: Five GopherTrunk investigations stalled or went sideways because the logs only spoke on success — zero lines from a dead decoder looks identical at every failure stage. The fix each time was a census — an unconditional per-unit count with denominators — and this post distills when and how to add one.
category: solution-postmortem
keywords: diagnostic logging, success-only log, unconditional census, denominators, per-call census, unhandled opcode census, stage logging, error wrapping, lock-state artifact, misleading log message, observability
tags: [from-the-issue-tracker, diagnostics, logging, p25, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 21
---

*Part 21 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that
fought back. [Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
covered tests that validate their own bugs. This one covers the operational
twin: logs that validate their own health. A log line that fires only when
things work tells you nothing when they don't — and a pipeline that is silent
about its own failures will happily run dead for weeks. The remedy that
recurs across the tracker is the census: count every unit of work,
unconditionally, with denominators.*

> **TL;DR:** When a multi-stage pipeline fails, "no log output" is compatible
> with a failure at *every* stage — silence is one bit of information spread
> over N possible causes. GopherTrunk's hardest diagnosis sessions ended the
> moment someone added an unconditional census: a per-call
> `superframes=N … mac_pdus=N` line that disambiguated three failure stages at
> once ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)), a
> per-opcode count that proved a negative and found dropped calls as a side
> effect ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)).
> The anti-patterns are just as instructive: a diagnostic that only fires in
> one lock state read as evidence ([#881](https://github.com/MattCheramie/GopherTrunk/issues/881)),
> and an error message that described the exact opposite of the truth
> ([#379](https://github.com/MattCheramie/GopherTrunk/issues/379)).

## The problem with silence

Consider the P25 Phase 2 voice chain: superframe sync → ISCH classification →
MAC FEC → MAC PDU parse → fields published. The original diagnostic was
`composer: p25p2 mac pdu`, logged on a successful MAC decode. On real encrypted
traffic the operator saw zero of those lines. Which stage failed? Zero lines is
the identical observation whether superframe sync never locked, ISCH never
classified, or MAC FEC rejected every block. Three hypotheses, one
indistinguishable symptom. Weeks of [#813](https://github.com/MattCheramie/GopherTrunk/issues/813)
were spent inside that ambiguity — including a plausible, well-argued carrier
recovery fix that turned out to be real but irrelevant to the field symptom.

The silence of a success-only log line carries no information. It cannot,
structurally: the line's absence is consistent with every possible failure,
plus the pipeline not running at all, plus the log level filtering it out.

## The census that cracked #813

The fix was one log line, changed from conditional to unconditional. At the end
of every voice call — every call, even a total failure — the composer now emits
a census:

```
composer: p25p2 call census superframes=0 voice_subframes=0 mac_subframes=0 mac_pdus=0
```

plus a histogram of slot types seen. Each counter is a stage of the pipeline,
so one line disambiguates all three hypotheses at a glance: `superframes=0`
means sync never locked; `superframes=40 mac_subframes=0` means sync fine, ISCH
classification dead; `mac_subframes=12 mac_pdus=0` means classification fine,
FEC failing. The very first field report with the census returned
`superframes=0` on 67 of 67 calls — failure upstream of the MAC entirely, which
instantly retired half the hypothesis space and pointed at the sync word
itself (the truncated constant from
[Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})).

Two properties made it work:

- **It fires once per unit of work, unconditionally.** The unit here is a
  call. Zero is a reported value, not an absence — `superframes=0` is data;
  no line at all is not.
- **It carries denominators.** `mac_pdus=7` means nothing alone;
  `mac_subframes=10 mac_pdus=7` is a 70% decode rate. Numerators without
  denominators are how "some lines appeared" gets mistaken for health.

## Proving a negative: the #376 opcode census

The talker-alias hunt ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376))
needed the opposite kind of answer: not "which stage is failing" but "is the
data even here?" Three successive transport theories had already failed —
standard voice LCOs, a vendor control-channel TSBK, a speculative Phase 2
opcode. The instrument that ended the guessing was a census over the control
channel itself: an Info-level count per (opcode, MFID) pair of every TSBK the
decoder did *not* handle, with the raw payload hex attached, capped at 8
samples per pair so it could run indefinitely without flooding.

It answered the question by exhaustion: with every unhandled opcode enumerated
and their payloads cross-checked against known radio IDs, the alias was
provably *not* on the control channel — the search moved to the traffic
channel's signalling, where the aliases actually live. And the census paid a
dividend on the way: two of the unhandled vendor opcodes turned out to be
patch-group voice grants, whose calls GopherTrunk had been silently dropping.
Counting everything found a bug nobody was looking for.

That is the general virtue of a census over targeted logging: a targeted line
answers the question you thought to ask; a census answers questions you
haven't asked yet.

## Counting the black hole: #345

Between "TSBK decoded" and "call recorded" sits the grant dispatch path, and in
[#345](https://github.com/MattCheramie/GopherTrunk/issues/345) it contained a
literal black hole: grants referencing band-plan channel id 10 arrived, no
identifier update for id 10 had been decoded (the TDMA variant of the
identifier-update opcode was never dispatched), and the grants were dropped
with no counter, no log, nothing. The system looked idle while calls flowed.

The fix that made the *class* of bug visible was stage-tagged drop accounting:
a grant that dies now says where — `stage=no-bandplan` — and deferred grants
sit in a bounded, observable ring instead of vanishing. The same issue
contributed the third element of the lesson shape: **name the layer in error
wrappers.** The decoder's retry loop matched a sentinel error, but the failure
after a USB re-enumeration surfaced as a bare `usb: device disconnected` from
a different layer and silently ended the retry loop. Wrapping the stream-open
error with its layer (and `%w` so the sentinel still matches) is what turned
"the daemon runs on, half dead" into "retries exhausted, exit non-zero, let
the supervisor restart." An error string that names its layer —
`USB warmup:` vs `r82xx init: burst write:` — is a one-line census of where
you are in the stack, and it has cracked hardware bugs on its own.

## The diagnostic that lied by omission: #881

A census must fire unconditionally, and [#881](https://github.com/MattCheramie/GopherTrunk/issues/881)
is the definitive demonstration of why. A wideband device never decoded P25,
and the reporter built a compelling theory on a log histogram: the failing
device logged `p25/phase1: no FSW hits in chunk` with `dibits=18/19/20` —
chunks shorter than the 24-dibit frame sync word — while the working device
showed no such lines. Conclusion: the channel plan geometry starves the
demodulator of dibits.

The theory was entirely wrong, and the evidence was a lock-state artifact:
that line only fires while the decoder is *unlocked*. The working device
produced byte-identical ~19-dibit chunks — it simply stopped logging the
moment it locked. The two devices' logs differed not because their data
differed but because a conditional diagnostic sampled them in different
states. (The sync detector keeps a 24-dibit history across chunks precisely so
short chunks cannot prevent sync; the real culprit, found by capturing raw IQ,
was a hard-saturated front end — about half the samples pinned to the rails.)

A diagnostic whose firing condition correlates with the thing under study is
not an instrument; it is a confounder with a timestamp. If the line had been an
unconditional per-second census — `chunks=N fsw_hits=N locked=bool` — the two
devices would have shown identical chunk geometry and different lock states,
and the theory would never have formed.

## The message that was backwards: #379

The floor for diagnostic quality is that the message describes reality.
[#379](https://github.com/MattCheramie/GopherTrunk/issues/379) shipped a
message that described its exact opposite: "voice pool full but no actives."
The branch fires when the pool has no free device *and* no active call to
preempt — which is only reachable when the pool contains **zero devices**. The
pool was never full; it was empty, because the config defined only a
`role: control` SDR and there was nothing to collect into the voice pool. An
operator reading that line goes hunting for load problems; the actual fix is a
config line. The repair was threefold: a one-shot actionable warning naming the
real condition, a startup warning when the voice pool is empty, and the
unreachable branch downgraded to an explicit `(engine bug)` error — a message
that indicts the code, not the operator, if it ever appears.

## The lesson shape

Across all five, the same three rules fall out:

1. **Log denominators, not just numerators.** `uncorrectable_ldus=1622` was
   only diagnosable because `ldus=1622` sat next to it — *exactly* 100% is a
   structural bug, ~90% is signal quality. A rate needs both halves. Every
   counter you emit should answer "out of how many?"
2. **Make diagnostics fire unconditionally, once per unit of work.** Pick the
   unit — a call, a chunk, a scan pass — and report at its boundary no matter
   what happened, with zero as a first-class value. Anything gated on success,
   failure, or lock state will be read as evidence of the gate, not the data.
3. **Name the layer in error wrappers.** A failure that says which stage
   raised it (`stage=no-bandplan`, `USB warmup:`, `r82xx init: burst write:`)
   is a census entry; a bare error string is a mystery with a timestamp. And
   wrap with `%w`, so retry logic keyed on sentinels survives the decoration.

There is a cost argument against always-on counting, and the tracker's answer
is that the cost is bounded and the alternative is unbounded: cap payload
samples (8 per opcode in #376), aggregate per unit of work (one line per call
in #813), and the census runs forever in production — which is precisely where
the bugs are.

## What we keep

- Silence is not evidence. Before reasoning from a missing log line, check
  what condition gates the line — the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  starts there, and the
  [audio pipeline tells]({{ '/reference/audio-pipeline-tells/' | relative_url }})
  entry catalogs the pipeline whose "recordings work fine" silence misled the
  longest.
- A metric that alarms you in the failing case must be checked in the passing
  case ([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)'s
  "AGC stuck at 10×" was the normal operating point; #881's short chunks were
  universal). The [signal signatures]({{ '/reference/signal-signatures/' | relative_url }})
  entry collects the readings that look damning and aren't.
- When the question is "where does it die?", add the census first and
  hypothesize second. It is one log line, it disambiguates N stages at once,
  and it keeps answering questions for every future issue on the same path.

[Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
closes the series with the third grand pattern: two parallel code paths, one
contract, and the drift between them that turned "the fix didn't work" into a
recurring genre of bug report.
