---
title: "Weak-Signal Engineering, Part 3: ISI & the Linear Channel — What an Equalizer Can & Can't Fix"
description: The convolution model y = h∗x + n that every equalizer in this series inverts — what multipath and band-edge group delay do to π/4-DQPSK and C4FM symbols, which impairments are linear and therefore recoverable, which are not, and the raw-symbol-domain fact that dictates the architecture of Parts 5 and 7.
category: deep-dives
keywords: intersymbol interference, linear channel model, multipath distortion, group delay, channel convolution, differential decoding nonlinearity, equalizer limits, pi/4-dqpsk isi, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, isi, multipath, dsp, equalizer, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 3
---

*Part 3 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime, where a receiver locks but under-decodes.
[Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
fixed the measurement rules — yield is the verdict, EVM and carrier SNR are
advisory at best. Now we can meet the enemy properly. The thread capture's
constellation isn't just noisy; it's *smeared*, and smear is a different kind
of damage with a different kind of cure. This part builds the linear channel
model everything later inverts, draws the line between what an equalizer can
and cannot fix, and lands the one domain fact that quietly shapes the design
of every equalizer in the rest of the series.*

> **TL;DR:** A dispersive channel is a **convolution**: `y = h∗x + n`, where
> each received symbol is a weighted sum of the current symbol and its
> neighbours — that weighted mixing is
> **[inter-symbol interference]({{ '/reference/intersymbol-interference/' | relative_url }})**,
> and because convolution is linear it is *invertible* by another filter. That
> is the whole license an equalizer operates under. Multipath, simulcast
> echoes, and band-edge **group delay** are linear — fixable. Phase noise,
> clipping, and compression are **not** linear — no FIR can unmix them. And the
> fact with teeth: the channel is a clean convolution only over **raw
> symbols**. After the differential product `s·conj(prev)` the channel enters
> *nonlinearly*, which is why GopherTrunk's receiver grew a raw-symbol
> `SymbolSink` alongside its differential `SoftSink` — an equalizer must live
> before, or be trained on, the raw-symbol domain.

**Key takeaways**

- **ISI is deterministic mixing, not noise.** Noise adds an unknowable random
  vector to each symbol; ISI adds a *known-structure* combination of
  neighbouring symbols. That structure is exactly what makes it removable —
  and what makes more transmit power useless against it.
- **Linear in, linear out — that's the whole test.** If the impairment can be
  written as a fixed filter acting on the signal, another filter can undo it.
  If it can't (phase noise, overload), the equalizer has nothing to grab.
- **Differential decoding destroys the linearity.** `s[n]·conj(s[n−1])` is a
  product of two filtered quantities; the channel taps enter it as cross-terms,
  not as a convolution. Equalize before the differential, always.
- **The thread capture is ISI-limited, not just SNR-limited.** Its ~10 dB
  in-channel SNR should decode far better than 12% — the gap between "should"
  and "does" is the smear, and it is the recoverable part.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The channel model | `y = h∗x + n` — FIR channel, additive noise | [`/reference/intersymbol-interference/`]({{ '/reference/intersymbol-interference/' | relative_url }}), [`/reference/multipath-propagation/`]({{ '/reference/multipath-propagation/' | relative_url }}) |
| Raw-symbol tap | post-timing/AFC symbols, pre-differential | `internal/radio/tetra/receiver/receiver.go` (`Options.SymbolSink`) |
| Differential tap | `s·conj(prev)` per symbol — LLR source, *not* equalizer input | `receiver.go` (`Options.SoftSink`) |
| Where the equalizer sits | between symbol timing and differential decode | `receiver.go` (`Options.EnableEqualizer`) |
| Synthetic ISI channel | multipath fixture the equalizer tests decode through | `internal/dsp/equalizer/snapshot_cma_test.go` (`TestSnapshotCMARecoversISIChannel`) |
| What's *not* linear | phase noise / reciprocal mixing (#764), overload | `internal/scanner/ccdecoder/ddc_highrate_test.go` |

## In this post

- **The channel as a filter** — echoes, convolution, and the two-ray picture.
- **What smear does to symbols** — π/4-DQPSK and C4FM under a dispersive channel.
- **Invertible vs. not** — the sorting rule for every impairment you'll meet.
- **The differential domain is not linear** — the fact that shapes Parts 5 and 7.
- **Reading the damage on the thread capture** — smear vs. noise, told apart.

## The channel as a filter

Radio between transmitter and antenna is astonishingly well modeled by one
idea: the channel is a filter. Your antenna receives the direct wave plus
delayed, attenuated copies —
[multipath]({{ '/reference/multipath-propagation/' | relative_url }}) bounces
off terrain and buildings, simulcast systems transmit deliberate copies from
multiple towers, and even a single clean path picks up dispersion from every
bandpass filter it crosses, including your own receiver's, whose
[group delay]({{ '/reference/group-delay/' | relative_url }}) stops being flat
near the band edge. Sum a few delayed copies and you have, exactly, a
finite-impulse-response filter:

`y[n] = h0·x[n] + h1·x[n−1] + … + hk·x[n−k] + noise[n]`

— or compactly, `y = h∗x + n`. When the delay spread of `h` is a meaningful
fraction of a symbol period, each received symbol contains echoes of its
neighbours: inter-symbol interference. Two properties of this model do all the
work in this series. It is **linear** — the channel treats a sum of inputs as
the sum of its outputs — and it is (approximately, over short spans)
**time-invariant**. Linear time-invariant systems compose: run `y` through a
second filter `w`, and the result is `(w∗h)∗x`. If you can find a `w` such
that `w∗h` is close to a pure delay, the smear is *gone* — not averaged down,
not powered through: algebraically removed. Finding that `w`, blindly or with
training, is Parts 4 through 7.

## What smear does to symbols

The two constellations GopherTrunk cares most about fail differently under the
same channel. [π/4-DQPSK]({{ '/reference/pi-4-dqpsk/' | relative_url }})
(TETRA) encodes data in the *phase difference* between consecutive symbols; a
two-ray channel adds to each symbol a scaled, rotated copy of an earlier one,
dragging each constellation point toward positions that depend on its
*history*. The tight eight-point ring blooms into overlapping clusters — the
"ISI-smeared" description on the thread capture's fixture comment is exactly
this. [C4FM]({{ '/reference/c4fm/' | relative_url }}) (P25 Phase 1) is a
four-level FSK: dispersion drags each symbol's frequency toward its
neighbours', closing the four-level
[eye]({{ '/reference/eye-diagram/' | relative_url }}) vertically until the
slicer's thresholds cut through the blur. Same convolution, two dialects of
damage — which is why the levers in this series are built protocol-agnostic,
on complex symbols, and why P25's C4FM path *not* having them yet is a gap
worth its own post (Part 13).

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A two-ray channel smearing a constellation. On the left, a transmitted impulse followed by a smaller delayed echo represents the channel impulse response h. In the middle, the clean eight-point pi over four DQPSK constellation. On the right, the received constellation: each point has bloomed into a cluster of points displaced by scaled rotated copies of neighbouring symbols, overlapping its neighbours. An arrow labeled convolution connects clean to smeared, and a dashed arrow labeled equalizer w points back.">
  <text x="90" y="20" text-anchor="middle" fill="currentColor" font-size="11">channel h</text>
  <line x1="30" y1="150" x2="150" y2="150" stroke="var(--fg-muted)"/>
  <line x1="55" y1="150" x2="55" y2="60" stroke="currentColor"/>
  <circle cx="55" cy="60" r="3" fill="currentColor"/>
  <text x="55" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">direct</text>
  <line x1="105" y1="150" x2="105" y2="105" stroke="var(--accent)"/>
  <circle cx="105" cy="105" r="3" fill="var(--accent)"/>
  <text x="105" y="97" text-anchor="middle" fill="var(--accent)" font-size="9">echo</text>
  <text x="90" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">delay ≈ a symbol</text>
  <circle cx="280" cy="110" r="58" fill="none" stroke="var(--fg-muted)"/>
  <g fill="currentColor">
    <circle cx="338" cy="110" r="4"/><circle cx="321" cy="69" r="4"/><circle cx="280" cy="52" r="4"/><circle cx="239" cy="69" r="4"/>
    <circle cx="222" cy="110" r="4"/><circle cx="239" cy="151" r="4"/><circle cx="280" cy="168" r="4"/><circle cx="321" cy="151" r="4"/>
  </g>
  <text x="280" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9">transmitted: 8 crisp points</text>
  <line x1="352" y1="110" x2="392" y2="110" stroke="currentColor"/><polygon points="392,106 402,110 392,114" fill="currentColor"/>
  <text x="377" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">∗ h</text>
  <circle cx="530" cy="110" r="58" fill="none" stroke="var(--fg-muted)"/>
  <g fill="var(--fg-muted)">
    <circle cx="585" cy="104" r="3"/><circle cx="592" cy="116" r="3"/><circle cx="578" cy="121" r="3"/>
    <circle cx="576" cy="66" r="3"/><circle cx="563" cy="75" r="3"/><circle cx="570" cy="58" r="3"/>
    <circle cx="533" cy="48" r="3"/><circle cx="522" cy="58" r="3"/><circle cx="541" cy="60" r="3"/>
    <circle cx="486" cy="72" r="3"/><circle cx="497" cy="62" r="3"/><circle cx="482" cy="84" r="3"/>
    <circle cx="470" cy="108" r="3"/><circle cx="478" cy="120" r="3"/><circle cx="466" cy="96" r="3"/>
    <circle cx="490" cy="148" r="3"/><circle cx="482" cy="136" r="3"/><circle cx="500" cy="156" r="3"/>
    <circle cx="532" cy="170" r="3"/><circle cx="522" cy="160" r="3"/><circle cx="544" cy="162" r="3"/>
    <circle cx="574" cy="150" r="3"/><circle cx="584" cy="138" r="3"/><circle cx="566" cy="140" r="3"/>
  </g>
  <text x="530" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9">received: clusters bloom &amp; overlap</text>
  <path d="M 470 40 C 420 14 360 14 330 36" fill="none" stroke="var(--accent)" stroke-dasharray="5 3"/>
  <polygon points="336,30 326,39 340,41" fill="var(--accent)"/>
  <text x="400" y="14" text-anchor="middle" fill="var(--accent)" font-size="9">equalizer w: w∗h ≈ delay</text>
</svg>
<figcaption>A two-ray channel is a two-tap FIR: each received symbol carries a scaled, rotated echo of its neighbour, blooming every constellation point into a history-dependent cluster. Because the damage is a convolution, a second filter can remove it.</figcaption>
</figure>

## Invertible vs. not

The linearity test sorts every impairment in this series into two bins, and
the sorting decides which series you're in:

| Impairment | Linear? | Fixable by an equalizer? | Where it's covered |
|---|---|---|---|
| Multipath / simulcast echoes | yes — FIR channel | yes | Parts 4–7 |
| Band-edge group delay | yes — allpass-ish filter | yes | Parts 4–7 |
| Static frequency offset | yes (a rotation ramp) | AFC's job, not the equalizer's | [sdr-internals-07]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}) |
| Additive noise | additive, not filtering | no — equalizers slightly *enhance* it | soft decisions help: Part 8 |
| Phase noise / reciprocal mixing | no — random multiplicative phase | **no** | Part 12, and [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) |
| Clipping / compression / intermod | no — memoryless nonlinearity | **no** | [The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}) |

