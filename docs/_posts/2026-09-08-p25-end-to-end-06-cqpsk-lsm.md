---
title: "P25 End to End, Part 6: CQPSK & LSM — The Linear Twin Path"
description: P25 Phase 1's second physical layer — the opt-in CQPSK/LSM demodulator GopherTrunk runs when an FM discriminator produces near-random dibits, its T/2 fractionally-spaced CMA equalizer and carrier-recovery chain, and the hard rule that simulcast does not imply LSM.
category: deep-dives
keywords: p25 cqpsk demodulator, p25 lsm linear simulcast modulation, cqpsk vs c4fm, simulcast p25 decoding, fractionally spaced equalizer cma, costas loop carrier recovery, p25 phase 1 demod mode, sdr p25 simulcast distortion, gophertrunk p25
tags: [p25-end-to-end, p25, cqpsk, lsm, equalization, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 6
---

*Part 6 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
turned channel numbers into hertz; every layer so far rode Part 1's FM
discriminator. This part is the first full twin pair of the series: the same
Phase 1 protocol, transmitted as a **linear** modulation that a discriminator
turns into near-random dibits. It is where the equalizer lives, where the
carrier-recovery lessons were paid for, and where GopherTrunk's own docs
were once the bug.*

> **TL;DR:** A minority of P25 Phase 1 sites transmit **LSM/CQPSK** — a
> π/4-DQPSK-family linear modulation — instead of C4FM. GopherTrunk's
> opt-in `DemodCQPSK` path (`internal/radio/p25/phase1/receiver/cqpsk.go`)
> runs coarse carrier seed → NCO → complex RRC matched filter → AGC →
> Gardner T/2 timing → **`equalizer.FSE`, a fractionally-spaced blind CMA
> equalizer** (the `fse` field) → Costas fine tracking → differential
> decode → `lsmDibitRemap`, emitting dibits interchangeable with the C4FM
> path. Getting it working took four acts on issue #492 (0/8 → 8/8 locks on
> real captures). And the selection rule is empirical, not inferential:
> **simulcast does not imply LSM** — forcing CQPSK on a C4FM simulcast site
> kills the decode (issue #935).

**Key takeaways**

- **Same dibits out, entirely different physics.** The CQPSK path ends in
  `lsmDibitRemap` so FSW detect, NID parse and the TSBK trellis never know
  which demodulator ran — one control-channel state machine, two physical
  layers, and a standing twin-drift risk.
- **A linear channel is an invertible channel.** Simulcast multipath is a
  linear distortion of complex symbols, so an equalizer can undo it — the
  structural reason this path carries a T/2 fractionally-spaced CMA
  equalizer while the discriminator path has none.
- **Differential decode removes phase, not rotation.** A residual carrier
  offset spins the constellation a constant angle per symbol, and at 4800
  baud it takes 3.75× less offset to break than TETRA's 18000 — why this
  path needed a coarse seed, an NCO and a Costas loop.
- **Choose `cqpsk` empirically, never by "the site is simulcast."** LSM is
  transmitter coordination, not a modulation; emission designators cover
  both; and a genuinely three-tower simulcast system decodes fine in C4FM
  (issue #935).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Mode selection | `"" / c4fm / fm` vs `cqpsk / lsm / linear` | `internal/radio/p25/phase1/receiver/modes.go` (`ParseDemodMode`) |
| The chain | seed → NCO → RRC → AGC → Gardner → FSE → Costas → decode | `cqpsk.go` (`cqpskDemod.process`) |
| Blind equalizer | T/2 fractionally-spaced CMA, span 6 symbols | `cqpsk.go` (`fse`, `equalizer.NewFSE`) |
| Coarse carrier seed | multi-lag fit + multipath modulus-CV gate | `cqpsk.go` (`robustSeedHz`, `seedModulusCV`) |
| Fine tracking | QPSK Costas loop, 120 Hz BW at 4800 baud | `cqpsk.go` (`sync.NewQPSKCostas`) |
| Dibit convention | LSM constellation → TIA-102.BAAA dibits | `cqpsk.go` (`lsmRotation`, `lsmDibitRemap`) |
| Voice-chain wiring | grant's demod mode string → receiver mode | `internal/voice/composer/p25p1_voice.go` (`resolveP25Phase1DemodMode`) |

## In this post

- **Why a linear path exists** — what simulcast does to a discriminator, and what LSM actually is.
- **The chain in one pass** — from raw IQ to remapped dibits.
- **The equalizer C4FM never got** — the T/2 FSE and what it opens.
- **The seed that lies on simulcast** — carrier recovery and its multipath gate.
- **Two paths, one truth** — twin drift, and the myth the docs taught.

## Why a linear path exists

"Compatible" is the load-bearing word in C4FM's name: the waveform family
was designed to be receivable either as four-level FM or as a phase
modulation. Some operators — often, but not only, on simulcast systems —
transmit **LSM (Linear Simulcast Modulation)**, a π/4-DQPSK-family linear
waveform, because a linear signal survives multi-transmitter overlap
better. When several towers transmit the same bits with small differential
delays, the receiver hears the *sum* of delayed copies. For a linear
modulation that sum is just a linear channel — inter-symbol interference an
equalizer can invert
([the ISI story in general form]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})).
For an FM discriminator the sum is a disaster: FM demodulation is
nonlinear, and overlapping carriers produce artifacts no downstream filter
can cleanly remove.

