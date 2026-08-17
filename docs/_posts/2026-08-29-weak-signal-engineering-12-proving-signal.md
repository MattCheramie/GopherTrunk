---
title: "Weak-Signal Engineering, Part 12: Proving It's the Signal — Rate Invariance & Independent Resamplers"
description: "Experimental design as an engineering skill, with issue #764 as the worked example — how an independent 4:1 resampler proved a ten-dB decode deficit was baked into the captured samples, how clipping was ruled out, and how a unit test now pins the invariant the experiment relied on."
category: deep-dives
keywords: rate invariance decode, independent resampler control, capture rate snr deficit, phase noise reciprocal mixing, ruling out clipping, ddc linearity, experimental design dsp, issue 764 mt anakie, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, debugging, ddc, phase-noise, capture, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 12
---

*Part 12 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }})
finished the lever list: equalize, go soft, diversify. This part is about the
skill that decides whether any of those levers is even the right tool —
because the most expensive mistake in weak-signal work is fixing the DSP when
the problem is in the samples. The worked example is
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764): a P25 control
channel that locked from a 2.5 MS/s capture and failed from a 10 MS/s capture
of the same site, same antenna, same day. Every instinct says "the decoder
mishandles the higher rate." The experiment said otherwise, and the way it
said it — an independent resampler as a control — is the template.*

> **TL;DR:** The Mt Anakie carrier (−812.5 kHz offset) replays at demod SNR
> ≈19.7 dB / EVM 7.4% from the 2.5 MS/s capture (locks) but ≈9.5 dB / EVM
> 22.5% from the 10 MS/s capture (no lock). The decisive control: decimate
> the 10 MS/s file 4:1 with an **independent resampler** and replay it
> through the proven 2.5 MS/s path — it reproduces the **same ≈9.5 dB**, so
> the ~10 dB deficit is baked into the captured samples, not GopherTrunk's
> DDC. Neither capture clips (both peak ≈−48 dBFS), and the wideband FFT
> *carrier* SNR is actually **higher** at 10 MS/s — carrier-clean but
> modulation-degraded is the signature of front-end **phase noise /
> reciprocal mixing** at the Airspy's native 10 MS/s clock.
> `TestDownconverterSNRInvariantAcrossRate`
> (`internal/scanner/ccdecoder/ddc_highrate_test.go`) pins the invariant the
> whole argument leans on: a noisy channel reaches the receiver at the same
> in-channel SNR whether decoded natively at 10 MS/s or decimated to
> 2.5 MS/s.

**Key takeaways**

- **Find the invariant your DSP guarantees, then test the symptom against
  it.** GopherTrunk's decode path is rate-invariant by construction — both
  down-converters normalise to the per-protocol channel rate and the
  receiver/AGC are sized from that output rate. A symptom that *follows the
  capture rate* therefore points at the captured data, not the steady-state
  DSP.
- **The control must be independent of the code under suspicion.** Decimating
  with the suspect DDC and getting the same answer proves nothing — a shared
  bug produces a shared answer. `dsp.NewResampler` shares no code with the
  `ccdecoder.Downconverter`, which is what gives the 4:1 control its force.
- **Rule out the boring explanations with numbers.** Overload was excluded
  because both captures peak ≈−48 dBFS — 48 dB of headroom is not clipping.
  Without that measurement, "the wider front end overloaded" survives forever
  as a plausible story.
- **Carrier-clean but modulation-degraded is a diagnosis, not a paradox.**
  Reciprocal mixing smears each strong neighbour's energy across the channel
  in proportion to the LO's phase-noise skirt — the carrier *peak* stays
  pretty while the modulation drowns. That reading sent the fix to the
  operator's sample-rate choice, not to a year of DSP archaeology.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Rate invariance | DDC normalises any input rate to the channel rate | `internal/scanner/ccdecoder/ddc.go` (`Downconverter`, `DDCTargetRateHz`) |
| The invariant's pin | same in-channel SNR at 10 MS/s and decimated 2.5 MS/s | `internal/scanner/ccdecoder/ddc_highrate_test.go` (`TestDownconverterSNRInvariantAcrossRate`) |
| Independent control | polyphase 4:1 decimation, zero shared DDC code | `internal/dsp` (`NewResampler(1, 4, 64, 8.6)`) |
| Level pin | DDC output level identical across capture rates | `ddc_highrate_test.go` (`TestDownconverterC4FMLevelRateInvariant`) |
| Alias pin | wideband neighbours rejected at both rates | `ddc_highrate_test.go` (`TestDownconverterRejectsWidebandNeighbours`) |
| The postmortem | the #764/#771 story at narrative length | [From the Issue Tracker, Part 5]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}) |

