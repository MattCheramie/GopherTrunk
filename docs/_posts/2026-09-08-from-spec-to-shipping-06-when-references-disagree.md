---
title: "From Spec to Shipping, Part 6: When References Disagree, the Capture Referees"
description: Two trusted decoders gave two different DMO burst geometries, and a single capture could not pin the colour-code bit offset at all. How GopherTrunk designs referee measurements where the right answer wins by a wide margin — and refuses to answer when it doesn't.
category: deep-dives
keywords: tetra dmo burst geometry, dnb block offsets, osmo-tetra-dmo comparison, dm colour code recovery, crc yield sweep, when reference implementations disagree, empirical protocol validation, dominance gate decoder, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, tetra, dmo, methodology, captures, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 6
---

*Part 6 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 5]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
drew the clean-room wall: implement from the spec, run other decoders as
oracles, let facts cross with a citation. But an oracle is only useful
while it's right — and this part is about the days it isn't. When two
references you trust disagree, or when neither can answer at all,
authority stops working and only one referee is left: a measurement on a
real capture, designed so the correct answer cannot lose narrowly.*

> **TL;DR:** GopherTrunk's TETRA DMO normal burst slices its blocks at
> **−108/+11 dibits** around the training-sequence lead
> (`dmDNBBKN1Start`/`dmDNBBKN2Start`, `internal/radio/tetra/dmo.go`) —
> derived from EN 300 396-2 Tables 15/16 — while osmo-tetra-dmo carries
> the TMO geometry, **−115/+19** (the very numbers in GT's own TMO
> `traffic.go`). The tie-breaker was never authority: on a real 438.9 MHz
> capture, TCH/S CRC yield shows a **sharp optimum** at −108/+11 and is
> measurably worse at −115/+19. The DM colour-code bit offset could *not*
> be pinned the same way (colour 3 lights only two LSBs — ambiguous), so
> `RecoverDMColourCode` recovers it empirically instead — maximize
> CRC-valid TCH/S over all 64 colours, behind a dominance gate
> (best ≥ 6 and ≥ 3× the runner-up) that **refused** to pick on the one
> capture where no colour dominated. That refusal was correct: the real
> cause was a non-zero network MNI
> ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)).

**Key takeaways**

- **Authority doesn't referee; captures do.** Between a spec table you
  derived and a proven implementation's different numbers, the decidable
  question is which geometry decodes more CRC-valid speech from real air.
- **Design the measurement so the right answer wins big.** CRC yield over
  a parameter sweep gives the true value a structural advantage — the
  correct colour descrambles class-2-protected speech while every wrong
  one sits at a ~1/256 chance floor. Wide margins survive noise;
  narrow ones are noise.
- **Refusing to answer is an answer.** The dominance gate that "failed"
  on the 15 Aug capture — several colours rising modestly, none 3× ahead —
  was reporting a physically impossible pattern, and the impossibility
  was the diagnostic clue.
- **A single capture can under-determine a field.** Colour 3 exercises
  two bits of a six-bit field; hardcoding an offset from it is the
  self-consistent trap wearing a lab coat. When the data can't pin the
  layout, recover the *value* empirically and leave the layout unpinned.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| DMO DNB geometry | BKN1 at L−108, BKN2 at L+11 (from EN 300 396-2 Tables 15/16) | `internal/radio/tetra/dmo.go` (`dmDNBBKN1Start`, `dmDNBBKN2Start`) |
| TMO NDB geometry | BKN1 at L−115, BKN2 at L+19 — what osmo-tetra-dmo reuses for DMO | `internal/radio/tetra/traffic.go` (`ndbBKN1Start`) |
| Colour recovery | picks the scramble seed maximizing CRC-valid TCH/S, 64 candidates | `internal/radio/tetra/dmo_decode.go` (`RecoverDMColourCode`) |
| Confidence gate | best ≥ 6 CRC-valid AND ≥ 3× runner-up, else "not confident" | `dmo_decode.go` (`dmColourMinCRC`, `dmColourDominance`) |
| Synthetic pin | colour-3 stream + noise distractors; all-noise must refuse | `dmo_decode_test.go` (`TestRecoverDMColourCode`) |
| Field diagnostic | per-burst colour map when the model itself is in doubt | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`TestTETRADMOColourScan`, `GT_TETRA_DMO_SCAN=1`) |

## In this post

- **The disagreement** — one burst, two geometries, both from sources we
  trust.
