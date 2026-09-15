# DMR direct-mode (simplex) real-air fixtures — issue #836

Two channelized slices of the captures @alvin275 (VE5XAM) posted for issue
#836: a DMR handheld keying up on **446.500 MHz simplex** (direct mode,
one burst per 60 ms frame), talkgroup 99, radio 3024109, colour code 1,
recorded with `gophertrunk capture -freq 446500000 -sample-rate 2400000
-format cs16 -gain 200` on 14 Sep 2026 (`cs16-20sec-gain200`). The dongle
sits ~1.6 kHz off the carrier and the handheld in the same room pins ~36 %
of the raw samples at the ADC rail; the decode survives both.

| File | Content |
| --- | --- |
| `dmr-directmode-446500-keyup-48k.cs16` | 1.8 s from t = 0.6 s: 0.47 s of noise, then the keyup — ten Voice LC Header copies 60 ms apart (0.6 s) followed by the first voice superframes |
| `dmr-directmode-446500-ptt2-48k.cs16` | 1.7 s from t = 10.2 s: 0.30 s of noise, then the second PTT's onset (its header train and first voice) |

Both were produced by the production `ccdecoder.Downconverter` (2.4 MS/s →
48 kHz, no tune offset — the capture is centred on the channel), then
peak-normalised to −3 dBFS and written as headerless interleaved int16 I/Q
(`cs16`, little-endian). `dmr_directmode_realair_test.go` runs them through
the production DMR receiver and Tier II state machine. The full captures
(three 20 s files, 163 MB) are linked from the issue.
