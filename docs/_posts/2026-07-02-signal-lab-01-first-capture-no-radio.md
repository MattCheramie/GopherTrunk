---
title: "Signal Lab, Part 1: Your First Capture — No Radio Required"
description: Signal Lab is GopherTrunk's offline signal-analysis workbench — replay, identify, synthesize, and grade recorded IQ captures through the exact production decode pipeline with no SDR, no daemon, and no network attached.
category: tutorials
keywords: signal lab, gophertrunk siglab, offline iq analysis, sdr capture replay, iq file format f32 u8, cfile, p25 decoder, signal analysis workbench, no radio decode, protocol picker
tags: [siglab, sdr, iq, getting-started, dsp, p25]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 1
---

*Part 1 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. Signal Lab is one leg of the **Lab Bench trilogy** —
three sister series (Signal Lab, [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}),
and [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }})) that follow the
same recorded captures from raw IQ to decoded meaning to broken cipher. A single
mystery signal, **Mercury**, threads through all thirty posts; you'll meet it
soon.*

> **TL;DR:** `gophertrunk siglab` is an offline workbench that replays a recorded
> IQ capture through the *same* decode pipeline the live daemon uses — so what
> you see in the lab is what you'd get on the air. No SDR, no daemon, no network.
> It ships in the default install (no build tag) and has three front-ends: a
> terminal app, a browser console (`siglab serve`), and a demod benchmark
> (`siglab sweep`). Point it at a `.cfile`, pick a protocol, read the dashboard.

**Key takeaways**

- **Signal Lab is an analysis workbench, not a scanner.** It works on recorded
  IQ files, so you can diagnose a marginal capture on a laptop on a plane.
- **Same pipeline as the daemon.** Every metric comes from the production
  decoder, so lab results transfer directly to live behavior.
- **Three front-ends, one binary** — `siglab`, `siglab serve`, `siglab sweep` —
  chosen by how much you want to *see* versus *script*.
- **Format matters.** GopherTrunk's own `capture` writes interleaved `f32`;
  `rtl_sdr`-style `.cu8`/`.cfile` files are `u8`. Set `-format` to match or the
  bytes are garbage.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `gophertrunk siglab -in capture.cfile -sample-rate 2400000` | Replay a capture in the terminal app |
| `-protocol p25p1` | Skip the picker and decode a named protocol |
| `-format u8 \| f32` | Sample format of the capture (default `f32`) |
| `-auto-tune` | Estimate carrier offset and tune to 0 Hz before demod |
| `gophertrunk siglab serve` | Launch the browser console at `127.0.0.1:8099` |
| `gophertrunk siglab sweep` | Benchmark demod quality across an SNR ladder |

## In this post

- **What Signal Lab actually is** — an offline workbench sharing the daemon's
  decode path.
- **The three front-ends** and when to reach for each.
- **Your first replay** — the one command, the protocol picker, the dashboard.
- **IQ formats** — why `f32` versus `u8` is the mistake everyone makes once.
- **The cast and the trilogy** — Ada, Reese, and the Mercury thread.

## What Signal Lab is

**Signal Lab is GopherTrunk's standalone signal-analysis workbench.** It runs
entirely offline against *recorded* IQ captures — no radio attached, no daemon
running, no network — which makes it the natural home for three jobs: diagnosing
a marginal capture that won't lock, proving out a demodulator change, and
building regression fixtures you can check into a repo.

The single most important thing to understand is this:
[everything Signal Lab shows comes from the same decode pipeline the live daemon
uses]({{ '/siglab.html' | relative_url }}). It is not a re-implementation, not a
simplified teaching model, not a separate "analysis" decoder that drifts out of
sync with production. When the lab says a P25 control channel locked at 19.7 dB
SNR with 7.4% EVM, that is the production receiver's own verdict on those exact
samples. So the lab is a *microscope on production behavior*, and the numbers
you learn to read here are the numbers that decide whether the daemon hears a
system on the air.

That equivalence is the whole reason the workbench is useful. A recorded capture
is deterministic — the same bytes, every run — so you can replay a failure a
hundred times, change one knob, and see exactly what moved. On a live radio you
can never step in the same river twice.

