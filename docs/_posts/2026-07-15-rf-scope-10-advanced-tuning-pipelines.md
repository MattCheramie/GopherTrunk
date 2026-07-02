---
title: "RF Scope, Part 10: Advanced — Tuning Segmentation, Selective Analyzers & Pipelines"
description: The advanced rfscope guide — run a subset of analyzers with dependencies auto-pulled, tune segmentation for unusual bands, emit machine formats (json/jsonl/yaml/csv) for pipelines, and close the reverse-engineering loop from -frames-out into cryptolab, with a note on how rfscope's downconverter relates to GopherTrunk's DSP paths.
category: tutorials
keywords: rfscope, advanced, selective analyzers, dependency resolution, segmentation tuning, output formats, csv, jsonl, pipeline, frames-out, cryptolab, ddc, ccdecoder, gophertrunk
tags: [rfscope, rf-analysis, advanced, gophertrunk, dsp, cryptolab]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 10
---

*Part 10 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer — the
advanced tour, and the close of the Mercury thread.*

> **TL;DR:** Run a subset with `-analyzers timeline,timing` and the registry
> topologically pulls in dependencies automatically. Tune segmentation for odd bands
> with `-fft`, `-peak-threshold-db`, `-min-spacing`, and `-channel-rate`. Emit
> machine formats with `-out-format json|jsonl|yaml|csv -out scene.json` for
> pipelines. Close the reverse-engineering loop with `-frames-out` into cryptolab.
> And note: RF Scope uses the single-tap `ccdecoder` downconverter, **not** the
> wideband `internal/dsp/tuner` channelizer.

**Key takeaways**

- **Selective analyzers save time** — ask for what you need; the registry pulls
  dependencies in and orders them.
- **Segmentation is the tuning surface** for unusual bands — four flags cover most
  cases.
- **Machine formats make RF Scope a pipeline stage** — `jsonl` streams records, `csv`
  is the burst table.
- **`-frames-out` is the RE loop** — detection to byte-analysis without leaving
  GopherTrunk.

## Cheat sheet

| Flag | What it does |
|---|---|
| `-analyzers timeline,timing` | Run a subset; dependencies auto-pulled |
| `-fft 8192` | Larger wideband FFT — finer carrier resolution |
| `-peak-threshold-db 6` | Lower bar — catch weaker carriers |
| `-min-spacing 6250` | Tighter channel raster |
| `-channel-rate 100000` | Wider per-channel baseband for wide signals |
| `-out-format jsonl -out scene.jsonl` | Stream one record per entity |
| `-out-format csv` | The burst table |
| `-frames-out frames.jsonl` | Emit the cryptolab `ks` frames file |

## In this post

- **Selective analyzers** and the dependency-resolving registry.
- **Segmentation tuning** for bands the defaults miss.
- **Machine output formats**, including the CSV burst columns.
- **The reverse-engineering loop** into Crypto Lab.
- **Sidebar:** which downconverter RF Scope uses, and why it matters.
- **Series wrap** and the Mercury reveal.

## Selective analyzers and the registry

You do not have to run everything. `-analyzers` takes a comma-separated list, and the
registry does the rest:

```bash
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 \
    -analyzers timeline,timing -out-format json -out scene.json
```

Ask for `timing` and you also get `timeline`, because `timing` declares it as a
dependency and `Run` pulls the transitive closure in, then **topologically sorts** so
every analyzer runs after the ones it depends on:

```go
// internal/rfscope/analyzer.go
type Analyzer interface {
    Name() string
    Synopsis() string
    DependsOn() []string
    Analyze(ctx context.Context, sc *Scene, in *Input) error
}
```

`rfscope list` prints the graph, with each analyzer's dependencies:

```text
Registered rfscope analyzers:
  entropy      bitstream entropy / encryption triage (cryptolab bridge)  (after: topology)
  expert       RF expert-info / anomaly flags  (after: topology, timeline, entropy)
  hierarchy    RF protocol hierarchy (class → bandwidth → protocol)
  timeline     per-channel activity timeline / I/O graph
  timing       burst timing, inter-arrival, and TDMA-period stats  (after: timeline)
  topology     emitter clustering + conversation graph
```

The design is worth appreciating: analyzers **register themselves from `init()`**, so
adding one is a new file plus a blank import — no central switch to edit. `DependsOn`
is the only coupling, and `Run` resolves it with a cycle-detecting topological sort.
Ask for just `expert` and you transparently get `topology`, `timeline`, and `entropy`
underneath it, in the right order. Skipping the analyzers you do not need is the
cheapest speed-up on a big capture: `-analyzers hierarchy` alone skips all the
per-emitter demodulation entropy and topology do.

