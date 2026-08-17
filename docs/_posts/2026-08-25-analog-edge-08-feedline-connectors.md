---
title: "The Analog Edge, Part 8: Feedline & Connectors — Where dB Go to Die"
description: Why the coax between your antenna and your SDR quietly costs more decode margin than any software setting — published loss figures per cable class at 800 MHz, the real price of every adapter, and why receive-side loss is worse than the half-your-power framing suggests.
category: tutorials
keywords: coax loss 800 mhz, rg-58 vs lmr-400, sdr feedline loss, coax cable for scanner antenna, sma bnc n connector, adapter loss, receive noise figure, feedline noise figure, gophertrunk analog edge
tags: [analog-edge, coax, feedline, connectors, rf, hardware]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 8
---

*Part 8 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system — the part no software update can fix.
[Part 7]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }}) put a
proper antenna on the mast and made the case that height beats gain — up to a
point. This part is that point. Our running reader — hardware scanner clean,
GopherTrunk garbled, on the same antenna — has forty feet of thin coax between
that antenna and the dongle, and this part is about what those forty feet cost.
The decoder can only be as good as the samples, and the samples can only be as
good as what survives the feedline.*

> **TL;DR:** Coax loss is frequency-dependent, cable-class-dependent, and — on
> receive — **worse than it looks**, because every dB of loss *ahead of the
> first amplifier* adds directly to the system's
> [noise figure]({{ '/reference/noise-figure/' | relative_url }}). At 800/900 MHz,
> RG-58 runs roughly **16 dB per 100 ft** while LMR-400 runs about **4 dB** —
> a 40 ft run is the difference between ~6.5 dB and ~1.6 dB gone before the
> tuner sees anything. Every adapter in the chain is another ~0.1–0.3 dB tax.
> None of this shows up in any GopherTrunk log line, because by the time the
> software runs, the dB are already dead.

**Key takeaways**

- **Receive loss is noise-figure loss, not just power loss.** "3 dB is half your
  power" undersells it: 3 dB of coax before any amplification means every weak
  signal arrives 3 dB closer to the noise floor, permanently.
- **Cable class dominates everything else in the feedline.** The spread between
  a thin pigtail and LMR-400 at 850 MHz is nearly a factor of eight in dB per
  foot — no connector choice or careful routing recovers that.
- **Adapters are a small tax paid per joint, forever.** A barrel, a
  gender-changer, and an SMA-to-BNC stack is easily half a dB — and a stack of
  adapters is also a stack of failure points that intermittent faults hide in.
- **You cannot see feedline loss in software.** A lossy run just looks like a
  weaker band. The fix for "everything is 6 dB down" is never in `config.yaml` —
  which is exactly why this series exists.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Cable loss classes | dB/100 ft by cable family, rises with frequency | table below; [coaxial cable]({{ '/reference/coaxial-cable/' | relative_url }}) |
| Receive-side rule | pre-LNA loss adds dB-for-dB to system NF | [noise figure]({{ '/reference/noise-figure/' | relative_url }}) (Friis) |
| Connector families | SMA / BNC / N / F / UHF ecosystems | [SMA]({{ '/reference/sma-connector/' | relative_url }}), [N-type]({{ '/reference/n-type-connector/' | relative_url }}), [PL-259]({{ '/reference/uhf-connector-pl259/' | relative_url }}) |
| Adapter strategy | fewest joints; one good pigtail beats a stack | [SMA adapter kit]({{ '/reference/sma-adapter-kit/' | relative_url }}) |
| Buying guidance | which coax and connectors for a GopherTrunk rig | [SDR cables & connectors guide]({{ '/sdr-cables-and-connectors/' | relative_url }}) |
| The next lever | when to amplify instead of just losing less | Part 9 ([LNA]({{ '/reference/low-noise-amplifier/' | relative_url }})) |

## In this post

