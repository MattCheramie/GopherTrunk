package dstar

// BitSink consumes the raw stream of bits a D-STAR receiver decodes
// from the GMSK demodulator + matched filter + symbol-clock recovery
// chain. D-STAR is 2-level (GMSK at 4800 bps) so every symbol carries
// exactly one bit — unlike the 4-level C4FM family which fans into
// DibitSink. Wire it through `internal/radio/dstar/receiver.Options`.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime so downstream sync detection / window slicing can report
// stable positions.
type BitSink func(bits []byte, baseIdx int)

// D-STAR sync words, in on-air time order (#1261).
//
// Earlier versions of this file carried 0xEAA060 and 0x552D16 here,
// unpacked MSB first. Neither is what a D-STAR radio sends, in either
// polarity: 0x552D16 is MMDVMHost's slow-data sync as BYTES, which go
// on air LSB first, and 0xEAA060 matched nothing. Every synthetic test
// built its signal from the same constants, so they agreed with each
// other and GopherTrunk never framed a real transmission. The values
// below are the MMDVM firmware's receive patterns (DStarDefines.h,
// DSTAR_FRAME_SYNC_DATA / DSTAR_DATA_SYNC_DATA / DSTAR_END_SYNC_DATA),
// which MMDVM matches bit by bit as they arrive, so MSB first here IS
// time order. Bit 1 is what MMDVM reads from a negative sample; GT's
// receiver slices the other way round, and radios differ, so every
// detector here also looks for the complement (see Polarity).
const (
	// FrameSyncHex opens a transmission's header: the tail of the
	// 1010… bit-sync preamble, the 15-bit frame sync 111011001010000
	// (MMDVM's TX sends it as 0xEA 0xA6, LSB first) and the four zero
	// bits that precede the 660-bit header FEC section, which starts
	// on the next bit.
	FrameSyncHex uint64 = 0x557650

	// SlowDataSyncHex is the slow-data field of DV frame 0 of every
	// 21-frame cycle (bytes 0x55 0x2D 0x16, LSB first).
	SlowDataSyncHex uint64 = 0xAAB468

	// EndSyncHex ends a transmission (0x55 0x55 0x55 0x55 0xC8 0x7A,
	// LSB first). It replaces the 97th..144th bits after a DV frame.
	EndSyncHex uint64 = 0xAAAAAAAA135E

	FrameSyncBits    = 24
	SlowDataSyncBits = 24
	EndSyncBits      = 48
)

// FrameSyncBitsSlice returns the 24 bits of the frame sync pattern in
// time order.
func FrameSyncBitsSlice() []uint8 { return hexToBits(FrameSyncHex, FrameSyncBits) }

// SlowDataSyncBitsSlice returns the 24 bits of the slow-data sync in
// time order.
func SlowDataSyncBitsSlice() []uint8 { return hexToBits(SlowDataSyncHex, SlowDataSyncBits) }

// EndSyncBitsSlice returns the 48 bits of the end-of-transmission
// pattern in time order.
func EndSyncBitsSlice() []uint8 { return hexToBits(EndSyncHex, EndSyncBits) }

// BitSyncBits returns n bits of the 1010… bit-sync preamble a
// transmitter sends before the frame sync (0xAA bytes, LSB first, so
// the stream opens 0101…). n should be even so the frame sync follows
// in phase.
func BitSyncBits(n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(i & 1)
	}
	return out
}

// BytesToBitsLSB unpacks bytes as D-STAR sends them: each byte LSB
// first.
func BytesToBitsLSB(b []byte) []uint8 {
	out := make([]uint8, 0, 8*len(b))
	for _, x := range b {
		for k := 0; k < 8; k++ {
			out = append(out, x>>uint(k)&1)
		}
	}
	return out
}

// Polarity is the sense in which a receiver's bits relate to MMDVM's.
// The frame and slow-data syncs are not their own complements, so a
// detector learns the polarity from which one matched.
type Polarity uint8

const (
	PolarityNormal Polarity = iota
	PolarityInverted
)

// invertBits returns the complement of a bit pattern.
func invertBits(p []uint8) []uint8 {
	out := make([]uint8, len(p))
	for i, b := range p {
		out[i] = b ^ 1
	}
	return out
}

func hexToBits(hex uint64, n int) []uint8 {
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		shift := uint(n - 1 - i)
		out[i] = uint8((hex >> shift) & 0x1)
	}
	return out
}

// SyncDetector slides a window over a bit stream and reports indices
// where the configured pattern matches within `tolerance` bit
// mismatches. Same shape as the Phase 1 / DMR / NXDN / dPMR / TETRA
// sync detectors, but operating on bits rather than dibits — D-STAR
// is GMSK at 4800 bps so the symbol stream is per-bit.
type SyncDetector struct {
	pattern   []uint8
	tolerance int
	hist      []uint8
	primed    int
	pos       int
}

// NewSyncDetector accepts an N-bit pattern (typically
// FrameSyncBitsSlice or SlowDataSyncBitsSlice) and a tolerance.
// A zero tolerance demands an exact match.
func NewSyncDetector(pattern []uint8, tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 1
	}
	cp := make([]uint8, len(pattern))
	copy(cp, pattern)
	return &SyncDetector{
		pattern:   cp,
		tolerance: tolerance,
		hist:      make([]uint8, len(cp)),
	}
}

// Process scans src and appends to dst the absolute bit indices where
// the sync pattern ends within tolerance.
func (s *SyncDetector) Process(dst []int, src []uint8, baseIndex int) ([]int, int) {
	n := len(s.pattern)
	for i, b := range src {
		s.hist[s.pos] = b & 1
		s.pos = (s.pos + 1) % n
		if s.primed < n {
			s.primed++
			continue
		}
		mismatch := 0
		idx := s.pos
		for k := 0; k < n; k++ {
			if s.hist[idx] != s.pattern[k] {
				mismatch++
				if mismatch > s.tolerance {
					break
				}
			}
			idx = (idx + 1) % n
		}
		if mismatch <= s.tolerance {
			dst = append(dst, baseIndex+i)
		}
	}
	return dst, baseIndex + len(src)
}
