package dstar

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// processState is the cross-call bit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call.
type processState struct {
	// det / detInv find the frame sync in each polarity.
	det, detInv *SyncDetector
	// remaining > 0: collecting header bits after a frame sync match;
	// counts down to 0 as Process feeds bits forward.
	remaining int
	// invert is the polarity of the window being collected: its bits
	// are complemented as they are stored.
	invert byte
	// pch accumulates either the 328-bit raw header (under FECOff)
	// or the 660-bit on-air FEC section (under FECOn).
	pch []byte
	// matchScratch / invScratch are reused across calls so
	// SyncDetector.Process doesn't allocate fresh slices.
	matchScratch, invScratch []int
	// captureMode is the FECMode the adapter sampled at the most
	// recent frame sync match. It's latched per window so a mid-window
	// mode change doesn't break the bit count.
	captureMode FECMode
}

// HeaderBits is the information size of the D-STAR header (41 bytes ×
// 8 = 328 bits) — what the Process adapter collects under FECOff.
const HeaderBits = 41 * 8

// frameSyncTolerance is MMDVM's FRAME_SYNC_ERRS: bit errors accepted in
// the 24-bit frame sync pattern.
const frameSyncTolerance = 2

// Process consumes a window of raw bits from the D-STAR receiver
// (the IQ → GMSK bit chain in internal/radio/dstar/receiver/), finds
// the frame sync in either polarity, collects the header that follows,
// decodes it (framing.DecodeDStarHeaderFEC under FECOn, the default),
// checks its CRC and forwards the parsed Header to Ingest.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime. The adapter's internal countdown survives across Process
// calls so a sync match in one chunk and the header in the next still
// decode cleanly.
//
// Returns baseIdx + len(bits) to match the Process contract every
// other ControlChannel package exposes.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det:    NewSyncDetector(FrameSyncBitsSlice(), frameSyncTolerance),
			detInv: NewSyncDetector(invertBits(FrameSyncBitsSlice()), frameSyncTolerance),
			pch:    make([]byte, 0, FECOnHeaderBits),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], bits, baseIdx)
	p.invScratch, _ = p.detInv.Process(p.invScratch[:0], bits, baseIdx)
	mi, ii := 0, 0

	for i, b := range bits {
		absPos := baseIdx + i
		// Collect first (this bit completes the header window if
		// remaining counts down to 0). Order matters: the sync match's
		// absolute index is the LAST bit of the sync pattern, so the
		// header window starts at the NEXT iteration.
		if p.remaining > 0 {
			p.pch = append(p.pch, (b^p.invert)&1)
			p.remaining--
			if p.remaining == 0 {
				if hdr, ok := c.parseHeader(p.pch, p.captureMode); ok {
					if c.ingestHook != nil {
						c.ingestHook(hdr)
					}
					c.Ingest(hdr)
				}
				p.pch = p.pch[:0]
			}
		}
		normal, inverted := false, false
		for mi < len(p.matchScratch) && p.matchScratch[mi] == absPos {
			normal = true
			mi++
		}
		for ii < len(p.invScratch) && p.invScratch[ii] == absPos {
			inverted = true
			ii++
		}
		// Ignore matches that fire while a window is still being
		// collected: a header's own bits can resemble the sync.
		if (normal || inverted) && p.remaining == 0 {
			c.mu.Lock()
			mode := c.fecMode
			c.mu.Unlock()
			p.captureMode = mode
			p.invert = 0
			if inverted && !normal {
				p.invert = 1
			}
			if mode == FECOn {
				p.remaining = FECOnHeaderBits
			} else {
				p.remaining = HeaderBits
			}
			p.pch = p.pch[:0]
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
//	FECOff: bits is the 41 header bytes unencoded, each byte LSB
//	        first as D-STAR sends bytes (no real radio sends this).
//	FECOn:  bits is the 660-bit on-air FEC section; the function
//	        runs framing.DecodeDStarHeaderFEC to recover the 41 bytes.
func (c *ControlChannel) parseHeader(bits []byte, mode FECMode) (Header, bool) {
	var bytesOut []byte
	switch mode {
	case FECOn:
		return DecodeHeaderBurst(bits)
	default:
		if len(bits) != HeaderBits {
			return Header{}, false
		}
		bytesOut = make([]byte, 41)
		for i := 0; i < HeaderBits; i++ {
			bytesOut[i/8] |= (bits[i] & 1) << uint(i%8)
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

// DecodeHeaderBurst decodes the 660-bit header FEC section that follows a
// frame sync (bits in time order, already in MMDVM's polarity) and checks
// its CRC. ok is true only for a header whose CRC matches.
func DecodeHeaderBurst(fec []uint8) (Header, bool) {
	if len(fec) != FECOnHeaderBits {
		return Header{}, false
	}
	b, _ := framing.DecodeDStarHeaderFEC(fec)
	h, err := ParseHeader(b)
	if err != nil || h.CRC != ComputeCRC(b[:39]) {
		return Header{}, false
	}
	return h, true
}
