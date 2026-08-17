---
title: "The Analog Edge, Part 7: Antennas for Trunking Bands"
description: Antenna craft for trunked-radio monitoring — which bands the systems actually live in, why antenna gain is a radiation shape rather than free signal, polarization, when height beats gain, and how to choose between a discone, a tuned vertical, and a yagi for a fixed system.
category: tutorials
keywords: scanner antenna trunking, 800 mhz antenna, discone vs vertical, antenna gain pattern, vertical polarization lmr, antenna height vs gain, yagi for trunking, sdr antenna choice, gophertrunk analog edge
tags: [analog-edge, antennas, rf, hardware, propagation, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 7
---

*Part 7 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 6]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }})
finished the desk work: our reader with the marginal system has now staged
gain, ruled out overload, checked the oscillator, and settled the rate — all
without touching hardware. What's left is the hardware, starting where the
signal does. The antenna is the one component that can *add* signal-to-noise
ratio instead of merely preserving it, which is why it's the first dollar
this series tells anyone to spend. It is also — on the evidence of the
project's own weak captures — the dollar most often left unspent.*

> **TL;DR:** Trunked systems live in a handful of bands — VHF-high, UHF,
> and the 700/800/900 MHz cluster — and an antenna is only "good" *at a
> frequency, in a direction*. **Gain is a shape, not a magnitude**: a
> high-gain vertical buys its dB by flattening its pattern toward the
> horizon, which is exactly wrong for a close, elevated tower. Match
> **vertical polarization** (land-mobile is vertical; cross-polarization
> costs real dB), put height before gain until feedline loss eats the
> difference (Part 8), and choose by situation: **discone** to survey
> everything, **tuned vertical** to monitor one band well, **yagi** to
> reach one distant system. The tracker's marginal captures — the −44 dBFS
> TETRA sessions, the −75 dBFS DMR file no decoder could use — are what
> under-antennaed systems look like from the samples' side.

**Key takeaways**

- **The antenna is the only free SNR in the chain.** Every later stage —
  LNA included — adds its own noise while amplifying; a better antenna
  delivers more signal *and* no more noise. Nothing downstream can match
  that trade.
- **Gain redistributes; it doesn't create.** An antenna's dB come from
  squeezing its radiation pattern toward the horizon. Whether that helps
  depends entirely on where your towers are — including the close one a
  flattened pattern can *miss*.
- **Polarization is a silent tax.** Land-mobile trunking is vertically
  polarized; a horizontally-mounted element or a random indoor wire pays a
  large cross-polarization penalty before any other factor is counted.
- **Buy for the band you actually monitor.** A wideband discone hears
  everything adequately; a band-tuned vertical hears your system *well*.
  Survey first, then specialize — the two-antenna strategy this series'
  diversity parts (11–13) will formalize.

## Cheat sheet

| Concern | The short answer | Where to go deeper |
|---|---|---|
| Which antenna to buy | discone to survey, tuned vertical to camp, yagi to reach | [Best scanner antennas]({{ '/best-scanner-antenna/' | relative_url }}), [best SDR antennas]({{ '/best-sdr-antenna/' | relative_url }}) |
| What "dBi/dBd" mean | gain relative to isotropic / dipole (dipole = 2.15 dBi) | [Antenna gain]({{ '/reference/antenna-gain/' | relative_url }}) |
| Wideband survey antenna | the classic broadband vertical monitor antenna | [Discone]({{ '/reference/discone-antenna/' | relative_url }}) |
| High-gain vertical mechanics | stacked elements, flattened elevation pattern | [Collinear]({{ '/reference/collinear-antenna/' | relative_url }}) |
| One distant system | directional gain toward a single azimuth | [Yagi-Uda]({{ '/reference/yagi-uda-antenna/' | relative_url }}) |
| Mounting, masts, grounding | doing the height part safely and legally | [Antenna mast & mounting guide]({{ '/antenna-mast-and-mounting-guide/' | relative_url }}) |
| Whether it worked | level and decode metrics, before vs after | Parts 2–3 of this series — re-run the same measurements |

