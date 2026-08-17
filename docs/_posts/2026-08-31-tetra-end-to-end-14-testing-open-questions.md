---
title: "TETRA End to End, Part 14: Testing TETRA Without a Network — & What's Still Open"
description: The finale — the four-layer test lattice that lets a TETRA stack be developed with no network in range, from synthetic round-trips through skip-guarded capture harnesses to full-daemon integration, and an unvarnished list of what is verified, what is staged, and what stays open until on-air evidence lands.
category: deep-dives
keywords: tetra testing, capture replay harness, skip-guarded tests, synthetic iq round trip, self-consistent test trap, dmo colour scan, integration testing sdr, on-air verification, gophertrunk tetra
tags: [tetra-end-to-end, tetra, testing, verification, dmo, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 14
---

*Part 14 — the finale — of **TETRA End to End**. Thirteen posts ago we started
with a π/4-DQPSK constellation on a 25 kHz carrier and a promise: every claim
pinned by a test or a capture. Since then we've built the slot grid, the
channel coding, the scrambler, TCH/S, a clean-room ACELP vocoder proven
bit-identical against the ETSI reference, soft decisions, two equalizers, a
hardened control channel, and a DMO path that went from "reads the spec" to a
production pipeline — pausing once to retract a wrong verdict in public. This
close-out is about the machinery that made that pace survivable: the **test
lattice** that lets a TETRA stack be developed hundreds of kilometers from the
nearest TETRA network — and, in the same breath, the honest list of what that
lattice cannot prove and what therefore remains open.*

> **TL;DR:** TETRA in GopherTrunk is tested in four layers, each catching what
> the previous one can't: **synthetic round-trips** (fast, but blind to
> self-consistent bugs unless the encode side models the *air* — the colour-0
> and slot-grid lessons); **skip-guarded capture harnesses**
> (`TestTETRAMultiSlotReplay`, `TestTETRADMOReplay`,
> `TestTETRADMOColourScan` — activated by `GT_TETRA_DMO_IQ`,
> `GT_TETRA_DMO_RATE`, `GT_TETRA_DMO_COLOUR`, `GT_TETRA_DMO_CLEAR`,
> `GT_TETRA_DMO_SCAN`, `GT_TETRA_LMS`, `GT_TETRA_DMO_LMS`); **pipeline tests on
> modulated IQ** (`pipelines_dmo_test.go`, the CC equalizer A/B); and
> **full-daemon integration** (`TestDaemonDecodesTETRADMO`, under
> `make integration`). Still open, stated plainly: the
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) on-air A/B
> through the full daemon; DMO call-control (recordings file under group 0);
> and the 15aug lesson — a colour scan with **no dominant winner means a
> marginal signal, not a broken guesser**, and the descramble must never be
> bent to fit a 33%-yield colour.

**Key takeaways**

- **Each test layer exists because the one below it lied at least once.**
  Round-trips passed through the colour-0 bug; only an encode side that
  scrambles unconditionally — modeling the transmitter, not the code under
  test — made it fail first.
- **Skip-guarded harnesses turn operators into the lab.** A test that skips
  without `GT_TETRA_DMO_IQ` costs CI nothing, yet gives anyone with a capture
  a one-command, fully instrumented replay — every major TETRA finding in this
  series arrived through one.
- **A refusal can be a correct answer.** `RecoverDMColourCode` declining to
  pick a colour on the 15aug capture wasn't a bug — the scan's shape (several
  modest risers, no 3× dominance) is physically impossible for one radio with
  one keystream, so the gate correctly diagnosed a marginal signal.
- **Green everything ≠ done.** The DMO daemon path passes synthetic, capture,
  pipeline, and integration layers — and #1003 stays open anyway, because the
  one test that counts is a recording an operator can listen to.

## Cheat sheet