<figure class="lab-figure">
<svg viewBox="0 0 660 140" width="660" height="140" role="img" aria-label="Pipeline from IQ capture through the production decoder to the dashboard">
  <rect x="8" y="46" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="68" text-anchor="middle" fill="currentColor" font-size="12">IQ capture</text>
  <text x="68" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="10">.cfile / .cu8</text>
  <line x1="128" y1="70" x2="180" y2="70" stroke="currentColor"/>
  <polygon points="180,66 190,70 180,74" fill="currentColor"/>
  <rect x="190" y="46" width="150" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="265" y="66" text-anchor="middle" fill="var(--accent)" font-size="12">production</text>
  <text x="265" y="82" text-anchor="middle" fill="var(--accent)" font-size="12">decode pipeline</text>
  <line x1="340" y1="70" x2="392" y2="70" stroke="currentColor"/>
  <polygon points="392,66 402,70 392,74" fill="currentColor"/>
  <rect x="402" y="46" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="462" y="68" text-anchor="middle" fill="currentColor" font-size="12">dashboard</text>
  <text x="462" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="10">metrics + events</text>
  <line x1="522" y1="70" x2="574" y2="70" stroke="currentColor"/>
  <polygon points="574,66 584,70 574,74" fill="currentColor"/>
  <rect x="584" y="46" width="68" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="618" y="74" text-anchor="middle" fill="currentColor" font-size="11">export</text>
  <text x="265" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the same code the live daemon runs — the lab just feeds it from a file</text>
</svg>
<figcaption>A sample flows from a recorded file through the production decoder to the dashboard — no radio in the loop.</figcaption>
</figure>

## The three front-ends

Signal Lab is one binary with three faces. They share the engine; they differ in
how much they let you *see* and how easily you can *script* them.

| Surface | Command | Best for |
|---|---|---|
| **Terminal app (TUI)** | `gophertrunk siglab` | quick one-capture replay + dashboard in a shell |
| **Browser console** | `gophertrunk siglab serve` | rich visuals, synthesis, comparing captures |
| **Demod benchmark** | `gophertrunk siglab sweep` | measuring demod quality across an SNR ladder |

The **terminal app** is where you start: one capture, one command, a live event
log, and a signal-quality dashboard. The **browser console** (Part 3) is the
richest surface — constellations, eye diagrams, spectrograms, a synthesis
engine, and side-by-side capture comparison — served locally at
`http://127.0.0.1:8099/`. The **demod benchmark** (Part 10) is the lab-coat
instrument: it synthesizes P25 across an SNR ladder on both demod paths and
prints measured quality against theory, so you can answer "did my change
actually help?" with a number.

None of them needs an opt-in build tag; Signal Lab is part of the default
install. On Windows the installer even drops a **Signal Lab console** shortcut in
the Start Menu next to the standard consoles.

## Your first replay

Here is the entire getting-started experience. Point the terminal app at a
capture and give it the sample rate the file was recorded at:

```bash
gophertrunk siglab -in capture.cfile -sample-rate 2400000
```

If you don't pass `-protocol`, the TUI opens on a **protocol picker** — the list
of every protocol the decoder knows, driven by the same registry the daemon
uses. Move with `↑`/`↓` (or `k`/`j`), press `enter`, and the engine runs. You'll
watch decode events stream past — NAC, color code, grants — and then land on the
dashboard, the signal-quality readout we take apart field by field in Part 2.

Prefer to skip the picker? Name the protocol and the format up front:

```bash
gophertrunk siglab -in capture.cfile -protocol p25p1 -format u8 -auto-tune
```

`-auto-tune` estimates the capture's carrier offset and tunes it to 0 Hz before
demodulation — handy when you recorded a little off-center. Under the hood, the
TUI is a small self-contained program that launches the engine in a goroutine
and pumps its events into the dashboard:

```go
// cmd/gophertrunk/siglab_tui.go
res, err := siglab.RunStream(capture, cfg, func(ev siglab.EventRecord) {
    select {
    case eventCh <- ev:
    default: // drop on a slow UI rather than block the engine
    }
})
```

That `RunStream` call is the same entry point the daemon's replay path leans on —
which is exactly why the lab and the air agree.

## IQ formats: `f32` versus `u8`

The one mistake nearly everyone makes once is feeding the lab a file in the wrong
sample format. An IQ capture is just interleaved I/Q samples on disk, but the
*width and encoding* of each sample varies by tool:

- **`f32`** — interleaved 32-bit floats, two per sample (I then Q). This is what
  GopherTrunk's own `gophertrunk capture` writes, and it's the lab's default.
- **`u8`** — interleaved unsigned 8-bit bytes, the classic `rtl_sdr` format.
  Files named `.cu8` (and many `.cfile`s from the RTL ecosystem) are `u8`.

The extension is a hint, not a contract. If you set `-format` wrong, the decoder
reads the bytes at the wrong stride and sees noise — no lock, garbage
histogram, nothing to diagnose. When a capture that *should* decode produces
pure noise, check the format before anything else. For the deeper story on what
IQ samples are, see the reference on
[IQ data]({{ '/reference/iq-data/' | relative_url }}) and
[software-defined radio]({{ '/learn/rf-sdr/' | relative_url }}).

The second most common surprise is the **sample rate**. The lab defaults to
`-sample-rate 2400000`, but if the file was recorded at, say, 2.5 MS/s and you
leave the default, the effective baud will read a few percent off and the
decoder may never settle. Part 2 turns that deviation into a diagnosis you can
trust.

