# MPT 1327 captures

The 16-bit Codeword Synchronisation Code (CWSC) detector in
[`internal/radio/mpt1327/process.go`](../../internal/radio/mpt1327/process.go)
**now matches against a configurable Hamming-distance threshold**
(default 2 bits out of 16, matching commercial MPT 1327 receivers).
Operators replaying pre-stripped synthesized fixtures opt back into
exact-match per system via `mpt1327_cwsc_tolerance: 0`.

Real-air **MPT 1327 control channel** recordings are still welcome
for empirical false-positive-rate validation — the combinatorial
math (C(16, 0..2) / 2^16 ≈ 0.21% per window, combined with the
BCH(64, 48, 2)-validated codeword that must follow) bounds the
false-lock rate, but a real capture closes the loop against actual
noise distributions.

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

## Why captures are still welcome

The tolerance default (2 out of 16) is grounded in the
combinatorial math + the BCH validation that gates downstream
codeword acceptance. What a real capture closes:

- **Empirical false-positive count.** Pre-PR-A theory said ~6e-8
  per bit position; a real capture lets us measure that directly
  and confirm the BCH gate behaves as expected on actual noise.
- **Per-vendor CWSC bit-error distribution.** Tait vs. Motorola
  vs. Simoco transmitters may have characteristic patterns in
  which sync bits flip; that signals which tolerance value
  operators in those ecosystems should pin.

## Acceptance criteria

A capture is considered "validating" when:

1. **No spurious locks in clean traffic.** Run the capture with
   `mpt1327_cwsc_tolerance: 2` (default) — every
   `events.KindCCLocked` must align with a sync sequence the
   metadata's `messages` list contains. False-positive locks
   (locks that don't correspond to a real message) bound the
   false-positive rate per Hamming distance.
2. **True-positive rate ≥ 95%.** ≥ **95% of CWSC sync
   sequences** in `metadata.expected.messages` must produce a
   downstream `events.KindGrant` or `events.KindCCLocked` at the
   default tolerance. Missing locks at tolerance ≥ 2 suggest a
   different failure mode than the CWSC threshold (signal level,
   adjacent-channel interference, etc.).
3. **Tolerance sweep stays monotone.** Running the capture at
   tolerance ∈ {0, 1, 2, 3} must produce monotonically
   non-decreasing lock counts (more permissive thresholds should
   never lose locks). A non-monotone sweep flags a bug in the
   matcher.

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