Push an actually-linear LSM signal through a quadrature FM discriminator
and you get near-random dibits and a frame sync word that never matches —
the issue #275 symptom that motivated this path. So the receiver carries a
second demod mode:

| | `DemodC4FM` (default) | `DemodCQPSK` |
|---|---|---|
| Front half | FM discriminator (real waveform) | complex RRC matched filter |
| Timing | Mueller-Müller | Gardner (mandatory, T/2 output) |
| Carrier error | DC bias → `CoarseAFC` | rotation → NCO seed + Costas |
| Equalizer | none | T/2 fractionally-spaced CMA |
| Decision | 4-level slicer | differential quadrant + remap |
| Right for | the large majority of sites — including most simulcast | sites actually transmitting a linear waveform |

The last row is the one the project got wrong for a long time, and we will
come back to it.

## The chain in one pass

`cqpskDemod` (`internal/radio/p25/phase1/receiver/cqpsk.go`) wraps the same
`demod.PiOver4DQPSK` primitive TETRA uses — rotation π/4, the TIA-102.BAAA
LSM constellation — and composes the rest around it:

```go
// internal/radio/p25/phase1/receiver/cqpsk.go (shape) — process
func (c *cqpskDemod) process(iq []complex64) []uint8 {
    if !c.seeded {
        c.seedBuf = append(c.seedBuf, iq...) // accumulate ≥2048 samples
        /* … robustSeedHz → nco.SetOffset; reset Costas + FSE … */
    }
    c.rotated = c.nco.Mix(c.rotated, iq)           // remove coarse offset
    c.matched = c.dq.MatchedFilter(c.matched, c.rotated)
    c.matched = c.agc.Process(c.matched, c.matched) // gain-independence
    c.twoSps = c.gardner.Process2x(c.twoSps, c.matched) // [mid, on-time]
    for i := 0; i < nSym; i++ {
        y, e := c.fse.Process(c.twoSps[2*i], c.twoSps[2*i+1]) // T/2 CMA
        c.cmaErr = e
        c.symbols = append(c.symbols, c.costas.Update(y)) // fine carrier
    }
    c.dibits = c.dq.Decode(c.dibits, c.symbols)
    for i, d := range c.dibits {
        c.dibits[i] = lsmDibitRemap[d&3] // → canonical TIA-102 dibits
    }
    return c.dibits
}
```

Read the last loop first — it is the twin-pair contract: after the remap,
the CQPSK path's dibit values match what `SymbolToDibit` produces from
C4FM, so everything from
[Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})'s
FSW detector onward is demod-agnostic. The AGC matters more than it looks:
the Gardner timing-error detector and the CMA weight update both use
un-normalised, amplitude-dependent error terms, so without it the path
locked only in a narrow RTL-SDR gain window — a regression issue #275's
reporters actually measured.