## Tuning segmentation for weird bands

When the defaults miss something, the fix is almost always in segmentation (Part 2),
not the analyzers. Four flags cover the common cases:

- **`-fft`** (default 4096, power of two). Raise to 8192 or 16384 for **finer carrier
  resolution** — closely-spaced channels the default blurs together become distinct.
  Lower it for more averaging and a smoother floor on a noisy band. Non-powers-of-two
  are rejected back to 4096.
- **`-peak-threshold-db`** (default 10). Lower to **6–8** to catch weak carriers near
  the noise floor; raise it to reject marginal humps on a band full of leakage.
- **`-min-spacing`** (default 12500). Set it to your band's channel raster — **6250**
  for 6.25 kHz channels, **25000** for a wide 25 kHz plan — so adjacent channels are
  neither merged nor split.
- **`-channel-rate`** (default 50000). Raise to **100000+** if you are chasing a signal
  wider than the default baseband can hold; the decimated channel must be wide enough
  to contain the whole occupied bandwidth.

A worked example — a dense 6.25 kHz NXDN-style band with weak carriers:

```bash
gophertrunk rfscope analyze -in dense.cfile -sample-rate 2400000 -freq 154000000 \
    -fft 16384 -peak-threshold-db 7 -min-spacing 6250
```

Reese's advice: *"change one knob at a time and re-read the channel table. If carriers
are merging, it is spacing or FFT size; if they are missing, it is the threshold; if
they are clipped, it is the channel rate."*

## Why the registry design matters

The self-registering, dependency-resolving registry is not just tidy — it is what keeps
RF Scope extensible without a rewrite. Because every analyzer is discovered at `init()`
and coupled to the others only through `DependsOn`, the pipeline has no central list of
"what to run in what order." Add an analyzer that depends on `entropy` and `timing`, and
`Run` will slot it after both automatically, everywhere: in `analyze`, in `live`, in the
cockpit, and in the web console, with no edit to any of them. The topological sort even
detects a dependency **cycle** and errors rather than looping, so a mistaken
`DependsOn` fails loudly at run time instead of hanging.

For an operator, the practical upshot is that `-analyzers` is safe to use aggressively.
You never have to know an analyzer's dependencies — ask for the *view* you want and the
machinery supplies the rest in the right order. Want only the anomaly list? `-analyzers
expert` transparently runs topology, timeline, and entropy first, because expert cannot
be computed without them. Want a fast structural summary of a huge capture? `-analyzers
hierarchy,timeline` skips the per-emitter demodulation that topology and entropy do,
which is where most of the analysis time goes. The dependency graph, printed by `rfscope
list`, is your map of what each request will actually cost.

## Machine output formats for pipelines

`summary` is for humans; the other four are for machines. Choose with `-out-format` and
write to a file with `-out`:

- **`json`** — the whole Scene as one indented object. Feed it to `jq`, a notebook, or
  a dashboard.
- **`jsonl`** — one JSON record per line: a `scene` header, then one record per
  `burst`, `channel`, `emitter`, `conversation`, and `anomaly`, each tagged with its
  `kind`. Stream-friendly and grep-friendly — pull just the anomalies with
  `jq 'select(.kind=="anomaly")'`.
- **`yaml`** — the same Scene as a single YAML document, for config-style consumption.
- **`csv`** — the **burst table**, one row per burst, with these columns:

```text
id, freq_hz, start_sec, end_sec, duration_sec, class,
occupied_bw_hz, channel_power_dbfs, spectral_flatness, snr_db, emitter_id
```

