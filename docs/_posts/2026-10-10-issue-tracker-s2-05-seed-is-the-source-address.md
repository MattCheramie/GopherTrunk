---
title: "From the Issue Tracker, Season 2, Part 5: The Seed Is the Source Address — Solving TETRA DMO's Scramble Seed in GF(2)"
description: "Why four 'colour codes' recovered from four TETRA DMO captures — 3, 39, 36, 31 — were all partial-keystream artefacts, and how an exact GF(2) solve of the 30-bit scramble seed from a single traffic burst (158 parity equations, 30 unknowns, 128 redundant checks) showed the seed changes on every PTT because it is the transmitting radio's source address under a fixed prefix."
category: solution-postmortem
keywords: tetra dmo scramble seed, tetra extended colour code 30 bit, solve lfsr seed linear algebra gf(2), tetra tch/s parity check matrix, dmo colour code per transmission, tetra dmo source address seed, dm-sync sch/h source address, recoverdmcolourcode artifact, dmseedtracker, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, tetra, dmo, scrambling, linear-algebra, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 5
---

*Part 5 of **From the Issue Tracker, Season 2**, postmortems of GopherTrunk
bugs told with receipts.
[Part 4]({{ '/blog/solution-postmortem/issue-tracker-s2-04-invented-dcs-codeword/' | relative_url }})
closed the analog-scanner quartet with a DCS word pinned by parity
equations. This part returns to the longest-running thread in the tracker,
TETRA direct mode
([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)), at the
moment its central question — *which colour code scrambles the voice?* —
turned out to be the wrong question. Every answer it had produced was an
artefact of asking it, and the instrument that showed this was not a
better search but linear algebra.*

