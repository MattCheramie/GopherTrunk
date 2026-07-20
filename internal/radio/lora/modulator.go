package lora

// LoRa transmit path: synthesise clean baseband IQ for a frame at sample
// rate osf*BW. Used to generate test fixtures and to exercise the receive
// chain end-to-end without an SDR. The frame structure produced here
// (preamble, 2 sync symbols, 2.25-symbol SFD, header block, payload
// blocks) is exactly what demod.syncFrame and decode.go expect.

// PreambleSymbols is the number of base upchirps emitted before the sync
// word. Real deployments use 8+; the receiver locks on preambleSyms of
// them.
const PreambleSymbols = 8

// syncSymbolValues maps a one-byte sync word to its two symbol values,
// following the common nibble*8 convention. Values stay below 2^SF for all
// SF>=7.
func syncSymbolValues(syncWord uint8) (int, int) {
	return int(syncWord>>4) << 3, int(syncWord&0x0f) << 3
}

// LoRaModulate builds baseband IQ (complex64, rate osf*BW) carrying payload
// as a complete LoRa frame. Low Data Rate Optimization is auto-detected
// from the (SF, BW) pair (the Semtech Ts >= 16 ms rule), matching what a
// NewDemodulator for the same pair expects. Use LoRaModulateMode to force
// it. bw is recorded for downstream metadata only — the sample values
// depend on sf and osf, not the absolute bandwidth (aside from the LDRO
// decision).
func LoRaModulate(payload []byte, sf, cr, osf int, bw Bandwidth, hasCRC bool, syncWord uint8) []complex64 {
	return LoRaModulateMode(payload, sf, cr, osf, bw, hasCRC, syncWord, LDROAuto)
}

// LoRaModulateMode is LoRaModulate with an explicit LDRO mode, for building
// fixtures that force reduced-rate on or off regardless of (SF, BW).
func LoRaModulateMode(payload []byte, sf, cr, osf int, bw Bandwidth, hasCRC bool, syncWord uint8, mode LDROMode) []complex64 {
	ldro := ResolveLDRO(mode, sf, bw)
	up := GenChirp(sf, osf, true)
	down := GenChirp(sf, osf, false)
	sps := len(up)

	out := make([]complex64, 0, (PreambleSymbols+5)*sps+len(payload)*sps)

	// Preamble: PreambleSymbols base upchirps.
	for i := 0; i < PreambleSymbols; i++ {
		out = append(out, up...)
	}
	// Sync word: two modulated upchirps.
	s0, s1 := syncSymbolValues(syncWord)
	out = append(out, GenSymbolChirp(sf, osf, s0)...)
	out = append(out, GenSymbolChirp(sf, osf, s1)...)
	// SFD: 2.25 downchirps.
	out = append(out, down...)
	out = append(out, down...)
	out = append(out, down[:sps/4]...)

	// Header + payload symbols.
	for _, v := range encodeFrameSymbols(payload, sf, cr, hasCRC, ldro) {
		out = append(out, GenSymbolChirp(sf, osf, int(v))...)
	}
	return out
}
