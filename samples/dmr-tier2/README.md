# DMR Tier II (conventional) captures

`TestDaemonCCDecodesDMRTier2` is **no longer skipped** — the
synthesized-fixture path was fixed by lowering the Tier II
pipeline's Mueller-Müller `ClockGain` from 0.025 to 0.015 (see
[`internal/scanner/ccdecoder/pipelines.go`](../../internal/scanner/ccdecoder/pipelines.go)'s
`newDMRTier2Pipeline` + the diagnostic test at
[`cmd/gophertrunk/dmr_tier2_diagnostic_test.go`](../../cmd/gophertrunk/dmr_tier2_diagnostic_test.go)).

Real-air **DMR Tier II repeater** captures are still useful for
secondary validation — the synthesized fixture confirms the
pipeline plumbs IQ → C4FM → state-machine end-to-end, but
real RF exercises the burst-error structure that synthesized IQ
doesn't model.

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex int16 IQ (`*.bin`, unsigned `*.cfile`) |
| Sample rate | 48 kHz nominal (matches the existing Tier II test harness) |
| Modulation | C4FM at 4800 symbols/s, 1944 Hz peak deviation per ETSI TS 102 361-1 §6.3 |
| Channel width | 12.5 kHz |
| Centre | Tuned on the repeater output frequency |
| Duration | ≥ 5 seconds — enough to capture a Voice LC Header + Terminator with LC burst pair |

## Metadata schema

```json
{
  "source": "Live Tier II repeater @ <location>",
  "tool_cross_check": "DSD-FME / MMDVMHost log",
  "expected": {
    "color_code": 7,
    "voice_lc_header": {
      "flco": "GroupVoiceUser",
      "group_address": "0x123",
      "source_id": "0x456789"
    },
    "terminator_with_lc": true
  },
  "notes": "Tier II is per-repeater conventional — every burst is on the same carrier; the Voice LC Header is the call-setup burst the state machine syncs on."
}
```

## Why captures are still welcome

The Tier II pipeline + Process adapter ship in
`internal/radio/dmr/tier2/`, and the synthesized integration test
passes end-to-end since the ClockGain fix. What a real-air capture
adds:

- **Burst-error structure validation.** Synthesized IQ has clean
  white-Gaussian noise margins; real RF has impulse interference,
  multipath, and SNR variation that the synthesized path can't
  exercise.
- **Across-payload distribution coverage.** The synthesized fixture
  uses one specific FLC (group voice user with `0x123` /
  `0x456789` IDs). Real captures sample the per-call bit-pattern
  diversity that surfaces any pathology the synthesized fixture
  happens to miss.

## Acceptance criteria

A capture is considered "validating" when:

1. **Voice LC Header lock.** The capture's first Voice LC Header
   burst produces `events.KindCCLocked` within **2 seconds** of
   the start of the recording. Tier II locks faster than Tier III
   because there's no CC-hunt phase.
2. **Group + source ID match.** The decoded FLC payload's
   `DstAddr` (group) + `SrcAddr` (source) match the
   metadata's `voice_lc_header.group_address` /
   `voice_lc_header.source_id` byte-for-byte.
3. **Terminator-with-LC ends the call cleanly.** When the
   capture includes a call end, the daemon publishes
   `events.KindCallEnd` with `EndReasonNormal` (not
   `EndReasonTimeout`); the in-tree recorder writes a
   well-formed WAV without truncation.

A capture missing the Terminator with LC is still useful for
items 1 + 2 — partial-call recordings should set
`expected.terminator_with_lc: false` in the metadata so the test
harness doesn't expect it.

## Recommended sources

- **A controlled Tier II repeater** transmitting known TG +
  source ID combinations.
- **MMDVMHost** in Tier II mode (rare; check
  `mmdvm.ini` for `[DMR]` → `Mode=4`).
- **DSD-FME** can validate the cleartext call-setup metadata.
