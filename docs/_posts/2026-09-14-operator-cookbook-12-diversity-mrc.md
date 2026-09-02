---
title: "The Operator's Cookbook, Part 12: Two Antennas, One Signal — A Diversity Build"
description: A complete GopherTrunk diversity build — a two-channel SDR over SoapyRemote with diversity mrc, reading the MRC health line (coherence, branch_phase_deg), choosing mrc vs mrc-static by hardware class, and the pre-combine capture A/B that proves whether combining actually helps.
category: tutorials
keywords: sdr diversity receiver setup, maximal ratio combining sdr, usrp x310 twinrx diversity, dual antenna scanner rig, mrc vs mrc-static, branch coherence sdr, pre-combine iq capture, gophertrunk diversity config, gophertrunk cookbook
tags: [operator-cookbook, diversity, mrc, usrp, soapyremote, rf]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 12
---

*Part 12 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 11]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }})
put the rig in a closet and taught it to survive unattended. This part is the
advanced RF build: **two antennas feeding one decoder**, phase-coherently
combined so a fade on one branch is covered by the other. Diversity is the
one upgrade that can silently do nothing — or, before the aligner, actively
hurt — so the log lines that prove it's working matter more here than
anywhere else in the series.*

> **TL;DR:** Diversity needs **one radio with two coherent RX channels** — a
> USRP B210 (shared LO) or an X310 with TwinRX daughterboards (independent
> PLLs) — mounted over `sdr.soapy_remote[]` with `diversity: "mrc"` and
> `antenna: [RX1, RX2]`. Health is one log line every 30 s:
> `soapyremote: MRC diversity branches` with `coherence`, `branch_gain_db` and
> `branch_phase_deg`. **`branch_phase_deg` is the instrument**: constant ⇒
> shared-LO hardware, `mrc-static` is fine; walking ⇒ independent PLLs, you
> want tracking `mrc`. The honest ceiling: MRC is guaranteed **≥ the best
> branch alone**, but a real gain *over* it only shows on weak signals — prove
> yours with a `diversity_capture` pre-combine dump and
> `TestDiversityCombinerReplay`, scored by decode yield, never by
> [dBFS]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }}).

**Key takeaways**

- **Diversity is the last lever, not the first.** A better antenna, feedline
  and gain staging (Parts 7–9 of
  [The Analog Edge]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }}))
  buy more margin per dollar. Build this when one antenna genuinely fades.
- **Hardware class decides the mode.** A shared-LO front end has one fixed
  inter-branch phase; independent PLLs lock at a random phase that then
  walks. `branch_phase_deg` tells you which class you own, before any capture.
- **Coherence, not dBFS, is the verdict on the combine.** The
  [coherence]({{ '/reference/coherence/' | relative_url }}) figure is
  independent of RF gain — raising gain to make diversity "engage" is never
  the answer, and the config file says so in as many words.
- **MRC ≥ best branch is a floor, not a promise of gain.** On a strong signal
  every combined arm just matches the better antenna. The gain case lives on
  weak signals, and the pre-combine capture A/B is how you check yours.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Enable combining | open RX0+RX1, combine into one stream | `sdr.soapy_remote[].diversity: "mrc"` |
| Freeze the estimate | one-shot calibration, for A/B or shared-LO rigs | `diversity: "mrc-static"` |
| Antenna ports | per-channel port assignment, validated + read back | `antenna: [RX1, RX2]` |
| Hardware-class instrument | constant vs walking inter-branch phase | `branch_phase_deg` in the MRC health line |
| Pre-combine evidence | per-branch IQ dump only the driver can make | `diversity_capture`, `diversity_capture_seconds` |
| Offline A/B | replay both branches through every combiner arm | `TestDiversityCombinerReplay` (`GT_DIVERSITY_CAPTURE`) |
| The concepts | why MRC works and where it can't | [antenna diversity]({{ '/reference/antenna-diversity/' | relative_url }}), [MRC gotchas]({{ '/reference/mrc-diversity-gotchas/' | relative_url }}) |

## In this post

