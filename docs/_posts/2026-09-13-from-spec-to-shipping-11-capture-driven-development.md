---
title: "From Spec to Shipping, Part 11: Capture-Driven Development"
description: How GopherTrunk turns operator IQ recordings into first-class test fixtures — the samples/ conventions, .metadata.json sidecars whose bounds turn log lines into pass/fail gates, env-gated replay harnesses that skip in CI and become field instruments, and the discipline of baselining before fixing.
category: deep-dives
keywords: iq capture test fixture, sdr capture metadata sidecar, replay test harness, env gated integration test, capture driven development, sample rate honesty, pre-combine diversity capture, regression fixture from capture, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, captures, testing, fixtures, replay, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 11
---

*Part 11 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
made the verification ladder explicit — and every rung above
synthetic-green stands on a recording of a real transmitter. This part is
about treating those recordings as what they are: **test fixtures with
provenance**. The `samples/` conventions, the metadata sidecars, the
env-gated harnesses that skip in CI and become instruments in the field,
and the two habits that decide whether a capture answers anything:
asking for the right one, and baselining before you fix.*

> **TL;DR:** A capture in GopherTrunk is a binary plus a committed
> `*.metadata.json` sidecar carrying provenance and **expected** bounds —
> absent bounds are reported, present bounds are asserted, so a fresh
> capture *measures* the decoder before anyone commits to a threshold
> (`samples/p25/README.md`; `TestReplayP25RealCaptureMetrics` in
> `cmd/gophertrunk/p25_realcapture_metrics_test.go`). Bigger captures
> drive **env-gated harnesses** — `GT_TETRA_DMO_IQ`/`RATE`/`MCC`/`MNC`,
> `GT_DIVERSITY_CAPTURE`, `GT_TETRA_LMS` — that skip cleanly with a
> one-line instruction and turn into field instruments when the file
> exists. The committed P25 control-channel fixture graded a live NAC
> 0x2C1 site at EVM 12.7% / SNR 14.5 dB / TSBK 36 decoded, 0 CRC-failed;
> those numbers are now floors. The rule underneath:
> **record the baseline before touching the code.**

**Key takeaways**

- **A capture without metadata is a smoke test; with metadata it's a
  regression gate.** The sidecar carries what the decoder *should* find —
  NAC, colour code, yield floors — and `gophertrunk test` grades against
  it with a CI-ready exit code.
- **Report first, assert later.** Every bound in the schema is optional:
  a new capture logs its metrics without failing anything, the measured
  numbers become the committed floors, and the floors tighten as the
  decoder improves — never the other way.
- **The harness must cost nothing when the capture is absent.** Skip-gated
  tests keep CI green with no fixture installed, while the skip message
  itself documents how to run them — the same file is a unit test in CI
  and an instrument on an operator's bench.
- **Most capture failures happen before recording starts.** Wrong tap
  (post-combine when the question is the combiner), wrong content (a
  silent PTT when the question is voice), wrong rate honesty — the
  asking-for-a-capture checklist is engineering, not admin.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Drop-zone conventions | per-protocol folders, format + acceptance criteria per README | `samples/README.md`, `samples/<proto>/README.md` |
| Sidecar schema | provenance + `expected` bounds; hex-tolerant field matching | `samples/README.md` (unified metadata schema) |
| P25 field-truth harness | replays `samples/p25/*.cfile`, reports/asserts EVM, SNR, sync margin, NID/TSBK yield | `cmd/gophertrunk/p25_realcapture_metrics_test.go` (`TestReplayP25RealCaptureMetrics`) |
| DMO capture harness | replay + colour recovery + verdict lines | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`GT_TETRA_DMO_*`) |
| Pre-combine diversity tap | one cs16 file per branch + `<prefix>.diversity.json`, alignment-guaranteed | `sdr.soapy_remote[].diversity_capture` → `TestDiversityCombinerReplay` (`GT_DIVERSITY_CAPTURE`) |
| Capture / grade tooling | `gophertrunk capture` writes IQ + sidecar; `gen` synthesizes; `test` grades, exit 0/1 | `samples/README.md` (Signal Lab section) |

## In this post

- **A fixture with provenance** — the `samples/` layout and what gets
  committed.
- **The sidecar schema** — bounds that report until you're ready to
  assert.
- **Env-gated harnesses** — one file, two lives: CI test and field
  instrument.
- **Asking for the right capture** — the checklist written in scar
  tissue.
- **Baseline before fix** — the habit that makes an A/B possible at all.

## A fixture with provenance