| Layer | What it proves | Where it lives |
|---|---|---|
| Vocoder conformance | bit-identical PCM vs the ETSI reference | `internal/voice/acelp/etsi_reference_test.go` |
| Synthetic round-trips | coding chains invert; encode models the air | `internal/radio/tetra/dmo_decode_test.go`, `tch_test.go` |
| TMO capture harness | grant-correlated multislot voice, LMS A/B | `cmd/gophertrunk/tetra_multislot_replay_test.go` (`GT_TETRA_LMS`) |
| DMO capture harness | lock, colour, TCH/S yield, verdict line | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`TestTETRADMOReplay`) |
| Colour diagnostics | full 64-colour yield map | `tetra_dmo_replay_test.go` (`TestTETRADMOColourScan`, `GT_TETRA_DMO_SCAN=1`) |
| Pipeline on modulated IQ | lock + colour + grant from synthesized bursts | `internal/scanner/ccdecoder/pipelines_dmo_test.go` |
| CC equalizer proof | 12%→100% BSCH on a real marginal fixture | `pipelines_tetra_equalizer_test.go` |
| Full daemon | config → DDC → pipeline → lock, no shortcuts | `cmd/gophertrunk/integration_cc_tetra_dmo_test.go` (`make integration`) |

## In this post

- **The lattice, bottom to top** — four layers and the failure each one owns.
- **Making synthetics honest** — the two fixture repairs that mattered.
- **The capture harness as an instrument** — env vars, verdict lines, A/B knobs.
- **The 15aug capture: reading a refusal** — when no colour dominates.
- **The open list** — what stays unverified, and the gate for each item.

## The lattice, bottom to top

The bottom layer is **conformance and round-trips**: `etsi_reference_test.go`
feeds the same 137-bit stream to GopherTrunk's ACELP and the ETSI reference
codec and demands bit-identical PCM
([Part 7]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }}));
encode/decode pairs pin every coding chain. Fast, deterministic, runs on every
`make vet test`. Its blind spot is the one this series hit twice: a bug on
*both* sides of a round-trip is invisible.

The second layer is **skip-guarded capture harnesses** — ordinary `go test`
functions that skip unless an environment variable points at real IQ. The third
is **pipeline tests on modulated IQ**: synthesize actual bursts, modulate them,
and drive the *production* pipeline object — `TestTETRADMOPipelineLocksAndGrants`
asserts lock, colour-3 recovery, and a grant from IQ in;
`TestTETRACCEqualizerLiftsMarginalBSCHYield` holds the live TMO pipeline to the
healthy regime on a real marginal fixture
([Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})).
The top layer is **full-daemon integration** under `make integration`:
`TestDaemonDecodesTETRADMO` boots the daemon from a `protocol: tetra-dmo`
config and requires the whole path — DDC, pipeline, bus — to lock, with no
hand-wired shortcuts. Above all four sits the layer no test framework
provides: an operator, a radio, and the air.