- **A referee with a sharp peak** — how CRC yield settled −108/+11.
- **The offset no capture could pin** — the colour field and the trap of
  hardcoding it.
- **Recovering the value instead of the layout** — `RecoverDMColourCode`
  and its dominance gate.
- **When the gate says no** — the 15 Aug refusal, and why it was the
  most informative output of the whole investigation.

## The disagreement

TETRA Direct Mode reuses the trunked-mode physical layer wholesale —
same π/4-DQPSK, same 18000 sym/s, same 255-symbol timeslots, same
training sequences — but its Direct Mode Normal Burst (DNB) lays its two
216-bit payload blocks out differently around the training sequence.
GopherTrunk derived the geometry from EN 300 396-2 Tables 15 and 16 by
halving the on-air bit numbers into dibit offsets:

```go
// internal/radio/tetra/dmo.go (shape)
// DNB (Table 15): BKN1 bits 15..230  → dibits 7..114 (108 dibits)
//                 norm train 231..252 → dibits 115..125 (lead L=115)
//                 BKN2 bits 253..468  → dibits 126..233 (108 dibits)
const (
    dmDNBBKN1Start = -108 // L-108 .. L
    dmDNBBKN2Start = 11   // L+11  .. L+119
)
```

osmo-tetra-dmo — a reference this project trusts enough to have
cross-checked its scrambler seed byte-for-byte
([Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }}))
— slices the same burst at the TMO offsets: BKN1 at **L−115**, BKN2 at
**L+19**. Those are not strange numbers; they're the *trunked-mode*
normal-burst geometry, the exact constants in GopherTrunk's own
`traffic.go` (`ndbBKN1Start = -115`). One reading says the spec tables
give DMO its own layout; the other says DMO inherited TMO's. Both come
from serious sources. Reading the PDF harder does not break the tie —
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})
already catalogued how easily two careful readers extract two different
offset sets from the same table.

## A referee with a sharp peak

What breaks the tie is that the two geometries are not equally good at a
job we can score. Slice a real DNB at the right offsets and the block
boundaries line up with the channel coding: descramble, deinterleave,
Viterbi, and the TCH/S class-2 CRC passes. Slice it seven dibits off and
every codeword straddles a boundary — the CRC yield collapses toward the
chance floor. So the referee is a sweep: decode the operator's real
438.9 MHz DMO capture at candidate geometries and count CRC-valid
speech frames.

The result was not a nudge. **−108/+11 sits at a sharp optimum; the
TMO-copied −115/+19 is measurably worse** — and sharpness is the point.
A CRC is a ~1-in-256-per-frame accident under a wrong hypothesis, so a
correct geometry doesn't beat a wrong one by a whisker; it towers over
it. That structural advantage is what makes the measurement a referee
rather than a coin flip, and it's the same yield-is-the-only-verdict
rule the
[Weak-Signal Engineering series]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
learned from equalizers: a metric that can only be flattered by being
*right* beats any amount of authority.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Two CRC-yield sweep curves side by side. Left: yield versus DNB block offset shows a single sharp peak at minus 108, towering over the chance floor, with the osmo geometry at minus 115 marked well below it — the referee can rule. Right: yield versus colour code on the 15 August capture shows several modest bumps of similar height above the floor with none three times the runner-up — the dominance gate refuses to pick.">
  <line x1="45" y1="20" x2="45" y2="180" stroke="var(--fg-muted)"/>
  <line x1="45" y1="180" x2="320" y2="180" stroke="var(--fg-muted)"/>
  <text x="20" y="30" fill="var(--fg-muted)" font-size="9">CRC</text>
  <text x="20" y="42" fill="var(--fg-muted)" font-size="9">yield</text>
  <text x="182" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BKN1 offset (dibits)</text>
  <polyline points="45,172 90,171 120,168 150,60 165,34 180,58 210,166 250,170 320,172" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <line x1="165" y1="34" x2="165" y2="180" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="165" y="212" text-anchor="middle" fill="var(--accent)" font-size="10">−108: sharp optimum</text>
  <circle cx="120" cy="168" r="4" fill="none" stroke="currentColor"/>
  <text x="112" y="156" text-anchor="end" fill="currentColor" font-size="9">−115 (osmo)</text>
  <text x="182" y="228" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">geometry: referee rules</text>
  <line x1="395" y1="20" x2="395" y2="180" stroke="var(--fg-muted)"/>
  <line x1="395" y1="180" x2="660" y2="180" stroke="var(--fg-muted)"/>
  <text x="527" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9">colour code 0..63</text>
  <polyline points="395,172 420,170 440,120 455,171 480,150 500,172 525,140 545,171 575,158 600,172 630,169 660,172" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="395" y1="176" x2="660" y2="176" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="656" y="170" text-anchor="end" fill="var(--fg-muted)" font-size="9">chance floor</text>
  <text x="440" y="108" text-anchor="middle" fill="currentColor" font-size="9">140</text>
  <text x="525" y="128" text-anchor="middle" fill="currentColor" font-size="9">74</text>
  <text x="575" y="147" text-anchor="middle" fill="currentColor" font-size="9">46</text>
  <text x="527" y="228" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">colour, 15 Aug: gate refuses (140 &lt; 3×74)</text>