- **What you're building** — two antennas, one coherent front end, one combined stream.
- **The shopping list** — the two hardware classes, honestly priced.
- **The config** — `diversity: mrc` on a SoapyRemote device, every key verified.
- **First run — what healthy looks like** — the MRC health line, field by field.
- **The honest ceiling & the pre-combine A/B** — what MRC can buy, and how to prove it.
- **When it doesn't work** — symptom → cause → fix, then variations.

## What you're building

The finished rig is the [Part 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
remote-radio build with a second antenna: a two-channel SDR runs
`SoapySDRServer` near the antennas, GopherTrunk opens **both** RX channels,
and the driver phase-aligns and sums them into one maximised-SNR stream
before the decoder ever sees IQ. Everything downstream is unchanged;
diversity is invisible except in the health log and, when the channel fades,
in the calls that keep decoding.

Three stages inside the driver do the work, in order: a **pre-combine capture
tap** (raw, so evidence stays untouched), an **inter-branch aligner** that
removes the fractional-sample start skew between the streams (on one field
rig, 2.6 samples of skew made the combine decode **22% worse** than the best
branch alone — the aligner is why that can't happen anymore), and the **MRC
combiner**, which estimates one complex gain per calibration window and
either tracks it (`mrc`) or freezes it (`mrc-static`). How each stage earned
its place is
[Weak-Signal Engineering Parts 10–11]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }});
this recipe is the operator's side.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Signal chain of the diversity build: two antennas feed the two RX channels of one coherent SDR running SoapySDRServer; inside GopherTrunk's soapyremote driver the branches pass a pre-combine capture tap, then the inter-branch aligner, then the MRC combiner, which emits one combined wideband stream to the decoder.">
  <rect x="10" y="40" width="64" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="42" y="59" text-anchor="middle" fill="currentColor" font-size="10">ant A</text>
  <rect x="10" y="150" width="64" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="42" y="169" text-anchor="middle" fill="currentColor" font-size="10">ant B</text>
  <rect x="104" y="30" width="120" height="160" rx="6" fill="none" stroke="currentColor"/>
  <text x="164" y="50" text-anchor="middle" fill="currentColor" font-size="10">2-ch SDR</text>
  <text x="164" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="9">B210 / X310+TwinRX</text>
  <text x="164" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RX1 → ch0</text>
  <text x="164" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RX2 → ch1</text>
  <text x="164" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SoapySDRServer</text>
  <line x1="74" y1="55" x2="104" y2="55" stroke="currentColor"/>
  <line x1="74" y1="165" x2="104" y2="120" stroke="currentColor"/>
  <line x1="224" y1="110" x2="262" y2="110" stroke="currentColor"/>
  <polygon points="256,106 264,110 256,114" fill="currentColor"/>
  <rect x="264" y="24" width="288" height="196" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="408" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="10">GopherTrunk soapyremote driver</text>
  <rect x="280" y="56" width="256" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="408" y="75" text-anchor="middle" fill="currentColor" font-size="10">pre-combine capture tap (diversity_capture)</text>
  <rect x="280" y="98" width="256" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="408" y="117" text-anchor="middle" fill="currentColor" font-size="10">inter-branch aligner (delay_samples)</text>
  <rect x="280" y="140" width="256" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="408" y="159" text-anchor="middle" fill="var(--accent)" font-size="10">MRC combiner (mrc / mrc-static)</text>
  <text x="408" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="9">health: coherence · branch_gain_db · branch_phase_deg</text>
  <text x="408" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="9">constant phase ⇒ shared LO · walking ⇒ tracking mode</text>
  <line x1="552" y1="155" x2="592" y2="155" stroke="var(--accent)"/>
  <polygon points="586,151 594,155 586,159" fill="var(--accent)"/>
  <rect x="594" y="128" width="76" height="54" rx="4" fill="none" stroke="currentColor"/>
  <text x="632" y="150" text-anchor="middle" fill="currentColor" font-size="10">one wideband</text>
  <text x="632" y="164" text-anchor="middle" fill="currentColor" font-size="10">stream → decode</text>