## In this post

- **The bands trunking lives in** — where to aim the whole exercise.
- **Gain is a shape, not a magnitude** — patterns, and the close-tower trap.
- **Polarization** — the cheapest dB you'll ever recover.
- **Height beats gain — until it doesn't** — the feedline caveat.
- **Choosing: discone, tuned vertical, or yagi** — a decision table.
- **What the tracker's weak captures teach** — the evidence from our side.

## The bands trunking lives in

Antennas are frequency-selective, so the first question is never "which
antenna?" but "which frequencies?" — and trunked radio concentrates into a
few well-known neighborhoods (US-centric here; check your region's band
plan):

| Band | Rough range | Who you'll find there |
|---|---|---|
| VHF-high | 150–174 MHz | statewide/rural public safety, utilities |
| UHF | 450–512 MHz | municipal systems, businesses; DMR and NXDN country |
| 700 MHz | 769–806 MHz | modern P25 public-safety systems |
| 800 MHz | 851–869 MHz | the classic trunking band — P25, EDACS, Motorola legacies |
| 900 MHz | 935–940 MHz | business/industrial trunking, DMR |

Look up your local systems before buying anything: a region whose activity
sits entirely at 700/800 MHz wants a very different antenna than one on
VHF. The wavelength spread is the point — a quarter-wave element is ~48 cm
at 155 MHz and ~9 cm at 860 MHz, and no single passive element is optimal
across that ratio. Wideband antennas exist by accepting compromise
everywhere; tuned antennas excel by refusing it in one place. (For the
propagation background — why 800 MHz is line-of-sight-ish while VHF bends
farther — the [propagation lesson]({{ '/learn/rf-sdr/propagation/' | relative_url }})
is the companion read.)

## Gain is a shape, not a magnitude

Antenna gain is the most mis-sold number in radio. An antenna is passive —
it cannot amplify. A "6 dB gain" vertical delivers more signal from the
horizon *by taking it from everywhere else*: picture the radiation pattern
as a donut around the element that higher gain squashes flatter and wider.
Those dB are real if your towers sit near the horizon. They are *negative*
if a tower is close and elevated — a downtown high-rise site a kilometer
away can sit meaningfully above a flattened pattern's main lobe, and
operators genuinely see a cheap unity-gain whip beat an expensive high-gain
collinear on exactly that one system. The
[antenna gain]({{ '/reference/antenna-gain/' | relative_url }}) entry covers
the units (dBi vs dBd — a half-wave dipole is the 2.15 dBi reference); the
operational rule is simpler: **know your towers' directions and elevations
before buying gain**, because gain is a bet on where the signal comes from.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Two elevation-pattern sketches. Left, a unity-gain vertical shows a plump rounded lobe reaching well above the horizon, and a nearby elevated tower sits inside the lobe. Right, a high-gain collinear shows a flattened lobe hugging the horizon; the same nearby tower now sits above the lobe and is missed, while a distant tower on the horizon is inside it. The caption states gain squeezes the pattern toward the horizon, helping distant sites and risking close elevated ones.">
  <line x1="20" y1="160" x2="320" y2="160" stroke="var(--fg-muted)"/>
  <text x="170" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">unity-gain whip: plump pattern</text>
  <line x1="80" y1="160" x2="80" y2="120" stroke="currentColor" stroke-width="2"/>
  <path d="M 80 158 Q 200 40 300 158" fill="none" stroke="currentColor"/>
  <rect x="196" y="96" width="8" height="34" fill="var(--accent)"/>
  <line x1="188" y1="96" x2="212" y2="96" stroke="var(--accent)"/>
  <text x="200" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">close, elevated tower — heard</text>
  <line x1="360" y1="160" x2="660" y2="160" stroke="var(--fg-muted)"/>
  <text x="510" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">high-gain collinear: flattened pattern</text>
  <line x1="420" y1="160" x2="420" y2="104" stroke="currentColor" stroke-width="2"/>
  <path d="M 420 158 Q 540 118 656 152" fill="none" stroke="currentColor"/>
  <rect x="536" y="96" width="8" height="34" fill="var(--accent)"/>
  <line x1="528" y1="96" x2="552" y2="96" stroke="var(--accent)"/>
  <text x="540" y="88" text-anchor="middle" fill="var(--accent)" font-size="9">same tower — above the lobe</text>
  <polyline points="648,142 652,128 656,142" fill="none" stroke="currentColor"/>
  <text x="640" y="120" text-anchor="end" fill="currentColor" font-size="9">distant site on the horizon — helped</text>
  <text x="340" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the dB on the box are taken from above and below the horizon — gain is a bet on tower geometry</text>
