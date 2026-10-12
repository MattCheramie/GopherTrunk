---
title: "From the Issue Tracker, Season 2, Part 6: False Exact Solves — When 24 Redundant Checks Were Fiction"
description: "How the 13 Sep on-air DMO run that finally decoded five clear PTTs also exposed three ways GopherTrunk ate the first seconds of each transmission — a grant that waited for four bursts after the seed had proved traffic, a voice chain with no IQ before the grant, and a sparse-check seed solver whose 2^-24 false-accept claim was fiction."
category: solution-postmortem
keywords: tetra dmo scramble seed, false exact solve, solve_rejects, dmo grant on seed adoption, dmo voice pre-roll, sparse parity checks gf2, tetra dmo first seconds lost, DMSeedTracker ObserveDNB, TestTETRADMOPipelineCaptureReplay, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, tetra, dmo, scrambling, voice-chain, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 6
---

*Part 6 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems that continues the
[first season]({{ '/blog/series/from-the-issue-tracker/' | relative_url }}):
one bug per part, with receipts.
[Part 5]({{ '/blog/solution-postmortem/issue-tracker-s2-05-seed-is-the-source-address/' | relative_url }})
replaced four wrong DMO "colour codes" with an exact GF(2) solve of the
per-transmission scramble seed. This part is what the first live run on
that build showed: DMO decoding — and three smaller defects, each measured
on the same capture, that together cost the head of every PTT.*

> **TL;DR:** On 13 Sep the operator's live
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) run
> decoded five consecutive clear DMO PTTs — seeds `0x0001498b2a`,
> `0x000162eab3`, `0x0001733855`, `0x00011f914e`, `0x0001088472` — with one
> complaint: "we are eating the first seconds of transmission, because of
> CC bruteforcing probably". It was not the solve.
> `TestTETRADMOPipelineCaptureReplay` measured three things: `maybeGrant`
> waited for `dmoGrantMinDNB` = 4 qualified DNBs after an exact CRC-decoded
> solve had proved traffic on the first (0.17–0.28 s offline, 1.0 and 1.8 s
> live); the voice chain saw no IQ before the grant, so `voiceFanout` now
> keeps a measured 1 s pre-roll (`dmoVoicePrerollSeconds`: 0 s → 320
> CRC-valid bursts, 1 s → 361, 3 s → 318); and two "scramble seed changed …
> changed back" flip-flops were **false exact solves** from the
> reliable-check solver — 7 of 1411 DNBs solved to seeds that decode
> nothing, because `tchSparseMinChecks`' "24 redundant checks" overlap so
> heavily that 2^-24 was fiction. `DMSeedTracker.ObserveDNB` now adopts a
> seed-changing solve only if the burst CRC-decodes at it, counted in
> `solve_rejects`.

**Key takeaways**

- **Redundancy is a property of independence, not of a count.** Fifty-four
  checks from overlapping 48-bit windows are not 24 independent extra
  equations; the false-accept rate was measured, not derived.
- **The strongest evidence should fire the grant.** A CRC-decoded exact
  solve outranks four correlator hits, so waiting for the count only delayed
  the recording.
- **A chain that starts at the grant cannot decode what came before it.**
  DMO grants *after* traffic starts; the pre-roll was swept, and longer loses.
- **A verification that costs nothing should always run.** The decode the
  tracker needed anyway is the gate that rejects a false solve.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Seed-changing solve gate | adopt only if the burst CRC-decodes at the solved seed; else `SolveRejects++` | `internal/radio/tetra/dmo_seed_tracker.go` (`DMSeedTracker.ObserveDNB`) |
| The solver's last resort | reliable LOCAL checks, `tchSparseMinChecks` = 30 + 24, windows of `tchSparseWindow` = 48 bits | `internal/radio/tetra/dmo_seed.go` (`solveTCHSeedReliableChecks`) |
| Literal false-solve vectors | two on-air DNBs that "solve" to `0x00028b4304` / `0x0003d69966` | `tetra/testdata/dmo_13sep_false_solve_*.dnb`, `TestDMSeedTrackerRejectsSolveThatDoesNotDecode` |
| Grant on adoption | `maybeGrant(seedAdopted=true)` short-circuits `dmoGrantMinDNB` = 4 | `internal/scanner/ccdecoder/pipelines_dmo.go`, `TestTETRADMOPipelineGrantsOnSeedAdoption` |
| Voice pre-roll | 1 s ring of channelised IQ, DMO only, delivered as the first chunk | `ccdecoder/voicetap.go` (`dmoVoicePrerollSeconds`, `SubscribeVoiceIQWithPreroll`) |
| The instrument | per-PTT timing, lost-before-grant, false-solve audit, pre-roll sweep | `ccdecoder/pipelines_dmo_capture_test.go` (`GT_TETRA_DMO_IQ`) |
## In this post