</svg>
<figcaption>The same referee, two verdicts: a sharp lone peak settles the geometry question, while several modest bumps — a pattern one radio with one scramble seed cannot produce — make refusal the only honest output.</figcaption>
</figure>

## The offset no capture could pin

The next unknown looked identical and wasn't. A DMO receiver needs the
DM **colour code** — the low six bits of the 30-bit scramble seed — to
descramble traffic. The signalling can't simply hand it over: the DSB's
SCH/S is *always* scrambled with colour 0 (like TMO's BSCH), so it
decodes regardless of the traffic colour and reads colour 0 forever. The
field genuinely lives in the DM-SYNC SYSINFO carried by the DSB's SCH/H
(EN 300 396-3) — so why not pin its bit offset from the capture, the way
the geometry was pinned?

Because the capture couldn't. The operator's radios ran colour **3** —
binary `000011` — which lights only the field's two least-significant
bits. Any 6-bit window whose bottom two bits match is consistent with
the data; an empirical scan across candidate offsets found **no unique
window**. Hardcoding one anyway would have been the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its most seductive form: the parser would decode *this* capture
correctly, a round-trip test would enshrine the guess, and the first
network running colour 40 would decode garbage with every test green.
The reference stable was no help either — this is exactly the "know what
a reference does NOT decide" rule from
[Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }}):
no independent decoder pinned that offset in a form we could
cross-check against a second one.

## Recovering the value instead of the layout

So GopherTrunk declines to parse what it cannot pin, and recovers the
value by measurement instead — the same referee, run at decode time:

```go
// internal/radio/tetra/dmo_decode.go (shape) — RecoverDMColourCode
base := baseMNI &^ 0x3F
var counts [64]int
for c := 0; c < 64; c++ {
    seed := base | uint32(c)
    n := 0
    for i := range bursts {
        b := bursts[i]
        /* … skip non-DNBs … */
        if len(DMBurstTCHSpeechSoft(b, seed)) == 2 ||
            len(DMBurstTCHSpeech(b, seed)) == 2 {
            n++
        }
    }
    counts[c] = n
}
/* … best / runner-up … */
confident = best >= dmColourMinCRC &&
    best >= dmColourDominance*max(second, 1)
```

Try all 64 colours, count CRC-valid speech frames at each, and pick the
winner — but only when the winner *deserves* it. The gate demands at
least `dmColourMinCRC = 6` CRC-valid bursts **and** a
`dmColourDominance = 3`× lead over the runner-up, thresholds set by the
physics of the measurement: on the capture that proved the mechanism,
the true colour scored **35 CRC-valid TCH/S bursts while every wrong
colour managed ≤ 3** — 70 speech frames, 2.1 s of clear PCM, recovered
with no manual override (the story
[TETRA End to End Part 12]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
tells end to end). An encrypted call, an unreceivable capture, or plain
noise clears neither bar, and the function says so instead of guessing.
The synthetic pin, `TestRecoverDMColourCode`, checks both faces: a
colour-3 stream with noise distractors must recover 3 confidently, and
an **all-noise set must refuse** — the refusal path is a tested feature,
not an error branch.

Note what was and wasn't decided. The colour *value* is recovered per
transmission; the SYSINFO field's *bit offset* stays unparsed until a
capture with a colour like 42 (`101010`) can pin it uniquely. The
unknown is quarantined exactly the way
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})
quarantined the unnamed comms-type enum: decoded around, logged,
never load-bearing.

## When the gate says no

Then came the capture that tested the gate's nerve. The 15 Aug
operator run — 25 s of clear, colour-0 PTT — decoded DSB signalling at
~90%, yet the 64-colour sweep (`TestTETRADMOColourScan`, the
`GT_TETRA_DMO_SCAN=1` diagnostic) produced the right-hand curve in the
figure: **several colours rising modestly at once** — 140, 74, 46
CRC-valid of 831 DNBs — none dominant. The gate refused: 140 < 3×74.
The operator read it as "the colour guessing is broken."