- **The receive-side rule** — why 3 dB of coax is worse than half your power.
- **The cable classes** — published loss figures at trunking frequencies.
- **Connectors and adapters** — the families, and the per-joint tax.
- **Impedance, 50 Ω and 75 Ω** — what RG-6 does and doesn't cost you.
- **A worked budget** — our reader's forty-foot run, in numbers.

## The receive-side rule

The framing everyone learns first is transmit-side: *3 dB of coax loss means
half your power reaches the antenna.* True, and on receive it sounds almost
tolerable — signals are tens of dB above the floor, what's 3 dB?

The receive-side truth is harsher. Loss placed **before the first
amplification stage** doesn't just shrink the signal; it raises the whole
system's [noise figure]({{ '/reference/noise-figure/' | relative_url }})
dB-for-dB. A passive 6 dB attenuation ahead of the tuner is mathematically a
6 dB noise figure sitting in front of whatever noise figure the tuner already
had — the Friis cascade formula says the first stage dominates, and a lossy
cable is a first stage with gain of *minus* six. Every weak signal on the band
arrives 6 dB closer to the noise floor, and no gain knob downstream can undo
it: turning up the tuner amplifies the noise floor and the signal together.

This is why the series keeps insisting the fix is not in software.
[Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
showed an operator raising gain 65 → 82 dB to chase a software constant;
feedline loss is the physical version of the same trap — the dB you lose here
are gone before any number GopherTrunk can display is even measured. A
marginal TETRA control channel that decodes at 18 dB in-channel SNR and drops
lock at 10 dB (the split we measured in the sync-loss captures) lives or dies
inside exactly the margin a bad feedline eats.

## The cable classes

Loss per length rises with frequency, and trunking bands sit at the expensive
end — 700/800/900 MHz costs roughly three times what VHF does on the same
cable. The table uses typical published manufacturer figures at ~900 MHz,
rounded; treat them as planning numbers, not datasheet gospel (actual loss
varies by make and by how the cable has been treated).

| Cable | Loss / 100 ft @ ~900 MHz | Loss / 100 ft @ ~150 MHz | Verdict for a trunking feed |
|---|---|---|---|
| RG-174 / RG-316 (thin pigtail) | ~30 dB | ~10 dB | inches only — a pigtail, never a run |
| RG-58 | ~16 dB | ~6 dB | desk patch leads only |
| RG-8X | ~11 dB | ~4.5 dB | short runs, portable rigs |
| RG-6 (75 Ω TV coax) | ~6 dB | ~2.7 dB | surprisingly good — see below |
| LMR-240 | ~8 dB | ~3 dB | good mid-weight choice |
| LMR-400 | ~4 dB | ~1.5 dB | the default for a mast run |
| LMR-600 | ~2.5 dB | ~1 dB | long runs, heavy and stiff |

Two readings of that table matter. First, the **spread**: at 850 MHz, the same
forty feet costs ~6.5 dB in RG-58 and ~1.6 dB in LMR-400. That 5 dB delta is
larger than most antenna upgrades buy you, at a fraction of the price of a
better SDR. Second, the **pigtail row**: RG-174 and RG-316 exist to make the
last few inches to an SMA jack flexible, and they are fine at that length —
but a 3 ft RG-174 "extension" at 850 MHz is already about a dB, which is why
the [cables guide]({{ '/sdr-cables-and-connectors/' | relative_url }}) keeps
repeating *pigtails are measured in inches*.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Cumulative feedline loss versus run length at 850 megahertz for two cable classes: RG-58 loses about 16 decibels per hundred feet, crossing 6 decibels around forty feet, while LMR-400 loses about 4 decibels per hundred feet and stays under 2 decibels at the same length; the gap between the two lines at forty feet is about 5 decibels of decode margin">
  <line x1="55" y1="20" x2="55" y2="165" stroke="var(--fg-muted)"/>
  <line x1="55" y1="165" x2="650" y2="165" stroke="var(--fg-muted)"/>
  <text x="14" y="30" fill="var(--fg-muted)" font-size="9">loss</text>
  <text x="14" y="42" fill="var(--fg-muted)" font-size="9">(dB)</text>
  <text x="600" y="183" fill="var(--fg-muted)" font-size="9">run (ft)</text>
  <text x="46" y="169" fill="var(--fg-muted)" font-size="9">0</text>
  <text x="285" y="181" fill="var(--fg-muted)" font-size="9">40</text>
  <text x="520" y="181" fill="var(--fg-muted)" font-size="9">80</text>
  <line x1="292" y1="20" x2="292" y2="165" stroke="var(--fg-muted)" stroke-dasharray="3,4"/>
  <polyline points="55,165 173,127 292,88 411,50 530,24" fill="none" stroke="currentColor"/>
  <text x="440" y="38" fill="currentColor" font-size="10">RG-58 (~16 dB/100 ft)</text>
  <polyline points="55,165 173,155 292,146 411,136 530,127 650,117" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="470" y="112" fill="var(--accent)" font-size="10">LMR-400 (~4 dB/100 ft)</text>
  <line x1="292" y1="88" x2="292" y2="146" stroke="var(--accent)"/>
  <text x="300" y="122" fill="var(--accent)" font-size="10">~5 dB at 40 ft</text>
  <text x="352" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the gap is decode margin lost before any software runs — no gain setting downstream recovers it</text>
</svg>
<figcaption>Cumulative loss at ~850 MHz for a forty-to-eighty-foot mast run: cable class alone swings the budget by ~5 dB — more than most antenna upgrades.</figcaption>
</figure>

## Connectors and adapters

Connector families are ecosystems, and the practical goal is to live in as few
of them as possible:

| Family | Where you meet it | At 800 MHz |
|---|---|---|
| [SMA]({{ '/reference/sma-connector/' | relative_url }}) | nearly every SDR (RTL-SDR, Airspy, HackRF) | excellent; small, torque-sensitive |
| BNC | scanners, lab gear, older antennas | fine; quick-connect, less weatherproof |
| [N-type]({{ '/reference/n-type-connector/' | relative_url }}) | base antennas, LMR-400 terminations, LNAs | excellent; the outdoor standard |
| F | TV / RG-6 ecosystem | fine electrically; 75 Ω world |
| [UHF / PL-259]({{ '/reference/uhf-connector-pl259/' | relative_url }}) | ham gear, CB-era antennas | avoid — non-constant impedance, worst of the lot up here |

Each mated pair — every adapter, barrel, and gender-changer — costs roughly
0.1–0.3 dB at UHF as a rule of thumb. One adapter is nothing. The failure mode
is the *stack*: antenna (N) → N-to-UHF barrel → PL-259 jumper → SO-239-to-BNC
→ BNC-to-SMA → dongle is five joints, plausibly over half a dB, and — worse
than the loss — five places for a cold joint or loose coupling to produce the
intermittent, weather-correlated flakiness that gets misfiled as a software
bug. The discipline: terminate the run in the connector your radio actually
wears, keep one good [adapter kit]({{ '/reference/sma-adapter-kit/' | relative_url }})
for experiments, and solve any permanent mismatch with a purpose-made pigtail
rather than a chain.

## Impedance: the 75 Ω question

RG-6 sits oddly in the table: it's 75 Ω TV coax, our gear is 50 Ω, and it
still often wins. The mismatch between 75 and 50 Ω costs a fixed ~0.18 dB of
reflection loss on receive — noise, essentially, next to the *per-foot* loss
difference it buys back: RG-6 at ~6 dB/100 ft beats RG-58's ~16 by a wide
margin, it's cheap everywhere, and quad-shield versions are well shielded. For
a receive-only GopherTrunk feed, a long RG-6 run with an F-to-SMA pigtail at
the radio end is a legitimate budget build; the
[cables guide]({{ '/sdr-cables-and-connectors/' | relative_url }}) covers the
parts. (Transmitting is a different story — this series is receive-only.)

## A worked budget: the forty-foot run

Back to our reader. The [Part 7]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})
antenna went up on the mast, and the hardware scanner — connected through
twenty feet of the same old RG-58 — got cleaner. GopherTrunk, at the end of
**forty** feet of RG-58 plus a three-adapter stack, barely moved. The budget
explains it:

