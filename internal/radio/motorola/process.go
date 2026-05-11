package motorola

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// processState is the cross-call bit buffering + sync-detection
// state the Process adapter holds. Lazily initialised on the first
// Process call.
type processState struct {
	det          *SyncDetector
	remaining    int
	osw          []byte
	matchScratch []int
}

// BCHMode controls whether the Process adapter runs the
// BCH(64,16,11) FEC over each on-air codeword pair before
// reassembling the 32-bit OSW. Configurable via SetBCHMode.
type BCHMode uint8

const (
	// BCHOff treats the 32 bits after sync as raw OSW info bits.
	// Works for test fixtures + clean signals; on-air traffic
	// usually fails OSW parsing.
	BCHOff BCHMode = iota
	// BCHOn reads two 64-bit BCH(64,16,11) codewords (128 wire
	// bits) after sync, decodes each via the framing primitive,
	// and concatenates the recovered 16-bit halves into the
	// 32-bit OSW. Uncorrectable codewords (> 11 errors) drop the
	// frame.
	BCHOn
)

// SetBCHMode configures whether the Process adapter runs
// BCH(64,16,11) FEC over each codeword pair. Call before the
// first Process call; switching mode mid-stream resets the
// adapter's buffer.
func (c *ControlChannel) SetBCHMode(m BCHMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bchMode = m
	if c.proc != nil {
		c.proc.remaining = 0
		c.proc.osw = c.proc.osw[:0]
	}
}

// oswInfoBits is the count of bits the adapter collects after each
// 24-bit sync match in BCHOff mode: 32 information bits = one OSW.
const oswInfoBits = 32

// oswBCHBits is the count of bits the adapter collects after each
// sync match in BCHOn mode: two 64-bit BCH(64,16,11) codewords =
// 128 channel bits, decoding to 32 information bits.
const oswBCHBits = 128

// Process consumes a window of raw bits from the Motorola receiver
// (the IQ → MSK bit chain in internal/radio/motorola/receiver/),
// runs the 24-bit outbound sync detector, slices the following
// codeword window (32 raw bits in BCHOff mode, 128 channel bits
// across two BCH(64,16,11) codewords in BCHOn mode) out of the
// stream, parses it via OSWFromBits, and forwards the result to
// Ingest.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime. The adapter's internal countdown survives across
// Process calls so a sync match in one chunk and the payload in
// the next still decode cleanly.
//
// Returns baseIdx + len(bits) to match the YSF / P25 Phase 1 /
// dPMR / NXDN / EDACS ControlChannel.Process contracts.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{
			det: NewSyncDetector(OutboundSyncBits(), 1),
			osw: make([]byte, 0, oswBCHBits),
		}
	}
	p := c.proc

	p.matchScratch, _ = p.det.Process(p.matchScratch[:0], bits, baseIdx)
	matchIdx := 0

	postSyncBits := c.codewordBitsForMode()

	for i, b := range bits {
		absPos := baseIdx + i
		if p.remaining > 0 {
			p.osw = append(p.osw, b&1)
			p.remaining--
			if p.remaining == 0 {
				c.tryIngestOSW(p.osw)
				p.osw = p.osw[:0]
			}
		}
		for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == absPos {
			p.remaining = postSyncBits
			p.osw = p.osw[:0]
			matchIdx++
		}
	}
	return baseIdx + len(bits)
}

// codewordBitsForMode returns how many bits Process collects
// after each sync, based on the current BCH mode.
func (c *ControlChannel) codewordBitsForMode() int {
	c.mu.Lock()
	m := c.bchMode
	c.mu.Unlock()
	if m == BCHOn {
		return oswBCHBits
	}
	return oswInfoBits
}

// tryIngestOSW reconstructs an OSW from the post-sync bits and
// hands it to Ingest. In BCHOff mode the bits are used directly;
// in BCHOn mode each 64-bit codeword runs through BCHDecode64_16
// and the two recovered 16-bit halves are concatenated.
func (c *ControlChannel) tryIngestOSW(post []byte) {
	c.mu.Lock()
	m := c.bchMode
	c.mu.Unlock()
	switch m {
	case BCHOn:
		if len(post) != oswBCHBits {
			return
		}
		half1, errs1 := framing.BCHDecode64_16(bitsToUint64(post[0:64]))
		if errs1 < 0 {
			return
		}
		half2, errs2 := framing.BCHDecode64_16(bitsToUint64(post[64:128]))
		if errs2 < 0 {
			return
		}
		var bits [32]byte
		for i := 0; i < 16; i++ {
			bits[i] = byte((half1 >> uint(15-i)) & 1)
		}
		for i := 0; i < 16; i++ {
			bits[16+i] = byte((half2 >> uint(15-i)) & 1)
		}
		if osw, err := OSWFromBits(bits[:]); err == nil {
			c.Ingest(osw)
		}
	default:
		if len(post) != oswInfoBits {
			return
		}
		if osw, err := OSWFromBits(post); err == nil {
			c.Ingest(osw)
		}
	}
}

// bitsToUint64 packs up to 64 MSB-first bits into a uint64. Bit
// 63 of the result is bits[0] (the wire-order leading bit).
func bitsToUint64(bits []byte) uint64 {
	var v uint64
	n := len(bits)
	if n > 64 {
		n = 64
	}
	for i := 0; i < n; i++ {
		if bits[i]&1 != 0 {
			v |= uint64(1) << uint(63-i)
		}
	}
	return v
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
