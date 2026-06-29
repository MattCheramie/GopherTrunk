---
layout: page
title: SigLab — offline signal workbench
description: Replay, analyze, identify, synthesize, and grade RF captures with no radio attached — gophertrunk siglab as a terminal app, a browser console, and a demod benchmark
nav_group: Operate
---

# SigLab — offline signal workbench

**SigLab** is GopherTrunk's standalone signal-analysis workbench. It runs
entirely offline against **recorded IQ captures** — no SDR, no daemon, no
network — so it is the natural home for diagnosing a marginal capture, proving
out a demod change, or building a regression fixture.

Everything SigLab shows comes from the *same* decode pipeline the live daemon
uses (see the [Architecture reference](architecture.html)), so what you see in
the lab is what you get on the air.

SigLab has three front-ends, all built from one binary:

| Surface | Command | Best for |
|---------|---------|----------|
| **Terminal app (TUI)** | `gophertrunk siglab` | quick one-capture replay + dashboard in a shell |
| **Browser console** | `gophertrunk siglab serve` | rich visualization, synthesis, and comparing captures |
| **Demod benchmark** | `gophertrunk siglab sweep` | measuring demodulator quality across an SNR ladder |

It needs no opt-in build tag and is part of the default install — on Windows the
installer also drops a **Signal Lab console** Start-Menu shortcut alongside the
standard and Config Builder consoles (see the
[Windows user guide](user-guide-windows.html)).

## What it does

- **Replay** a raw IQ capture (`u8` or `f32`) through the production decoder and
  watch live decode events stream past.
- **Identify** the protocol of an unknown capture — ranked candidates rather
  than a guess.
- **Grade** a decode against acceptance criteria (the `test` surface), the way
  you would build or check a regression fixture.
- **Synthesize** an idealized or deliberately impaired capture (SNR, carrier
  offset/drift, multipath, DC, I/Q imbalance) to use as a clean reference.
- **Capture** a fixed-length raw-IQ recording from a live tuner — only when the
  console is served by a running daemon with an SDR — then analyze it
  immediately and download the `.cfile`.
- **Visualize** the result: symbol histogram, [constellation](constellation.html),
  PSD, spectrogram/waterfall, [eye diagram](eye-diagram.html),
  [symbol scope](symbol-scope.html), sync-landscape heatmap, receiver-state
  series, grants, and an event timeline.
- **Measure** lab-grade modulation quality (VSA): carrier-frequency error,
  RMS/peak EVM split into magnitude and phase error, IQ gain imbalance and
  quadrature skew, origin offset, an EVM-vs-symbol trace, and an error-vector
  spectrum — plus spectral occupancy off the PSD (99% occupied bandwidth,
  channel power, adjacent-channel power ratio, spectral flatness).
- **Name the unknown**: when a capture or a surveyed carrier does not decode,
  a blind symbol-rate / modulation estimate is ranked against an offline
  signal-ID reference database, so an undecodable carrier is still named a
  best guess rather than dropped.
- **Dissect the data**: with `collect_pdus`, every P25 TSBK signaling block
  (decoded and CRC-failed alike) is surfaced as a field-level dissection —
  opcode, source/destination/talkgroup, FEC/CRC status, and raw payload —
  filterable in the console and exportable as `csv-pdus`.
- **Compare** several analyzed captures — overlay their power spectra and diff
  their metrics — against each other or against a synthesized ideal.
- **Export** the structured result (JSON / JSONL / YAML / CSV) from the engine's
  own serializers.

## Terminal app: `gophertrunk siglab`

A self-contained replay/analysis TUI for a single capture.

```bash
gophertrunk siglab -in capture.cfile -sample-rate 2400000
gophertrunk siglab -in capture.cfile -protocol p25p1 -format u8 -auto-tune
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-in <path>` | *(required)* | raw IQ capture file |
| `-protocol <p>` | *(pick in the TUI)* | protocol to decode — empty prompts a picker |
| `-format u8\|f32` | `f32` | sample format of the capture |
| `-sample-rate <Hz>` | `2400000` | IQ sample rate of the capture |
| `-freq <Hz>` | `0` | informational nominal centre frequency |
| `-auto-tune` | off | estimate the carrier offset and tune to 0 Hz before demod |