| Segment | Loss @ ~850 MHz |
|---|---|
| 40 ft RG-58 | ~6.5 dB |
| 3 adapters (N→BNC→SMA stack) | ~0.5 dB |
| 3 ft RG-174 pigtail at the desk | ~1 dB |
| **Total, before the tuner** | **~8 dB** |

Eight dB of noise figure before the first transistor. Replacing the run with
LMR-400 (~1.6 dB), terminating it in a single N-to-SMA pigtail (~0.3 dB with
one joint), and retiring the desk pigtail brings the same path to ~2 dB — a
**6 dB** improvement, which in [Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})'s
terms moves a marginal channel from "drops lock under load" to "decodes."
That's the entire gap between the 10 dB and 18 dB in-channel SNR regimes we
measured on real TETRA sync-loss captures, bought with a spool of cable. No
config key was harmed.

## Where this goes next

Losing less is the first lever; the second is *adding gain in the right
place*. [Part 9]({{ '/blog/tutorials/analog-edge-09-filters-lnas/' | relative_url }})
covers filters and LNAs — why an amplifier helps only when it sits *before*
the loss, why the same LNA that rescues a quiet site wrecks a hot one, and the
order-of-operations rules that keep Part 4's intermod problems from coming
back with 20 dB more gain behind them.

## FAQ