> **TL;DR:** TETRA scrambles TCH/S with a 32-stage LFSR seeded from a
> **30-bit extended colour code**, and across four DMO captures GopherTrunk's
> `RecoverDMColourCode` — a 64-way brute force scored by CRC-valid speech —
> had "recovered" colour 3, then 39, then 36, then 31. All four were wrong:
> a related wrong seed CRC-passes ~9 % of bursts, so a CRC count is not
> proof of a seed. `tetra.SolveTCHScrambleSeed` (`internal/radio/tetra/dmo_seed.go`)
> replaces the search with an exact solve: the scrambler is **affine in its
> seed** over GF(2) and the TCH/S coding (CRC parity matrix, K=5
> convolutional code, puncturing, 24×18 interleave) is a **linear block
> code**, so `H'·(r ⊕ PN(0)) = (H'·P)·seed` — 158 equations, 30 unknowns,
> 128 redundant checks, one error-free DNB. `TestTETRADMOSeedScan` on the
> 12 September three-PTT capture gave three seeds — `0x012915c0`,
> `0x015d9c07`, `0x01671384` — one per transmission, each equal to the
> 24-bit DSB SCH/H field at bits 42..65 (the **source address**, right
> before the MNI at 66..89 decoding as the operator's MCC 250 / MNC 1)
> under prefix `000001`. `DMSeedTracker` now runs in the pipeline and the
> voice chain; the capture went from 58 CRC-valid bursts of one PTT to 159
> across all three and 9.5 s of speech. On-air verified 13 September.

**Key takeaways**

- **A CRC count can confirm a wrong seed.** Related seeds share keystream
  structure; a "close" wrong seed CRC-passed ~9 % of real bursts, enough to
  win a brute force and look like evidence.
- **When the unknown is a seed and the code is linear, solve — do not
  search.** The affine scrambler and the linear block code make the seed a
  system of equations, 128 checks over-determined, with no colour space, no
  field layout and no configuration.
- **The thing was not a network constant.** The seed changes every PTT
  because it is the transmitting radio's address; no `tetra_mcc`/`tetra_mnc`
  fold could ever have described it.
- **An exact per-burst answer makes the field layout a hint, not an
  authority.** The DSB announces the seed; the traffic proves it.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Exact solve | 158×31 GF(2) elimination per DNB; `ok` only for an error-free TCH/S codeword | `internal/radio/tetra/dmo_seed.go` (`SolveTCHScrambleSeed`, `solveSeedRows`) |
| Linear structure | PN(0), 30 seed impulse responses, parity checks moved to type-5 order | `buildTCHSeedLinear`, `tchSeedLinear` (built once, lazily) |
| Vote + CRC cross-check | ≥ `dmSeedMinVotes` (2) agreeing solves, CRC-decodes ≥ votes | `RecoverDMScrambleSeed` |
| Per-transmission tracking | exact solve adopts at once; SCH/H hint after 3 CRC-valid decodes | `dmo_seed_tracker.go` (`DMSeedTracker`, `dmSeedHintMinCRC`) |
| Seed hint from the DSB | `DMScrambleSeedPrefix` (1) ‖ SCH/H bits 42..65 | `DMSyncSCHHScrambleSeed`, `DMSCHHSeedFieldOffset` |
| Linearity pins | 158 checks annihilate codewords; PN affine; H'·P rank 30 | `dmo_seed_test.go` (`TestTCHSeedLinearStructure`) |
| Capture instrument | per-burst seed next to each DSB's SYNC PDU and raw SCH/H | `cmd/gophertrunk` (`TestTETRADMOSeedScan`, `GT_TETRA_DMO_SEEDS=1`) |
| Pin one seed | diagnostics only; a full 30-bit value | `tetra_colour_code` (config.example.yaml `tetra-dmo` example) |

## In this post

- **Four colours, four captures** — what the brute force reported and why each was believed.
- **Why a CRC count is not proof** — the 9 % problem.
- **The seed is a system of equations** — affine scrambler, linear code, 128 redundant checks.
- **Three PTTs, three seeds** — the scan, and the field that matched.
- **From solver to tracker** — precedence rules the pipeline and voice chain share.
- **Verified, and what Part 6 takes up** — 159 bursts, the live run, the false solves.

## Four colours, four captures

The DMO story up to this point is told in
[TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}):
the "encrypted" verdict that was a colour-0 descramble skip, and the second
capture where voice descrambled at colour **3** while the signalling
advertised 0 — `tch_crc` 1/269 → 35/269 — which led to
`RecoverDMColourCode`, a brute force over the 64 colours that keeps the one
maximising CRC-valid TCH/S behind a dominance gate (best ≥ 3× runner-up).
That design was explicitly chosen to avoid hardcoding a bit offset one
capture could not pin — the self-consistent trap, Season 1's
[Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
case seven.

Then the colours kept changing. The 20 August run's control pipeline
recovered **39** while its voice chain, falling back to colour 0, decoded
nothing. The 15 August silent-PTT capture produced *no* dominant colour —
several rose modestly at once (28→140, 57→74, 30→46 of 831 DNBs), which the
gate correctly refused; that was read first as a marginal signal, then as
an MNI blind spot (the radios run MCC 250 / MNC 1 and the search only ever
tried MNI 0), so `tetra_mcc`/`tetra_mnc` were added to fold the configured
MNI into every candidate. And on the 12 September three-PTT capture the
brute force reported colour **36** with 58 CRC-valid bursts — for one PTT
only. Four captures, four answers, each with evidence. The reporter was
owed a better instrument.

## Why a CRC count is not proof

The brute force's score was the number of DNBs whose TCH/S decoded with a
valid class-2 CRC under a candidate seed. The class-2 CRC is 8 bits, so a
random seed passes ~1/256 of bursts — the "chance floor" every DMO status
line is read against. But seeds are not random with respect to each other.
The scrambling sequence is the output of one LFSR, and two seeds that
differ in a few bits produce keystreams that agree over long stretches;
**measured on the 12 September capture, a related wrong seed CRC-passes
~9 % of bursts.** Against 269 or 831 DNBs that is dozens of "valid" decodes
— enough to top a 64-way search, enough to look like a dominant winner at
3× the runner-up, and enough to synthesise a few seconds of garbled speech
that a listener might accept as marginal audio. Every colour the brute
force ever reported was a seed *related* to the true one in exactly this
way: a partial-keystream artefact. The `tetra_mcc`/`tetra_mnc` A/B
confirmed it from the other side — at MNI 250/1 the search decoded nothing
at all.

## The seed is a system of equations

The way out is to stop scoring candidates and solve for the seed directly.
Two properties of code GopherTrunk already shipped make that exact, and
`dmo_seed.go`'s header states them:

- The scrambler (`framing.NewScramblerTetra`) is a 32-stage LFSR initialised
  to `(e(1..30), 1, 1)` (EN 300 392-2 §8.2.5.2). An LFSR's output is linear
  in its state, so the 432-bit PN sequence is **affine in the seed**:
  `PN(seed) = PN(0) ⊕ Σᵢ seedᵢ · (PN(1<<i) ⊕ PN(0))`.
- The TCH/S channel coding (EN 300 395-2 §5.5) is a **linear block code**
  once the four tail bits are fixed: 112 class-1 and 60 class-2 bits → an
  8-bit CRC that is a fixed parity-check matrix (`tchCRCTaps`) → the K=5
  convolutional mother code → puncturing to 168 + 162 = 330 coded bits.
  Whatever the speech, the coded bits `c` satisfy `H·c = 0`. The 102 class-0
  bits are uncoded and carry no constraint. The 24×18 interleave is a
  permutation.

So for an error-free received type-5 block `r = interleave(c) ⊕ PN(seed)`:

```text
H' · (r ⊕ PN(0)) = (H' · P) · seed
```

with `H'` the parity checks moved to type-5 coordinates and `P` the 432×30
matrix of seed impulse responses. That is **158 linear equations in 30
unknowns** — 330 coded bits minus 172 information bits — and one
error-free burst pins the seed with 128 redundant checks. A burst with bit
errors is almost surely inconsistent and rejected. No candidate colours, no
assumption about how MCC, MNC and colour compose the seed, no knowledge of
the speech.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A pipeline from a received 432-bit type-5 block through XOR with the PN sequence of seed zero, then de-interleave, then 158 parity checks, producing a 158-bit syndrome. Beside it the seed impulse responses form a 158 by 30 matrix H prime P; the syndrome equals that matrix times the 30-bit seed. Gaussian elimination over GF(2) yields the seed if the 128 redundant equations are consistent, and rejects the burst otherwise.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one DNB → one seed: H'·(r ⊕ PN(0)) = (H'·P)·seed</text>
  <rect x="20" y="40" width="110" height="34" fill="none" stroke="currentColor"/>
  <text x="75" y="54" text-anchor="middle" fill="currentColor" font-size="8">received r</text>
  <text x="75" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">432 type-5 bits</text>
  <line x1="130" y1="57" x2="160" y2="57" stroke="currentColor"/>
  <rect x="160" y="40" width="90" height="34" fill="none" stroke="currentColor"/>
  <text x="205" y="54" text-anchor="middle" fill="currentColor" font-size="8">⊕ PN(0)</text>
  <text x="205" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pn0, packed</text>
  <line x1="250" y1="57" x2="280" y2="57" stroke="currentColor"/>
  <rect x="280" y="40" width="120" height="34" fill="none" stroke="currentColor"/>
  <text x="340" y="54" text-anchor="middle" fill="currentColor" font-size="8">158 parity checks H'</text>
  <text x="340" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CRC · K=5 conv · puncture</text>
  <line x1="400" y1="57" x2="430" y2="57" stroke="currentColor"/>
  <rect x="430" y="40" width="90" height="34" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="475" y="54" text-anchor="middle" fill="var(--accent)" font-size="8">syndrome y</text>
  <text x="475" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">158 bits</text>
  <rect x="280" y="104" width="120" height="34" fill="none" stroke="currentColor"/>
  <text x="340" y="118" text-anchor="middle" fill="currentColor" font-size="8">H'·P  (158 × 30)</text>
  <text x="340" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">checks on 30 seed impulse responses</text>
  <line x1="400" y1="121" x2="430" y2="121" stroke="currentColor"/>
  <rect x="430" y="104" width="90" height="34" fill="none" stroke="currentColor"/>
  <text x="475" y="118" text-anchor="middle" fill="currentColor" font-size="8">· seed</text>
  <text x="475" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">30 unknowns</text>
  <text x="475" y="92" text-anchor="middle" fill="currentColor" font-size="9">=</text>
  <line x1="520" y1="57" x2="560" y2="57" stroke="currentColor"/>
  <line x1="520" y1="121" x2="560" y2="121" stroke="currentColor"/>
  <line x1="560" y1="57" x2="560" y2="121" stroke="currentColor"/>
  <line x1="560" y1="89" x2="580" y2="89" stroke="currentColor"/>
  <rect x="580" y="72" width="90" height="34" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="625" y="86" text-anchor="middle" fill="var(--accent)" font-size="8">GF(2) eliminate</text>
  <text x="625" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="8">158×31 rows</text>
  <text x="625" y="130" text-anchor="middle" fill="currentColor" font-size="8">consistent → seed</text>
  <text x="625" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="8">128 redundant checks</text>
  <text x="625" y="160" text-anchor="middle" fill="currentColor" font-size="8">inconsistent → reject</text>
  <text x="625" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bit errors, not TCH/S</text>
  <text x="200" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no colour space · no field layout · no MCC/MNC · no knowledge of the speech</text>
