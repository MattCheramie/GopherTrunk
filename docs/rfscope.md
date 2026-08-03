---
layout: page
title: RF Scope — protocol-agnostic RF network analysis
description: rfscope is Wireshark for the RF physical layer — point it at any band, a live SDR or an IQ capture, with no prior knowledge of the modulation, framing, or encryption, and get a structured analysis of what is on the air and how it behaves.
nav_group: Operate
---

# RF Scope — protocol-agnostic RF network analysis

`rfscope` is **Wireshark for the RF physical layer**. Point it at any band — a
recorded IQ capture or a live SDR — with **no prior knowledge of the technology,
modulation, framing, or encryption**, and it produces a structured analysis of
*what is on the air and how it behaves*: an RF protocol hierarchy, per-channel
activity over time, burst timing and periodicity, an emitter/conversation graph,
an entropy/encryption triage, and an expert-info anomaly list.

Where `hunt` discovers and maps *known* trunked systems and `siglab` decodes the
13 named protocols, `rfscope` is the layer below: it says something structural
about a signal even when it is *not* one of those — an unknown IoT mesh, a
telemetry link, a paging variant, an encrypted data burst, a frequency hopper.
It optimizes for non-WiFi RF (slow, channelized, bursty LMR/IoT/paging traffic)
but the same primitives work for WiFi-style bursty traffic too.

It adds no DSP of its own: it orchestrates the primitives the rest of GopherTrunk
already ships (the `survey` blind classifier, `carriers` peak detection,
`spectrum` occupancy metrics, `siglab` protocol identification, the `cryptolab`
randomness/keystream engines) and accumulates their output into one
JSON-exportable **Scene**.

## The analyzers

A Scene is built by a registry of pluggable analyzers (`rfscope list` prints
them). Each is the RF analog of a Wireshark feature:

| Analyzer | Wireshark analog | What it produces |
|---|---|---|
| `hierarchy` | Protocol Hierarchy | bursts grouped by modulation class → occupied-bandwidth bucket → identified protocol, with counts, airtime, spectrum/time share |
| `timeline` | I/O Graphs | per-channel occupancy/power over time, duty cycle, burst rate |
| `timing` | (timing stats) | burst-length & inter-arrival histograms, and the TDMA/frame **period** recovered by autocorrelating channel occupancy |
| `topology` | Conversations / Endpoints | bursts clustered into **emitters** by RF fingerprint (frequency hoppers collapse into one), then linked into **conversations** (co-active / request-response / hop-sequence); defers to `hunt`'s authoritative map when a trunking control channel is present |
| `entropy` | entropy-based encryption detection | for unidentified digital emitters, a byte-level triage (plaintext / substitution / repeating-xor / periodic-scrambler / lfsr-or-keyless-scrambler / strong-encrypted) built on the cryptolab randomness battery |
| `expert` | Expert Information | anomaly flags: frequency hoppers, intermittent emitters, abnormally wide/narrow carriers, noise-like (high spectral-flatness) carriers, and the encrypted/obfuscated/unknown findings from the entropy triage |

## Command-line

```
gophertrunk rfscope analyze -in <capture> [flags]            analyze a recorded IQ capture
gophertrunk rfscope live -serial <sdr> -freq Hz [flags]      analyze a live SDR span
gophertrunk rfscope cockpit [-in <cap> | -serial <sdr> -freq Hz]   live scene TUI
gophertrunk rfscope serve [-addr host:port] [-open]          web console (browser)
gophertrunk rfscope list                                     list registered analyzers
```

Common flags (shared by `analyze` and `live`): `-format u8|f32`, `-sample-rate`,
`-freq` (capture centre), `-fft`, `-peak-threshold-db`, `-min-spacing`,
`-channel-rate`, `-analyzers hierarchy,timing,…` (default: all),
`-out-format summary|json|jsonl|yaml|csv`, `-out <path>`, and `-frames-out
<path>` to emit a cryptolab `ks` frames file from any unknown payloads.
`analyze` adds `-window <sec>`; `live` adds `-duration <sec>` and `-tui` (open
the cockpit).

### Examples

```
# Summarize a wideband capture's RF scene
gophertrunk rfscope analyze -in wide.cfile -format f32 -sample-rate 2400000 -freq 451000000

# Full JSON scene, only the timeline + timing analyzers
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 \
    -analyzers timeline,timing -out-format json -out scene.json

# Live scene monitor on an SDR
gophertrunk rfscope cockpit -serial 00000001 -freq 451000000

# Emit a cryptolab frames file for the unknown payloads, then triage it
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 -frames-out frames.jsonl
gophertrunk cryptolab classify auto -in frames.jsonl     # needs -tags cryptolab
```

## Output formats

`summary` is a human-readable report (hierarchy tree, channel table, top talkers,
conversations, expert info). `json`/`yaml` emit the whole Scene; `jsonl` streams
one record per scene-header / burst / channel / emitter / conversation / anomaly;
`csv` is the burst table. Every float is clamped before encoding so the output is
always valid JSON.

## The cockpit (TUI)

`rfscope cockpit` (and `rfscope live -tui`) is a Bubbletea terminal dashboard:
panels for the protocol hierarchy, per-channel I/O-graph sparklines, top talkers,
conversations, and severity-colored expert info. Keys: `space` freeze/resume, `e`
export the current scene as JSON, `?` help, `q` quit.

## The web console

`rfscope serve` runs a standalone browser console (default
`http://127.0.0.1:8098/`): upload a capture and view the same scene with a
one-click **Crypto Lab** hand-off that downloads the recovered `ks` frames file
with the suggested next command. Build the SPA first with `make rfscope-web-build`
(the default binary serves an API-only placeholder until then).

## The Crypto Lab bridge

For a digital signal `siglab` cannot identify, `rfscope` recovers a byte payload
and triages it with the same measurements cryptolab's `classify auto` uses
(Shannon entropy, index of coincidence, repeating-XOR key length, a NIST
SP 800-22 randomness battery). It can emit those payloads as a cryptolab `ks`
frames file (`{"label","iv","ct"}` JSONL) that feeds `cryptolab classify auto`,
`ks reuse`/`mtp`, and `brute` directly — the detection→byte-analysis loop, closed
without leaving GopherTrunk. The in-binary triage needs no build tag; the full
`ks reuse` hand-off runs in a `-tags cryptolab` build.