## In this post

- **A symptom that follows the capture rate** — the #764 report.
- **The invariant** — why the decode path cannot care about capture rate.
- **The decisive experiment** — an independent resampler as the control.
- **Ruling out the alternatives** — clipping, then the FFT surprise.
- **Pinning it forever** — the regression test, and the generalized recipe.

## A symptom that follows the capture rate

The report was clean and reproducible: a P25 control channel at −812.5 kHz
from centre (Mt Anakie) locks every time when replayed from a 2.5 MS/s
Airspy capture, and never locks from a 10 MS/s capture — offline, from
files, no live radio in the loop. Measured at the demod: **≈19.7 dB SNR /
7.4% EVM** from the 2.5 MS/s file, **≈9.5 dB / 22.5% EVM** from the 10 MS/s
file. Ten dB is not a subtle difference; it is the whole marginal regime of
[Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
crossed in one config change.

The natural hypothesis — the one the issue was filed under — is that the
decoder mishandles the higher rate: a decimation filter with wrong cutoff, an
AGC sized for the wrong bandwidth, an alias folding in. It is a *good*
hypothesis; an earlier "fix" for this very issue had already been merged
once on weaker evidence, which is how the project learned the verification
discipline it now enforces. The question is how to test it in a way that can
actually lose.

## The invariant: the decode path cannot care

Start from what the code guarantees. Every capture, at whatever rate, meets
a down-converter that mixes the target channel to baseband and resamples it
to a fixed per-protocol channel rate — 48 kHz for the 4800-baud C4FM family,
144 kHz for TETRA. The receiver, its filters, and its AGC are all sized from
that *output* rate. Nothing downstream of the DDC ever sees the capture
rate. So the decode path is **rate-invariant by design**: if the DDC is
correct, a given over-the-air signal must produce the same in-channel SNR —
and the same decode — whether it was captured at 2.5 or 10 MS/s.

That converts the vague "maybe the decoder mishandles 10 MS/s" into a sharp
dichotomy. Either the DDC violates its invariant (a real, findable bug), or
the two captures do not contain the same signal quality (an RF fact). No
third option. The experiment's only job is to tell those two apart.

## The decisive experiment: an independent resampler

Here is the move worth stealing. Take the *failing* 10 MS/s file. Decimate
it 4:1 to 2.5 MS/s using a resampler that shares **no code** with the DDC
under suspicion — `dsp.NewResampler`, the general polyphase resampler from a
different package. Now replay the result through the 2.5 MS/s path that is
*proven* good by the passing capture. Two possible outcomes, each decisive:

- **It decodes.** The signal quality was in the file all along, and the
  10 MS/s DDC path is destroying it. Bug found; go fix the DDC.
- **It fails the same way.** The proven-good path, fed the independently
  decimated samples, still sees ≈9.5 dB — so the deficit is *in the
  samples*, and no amount of DDC work can recover what the front end never
  delivered.

It failed the same way: **the same ≈9.5 dB**, through a path with zero
shared code. That single measurement moved the entire investigation out of
GopherTrunk and into the capture hardware. The independence is what makes
the logic sound — decimating with the suspect DDC itself would have been
circular, the same self-consistency failure this series keeps naming (a
synthetic test that scrambles and descrambles with the same wrong code
passes forever; a fake server that switches on the same wrong constant as
the client validates nothing).

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Block diagram of the rate-invariance experiment. The failing ten-megasample capture feeds two paths: the native ten-megasample DDC path, and an independent four-to-one polyphase resampler followed by the proven two-and-a-half-megasample DDC path. Both paths measure about nine and a half dB in-channel SNR, so an equals sign joins them and the conclusion box says the deficit is in the captured samples, not the DDC.">
  <rect x="8" y="86" width="140" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="78" y="107" text-anchor="middle" fill="currentColor" font-size="10">10 MS/s capture</text>
  <text x="78" y="123" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the failing file, peak ≈−48 dBFS</text>
  <line x1="148" y1="98" x2="196" y2="56" stroke="currentColor"/><polygon points="192,58 202,52 196,64" fill="currentColor"/>
  <line x1="148" y1="126" x2="196" y2="168" stroke="var(--accent)"/><polygon points="192,164 202,172 194,174" fill="var(--accent)"/>
  <rect x="204" y="30" width="180" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="294" y="50" text-anchor="middle" fill="currentColor" font-size="10">native 10 MS/s DDC path</text>
  <text x="294" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Downconverter → 48 kHz</text>
  <rect x="204" y="146" width="180" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="294" y="164" text-anchor="middle" fill="var(--accent)" font-size="10">independent 4:1 resampler</text>
  <text x="294" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dsp.NewResampler(1, 4, 64, 8.6)</text>
  <line x1="384" y1="170" x2="420" y2="170" stroke="var(--accent)"/><polygon points="416,166 426,170 416,174" fill="var(--accent)"/>
  <rect x="428" y="146" width="150" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="503" y="164" text-anchor="middle" fill="var(--accent)" font-size="10">proven 2.5 MS/s path</text>
  <text x="503" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the one that locks</text>
  <line x1="384" y1="54" x2="596" y2="54" stroke="currentColor"/><polygon points="592,50 602,54 592,58" fill="currentColor"/>
  <line x1="578" y1="170" x2="596" y2="170" stroke="var(--accent)"/><polygon points="592,166 602,170 592,174" fill="var(--accent)"/>
  <text x="626" y="58" text-anchor="middle" fill="currentColor" font-size="10">≈9.5 dB</text>
  <text x="626" y="174" text-anchor="middle" fill="var(--accent)" font-size="10">≈9.5 dB</text>
  <line x1="626" y1="70" x2="626" y2="152" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="640" y="115" fill="var(--fg-muted)" font-size="12">=</text>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same SNR through zero shared code ⇒ the ~10 dB deficit is in the samples, not the DDC</text>
</svg>
<figcaption>The control that settled #764: the failing capture, decimated by an independent resampler and replayed through the proven path, reproduces the same ≈9.5 dB — so the deficit travels with the samples.</figcaption>
</figure>

## Ruling out the alternatives

"It's in the samples" narrows to a family of causes, and the same
measurement-first discipline sorts them.

**Overload / intermod?** No. Both captures peak at ≈**−48 dBFS** — nearly
50 dB of digital headroom, with no clipping in either file. A wider front
end admitting more total power *can* drive intermodulation, which is why
this had to be checked rather than dismissed; but IMD products of an
unclipped, −48 dBFS-peak stream are not a 10 dB in-channel deficit.

**Just more noise bandwidth?** That is not how it works — noise *density* is
what matters, and the DDC's channel filter passes the same 48 kHz either
way. The invariant test below is exactly this statement made executable.

**The surprise:** the wideband FFT **carrier** SNR — carrier peak over the
adjacent noise floor — was actually *higher* in the 10 MS/s capture. A
capture whose carrier looks better while its demodulated constellation is
10 dB worse is the fingerprint of **phase noise / reciprocal mixing**: the
LO's phase-noise skirt, integrating differently at the Airspy's native
10 MS/s clock configuration, lets every strong neighbour smear energy across
the victim channel. The carrier spike survives (it's narrowband and strong);
the modulation — which lives in precise phase transitions — drowns. The RF
physics of that mechanism gets its own operator-level treatment in
[The Analog Edge, Part 5]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }});
what matters here is the diagnostic shape: *carrier-clean but
modulation-degraded means the impairment is multiplicative in phase, and no
linear equalizer, soft decoder, or combiner will fix it.* The levers of
Parts 4–11 are the wrong aisle. The fix was operational: capture at the rate
where the front end is clean.