</svg>
<figcaption>The scrambler is affine in its seed and the channel code is linear, so the seed falls out of one burst's parity checks by Gaussian elimination — over-determined by 128 equations, which is what makes a solution proof rather than a score.</figcaption>
</figure>

`buildTCHSeedLinear` constructs all of this once: PN(0) and the thirty
impulse responses from `NewScramblerTetra(1<<i)`; the generator matrix by
encoding each of the 172 information unit vectors through the real
`EncodeTCHS` chain (`tchCodedBitsFor`); the parity checks as that
generator's null space, each moved to its type-5 position through
`tchType5IndexOfCoded`; and every check's action on the seed columns and
on PN(0). `SolveTCHScrambleSeed` forms 158 augmented rows — coefficients in
the low 30 bits, the right-hand side in bit 30 — and `solveSeedRows`
eliminates:

```go
// internal/radio/tetra/dmo_seed.go
func SolveTCHScrambleSeed(type5 []byte) (seed uint32, ok bool) {
    lin := tchSeedLinearInstance()
    r := packBits432(type5[:tchType3Bits])
    rows := make([]uint32, len(lin.checks))
    for i := range lin.checks {
        y := parityWords(&lin.checks[i], &r) ^ lin.b0[i]
        rows[i] = lin.a[i] | uint32(y)<<tchSeedBits
    }
    return solveSeedRows(rows)
}
```

