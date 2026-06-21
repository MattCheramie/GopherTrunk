package hunt

import (
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// NetworkReport flattens the discovered system into the protocol-neutral
// trunking.NetworkReport the shared renderer consumes. Unlike the single-site
// CLI/daemon path, a DiscoveredSystem aggregates several sites, so it emits one
// ReportSite per DiscoveredSite under a shared identity header + shared band
// plan. Control channels are stored as resolved frequencies (no band-plan
// coordinates), so site control channels carry a downlink frequency only;
// neighbours keep their (id, number) and get an uplink derived from the matching
// band's transmit offset.
func (s *DiscoveredSystem) NetworkReport() trunking.NetworkReport {
	offsets := make(map[uint8]int64, len(s.BandPlan))
	for _, b := range s.BandPlan {
		offsets[b.ChannelID] = b.TxOffsetHz
	}

	r := trunking.NetworkReport{
		Name:     s.DisplayName(),
		Protocol: s.Protocol,
		WACN:     s.WACN,
		SystemID: uint32(s.SystemID),
		NAC:      s.NAC,
	}
	for _, st := range s.Sites {
		rs := trunking.ReportSite{RFSS: st.RFSS, Site: st.SiteID}
		primarySet := false
		for _, ch := range st.ControlChannels {
			if !ch.IsControl {
				continue
			}
			c := trunking.ReportChannel{DownlinkHz: ch.FrequencyHz}
			if !primarySet {
				rs.PrimaryCC = c
				primarySet = true
			} else {
				rs.SecondaryCC = append(rs.SecondaryCC, c)
			}
		}
		for _, hz := range st.Secondary {
			rs.SecondaryCC = append(rs.SecondaryCC, trunking.ReportChannel{DownlinkHz: hz})
		}
		for _, n := range st.Neighbors {
			c := trunking.ReportChannel{ChannelID: n.ChannelID, ChannelNumber: n.ChannelNumber, DownlinkHz: n.FrequencyHz}
			if n.FrequencyHz != 0 {
				if off, ok := offsets[n.ChannelID]; ok && off != 0 {
					if up := int64(n.FrequencyHz) + off; up > 0 {
						c.UplinkHz = uint32(up)
					}
				}
			}
			rs.Neighbors = append(rs.Neighbors, trunking.ReportNeighbor{RFSS: n.RFSS, Site: n.Site, Channel: c})
		}
		r.Sites = append(r.Sites, rs)
	}
	for _, b := range s.BandPlan {
		r.Bands = append(r.Bands, trunking.ReportBand{
			ChannelID:   b.ChannelID,
			BaseHz:      b.BaseHz,
			SpacingHz:   b.SpacingHz,
			BandwidthHz: b.BandwidthHz,
			TxOffsetHz:  b.TxOffsetHz,
		})
	}
	return r
}

// writeSummary renders the human-readable network-configuration report for the
// discovered system. Callers (WriteWithRRDiff) sortAll() first so neighbour
// frequencies are resolved and site order is stable.
func writeSummary(w io.Writer, sys *DiscoveredSystem) error {
	trunking.RenderNetworkReport(w, sys.NetworkReport())
	return nil
}
