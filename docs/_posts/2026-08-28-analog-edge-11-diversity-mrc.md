---
title: "The Analog Edge, Part 11: Two Antennas — Diversity & MRC From the Operator's Seat"
description: What a second antenna and maximal-ratio combining actually buy on a fading trunking band, how to configure GopherTrunk's diversity mode and read its health log line, and the coherence signature that tells you when two antennas cannot be combined with one wideband gain.
category: tutorials
keywords: antenna diversity sdr, maximal ratio combining, mrc sdr, two antenna receiver, diversity combining trunking, soapy remote diversity, coherence log line, multipath fading scanner, gophertrunk analog edge
tags: [analog-edge, diversity, mrc, antennas, sdr, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 11
---

*Part 11 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system. [Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
gave us capture discipline — including the one pre-combine tap that makes this
part's claims checkable. Our marginal reader's system now decodes most of the
day, but fades still eat calls when trucks park in the Fresnel zone. The next
purchase isn't a better antenna; it's a **second** one. This part is the
operator's view of diversity and maximal-ratio combining: what to configure,
what the log line means, and the one number that says whether the pair is
working. The algorithm internals live in Weak-Signal Engineering — here we
stay in the operator's seat.*

> **TL;DR:** Two antennas fade differently, and a coherent combiner
> (**MRC** — maximal-ratio combining) can add their SNRs instead of picking
> one. In GopherTrunk this is `diversity: "mrc"` on an `sdr.soapy_remote`
> device with two RX channels (ports selected via `antennas: [RX1, RX2]`).
> The combiner aligns the branches with **one complex gain across the whole
> wideband stream** — exact only when the branches differ by a frequency-flat
> constant, which co-located antennas satisfy and metres-apart antennas on a
> busy band do not. The health line's `coherence` figure is the verdict:
> ~0.7+ combining well; **stuck at 0.3–0.5 is the wideband-scalar signature**,
> not a gain problem. `mrc` tracks a drifting front end; `mrc-static` freezes
> after one estimate. A/B them offline against your own `diversity_capture`
> with `TestDiversityCombinerReplay` — decode yield is the verdict, never EVM.

**Key takeaways**

- **Diversity buys fade insurance, not raw gain.** Two antennas rarely fade
  together; MRC's output SNR approaches the *sum* of the branch SNRs when the
  branches cohere — the win shows up on the marginal calls, not the strong ones.
- **One complex scalar combines the whole band.** That's cheap and right for a
  shared-mast pair; it is structurally unable to align every carrier when the
  antennas are far apart, because each carrier then has its own phase difference.
- **`coherence` in the log line is the health number.** It's scale-invariant —
  raising RF gain never moves it — and the WARN messages name the actual fix
  (antenna, band, polarization, clocking), not a gain knob.
- **Captures decide the mode.** The pre-combine `diversity_capture` from
  Part 10 replayed through the four-arm harness (each branch alone, static,
  tracking) is the only honest A/B; tracking-as-default is gated on exactly that.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Enable diversity | open RX0+RX1, combine into one stream | `sdr.soapy_remote[].diversity: "mrc"` (`config.example.yaml`) |
| Antenna ports | select each channel's RX port | `sdr.soapy_remote[].antennas: [RX1, RX2]` |
| The combiner | coherent MRC with tracked branch gain | `internal/sdr/soapyremote/mrc.go` (`mrcCombiner`), `internal/dsp/diversity` |
| Health report | per-branch dBFS, coherence, gain, phase every 30 s | `diversityReporter` (`mrc.go`, issue [#1062](https://github.com/MattCheramie/GopherTrunk/issues/1062)) |
| Pre-combine capture | per-branch IQ for offline A/B | `diversity_capture` / `diversity_capture_seconds` |
| Offline A/B harness | four decode arms scored by CRC-clean BSCH | `TestDiversityCombinerReplay` (`cmd/gophertrunk/diversity_replay_test.go`) |
| Hardware for two channels | dual-RX SDRs and multi-dongle alternatives | [multi-dongle setup guide]({{ '/multi-dongle-sdr-setup/' | relative_url }}) |

## In this post

- **What a second antenna buys** — fading, multipath, and the MRC idea.
- **Configuring it** — the YAML, the ports, and what to expect at startup.
- **Reading the health line** — dBFS per branch, coherence, and the WARNs.
- **The wideband-scalar limit** — why antenna placement is part of the DSP.
- **`mrc` vs `mrc-static`** — and the capture-driven way to choose.

## What a second antenna buys

A trunking band fade is local: multipath from buildings and traffic carves
nulls that move with the scatterers, and a null at one antenna is usually not
a null at an antenna half a wavelength away. [Selection
diversity]({{ '/reference/antenna-diversity/' | relative_url }}) — switch to
the better branch — already removes most deep fades. MRC goes further: weight
each branch by its channel gain and sum *coherently*, so both antennas
contribute even when neither is best; in white noise with aligned branches the
output SNR is the sum of the branch SNRs (up to 3 dB for an equal pair, more
than that in effect during fades, which is when it matters). The catch is the
word *coherently*: the branches must be phase-aligned before they can add, and
estimating that alignment from the stream itself — continuously, without
disturbing the decoders downstream — is the whole trick. How GopherTrunk's
calibrator does it (coherence-gated least squares, then a slow tracking loop
that is provably safe ahead of a differential demodulator) is
[Weak-Signal Engineering Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
and [Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }});
from the operator's seat you only need what the knobs do and what the log says.

## Configuring it

Diversity rides the SoapyRemote driver, on hardware with two RX channels fed
from the same clock — a USRP B210, an X310 with TwinRX cards, and friends (see
the [multi-dongle guide]({{ '/multi-dongle-sdr-setup/' | relative_url }}) for
what qualifies; two independent RTL dongles do not — they don't share a clock):

```yaml
# config.yaml — sdr section (shape; see config.example.yaml)
soapy_remote:
  - addr: "192.168.1.60:55132"
    driver: "uhd"
    role: control
    format: "CS16"
    diversity: "mrc"          # "" | "mrc" | "mrc-static"
    antennas: [RX1, RX2]      # RX port per channel, checked against the device
    diversity_capture: ""     # set a path prefix for the Part 10 pre-combine dump
    diversity_capture_seconds: 0   # 1..60 (0 = 5 s)
```

Three notes from the field. `antennas` exists because a comma-separated
antenna can't live in the flat `args` string — and port names are
device-specific (a B210 has `TX/RX` and `RX2`; a TwinRX has `RX1` and `RX2`),
so a config moved between rigs fails loudly rather than silently keeping a
driver default (that silent-default failure was a real bug, and both branches
must be *gained* — an ungained second channel is a dead branch). At startup
the very first datagram logs a health line, so an operator enabling `mrc`
sees both branches' levels immediately rather than 30 seconds later. And
until the calibrator's first estimate is accepted, the combiner passes the
reference branch through verbatim — enabling diversity never makes you
*worse* than your primary antenna while it's warming up.

## Reading the health line

Every 30 seconds (`mrcHealthInterval`), the `diversityReporter` prints one
line — INFO when healthy, WARN with a named fix when not:

```
INFO soapyremote: MRC diversity branches addr=… branch_dbfs="ch0=-31.2 ch1=-33.8"
     reference_branch=0 calibrated=true coherence=0.78 branch_gain_db=-2.1
     branch_phase_deg=41.3 mode=mrc updates=412 holds=9
```

| Field | What to read from it |
|---|---|
| `branch_dbfs` | both branches alive and within ~20 dB of each other; a branch >20 dB below the reference is declared **dead** with a WARN naming the antenna/gain fix |
| `coherence` | the health number — the normalised cross-correlation of the last calibration window; see below |
| `branch_gain_db` / `branch_phase_deg` | the measured branch imbalance; the *phase* field is Part 12's hardware-class instrument |
| `updates` / `holds` | accepted vs gate-rejected calibration windows; climbing holds against flat updates = the branches aren't seeing the same signal |
| `calibrated` | false forever + windows completing = the not-coherent WARN below |

Two WARNs matter. **Dead branch** (`ch1` missing or ≥20 dB down): a
disconnected antenna, an ungained channel, or a server that honored only one
channel of the two-channel request — before [#1062](https://github.com/MattCheramie/GopherTrunk/issues/1062)
this was invisible and just looked like a weak single receiver. **Not
coherent** (both branches alive, `calibrated=false` after windows have
completed): the two receivers are not seeing the same signal through a
constant complex gain, and the WARN says explicitly that raising RF gain will
NOT help — the gate is scale-invariant
([Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
is entirely about why). Check band, polarization, co-location, and the front
end's `clock_source` instead.

## The wideband-scalar limit

Here is the design fact that makes antenna *placement* part of the DSP. The
combiner aligns branches with **one complex gain applied to the whole
wideband stream**. That is exact only if the branches differ by a
frequency-flat constant — true when both antennas are effectively at the same
point (same mast, stacked). Put the antennas metres apart and every carrier in
the band acquires its *own* phase difference, set by geometry and direction of
arrival: the scalar aligns whichever carrier dominates the calibration window
and partially cancels others.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Two antennas feeding two receiver branches into a combiner that applies a single complex gain to align branch one with branch zero before summing. Below, an inset shows the failure case: with widely separated antennas, three carriers across the band each arrive with a different phase difference, so no single complex gain can align all of them and coherence sticks around 0.3 to 0.5.">
  <rect x="10" y="30" width="86" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="53" y="51" text-anchor="middle" fill="currentColor" font-size="10">antenna A</text>
  <rect x="10" y="86" width="86" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="53" y="107" text-anchor="middle" fill="currentColor" font-size="10">antenna B</text>
  <line x1="96" y1="47" x2="150" y2="47" stroke="currentColor"/><polygon points="150,43 160,47 150,51" fill="currentColor"/>
  <line x1="96" y1="103" x2="150" y2="103" stroke="currentColor"/><polygon points="150,99 160,103 150,107" fill="currentColor"/>
  <rect x="160" y="30" width="96" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="208" y="51" text-anchor="middle" fill="currentColor" font-size="10">branch 0 (ref)</text>
  <rect x="160" y="86" width="96" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="208" y="101" text-anchor="middle" fill="var(--accent)" font-size="10">branch 1</text>
  <text x="208" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">× one complex gain h</text>
  <line x1="256" y1="47" x2="316" y2="70" stroke="currentColor"/>
  <line x1="256" y1="103" x2="316" y2="80" stroke="currentColor"/>
  <circle cx="326" cy="75" r="12" fill="none" stroke="currentColor"/>
  <text x="326" y="79" text-anchor="middle" fill="currentColor" font-size="12">+</text>
  <line x1="338" y1="75" x2="392" y2="75" stroke="currentColor"/><polygon points="392,71 402,75 392,79" fill="currentColor"/>
  <rect x="402" y="58" width="120" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="462" y="79" text-anchor="middle" fill="var(--accent)" font-size="10">combined stream</text>
  <text x="546" y="70" fill="var(--fg-muted)" font-size="9">co-located pair:</text>
  <text x="546" y="82" fill="var(--fg-muted)" font-size="9">coherence ~0.7+</text>
  <line x1="20" y1="140" x2="660" y2="140" stroke="var(--fg-muted)" stroke-dasharray="3,4"/>
  <text x="24" y="158" fill="var(--fg-muted)" font-size="9">metres-apart antennas, busy band:</text>
  <text x="290" y="163" fill="currentColor" font-size="10">carrier 1: Δφ = 15°</text>
  <text x="290" y="178" fill="currentColor" font-size="10">carrier 2: Δφ = 130°</text>
  <text x="290" y="193" fill="currentColor" font-size="10">carrier 3: Δφ = −80°</text>
  <text x="470" y="170" fill="var(--accent)" font-size="10">one h cannot align all three</text>
  <text x="470" y="185" fill="var(--accent)" font-size="10">→ coherence stuck ~0.3–0.5</text>
</svg>
<figcaption>MRC aligns the branches with one wideband complex gain — exact for a co-located pair, structurally impossible when each carrier has its own phase difference.</figcaption>
</figure>

The signature is a wideband `coherence` stuck around **0.3–0.5 that no amount
of tracking improves**. That is not a fault and not a gain problem; it's the
architecture's honest report that your antennas are too far apart for a
wideband combine. The correct fix at that point — combining *after* the
per-channel DDC, one gain per narrowband channel — is a much larger change
and is **not built**; the coherence figure existing in the health line is
what makes the limitation visible instead of silent. Operator guidance:
mount the diversity pair on the same mast, same band, same polarization, and
save the widely-spaced-antennas idea for a second independent SDR instead.

## `mrc` vs `mrc-static`

Both modes calibrate the same way; they differ in what happens after the
first accepted estimate. `mrc` keeps re-estimating (a slow ~200 ms tracking
loop) — right when the two RX channels sit on hardware whose relative phase
can drift. `mrc-static` freezes after one estimate — the classic one-shot
calibration, right when the front end shares one LO and the branch phase
genuinely is a constant. Which hardware you have is
[Part 12]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})'s
whole subject (and the log's `branch_phase_deg` answers it without buying
anything). When in doubt: record a `diversity_capture`
([Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}))
and run the offline A/B —

```
GT_DIVERSITY_CAPTURE=<prefix>.diversity.json go test ./cmd/gophertrunk \
    -run TestDiversityCombinerReplay -v
```

— which prints a windowed coherence/gain/**phase** trace and decodes four
arms through identical downstream wiring: branch 0 alone, branch 1 alone,
static combine, tracking combine, scored by CRC-clean decode count. Yield is
the verdict, never EVM — the repo has measured a combiner "improving" a
constellation while decoding nothing. If neither combined arm beats your
better branch alone, the second antenna isn't paying for its coax on that
signal, and that's worth knowing before winter.

## Where this goes next

The `mrc` vs `mrc-static` choice is really a question about what's inside
your radio: one synthesizer or two.
[Part 12]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})
sorts front ends into those two classes — shared LO versus independent PLLs —
and shows how `branch_phase_deg` identifies yours from the log line alone,
no capture required.

## FAQ

**Will diversity help my weak-signal problem?**
It helps *fading* — the calls that come and go. It does not fix a system
that's uniformly 10 dB short; that's Parts 7–9 territory (antenna, feedline,
LNA). If your marginal channel fails steadily rather than intermittently, fix
the single-antenna chain first — it's cheaper and helps every hour of the day.

**Can I do diversity with two RTL-SDR dongles?**
Not coherent MRC — the branches must share a sampling clock and be delivered
sample-aligned, which the SoapyRemote dual-channel path provides and two free-
running dongles do not. Two dongles are still great for coverage (separate
control + voice, or two bands): see the
[multi-dongle guide]({{ '/multi-dongle-sdr-setup/' | relative_url }}).

**What coherence number should I expect?**
Healthy co-located pair on a live band: roughly 0.6–0.9 depending on how much
correlated signal the window holds (the mapping to per-branch SNR is
[Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})).
Stuck at 0.3–0.5: wideband-scalar limit — antennas too separated. Near zero
with both branches alive: different bands/polarizations or a clocking problem.

**Does enabling `mrc` risk making things worse?**
Before calibration the combiner passes the reference branch through, and a
window that fails its quality gates *holds* the previous weights rather than
lurching — so the designed floor is "your primary antenna, as-is." The honest
answer beyond design intent is your own four-arm A/B, which is exactly what
the harness exists for.

**Which branch is the reference?**
Branch 0 by default; the driver re-selects only if the reference goes
genuinely dead (persistently ~20 dB below a challenger) — deliberately sticky,
because swapping the phase anchor mid-stream is itself a glitch. An early
version re-picked the loudest branch every datagram while uncalibrated, and an
ordinary ~1 dB crossover between two healthy receivers kept swapping the
anchor — since fixed.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Capture Discipline — cfile, cs16, SigMF & Metadata]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
· Next →
[Part 12: Front-End Classes — Shared LO vs Independent PLLs]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})
