// Package hunt is GopherTrunk's site/system "hunting" layer: it turns the
// output of the signal-lab decoders into a DiscoveredSystem — a structured
// map of a trunked radio system observed off the air (or from a capture) —
// and exports that map to standardized files plus a RadioReference.com
// submission package.
//
// Why this exists: many states have trunked systems that are undocumented on
// RadioReference. GopherTrunk can already decode a control channel once you
// know its frequency and identity; this package closes the loop by
// accumulating what the decoder observes (system identity, per-site control
// channels, neighbor sites, talkgroups) into a single model that round-trips
// straight back into GopherTrunk's import bundle and into other scanners.
//
// The model is intentionally protocol-neutral. Full multi-site topology
// (WACN/SYSID/RFSS/Site/neighbors) is only available for P25 today; for the
// other protocols the accumulator records the identity fields the decoder
// surfaces plus the single observed site and its talkgroups, which is still
// enough to export and submit.
package hunt

import (
	"fmt"
	"sort"
	"time"
)

// DiscoveredSystem is the accumulated map of one trunked system. It is built
// up incrementally as captures/observations are folded in (see Accumulator),
// then handed to the exporters. Its fields are shaped to convert cleanly onto
// the importer's parsedSystem (see cmd/gophertrunk/hunt_export.go) so a
// discovery exported as a bundle re-imports without loss.
type DiscoveredSystem struct {
	// Name is operator-assigned, or synthesized from the identity when blank.
	Name string `json:"name"`
	// Protocol is the trunking.Protocol.String() spelling (e.g. "p25").
	Protocol string `json:"protocol"`

	// P25 / generic identity. Zero ⇒ unknown.
	WACN     uint32 `json:"wacn,omitempty"`
	SystemID uint16 `json:"system_id,omitempty"`
	NAC      uint16 `json:"nac,omitempty"`

	// Identity carries the remaining per-protocol identity fields the decoder
	// surfaced (DMR ColorCode, NXDN RAN, TETRA MCC/MNC, Motorola/EDACS system
	// id, …) keyed by the decoder's field name. It is informational and is
	// rendered into the RR package.
	Identity map[string]any `json:"identity,omitempty"`

	// Geography — operator-supplied; used by the RR duplicate check and the
	// submission package.
	State    string `json:"state,omitempty"`
	County   string `json:"county,omitempty"`
	Location string `json:"location,omitempty"`

	Sites      []DiscoveredSite      `json:"sites"`
	Talkgroups []DiscoveredTalkgroup `json:"talkgroups"`
	BandPlan   []BandPlanEntry       `json:"band_plan,omitempty"`

	// Confidence is the min protocol-identification confidence across the
	// control channels folded into this system (0..1).
	Confidence float64   `json:"confidence"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

// DiscoveredSite is one RF site of the system, keyed by (RFSS, SiteID).
type DiscoveredSite struct {
	RFSS            uint8               `json:"rfss"`
	SiteID          uint8               `json:"site_id"`
	SiteName        string              `json:"site_name"`
	County          string              `json:"county,omitempty"`
	ControlChannels []DiscoveredChannel `json:"control_channels"`
	Secondary       []uint32            `json:"secondary,omitempty"`
	Neighbors       []NeighborRef       `json:"neighbors,omitempty"`
}

// DiscoveredChannel is one observed frequency on a site.
type DiscoveredChannel struct {
	FrequencyHz uint32  `json:"frequency_hz"`
	IsControl   bool    `json:"is_control"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// DiscoveredTalkgroup is one talkgroup observed on the control channel. On a
// blind discovery only the numeric id and activity are known; the descriptive
// fields are left blank for the operator (or RR) to fill in.
type DiscoveredTalkgroup struct {
	Dec       uint32    `json:"dec"`
	Hex       string    `json:"hex"`
	Encrypted bool      `json:"encrypted,omitempty"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
}

// NeighborRef is an adjacent site advertised by the control channel.
type NeighborRef struct {
	RFSS          uint8  `json:"rfss"`
	Site          uint8  `json:"site"`
	ChannelID     uint8  `json:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty"`
	// FrequencyHz is the neighbour's control-channel frequency, resolved from
	// (ChannelID, ChannelNumber) against the system band plan at finish. Zero
	// when the band plan doesn't cover the channel (e.g. no IDEN_UP seen yet).
	FrequencyHz uint32 `json:"frequency_hz,omitempty"`
}

// BandPlanEntry is one P25 IDEN_UP-equivalent band-plan mapping.
type BandPlanEntry struct {
	ChannelID   uint8  `json:"channel_id"`
	BaseHz      uint64 `json:"base_hz"`
	SpacingHz   uint32 `json:"spacing_hz"`
	BandwidthHz uint32 `json:"bandwidth_hz,omitempty"`
	TxOffsetHz  int64  `json:"tx_offset_hz,omitempty"`
}

// DisplayName returns Name when set, otherwise a synthesized identifier from
// the protocol + system id so an unnamed discovery still has a stable handle.
func (s *DiscoveredSystem) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	proto := s.Protocol
	if proto == "" {
		proto = "unknown"
	}
	switch {
	case s.WACN != 0 && s.SystemID != 0:
		return fmt.Sprintf("Unknown-%s-%05X-%03X", proto, s.WACN, s.SystemID)
	case s.SystemID != 0:
		return fmt.Sprintf("Unknown-%s-%03X", proto, s.SystemID)
	default:
		return fmt.Sprintf("Unknown-%s", proto)
	}
}

