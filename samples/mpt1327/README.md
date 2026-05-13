# MPT 1327 captures

Drop **MPT 1327 control channel** bit-stream / IQ recordings here to
calibrate the bit-error tolerance threshold on the 16-bit Codeword
Synchronisation Code (CWSC) detector
[`internal/radio/mpt1327/process.go`](../../internal/radio/mpt1327/process.go)
ships today as exact-match.

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex float32 IQ (`*.cfile`), complex int16 (`*.bin`), or 8 kHz PCM audio (`*.wav`, demodulated FFSK output) |
| Sample rate | 48 kHz IQ or 8 kHz audio |
| Modulation | FFSK at 1200 baud (mark = 1200 Hz / space = 1800 Hz CCIR) on NBFM |
| Channel width | 12.5 kHz NBFM |
| Centre | Tuned on the control channel carrier |
| Duration | ≥ 60 seconds — captures dozens of messages to characterise the noisy-CWSC distribution |

## Metadata schema

```json
{
  "source": "Live MPT 1327 control channel @ <location>",
  "tool_cross_check": "TrunkTracker / SDR-Trunk log",
  "expected": {
    "prefix": "0x05",
    "system_id": "0x1234",
    "message_count": 42,
    "messages": [
      { "type": "Aloha", "prefix": "0x05" },
      { "type": "GoToChannel", "prefix": "0x05", "ident": "0x123", "channel": 7 }
    ]
  },
  "snr_estimate_db": 14.0,
  "notes": "The CWSC tolerance experiment counts how many messages decode correctly at each Hamming-distance threshold (0, 1, 2 bit errors allowed in the 16-bit sync). 1-bit is the safe default; 2-bit may produce false locks on noisy captures."
}
```

## Why captures are needed

The exact-match CWSC detector in
[`process.go::findCWSC`](../../internal/radio/mpt1327/process.go)
locks reliably on synthesized fixtures but is fragile when real-air
captures introduce occasional bit errors in the sync sequence. The
spec doesn't mandate a tolerance threshold — implementations
typically choose 1 or 2 bit errors based on empirical SNR vs.
false-lock measurements.

The calibration experiment:

1. Walk a labeled capture with `findCWSC` at tolerance ∈ {0, 1, 2}.
2. For each tolerance, measure (true-positive locks / false-positive
   locks / lock latency) against `metadata.expected.messages`.
3. Pick the tolerance that maximises true-positive rate while
   keeping false positives within the noise floor.

## Recommended sources

- **TrunkTracker / SDR-Trunk** decoding a known MPT 1327 site —
  produces a labeled log GopherTrunk can cross-check against.
- **A controlled MPT 1327 transmitter** (most are Tait or Motorola
  legacy infrastructure) keyed with known Aloha + GoToChan
  sequences.

## Spec reference

[MPT 1327 standard](https://www.sigidwiki.com/images/8/85/Mpt1327.pdf)
defines the CWSC as the bit pattern `1100010011010111` immediately
preceding the first codeword of every control-channel message.
