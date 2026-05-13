# DMR Tier II (conventional) captures

Drop **DMR Tier II repeater** IQ recordings here to lift the
`t.Skip` on `TestDaemonCCDecodesDMRTier2` in
[`cmd/gophertrunk/integration_cc_dmr_tier2_test.go`](../../cmd/gophertrunk/integration_cc_dmr_tier2_test.go).

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

## Why captures are needed

The Tier II pipeline + Process adapter ship in
`internal/radio/dmr/tier2/`. The integration test is currently
`t.Skip`'d because the **synthesized** Voice LC Header IQ fixture
doesn't round-trip cleanly through the C4FM modulator+demod chain —
multiple prior debug attempts couldn't get it to lock.

A real-air capture sidesteps the fixture problem entirely: real
repeater RF has the burst-symbol distribution and SNR characteristics
the Mueller-Müller clock recovery + sync detector need. Drop a
capture here and the test can be re-enabled by:

1. Replacing `t.Skip(...)` with a `loadIQFromFile(t, "...")` call.
2. Pointing the SDR mock driver at the capture path.
3. Asserting the metadata's `voice_lc_header` matches the
   `events.KindCCLocked` payload.

## Recommended sources

- **A controlled Tier II repeater** transmitting known TG +
  source ID combinations.
- **MMDVMHost** in Tier II mode (rare; check
  `mmdvm.ini` for `[DMR]` → `Mode=4`).
- **DSD-FME** can validate the cleartext call-setup metadata.