- **The run that worked, and the complaint that came with it** — five PTTs decoded live.
- **An instrument before a theory** — what the capture replay measures.
- **The grant that waited for four** — the cheapest second anyone ever lost.
- **A chain with no past** — measuring the pre-roll instead of guessing it.
- **The 2^-24 that wasn't** — how a sparse solver manufactured seeds from noise.
- **The gate, the rescue and what is still open** — the failing-first pin.

## The run that worked, and the complaint that came with it

[TETRA End to End Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
built the production DMO pipeline — slot-grid votes, sticky lock,
edge-triggered grants — and Part 5 of this season gave it the one thing it
had always lacked: the real scramble seed, solved exactly from a single
error-free DNB. On 13 Sep the operator ran that build on air and reported
that DMO was finally working: a debug log and a 60 s, 144 kHz FLAC capture
in sync, and the seeds the live `tetra.DMSeedTracker` recovered — one per
PTT, every one under the capture-pinned `DMScrambleSeedPrefix` — matching
the offline replay burst for burst.

The residual complaint: the first seconds of each transmission were
missing, and the operator blamed the seed search. Reasonable — the previous
design *was* a 64-way brute force
([TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}))
— but the exact solve is one 158×31 GF(2) elimination on the first
grid-qualified DNB. Something else was eating the head of every PTT. The
rule from
[Season 1 Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
applies: before theorising, build the thing that counts.

## An instrument before a theory

`TestTETRADMOPipelineCaptureReplay` (`internal/scanner/ccdecoder/`,
skip-guarded on `GT_TETRA_DMO_IQ`) replays a flac or cs16 capture through
the **production** `newTETRADMOPipeline` — the daemon's own receiver,
stream extractor, slot grid, seed tracker and grant logic — in 512-sample
chunks under a sample clock, stamps every burst with the pipeline's
verdicts, and prints per transmission when it locked, learned the seed and
granted, ending in a `SUMMARY:` of CRC-valid bursts lost before the grant. Three counters
do the work. An "ideal" `DMSeedTracker` over the pipeline's whole burst
stream counts CRC-valid TCH/S bursts that precede the grant — speech a chain
fed from the grant onward can never record (`crcBeforeGrant`). A
**false-solve audit** runs `tetra.DMBurstScrambleSeed` over every DNB,
qualified or not, flags any solution that disagrees with the transmission's
seed and dumps it as a gob fixture under `GT_TETRA_DMO_DUMP`. And a phase-2
sweep starts a cold voice chain (`simulateDMOVoiceChain`) at grant −
pre-roll for 0, 0.5, 1, 2 and 3 s and counts what it decodes.
`GT_TETRA_DMO_MAX_LOST_SECONDS` turns the summary into an assertion. Three
findings came out, in increasing order of surprise.

## The grant that waited for four

`maybeGrant` guarded the grant with `dmoGrantMinDNB` = 4 slot-grid-qualified
DNBs after lock — sensible when a qualified DNB was the *only* evidence of
traffic. By 13 Sep the seed tracker ran on every qualified burst, and an
exact solve that CRC-decodes is a 128-check-redundant proof that the burst
is a TCH/S codeword: stronger evidence than four correlator hits. The grant
still waited for the count. Offline, that cost the first three decodable bursts
of every PTT — 0.17–0.28 s. Live it was worse: two weak transmissions lost
1.0 and 1.8 s, because the live grid latched slowly (18 qualified DNBs on
one PTT where the offline replay of the same PTT counted 65).

The fix is one parameter:

```go
// internal/scanner/ccdecoder/pipelines_dmo.go (shape)
func (p *tetraDMOPipeline) learnSeed(b tetra.DMBurst) {
    frames, seed, adopted := p.seeds.ObserveDNB(b, true)
    if adopted {
        p.colour, p.colourKnown = seed, true
        p.log.Info("tetra dmo scramble seed recovered", "seed", fmt.Sprintf("%#010x", seed), …)
        p.maybeGrant(true) // adoption IS traffic: don't wait for dmoGrantMinDNB
    }
}

func (p *tetraDMOPipeline) maybeGrant(seedAdopted bool) {
    if p.grantActive || p.bus == nil || !p.locked { return }
    if !seedAdopted && p.dnbSinceLock < dmoGrantMinDNB { return }
    …
}
```

`TestTETRADMOPipelineGrantsOnSeedAdoption` records the qualified-DNB index
of the seed adoption and of the grant and demands they be equal; against
the old code the seed lands at #1 and the grant at #4.

## A chain with no past

The second finding is structural. Trunked protocols grant *before* their
traffic starts, so a chain opened at the grant misses nothing. DMO has no
control channel; the burst train is the only evidence, so the grant trails
the traffic by construction. Each PTT opens with a DSB-only setup phase of
0.3–1.4 s — protocol, not loss — then 3–8 decodable DNBs before the 6-vote
slot grid latches. A chain started cold at the grant can never decode
those, and loses more while its timing, AFC and equalizer loops acquire.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Timeline of one DMO transmission: a DSB-only setup phase, then DNB traffic with the slot grid latching, the seed solving on the first qualified burst, the old grant three bursts later and the new grant at the seed. A bracket marks the one second pre-roll ending at the grant, and a row gives CRC-valid burst counts for pre-rolls of 0, 0.5, 1, 2 and 3 seconds: 320, 359, 361, 346, 318.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one DMO PTT: where the first seconds went</text>
  <line x1="30" y1="60" x2="650" y2="60" stroke="var(--fg-muted)"/>
  <rect x="30" y="46" width="150" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 2"/>
  <text x="105" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DSB-only setup 0.3–1.4 s (protocol)</text>
  <rect x="180" y="46" width="470" height="28" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="415" y="40" text-anchor="middle" fill="currentColor" font-size="8">DNB traffic (~17 bursts/s)</text>
  <line x1="300" y1="46" x2="300" y2="74" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="300" y="90" text-anchor="middle" fill="var(--accent)" font-size="8">grid latch (6 votes)</text>
  <line x1="318" y1="46" x2="318" y2="74" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="330" y="104" text-anchor="middle" fill="var(--accent)" font-size="8">seed solved + NEW grant</text>
  <line x1="372" y1="46" x2="372" y2="74" stroke="var(--fg-muted)" stroke-width="1.5"/>
  <text x="372" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="8">old grant (+3 bursts)</text>
  <text x="240" y="90" text-anchor="middle" fill="currentColor" font-size="8">3–8 decodable, pre-latch</text>
  <path d="M174 130 L318 130" stroke="var(--accent)" stroke-width="2"/>
  <line x1="174" y1="124" x2="174" y2="136" stroke="var(--accent)"/>
  <line x1="318" y1="124" x2="318" y2="136" stroke="var(--accent)"/>
  <text x="246" y="146" text-anchor="middle" fill="var(--accent)" font-size="8">1 s pre-roll ring → first chunk of the voice chain</text>
  <text x="26" y="186" text-anchor="end" fill="var(--fg-muted)" font-size="8">pre-roll</text>
  <text x="26" y="204" text-anchor="end" fill="var(--fg-muted)" font-size="8">CRC-valid</text>
  <g font-size="9" text-anchor="middle">
    <text x="90" y="186" fill="currentColor">0 s</text><text x="90" y="204" fill="currentColor">320</text>
    <text x="210" y="186" fill="currentColor">0.5 s</text><text x="210" y="204" fill="currentColor">359</text>
    <text x="330" y="186" fill="var(--accent)" font-weight="bold">1 s</text><text x="330" y="204" fill="var(--accent)" font-weight="bold">361</text>
    <text x="450" y="186" fill="currentColor">2 s</text><text x="450" y="204" fill="currentColor">346</text>
    <text x="570" y="186" fill="currentColor">3 s</text><text x="570" y="204" fill="currentColor">318</text>
  </g>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="8">cold receiver + extractor started at grant − pre-roll, summed over five PTTs</text>
</svg>
<figcaption>The grant now lands on the seed, and a 1 s pre-roll reaches back past the grid latch. The sweep is the reason it is 1 s and not 3: a receiver that starts on inter-transmission noise conditions on the noise.</figcaption>
</figure>

`voiceFanout` (`ccdecoder/voicetap.go`) now keeps a ring of the most recent
post-DDC samples, sized by `voicePrerollSamples` — `dmoVoicePrerollSeconds`
= 1.0 for `ProtocolTETRADMO`, zero for anything else — and
`SubscribeVoiceIQWithPreroll` hands a new subscriber the ring as its first
chunk, oldest first, then the live stream. `CCVoiceSource.StreamIQ` uses
it; the auto-record DDC tap keeps plain `SubscribeVoiceIQ`. The ring
records whether or not anyone is subscribed — the subscriber that needs it
does not exist yet.

The length is **measured, not guessed**: over the five PTTs the cold chain
decoded 320 CRC-valid bursts with no pre-roll, 359 at 0.5 s, **361 at
1 s**, 346 at 2 s and 318 at 3 s. Longer loses — a receiver started on
inter-transmission noise lets its blind CMA equalizer and cumulative-mean
normaliser condition on the floor, and one weak PTT fell from 60 bursts to
20 between 0 s and 3 s — which is likely also why the warm live pipeline,
sitting in silence for minutes, decoded that PTT so much worse than the
offline replay. And 1 s is short enough that the previous
transmission, at least `dmoGrantRearm` = 3 s of silence ago, can never be
inside it.

## The 2^-24 that wasn't

The third finding gives this part its title. Two of the five calls logged
`composer: tetra DMO scramble seed changed` and then changed straight back
— once within 56 ms. Mid-transmission, a changed seed means a burst decoded
at the wrong seed and, on a weak stretch, every burst after it until the
next clean solve.

Recall the solver's shape from Part 5. The dense `SolveTCHScrambleSeed`
puts 158 parity checks against 30 unknowns; an error-free burst solves
exactly and an errored one is inconsistent across 128 redundant checks — a
sound claim, because the dense checks are the code's full parity-check
matrix. `DMBurstScrambleSeed` then tries
`SolveTCHScrambleSeedSoft` — flip the `tchSeedSoftSingles` = 12 least
reliable coded bits singly, then pairs among `tchSeedSoftPairs` = 8 — and
as a last resort `solveTCHSeedReliableChecks`: solve from only the burst's
most *reliable* LOCAL checks, each on a `tchSparseWindow` = 48-bit window
of the convolutional code's short memory, so a bit error spoils only the
windows containing it. It accepts once `tchSparseMinChecks` = 30 + 24
reliable checks agree, widening in steps of 8 while consistent.

The comment in the tree now says what the design assumed and why it was
wrong:

```go
// internal/radio/tetra/dmo_seed.go (shape)
// tchSparseMinChecks is how many of the most reliable sparse checks must agree
// on a seed for the reliable-check solve to accept it: 30 to determine the seed
// plus 24 nominally redundant ones. The redundancy is NOMINAL: the sparse
// checks overlap heavily (neighbouring ~48-bit windows) and are far from
// independent, so a burst that is not a TCH/S codeword passes far more often
// than 2^-24 — on the 13 Sep #1003 capture the reliable-check path "solved"
// 7 of 1411 DNBs (noise and errored real bursts) to seeds that decode nothing
const tchSparseMinChecks = tchSeedBits + 24
```

Windows 48 bits wide and `tchSparseStep` = 12 bits apart share three
quarters of their support with their neighbours; twenty-four such checks
are a handful of coin flips restated. The false-solve audit counted the
real rate: **7 of 1411 DNBs** — six
correlator false alarms off the idle channel and one real, mildly errored
burst of the third transmission — solved to seeds at which they decode
nothing. That is not 2^-24; it is 5×10^-3 — at ~17 bursts per second, a
flip-flop every few seconds on a weak stretch. It is the
[Season 1 Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
trap in a new dress: a correctness claim derived from the model's own
assumptions, never measured against air.

## The gate, the rescue and what is still open

The fix costs nothing, which is the best kind. A solve that would **change**
the seed must first decode this very burst CRC-valid at the seed it claims;
the dense solve always does, and the tracker needed that decode anyway:

```go
// internal/radio/tetra/dmo_seed_tracker.go (shape) — ObserveDNB
if s, ok := DMBurstScrambleSeed(b); ok && !t.pinned {
    if t.known && s == t.seed {
        t.Solved++
        return dmDecodeDNB(b, s), s, false
    }
    if frames = dmDecodeDNB(b, s); frames != nil {   // the decode it needed anyway
        t.Solved++
        t.seed, t.known, t.verified = s, true, true
        t.ExactAdopts++
        return frames, t.seed, true
    }
    t.SolveRejects++   // fall through: decode at the known seed, or let the hint earn adoption
}
```

A rejected solve is counted as `solve_rejects` in the status and ended
lines ([Field Notebook Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }}))
and the burst decodes at the known seed instead — a rescue, not just a
non-flip: the real errored burst of the third PTT yields **two speech
frames at the true seed** once the bogus solve stops displacing it.

