---
title: "The Analog Edge, Part 9: Filters & LNAs — Adding Gain Without Adding Garbage"
description: When a low-noise amplifier actually helps, why its position in the chain matters more than its gain figure, how bandpass and FM-notch filters buy back headroom on a hot band, and the decision rules for ordering filter, LNA, and coax so you don't amplify your own problems.
category: tutorials
keywords: sdr lna placement, low noise amplifier antenna, friis noise figure, fm broadcast notch filter, sdr bandpass filter, bias tee sdr, lna overload intermod, filter before or after lna, gophertrunk analog edge
tags: [analog-edge, lna, filters, noise-figure, rf, hardware]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 9
---

*Part 9 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system. [Part 8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})
audited our marginal reader's feedline and found ~8 dB dying in forty feet of
RG-58 and an adapter stack. New cable clawed back six of them. This part is
about the next lever — active gain — and its sharp edge: an LNA in the wrong
place amplifies nothing useful, and an LNA on a hot band manufactures the
exact intermod garbage [Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
taught you to recognize. Gain is easy; *clean* gain is placement and
filtering.*

> **TL;DR:** An [LNA]({{ '/reference/low-noise-amplifier/' | relative_url }})
> helps exactly in proportion to how much loss comes **after** it, because the
> Friis cascade makes the first stage's
> [noise figure]({{ '/reference/noise-figure/' | relative_url }}) dominate the
> system. Mast-mounted LNA ahead of the coax: the run's loss nearly vanishes
> from the noise budget. Desk-mounted LNA after the coax: you amplify signal
> and accumulated noise together and gain almost nothing. The tax is dynamic
> range — every dB of LNA gain is a dB less headroom against the FM
> broadcast / pager blowtorches, so on a hot band the LNA needs a
> [filter]({{ '/sdr-filters/' | relative_url }}) riding with it. Power rides
> the coax via a [bias tee]({{ '/reference/bias-tee/' | relative_url }})
> (`bias_tee: true` on the device block for SDRs that provide one).

**Key takeaways**

- **Placement beats specification.** A mediocre LNA at the antenna outperforms
  an excellent LNA at the desk, because loss *after* amplification barely
  counts while loss *before* it counts dB-for-dB.
- **Gain without filtering is a bet that your band is quiet.** An LNA lifts
  every FM broadcast, pager, and cellular carrier in its passband along with
  your target; if the sum approaches the front end's limits, you've traded
  noise floor for intermod.
- **Filters spend dB to buy headroom.** A bandpass or FM-notch filter has
  insertion loss — position decides whether that loss costs noise figure
  (before the LNA) or almost nothing (after it), and overload decides which
  you need.
- **The decode-quality sweep is still the referee.** Whether an LNA/filter
  combination netted out positive is answered the same way as gain staging in
  [Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }}):
  by decode error rate, never by how loud the waterfall got.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LNA fundamentals | sub-1 dB NF first stage; sets the cascade | [low-noise amplifier]({{ '/reference/low-noise-amplifier/' | relative_url }}), [best SDR LNAs]({{ '/best-sdr-lna/' | relative_url }}) |
| Friis cascade | why stage 1 dominates system NF | [noise figure]({{ '/reference/noise-figure/' | relative_url }}) |
| FM broadcast notch | kills the 88–108 MHz blowtorch before it intermods | [FM broadcast filter]({{ '/reference/fm-broadcast-filter/' | relative_url }}), [SDR filters guide]({{ '/sdr-filters/' | relative_url }}) |
| Band-specific filtering | bandpass / cavity for one service band | [cavity filter]({{ '/reference/cavity-filter/' | relative_url }}) |
| Powering a mast LNA | DC up the coax | [bias tee]({{ '/reference/bias-tee/' | relative_url }}); `bias_tee` in the `sdr` device blocks (`config.example.yaml`) |
| Judging the result | decode error rate across a gain sweep | [autogain, The Hunt Part 5]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }}) |

## In this post

- **Why the first stage wins** — the Friis rule in operator terms.
- **LNA at the antenna vs at the dongle** — the same part, two outcomes.
- **The overload tax** — when gain makes decoding worse.
- **Filters: notch, bandpass, and where they go** — order-of-operations rules.
- **Bias tees and powering** — DC up the coax without surprises.

## Why the first stage wins

The Friis noise cascade says the system noise figure is dominated by the first
stage, with every later stage's contribution divided by the gain in front of
it. You don't need the formula to use the consequence:

- **Loss before the first amplifier counts in full.** Part 8's 6.5 dB of RG-58
  ahead of a bare dongle is 6.5 dB of system noise figure.
- **Loss after ~20 dB of low-noise gain barely counts.** The same 6.5 dB of
  coax *behind* a 20 dB LNA contributes a fraction of a dB to the cascade.

That asymmetry is the entire placement argument. A typical SDR-grade LNA has a
noise figure under 1 dB and ~20 dB of gain; put it at the antenna and the
system noise figure becomes "about 1 dB, plus a little" almost regardless of
the feedline behind it. The coax run stops being a noise problem and becomes a
mere plumbing problem.

