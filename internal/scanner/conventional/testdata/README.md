# Conventional-scanner test fixtures

`ctcss100_radtel_447100k_48k.cs16` and `ctcss_none_radtel_447100k_48k.cs16`
are 1.2 s slices (3.0 s–4.2 s) of on-air captures posted by @v2maldo on
issue #1184: a Radtel RT-880G keyed on 447.100 MHz NFM with and without a
100.0 Hz CTCSS tone, recorded by GopherTrunk's per-call voice IQ debug tap
(RTL-SDR, 2.4 MS/s cs16, tuned on-channel). Each slice was channelized to
48 kHz (`dsp.NewResampler(1, 50, 401, 8.6)`) to keep the fixtures small;
`ctcss_realair_test.go` interpolates them back to 2.4 MS/s so the tone
detector runs at the rate the scanner feeds it.

Format: interleaved little-endian int16 I/Q, 48 000 samples/s.
