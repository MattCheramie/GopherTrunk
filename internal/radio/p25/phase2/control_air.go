package phase2

// airState is ProcessAir's cross-call state. Lazily initialised.
type airState struct {
	framer *AirFramer
	seed   uint64
	masks  *AirMasks
}

// ProcessAir is the live control-channel adapter: it frames the receiver's
// dibits in the real TDMA air format (AirFramer), decodes every signalling
// burst (LCCH, SACCH, FACCH) with DecodeAirMAC and hands each MAC message
// to Ingest. The scrambled DUIDs are descrambled with the current PN44 seed
// (configured, or installed from a Network Status Broadcast — which the
// clear LCCH carries, so a site with no configured identity bootstraps
// itself). Unlike the voice chain, the control channel takes signalling
// from every timeslot.
//
// baseIdx is the absolute dibit index of dibits[0]; a jump (a receiver
// reset restarting its count) unlocks the framer, which re-hunts. The
// trellis / RS / interleave / scrambler modes are not consulted: the air
// format fixes the channel coding. Process (the flat sync-window adapter
// on the older working model) remains for its own tests.
func (c *ControlChannel) ProcessAir(dibits []uint8, baseIdx int) int {
	if c.air == nil {
		c.air = &airState{framer: NewAirFramer()}
	}
	a := c.air
	c.syncAirMasks()
	for _, b := range a.framer.Process(dibits, baseIdx) {
		if !b.DUID.IsSignalling() {
			continue
		}
		m, ok := DecodeAirMAC(b, a.masks)
		if !ok {
			continue
		}
		for _, pdu := range m.Messages() {
			c.Ingest(pdu)
		}
		// An NSB may just have installed the seed: the scrambled
		// bursts that follow in this same chunk need it.
		c.syncAirMasks()
	}
	return baseIdx + len(dibits)
}

// syncAirMasks rebuilds the descramble masks when the seed changed.
func (c *ControlChannel) syncAirMasks() {
	c.mu.Lock()
	seed := c.scramblerSeed
	c.mu.Unlock()
	if a := c.air; seed != a.seed {
		a.seed, a.masks = seed, nil
		if seed != 0 {
			a.masks = NewAirMasks(seed)
		}
	}
}