`samples/` is GopherTrunk's capture drop-zone: one folder per protocol
(`p25/`, `tetra/`, `nxdn/`, `dmr-tier2/`, `mpt1327/`, …), each with a
README stating the capture format the loader expects, the metadata needed
to validate a decode, and **numerical acceptance criteria** — the
explicit thresholds a contributor with hardware can run a capture against
to close a follow-up. The binaries themselves are deliberately
gitignored (multi-megabyte IQ does not belong in source control; anything
under ~1 MB may be committed directly); what *is* committed is the
sidecar, and the sidecar is where the value lives.

The committed `samples/p25/p25-450875-cc.metadata.json` shows the shape:
a live UHF C4FM control channel at 449.875 MHz, NAC `0x2C1`, channelised
to 48 kHz — and, in its `notes`, the measured grades: **EVM 12.7%, SNR
14.5 dB, sync-margin min=3/median=5, NID trusted=31/failed=0, TSBK
decoded=36/CRC-failed=0**. The `.cfile` never entered git; the fixture is
reproducible from the raw 2 MSPS recording via `TestGenerateP25Fixture`,
and the sidecar's bounds are set *below* the measured values — floors,
not targets, tightened as the demod improves. A capture handled this way
is not a debugging aid that evaporates when the issue closes; it is a
regression fixture with a birth certificate.

## The sidecar schema: report until you can assert

The harness side of that contract is `TestReplayP25RealCaptureMetrics`,
and its expected-payload struct encodes the discipline directly:

```go
// cmd/gophertrunk/p25_realcapture_metrics_test.go (shape)
// p25MetricsExpected is the "expected" payload for samples/p25/*.metadata.json.
// Every bound is optional: a zero/absent field is reported but not asserted,
// so a capture can be dropped in to *measure* the demod before anyone
// commits to a pass/fail threshold.
type p25MetricsExpected struct {
    DemodMode     string  `json:"demod_mode"` // "c4fm" (default) or "cqpsk"
    NAC           string  `json:"nac"`        // hex, e.g. "0x167"
    MinNIDTrusted int     `json:"min_nid_trusted"`
    MinTSBK       int     `json:"min_tsbk"`
    MaxEVMPct     float64 `json:"max_evm_pct"`
    MinSNRdB      float64 `json:"min_snr_db"`
    MinSyncMargin int     `json:"min_sync_margin"`
}
```

**Every bound optional** is the load-bearing design choice. Day one, an
operator drops a capture with only `demod_mode` and `nac` filled in; the
harness logs pre-FEC EVM, estimated SNR, FSW sync margin and NID/TSBK
yields without failing anything. Those logged numbers *are* the baseline.
Once trusted, they go into the sidecar as floors, and from that commit on
the capture bites: any regression in the demod fails a named test against
a named recording. The same pattern generalizes across the tree — the
unified schema in `samples/README.md` adds `lock`, `lock_latency_max_sec`
and hex-tolerant `lock_fields`, and `gophertrunk test -capture x.cfile`
grades any capture against its auto-discovered sidecar with a CI-ready
exit code. `gophertrunk capture` and `gen` both *write* a sidecar
alongside every file they produce, so a capture is never born naked.

## Env-gated harnesses: one file, two lives

Committed fixtures cover the small, stable end. The investigations that
drive this series run on captures too big, too private, or too
operator-specific to commit — and for those, GopherTrunk uses
**env-gated harnesses**: ordinary `go test` functions that skip with a
one-line instruction when their variable is unset.

| Variable(s) | Harness | The question it answers |
|---|---|---|
| `GT_TETRA_DMO_IQ`, `GT_TETRA_DMO_RATE` | `TestTETRADMOReplay` | does a real DMO transmission decode to speech? |
| `GT_TETRA_DMO_MCC`/`MNC`, `GT_TETRA_DMO_COLOUR`, `GT_TETRA_DMO_CLEAR` | same | fold in the operator's ground truth ([Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})) |
| `GT_TETRA_DMO_SCAN=1` | `TestTETRADMOColourScan` | the full 64-colour CRC-yield map |
| `GT_TETRA_LMS=1` | `TestTETRAMultiSlotReplay` | does the trained equalizer beat CMA on this capture? |
| `GT_DIVERSITY_CAPTURE` | `TestDiversityCombinerReplay` | is MRC helping, hurting, or irrelevant on this rig? |