// site returns a pointer to the site with the given (rfss, siteID), creating
// it if absent. Callers mutate the returned site in place.
func (s *DiscoveredSystem) site(rfss, siteID uint8) *DiscoveredSite {
	for i := range s.Sites {
		if s.Sites[i].RFSS == rfss && s.Sites[i].SiteID == siteID {
			return &s.Sites[i]
		}
	}
	s.Sites = append(s.Sites, DiscoveredSite{RFSS: rfss, SiteID: siteID})
	return &s.Sites[len(s.Sites)-1]
}

// addControlChannel records a control-channel frequency on (rfss, siteID),
// de-duplicating by frequency and keeping the highest confidence seen.
func (s *DiscoveredSystem) addControlChannel(rfss, siteID uint8, hz uint32, confidence float64) {
	site := s.site(rfss, siteID)
	for i := range site.ControlChannels {
		if site.ControlChannels[i].FrequencyHz == hz {
			site.ControlChannels[i].IsControl = true
			if confidence > site.ControlChannels[i].Confidence {
				site.ControlChannels[i].Confidence = confidence
			}
			return
		}
	}
	site.ControlChannels = append(site.ControlChannels, DiscoveredChannel{
		FrequencyHz: hz, IsControl: true, Confidence: confidence,
	})
}

// addTalkgroup records (or bumps the activity count of) a talkgroup.
func (s *DiscoveredSystem) addTalkgroup(dec uint32, encrypted bool, at time.Time) {
	for i := range s.Talkgroups {
		if s.Talkgroups[i].Dec == dec {
			s.Talkgroups[i].Count++
			if encrypted {
				s.Talkgroups[i].Encrypted = true
			}
			return
		}
	}
	s.Talkgroups = append(s.Talkgroups, DiscoveredTalkgroup{
		Dec:       dec,
		Hex:       fmt.Sprintf("%x", dec),
		Encrypted: encrypted,
		Count:     1,
		FirstSeen: at,
	})
}

