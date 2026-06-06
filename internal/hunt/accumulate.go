package hunt

import (
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// Observation is one control-channel decode folded into a DiscoveredSystem.
// It is the bridge between the signal lab (which produces a siglab.Result per
// capture) and the discovery model. One Observation typically corresponds to
// one captured control channel of one site.
type Observation struct {
	// Protocol is the trunking.Protocol.String() spelling of the decoded CC.
	Protocol string
	// Confidence is the protocol-identification confidence (0..1). When the
	// protocol was supplied by the operator rather than auto-identified, pass
	// 1.0.
	Confidence float64
	// Result is the signal-lab decode of the capture. Lock + Grants + Events
	// are mined for identity and talkgroups.
	Result *siglab.Result
	// FallbackFreqHz is the capture's nominal center frequency, used as the
	// control-channel frequency when the decoder's lock payload doesn't carry
	// one (e.g. a baseband capture). 0 ⇒ no fallback.
	FallbackFreqHz uint32
	// At is the wall-clock time the capture was taken (defaults to now).
	At time.Time
}

// identityKeys are the per-protocol identity field names harvested from the
// lock payload and from event payloads. flattenPayload exposes exported
// struct fields by name, so any decoder that carries these surfaces them here
// without a per-protocol switch. WACN/SystemID/RFSS/Site are P25; ColorCode
// (DMR), RAN (NXDN), MCC/MNC (TETRA), SystemID (Motorola/EDACS) cover the rest.
var identityKeys = []string{
	"WACN", "SystemID", "RFSS", "Site", "NAC",
	"ColorCode", "RAN", "MCC", "MNC", "LRA",
}

// Accumulate folds an Observation into dst, creating sites/talkgroups and
// merging identity. It de-duplicates control channels by frequency, sites by
// (RFSS, Site), and talkgroups by decimal id, so re-observing the same control
// channel is idempotent (apart from bumping talkgroup activity counts).
func Accumulate(dst *DiscoveredSystem, obs Observation) {
	at := obs.At
	if at.IsZero() {
		at = time.Now()
	}
	if dst.FirstSeen.IsZero() || at.Before(dst.FirstSeen) {
		dst.FirstSeen = at
	}
	if at.After(dst.LastSeen) {
		dst.LastSeen = at
	}
	if dst.Protocol == "" {
		dst.Protocol = obs.Protocol
	}
	if dst.Confidence == 0 || (obs.Confidence > 0 && obs.Confidence < dst.Confidence) {
		dst.Confidence = obs.Confidence
	}
	if obs.Result == nil {
		return
	}

	// Harvest identity from the lock payload first, then from every event
	// payload (the network/RFSS-status TSBKs ride in events, not the lock).
	if dst.Identity == nil {
		dst.Identity = map[string]any{}
	}
	var ccFreq uint32
	if lk := obs.Result.Lock; lk != nil {
		ccFreq = lk.FrequencyHz
		harvestIdentity(dst, lk.Fields)
	}
	for _, ev := range obs.Result.Events {
		harvestIdentity(dst, ev.Fields)
	}
	// Fold the decoder's accumulated topology (the only source of
	// WACN/SYSID/RFSS/Site + neighbors + band plan — these never ride on
	// events). This runs before site placement so RFSS/Site are authoritative.
	foldTopology(dst, obs.Result.Topology)

	// Fall back to the capture's nominal center when the lock didn't carry an
	// absolute frequency (baseband captures lock at 0 Hz).
	if ccFreq == 0 {
		ccFreq = obs.FallbackFreqHz
	}

	// Place this control channel on its site. RFSS/Site come from the
	// harvested identity (0,0 when the decoder didn't surface them). A lock
	// with no usable frequency still records the site so identity and
	// talkgroups attach — but only a non-zero frequency is recorded as a
	// channel (a 0 Hz channel can't be exported).
	rfss := uint8(dst.RFSS())
	site := uint8(dst.SiteNum())
	switch {
	case ccFreq != 0:
		dst.addControlChannel(rfss, site, ccFreq, obs.Confidence)
	case obs.Result.Locked || obs.Result.Topology != nil:
		dst.site(rfss, site) // ensure the site exists for identity/talkgroups
	}

	// Attach neighbor (adjacent) sites + secondary control channels advertised
	// for the camped site, now that the site exists.
	if t := obs.Result.Topology; t != nil {
		for _, n := range t.Neighbors {
			dst.addNeighbor(rfss, site, NeighborRef{
				RFSS:          n.RFSS,
				Site:          n.Site,
				ChannelID:     n.ChannelID,
				ChannelNumber: n.ChannelNumber,
			})
		}
		if len(t.Secondary) > 0 {
			st := dst.site(rfss, site)
			for _, s := range t.Secondary {
				if s.FrequencyHz != 0 {
					st.Secondary = appendUniqueFreq(st.Secondary, s.FrequencyHz)
				}
			}
		}
	}

	// Talkgroups: every grant's destination group is an observed talkgroup.
	for _, g := range obs.Result.Grants {
		if g.GroupID == 0 {
			continue
		}
		dst.addTalkgroup(g.GroupID, g.Encrypted, at)
	}
}

// foldTopology merges a siglab TopologySnapshot into the discovered system:
// typed identity (WACN/SystemID), the Identity map (RFSS/Site/LRA + the
// per-protocol fields used for display and site placement), and the band plan.
// First non-zero value wins, mirroring harvestIdentity.
func foldTopology(dst *DiscoveredSystem, t *siglab.TopologySnapshot) {
	if t == nil {
		return
	}
	if dst.WACN == 0 && t.WACN != 0 {
		dst.WACN = t.WACN
	}
	if dst.SystemID == 0 && t.SystemID != 0 {
		dst.SystemID = uint16(t.SystemID)
	}
	setIdentity := func(key string, v uint64) {
		if v == 0 {
			return
		}
		if _, present := dst.Identity[key]; !present {
			dst.Identity[key] = v
		}
	}
	setIdentity("RFSS", uint64(t.RFSS))
	setIdentity("Site", uint64(t.Site))
	setIdentity("LRA", uint64(t.LRA))
	setIdentity("ColorCode", uint64(t.ColorCode))
	setIdentity("RAN", uint64(t.RAN))
	setIdentity("MCC", uint64(t.MCC))
	setIdentity("MNC", uint64(t.MNC))
	setIdentity("LocationArea", uint64(t.LocationArea))

	for _, e := range t.BandPlan {
		dst.addBandPlanEntry(BandPlanEntry{
			ChannelID:   e.ChannelID,
			BaseHz:      e.BaseHz,
			SpacingHz:   e.SpacingHz,
			BandwidthHz: e.BandwidthHz,
			TxOffsetHz:  e.TxOffsetHz,
		})
	}
}

// appendUniqueFreq appends hz to s unless already present.
func appendUniqueFreq(s []uint32, hz uint32) []uint32 {
	for _, e := range s {
		if e == hz {
			return s
		}
	}
	return append(s, hz)
}

// harvestIdentity copies recognised identity fields from a flattened payload
// into the system's typed identity (WACN/SystemID/NAC) and Identity map. It
// only overwrites a typed field when the current value is zero, so the first
// non-zero observation wins and later ones don't clobber it with a 0.
func harvestIdentity(dst *DiscoveredSystem, fields map[string]any) {
	if fields == nil {
		return
	}
	for _, k := range identityKeys {
		v, ok := fields[k]
		if !ok {
			continue
		}
		u, ok := toUint64(v)
		if !ok || u == 0 {
			continue
		}
		switch k {
		case "WACN":
			if dst.WACN == 0 {
				dst.WACN = uint32(u)
			}
		case "SystemID":
			if dst.SystemID == 0 {
				dst.SystemID = uint16(u)
			}
		case "NAC":
			if dst.NAC == 0 {
				dst.NAC = uint16(u)
			}
		}
		if _, present := dst.Identity[k]; !present {
			dst.Identity[k] = u
		}
	}
}

// RFSS returns the RFSS id harvested into the Identity map (0 when unknown).
func (s *DiscoveredSystem) RFSS() uint64 { return identityUint(s.Identity, "RFSS") }

// SiteNum returns the Site id harvested into the Identity map (0 when unknown).
func (s *DiscoveredSystem) SiteNum() uint64 { return identityUint(s.Identity, "Site") }

func identityUint(m map[string]any, key string) uint64 {
	if m == nil {
		return 0
	}
	u, _ := toUint64(m[key])
	return u
}

// toUint64 coerces the numeric types reflect produces (and JSON's float64)
// into a uint64. Returns ok=false for non-numeric or negative values.
func toUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint:
		return uint64(n), true
	case uint8:
		return uint64(n), true
	case uint16:
		return uint64(n), true
	case uint32:
		return uint64(n), true
	case uint64:
		return n, true
	case int:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case int8:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case int16:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case int32:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case int64:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case float64:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	case float32:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true
	default:
		return 0, false
	}
}
