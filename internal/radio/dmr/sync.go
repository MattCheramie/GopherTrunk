// Package dmr decodes ETSI TS 102 361 (DMR) burst structure: sync patterns,
// slot-type fields, and Tier II / III control signaling. DMR is a 2-slot
// TDMA system at 4.8 kbaud per slot; each 30 ms frame carries one burst per
// timeslot, framed by a 48-bit sync pattern.
package dmr

// SyncPattern enumerates the 48-bit sync words defined in ETSI
// TS 102 361-1 §9.1.1, encoded as 24 dibits MSB-first.
type SyncPattern struct {
	Name   string
	Hex    uint64 // low 48 bits
	Dibits [24]uint8
}

// All sync patterns are listed below. Hex is the 48-bit value spanning the
// burst sync field; Dibits is the same value expressed as 24 dibits.

var (
	BSVoice  = mkSync("BS-Voice", 0x755FD7DF75F7)
	BSData   = mkSync("BS-Data", 0xDFF57D75DF5D)
	MSVoice  = mkSync("MS-Voice", 0x7F7D5DD57DFD)
	MSData   = mkSync("MS-Data", 0xD5D7F77FD757)
	MSRC     = mkSync("MS-RC", 0x77D55F7DFD77)
	DMVoice1 = mkSync("DM-Voice-TS1", 0x5D577F7757FF)
	DMVoice2 = mkSync("DM-Voice-TS2", 0x7DFFD5F55D5F)
	DMData1  = mkSync("DM-Data-TS1", 0xF7FDD5DDFD55)
	DMData2  = mkSync("DM-Data-TS2", 0xD7557F5FF7F5)
)

// AllSyncs lists every defined sync pattern. The order is stable so callers
// can rely on indexing.
var AllSyncs = []SyncPattern{
	BSVoice, BSData, MSVoice, MSData, MSRC,
	DMVoice1, DMVoice2, DMData1, DMData2,
}

func mkSync(name string, hex uint64) SyncPattern {
	var d [24]uint8
	for i := 0; i < 24; i++ {
		// Highest dibit lives in bits 47..46 (i=0); shift accordingly.
		shift := uint(46 - 2*i)
		d[i] = uint8((hex >> shift) & 0x3)
	}
	return SyncPattern{Name: name, Hex: hex, Dibits: d}
}

// SyncDetector slides a 24-dibit window over a stream and emits matches
// against any of the supplied patterns within the configured tolerance.
type SyncDetector struct {
	patterns  []SyncPattern
	tolerance int
	hist      [24]uint8
	primed    int
	pos       int
}

func NewSyncDetector(patterns []SyncPattern, tolerance int) *SyncDetector {
	if tolerance < 0 {
		tolerance = 4
	}
	if len(patterns) == 0 {
		patterns = AllSyncs
	}
	return &SyncDetector{patterns: patterns, tolerance: tolerance}
}

// Match reports a single hit: the dibit index where the sync ends and the
// matched pattern name.
type Match struct {
	Index   int
	Pattern SyncPattern
}

// Process scans src and appends matches to dst. baseIndex is the absolute
// dibit index of src[0]. Returns the new baseIndex.
func (s *SyncDetector) Process(dst []Match, src []uint8, baseIndex int) ([]Match, int) {
	for i, d := range src {
		s.hist[s.pos] = d
		s.pos = (s.pos + 1) % 24
		if s.primed < 24 {
			s.primed++
			continue
		}
		bestErrs := s.tolerance + 1
		var best SyncPattern
		for _, p := range s.patterns {
			errs := 0
			idx := s.pos
			for k := 0; k < 24; k++ {
				if s.hist[idx] != p.Dibits[k] {
					errs++
					if errs >= bestErrs {
						break
					}
				}
				idx = (idx + 1) % 24
			}
			if errs < bestErrs {
				bestErrs = errs
				best = p
			}
		}
		if bestErrs <= s.tolerance {
			dst = append(dst, Match{Index: baseIndex + i, Pattern: best})
		}
	}
	return dst, baseIndex + len(src)
}
