# NXDN captures

Drop **NXDN-TS-1-A** outbound control-channel IQ recordings here to
unblock end-to-end validation of the interleaver + puncture + K=5
Viterbi + CRC chain shipping in
[`internal/radio/nxdn/cac_channel.go`](../../internal/radio/nxdn/cac_channel.go).

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex float32 IQ (`*.cfile`) or complex int16 (`*.bin`) |
| Sample rate | Any rate ≥ 48 kHz; 48 kHz nominal |
| Modulation | 4-level FSK at 4800 symbols/s (NXDN-TS-1-A §3.2) |
| Channel width | 6.25 kHz or 12.5 kHz |
| Centre | Tuned on the outbound RCCH carrier (any DC offset OK; receiver re-tunes) |
| Duration | ≥ 5 seconds — enough to capture multiple CAC bursts |

## Metadata schema

Alongside each `*.cfile` / `*.bin`, place a `*.metadata.json` with at
least:

```json
{
  "source": "MMDVMHost log @ <site>",
  "tool_cross_check": "DSDcc 1.9.5",
  "expected": {
    "system_id": "0x1234",
    "site_id": "0x01",
    "ran": 1,
    "messages": [
      { "type": "Site Information", "details": "..." },
      { "type": "Voice Call Request", "from": "1234", "to": "G:100" }
    ]
  },
  "notes": "Optional free-text describing capture conditions, SNR, etc."
}
```

The decoder test will:

1. Stream the IQ through `newNXDNPipeline` (factory in
   `internal/scanner/ccdecoder/pipelines.go`).
2. Subscribe to the bus, collect `events.KindCCLocked` + `Grant`
   events.
3. Assert the decoded system ID / RAN / message sequence matches
   `metadata.expected`.

## Recommended sources

- **MMDVMHost** running on a clean RF path — its log file is the
  ground truth for `expected.messages`.
- **DSDcc** in MMDVM mode — cross-check decoder output.
- A **controlled test transmitter** (a known radio keyed up with
  known TG/site config) — easiest to label.

## Why captures are needed (not synthesized fixtures)

The synthesized round-trip in
[`process_spec_test.go`](../../internal/radio/nxdn/process_spec_test.go)
proves `EncodeCACChannel` → `DecodeCACChannel` is bit-correct but
doesn't catch:

- bit-ordering / endianness mismatches against on-air transmitters,
- vendor-specific deviations from §4.5.1.1 (some MMDVM forks
  diverge slightly in puncture index ordering),
- noise-margin behaviour the Viterbi corrector needs to handle.

A single captured + labeled outbound RCCH burst closes all three.