The CSV drops straight into a spreadsheet or pandas for slicing — group by `class`,
scatter `spectral_flatness` against `occupied_bw_hz`, filter by `emitter_id`. Every
float is clamped to a finite value before encoding, so the output is always valid — no
stray `NaN` or `Inf` reaches the file (or the web console's live wire).

```bash
# Every burst as CSV, for a notebook
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 -out-format csv -out bursts.csv

# Just the anomalies, via jsonl + jq
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 -out-format jsonl \
  | jq -c 'select(.kind=="anomaly")'
```

## The reverse-engineering loop

The most powerful pipeline RF Scope enables is the **detection → byte-analysis** loop,
closed without leaving GopherTrunk:

```bash
# 1. Segment, analyze, and emit frames for every unknown payload
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 -frames-out frames.jsonl

# 2. Triage the bytes in the cryptolab toolkit
gophertrunk cryptolab classify auto -in frames.jsonl        # needs -tags cryptolab
gophertrunk cryptolab ks reuse -in frames.jsonl             # if IVs/MIs repeat
```

`-frames-out` writes the `{label, iv, ct}` JSONL that cryptolab parses directly (Part
7), and RF Scope even prints a `cryptolab ks reuse` suggestion when it detects frames
sharing an IV. Detection lives in RF Scope; the byte-level attack lives in Crypto Lab;
the frames file is the seam between them.

<aside class="lab-note">
<p><strong>Sidebar — which downconverter?</strong> RF Scope's segmentation shifts each
carrier to baseband with the single-tap <code>ccdecoder</code> downconverter
(<code>ccdecoder.NewDownconverterWithOffset</code>) — the same single-channel path
<code>replay -tune-hz</code> uses — <em>not</em> the wideband multi-tap
<code>DDCBank</code>/channelizer in <code>internal/dsp/tuner</code>. The two are
separate code paths: a fix to one does not touch the other. RF Scope processes one
discovered carrier at a time, so a per-carrier single-tap DDC is the right tool; the
wideband channelizer exists for the live scanner that must hold dozens of taps open at
once. Both normalize to a per-channel rate, which is why RF Scope's analysis is
rate-invariant — the capture rate never leaks into a measurement.</p>
</aside>

## Series wrap: the Mercury reveal

Ten parts ago, Mercury was a vague unease — a short, intermittent burst near 453 MHz
that [Signal Lab]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
could not name. Trace what RF Scope did with it:

- **Part 2** sliced its faint transmissions into bursts.
- **Part 3** put it in the hierarchy as an FSK 12.5 kHz bucket with **no protocol
  child** — a genuine unknown.
- **Part 6** collapsed its scattered bursts into **one frequency hopper**, explaining
  why no single channel showed a clean period.
- **Part 7** blind-demodulated its payload, graded it **not-obviously-strong**
  obfuscation, and wrote its bytes to a frames file.
- **Part 8** stacked the flags — hopper, intermittent, obfuscated — into the expert
  panel.

RF Scope's job ends at the frames file. The
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series picks Mercury up
from there, and the twist lands in
[Crypto Lab Part 10]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }}):
Mercury was **not strong encryption at all**, but a **keyless, length-seeded byte
obfuscator** — talker-alias style — recovered by the `alias` subject framework and
graded BROKEN. RF Scope was right to grade it *not-obviously-strong*: the structure was
there to be found, and the entropy triage said as much. *Obfuscation is not
encryption*, and RF Scope's entire reason to exist is to help you tell the difference
before you assume the worst.

That is the trilogy. [Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) names
and measures one signal; **RF Scope** maps the band and triages the unknowns;
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) breaks what comes out.
The full [rfscope reference]({{ '/rfscope.html' | relative_url }}) has every flag; this
series was the tour.

## FAQ

**If I run one analyzer, do I get its dependencies automatically?**
Yes. The registry expands the transitive `DependsOn` closure and topologically sorts
it, so `-analyzers timing` also runs `timeline`, and `-analyzers expert` runs topology,
timeline, and entropy underneath — in the correct order.

**Which output format should I script against?**
`jsonl` for streaming and filtering (one tagged record per entity), `csv` for the burst
table in a spreadsheet or notebook, `json`/`yaml` for the whole Scene as one document.
All are clamped to finite floats, so they always parse.

**Why does RF Scope not use the wideband channelizer?**
It analyzes one discovered carrier at a time, so the single-tap `ccdecoder`
downconverter is the right fit. The wideband `internal/dsp/tuner` channelizer serves the
live scanner holding many taps open at once — a separate path.

**How do I actually break a signal RF Scope flags?**
Emit it with `-frames-out`, then follow the recommended cryptolab command from the
entropy verdict — `brute xor`, `lfsr bm`, `ks reuse`, and so on. RF Scope detects and
hands off; Crypto Lab breaks.

## Series navigation

**Part 10 of 10** · ←
[Part 9: The Scene Cockpit]({{ '/blog/tutorials/rf-scope-09-scene-cockpit-tui-web/' | relative_url }})
· Back to the [RF Scope series hub]({{ '/blog/series/rf-scope/' | relative_url }})
