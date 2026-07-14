---
layout: page
title: GopherTrunk Bundle — the capture-to-analysis case format
description: Package a capture, logs, signal analysis, crypto analysis, and site/network mapping into one sharable .gtb.tar.gz that flows from the scanner to SigLab to CryptoLab and back
nav_group: Operate
---

# GopherTrunk Bundle (`.gtb.tar.gz`)

A **GopherTrunk Bundle** packages one SDR *case* — a capture together with
everything an investigation produces from it — into a single archive you can
share with the community as one file. It carries the raw IQ capture, a small
narrowband slice, the session logs, the SigLab signal analysis, the CryptoLab
crypto analysis, and the hunt site/network mapping, all indexed by a
human-and-machine-readable `MANIFEST.yaml`.

The goal is a **seamless loop**: hunt a signal → capture it into a bundle →
analyze it in SigLab → assess its encryption in CryptoLab → commit the enriched
mapping back into `config.yaml` and start scanning it — without hand-collecting a
dozen loose files at each step.

> **Naming note.** "Bundle" is overloaded in GopherTrunk. The *hunt CSV import
> bundle* (`hunt … -formats bundle`) is a multi-section CSV that round-trips a
> discovered system into `config.yaml`. That CSV is **stored inside** a
> GopherTrunk Bundle (as `mapping/system.bundle.csv`); it is not the archive
> itself. This page is always about the `.gtb.tar.gz` archive.

## What's inside

A single top-level directory (named after the case) holds:

```
mesa-p25-2026-07-14/
  MANIFEST.yaml          index: schema, provenance, and a role-tagged file list with sha256
  README.md              auto-generated human summary (open the tarball and read it)
  capture/               raw IQ (.cfile), a narrowband DDC slice (.wav), metadata (.yaml)
  logs/                  events.jsonl (all bus events), messages.log (human decoded messages)
  mapping/               system.json (the discovered system), survey.jsonl, hunt exports
  siglab/                result.yaml (SigLab analysis), events.jsonl
  cryptolab/             frames.jsonl (encrypted frames), assess.result.yaml (verdict)
```

Every file is listed in the manifest with its **role**, size, and **sha256**, so
`bundle verify` catches any corruption or tampering. The manifest and analysis
summaries are **YAML** (snake_case, editable by hand); streams are **JSONL**; the
IQ stays raw binary. Frequencies are integer Hz, timestamps UTC RFC3339 — the
same conventions as the rest of GopherTrunk.

## Capture-length policy

The manifest records a **capture intent** that documents why the case was
collected and sets the length of the auto-carved narrowband slice:

| Intent | Full IQ | DDC slice | Why |
|---|---|---|---|
| `cc-map` | ~60 s | 30 s | several status broadcasts + band plan + neighbor list |
| `crypto` | 60–120 s | 60 s | gather IV/MI reuse across calls |
| `survey` | ~30 s | 10 s | classification only |
| `general` | operator | 15 s | default |

The full wideband IQ is the source of truth; the slice is the small, shareable
channelized stream (48 kHz for the C4FM family, 144 kHz for TETRA — exactly what
the decoder sees) that replays identically with `siglab -format wav`.

## The workflow, end to end

```sh
# 1. Capture a control channel straight into a bundle (raw IQ + metadata + slice)
gophertrunk capture -freq 851012500 -sample-rate 2400000 -seconds 60 \
  -protocol p25 -out cc.cfile -bundle mesa-p25.gtb.tar.gz -intent cc-map

# 2. Or record one signal off a survey list and route it, packaging as you go
gophertrunk hunt -survey-capture '#3' -from survey.json -to cryptolab \
  -bundle mesa-p25.gtb.tar.gz

# 3. Analyze the bundled capture — protocol/rate come from the bundle itself —
#    and the SigLab result is written back into the same bundle
gophertrunk analyze -bundle mesa-p25.gtb.tar.gz

# 4. Inspect and check integrity
gophertrunk bundle info   -in mesa-p25.gtb.tar.gz -files
gophertrunk bundle verify -in mesa-p25.gtb.tar.gz

# 5. Hand just the crypto frames to a colleague
gophertrunk bundle extract -in mesa-p25.gtb.tar.gz -role cryptolab-frames -out ./send

# 6. Commit the discovered system back into config.yaml. Talkgroups the bundle's
#    crypto frames show under a non-clear algorithm are marked encrypted first.
gophertrunk bundle commit -in mesa-p25.gtb.tar.gz -config config.yaml -dry-run
```

## The `bundle` subcommands

| Subcommand | What it does |
|---|---|
| `pack` | assemble a bundle from a capture and/or analysis files, or a laid-out `-from-dir` |
| `info` (`ls`) | print the manifest summary; `-files` lists the full inventory |
| `verify` | recompute and check every file's sha256; non-zero exit on a mismatch |
| `extract` | unpack the whole tree, or one role's files with `-role` |
| `add` | append an artifact to an existing bundle |
| `commit` (`import`) | merge the bundle's discovered system into `config.yaml`, enriched by the crypto verdict |

Seams that write bundles directly: `capture -bundle`, `hunt -bundle` (mapping),
`hunt -survey-capture … -bundle` (a recorded signal + its SigLab/CryptoLab
handoff), and `analyze -bundle` (reads the capture from a bundle and writes the
result back).

## In the web consoles

The daemon serves the bundle endpoints under `/api/v1/bundle/*`
(`info`, `verify`, `download`), gated behind the `bundle` capability in
`GET /api/v1/runtime` so the consoles only surface them when reachable. The
SigLab and CryptoLab consoles use them to open a `.gtb`, load its capture or
frames into the workbench, and save results back; the main console's **Bundles**
view lists, inspects, verifies, and commits cases. Packing runs through the CLI;
the web side is read/inspect/commit.

## Forward compatibility

The manifest carries a `schema_version` (currently `1`). A newer bundle opens
best-effort in an older build — unknown manifest keys and unrecognized file
roles are preserved, not rejected — so the format can grow without breaking
older tools. Roles are only ever appended, never renamed.
