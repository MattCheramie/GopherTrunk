---
title: "The Analog Edge, Part 14: The Field Checklist — Is It RF or Is It Software?"
description: "The whole series folded into one triage flow — dBFS, clipping, gain sweep, rate A/B, capture, coherence — a decision tree for splitting radio problems from software problems, the order to spend hardware money in, and the send-off for the marginal system that started it all."
category: tutorials
keywords: sdr troubleshooting checklist, rf or software problem, decode triage flowchart, sdr gain sweep checklist, iq capture debugging, scanner reception troubleshooting, hardware upgrade order sdr, gophertrunk analog edge
tags: [analog-edge, checklist, troubleshooting, rf, workflow]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 14
---

*Part 14 — the finale — of **The Analog Edge**. Thirteen posts ago we met a
reader whose hardware scanner decoded a system cleanly while GopherTrunk, on
the same antenna, produced garble — and we made a promise: the causes were on
the analog side, and they could be eliminated one measurement at a time.
[Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
delivered the last instrument, the scale-invariant health number. This part is
the whole series folded into a single field checklist — and the close of that
reader's story. The motto we planted in
[Part 1]({{ '/blog/tutorials/analog-edge-01-where-software-ends/' | relative_url }})
has earned its repetition by now: **the decoder can only be as good as the
samples — and half the hard bugs in the issue tracker were in the samples.***

> **TL;DR:** Triage in a fixed order, cheapest measurement first:
> **dBFS** (headroom, not quality — [Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }}))
> → **clipping/histogram** ([Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }}))
> → **gain ladder scored by decode quality** ([Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }}))
> → **sample-rate A/B** ([Parts 5–6]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }}))
> → **raw capture** ([Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}))
> → **coherence** on diversity rigs ([Parts 11–13]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})).
> The RF-vs-software verdict comes from one experiment: **replay the capture** —
> a symptom that reproduces offline from the file is in the samples (RF side);
> one that doesn't is downstream (software/load side). Spend money in order:
> antenna → feedline → filter/LNA → second device. And when you file an issue,
> file it **with the capture** — that's the difference between an anecdote and
> a fix.

**Key takeaways**

- **The chain has an address for every symptom.** Uniformly weak → antenna/
  feedline (Parts 7–8). Worse as gain rises → overload (Part 4). Rate-dependent
  → front-end cleanliness (Part 5). Intermittent fades → diversity (Part 11).
  Nothing in this list is fixed in `config.yaml`.
- **Replay is the judge.** GopherTrunk's decode is rate-invariant and
  deterministic on a given file — so an offline replay that reproduces the
  failure convicts the samples, and one that decodes cleanly acquits them.
- **Buy hardware in noise-figure order.** Antenna placement, then feedline,
  then LNA/filter, then a second device — each step is cheaper than the next
  and its gain compounds through everything downstream.
- **A capture attached to an issue is a contribution.** Every decoder
  validated this year was closed by an operator's capture becoming a
  regression test — the [decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }})
  page lists what's still waiting.

## Cheat sheet

| Symptom | First suspect | Measure with | Part |
|---|---|---|---|
| Everything weak, all channels | antenna / feedline | dBFS tiles + a known-good jumper A/B | [7]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})–[8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }}) |
| Worse when gain goes up | overload / intermod | sample histogram, [attenuator test]({{ '/reference/sdr-gain-overload/' | relative_url }}) | [4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }}) |
| Decodes at one rate, not another | front-end phase noise | capture at both rates, replay A/B | [5]({{ '/blog/tutorials/analog-edge-05-phase-noise-reciprocal-mixing/' | relative_url }})–[6]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }}) |
| Comes and goes with traffic/weather | fading / feedline moisture | time-of-day log review; diversity | [8]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }}), [11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }}) |
| Diversity not helping | branch health / placement | `coherence`, `branch_phase_deg` log fields | [11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})–[13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }}) |
| CPU warnings, drops under load | downstream consumer, not RF | `decode can't keep up with real time` WARN, `host_drops` | [6]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }}) |
| Anything you can't classify | the samples | **capture it**, replay it | [10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}) |

## In this post

- **The triage ladder** — six measurements in cost order.
- **The verdict experiment** — replay decides RF vs software.
- **The buying order** — where the next dollar does the most work.
- **Filing an issue that gets fixed** — the capture contract.
- **The reader's system, fixed** — the thread, resolved.

## The triage ladder

Run these in order; each step is cheaper than the one after it, and each
step's result tells you whether to continue or stop and fix.

1. **Read dBFS as headroom.** Healthy sits around −25 dBFS peak with no
   clipping; −48 dBFS peak is lean but can still be a perfectly decodable
   signal (one of [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)'s
   captures was), and ~−75 dBFS RMS is a dead input (the DMR investigation's
   unusable capture). dBFS out of range points at gain or a broken feed; dBFS
   *in* range clears nothing else — move on.