The regression is pinned against **literal capture bursts**.
`GT_TETRA_DMO_DUMP` wrote the seven as fixtures; two are committed as `tetra/testdata/dmo_13sep_false_solve_0x00028b43.dnb`
(the qualified real burst) and `…_0x0003d699.dnb` (a correlator false
alarm) — rotation, hard dibits and receiver differentials exactly as the
extractor emitted them.
`TestDMSeedTrackerRejectsSolveThatDoesNotDecode` asserts that the solver
*still* returns the bogus seed (the fixture still exercises the path), that
neither burst decodes at it, that a tracker holding the verified
`0x0001733855` keeps it with `SolveRejects` = 1 — decoding 2 frames from the
real burst, 0 from the noise — and that a fresh tracker adopts nothing. The
old tracker adopts both.

Operator-side, the config had `gain: 50` — 5 dB, as the startup WARN says
— with the CC at −70 dBFS; `gain: 500` is the first thing to try. What is
**not** verified: grant-on-adoption and the pre-roll in the live daemon.
Both were validated on the capture only; per #764/#771 the next live run
decides. The same day's `siglab: capture started/ended/aborted` INFO lines
([Field Notebook Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }}))
landed too.

## Where this goes next

The next two parts leave DSP for the capture tooling. The 15 Sep IPSC
field material arrived as a 60 s "442.8125 MHz" grab in which neither
repeater existed, because the SigLab form had coerced the centre with
`Number()` and fallen back silently.
[Part 7]({{ '/blog/solution-postmortem/issue-tracker-s2-07-capture-at-the-wrong-centre/' | relative_url }})
follows that NaN from the text field to the tuner centre.

