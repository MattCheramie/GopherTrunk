package composer

import (
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// P25 Phase 2 descramble seed learned after the grant (issue #1364).
//
// A Phase 2 traffic channel is scrambled with a PN44 sequence seeded from the
// site's (WACN, System ID, Color Code = NAC). The control channel learns that
// identity from its Network Status Broadcast, which a site sends every few
// seconds; a call granted before the first one carries no seed. Until this
// change the control channels filled that gap with a seed built from zeros
// and the system's configured Site (which the configuration never sets, and
// which is 8 bits where the Color Code is 12), so a call granted in the first
// seconds after lock decoded nothing for its whole length, and the chain could
// not even tell it had the wrong seed. Grants now carry no seed until the
// identity is known, and the chain takes it from the site's identity as soon
// as the control channel publishes it.

// p25SiteIdentity is the part of a site's identity the PN44 seed needs.
type p25SiteIdentity struct {
	wacn  uint32
	sysid uint16
	nac   uint16
}

// p25Identities is the latest P25 site identity per system, fed from
// KindSiteUpdate. The zero value is ready.
type p25Identities struct {
	mu   sync.Mutex
	byID map[string]p25SiteIdentity
}

// note records a site update that names the network (WACN and System ID).
// Updates that do not, such as an RFSS Status Broadcast seen before the
// Network Status Broadcast, are ignored: they cannot name a seed.
func (p *p25Identities) note(su trunking.SiteUpdate) {
	if su.System == "" || su.WACN == 0 || su.SystemID == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.byID == nil {
		p.byID = make(map[string]p25SiteIdentity)
	}
	id := p25SiteIdentity{wacn: su.WACN, sysid: su.SystemID, nac: su.NAC}
	if id.nac == 0 {
		// Keep a NAC learned earlier rather than forget it on an update
		// that does not carry one.
		id.nac = p.byID[su.System].nac
	}
	p.byID[su.System] = id
}

// seed returns the PN44 seed for system, or 0 while its identity is unknown.
// grantNAC (the NAC the grant was heard on) stands in when the site update
// did not carry one; with no NAC at all there is no seed.
func (p *p25Identities) seed(system string, grantNAC uint16) uint64 {
	p.mu.Lock()
	id, ok := p.byID[system]
	p.mu.Unlock()
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

// p25p2SeedWaitMaxBursts bounds the bursts a Phase 2 chain holds while it
// waits for its seed: 10 s of one carrier (a burst every 30 ms). A site
// repeats its Network Status Broadcast every few seconds, so this covers the
// wait; a call that outlasts it keeps its newest bursts.
const p25p2SeedWaitMaxBursts = 334
