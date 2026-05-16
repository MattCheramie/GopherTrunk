# samples/

Drop real-air captures here to close the remaining FEC follow-ups in
[`docs/opt-in-features.md`](../docs/opt-in-features.md) §5. Each
protocol subfolder has a `README.md` that describes:

- the capture format the loader expects (typically complex IQ at a
  documented sample rate),
- the metadata GopherTrunk needs alongside the capture to validate
  the decode (System ID, Color Code, NAC, expected message
  contents, etc.),
- the open-source tool or reference receiver each capture should
  cross-check against (MMDVMHost, DSDcc, DSD-FME, OP25, etc.).

## What lives here

| Subfolder | Protocol | Status | What captures buy |
| --- | --- | --- | --- |
| [`nxdn/`](nxdn/) | NXDN (NXDN-TS-1-A) | ⏳ Real-air capture pending (harness ready: [`integration_cc_nxdn_realair_test.go`](../cmd/gophertrunk/integration_cc_nxdn_realair_test.go)) | ≥ 80% CRC-verified CAC bursts + SystemID match + 3 s lock latency |
| [`ysf/`](ysf/) | Yaesu System Fusion | ⏳ Real-air capture pending | Validates MMDVMHost schedule choice for `EncodeFICHOnAir` / `DecodeFICHOnAir`; swap to DSDcc alternate if CRC fails |
| [`tetra/`](tetra/) | ETSI TETRA | ⏳ Real-air capture pending | 5 s lock latency + ≥ 90% frame recovery + Viterbi correction-depth histogram |
| [`dmr-tier2/`](dmr-tier2/) | DMR Tier II (conventional) | ✅ Pipeline closed (PR-C); captures optional | Burst-error structure validation + per-call payload diversity |
| [`mpt1327/`](mpt1327/) | MPT 1327 | ✅ CWSC tolerance closed (PR-A); captures optional | Empirical false-positive count + per-vendor sync bit-error patterns |

Each subfolder's README documents the capture format, metadata
schema, and **acceptance criteria** — the explicit numerical
thresholds a contributor with hardware can run the capture against
to close the corresponding follow-up.

## Audio-vs-IQ caveat

GopherTrunk's production pipelines start at **complex IQ baseband**
(`*.cfile` / `*.bin` / `*.iq`). Audio recordings (`*.mp3` / `*.wav`)
are post-FM-demodulation — they sit ONE STAGE DOWNSTREAM of the
receiver's first block.

Audio captures still work for protocols whose FEC chain operates on
audio-band tones (MPT 1327 FFSK, sub-audible LTR Manchester). They
**don't** work for protocols whose recovery needs IQ-domain
information:

- **TETRA** is π/4-DQPSK — phase information is lost in FM demod,
  so audio captures can't be decoded back to symbols.
- **NXDN / YSF** are 4-level FSK; the 4-level constellation lives
  in the audio amplitude, but MP3 compression at typical bitrates
  (128 kbps) blurs the levels enough that the matched filter
  collapses into the inner ±1 bins (confirmed empirically with the
  uploaded samples).
- **DMR Tier II** is C4FM at 4800 sym/s; same caveat as NXDN.

For the protocols that need IQ, drop a `*.cfile` / `*.bin` / `*.iq`
recording rather than an MP3.

## Smoke-test harness

[`cmd/audio_smoketest/main.go`](cmd/audio_smoketest/main.go) is a
small bypass-the-FM-stage tool that ingests post-FM-demod audio and
runs it through the FFSK / C4FM matched filter + MM clock recovery
+ state-machine chain. Build and run:

```
go run ./samples/cmd/audio_smoketest -file samples/mpt1327/MPT1327_423.6_1.mp3
```

The harness uses `ffmpeg` to decode the audio to PCM, so install
`ffmpeg` first (`apt-get install ffmpeg`). Per-protocol behaviour:

| Protocol | Sample | Result |
| --- | --- | --- |
| MPT 1327 | `MPT1327_423.6_1.mp3` (audio) | **Works** — 7 cc.locked + 1 grant, BCH-verified |
| MPT 1327 | `MPT1327_423.6_2.mp3` (audio) | **Works** — 7 cc.locked + 4 grants, SystemID `0x1fd7` |
| NXDN | `NXDN48 IQ.wav` (IQ) | Chain runs, 4-level slicer produces balanced dibits (26/27/15/32 %), but **FSW sync doesn't match** — file likely 4800-bps BFSK rather than 9600 4-FSK |
| NXDN | `NXDN96 IQ.wav` (IQ) | Chain runs, dibit distribution is **bimodal (3/50/3/44 %)** — outer ±3 levels dominate, inner ±1 underrepresented; consistent with a different deviation than the spec value |
| TETRA | `TETRA IQ.wav` (IQ) | **Sample rate too low** — 48 kHz gives only 2.67 sps at 18 ksym/s π/4-DQPSK; receiver needs ≥ 4 sps for reliable Gardner lock |
| YSF | (none retained) | Audio-only WAV removed; needs IQ to recover the 4-FSK constellation |
| DMR Tier II | (none retained) | n/a |

### What's retained

Only samples that **work today** or are **structurally usable**:

- `samples/mpt1327/MPT1327_423.6_1.mp3` and `MPT1327_423.6_2.mp3` — the two long captures that decode end-to-end through the production CWSC + BCH chain.
- `samples/nxdn/NXDN48 IQ.wav` and `NXDN96 IQ.wav` — real IQ captures; the receiver chain runs cleanly. Decode is blocked on protocol-variant (NXDN-BFSK support) or deviation calibration, not on the sample itself.
- `samples/tetra/TETRA IQ.wav` — real IQ capture; blocked on sample rate (need ≥ 72 kHz instead of 48 kHz).

