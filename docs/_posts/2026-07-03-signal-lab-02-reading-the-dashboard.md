---
title: "Signal Lab, Part 2: Reading the Dashboard — Symbols, Baud, Lock & the 2% Rule"
description: Signal Lab's terminal dashboard, field by field — protocol, symbols, effective baud versus expected, lock status and latency, symbol histogram, IQ imbalance, decode-error rate, EVM, and SNR — plus the ~2% baud-deviation rule that catches a wrong sample rate.
category: tutorials
keywords: siglab dashboard, effective baud, symbol rate deviation, iq imbalance, evm, snr estimate, decode error rate, symbol histogram, sample rate mismatch, p25 lock, auto-tune
tags: [siglab, dsp, iq, getting-started, p25, demodulation]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 2
charts: true
---

*Part 2 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. Last time we ran a capture; now we learn to read what
the run tells us.*

> **TL;DR:** The Signal Lab dashboard is a one-screen verdict on a capture:
> protocol, sample and symbol counts, **effective baud versus expected** (with a
> signed % deviation), lock status/frequency/latency, grants, a symbol
> histogram, IQ imbalance, a decode-error rate per 1000 symbols, and a demod
> block with EVM% and an SNR estimate. The single most useful number is the baud
> deviation: if it drifts more than about **2%**, your `-sample-rate` is wrong,
> not your signal.

**Key takeaways**

- **Effective baud vs expected is your rate check.** A deviation past ~2% almost
  always means the capture's true rate ≠ `-sample-rate`.
- **Lock is binary, but latency is diagnostic.** *When* it locked tells you as
  much as *whether* it locked.
- **The symbol histogram shows modulation health** before any fancy plot — four
  clean bins for a 4-level C4FM system, smeared bins for a sick one.
- **EVM, SNR, IQ imbalance, and decode-error rate** are the quality quartet; read
  them together, never alone.
- **Press `e` to export** the whole result as `<capture>.siglab.json`.

## Cheat sheet

| Command / flag / key | What it does |
|---|---|
| `gophertrunk siglab -in cap.cfile -sample-rate 2400000` | Run and open the dashboard |
| `-protocol p25p1` | Skip the picker |
| `-format u8 \| f32` | Match the capture's sample format |
| `-freq <Hz>` | Informational nominal center frequency |
| `-auto-tune` | Tune out carrier offset before demod |
| `-verbose-errors` | Full error chain + stack on failure |
| `↑`/`↓` or `k`/`j`, `enter` | Move the picker, run the decode |
| `e` | Export `<capture>.siglab.json` |
| `q` | Quit |

## In this post

- **The dashboard, field by field** — what each line means.
- **The 2% rule** — turning baud deviation into a sample-rate diagnosis.
- **Reading the symbol histogram** — modulation health at a glance.
- **The quality quartet** — EVM, SNR, IQ imbalance, decode-error rate.
- **Keys and exports** — driving the TUI and getting structured output.

## The dashboard, field by field

When a run finishes, the terminal app lands on the results screen. Here's a
representative dashboard for a healthy P25 Phase 1 control channel:

```text
Signal Lab — Results

Protocol:   P25 Phase 1
Samples:    12,000,000 (5.00s @ 2400000 Hz)
Symbols:    23,988   effective baud 4797.6 (expected 4800, -0.1%)
Lock:       LOCKED @ -12500 Hz (0.42s)
Grants:     3

Symbol histogram
  0  ████████·················· 18.9%
  1  ███████··················· 30.6%
  2  ███████··················· 31.1%
  3  ████······················ 19.4%

IQ imbalance: gain +0.12 dB  phase +0.4°  image rej 43.8 dB
Decode-error rate: 1.20 / 1000 symbols
Demod (C4FM): EVM 7.4%  SNR≈19.7 dB
```

Read it top to bottom:

- **Protocol** — what the engine decoded as (the one you picked or passed).
- **Samples** — total IQ samples, the capture duration in seconds, and the
  sample rate it was decoded at. If the rate here isn't the file's true rate,
  everything below is suspect.
- **Symbols** — recovered symbol count, then the headline pair: **effective baud**
  (the symbol rate the demod actually measured) against the protocol's
  **expected** baud, plus a signed **% deviation**.
- **Lock** — `LOCKED @ <freq> (<latency>)` or `NOT LOCKED`. The frequency is the
  residual carrier offset the receiver settled on; the latency is how long into
  the capture it took to lock.
- **Grants** — trunking channel grants seen (control-channel activity).
- **Symbol histogram** — the distribution of recovered symbols across the
  modulation's levels.
- **IQ imbalance** — gain (dB), phase (degrees), and image-rejection (dB) of the
  front-end's I/Q balance, shown only when the lab observed enough signal.
- **Decode-error rate** — errored symbols per 1000, a direct FEC-adjacent health
  number.
- **Demod** — the modulation the demod ran, its **EVM%**, and an **SNR estimate**.

## The 2% rule: baud deviation as a rate check

The most valuable habit you can build is reading the baud deviation *first*.
Both of GopherTrunk's down-converters normalize a capture to the per-protocol
channel rate before demod, which makes the decode path essentially
**rate-invariant** to the *capture* rate — a clean signal decodes the same
whether it was recorded at 2.4 MS/s or 10 MS/s. That invariance is what makes the
deviation such a sharp tool: if the true symbol rate is fixed by the protocol and
the demod's measured baud drifts, the drift is telling you the *ratio* between
the rate you claimed and the rate the file was actually recorded at.

The rule of thumb, straight from the docs: **if the effective baud drifts more
than about 2% from the expected symbol rate, the capture's true sample rate
probably doesn't match `-sample-rate`.** Fix the rate and re-run before trusting
any other metric.

A worked example. Ada's capture reads:

```text
Symbols:    24,010   effective baud 4996.9 (expected 4800, +4.1%)
Lock:       NOT LOCKED
```

That `+4.1%` is the whole story. 4800 × (2500000 / 2400000) ≈ 5000 — the file was
recorded at 2.5 MS/s, but she left the default `-sample-rate 2400000`. Re-running
with `-sample-rate 2500000` snaps the deviation back under a percent and the
channel locks. No amount of squinting at the constellation would have helped;
the number named the fault directly.

The corollary, which Reese never tires of repeating: once the rate is right and
the deviation is small, a capture that *still* won't lock or reads a low SNR is
telling you about the **captured samples**, not the DSP — front-end overload,
intermod, or gain staging at record time. The lab has done its job by pointing
you at the file rather than the code.

## Reading the symbol histogram

Below the top block, the histogram shows how recovered symbols distribute across
the modulation's levels. For a 4-level system like C4FM P25 you want four
distinct, well-populated bins; a healthy control channel is roughly balanced
across them. When bins smear together or one collapses, the demod is struggling
to separate symbol levels — the eye is closing (Part 4), the SNR is low, or the
rate is still off.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="560" height="300" role="img"
        aria-label="Symbol histogram: four populated levels for a healthy C4FM channel"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["level 0","level 1","level 2","level 3"],
"values":[18.9,30.6,31.1,19.4],
"pass":[true,true,true,true],
"ylabel":"% of symbols" }
</script>
<figcaption>Illustrative C4FM symbol histogram — four clearly separated levels is the healthy signature. Smeared or collapsed bins mean the demod can't tell symbols apart.</figcaption>
</figure>

The histogram is a fast, no-plot sanity check: you can read modulation health
from four bars before you ever open a constellation.

## The quality quartet

Four fields together describe *how well* the capture demodulated. Read them as a
set, because any one alone can mislead.

| Field | What it tells you | Watch for |
|---|---|---|
| **EVM %** | RMS error-vector magnitude — distance from ideal symbol points | Rises sharply as SNR drops; the demod-quality headline |
| **SNR estimate (dB)** | In-channel signal-to-noise the receiver saw | Below ~10 dB on C4FM, lock gets fragile |
| **IQ imbalance** | Front-end gain/phase balance + image rejection | Large gain/phase or low image-rejection dB points at the SDR, not the signal |
| **Decode-error rate** | Errored symbols per 1000 | Climbs before lock is lost — an early-warning gauge |

Reese's mental model: **EVM and SNR move together and describe the channel; IQ
imbalance describes the front-end; decode-error rate describes the consequences.**
A capture with great SNR but ugly IQ imbalance was recorded on a radio with a
gain/phase problem. A capture with fine IQ balance but low SNR and high EVM is
just weak. The four fields let you tell those apart without guessing.

## Keys, auto-tune, and exports

The TUI is deliberately small. On the protocol picker, `↑`/`↓` (or `k`/`j`) move
the cursor and `enter` runs the selected protocol. On the dashboard, `e` exports
and `q` (or `Ctrl-C`) quits. A few flags shape the run:

- **`-auto-tune`** estimates the carrier offset and tunes the capture to 0 Hz
  before demod — reach for it when a capture was recorded off-center and the
  residual-offset field in the lock line is large.
- **`-protocol`** and **`-format`** skip the picker and set the sample encoding.
- **`-freq`** is purely informational — a nominal center frequency for the report;
  it doesn't change decoding.
- **`-verbose-errors`** prints the full error chain and a stack on failure, which
  is what you want when a run dies instead of just decoding badly.

Pressing `e` writes the complete structured result next to your capture as
`<capture>.siglab.json`:

```go
// cmd/gophertrunk/siglab_tui.go
path := strings.TrimSuffix(m.capture, ext(m.capture)) + ".siglab.json"
// ...
siglab.WriteResult(f, m.result, siglab.FormatJSON)
```

That JSON carries every field on the dashboard and more, which is how you turn a
one-off replay into a fixture or a diff. The browser console (Part 3) exposes the
same result through JSON/JSONL/YAML/CSV serializers.

## Where this goes next

You can now read a capture's health from one screen and tell a rate problem from
a signal problem. [Part 3]({{ '/blog/tutorials/signal-lab-03-browser-console-live-decode/' | relative_url }})
moves to the browser console — `gophertrunk siglab serve` — where the same
metrics arrive over a live event stream alongside constellations, spectrograms,
and side-by-side comparison. For the canonical field list, the
[SigLab docs]({{ '/siglab.html' | relative_url }}) page stays the source of truth.

## FAQ

**My baud deviation is +4%. What do I change?**
Your `-sample-rate`. A deviation past ~2% means the value you passed doesn't
match the file's true recording rate. Multiply the expected baud by
(true rate / claimed rate) to sanity-check, set the right rate, and re-run.

**It locked but SNR is only 9 dB — is that the lab's fault?**
No. Because the decode path is rate-invariant, a low in-channel SNR on a
correctly-rated capture is a property of the recorded samples (weak signal,
front-end overload, gain staging), not the DSP. Get a cleaner capture.

**What does image rejection tell me?**
It's the front-end's I/Q balance quality in dB — higher is better. A low
image-rejection number with visible gain/phase imbalance points at the SDR or
recording chain, not the signal.

**Where does the exported JSON go?**
Next to the capture, as `<capture>.siglab.json`, written by the engine's own
serializer when you press `e`.

## Series navigation

**Part 2 of 10** · ←[Part 1]({{ '/blog/tutorials/signal-lab-01-first-capture-no-radio/' | relative_url }}) · Next →
[Part 3: The Browser Console]({{ '/blog/tutorials/signal-lab-03-browser-console-live-decode/' | relative_url }})