The skip messages are documentation: `set GT_TETRA_DMO_IQ (cs16 IQ) +
GT_TETRA_DMO_RATE to run the DMO replay`. CI never sees the captures and
stays green; an operator with a `.raw` file is one env var from the full
instrument — windowed traces, decode arms, verdict lines. One source
file, two lives, and the two can never drift apart because they are the
same code.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="The capture lifecycle as five stages joined by arrows: an issue report saying voice does not decode; a capture request specifying tap, content and rate; the recording plus its metadata sidecar, with only the sidecar committed to git; the env-gated replay harness reporting a baseline; and finally a regression fixture whose measured numbers become asserted floors. A return arrow labelled next investigation starts here loops from the fixture back toward the issue stage.">
  <rect x="12" y="60" width="118" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="71" y="82" text-anchor="middle" fill="currentColor" font-size="10">issue report</text>
  <text x="71" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">"voice doesn't decode"</text>
  <line x1="130" y1="86" x2="148" y2="86" stroke="var(--fg-muted)"/><polygon points="146,82 154,86 146,90" fill="var(--fg-muted)"/>
  <rect x="154" y="60" width="118" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="213" y="78" text-anchor="middle" fill="currentColor" font-size="10">capture request</text>
  <text x="213" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">right tap, content,</text>
  <text x="213" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">honest rate</text>
  <line x1="272" y1="86" x2="290" y2="86" stroke="var(--fg-muted)"/><polygon points="288,82 296,86 288,90" fill="var(--fg-muted)"/>
  <rect x="296" y="60" width="118" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="355" y="78" text-anchor="middle" fill="currentColor" font-size="10">IQ + sidecar</text>
  <text x="355" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">binary gitignored,</text>
  <text x="355" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">.metadata.json committed</text>
  <line x1="414" y1="86" x2="432" y2="86" stroke="var(--fg-muted)"/><polygon points="430,82 438,86 430,90" fill="var(--fg-muted)"/>
  <rect x="438" y="60" width="118" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="497" y="78" text-anchor="middle" fill="currentColor" font-size="10">gated harness</text>
  <text x="497" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">skips in CI, reports</text>
  <text x="497" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the baseline in the field</text>
  <line x1="556" y1="86" x2="574" y2="86" stroke="var(--fg-muted)"/><polygon points="572,82 580,86 572,90" fill="var(--fg-muted)"/>
  <rect x="580" y="60" width="92" height="52" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="626" y="82" text-anchor="middle" fill="var(--accent)" font-size="10">regression</text>
  <text x="626" y="97" text-anchor="middle" fill="var(--accent)" font-size="10">fixture</text>
  <path d="M 626 112 L 626 160 L 71 160 L 71 118" fill="none" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <polygon points="67,122 71,114 75,122" fill="var(--accent)"/>
  <text x="348" y="152" text-anchor="middle" fill="var(--accent)" font-size="10">measured numbers become asserted floors — the next investigation starts ahead</text>
  <text x="340" y="200" text-anchor="middle" fill="currentColor" font-size="10">every stage leaves an artifact; the capture outlives the bug that requested it</text>
</svg>
<figcaption>The capture lifecycle: an issue buys a capture, the capture buys a baseline, and the baseline becomes a permanent floor under the decoder.</figcaption>
</figure>

## Asking for the right capture

The expensive failures in this workflow happen before a single sample is
recorded, and GopherTrunk's DMO and diversity investigations paid for the
checklist the hard way:

**Right tap.** The MRC combiner lives inside the SoapyRemote driver, so
every ordinary recording path — `baseband.auto_record`, the scope taps —
sits *downstream* of it: a capture from any of them has one combiner
already baked in and can never A/B another.
`sdr.soapy_remote[].diversity_capture` exists solely to tap the branches
before the combine, writing one headerless cs16 file per branch plus a
`.diversity.json` sidecar — and a datagram that didn't carry every branch
is dropped from **both** files and counted, never written short, because
one short write silently desynchronises the branches and poisons every
later conclusion.

**Right content.** A 25-second *silent* PTT looked like the perfect
clean DMO vector and turned out to be a poor one — largely comfort-noise
and discontinuous-transmission frames, exactly wrong for grading a speech
descramble. A later run had MRC enabled, a single antenna and no cavity
filter, which contaminated both decode paths at once. The standing ask is
now written into the harness notes: **known colour code (from the
codeplug), actually talking, single antenna, combiner off**. Similarly,
`samples/p25/README.md`'s priority ask is not another clean control
channel — it is a *marginal voice-channel* call, because a strong capture
cannot exercise the missing equalizer it exists to justify.