## Pinning it forever — and the recipe

An investigation that ends in a conversation evaporates. This one ended in a
test that makes the invariant executable:

```go
// internal/scanner/ccdecoder/ddc_highrate_test.go (shape) — TestDownconverterSNRInvariantAcrossRate
symbols := c4fmSymbols(9600, 0x764) // ~2 s of C4FM at the #764 seed
clean10 := c4fmChannel(symbols, offsetHz, 10_000_000, sigAmp) // −812.5 kHz: Mt Anakie
noise10 := whiteNoise(len(clean10), noiseAmp, 0x5151)

// Native 10 MS/s path.
sigHi := cmplxRMS(NewDownconverterWithOffset(10_000_000, DDCTargetRateHz, offsetHz).Process(nil, clean10))
noiHi := cmplxRMS(NewDownconverterWithOffset(10_000_000, DDCTargetRateHz, offsetHz).Process(nil, noise10))

// Independent 4:1 decimation to 2.5 MS/s, then the proven 2.5 MS/s path.
clean25 := dsp.NewResampler(1, 4, 64, 8.6).Process(nil, clean10)
/* … same for noise, then the 2.5 MS/s Downconverter … */

ratio := snrHi / snrLo
if ratio < 0.8 || ratio > 1.25 {
    t.Errorf("in-channel SNR not rate-invariant: … the DDC must neither create nor hide an SNR deficit (issue #764)")
}
```

