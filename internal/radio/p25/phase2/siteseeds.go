package phase2

import (
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// SiteSeeds is the latest P25 site identity per system, fed from the control
// channels' site updates (trunking.SiteUpdate), and the PN44 descramble seed
// it names (issue #1364).
//
// A Phase 2 traffic channel is scrambled with a PN44 sequence seeded from the
// site's (WACN, System ID, Color Code = NAC). A control channel learns that
// identity from its Network Status Broadcast, which a site sends every few
// seconds; a call granted before the first one carries no seed. The voice
// chain and the signalling follower take the seed from here as soon as the
// control channel publishes the identity. The zero value is ready and it is
// safe for concurrent use.
type SiteSeeds struct {
	mu   sync.Mutex
	byID map[string]siteIdentity
}

type siteIdentity struct {
	wacn  uint32
	sysid uint16
	nac   uint16
}

// Note records a site update that names the network (WACN and System ID).
// Updates that do not, such as one triggered by an RFSS Status Broadcast seen
// before the Network Status Broadcast, are ignored: they cannot name a seed.
func (s *SiteSeeds) Note(su trunking.SiteUpdate) {
	if su.System == "" || su.WACN == 0 || su.SystemID == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byID == nil {
		s.byID = make(map[string]siteIdentity)
	}
	id := siteIdentity{wacn: su.WACN, sysid: su.SystemID, nac: su.NAC}
	if id.nac == 0 {
		// Keep a NAC learned earlier rather than forget it on an update
		// that does not carry one.
		id.nac = s.byID[su.System].nac
	}
	s.byID[su.System] = id
}

// Seed returns the PN44 seed for system, or 0 while its identity is unknown.
// grantNAC (the NAC the grant was heard on) stands in when the site update
// did not carry one; with no NAC at all there is no seed.
func (s *SiteSeeds) Seed(system string, grantNAC uint16) uint64 {
	s.mu.Lock()
	id, ok := s.byID[system]
	s.mu.Unlock()
	if !ok {
		return 0
	}
	nac := id.nac
	if nac == 0 {
		nac = grantNAC
	}
	if nac == 0 {
		return 0
	}
	return framing.PN44SeedFromIdentity(id.wacn, id.sysid, nac&0x0FFF)
}

// SoftReliability turns the receiver's soft symbols (the SoftDibitSink's
// differentials, rotated so the two bits are the signs of the real and
// imaginary parts) into each dibit's reliability for AirFramer.ProcessSoft:
// the distance of its less certain bit from the decision boundary. dst is
// reused when it has room.
func SoftReliability(soft []complex64, dst []float32) []float32 {
	if cap(dst) < len(soft) {
		dst = make([]float32, len(soft))
	}
	dst = dst[:len(soft)]
	for i, v := range soft {
		re, im := real(v), imag(v)
		if re < 0 {
			re = -re
		}
		if im < 0 {
			im = -im
		}
		dst[i] = min(re, im)
	}
	return dst
}