The bottom two rows deserve emphasis because they are where effort goes to
die. #764's ~10 dB deficit *looked* like something a cleverer decoder should
recover — but reciprocal mixing multiplies the signal by a random phase
process, and no fixed (or slowly-adapted) filter inverts a random process.
Likewise a clipped front end has already discarded amplitude information
irreversibly. The diagnostic discipline for telling these apart from linear
smear — before spending a month on an equalizer that cannot work — is Part 12.
The operator-side prevention (gain staging, overload hygiene) lives in the
concurrent [Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }})
series. The rule of thumb: an equalizer buys you back what the *channel* mixed
together; nothing buys back what the *hardware* threw away.

## The differential domain is not linear

Now the fact that dictates architecture. TETRA's receiver decodes
π/4-DQPSK differentially: the information is in `d[n] = s[n]·conj(s[n−1])`.
It is tempting to equalize `d` directly — the differentials are what the soft
decoder consumes, they're already conveniently rotation-free, and the receiver
already exports them. But substitute the channel model and watch the algebra
break. If `s' = h∗s`, then

`d'[n] = (h∗s)[n] · conj((h∗s)[n−1])`

— every product of the form `h_i·conj(h_j)·s[n−i]·conj(s[n−1−j])` appears. The
channel taps enter as *cross-terms in a product*, not as coefficients of a
convolution. There is no filter `w` acting on `d'` that yields `d`, because
the map from `d` to `d'` isn't a filter at all. The receiver's own doc comment
states the consequence precisely:

```go
// internal/radio/tetra/receiver/receiver.go (shape) — Options
// SymbolSink, when non-nil, receives the RAW post-timing/AFC/equalizer
// complex symbols (before the differential decode) … Unlike the
// SoftSink differential (a nonlinear product s·conj(last), in which the
// channel is no longer a clean convolution), the symbol stream is where a
// linear channel IS a convolution — so it is the input a training-sequence
// equalizer … must train on and equalize per burst.
SymbolSink func(symbols []complex64, baseIdx int)
```

This one fact fans out into three design decisions you'll watch land in later
parts. The **blind** equalizer (`SnapshotCMA`) sits *inside the receiver*,
between symbol-timing recovery and the differential decoder — before the
nonlinearity (Parts 4–5). The **trained** equalizer (`SnapshotLMS`) runs in
the burst extractor, which therefore needs the *raw symbols* carried down to
it in parallel with the dibits — that's the `SymbolSink → StashSymbols` plumbing
(Part 7, architecture in Part 9). And after either equalizer, the
differentials are *re-derived* from equalized symbols rather than patched up
in place. In every case the convolution is inverted where it still *is* a
convolution.

## Reading the damage on the thread capture

Close the loop on our running capture with the sorting rule in hand. Its
in-channel SNR is ~10 dB — low, but a differential QPSK with rate-compatible
convolutional coding should do considerably better than 12% BSCH yield on
noise alone. The constellation is smeared into history-dependent clusters, not
uniformly fattened — noise fattens, ISI *structures*. It peaks at −44 dBFS
with no clipping, ruling out the nonlinear bin. And the same site decodes
~100% at ~18 dB through identical hardware, so the channel, not the rig, is
the variable. Every sign points the same way: a real noise deficit (which
soft decisions will help with in Part 8) *compounded by linear ISI* — which is
the recoverable part, and which is why a blind equalizer will move this
capture from ~12% to ~100% two parts from now. Diagnosis before surgery.