</svg>
<figcaption>The whole diversity build: the decoder never knows there were two antennas — the capture tap, the aligner and the combiner live inside the driver, and one log line reports whether they're earning their keep.</figcaption>
</figure>

## The shopping list

This is the expensive recipe, and the honest first line is: **most rigs don't
need it.** You need one radio with two RX channels sharing a frequency
reference — two separate dongles will not do (independent clocks, no common
stream, nothing to phase-align).

| Item | Class | Notes |
|---|---|---|
| USRP B210 (or similar 2-ch shared-LO SDR) | shared LO | one chip, both channels — inter-branch phase is a hardware constant; `mrc-static` territory |
| USRP X310 + 2× TwinRX | independent PLLs | separate daughterboards (`rx_subdev_spec=B:0 A:0`): frequency-locked, but the relative phase is random per lock and walks after it — tracking `mrc` territory ([USRP notes]({{ '/reference/usrp-ettus/' | relative_url }})) |
| Two antennas, same band + polarisation | — | co-located — a wavelength or two apart, not metres — with decent [feedline]({{ '/blog/tutorials/analog-edge-08-feedline-connectors/' | relative_url }}) |
| A host for `SoapySDRServer` | — | the Part 8 remote pattern; wired network, budget for two channels of CS16 |

No external reference oscillator is required for the X310/TwinRX class: a
field capture proved the dual-daughterboard stream coherent as delivered
(wideband coherence 0.95+ on a healthy run) — handling the phase is the
calibrator's job.

## The config

Everything below is the [Part 8]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }})
SoapyRemote block plus three diversity keys, all verified against
`config.example.yaml`:

```yaml
sdr:
  sample_rate: 250_000        # soapy_remote isn't bound to RTL rate limits
  soapy_remote:
    - addr: "192.168.1.60:55132"
      driver: "uhd"
      args: "rx_subdev_spec=B:0 A:0"   # X310: one channel per TwinRX
      serial: "usrp-attic"
      role: control
      format: "CS16"
      gain: "auto"
      diversity: "mrc"                 # "" | "mrc" | "mrc-static"
      antenna: [RX1, RX2]              # RX1→channel 0, RX2→channel 1
      # diversity_capture: "../iq/mrc/precombine"
      # diversity_capture_seconds: 30

trunking:
  systems:
    - name: "Metro-TETRA"
      protocol: tetra
      control_channels:
        - 467_912_500
```

Three notes. **`antenna:`** is the only correct way to pick ports — names are
device-specific (a B210 has `TX/RX` and `RX2`, a TwinRX has `RX1` and `RX2`),
and GopherTrunk validates the list against the device and reads it back, so a
config moved between rigs fails loudly instead of silently keeping driver
defaults. **`diversity: "mrc"`** re-estimates the branch gain continuously;
**`"mrc-static"`** freezes it after one estimate — the first run tells you
which to pick. And the combine happens below the protocol layer, so a
[Part 4]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }})
TETRA rig or a Part 1 P25 rig rides it identically.

## First run — what healthy looks like