### What was removed (and why)

Audio recordings that **structurally cannot decode** for protocols that need IQ:

- **NXDN MP3s** (8 files) — NXDN encodes information in the amplitude of a 4-FSK signal. 128 kbps MP3 compression on a 4-level discriminator output collapses the levels into the inner ±1 bins (empirically confirmed). Needs lossless IQ.
- **TETRA MP3s** (3 files) — TETRA is π/4-DQPSK; the information lives in the phase of the carrier, which is destroyed by FM demodulation regardless of compression. Audio captures cannot be decoded.
- **YSF WAV** (`Yaesu_sys_fusion.wav`) — same 4-level FSK collapse as NXDN MP3s, even though the WAV itself is lossless: the underlying recording is post-FM-demod audio, not IQ.
- **Short MPT 1327 MP3s** (7 files, all ≤ 30 s) — too short for the BCH alignment search to converge and produce a measurable decode outcome.

These files were tracked in git history; the deletion makes the
sample drop-zone reflect only captures that contribute to
validating decoders. Re-upload IQ-format equivalents to put any
of these protocols back on the radar.

### What works today

- **MPT 1327** — fully validated end-to-end against captured RF (audio path works because FFSK tones live in the audio band). The CWSC + BCH(64, 48, 2) chain shipped via PR #189 is confirmed working on real-air signals.

### What's blocked by the available samples

- **NXDN** — the IQ chain runs cleanly and produces dibits at the right rate, but no FSW sync. Two likely causes:
  - `NXDN48` is probably NXDN-BFSK (4800 bps, 2-level FSK) which our receiver doesn't support — the production receiver is hardcoded to `nxdn.Rate9600` (9600 bps, 4-level FSK). A BFSK-mode receiver is a separate body of work.
  - `NXDN96`'s bimodal dibit distribution suggests its on-air deviation differs from the 1800 Hz default the slicer is calibrated against. Tunable deviation (per-system YAML key) would help.
- **TETRA** — needs a higher sample-rate IQ capture (≥ 72 kHz) for reliable Gardner clock lock at 18 ksym/s.

### Smoketest harness

[`cmd/audio_smoketest/main.go`](cmd/audio_smoketest/main.go) ingests either:

- **Audio** (MP3/WAV) — converted via ffmpeg, routed through the protocol's matched filter + MM clock + state machine. Bypasses the receiver's FM-demod stage.
- **IQ WAV** (stereo 16-bit, SDR# convention with I in left channel, Q in right) — auto-detected from `IQ` in the filename, routed through the **real** receiver pipeline (`nxdnrx.New`, `tetrarx.New`) end-to-end.

Useful flags:

| Flag | Purpose |
| --- | --- |
| `-file <path>` | required |
| `-protocol mpt1327\|nxdn\|tetra\|ysf\|auto` | overrides folder-based auto-detection |
| `-swap-iq` | swaps I and Q channels (try if a capture won't lock) |
| `-conj-iq` | conjugates IQ (negate Q) — alternative spectrum-flip experiment |
| `-rate <hz>` | force a specific resample rate for audio input |
| `-gain <f>` | Mueller-Müller loop gain (default 0.05) |

Requires `ffmpeg` for MP3/audio input (`apt-get install ffmpeg`).

The MPT 1327 result is the actionable win: real signaling traffic
decoded from a downloadable audio sample, confirming the receiver
chain works end-to-end on captured RF — which validates the
CWSC sync + BCH(64, 48, 2) + codeword parsing path shipped in
PR #185 / PR #188 (commit `94464d0`).

## What's expected vs. what's committed

The per-protocol README in each subfolder lays out the **format** and
**metadata** GopherTrunk expects. The capture files themselves are
deliberately gitignored — they're typically multi-megabyte IQ
recordings that don't belong in source control. Two ways to share:

- **Small representative samples (≤ 1 MB)** — commit them directly;
  the `.gitignore` here only excludes large binary formats.
- **Larger captures** — drop them in the subfolder locally and link
  them from the subfolder README (Git LFS, GitHub Releases, or a
  separate fixture bucket).

A capture without a `metadata.json` describing the expected decode
output is fine for "does the decoder not crash" smoke tests but
isn't enough to **validate** correctness — the schema each subfolder
documents is what unblocks the corresponding follow-up.

## Wiring captures into tests

Captures dropped here aren't auto-loaded. To exercise one through a
decoder:

1. Read the subfolder README for the expected format.
2. Write a test (typically under `cmd/gophertrunk/` with
   `//go:build integration`) that streams the IQ file through the
   target protocol's pipeline factory and asserts the decoded events
   match `metadata.json`.

Example skeleton for an NXDN integration test:

```go
//go:build integration

package main

import "testing"

func TestNXDNAgainstCapture(t *testing.T) {
    iqPath := "../../samples/nxdn/example.cfile"
    meta := loadMetadata(t, "../../samples/nxdn/example.metadata.json")
    // ... feed iqPath through newNXDNPipeline + assert events.KindCCLocked
    //     + Grant payloads match meta.
}
```

See [`cmd/gophertrunk/integration_cc_dmr_test.go`](../cmd/gophertrunk/integration_cc_dmr_test.go)
for a fully-worked synthesized-fixture example whose shape applies
to real-air captures with minor adaptations.

