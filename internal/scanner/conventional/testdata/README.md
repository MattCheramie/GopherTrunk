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

`am_kric_126400k_carrier_48k.cs16` and `am_kric_126400k_noise_48k.cs16` are
1.2 s slices (0.1 s–1.3 s and 1.7 s–2.9 s) of an on-air capture posted by
@v2maldo on issue #1219: 126.400 MHz AM (KRIC Potomac Departure), recorded
by GopherTrunk's per-call voice IQ debug tap (2.4 MS/s cs16, channel at DC).
The transmitter is keyed for the first ~1.5 s of the capture and receiver
noise follows, so the two slices are carrier + voice and noise only, from
the same receiver at the same gain. Channelized to 48 kHz the same way as
the CTCSS slices. Used by `am_realair_test.go` (squelch) and by
`internal/voice/composer/am_realair_test.go` (carrier tracking).

`ysf_issue1261_analog_tone_448900k_48k.cs16` is a 1 s slice (1.0 s–2.0 s)
of the capture @v2maldo posted on issue #1261 (11 Nov) as a System Fusion
test on 448.900 MHz: GopherTrunk's per-call voice IQ debug tap on a `ysf`
scan-list channel (RTL-SDR Blog V4, 2.4 MS/s cs16, tuned on-channel),
channelized to 48 kHz with `ccdecoder.NewDownconverter`. It is NOT a C4FM
transmission: the discriminator output is a single 1 kHz sine (an analog
FM test tone at about ±1.8–2.6 kHz deviation, 1.4 kHz off frequency), and
57 % of the original's samples have I or Q at the ADC rail (the
constellation is the clipping square: the handheld was next to the dongle). `ysf_gate_test.go` pins that the YSF call gate stays shut on
it: on power alone the scanner had opened a "ysf" call that recorded
nothing.