Start the daemon and let it lock as usual (`tetra cc locked freq=467912500
mcc=… mnc=…` on this recipe's system). The diversity-specific heartbeat is one
INFO line every 30 seconds:

```
INF soapyremote: MRC diversity branches addr=192.168.1.60:55132 branch_dbfs="ch0=-51.2 ch1=-50.6" reference_branch=1 calibrated=true coherence=0.95 branch_gain_db=-1.2 branch_phase_deg=145.3 mode=mrc updates=6682 holds=0
```

Read it field by field — this line is the whole cockpit:

- **`branch_dbfs`** — both branches alive and within a few dB of each other.
  A branch ~10 dB down is a weak antenna or feedline, and MRC on a
  floor-limited branch is roughly no-gain: fix *that branch's* RF first.
- **`coherence`** — the normalised cross-correlation between branches,
  **gain-independent by construction**. A healthy co-located pair reads high
  (0.9+) at moderate bandwidth; on wide captures where the wanted carrier is a
  small fraction of the band, lower numbers are normal and the lock gates
  scale with the window to allow for it — the
  [coherence-not-dBFS]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
  post is the full story.
- **`branch_phase_deg`** — the no-capture hardware-class instrument.
  **Constant** across lines means a shared-LO front end and `mrc-static`
  would serve; **walking** (a TwinRX rig measured about −0.2°/s — a frozen
  constant decays over minutes) means independent PLLs and tracking `mrc`.
- **`updates` / `holds`** — accepted vs rejected calibration windows.
  Healthy is `updates` climbing with `holds` near zero; `updates` frozen
  while `holds` climbs means coasting on a stale gain, and it WARNs after
  90 s of that.

One more line worth knowing on sight, once per stream or retune:

```
INF soapyremote: MRC inter-branch delay measured — delaying the early branch to align the combine delay_samples=0.41 peak_rho=0.94
```

That is the aligner latching the start skew between the two streams. The skew
is per-stream (a field pair measured 2.60 samples on one run and 0.41 on the
next, same rig), which is exactly why it's re-measured every time rather than
calibrated once.

## The honest ceiling — and the A/B that tests it

Now the part a recipe owes you before you spend this money. Post-aligner, MRC
is a **no-harm** combine: every capture A/B run to date scores the combined
arms within one decoded frame of the best branch alone. But those captures
decode at their ~100% yield ceiling — matching the best branch *is* the
ceiling there, and a real gain **over** the best branch is still
undemonstrated on air. Theory says it shows up on weak, fading signals; a
weak-signal pre-combine capture where the combined arm beats both branches is
genuinely wanted upstream.

Second limit: the combine applies **one complex gain to the whole wideband
stream**. Antennas metres apart give every carrier its own phase difference,
and a single scalar can only align one of them — the signature is coherence
stuck around 0.3–0.5 that no tracking improves. Co-locate the antennas;
per-channel combining after the DDC is known future work, not a config option
([MRC gotchas]({{ '/reference/mrc-diversity-gotchas/' | relative_url }})).

### The pre-combine A/B

Every other IQ tap —
[`baseband.auto_record`]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}),
the scope feeds — sits *downstream* of the combiner, so a capture from any of
them has one combiner baked in. `diversity_capture` is the one tap that
records the branches raw, straight after de-interleave:

```
INF soapyremote: diversity capture armed — dumping pre-combine per-branch IQ prefix=../iq/mrc/precombine seconds=30 branches=2
INF soapyremote: diversity capture complete — replay it with GT_DIVERSITY_CAPTURE to A/B combiners offline
```

You get `<prefix>.br0.cs16`, `<prefix>.br1.cs16` and a `.diversity.json`
sidecar; a datagram that didn't carry every branch is dropped from **both**
files, never written short — one short write silently desynchronises the
pair. Then grade every combiner against your own air:

```sh
GT_DIVERSITY_CAPTURE=../iq/mrc/precombine.diversity.json \
  go test ./cmd/gophertrunk -run TestDiversityCombinerReplay -v
```