## Meet the cast

Two people will walk through this series with you.

<aside class="lab-cast">
  <span class="lab-cast__badge">A</span>
  <div class="lab-cast__body">
    <p class="lab-cast__who">Meet the cast.</p>
    <p><strong>Ada</strong> just unboxed her first SDR and has a shoebox of
    captures she doesn't fully understand yet — she's our getting-started point
    of view, learning to read the dashboard one field at a time.
    <strong>Reese</strong> has been chasing signals for twenty years; he supplies
    the "why" behind the metrics and the war stories about the captures that lie
    to you.</p>
  </div>
</aside>

Ada's first replay locks on the third try — the first two were the wrong
`-format`, then the wrong `-sample-rate`. That's the normal path, and it's why we
spend Part 2 on reading the dashboard before anything fancier.

## The Mercury thread

Somewhere in Ada's shoebox is a capture near **453 MHz** — UHF business band —
that doesn't behave. Short, intermittent bursts on a roughly 12.5 kHz channel,
here and gone, and when she replays it nothing named decodes. Reese calls it
**Mercury**, half joking, and the name sticks.

Mercury is the thread that ties the whole Lab Bench trilogy together. In this
series it stays stubbornly unnamed until Part 8, where blind signal-ID gives a
best guess but no lock and Ada hands the wideband capture off to
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}). RF Scope triages it as
an unknown, apparently-encrypted emitter and passes its frames to
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}), where the twist
finally lands. Keep it in the back of your mind; we'll pick it up piece by piece.

## The series map

| Part | Topic | What you'll do |
|---|---|---|
| 1 | Your first capture (this post) | Replay a file with no radio |
| [2]({{ '/blog/tutorials/signal-lab-02-reading-the-dashboard/' | relative_url }}) | Reading the dashboard | Read lock, baud, EVM, SNR |
| [3]({{ '/blog/tutorials/signal-lab-03-browser-console-live-decode/' | relative_url }}) | The browser console | Drive `siglab serve` |
| [4]({{ '/blog/tutorials/signal-lab-04-constellations-and-eye-diagrams/' | relative_url }}) | Constellations & eye diagrams | See modulation quality |
| [5]({{ '/blog/tutorials/signal-lab-05-psd-spectrogram-occupancy/' | relative_url }}) | PSD, spectrogram, occupancy | Measure spectral shape |
| [6]({{ '/blog/tutorials/signal-lab-06-synthesize-references/' | relative_url }}) | Synthesize references | Build ideal/impaired fixtures |
| [7]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }}) | VSA & EVM | Lab-grade modulation quality |
| [8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }}) | Naming the unknown | Blind ID + wideband survey |
| [9]({{ '/blog/tutorials/signal-lab-09-dissecting-p25-pdus/' | relative_url }}) | Dissecting P25 PDUs | Field-level TSBK dissection |
| [10]({{ '/blog/tutorials/signal-lab-10-demod-bench-sweep/' | relative_url }}) | The demod bench | Sweep, regress, export |

## Where this goes next

[Part 2]({{ '/blog/tutorials/signal-lab-02-reading-the-dashboard/' | relative_url }})
takes the signal-quality dashboard apart field by field — protocol, symbols,
effective baud versus expected, lock status and latency, the symbol histogram,
IQ imbalance, decode-error rate, EVM, and the SNR estimate — and turns the
"~2% baud deviation" rule into a reliable wrong-sample-rate diagnosis. If you'd
rather see the whole workbench first, the [SigLab docs]({{ '/siglab.html' | relative_url }})
page is the canonical reference for every flag. And when you're ready to follow a
capture past the decoder into segmentation and protocol hierarchy, that's the
sister series, [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}).

## FAQ

**Do I need an SDR to use Signal Lab?**
No. That's the point. Signal Lab works entirely on recorded IQ files with no
radio, no daemon, and no network. You only need hardware to *make* a capture (or
grab one from a live daemon's console); analyzing it is fully offline.

**Is the lab decoder the same as the live one?**
Yes. Everything Signal Lab reports comes from the same production decode pipeline
the daemon runs. The lab just feeds it from a file instead of an antenna, so lab
results transfer directly to on-air behavior.

**What protocols can it decode?**
Every protocol the engine knows — P25 Phase 1/2, DMR, NXDN, dPMR, YSF, TETRA,
EDACS, MPT-1327, D-STAR, and LTR. Pass the name to `-protocol` or let the picker
rank candidates for you.

**Why did my capture decode to noise?**
Almost always the wrong `-format` (an `f32` default against a `u8`/`.cu8` file)
or the wrong `-sample-rate`. Fix the format first, then the rate, then re-run.

## Series navigation

**Part 1 of 10** · Next →
[Part 2: Reading the Dashboard]({{ '/blog/tutorials/signal-lab-02-reading-the-dashboard/' | relative_url }})
