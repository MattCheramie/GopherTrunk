---
title: "Signal Lab, Part 10: The Demod Bench — siglab sweep, Regression & Export"
description: Signal Lab's demod benchmark, gophertrunk siglab sweep, synthesizes P25 Phase 1 across an SNR ladder on both the c4fm and cqpsk/lsm demod paths, decodes through the production pipeline, and prints measured demod quality against a theoretical symbol-error-rate reference with CSV export.
category: tutorials
keywords: siglab sweep, demod benchmark, snr ladder, symbol error rate, es/n0, evm vs snr, demod regression, c4fm cqpsk lsm, csv export, p25 phase 1, demod calibration
tags: [siglab, demodulation, benchmark, regression, p25, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 10
charts: true
---

*Part 10 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. We close where the workbench earns its keep — a
repeatable demod benchmark that answers "did my change help?" with a curve.*

> **TL;DR:** `gophertrunk siglab sweep` synthesizes P25 Phase 1 across an SNR
> ladder on **both** demod paths (`c4fm` and `cqpsk`/`lsm`), decodes each rung
> through the production pipeline, and prints measured demod quality — lock, EVM,
> estimated SNR — against a **theoretical symbol-error-rate reference**. Flags
> set the ladder (`-snr-min`, `-snr-max`, `-snr-step`), the AWGN `-seed`, and an
> optional `-csv` export. It's the regression instrument that turns a demod tweak
> from a hunch into evidence.

**Key takeaways**

- **The sweep is a curve, not a single test.** It measures across an SNR ladder,
  so you see *where* a demod holds and where it falls apart.
- **Both paths, every run.** `c4fm` and `cqpsk`/`lsm` are swept together against
  their own theoretical references.
- **Measured vs theory is the read.** Tracking theory down to low SNR is good;
  an early divergence is a demod deficiency.
- **Reproducible by seed.** A fixed `-seed` makes the ladder a valid regression
  fixture.
- **`-csv` exports** the whole sweep for diffing across code changes.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `gophertrunk siglab sweep` | Sweep both paths, 2–30 dB |
| `-snr-min <dB>` | Minimum injected SNR (default `2`) |
| `-snr-max <dB>` | Maximum injected SNR (default `30`) |
| `-snr-step <dB>` | SNR step (default `2`) |
| `-seed <n>` | AWGN seed (default `0x5175`) |
| `-csv <path>` | Also write the sweep as CSV |

## In this post

- **What the sweep measures** — quality across an SNR ladder.
- **Reading the curve** — measured against theory.
- **Both demod paths** — c4fm and cqpsk/lsm side by side.
- **The CSV** — columns and regression workflow.
- **Series wrap** — where the Lab Bench trilogy goes from here.

## What the sweep measures

Every earlier part measured *one* capture. The **demod benchmark** measures the
*demodulator itself* — how well it performs as conditions get worse. It
synthesizes P25 Phase 1 at a ladder of injected SNRs, decodes each rung through
the exact production pipeline, and records the measured quality at each level.
The result is a curve, and a curve says something a single pass can't: not just
"does it lock?" but "*down to what SNR* does it lock, and how gracefully does it
degrade on the way down?"

```bash
gophertrunk siglab sweep                                            # both paths, 2–30 dB
gophertrunk siglab sweep -snr-min 6 -snr-max 20 -snr-step 1 -csv sweep.csv
```

The default ladder runs 2 dB to 30 dB in 2 dB steps. Because it's built on the
synthesis engine from Part 6, the whole thing is deterministic for a given
`-seed` — the same noise realization every run, which is precisely what a
regression fixture requires. The sweep is the human-readable, on-demand companion
to a hard-gated regression test that lives in CI: same instrument, run whenever
you want an answer.

## Reading the curve: measured vs theory

The sweep's output is a table per path, and the important column is the
comparison to a **theoretical symbol-error-rate reference**. Theory says, for a
given Es/N0, what the best-achievable symbol-error behavior is; the sweep plots
your *actual* demod against that ideal. Here's a representative run:

```text
c4fm (vs theoretical reference):
  SNR dB   Es/N0    lock    EVM %    SNR~ dB
  2.0      -1.8     no      34.0     3.1
  6.0       2.2     yes     22.0     6.8
  10.0      6.2     yes     12.0     10.4
  14.0     10.2     yes      6.0     14.1
  18.0     14.2     yes      3.0     18.0
```

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="line" width="560" height="320" role="img"
        aria-label="Measured demod EVM versus injected SNR against the theoretical curve"></canvas>
<script type="application/json" class="lab-chart-data">
{ "series":[
{"label":"measured","dots":true,"points":[[2,34],[6,22],[10,12],[14,6],[18,3],[22,2.1],[26,1.6],[30,1.3]]},
{"label":"theory","dashed":true,"points":[[2,30],[6,19],[10,10],[14,5],[18,2.5],[22,1.8],[26,1.4],[30,1.2]]}],
"xlabel":"injected SNR (dB)","ylabel":"EVM (%)","xmin":2,"xmax":30,"ymin":0,"ymax":40 }
</script>
<figcaption>Measured demod EVM (solid, dotted markers) tracks the theoretical curve (dashed) closely down to ~6 dB, then diverges as the demod runs out of margin — the divergence point is the number you're really benchmarking.</figcaption>
</figure>

The read is the *gap* between measured and theory. A demod that hugs the
theoretical curve down to low SNR and only diverges near the noise floor is
performing well; a demod that peels away early — losing lock or ballooning EVM
several dB above where theory says it should still be fine — has a deficiency
worth chasing. The **divergence point** is the single most useful number the
sweep produces: it's the honest floor of your demodulator.

## Both demod paths

The sweep runs **both** P25 Phase 1 demod paths in one go, each against its own
theoretical reference, because they're different algorithms with different
error curves:

```go
// cmd/gophertrunk/siglab_sweep.go
modes := []struct{ ... }{
    {"c4fm", "c4fm", "", metrics.SER4PAM, 2},
    {"cqpsk", "lsm", "cqpsk", metrics.SERCoherentQPSK, 2},
}
```

The **c4fm** path (the four-rail FM family) is checked against a 4-PAM
symbol-error reference; the **cqpsk** path (synthesized as `lsm`, the
linear-simulcast-friendly modulation) is checked against a coherent-QPSK
reference. Sweeping them together is what lets you answer a real design question —
"is my change an improvement on *both* paths, or did I help one and hurt the
other?" A tweak that lowers the c4fm divergence point but raises cqpsk's isn't a
clean win, and only a two-path sweep exposes that trade.

## The CSV and the regression workflow

Pass `-csv` and the sweep also writes a machine-readable table with a fixed
column set:

```text
mode,snr_db,es_n0_db,locked,evm_pct,snr_est_db
c4fm,2.0,-1.8,false,34.00,3.10
c4fm,6.0,2.2,true,22.00,6.80
cqpsk,6.0,2.2,true,24.50,6.40
```

Those six columns — `mode`, `snr_db`, `es_n0_db`, `locked`, `evm_pct`,
`snr_est_db` — are the whole regression story. The workflow Reese runs on every
demod change: sweep with a fixed seed *before* the change and save the CSV, make
the change, sweep again with the *same* seed, and diff the two CSVs. Same
synthesized input, same noise draw, so any difference in `locked`, `evm_pct`, or
`snr_est_db` is attributable to the change alone. Load both into the **Compare**
tab (Part 3) to overlay them, and export the merged result as JSON / JSONL / YAML
/ CSV for the record. That's how a demod change earns its way into the tree — a
curve that moved the right way on both paths, reproducibly.

## SNR, Es/N0, and the lock threshold

Two of the sweep's columns deserve a word, because conflating them is a common
mistake. `snr_db` is the SNR the sweep *injected* — the knob you turned.
`es_n0_db` is the energy-per-symbol to noise-density ratio the theoretical
reference is expressed in, derived from the injected SNR and the samples-per-symbol
(P25 Phase 1 runs 48 kHz / 4800 = 10 samples per symbol). They move together but
they aren't the same number, and the theoretical symbol-error curve lives in
Es/N0 space, which is why the sweep computes and prints both — so measured and
theory are compared on the same footing rather than across a hidden conversion.

The number most operators actually care about, though, is the **lock threshold** —
the lowest SNR rung where `locked` flips from `no` to `yes`. That threshold is a
compact, honest figure of merit for the demodulator: a lower threshold means the
demod holds on to weaker signals, which is exactly the improvement most demod work
is chasing. Watch it alongside the EVM curve, because the two tell a fuller story
together. A change that lowers the lock threshold by 2 dB but *worsens* EVM at the
top of the ladder has traded steady-state cleanliness for weak-signal
sensitivity — sometimes a good trade, sometimes not, but never one you'd notice
from a single capture. The sweep makes the trade visible, which is the entire
reason it exists: real demod decisions are about the shape of the whole curve, not
one point on it.

One caution Reese repeats: the sweep is synthesized AWGN, the cleanest possible
adversary. A demod that looks great on the ladder can still struggle on a real
capture full of multipath, phase noise, and interference the sweep never
injected. Treat a good sweep as *necessary but not sufficient* — it proves the
demod is sound against the textbook channel, and then you confirm against real
recordings and the stacked-impairment fixtures from Part 6. The ladder is the
controlled experiment; the field is the exam.

## Series wrap: the Lab Bench trilogy from here

That closes **Signal Lab**. Across ten parts you went from a first no-radio
replay to reading the dashboard, driving the browser console, seeing modulation
quality in constellations and eyes, measuring spectrum and occupancy,
synthesizing references, taking VSA measurements, naming the unknown, dissecting
P25 to the signaling block, and benchmarking the demodulator itself. The unifying
idea never changed: it's the *same production pipeline* the daemon runs, fed from
a file — so everything you measured here is what you'd get on the air.

But the story isn't over, because **Mercury** isn't decoded. Signal Lab named it
a candidate — ~4800 sym/s, 4-level-FSK-like — and handed the wideband capture on.
The next instrument in the trilogy is
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}), which treats the
airwaves like a protocol analyzer: it segments IQ into bursts, builds a protocol
hierarchy, graphs channel activity, and triages payload entropy — flagging
Mercury as an unknown, intermittent emitter and emitting its frames for the last
stage. Those frames land in
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}), where the trilogy
resolves: Mercury turns out not to be strong encryption at all, and "obfuscation
is not encryption" gets the last word. If Signal Lab taught you to *measure* a
signal, RF Scope teaches you to *map* it and Crypto Lab teaches you to *break* it.

## FAQ

**What does the sweep actually vary?**
Injected SNR, across a ladder from `-snr-min` to `-snr-max` in `-snr-step`
increments (default 2–30 dB by 2). At each rung it synthesizes P25 Phase 1,
decodes it, and records lock, EVM, and estimated SNR against a theoretical
symbol-error reference.

**Why sweep both c4fm and cqpsk?**
They're separate demod algorithms with separate error curves. Sweeping both
against their own references shows whether a change helps universally or trades
one path against the other.

**How do I use it as a regression test?**
Fix the `-seed`, sweep and save the CSV before your change, sweep again after with
the same seed, and diff. Identical input and noise draw mean any difference is
your change. The default seed is `0x5175`.

**Where do I go after Signal Lab?**
To [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}) to map signals into a
protocol hierarchy, and to
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) to analyze their
payloads — the other two legs of the Lab Bench trilogy, where the Mercury thread
finally resolves.

## Series navigation

**Part 10 of 10** · ←[Part 9]({{ '/blog/tutorials/signal-lab-09-dissecting-p25-pdus/' | relative_url }}) · Back to the [Signal Lab series hub]({{ '/blog/series/signal-lab/' | relative_url }})