Pick a protocol (if you didn't pass one), watch the decode run, read the
signal-quality dashboard, and press **`e`** to export a JSON report.

If the reported **effective baud** drifts more than ~2 % from the protocol's
expected symbol rate, the capture's true sample rate probably doesn't match
`-sample-rate` — fix the rate and re-run before trusting the metrics.

## Browser console: `gophertrunk siglab serve`

A standalone, fully offline web console served at `http://127.0.0.1:8099/`. It
ships the same RF analysis bundled into the binary — no Node.js at runtime, no
CDN assets — and is the richest way to *visualize* and *compare* captures.

```bash
gophertrunk siglab serve              # serve on 127.0.0.1:8099
gophertrunk siglab serve -open        # ...and open it in your browser
gophertrunk siglab serve -addr 0.0.0.0:9000
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-addr host:port` | `127.0.0.1:8099` | listen address |
| `-open` | off | open the console in the system browser once it is up |
| `-tmp-dir <dir>` | fresh temp dir | directory for staged capture uploads |
| `-max-upload <bytes>` | `512 MiB` | maximum capture upload size (`0` = default) |

A typical session: **upload** (or **synthesize**) a capture, **configure** the
engine (protocol, sample rate, tune, auto-tune, conjugate, IQ-correct, P25 deep
knobs), **run** it with a live event stream, read the results dashboard, then
**compare** it against another capture or a synthesized ideal and **export**.

The running daemon mounts the same `/api/v1/siglab/*` routes, so if you already
have a daemon up you can reach the console (and the live-capture surface) there
instead of starting a separate server.

## Demod benchmark: `gophertrunk siglab sweep`

Synthesizes P25 Phase 1 at a ladder of injected SNRs on both demod paths,
decodes each through the production pipeline, and prints the measured
demod-quality curve (lock, EVM, estimated SNR) against the theoretical
symbol-error-rate reference. It's the on-demand companion to the gated sweep
regression test — the answer to *"did my demod change actually help?"*

```bash
gophertrunk siglab sweep                                            # both paths, 2–30 dB
gophertrunk siglab sweep -snr-min 6 -snr-max 20 -snr-step 1 -csv sweep.csv
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-snr-min <dB>` | `2` | minimum injected SNR |
| `-snr-max <dB>` | `30` | maximum injected SNR |
| `-snr-step <dB>` | `2` | SNR step |
| `-seed <n>` | `0x5175` | AWGN seed (reproducible draws) |
| `-csv <path>` | *(none)* | also write the sweep as CSV |

See [Demod calibration](demod-calibration.html) for how to read the curve.

## SigLab vs. the bare subcommands

SigLab wraps a workbench of lower-level subcommands that you can also drive
directly when scripting:

- `gophertrunk capture` — record raw IQ off a live SDR to a `.cfile`.
- `gophertrunk replay` / `analyze` — decode a capture offline and export it.
- `gophertrunk identify` — auto-detect a capture's protocol, then analyze it.
- `gophertrunk gen` — synthesize a test IQ capture + metadata.
- `gophertrunk test` — grade a decode against acceptance criteria.
- `gophertrunk spectrum` — a capture's power spectrum and detected carriers.

The TUI and browser console are friendlier front-ends onto exactly these. To
discover and map an *unknown* trunked system from captures, reach for
[Hunt](hunt.html) instead.

## Supported protocols

SigLab decodes every protocol the engine knows, including P25 Phase 1/2, DMR
(Tier I/II/III), NXDN, dPMR, YSF, TETRA, EDACS, MPT-1327, D-STAR, and LTR. Pass
the protocol name to `-protocol`, or let `identify` / the TUI picker rank the
candidates for you.

## Notes & gotchas

- **Match the sample rate.** Most "it won't lock" surprises are a wrong
  `-sample-rate`. Watch the effective-baud deviation warning.
- **The capture, not the rate, decides quality.** Both down-converters normalize
  to the per-protocol channel rate, so the decode path is rate-invariant to the
  capture rate. A symptom that only appears at a higher capture rate but
  reproduces in offline replay points at the *captured samples* (front-end
  overload / intermod / gain staging), not GopherTrunk's DSP.
- **`f32` vs `u8`.** GopherTrunk's own `capture` writes interleaved `f32`;
  many SDR tools (and `.cu8`/`.cfile` from `rtl_sdr`) write `u8`. Set `-format`
  to match.