Note the trick that makes the measurement exact: the DDC is linear, so
`DDC(signal+noise) = DDC(signal) + DDC(noise)` — the test runs signal and
noise through separately and reads post-DDC SNR directly from the two RMS
values, no estimator needed. Companion tests pin the level
(`TestDownconverterC4FMLevelRateInvariant`) and alias rejection
(`TestDownconverterRejectsWidebandNeighbours`) across the same rate pair. If
anyone ever makes the DDC rate-sensitive, the #764 argument stops being true
and these tests say so before a reporter does.

The generalized recipe, applicable far beyond one issue:

1. **State the invariant** your pipeline guarantees (here: in-channel SNR is
   capture-rate-invariant). If you can't state one, that's the first bug.
2. **Build the experiment that isolates it** — route the failing input
   through the proven path so only one variable moves.
3. **Use an independent implementation for the cross-check.** Shared code
   shares bugs; independence is what converts "consistent" into "correct."
4. **Rule out the boring causes with numbers**, not plausibility (−48 dBFS
   peak is a fact; "probably overload" is a mood).
5. **Leave the invariant pinned in the suite**, named after the issue.

## Where this goes next

The recipe's honest corollary: sometimes the investigation proves the code
*is* the weak link — and the fix still can't land, because there is nothing
to A/B it against.
[Part 13]({{ '/blog/deep-dives/weak-signal-engineering-13-p25-c4fm-gap/' | relative_url }})
is that case: P25 Phase 1 C4FM voice, the one decode path in GopherTrunk
with neither an equalizer nor soft FEC — diagnosed structurally, fix
sketched, and deliberately unfixed until a real weak-signal capture exists
to measure it against.

## FAQ

**Why not just trust the demod SNR numbers and skip the experiment?**
Because they don't localise. ≈9.5 dB at the demod is consistent with *both*
"the DDC damaged the samples" and "the samples arrived damaged." The
independent-resampler control is what separates the two — it holds the
decode path fixed and moves only the provenance of the samples.

**Couldn't the resampler itself have introduced the deficit?**
Then the control would have *failed the other way* on the healthy file — and
the pinned tests run the same resampler on synthetic signal-plus-noise and
require SNR preservation within 0.8–1.25×. The control is validated by the
same suite that uses it.

**What made phase noise the conclusion rather than just 'unknown RF'?**
The conjunction of three measurements: no clipping (−48 dBFS peaks), *higher*
wideband carrier SNR at 10 MS/s, and a 10 dB in-channel modulation deficit
that survives independent decimation. Additive noise can't produce that
pattern; a multiplicative phase impairment does, and the rate-dependence
matches a clock-configuration-dependent LO skirt.

**Does this mean higher capture rates are bad?**
No — it means capture rate is a front-end operating point, not a free
parameter, and some hardware is cleaner at some clocks. The decode path
genuinely does not care. Measure your own front end: capture the same
carrier at both rates and compare demod SNR, exactly as #764 did.

**Where does the #764/#771 story continue?**
The full postmortem — including the first "fix" that closed the issue
without verification, and the policy that grew out of it — is
[From the Issue Tracker, Part 5]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}).
This post extracts the reusable method; that one tells the cautionary tale.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Diversity II — Tracking Without Breaking the Differential]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }})
· Next →
[Part 13: The Odd Path Out — P25 Phase 1 C4FM]({{ '/blog/deep-dives/weak-signal-engineering-13-p25-c4fm-gap/' | relative_url }})