## Where this goes next

We now know the damage is a convolution and know where it must be inverted.
The next question is *how to find the inverse filter when you don't know the
channel* — no pilot, no preamble, nothing but the signal's own statistics.
[Part 4]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
builds the Constant Modulus Algorithm from first principles: the Godard cost
`J = E[(|y|²−R²)²]`, the stochastic-gradient tap update as GopherTrunk's
`cma.go` actually implements it, why constant-envelope PSK suits it — and the
spurious minima that make Part 2's EVM trap a structural hazard rather than
bad luck.

## FAQ

**How do I tell ISI from noise on a live system, without a lab?**
Shape and structure. Noise fattens every constellation cluster uniformly and
isotropically; ISI blooms points into *patterned* sub-clusters (one per
recent-symbol history) and often stretches them anisotropically. On the eye
diagram, noise fuzzes the whole trace while ISI closes the eye with distinct
crossing trajectories. GopherTrunk's constellation and eye panels
([operator-cockpit-08]({{ '/blog/deep-dives/operator-cockpit-08-constellation-eye-symbol/' | relative_url }}))
show both live.

**If ISI is deterministic, why can't the decoder just power through with FEC?**
FEC trades margin against random errors; ISI consumes that margin
systematically, on every symbol, with errors correlated by the channel memory —
the worst case for a convolutional code's error events. Removing structured
distortion *before* the decoder and spending the code on what's left (noise)
is strictly better, which is why the equalizer-plus-soft-decision combination
in this series stacks rather than overlaps.

**Why not equalize the wideband stream once, before channelization?**
Because `h` is different for every carrier — each channel's multipath geometry
and band-edge position differ — and a single wideband filter would need to be
all of those inverses at once. Equalizers in GopherTrunk run per channel, at
symbol rate, where the channel they're inverting is one channel. (The same
physics limits wideband diversity combining to a scalar, a story Part 10
tells.)

**Does a stronger signal fix ISI?**
No — and this is the cleanest way to recognize the regime. ISI scales *with*
the signal: crank the gain and the echoes crank too, so yield plateaus well
below 100% no matter what the S-meter says. If more signal doesn't move yield,
stop shopping for antennas and start suspecting the channel (or Part 12's
nonlinear impairments).

**Is symbol-timing error a form of ISI?**
Sampling off the symbol centre does mix neighbouring symbols through the pulse
shape's skirts, so mistimed sampling *manufactures* ISI from a clean channel.
That's why timing recovery ([sdr-internals-07]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}))
runs before the equalizer, and why the equalizer only gets credit for what
timing can't fix: dispersion that exists even at the perfect sampling instant.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Metrics That Lie — EVM vs CRC Yield]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
· Next →
[Part 4: Blind Equalization — CMA From First Principles]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