`ok` is false when the system is inconsistent *or* fails to determine every
seed bit. `TestTCHSeedLinearStructure` pins the two linearity claims
numerically rather than by argument: exactly 158 checks, each annihilating
random unscrambled codewords; `PN(s1⊕s2) = PN(s1) ⊕ PN(s2) ⊕ PN(0)` for
random seed pairs; and `H'·P` of full column rank 30, so every seed bit is
observable from one burst. `TestSolveTCHScrambleSeedRoundTrip` scrambles
200 random speech frames with arbitrary 30-bit seeds — including the
structured shapes an operator meets, bare colours and
`ExtendedColourCode(250, 1, c)` — and solves each exactly; one flipped
coded bit is rejected, and a flipped class-0 bit does not disturb the
solve.

## Three PTTs, three seeds

`TestTETRADMOSeedScan` (`GT_TETRA_DMO_IQ`, `GT_TETRA_DMO_SEEDS=1`) runs the
solver burst by burst over a capture and prints every CRC-valid DSB's SYNC
PDU and raw SCH/H bits alongside, so each transmission's seed can be lined
up with the fields that announce it. On the 12 September capture — three
consecutive clear TEA0 PTTs on 438.9 MHz from a Motorola MTP8500Ex — it
printed three seeds, one per PTT:

```text
0x012915c0   0x015d9c07   0x01671384
```

Not a network constant. Not reachable by any colour search, with or without
an MNI fold. And each seed's low 24 bits sit **verbatim at DSB SCH/H bits
42..65**, the 24-bit field immediately before the one at 66..89 — which
decodes as MCC 250 / MNC 1, the operator's codeplug. That is the standard
DMAC-SYNC layout — *… source address type, source address, MNI, message
type …* — so the field is the transmitting MS's **source address**, and
the seed's top six bits were `000001` on all three. `DMScrambleSeedPrefix`
= 1 and `DMSCHHSeedFieldOffset` = 42 are therefore **capture-pinned, not
spec-quoted**: neither reference decoder (osmo-tetra-dmo, TetraDMO-Receiver)
carries any DMO traffic-seed rule at all, so there is no reference to
quote, and the constants are used only as a hint that traffic must
confirm. The exact per-burst solve is the authority.

## From solver to tracker

Production needed the solve to run live, in both DMO decode paths, under
one rule. `DMSeedTracker` (`dmo_seed_tracker.go`) is that rule, and the
same instance type runs in the control pipeline and the composer's voice
chain. Its precedence:

- **An exact solve adopts immediately** and preempts a previously adopted
  seed — that is how a new PTT with a new seed takes over. `ObserveDNB`
  solves *before* decoding at the known seed, because a stale seed from the
  previous PTT can still CRC-pass an occasional burst of the next one (the
  ~9 % problem again) and would emit garbage speech while delaying the
  switch.
- **The SCH/H hint** (`ObserveDSB` → `DMSyncSCHHScrambleSeed`) or an external
  hint is adopted only after `dmSeedHintMinCRC` = 3 CRC-valid decodes within
  a `dmSeedHintWindow` of 12 slot-grid-qualified bursts — two was measurably
  too few; a synthetic 90-burst random payload confirmed a wrong hint — and
  only while no exact solve has spoken for the transmission.
- **A traffic drought resets** the seed and hint; a `Pin` from
  `tetra_colour_code` survives everything.

`RecoverDMScrambleSeed` keeps the offline vote: the winner needs
`dmSeedMinVotes` = 2 agreeing solves *and* at least as many CRC-valid
decodes as votes — the cross-check that the linear model matches the
decoder (`TestRecoverDMScrambleSeed`: twelve bursts at
`ExtendedColourCode(250, 13, 39)` interleaved with twelve of noise). `tetra_mcc`
and `tetra_mnc` are no longer needed for DMO; `RecoverDMColourCode`
survives only behind the diagnostic colour scan.

## Verified, and what Part 6 takes up