## LNA at the antenna vs at the dongle

Same amplifier, two different systems:

| Configuration | Approx. system NF | What actually happened |
|---|---|---|
| antenna → 40 ft RG-58 (~6.5 dB) → tuner (NF ~6 dB) | ~12.5 dB | baseline: coax + tuner both count in full |
| antenna → 40 ft RG-58 → **LNA at desk** → tuner | ~7.5 dB | the coax's 6.5 dB still leads the cascade |
| antenna → **LNA at mast** → 40 ft RG-58 → tuner | ~1.5 dB | coax and tuner both hide behind the LNA's gain |

The desk-mounted LNA looks busy — the waterfall brightens, dBFS rises — but
most of what it amplified was a signal already degraded by the run. Six dB of
the improvement you paid for never existed. The mast-mounted unit changes the
system class. This is also the honest way to read
[Part 8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})'s
budget: replacing cable and adding a mast LNA attack the same dB from two
sides, and the LNA is usually the bigger single win — *if* the band lets you
spend the headroom.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Two receive chains compared. Top chain: antenna into forty feet of lossy coax, then an LNA at the desk, then the tuner — the loss sits before the amplifier and the system noise figure stays high. Bottom chain: antenna into an LNA at the mast, then the same coax, then the tuner — the loss sits after the amplifier and the system noise figure is set by the LNA alone, about 1.5 dB.">
  <text x="10" y="30" fill="var(--fg-muted)" font-size="10">LNA at the desk</text>
  <rect x="10" y="40" width="88" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="54" y="62" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="98" y1="58" x2="128" y2="58" stroke="currentColor"/><polygon points="128,54 138,58 128,62" fill="currentColor"/>
  <rect x="138" y="40" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="213" y="56" text-anchor="middle" fill="currentColor" font-size="10">40 ft coax</text>
  <text x="213" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">−6.5 dB, counts in full</text>
  <line x1="288" y1="58" x2="318" y2="58" stroke="currentColor"/><polygon points="318,54 328,58 318,62" fill="currentColor"/>
  <rect x="328" y="40" width="96" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="376" y="56" text-anchor="middle" fill="currentColor" font-size="10">LNA</text>
  <text x="376" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">too late</text>
  <line x1="424" y1="58" x2="454" y2="58" stroke="currentColor"/><polygon points="454,54 464,58 454,62" fill="currentColor"/>
  <rect x="464" y="40" width="88" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="508" y="62" text-anchor="middle" fill="currentColor" font-size="10">tuner</text>
  <text x="600" y="62" fill="var(--fg-muted)" font-size="10">NF ≈ 7.5 dB</text>
  <text x="10" y="128" fill="var(--fg-muted)" font-size="10">LNA at the mast</text>
  <rect x="10" y="138" width="88" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="54" y="160" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <line x1="98" y1="156" x2="128" y2="156" stroke="currentColor"/><polygon points="128,152 138,156 128,160" fill="currentColor"/>
  <rect x="138" y="138" width="96" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="186" y="154" text-anchor="middle" fill="var(--accent)" font-size="10">LNA first</text>
  <text x="186" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NF &lt;1 dB, +20 dB</text>
  <line x1="234" y1="156" x2="264" y2="156" stroke="currentColor"/><polygon points="264,152 274,156 264,160" fill="currentColor"/>
  <rect x="274" y="138" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="349" y="154" text-anchor="middle" fill="currentColor" font-size="10">40 ft coax</text>
  <text x="349" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">−6.5 dB, hidden by gain</text>
  <line x1="424" y1="156" x2="454" y2="156" stroke="currentColor"/><polygon points="454,152 464,156 454,160" fill="currentColor"/>
  <rect x="464" y="138" width="88" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="508" y="160" text-anchor="middle" fill="currentColor" font-size="10">tuner</text>
  <text x="600" y="154" fill="var(--accent)" font-size="10">NF ≈ 1.5 dB</text>
  <text x="340" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same parts, ~6 dB apart — placement, not specification, sets the system noise figure</text>
</svg>
<figcaption>The Friis rule in one picture: loss ahead of the LNA counts in full; loss behind it hides under the gain. The mast chain wins by ~6 dB with identical parts.</figcaption>
</figure>

## The overload tax

Here is the trade nobody prints on the LNA's box: **every dB of gain ahead of
the tuner is a dB less headroom inside it.** The LNA doesn't know which
carrier you want. It lifts the 100 kW FM broadcast transmitter at 98.7, the
paging blowtorch at 152.48, and the neighbor's cellular uplink right along
with your 851 MHz control channel — and
[Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
showed what happens when the *sum* gets big: intermod products that look like
phantom carriers, a rail-pinned ADC behind a deceptively normal FFT
([the nineteen-dibits postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})),
and a decode error rate that *rises* with gain on the far side of
[Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})'s
U-curve.

