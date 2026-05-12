package dstar

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// processState is the cross-call bit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call.
type processState struct {
	det *SyncDetector
	// remaining > 0: collecting PCH header bits after a Frame Sync
	// match; counts down to 0 as Process feeds bits forward.
	remaining int
	// pch accumulates either the 328-bit raw header (under FECOff)
	// or the 660-bit on-wire FEC-encoded stream (under FECOn).
	pch []byte
	// matchScratch is reused across calls so SyncDetector.Process
	// doesn't allocate fresh slices.
	matchScratch []int
	// captureMode is the FECMode the adapter sampled at the most
	// recent Frame Sync match. It's latched per-window so a
	// mid-window mode change doesn't break the byte count.
	captureMode FECMode
}

// HeaderBits is the information size of the D-STAR PCH header (41
// bytes × 8 = 328 bits) — what the Process adapter collects under
// FECOff.
const HeaderBits = 41 * 8

// Process consumes a window of raw bits from the D-STAR receiver
// (the IQ → GMSK bit chain in internal/radio/dstar/receiver/), runs
// the 32-bit Frame Sync detector, slices the following 328-bit PCH
// header window out of the stream, packs it into 41 bytes, runs
// CRC-CCITT verify, and forwards the parsed Header to Ingest.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime. The adapter's internal countdown survives across
// Process calls so a sync match in one chunk and the payload in
// the next still decode cleanly.
//
// The convolutional rate-1/2 inner code / scrambler / interleaver
// the on-air PCH carries are documented follow-ups; this adapter
// works on synthesized fixtures and pre-FEC-stripped inputs.
//
// Returns baseIdx + len(bits) to match the Process contract every
// other ControlChannel package exposes.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		// tolerance=2 tracks the JARL Header Frame Sync (0xEAA060)
		// through up to 2 bit errors per 24-bit window. The pattern
		// is non-periodic, so near-matches a few bits off the real
		// position don't fire — the false-alignment problem the
		// 0x55555555 toggle placeholder used to cause is gone.
		c.proc = &processState{
			det: NewSyncDetector(FrameSyncBitsSlice(), 2),
			pch: make([]byte, 0, FECOnHeaderBits),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], bits, baseIdx)
	matchIdx := 0

	for i, b := range bits {
		absPos := baseIdx + i
		// Collect first (this bit completes the PCH window if
		// remaining counts down to 0). Order matters: the sync
		// match's absolute index is the LAST bit of the 24-bit
		// Frame Sync, so the PCH window starts at the NEXT
		// iteration.
		if p.remaining > 0 {
			p.pch = append(p.pch, b&1)
			p.remaining--
			if p.remaining == 0 {
				if hdr, ok := c.parseHeader(p.pch, p.captureMode); ok {
					c.Ingest(hdr)
				}
				p.pch = p.pch[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			// Skip matches that fire while we're still
			// collecting the previous window — guards against
			// duplicate sync detections inside a single burst.
			if p.remaining == 0 {
				c.mu.Lock()
				mode := c.fecMode
				c.mu.Unlock()
				p.captureMode = mode
				if mode == FECOn {
					p.remaining = FECOnHeaderBits
				} else {
					p.remaining = HeaderBits
				}
				p.pch = p.pch[:0]
			}
			matchIdx++
		}
	}
	return baseIdx + len(bits)
}

// parseHeader interprets the supplied bit window per the supplied
// FECMode, runs CRC-CCITT verify on the recovered information field,
// and returns the structured Header. Returns (zero, false) when the
// length is wrong, the FEC chain can't recover the info, or the
// CRC mismatches.
//
//   FECOff: bits is 328 information bits MSB-first.
//   FECOn:  bits is 660 on-wire bits (FEC-encoded); the function
//           runs framing.DecodeDStarHeaderFEC to recover the 41-byte
//           information field.
func (c *ControlChannel) parseHeader(bits []byte, mode FECMode) (Header, bool) {
	var bytesOut []byte
	switch mode {
	case FECOn:
		if len(bits) != FECOnHeaderBits {
			return Header{}, false
		}
		recovered, ok := framing.DecodeDStarHeaderFEC(bits)
		if !ok {
			return Header{}, false
		}
		bytesOut = recovered
	default:
		if len(bits) != HeaderBits {
			return Header{}, false
		}
		bytesOut = make([]byte, 41)
		for i := 0; i < 41; i++ {
			var v byte
			for j := 0; j < 8; j++ {
				v = (v << 1) | (bits[i*8+j] & 1)
			}
			bytesOut[i] = v
		}
	}
	hdr, err := ParseHeader(bytesOut)
	if err != nil {
		return Header{}, false
	}
	if hdr.CRC != ComputeCRC(bytesOut[:39]) {
		return Header{}, false
	}
	return hdr, true
}

// Reset clears the SyncDetector's history so a stale match doesn't
// fire after a stream re-sync.
func (s *SyncDetector) Reset() {
	for i := range s.hist {
		s.hist[i] = 0
	}
	s.primed = 0
	s.pos = 0
}