2. **Check for rail-pinning.** The sample histogram tells you what the FFT
   hides — the [nineteen-dibits capture]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
   was 50% pinned behind a plausible-looking waterfall. Pinned → turn gain
   down and re-test; the [overload reference]({{ '/reference/sdr-gain-overload/' | relative_url }})
   has the full signature list.
3. **Run the gain ladder, scored by decode.** Sweep gain and watch decode
   error rate — never power — exactly as
   [autogain]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
   does: prefer a lock, then lower error rate, then *lower* gain. While
   you're in the config: `gain: "300"` is 30.0 dB — the
   [tenths-of-dB trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
   is still the cheapest bug in the tracker.
4. **A/B the sample rate.** If the symptom tracks the rate — decodes at
   2.4 MS/s, garbles at 10 — the decode path is acquitted in advance (it
   normalises everything to the per-protocol channel rate) and your front
   end's cleanliness at the higher clock is the suspect. Capture at both
   rates and replay.
5. **Capture the failure.** Same rate, same gain, same antenna, sidecar
   written at capture time ([Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})).
   From here on you're doing science instead of remembering.
6. **On diversity rigs, read the health line.** Dead branch → antenna/gain.
   `coherence` near floor → branches see different signals. Stuck 0.3–0.5 →
   antennas too far apart for the wideband combine. Walking
   `branch_phase_deg` → use `mrc`, not `mrc-static`
   ([Part 12]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})).

## The verdict experiment

Every path above converges on one experiment, because one experiment settles
the question this series is named for:

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Decision tree for splitting RF problems from software problems. Start: capture the failing signal as raw IQ. Replay it offline through gophertrunk replay. If the symptom reproduces from the file, the problem is in the samples — follow the RF branch: check headroom and clipping, then gain staging, then rate cleanliness, then antenna and feedline. If the replay decodes cleanly, the problem is downstream — follow the software and load branch: check CPU warnings, dropped chunks, and configuration, and file an issue with the capture attached.">
  <rect x="250" y="10" width="180" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="26" text-anchor="middle" fill="var(--accent)" font-size="10">capture the failure (raw IQ</text>
  <text x="340" y="39" text-anchor="middle" fill="var(--accent)" font-size="10">+ sidecar, at the failing settings)</text>
  <line x1="340" y1="46" x2="340" y2="70" stroke="currentColor"/><polygon points="336,70 340,80 344,70" fill="currentColor"/>
  <rect x="255" y="80" width="170" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="100" text-anchor="middle" fill="currentColor" font-size="10">replay it offline — does it reproduce?</text>
  <line x1="255" y1="96" x2="150" y2="130" stroke="currentColor"/><polygon points="148,126 142,136 156,133 " fill="currentColor"/>
  <line x1="425" y1="96" x2="530" y2="130" stroke="currentColor"/><polygon points="524,133 538,136 532,126" fill="currentColor"/>
  <text x="185" y="118" fill="var(--fg-muted)" font-size="9">yes → it's in the samples</text>
  <text x="440" y="118" fill="var(--fg-muted)" font-size="9">no → it's downstream</text>
  <rect x="30" y="136" width="230" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="145" y="152" text-anchor="middle" fill="var(--accent)" font-size="10">RF side (Parts 2–9, 11–13)</text>
  <text x="145" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">headroom → clipping → gain ladder →</text>
  <text x="145" y="179" text-anchor="middle" fill="var(--fg-muted)" font-size="9">rate A/B → antenna / feedline / LNA</text>
  <rect x="420" y="136" width="230" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="535" y="152" text-anchor="middle" fill="currentColor" font-size="10">software / load side</text>
  <text x="535" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CPU WARNs, host_drops, config —</text>
  <text x="535" y="179" text-anchor="middle" fill="var(--fg-muted)" font-size="9">file the issue WITH the capture</text>
  <text x="340" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the replay is deterministic on a file — whichever way it answers, you've halved the search space with one command</text>
</svg>
<figcaption>One experiment splits the world: a failure that replays from the file is in the samples; one that doesn't is downstream of them.</figcaption>
</figure>

Replay the capture through the same decode the daemon runs
(`gophertrunk replay -in fail.raw -format cs16 -sample-rate … -protocol …`).
**Reproduces offline** → the problem rode in with the samples: work the RF
branch, and note the offline replay just eliminated CPU, USB, scheduling, and
every live-only variable in one step — this is exactly how #764's deficit was
pinned to the capture itself. **Decodes cleanly offline** → the samples are
fine and the live system is dropping or starving something: look for the
`ccdecoder: decode can't keep up with real time` WARN and `soapyremote: SDR
overruns … host_drops` (both are *downstream* signals — the consumer stopped
draining, not the driver), and check what got slower recently. Either way,
you now hold the file that proves it.

## The buying order

When the triage says "hardware," spend in this order — each item multiplies
the value of everything after it, and the early items are embarrassingly
cheap compared to a new SDR:

| Priority | Upgrade | Why this order | Guide |
|---|---|---|---|
| 1 | Antenna (and its placement) | sets the signal everything else can only degrade | [scanner antennas]({{ '/best-scanner-antenna/' | relative_url }}), [SDR antennas]({{ '/best-sdr-antenna/' | relative_url }}), [mast & mounting]({{ '/antenna-mast-and-mounting-guide/' | relative_url }}) |
| 2 | Feedline + connectors | recovers dB you already paid for; pure noise figure | [cables & connectors]({{ '/sdr-cables-and-connectors/' | relative_url }}), Part 8 |
| 3 | Filter and/or mast LNA | buys NF or headroom — *after* the run is worth amplifying | [LNAs]({{ '/best-sdr-lna/' | relative_url }}), [filters]({{ '/sdr-filters/' | relative_url }}), Part 9 |
| 4 | Second device / diversity | coverage or fade insurance, at 2× complexity | [multi-dongle]({{ '/multi-dongle-sdr-setup/' | relative_url }}), Parts 11–12 |

A better SDR is deliberately absent from the top of that list: an upgraded
receiver behind a bad antenna and 8 dB of RG-58 receives the same garbage
with more dynamic range. The [what-do-I-need guide]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
walks complete builds in the same spirit.

## Filing an issue that gets fixed

If the triage lands on something GopherTrunk should handle better, file it —
but file it with the capture. The pattern that closes issues in this tracker
is always the same: symptom description, the sidecar-equipped IQ file (or an
`auto_record` grab), and the settings that produced it. From there the
maintainers' side of the contract kicks in: the capture becomes a
failing-first test, the fix makes it pass, and the file stays in `samples/`
so the bug can never quietly return — that's the
[#764/#771 discipline]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}),
and it's also why several decoders are *currently blocked* waiting on
nothing but a capture: the
[decoder capture needs]({{ '/decoder-capture-needs.html' | relative_url }})
page is effectively a wish list where one good recording from your antenna
closes a validation that no amount of code can. An issue without a capture
can still be triaged; an issue with one can be *finished*.

## The reader's system, fixed

So what was wrong with the system that started this series — hardware scanner
clean, GopherTrunk garbled, same antenna? Three things, none of them in
software, every one of them found by a measurement from this checklist. The
gain was parked 17 dB hot from an old attempt to satisfy a threshold; the
ladder (step 3) walked it back down the U-curve and the histogram stopped
brushing the rails. The dongle sat at the end of forty feet of RG-58 and an
adapter stack — ~8 dB of noise figure the scanner, on its shorter run, never
paid; LMR-400 and one pigtail bought six of them back (Part 8). And the last
stubborn garble tracked the sample rate: a capture at each rate, replayed
side by side, showed the deficit baked into the high-rate samples — the
Part 5 signature — and the config moved to the rate the front end runs clean
at. No single villain, which is the honest shape of most marginal systems:
three small analog taxes that only ever showed up in software, as a decoder
doing exactly what it was told with samples that were 10 dB worse than they
needed to be. The scanner doesn't win anymore.

That's the series. The chain from antenna to ADC is measurable end to end
with the instruments you already have — a dBFS tile, a histogram, a gain
ladder, a replay command, two log fields — and every hard problem along it
converts into a file that proves what it was. The decoder can only be as good
as the samples. Now you know how to make the samples good.

## FAQ

**What's the single fastest check when a system goes bad suddenly?**
dBFS, ten seconds. A level that dropped off a cliff is a broken feed
(connector, water, bias-tee supply); a level that's normal points onward.
Sudden-onset problems are overwhelmingly physical — check what the weather
did before you check what you upgraded.

**How do I A/B two setups fairly?**
Change one thing, and score with a scale-invariant number — decode error
rate, CRC-clean count, sync margin — never "the waterfall looks better."
Where possible, capture once and replay through both configurations; the
file removes time-of-day and traffic variance from the comparison.

**My symptom is intermittent and I can never catch it. Now what?**
Let the daemon catch it: `baseband.auto_record` triggers on the events that
matter (`on_cc_sync_loss`, `on_encrypted`, `on_concurrent_calls`) with a
cooldown, writing replay-ready files with sidecars. The TETRA sync-loss
investigation was built entirely on eleven such automatic captures.

**When is it actually a GopherTrunk bug?**
When the capture replays badly *and* a reference decoder (or a hardware
radio on the same samples) does better — that gap is real and reportable.
Plenty of tracker history is exactly this shape, and it's the productive
kind of report: the capture that demonstrates the gap is the same file the
fix gets tested against.

**Do I need all fourteen parts to use the checklist?**
No — the ladder above is self-contained, and each step links the part that
explains it when a step's result surprises you. If you read only one part in
full, make it [Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}):
capture discipline is the skill the other thirteen lean on.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Coherence, Not dBFS — Scale-Invariant Health]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
· This is the finale — back to the [series index]({{ '/blog/series/analog-edge/' | relative_url }}).

*Where to next? Your front end is clean and your samples are trustworthy — put them to work finding systems you didn't know were there in [**The Hunt**]({{ '/blog/series/the-hunt/' | relative_url }}).*