**Right physics.** IQ or nothing, for phase protocols: an FM-demodulated
audio recording of TETRA has already destroyed the phase the decoder
needs, and 128 kbps MP3 collapses a 4-FSK constellation's levels —
`samples/README.md` documents both empirically, from files contributors
actually uploaded. And rate honesty: a 48 kHz TETRA capture gives 2.67
samples/symbol and can never lock, however real the signal. The full
operator-side treatment — formats, sidecars, SigMF, rates — is
[The Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})'s
territory; this series' concern is what the engineering side must
*specify* when it asks.

The request itself is part of the diagnostic loop from
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }}):
each DMO round ended with a sharper ask, and the sharper ask is what made
the next round decisive.

## Baseline before fix

The last habit is the one that turns a capture into an experiment:
**run the metrics harness and record its output before touching the
code.** The P25 weak-signal plan in `samples/p25/README.md` spells the
order out — baseline the capture with `TestReplayP25RealCaptureMetrics`
(pre-FEC EVM, SNR, FSW margin, LDU yield), *then* port the equalizer or
add soft decisions, then A/B against the same file. Without the recorded
baseline there is no A/B — only a vague memory of "it seemed worse
before," which is how wishful DSP ships.

This is also why the harnesses print yields rather than verdicts wherever
a number will do: CRC-valid counts on the same capture are comparable
across months and branches. The
[ten-megasamples investigation]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
([#764](https://github.com/MattCheramie/GopherTrunk/issues/764)) was
settled exactly this way: the same capture, decimated by an independent
resampler, replayed through the proven path — deficit unchanged,
therefore in the samples, not the DSP. The capture was the experiment's
control group, and it could be, because nothing about it changed between
runs.

### How that principle shaped the Go code

- **Fixtures load through the production pipeline, not a shortcut.** The
  replay harnesses build the same down-converters and receivers the
  daemon runs, so a capture grades the shipping code path.
- **Sidecar bounds are floors with provenance.** Committed thresholds sit
  below measured values, with the measurement date and numbers in
  `notes` — so a failing gate points at a regression, not an optimistic
  guess.
- **Skip messages are the manual.** Every gated test's `t.Skip` string
  names the variables and the file format it wants; discovering how to
  run the instrument *is* running the test suite.
- **Capture tools emit metadata by default.** `gophertrunk capture` and
  `gen` write the sidecar with the file, so provenance exists from the
  moment the samples do.

## Where this goes next

A capture tells you what the decoder does; it doesn't yet force you to
fix it honestly.
[Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
is the rule that governs every fix in this repo: one narrow commit plus a
regression test that **fails without the fix and passes with it** — and
if you can't write the failing test, you haven't reproduced the bug,
whether or not you think you understand it.

## FAQ

**How do I contribute a capture to GopherTrunk?**
Read the protocol folder's README in `samples/`, record IQ in the
documented format (cfile/cs16 at an honest sample rate), and write the
`*.metadata.json` sidecar — protocol, rates, and whatever ground truth
you have (NAC, colour code, talkgroups). Commit the sidecar; link or
attach the binary per the README. A capture with expected values
validates a decoder; one without is only a does-it-crash smoke test.

**Why aren't the capture binaries committed to the repo?**
Size and churn — multi-megabyte opaque binaries bloat every clone
forever. The committed sidecar preserves the provenance and the measured
grades; small (≤ ~1 MB) representative slices are committed directly
when a regression test needs to run everywhere, like the
`mmr-s9-cc.cfile`-style fixtures under `cmd/gophertrunk/testdata/`.

**What belongs in a capture's metadata sidecar?**
Provenance (source, center frequency, sample rate, format), the receiver
configuration it implies (`demod_mode`, colour code), and the `expected`
block: identity fields the decode must match and yield/quality floors.
Start with identity only — the harness reports the rest — and promote
measured numbers to floors once you trust them.

**What sample rate should I record at?**
Higher than you think, honestly labelled. The decoder resamples to its
per-protocol channel rate, but information missing from the file never
comes back — 48 kHz TETRA structurally cannot lock — and a mislabelled
rate poisons everything downstream. When in doubt, capture wide and keep
the raw file; GopherTrunk's tooling can decimate later, and #764 was only
solvable because the raw recordings still existed.

**Do env-gated tests rot when nobody has the capture?**
Less than you'd expect, because they compile and their non-gated
neighbours share their helpers — and the ones that matter get run every
time an investigation touches their protocol. The real rot risk is the
opposite arrangement: instruments that live outside the test tree stop
building, silently, the week nobody needs them.

## Series navigation

**Part 11 of 14** · ←
[Part 10: The On-Air Gate — Green Synthetics Prove Nothing]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
· Next →
[Part 12: Failing First — The Regression Rule]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