The harness prints a windowed coherence/gain/**phase** trace with plain-text
verdicts ("branch phase is essentially CONSTANT" vs "branch phase WALKS"),
measures the inter-branch delay, and decodes the capture through each branch
alone plus static, tracking, aligned and narrowband combine arms — **scored
by CRC-clean decode yield**, the one metric that can't flatter a combiner. If
tracking beats static on *your* capture, the mode question is answered with
evidence instead of hardware folklore.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| WARN `MRC diversity got 1 of 2 channels in the stream` | the server never delivered the second RX channel | check `args` (`rx_subdev_spec`) and that the device really has two RX channels |
| WARN `MRC diversity branch is dead` | that antenna's connection, feedline or per-branch gain | reseat/re-test that branch's RF path; port assignment is `antenna:` |
| WARN `MRC diversity branches are not coherent` | different bands/polarisation, widely separated antennas, or no frequency lock | co-locate, match polarisation, check `clock_source`/`time_source`; a far-down `branch_dbfs` wants *its* gain staging fixed |
| Calibrated once, then WARN `has not accepted a calibration window` | the wideband-scalar limitation — one gain can't align every carrier from separated antennas | move the antennas together, or freeze deliberately with `mrc-static` |
| Combine decodes *worse* than one antenna alone | not skew (the aligner handles it) — a branch is dragging the estimate | run the pre-combine A/B; the arm table names the culprit |
| Tempted to raise gain until diversity "engages" | the old absolute-power trap | don't — `coherence` is gain-independent ([gain staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})) |

That last row has history: an earlier build gated calibration on a fixed
coherence constant, and its WARN told a bandwidth-diluted operator that
raising RF gain would *not* help — exactly wrong in their regime (their weak
branch needed +5 dB of per-branch gain). The gates now bound the estimate's
phase error instead, and the WARN points at per-branch gain staging. The
series-wide lesson: **never trust an absolute-dBFS rule; trust coherence and
decode yield.**

### Variations

- **`diversity: "mrc-static"`** — one-shot calibration, frozen. Correct for
  shared-LO hardware (constant `branch_phase_deg`), and the standard A/B
  reference arm on any rig.
- **Longer evidence** — `diversity_capture_seconds` accepts up to 120 s (a
  1 GiB/branch cap always applies). At 200–250 kS/s two CS16 branches are only
  ~1.6 MB/s total, so take the long capture.
- **Single-channel, port-pinned** — `diversity: ""` with `antenna: [RX2]`:
  the same radio as a plain remote SDR with the port explicit and validated —
  the "each branch alone" baseline before you commit.
- **Voice under diversity** — the combined stream is an ordinary tuner to the
  rest of the daemon; `role: wideband` with `voice_taps` on top works exactly
  as in Part 1.

## Where this goes next

The rig now has every piece of hardware this series will ask you to buy. What
it doesn't have yet is *names* — every talkgroup, radio and site is still a
number. [Part 13]({{ '/blog/tutorials/operator-cookbook-13-naming-everything/' | relative_url }})
is the zero-hardware recipe: alias CSVs with their real column formats, naming
things live from the browser, and exporting the result back to files you own.

## FAQ

**Do I need a diversity setup for a scanner rig?**
Almost certainly not first. Diversity is the only lever that costs serious
hardware, and it addresses one regime: a signal that *fades* at your location.
A better antenna, low-loss feedline and correct gain staging buy more decode
margin per dollar; build this when those are done and a marginal system still
swings.

**Should I use mrc or mrc-static?**
Let `branch_phase_deg` answer: run `mrc` and watch the health line for a few
minutes. Constant phase means shared-LO hardware — `mrc-static` is equivalent
and simpler. Walking phase — TwinRX-style independent PLLs — means stay on
`mrc`, because a frozen constant decays over minutes.

**Can I do diversity with two RTL-SDR dongles?**
No. The combine needs two RX channels behind one clock in one sample stream —
`diversity: mrc` opens RX0+RX1 on a single SoapyRemote device. Separate
dongles have independent oscillators and USB timelines; there is no constant
phase or delay for the calibrator to find.

**Why is my coherence only 0.3 when everything decodes fine?**
Wideband coherence is diluted by every hertz of noise-only bandwidth around
your carrier — a clean channel occupying a small fraction of a wide capture
reads low even when the branches agree perfectly in-channel, which is why the
lock gates scale with the estimation window. Judge the combine by decode
yield and the WARN lines, not by wishing the number toward 1.0.

**Does MRC help against a strong interferer?**
Not blind, no — that's interference rejection combining (IRC). Measured on a
synthetic co-channel scene, blind IRC buys 0.0 dB (the channel estimate is
contaminated by the interferer itself); the same code given a training
sequence buys over 20 dB. That's why IRC exists only as an offline harness
arm, not a `diversity:` mode.

## Series navigation

**Part 12 of 14** · ←
[Part 11: The Closet Appliance — Pi, systemd & Docker]({{ '/blog/tutorials/operator-cookbook-11-closet-appliance/' | relative_url }})
· Next →
[Part 13: Naming Everything — Aliases, Labels & Exports]({{ '/blog/tutorials/operator-cookbook-13-naming-everything/' | relative_url }})
