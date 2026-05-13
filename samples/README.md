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

| Subfolder | Protocol | Follow-up it unblocks |
| --- | --- | --- |
| [`nxdn/`](nxdn/) | NXDN (NXDN-TS-1-A) | Interleaver + puncture inner-layer end-to-end validation |
| [`ysf/`](ysf/) | Yaesu System Fusion | FICH interleaver / puncture schedule calibration |
| [`tetra/`](tetra/) | ETSI TETRA | On-air recovery margins (Viterbi correction depth profiling) |
| [`dmr-tier2/`](dmr-tier2/) | DMR Tier II (conventional) | Lifts the `TestDaemonCCDecodesDMRTier2` skip |
| [`mpt1327/`](mpt1327/) | MPT 1327 | CWSC bit-error tolerance threshold calibration |

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

| Protocol | Audio decode viability | Sigidwiki MP3 sample results |
| --- | --- | --- |
| MPT 1327 | **Works** — FFSK tones in audio band survive MP3 compression cleanly | 2 of 9 samples (the long `423.6_1` / `423.6_2` captures) emit real BCH-verified cc.locked + grant events |
| YSF | Partial — symbols flow through MM clock recovery but the 4-level amplitude structure collapses into ±1 (MP3-compression artefact) | 1 sample tested; no decode |
| NXDN | Partial — same 4-level audio-amplitude limitation as YSF | 8 samples tested; no decode |
| TETRA | **Not viable from audio** — phase modulation, FM demod is lossy | 3 samples (audio only; need IQ) |
| DMR Tier II | Same as NXDN; needs IQ | No samples uploaded |

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