The operator-level rule: **an LNA is a rural instrument by default.** On a
quiet band it's nearly free noise figure. In an urban RF environment it must
arrive with a filter, and you should expect to *reduce* the tuner's own gain
after installing it — re-run the gain ladder from Part 3 and let decode
quality, not dBFS, pick the new operating point. Remember the standing rule
from [#764](https://github.com/MattCheramie/GopherTrunk/issues/764): neither
of those captures clipped, and the front end was still the problem — "no
clipping" does not clear an overloaded or noisy chain, and a brighter
waterfall proves nothing.

## Filters: notch, bandpass, and where they go

A filter spends a little in-band insertion loss (typically 0.5–3 dB depending
on type) to remove out-of-band power before it can do harm. The two workhorses
for a trunking rig, both covered with buying picks in the
[SDR filters guide]({{ '/sdr-filters/' | relative_url }}):

- **FM broadcast notch** ([reference]({{ '/reference/fm-broadcast-filter/' | relative_url }})) —
  kills 88–108 MHz, which is the single most common overload source feeding a
  wideband antenna. Cheap, nearly universal benefit for anything above VHF.
- **Bandpass for your service band** — e.g. a 700–900 MHz pass for a trunking
  rig, or a proper [cavity filter]({{ '/reference/cavity-filter/' | relative_url }})
  when one specific neighbor is the problem. Narrower is better protection and
  less flexibility.

Position is a decision rule, not a dogma:

| Situation | Order | Why |
|---|---|---|
| Quiet band, weak target | antenna → LNA → filter → coax | filter's insertion loss hides behind the LNA's gain; NF stays minimal |
| Hot band (FM/pager towers in sight) | antenna → **filter** → LNA → coax | the LNA itself is what overloads; it must be protected even at the cost of NF |
| Overload only at the tuner, LNA clean | antenna → LNA → coax → filter → tuner | spend the loss where it's cheapest |

The first row is the textbook default; the second is the one that saves urban
installs. If you're unsure which regime you're in, the test from Part 4
applies: insert 10 dB of [attenuation]({{ '/reference/attenuator/' | relative_url }})
at the tuner — if signals *appear* or decode *improves*, you're overloaded
somewhere, and the filter goes in front of whatever stage the attenuator just
rescued.

## Bias tees and powering

A mast-mounted LNA needs DC, and running a second cable up the mast defeats
the purpose. A [bias tee]({{ '/reference/bias-tee/' | relative_url }}) injects
supply voltage onto the coax's center conductor; the LNA end extracts it, and
the RF rides through both ways. Many SDRs can source this directly — the
device blocks in `config.example.yaml` expose it as `bias_tee: true`
(RTL-SDR Blog [V3/V4]({{ '/rtl-sdr-blog-v3-vs-v4/' | relative_url }}) have one
built in; on `sdr.soapy_remote` devices it's best-effort, driver-dependent). Two cautions: a DC short anywhere in the run (some antennas
are DC-grounded by design) will make the dongle fold back its supply — check
the antenna's spec before enabling it; and an inline component that *doesn't*
pass DC (most filters don't) must sit on the radio side of the injection
point, not between the bias tee and the LNA.

## Where this goes next

The chain from antenna to ADC is now as good as we can make it: the right
antenna (Part 7), low-loss feed (Part 8), clean gain in the right order (this
part). Before we spend money on a *second* antenna, we need the skill that
turns any remaining mystery into evidence:
[Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
covers capture discipline — the formats, sidecars, and tap points that let a
"sounds bad" complaint become a replayable file, which is the tracker's most
repeated request and the reason half its hard bugs ever got solved.

## FAQ

**Do I need an LNA at all with a modern SDR?**
Maybe not. If your feedline is short and low-loss and your decode is clean, an
LNA buys you nothing but overload risk. The LNA's case is strongest with long
runs (it neutralizes the coax) and weak signals. Fix the feedline first —
Part 8 is cheaper than [any amplifier]({{ '/best-sdr-lna/' | relative_url }}).

**How much LNA gain should I buy?**
Enough to bury the losses behind it — 15–20 dB covers any sane feedline. More
gain than that subtracts straight from overload headroom without improving
noise figure, since the cascade is already dominated by the first stage.

**Filter first or LNA first?**
Quiet band: LNA first (its gain hides the filter's insertion loss). Hot band:
filter first, because the LNA is the stage being overloaded and a fried noise
budget beats a spur farm. When in doubt, run the attenuator test from Part 4
and see which stage the overload lives in.

**Will GopherTrunk tell me my LNA is helping?**
Indirectly and honestly: sweep tuner gain and compare decode error rate before
and after, the same way [autogain]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
scores gains. Expect the optimal tuner gain to *drop* after adding an LNA. If
error rate at the new optimum didn't improve, the LNA is amplifying a problem —
or sitting in the wrong place.

**Can I power a mast LNA through a splitter or a second receiver?**
Treat DC paths explicitly. Splitters, filters, and some antennas block or
short DC, and a bias-tee feed that works through one path can vanish through
another. One injection point, one extraction point, and check every inline
component's DC behavior — "LNA stopped amplifying after I added a filter" is
almost always a starved bias tee.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Feedline & Connectors — Where dB Go to Die]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})
· Next →
[Part 10: Capture Discipline — cfile, cs16, SigMF & Metadata]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