</svg>
<figcaption>Two elevation patterns, one nearby elevated tower: the high-gain antenna's flattened lobe helps the distant site and can miss the close one. Gain redistributes; it never creates.</figcaption>
</figure>

## Polarization

Land-mobile radio is **vertically polarized** — mobile whips point up, so
the infrastructure does too. Receive with a matched vertical element and
you collect the full field; turn the element horizontal and, in the
textbook case, the cross-polarization loss is severe (tens of dB in free
space; scattering in real environments softens it to "merely large").
This is the cheapest audit in the series: telescopic antennas angled for
looks, mag-mounts on their side on a windowsill, random lengths of wire
draped where convenient — each is quietly paying a polarization tax on
every tower at once. Stand the element vertical, with a decent ground
plane under a mag-mount (a cookie sheet genuinely works), before spending
anything. The [polarization]({{ '/reference/polarization/' | relative_url }})
entry has the theory; the audit takes thirty seconds.

## Height beats gain — until it doesn't

At trunking frequencies, propagation is dominated by what stands between
you and the tower. Height fixes obstruction directly: getting an antenna
above the roofline changes the *path*, which routinely dwarfs any
achievable pattern gain — going from an indoor desk antenna to a modest
outdoor mount is frequently a 10–20 dB swing at UHF and up, whereas the
best consumer verticals advertise 6–9 dB against a reference dipole. If
you can only change one thing about a marginal installation, raise the
antenna. Indoors, even a window facing the tower instead of an interior
room can be the difference the decoder needed.

The caveat that pairs with this part: every meter of height is a meter of
coax, and at 800 MHz cheap coax charges by the meter — enough that a long
run of the wrong cable can spend the entire height dividend before it
reaches the tuner. That arithmetic (loss per cable class, and why receive
loss before the first amplifier is especially costly) is precisely
[Part 8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }}),
so plan mast and feedline as one decision — the
[mast & mounting guide]({{ '/antenna-mast-and-mounting-guide/' | relative_url }})
covers the mechanical and safety side.

## Choosing: discone, tuned vertical, or yagi

For a fixed monitoring installation, three archetypes cover nearly every
case:

| | Discone | Tuned vertical (¼-wave / collinear) | Yagi |
|---|---|---|---|
| Bandwidth | very wide (an octave-plus) | one band, done well | narrow, one band |
| Gain | ~unity | unity (¼-wave) to ~6–9 dBd (collinear) | high, in one direction |
| Pattern | omnidirectional | omnidirectional (flatter with gain) | one azimuth lobe |
| Best for | surveying, hunting, many bands at once | camping on your area's main band | one distant/weak system |
| Watch out | mediocre everywhere by design | close-elevated-tower trap above | must be aimed; everything else fades |