It was the opposite. One radio scrambling with one seed *cannot* produce
three partial winners — a wrong model can. The refusal, plus the
per-burst colour map the diagnostic preserved, is what eventually
cracked the real cause: the radios ran a **non-zero network MNI**
(MCC 250 / MNC 1), and TETRA seeds its scrambler from the full 30-bit
extended colour code — so a search over bare colours 0..63 was
searching a 64-seed slice of a space whose true seed it could never
reach. Every candidate sat near the floor; partial keystream overlaps
made a few rise together. The fix threads the configured MNI into the
search (`baseMNI` in the excerpt above, from the `tetra_mcc`/`tetra_mnc`
config keys), pinned failing-first by `TestRecoverDMColourCodeNonZeroMNI`
scrambling with the independently-derived osmo seed formula.

Had the gate picked the 140-yield colour — a "33% solution" — the
decoder would have shipped a wrong descramble model that half-worked on
one capture, and the MNI blind spot might still be undiagnosed. The
capture referees, but only if the machinery is allowed to say *no
contest*.

### How that principle shaped the Go code

- **Sweeps are decode-time citizens, not notebook scripts.**
  `RecoverDMColourCode` ships in the production pipeline; the same
  yield-maximization that settled the lab question runs on every
  unconfigured DMO transmission.
- **Every empirical pick carries a confidence bit.** The function returns
  `confident`, and callers keep their configured default when it's false —
  a chance-floor winner never silently becomes state.
- **Diagnostics outlive the bug they were built for.**
  `TestTETRADMOColourScan` asserts nothing; it prints the burst→colour
  map so the *next* wrong model reveals its shape too — the instrument
  mindset [Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
  makes systematic.
- **Under-determined layouts stay unparsed.** No struct field exists for
  the SYSINFO colour offset; the absence is deliberate and commented, so
  nobody "helpfully" fills it in from one capture.

## Where this goes next

The colour saga's near-misses all share a shape: a test that would have
agreed with the code because both encoded the same wrong assumption.
[Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
engineers that failure mode away — fixture transmitters that scramble
like real radios, fake servers that enforce the real protocol's
strictness, and control paths built from components the code under test
doesn't share.

## FAQ

**What should I do when two reference implementations disagree about a
wire format?**
Stop reading and start scoring. Find a metric where the correct answer
has a structural advantage — CRC yield is the usual one, because a wrong
hypothesis sits at a chance floor — and run both candidates over a real
capture. If the winner doesn't win by a wide margin, the capture is
telling you your model has a deeper problem than the parameter you're
sweeping.

**Why is CRC yield a better referee than signal metrics like EVM or
SNR?**
Because it can't be flattered by anything except correctness. EVM
improved 34%→8% on a GopherTrunk equalizer variant that decoded zero
frames; a CRC passes only when descramble, deinterleave, FEC and layout
are all simultaneously right, and a wrong hypothesis passes it ~1/256 of
the time by luck. Sharp peaks in yield are evidence; smooth improvements
in proxies are often self-deception.

**How does GopherTrunk recover the TETRA DMO colour code without
decoding it from signalling?**
`RecoverDMColourCode` brute-forces all 64 colours (folded with the
configured network MNI) and picks the one that maximizes CRC-valid
TCH/S speech across the buffered bursts — accepting only a winner with
≥ 6 CRC-valid bursts and a 3× lead over the runner-up. On the verifying
capture the true colour scored 35 against ≤ 3 for every other candidate.

**Isn't refusing to pick a colour just failing more politely?**
No — it's preserving the information a forced pick destroys. The 15 Aug
refusal encoded a physically impossible yield pattern, which pointed the
investigation at the descramble *model* (the MNI blind spot) instead of
at a plausible-but-wrong colour. A decoder that always answers converts
model errors into silent misconfigurations.

**Can one capture ever fully validate a parsed field?**
Only if it exercises enough of the field's range to make the layout
unique — colour 3 in a 6-bit field cannot. Until a discriminating
capture exists, GopherTrunk's pattern is to recover the value
empirically, log the raw bits for later confirmation, and keep the
offset out of load-bearing code — the on-air gate
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
formalizes.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Clean-Room Rules — Reading Without Copying]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
· Next →
[Part 7: Tests That Can Disagree With You]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})