**My run is only ten feet. Does any of this matter?**
Less, but not zero: 10 ft of RG-58 at 850 MHz is ~1.6 dB plus your adapters.
If your system decodes cleanly, spend nothing. If you're marginal — and this
series' reader is — 2 dB is real margin, and the pigtail-stack audit is free.

**Can GopherTrunk detect feedline loss?**
No, and nothing can from the software side — a lossy feedline is
indistinguishable from a quieter band. What you *can* do is compare: the same
antenna through a short known-good jumper versus through your installed run,
watching per-channel SNR or decode quality
([the gain-sweep method]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})).
A constant offset between the two is your feedline, in dB.

**Is "low-loss" coax ever a bad idea?**
Only practically: LMR-400 is stiff, heavy, and hates tight bends, so it's
wrong for the last flexible half-meter to the radio. The standard pattern is a
low-loss trunk run terminated in N, then a short flexible pigtail (RG-316,
inches) to SMA. One joint, best of both.

**Why is PL-259 singled out?**
The UHF connector predates the idea of constant impedance — through the
connector body the line isn't 50 Ω, which matters little at HF/VHF and
increasingly at 800 MHz. It's also not weatherproof by design. On a trunking
feed there is always a better choice (N outdoors, SMA at the radio).

**Should I trust the printed loss figures on cheap cable?**
Trust the class, verify the sample: no-name "RG-58" varies widely, and a
damaged or ancient run can be far worse than any table. The A/B in the second
FAQ answers what *your* cable does, which is the only number that matters —
tables are for choosing what to buy, measurements are for judging what you have.

**Does weather really change coax loss?**
Water does. Coax that has wicked moisture through an unsealed outdoor joint
can double its loss and never recover — foam-dielectric cables are especially
vulnerable. Seal outdoor connectors (self-amalgamating tape), drip-loop the
run, and treat a system that got worse after a storm as a feedline suspect
first ([mounting guide]({{ '/antenna-mast-and-mounting-guide/' | relative_url }})).

## Series navigation

**Part 8 of 14** · ←
[Part 7: Antennas for Trunking Bands]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})
· Next →
[Part 9: Filters & LNAs — Adding Gain Without Adding Garbage]({{ '/blog/tutorials/analog-edge-09-filters-lnas/' | relative_url }})
