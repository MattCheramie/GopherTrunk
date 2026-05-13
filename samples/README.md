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