The strategy that falls out: **survey wide, then camp tuned.** A discone
plus [the hunt]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
tells you what's actually receivable at your site; once you know which
system you care about, a vertical tuned to its band converts that
discovery into decode margin. The yagi is the specialist's move — one
distant system, direction known, everything else sacrificed. If you run
more than one dongle, run more than one antenna philosophy at once (the
[multi-dongle guide]({{ '/multi-dongle-sdr-setup/' | relative_url }}) covers
the plumbing): discone on the hunting dongle, tuned vertical on the
production system. Specific current models are kept in the
[scanner antenna]({{ '/best-scanner-antenna/' | relative_url }}) and
[SDR antenna]({{ '/best-sdr-antenna/' | relative_url }}) guides rather than
here, so this post can age gracefully.

## What the tracker's weak captures teach

This series keeps its claims tied to the project's evidence, so: what does
an under-antennaed system look like from the samples' side? Like the
captures our decoders keep meeting. The TETRA control-channel sync-loss
investigation ran on captures peaking around **−44 dBFS** — weak enough
that the marginal-SNR regime caused hard sync losses, and the fix notes
say it plainly: the equalizer mitigates a weak front end *but the residual
condition is RF/gain/antenna — raise the signal level too*. The DMR
two-slot verification from Part 1 is still parked on a **−75 dBFS** capture
with no recoverable frame sync at all. In both cases sophisticated software
(equalizers, soft decision) recovered real margin, and in both cases the
notes end by pointing back across Part 1's line. The decoder can only be as
good as the samples — and the antenna is where the samples are born. The
[antennas lesson]({{ '/learn/rf-sdr/antennas/' | relative_url }}) makes a
good deeper companion to this whole part.

## Where this goes next

A better antenna earns you dB at the top of the mast; the feedline decides
how many survive the trip down. [Part 8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})
follows the coax — loss per hundred feet by cable class at 800 MHz, why
every adapter is a small tax, the SMA/BNC/N/F connector ecosystems, and why
a dB lost *before* the first amplifier is the most expensive dB in the
whole chain.

## FAQ

**Will a better antenna fix my garbled audio?**
If the problem is signal-starvation, it's the single most effective fix
available — and Parts 2–3 give you the before/after instruments to prove
it (level regime, decode error rate, demod SNR). If your levels are already
healthy and the decode error rate is low, your problem is elsewhere in this
series, and a new antenna will disappoint. Measure first.

**Is the antenna that came with my dongle good enough?**
For strong local systems, sometimes. Structurally it's a short, often
poorly-grounded whip at desk height — the polarization, ground-plane, and
height mistakes bundled together. It's fine for *proving the software
works*; treat it as the baseline the real antenna gets measured against.

**Can one antenna cover VHF and 800 MHz?**
A discone genuinely can, at the price of being merely adequate at both —
that's its design contract. What can't cover both well is a *tuned*
antenna: a vertical cut for 155 MHz is far off-resonance at 860 MHz. If
you monitor two distant bands seriously, that's two antennas (and ideally
two dongles) — not one compromise.

**Do amplified ("active") antennas help?**
They move Part 9's problem into the antenna housing: an amplifier helps
only if it comes *before* significant loss and *after* adequate
selectivity, and a wideband amp at the antenna also amplifies every pager
and broadcast blaster in view (Part 4). Passive antenna + separate,
chosen LNA where the math says it belongs (Part 9's whole subject —
funnel: [best SDR LNAs]({{ '/best-sdr-lna/' | relative_url }})) beats an
opaque bundled amp you can't reason about.

**How do I verify a new antenna actually improved things?**
Re-run the Part 2/Part 3 measurements you already know: same channel, same
gain staging procedure, compare `iq_power_dbfs` regime, clip ratio, decode
error rate at the chosen rung, and demod SNR on a replayed capture. A real
upgrade shows up as decode-quality improvement at *equal or lower* gain —
if you had to raise gain to see a difference, you measured the knob, not
the antenna.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Sample Rate — The Decode Path Doesn't Care; the Front End Does]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }})
· Next →
[Part 8: Feedline & Connectors — Where dB Go to Die]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }})