## FAQ

**What is a false exact solve in GopherTrunk's TETRA DMO decoder?**
A seed returned by `tetra.DMBurstScrambleSeed`'s reliable-check path
(`solveTCHSeedReliableChecks`) at which the burst does not CRC-decode. The
dense 158-equation solve cannot produce one; the sparse local checks can,
because their 48-bit windows overlap. On the 13 Sep capture 7 of 1411 DNBs
solved falsely.

**What does solve_rejects count in the DMO status line?**
Seed-changing solves that `DMSeedTracker.ObserveDNB` discarded because the
burst failed to decode at the solved seed; the burst then decodes at the
known seed. Zero to one per PTT is normal; a climbing count marks a weak
stretch. `TestDMSeedTrackerRejectsSolveThatDoesNotDecode` pins the gate on
two literal on-air bursts.

**Why does the DMO grant now fire before four qualified DNBs?**
Because `maybeGrant(seedAdopted=true)` treats a seed adoption — a
CRC-decoded exact solve on a grid-qualified burst, or three CRC-valid
decodes at the DSB's announced seed — as stronger traffic evidence than
`dmoGrantMinDNB` = 4 correlator hits. Waiting cost 0.17–0.28 s offline and
1.0–1.8 s live. `TestTETRADMOPipelineGrantsOnSeedAdoption` pins it.

**Is the first-seconds fix verified on air?**
Not yet. The seed solve is on-air verified (five live PTTs matching the
offline replay burst for burst); grant-on-adoption and the 1 s pre-roll
were validated on that capture only. The operator's next live run on a
build with them is the gate, per the #764/#771 rule.

## Series navigation

**Part 6 of 14** · ←
[Part 5: The Seed Is the Source Address — Solving TETRA DMO's Scramble Seed in GF(2)]({{ '/blog/solution-postmortem/issue-tracker-s2-05-seed-is-the-source-address/' | relative_url }})
· Next →
[Part 7: The Capture Carved at the Wrong Centre — A NaN That Fell Back Silently]({{ '/blog/solution-postmortem/issue-tracker-s2-07-capture-at-the-wrong-centre/' | relative_url }})