## The equalizer C4FM never got

The `fse` field is the piece the series brief keeps pointing at, because it
is the lever the default C4FM voice path lacks (that gap is
[Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})'s
whole subject). Post-Gardner symbols are unit-modulus π/4-DQPSK points, so
the **Constant Modulus Algorithm** applies — a blind equalizer needing no
training sequence. Two design choices carry the weight:

- **Fractionally spaced (T/2), not symbol spaced.** Gardner emits the
  on-time *and* half-symbol samples, and the equalizer runs two taps per
  symbol (`cqpskFSESymbolSpan = 6` symbols). A T/2 equalizer synthesizes
  the receive matched filter implicitly, so it opens both simulcast
  multipath ISI **and** the C4FM-transmit-vs-RRC-receive pulse mismatch —
  the latter is what left a real C4FM-shaped signal's constellation closed
  through the linear path (issue #492). A symbol-spaced equalizer cannot
  fix a pulse-shape mismatch; it only ever sees the symbols the wrong
  filter already produced.
- **Brisk step, leaky taps.** `cqpskFSEStep = 0.025` because real
  control-channel captures are short — the blind equalizer must open the
  constellation within a few hundred symbols of lead-in before the FSW
  arrives — and `cqpskFSELeak = 5e-4` pulls the FSE's larger null space
  toward identity so the taps don't wander on clean, ISI-free input.

The safety analysis is the one the
[TETRA equalizer work]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
made a series-wide rule: a continuously-adapting filter ahead of a
differential decoder is only safe if its phase is constrained, and here the
Costas loop *after* the FSE pins the constellation to the decision grid
symbol by symbol, so the differential product sees a stable frame. The
receiver also retains `cmaErr` — the equalizer's `|y|²−R²` convergence
proxy — for the replay diagnostics, because "is the equalizer converged" is
a question you want answerable from a log, not a debugger.

## The seed that lies on simulcast

The differential decode removes a constant carrier *phase* but not a
constant per-symbol *rotation*: a residual offset Δf turns every symbol by
2π·Δf/4800, and the FSW never correlates. (TETRA gets away without any
carrier recovery only because 18000 baud rotates 3.75× less per symbol —
the comment in `cqpsk.go` says exactly this.) So the path seeds an NCO from
a one-shot coarse estimate, then lets a 120 Hz Costas loop track the
residual.

The war story is in the *gate* on that estimate. The bare lag-1 (Kay)
autocorrelation estimator reads a simulcast channel's multipath as a
spurious carrier offset — ~650–750 Hz on issue #492's captures — because a
deep simulcast null shifts the spectral centroid, and an autocorrelation
estimator cannot tell that bias from a real offset. Mis-tune the NCO by it
and the Costas loop rails. `robustSeedHz` therefore sharpens the estimate
with a multi-lag phase-ramp fit, then **checks its own answer**: de-rotate
by the candidate, run the matched filter, and measure the coefficient of
variation of the symbol modulus (`seedModulusCV`). A true carrier offset
only *rotates* a constant-modulus constellation (low CV); multipath *blurs*
the modulus (high CV, gated at `cqpskSeedMaxModulusCV = 0.24`). A rejected
seed leaves the NCO identity and lets the loop acquire the necessarily
in-range true offset on its own.

Issue #492 took four acts to reach that design — no carrier recovery at
all, then the poisoned seed, then the FSE, then a brute-force BCH hot path
that only appeared once locking finally worked — and the score went **0/8 →
3/8 → 8/8 locks** on the reporter's real captures. The full postmortem,
including the diagnostic line that was reading the *wrong demodulator's*
state, is
[From the Issue Tracker Part 6]({{ '/blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two transmit towers reach one receiver with a differential delay; through the FM discriminator path the summed copies close the C4FM eye with no recovery stage, while through the linear CQPSK path the same sum is a linear channel that the fractionally spaced equalizer inverts to reopen the constellation">
  <!-- towers -->
  <polygon points="40,86 52,86 46,40" fill="none" stroke="currentColor"/>
  <polygon points="40,196 52,196 46,150" fill="none" stroke="currentColor"/>
  <text x="20" y="104" fill="var(--fg-muted)" font-size="10">tower A</text>
  <text x="20" y="214" fill="var(--fg-muted)" font-size="10">tower B (+Δt)</text>
  <line x1="56" y1="60" x2="200" y2="120" stroke="var(--fg-muted)"/>
  <line x1="56" y1="170" x2="200" y2="130" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <circle cx="208" cy="125" r="7" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="182" y="150" fill="var(--fg-muted)" font-size="10">sum of delayed copies</text>
  <!-- discriminator branch -->
  <line x1="215" y1="118" x2="300" y2="70" stroke="currentColor"/>
  <rect x="302" y="52" width="128" height="30" fill="none" stroke="currentColor"/>
  <text x="366" y="71" text-anchor="middle" fill="currentColor" font-size="10">FM discriminator</text>
  <line x1="430" y1="67" x2="472" y2="67" stroke="currentColor"/>
  <path d="M 480 82 C 500 46 540 90 560 54 M 480 54 C 500 92 540 44 560 82" fill="none" stroke="currentColor"/>
  <text x="576" y="62" fill="currentColor" font-size="10">closed eye —</text>
  <text x="576" y="76" fill="currentColor" font-size="10">nonlinear mix,</text>
  <text x="576" y="90" fill="currentColor" font-size="10">nothing inverts it</text>
  <!-- linear branch -->
  <line x1="215" y1="132" x2="300" y2="180" stroke="var(--accent)"/>
  <rect x="302" y="165" width="128" height="30" fill="none" stroke="var(--accent)"/>
  <text x="366" y="184" text-anchor="middle" fill="var(--accent)" font-size="10">linear path (CQPSK)</text>
  <line x1="430" y1="180" x2="446" y2="180" stroke="var(--accent)"/>
  <rect x="448" y="165" width="70" height="30" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="483" y="184" text-anchor="middle" fill="var(--accent)" font-size="10">T/2 FSE</text>
  <line x1="518" y1="180" x2="540" y2="180" stroke="var(--accent)"/>
  <circle cx="580" cy="180" r="26" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <circle cx="598" cy="167" r="3" fill="var(--accent)"/><circle cx="562" cy="167" r="3" fill="var(--accent)"/>
  <circle cx="562" cy="193" r="3" fill="var(--accent)"/><circle cx="598" cy="193" r="3" fill="var(--accent)"/>
  <text x="548" y="226" fill="var(--accent)" font-size="10">ISI inverted — open again</text>
  <text x="236" y="238" fill="var(--fg-muted)" font-size="10">same air, two demodulators: only the linear one gives an equalizer something invertible</text>
</svg>
<figcaption>Simulcast differential delay is fatal noise after an FM discriminator but an invertible linear channel on the CQPSK path — which is why the equalizer lives here.</figcaption>
</figure>

## Two paths, one truth

Twin paths drift, and this pair drifted twice in ways worth naming.

First, **selection folklore**. GopherTrunk's docs, config comments and UI
labels all once taught "simulcast ⇒ CQPSK." Issue #935 disproved it on air:
a genuinely three-tower simulcast site decodes reliably in **C4FM** — in
GopherTrunk and SDRTrunk alike — and forcing CQPSK kills it, because LSM is
transmitter *coordination*, not a modulation, and licensing metadata (the
site's `10K1D7W` emission designator covers both) can't decide either. The
`DemodMode` doc comment in `modes.go` now carries the corrected rule in
permanent form: **C4FM is the default; `cqpsk` is warranted only when a
strong, clean signal will not lock in C4FM** — near-random dibits, no FSW.
The full story of unwinding the project's own guidance is
[From the Issue Tracker Part 7]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }}).
(The reporter's actual problem was gain entered in the wrong units. The
myth just kept everyone looking elsewhere.)

Second, **plumbing drift**. The demod mode is a per-system (and per-channel)
string that has to survive three hand-offs: config → control-channel
pipeline → grant → voice chain. The composer's
`resolveP25Phase1DemodMode` (`internal/voice/composer/p25p1_voice.go`)
parses the string off the grant, warn-logs unknown values, and falls back
to C4FM — and the voice chain now prints
`composer: p25p1 voice chain started demod_mode=…` precisely because a
field log once showed the CC locked on cqpsk with no way to verify the
voice chain wasn't silently still on c4fm (issue #356 follow-up). The
wideband pipeline, meanwhile, stamped *no* mode onto its grants at all
until #935 — CQPSK was decorative on that whole path. Same lesson as the
[Two Pipelines postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}):
when a property exists on one side of a twin pair, audit the other side for
the entire class.

### How the linear-channel principle shaped the Go code

- **The equalizer sits where the channel is linear.** The FSE runs on
  complex symbols before the differential decode — the only domain where
  simulcast delay is a clean convolution — with the Costas loop pinning
  phase so the differential product stays safe.
- **Every estimator gates its own output.** `robustSeedHz` rejects its
  estimate when de-rotation doesn't restore constant modulus; a wrong seed
  held back is recoverable, a wrong seed applied rails the loop.
- **The twin contract is enforced at the boundary.** `lsmDibitRemap`
  converts to canonical TIA-102 dibits inside the demod, so no downstream
  code ever branches on which physics ran.
- **Mode strings parse in one place.** `ParseDemodMode` accepts
  `cqpsk`/`lsm`/`linear`, returns `ok=false` on typos so callers warn and
  fall back — a config mistake degrades to the default, never to silence.

## Where this goes next

CQPSK is Phase 1 wearing linear clothes; the next twin is a different
protocol generation on the same control channel.
[Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
takes on Phase 2 TDMA: H-DQPSK at 6000 symbols/s carrying two voice slots
per 12.5 kHz carrier, the shared differential primitive it borrows from
TETRA with a π/8 twist, and the FEC-knob hand-off that issue #882 proved
can silently default to zero on one pipeline.

## FAQ

**How do I know if my P25 site needs CQPSK mode?**
Empirically, and only empirically: a strong, clean signal that will not
lock in C4FM — near-random dibits, no frame sync — is the CQPSK signature.
Do not infer it from "the site is simulcast" (most simulcast is C4FM) or
from licensing data (emission designators cover both). See issue #935 and
the `DemodMode` documentation.

**Why does the CQPSK path need carrier recovery when TETRA's π/4-DQPSK
doesn't?**
Baud rate. A residual offset rotates the differential constellation by
2π·Δf per symbol interval; at 4800 baud the same hertz of error costs
3.75× more rotation than at TETRA's 18000. P25's linear path crosses the
quadrant boundary at offsets an ordinary tuner routinely has, so it seeds
an NCO and tracks with a Costas loop.

**What is the difference between LSM and CQPSK?**
In this context, none that the receiver cares about: LSM is the
TIA-102.BAAA linear simulcast waveform, CQPSK the modulation family it
belongs to, and GopherTrunk accepts `cqpsk`, `lsm` and `linear` as aliases
for the same `DemodCQPSK` path. The important distinction is
LSM-the-waveform versus simulcast-the-deployment — only the former needs
this demodulator.

**Can the CQPSK path decode a normal C4FM site?**
Often, yes — "compatible" 4-level FM is receivable as a phase modulation,
and the T/2 equalizer exists partly to absorb the C4FM-vs-RRC pulse
mismatch. But it is the harder road: more stages, more acquisition time,
and no benefit on a discriminator-friendly signal. That's why C4FM stays
the default and CQPSK stays opt-in.

**Why is the equalizer only on this path and not on C4FM voice?**
Because the discriminator output isn't a linear function of the channel —
there is no clean convolution to invert after nonlinear FM demodulation.
Equalizing P25 means either the linear path (here) or porting an
equalizer *ahead* of the decision chain on C4FM — the open weak-signal work
[Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})
lays out, gated on a real capture.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Channel Identifiers & Band Plans — From Channel Number to Hertz]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
· Next →
[Part 7: Phase 2 TDMA — Two Voices per Carrier]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})
