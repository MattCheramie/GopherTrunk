package motorola

// processState is the cross-call framing state the Process adapter
// holds: an 8-bit sync correlator, the sync-bracketed frame window,
// and the in-sync countdown. Lazily initialised on the first Process
// call. Port of OP25 rx_smartnet::rx_sym's framer.
type processState struct {
	syncReg uint8
	inSync  bool
	// count is how many bits have arrived since the last accepted
	// sync ended. A frame is complete when count reaches FrameBits
	// (76 payload bits + the NEXT frame's 8 sync bits, which must
	// match for the frame to be trusted).
	count int
	// ring holds the last FrameBits bits, ringPos the write index.
	ring    [FrameBits]byte
	ringPos int
	// payload is the scratch buffer the completed 76-bit window is
	// unrolled into.
	payload [PayloadBits]byte
}

// Process consumes a window of raw bits from the Motorola receiver
// (the IQ → FSK bit chain in internal/radio/motorola/receiver/),
// frames them on the 8-bit outbound sync, and decodes each
// sync-bracketed 76-bit payload (deinterleave → convolutional-parity
// ECC → CRC-10, frame.go) into an OSW for the sequencer.
//
// Framing follows OP25's rx_smartnet: the first sync starts
// collection, and a frame is accepted only when ANOTHER sync arrives
// exactly PayloadBits later — control-channel frames run
// back-to-back, so every valid frame is bracketed by two syncs. That
// plus the CRC-10 makes the short 8-bit sync word safe against false
// matches. A missing trailing sync drops framing entirely and waits
// for a fresh sync (any early sync inside a frame is treated as
// payload data, exactly like the reference).
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime; the framing state survives across Process calls so a
// frame spanning a chunk boundary still decodes. Returns
// baseIdx + len(bits) to match the YSF / P25 Phase 1 / dPMR / NXDN /
// EDACS ControlChannel.Process contracts.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{}
	}
	p := c.proc

	for _, b := range bits {
		b &= 1
		p.syncReg = p.syncReg<<1 | b
		syncDetected := p.syncReg == uint8(OutboundSyncHex)
		p.ring[p.ringPos] = b
		p.ringPos = (p.ringPos + 1) % FrameBits
		p.count++

		if syncDetected && !p.inSync {
			p.inSync = true
			p.count = 0
			continue
		}
		if !p.inSync || p.count < FrameBits {
			continue
		}
		if !syncDetected {
			// The bracket sync didn't arrive where the frame should
			// end — framing is wrong; hunt for a fresh sync.
			p.inSync = false
			p.count = 0
			continue
		}
		p.count = 0
		// The ring now holds [payload(76) | next sync(8)] with the
		// oldest bit at ringPos; unroll the payload.
		for i := 0; i < PayloadBits; i++ {
			p.payload[i] = p.ring[(p.ringPos+i)%FrameBits]
		}
		if osw, ok := DecodeOSWPayload(p.payload[:]); ok {
			c.Ingest(osw)
		}
	}
	return baseIdx + len(bits)
}

// ResetFraming clears the Process adapter's sync/framing state. The
// receiver calls this via its own Reset on stream re-sync so a stale
// half-frame doesn't bridge two IQ epochs.
func (c *ControlChannel) ResetFraming() {
	c.proc = nil
}