// addNeighbor records an adjacent site on (rfss, siteID), de-duplicating by
// the neighbor's (RFSS, Site).
func (s *DiscoveredSystem) addNeighbor(rfss, siteID uint8, n NeighborRef) {
	site := s.site(rfss, siteID)
	for i := range site.Neighbors {
		if site.Neighbors[i].RFSS == n.RFSS && site.Neighbors[i].Site == n.Site {
			// First identity wins, but backfill a frequency once a later
			// observation resolves one (the decoder may not have had a band-plan
			// resolver on the first sighting).
			if site.Neighbors[i].FrequencyHz == 0 && n.FrequencyHz != 0 {
				site.Neighbors[i].FrequencyHz = n.FrequencyHz
			}
			return
		}
	}
	site.Neighbors = append(site.Neighbors, n)
}

// addBandPlanEntry records a band-plan slot, de-duplicating by channel ID
// (the first observation of a channel ID wins).
func (s *DiscoveredSystem) addBandPlanEntry(e BandPlanEntry) {
	for _, existing := range s.BandPlan {
		if existing.ChannelID == e.ChannelID {
			return
		}
	}
	s.BandPlan = append(s.BandPlan, e)
}

// sortAll puts sites, channels and talkgroups in a deterministic order so the
// exporters produce stable output (important for golden tests and diffs).
func (s *DiscoveredSystem) sortAll() {
	sort.Slice(s.Sites, func(i, j int) bool {
		if s.Sites[i].RFSS != s.Sites[j].RFSS {
			return s.Sites[i].RFSS < s.Sites[j].RFSS
		}
		return s.Sites[i].SiteID < s.Sites[j].SiteID
	})
	for i := range s.Sites {
		cc := s.Sites[i].ControlChannels
		sort.Slice(cc, func(a, b int) bool { return cc[a].FrequencyHz < cc[b].FrequencyHz })
		nb := s.Sites[i].Neighbors
		sort.Slice(nb, func(a, b int) bool {
			if nb[a].RFSS != nb[b].RFSS {
				return nb[a].RFSS < nb[b].RFSS
			}
			return nb[a].Site < nb[b].Site
		})
	}
	sort.Slice(s.Talkgroups, func(i, j int) bool { return s.Talkgroups[i].Dec < s.Talkgroups[j].Dec })
	sort.Slice(s.BandPlan, func(i, j int) bool { return s.BandPlan[i].ChannelID < s.BandPlan[j].ChannelID })
	// Resolve neighbour control-channel frequencies now that the band plan is
	// fully accumulated (it may be incomplete on any single observation).
	s.resolveNeighborFreqs()
}

// bandPlanFreq resolves a (channelID, channelNumber) pair to a downlink
// frequency using the accumulated band plan, mirroring P25's IDEN_UP math
// (BaseHz + channelNumber*SpacingHz). It deliberately does not apply TxOffsetHz
// — that is the uplink offset; a control-channel reference is the downlink.
// Returns false when no band-plan entry covers channelID or the result is out
// of range.
func (s *DiscoveredSystem) bandPlanFreq(channelID uint8, channelNumber uint16) (uint32, bool) {
	for _, e := range s.BandPlan {
		if e.ChannelID != channelID {
			continue
		}
		hz := e.BaseHz + uint64(channelNumber)*uint64(e.SpacingHz)
		if hz == 0 || hz > 0xFFFFFFFF {
			return 0, false
		}
		return uint32(hz), true
	}
	return 0, false
}

// resolveNeighborFreqs fills in NeighborRef.FrequencyHz for every neighbour
// whose (ChannelID, ChannelNumber) the band plan can resolve. It is idempotent
// (a neighbour already carrying a frequency is left untouched), so it is safe to
// call from sortAll on every finish/export.
func (s *DiscoveredSystem) resolveNeighborFreqs() {
	if len(s.BandPlan) == 0 {
		return
	}
	for i := range s.Sites {
		for j := range s.Sites[i].Neighbors {
			n := &s.Sites[i].Neighbors[j]
			if n.FrequencyHz != 0 || (n.ChannelID == 0 && n.ChannelNumber == 0) {
				continue
			}
			if hz, ok := s.bandPlanFreq(n.ChannelID, n.ChannelNumber); ok {
				n.FrequencyHz = hz
			}
		}
	}
}
