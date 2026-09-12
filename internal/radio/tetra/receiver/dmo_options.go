package receiver

// DMOOptions returns the receiver configuration every TETRA DMO (Direct Mode)
// consumer shares. The control pipeline (ccdecoder's tetra-dmo pipeline) and the
// voice composer's DMO chain run the SAME π/4-DQPSK receiver on the SAME carrier
// tap, and they have to slice the same bursts: the pipeline recovers the DM
// traffic colour code from its stream and hands it to the voice chain, which
// verifies it against ITS stream before adopting it. On the 20 Aug #1003 on-air
// run the two receivers were configured differently — the voice chain
// additionally ran the DC-removal high-pass — and they disagreed: the pipeline
// decoded ~200 CRC-valid TCH/S at colour 39 in one PTT while the voice chain could
// not verify that colour on its own bursts and decoded nothing. Sharing the knobs
// makes that drift structurally impossible.
//
// Callers attach their own sinks (DibitSink / SoftSink / SymbolSink) and may
// override ClockMode from the system config; nothing else should differ.
//
//   - EnableEqualizer: the blind SnapshotCMA is REQUIRED for DMO, not optional —
//     on the reporter's 438.9 MHz capture it lifts CRC-valid SCH/S from ~6 to ~64
//     by inverting the ISI that smears the constellation (the same lever the TMO
//     CC path and the offline TestTETRADMOReplay run by default).
//   - EnableDCBlock stays OFF, as on the control pipeline that decodes on air.
//     The DC blocker is the TMO voice-only stage (receiver.go); DMO's pipeline has
//     never run it, and it is the prime suspect for the 20 Aug divergence.
func DMOOptions(sampleRateHz float64) Options {
	return Options{
		SampleRateHz:        sampleRateHz,
		ClockMode:           ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     true,
	}
}
