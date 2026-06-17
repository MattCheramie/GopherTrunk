package api

import (
	"net/http"
	"sort"
)

// handleListSites serves GET /api/v1/sites: the union of P25 sites GT
// has discovered from the control channel and the sites named in the
// system config, keyed by (system, RFSS, site). Discovered rows carry
// control_channel_hz (and WACN / SystemID); configured names are merged
// onto matching rows, and configured sites GT has not yet heard are
// listed with control_channel_hz omitted so a site's name always shows
// (issue #698).
func (s *Server) handleListSites(w http.ResponseWriter, _ *http.Request) {
	type key struct {
		system string
		rfss   uint8
		site   uint8
	}
	byKey := make(map[key]*SiteDTO)

	// Discovered sites first — these carry the live control-channel
	// frequency and network identity.
	if s.sites != nil {
		for _, si := range s.sites.Sites() {
			k := key{si.System, si.RFSSID, si.SiteID}
			byKey[k] = &SiteDTO{
				System:           si.System,
				RFSSID:           si.RFSSID,
				SiteID:           si.SiteID,
				ControlChannelHz: si.ControlChannelHz,
				WACN:             si.WACN,
				SystemID:         si.SystemID,
			}
		}
	}

	// Merge operator-supplied names; append configured-but-unseen sites.
	for _, sys := range s.systems {
		for _, cs := range sys.Sites {
			k := key{sys.Name, cs.RFSS, cs.Site}
			if dto, ok := byKey[k]; ok {
				dto.Name = cs.Name
				continue
			}
			byKey[k] = &SiteDTO{
				System: sys.Name,
				RFSSID: cs.RFSS,
				SiteID: cs.Site,
				Name:   cs.Name,
			}
		}
	}

	out := make([]SiteDTO, 0, len(byKey))
	for _, dto := range byKey {
		out = append(out, *dto)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].System != out[j].System {
			return out[i].System < out[j].System
		}
		if out[i].RFSSID != out[j].RFSSID {
			return out[i].RFSSID < out[j].RFSSID
		}
		return out[i].SiteID < out[j].SiteID
	})
	writeJSON(w, http.StatusOK, map[string]any{"sites": out})
}
