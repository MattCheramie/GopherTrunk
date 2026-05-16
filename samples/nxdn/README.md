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

## Acceptance criteria

A capture is considered "validating" when:

1. **CRC-verified CAC burst rate.** ≥ **80% of CAC bursts**
   recovered through the IQ → 4-FSK slicer → §4.5.1.1 chain pass
   the trailing CRC-16. The threshold is intentionally lower than
   TETRA's 90% — NXDN's 6.25 kHz channel width + minimum-shift
   deviation gives less margin against adjacent-channel
   interference, and 80% is the rate MMDVMHost reports on
   comparable hardware.
2. **System metadata match.** The decoded SystemID + SiteID + RAN
   from the captured CAC bursts must match `metadata.expected`'s
   values byte-for-byte. CRC-passing bursts whose payload doesn't
   match the metadata flag a bit-ordering / endianness regression
   rather than a noise issue.
3. **Lock latency.** `events.KindCCLocked` within **3 seconds** of
   the first valid CAC burst's start in the capture (NXDN locks
   faster than TETRA because there's no Gardner step in the
   receiver chain).

The validation wires through `newNXDNPipeline`'s existing
integration test
[`cmd/gophertrunk/integration_cc_nxdn_test.go`](../../cmd/gophertrunk/integration_cc_nxdn_test.go)
— add a `_realair_test.go` sibling pointing the mock SDR at the
capture path.

## Tuning deviation for non-spec captures

The NXDN receiver's 4-FSK slicer is calibrated against the Common
Air Interface spec value of 1800 Hz peak deviation. On-air
transmitters that deviate from spec produce a **bimodal dibit
distribution** through the slicer — outer ±3 levels dominate, inner
±1 levels under-fire (e.g. `samples/nxdn/NXDN96 IQ.wav` reports
3 / 50 / 3 / 44 % through the production pipeline). Override the
deviation per-system in `config.yaml`:

```yaml
trunking:
  systems:
    - name: "Local NXDN"
      protocol: "nxdn"
      control_channels: [851062500]
      # Override only if the default 1800 Hz produces a bimodal
      # dibit distribution. Typical values for non-spec
      # transmitters land between 2200 and 2700 Hz.
      nxdn_deviation_hz: 2400
```

Zero / unset (the default) uses the 1800 Hz spec value. The
existing `cmd/audio_smoketest` tool prints the empirical dibit
distribution after slicing — sweep `nxdn_deviation_hz` until the
distribution flattens toward the spec-ideal 25 / 25 / 25 / 25 %
before locking in a value.