On the capture: **159 CRC-valid TCH/S across all three PTTs**, speech in
seconds 2–5, 13–16 and 24–27, 9.5 s of PCM — against 58 bursts of one PTT
at a bogus colour before. `TestTETRADMOReplay` now reads wav/flac and
prints per-transmission seed runs (`seed … (source field …) … dnbs=…
tch_crc=…`), with `GT_TETRA_DMO_COLOUR` pinning one full 30-bit seed for
diagnostics. On air: the 13 September run — the operator's "DMO is finally
working" — decoded five consecutive clear PTTs live, and the seeds the
pipeline recovered (`0x0001498b2a`, `0x000162eab3`, `0x0001733855`,
`0x00011f914e`, `0x0001088472`, all under the `000001` prefix) match the
offline replay burst for burst.
[The Field Notebook Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
reads the status-line fields this produced — `seed`, `seed_known`,
`seed_verified`, `bursts_solved`, `solve_rejects`.

What the live run also showed is the subject of the next part. The dense
solve is a proof; the *soft-assisted* fallback for errored bursts is not.
`DMBurstScrambleSeed` flips the least reliable coded bits singly and in
pairs (`tchSeedSoftSingles` = 12, `tchSeedSoftPairs` = 8) and then falls
back to a solve over local ~48-bit sparse parity checks
(`solveTCHSeedReliableChecks`, `tchSparseMinChecks` = 30 + 24). Those
sparse checks overlap so heavily that "24 redundant checks" was nominal,
and on the 13 September capture that path "solved" 7 of 1411 DNBs to seeds
that decode nothing — the "scramble seed changed … changed back" flip-flops
in the operator's log.

## Where this goes next

[Part 6]({{ '/blog/solution-postmortem/issue-tracker-s2-06-false-exact-solves/' | relative_url }})
takes the three residual defects of the 13 September run one at a time:
the false exact solves and the decode-at-the-solved-seed gate that rejects
them (`TestDMSeedTrackerRejectsSolveThatDoesNotDecode`, pinned against two
literal capture bursts), a grant that waited for four qualified DNBs when
the seed had already proved traffic on the first, and the one-second
DMO pre-roll whose length was measured rather than guessed.

## FAQ

**Why did GopherTrunk keep recovering different TETRA DMO colour codes?**
Because there was no colour code to recover. The TCH/S scramble seed is per
transmission — the transmitting radio's 24-bit source address under a fixed
6-bit prefix — and the 64-way brute force in `RecoverDMColourCode` could
only ever land on a related wrong seed that CRC-passes ~9 % of bursts.
Colours 3, 39, 36 and 31 were all partial-keystream artefacts.

**How does SolveTCHScrambleSeed recover a 30-bit seed from one burst?**
The LFSR scrambler is affine in its seed and the TCH/S coding is a linear
block code, so an error-free type-5 block satisfies
`H'·(r ⊕ PN(0)) = (H'·P)·seed`: 158 parity equations in 30 unknowns.
`solveSeedRows` eliminates over GF(2); a consistent system gives the seed
exactly, an inconsistent one (bit errors) is rejected.

**Is a CRC count enough to confirm a scramble seed?**
No. The class-2 CRC is 8 bits, and seeds that differ in a few bits share
keystream structure — a related wrong seed CRC-passed ~9 % of real bursts
on the 12 September capture, enough to win a brute force. Only an exact
solve is proof; `RecoverDMScrambleSeed` uses CRC decodes as a cross-check
on a solved seed, never as the search metric.

**Where does the seed come from in the DM-SYNC?**
The seed's low 24 bits equal the DSB SCH/H field at bits 42..65 — the
source address, immediately before the MNI at 66..89 — under prefix
`000001`. `DMSyncSCHHScrambleSeed` derives that hint; `DMSeedTracker`
adopts it only after three CRC-valid decodes, because the constants are
capture-pinned and no reference decoder quotes a DMO traffic-seed rule.

**Do I still need tetra_mcc and tetra_mnc for DMO?**
No. They were added for an MNI-fold theory the 12 September capture
refuted (at MNI 250/1 the search decoded nothing). The solver needs no
configuration; `tetra_colour_code` can pin one full 30-bit seed for
diagnostics only.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The Invented DCS Codeword — Pinned by Parity Equations, Not the Encoder]({{ '/blog/solution-postmortem/issue-tracker-s2-04-invented-dcs-codeword/' | relative_url }})
· Next →
[Part 6: False Exact Solves — When 24 Redundant Checks Were Fiction]({{ '/blog/solution-postmortem/issue-tracker-s2-06-false-exact-solves/' | relative_url }})