<figure class="lab-figure">
<svg viewBox="0 0 680 218" width="680" height="218" role="img" aria-label="The TETRA test lattice drawn as five stacked layers. From bottom to top: synthetic round-trips and ETSI conformance, skip-guarded capture harnesses driven by environment variables, pipeline tests on modulated IQ, full-daemon integration under make integration, and at the top on-air verification by an operator — the only layer that can close an issue. Annotations on the right name the failure class each layer catches that the layer below cannot.">
  <rect x="60" y="176" width="440" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="280" y="196" text-anchor="middle" fill="currentColor" font-size="10">synthetic round-trips · ETSI conformance (every make vet test)</text>
  <text x="512" y="196" fill="var(--fg-muted)" font-size="9">fast; blind to self-consistent bugs</text>
  <rect x="85" y="138" width="390" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="280" y="158" text-anchor="middle" fill="currentColor" font-size="10">skip-guarded capture harnesses (GT_TETRA_* env vars)</text>
  <text x="512" y="158" fill="var(--fg-muted)" font-size="9">real air, offline decoders</text>
  <rect x="110" y="100" width="340" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="280" y="120" text-anchor="middle" fill="currentColor" font-size="10">pipeline tests on modulated IQ</text>
  <text x="512" y="120" fill="var(--fg-muted)" font-size="9">production objects, synthetic RF</text>
  <rect x="135" y="62" width="290" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="280" y="82" text-anchor="middle" fill="currentColor" font-size="10">full daemon (make integration)</text>
  <text x="512" y="82" fill="var(--fg-muted)" font-size="9">wiring: config → DDC → bus</text>
  <rect x="160" y="24" width="240" height="32" rx="5" fill="var(--accent)" opacity="0.15" stroke="var(--accent)"/>
  <text x="280" y="44" text-anchor="middle" fill="var(--accent)" font-size="10">on-air A/B — the operator loop</text>
  <text x="512" y="44" fill="var(--accent)" font-size="9">the only layer that closes an issue</text>
  <text x="280" y="215" text-anchor="middle" fill="var(--fg-muted)" font-size="10">each layer exists because the one below it passed while the system was wrong (#764/#771)</text>
</svg>
<figcaption>The four automated layers of the TETRA test lattice, and the fifth — on-air verification — that no green run below it can substitute for.</figcaption>
</figure>

## Making synthetics honest

Two fixture repairs from the DMO arc are worth restating as method, because
they generalize past TETRA. First: **the encode side must model the
transmitter, not the code under test.** The colour-0 descramble skip
([Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}))
survived a targeted sweep because the synthetic encoder shared the decoder's
conditional; `dmo_decode_test.go` now scrambles unconditionally — real-air
behaviour — and the colour-0 rounds became the failing-first regression.
Second: **synthetic timing must be as structured as real timing.** The noise-grant
bug ([Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }}))
was uncatchable while `buildDMODibitStream` laid bursts at arbitrary filler
offsets; a fixture transmitter that keeps the 255-dibit slot grid is what lets
the `DMSlotGrid` regressions mean anything. The same honesty shows up smaller:
blind CMA is degenerate on a noise-free constant-modulus input, so the daemon's
synthetic TETRA test adds 40 dB AWGN on purpose. A fixture flattered is a bug
deferred — the full pattern catalog is in
[From the Issue Tracker #20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).

## The capture harness as an instrument

The harness layer deserves its manual, because it is how operators and the
project collaborate. Everything keys off environment variables, so the tests
cost CI nothing and cost an operator one shell line:

```go
// cmd/gophertrunk/tetra_dmo_replay_test.go (shape) — the skip guard
path := os.Getenv("GT_TETRA_DMO_IQ")
if path == "" {
    t.Skip("set GT_TETRA_DMO_IQ (cs16 IQ) + GT_TETRA_DMO_RATE to run the DMO replay")
}
```

The knobs, as of this series' close: `GT_TETRA_DMO_IQ` / `GT_TETRA_DMO_RATE`
select the capture; `GT_TETRA_DMO_COLOUR` pins the DM colour (unset ⇒
auto-recovery); `GT_TETRA_DMO_CLEAR=1` asserts the capture is known-clear,
flipping the VERDICT line so a chance floor reads as a decode defect, never
encryption; `GT_TETRA_DMO_SCAN=1` runs `TestTETRADMOColourScan`, the full
64-colour yield map; `GT_TETRA_DMO_LMS=1` A/Bs the per-burst trained equalizer
on DMO, as `GT_TETRA_LMS=1` does for TMO in `TestTETRAMultiSlotReplay`
(compare `traffic_marked_crc_soft` across runs). The reproduction line for the
Part 12 result is one command:
`GT_TETRA_DMO_IQ=<10aug .raw> GT_TETRA_DMO_RATE=144000 GT_TETRA_DMO_CLEAR=1
go test ./cmd/gophertrunk -run TestTETRADMOReplay -v`. Every headline number in
Parts 9–13 — 410→778, 12%→100%, 6→64, 1/269→35/269 — is re-derivable this way
from the operator's own files. That reproducibility *is* the series' method.

## The 15aug capture: reading a refusal

The most recent capture is the best closing exhibit, because its result is a
**correct refusal**. The operator recorded a purpose-built vector — 25 s of
clear, colour-0, *silent* PTT
(`dmo_test_15aug/25sec_ptt_then_off_30sec_cs16_144khz.raw`). Signalling: DSB
SCH/S at ~90% (105/117). Voice: near the chance floor at **every** colour. The
colour scan shows several colours rising modestly at once — 28→140, 57→74,
30→46 of 831 DNBs — rather than one dominating. One radio scrambling with one
keystream *cannot* produce that shape, so `RecoverDMColourCode`'s dominance
gate refuses to pick (140 < 3×74) — which the operator experienced as "the
colour guesser is broken" and which is actually the gate working: a marginal
signal with partial-keystream artifacts, not a recoverable colour. The
receiver's side of the ledger is already maxed — the CMA equalizer lifts DSB
77→105 and TCH 80→140; LMS doesn't move it — and the SYNC PDU's extended-colour
parse matches osmo-tetra-dmo's scrambler-init offsets, so that path isn't the
gap. The conclusions are procedural: a silent PTT is a poor test vector
(DTX/comfort-noise frames), the next capture must be known-colour and
*actually talking* — and above all, **do not bend the descramble to fit a
33%-yield non-dominant colour**. That would be the self-consistent trap,
volume three.

## The open list

What this series leaves open, each with its closing gate:

- **The #1003 on-air A/B.** A clear DMO capture through the *full daemon*
  (`protocol: tetra-dmo`) must land an intelligible recording. Known sharp
  edges for whoever runs it: recordings file under group **0** (no talkgroup
  on the air), colour recovery needs ~20 qualified DNBs (a very short first
  PTT grants before the colour is known — the voice chain re-recovers), and
  the grant lands ~0.5 s into a transmission by design. `dnb_qualified` is the
  traffic number; `dnb_total` is a noise meter.
- **DMO call control.** EN 300 396-3's source/destination/group PDUs are not
  yet decoded — attribution and group filing wait on it.
- **The trained LMS as a production default.** Staged on both TMO and DMO
  soft paths, pinned synthetically (13%→0% and 12%→0% payload error), gated on
  capture A/Bs that so far say "no harm, no clear win."
- **A decisive DMO voice capture.** The 15aug refusal stands until a
  known-colour, talking capture either recovers cleanly or fails informatively.

That is the honest ledger. The arc it closes is real, though: one carrier,
followed from a rotating constellation to a conformant vocoder to recorded
clear voice — TMO hardened end to end, DMO taken from spec-reading to a
production pipeline — with every claim along the way carrying either a test, a
capture, or an explicit "not yet verified." The villain of the series never
changed: the green test that wasn't evidence. The method never changed either:
make it fail first, then make it pass, then make the air agree.

## FAQ

**Can I develop against this stack with no TETRA network in range?**
Yes — that's the lattice's whole point. The bottom three layers need no RF at
all, and the capture harnesses run on files. What you cannot do without air is
*close* anything: on-air verification is a distinct layer, not a formality.

**Why environment variables instead of test flags or fixtures in the repo?**
Captures are large, operator-owned, and often sensitive, so they can't live in
testdata. Env-var skip guards keep the tests permanently wired into `go test`
(no bit-rot) while making CI cost zero and operator activation trivial.

**How do I contribute a capture that actually helps?**
Record cs16 IQ at a known rate, note everything the radio's config declares
(colour, security class, frequency), and prefer a *talking* transmission —
the 15aug lesson is that silent PTT vectors mostly exercise DTX frames. Then
run the relevant harness and file the full log output; the verdict lines are
written to be pasted into issues.

**What would make the colour-scan gate pick wrongly?**
Very little, by construction — it needs 6 CRC-valid bursts *and* 3× dominance,
and wrong colours live at the ~1/256 floor. The realistic failure is the one
observed: a signal too marginal for any colour to dominate, where the gate
refuses. Trusting a non-dominant winner is exactly what the gate exists to
prevent.

**Is TMO TETRA "done"?**
The decode path is verified end to end — conformant vocoder, capture-proven
soft/equalized TCH/S, a control channel that rides through the marginal
regime. Open engineering remains around the edges (LMS default, symbol-scope
pooling), but nothing in the TMO voice path currently waits on evidence.

## Series navigation

**Part 14 of 14** · ←
[Part 13: DMO III — A Production Pipeline: Grid Votes, Grants & Noise]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
· This is the finale — back to the [series index]({{ '/blog/series/tetra-end-to-end/' | relative_url }}).

*Where to next? The equalizers, soft decisions, and metric discipline this series applied to one protocol are a general toolkit — see how they generalize in [Weak-Signal Engineering]({{ '/blog/series/weak-signal-engineering/' | relative_url }}), which owns the theory these TETRA posts kept pointing at.*
