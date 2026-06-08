package main

import (
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
)

// discoveredToParsed converts a hunt.DiscoveredSystem into the importer's
// parsedSystem so a discovery can be merged into config.yaml through the exact
// same mergeIntoConfig path a PDF/CSV import uses. Keeping the conversion here
// (rather than in internal/hunt) avoids an import cycle: parsedSystem lives in
// package main.
func discoveredToParsed(sys *hunt.DiscoveredSystem) parsedSystem {
	ps := parsedSystem{
		Name:     sys.DisplayName(),
		Protocol: sys.Protocol,
		County:   sys.County,
		Location: sys.Location,
	}
	if sys.SystemID != 0 {
		ps.SysID = fmt.Sprintf("%X", sys.SystemID)
	}
	if sys.WACN != 0 {
		ps.WACN = fmt.Sprintf("%X", sys.WACN)
	}

	for _, st := range sys.Sites {
		name := st.SiteName
		if name == "" {
			name = fmt.Sprintf("Site %d-%d", st.RFSS, st.SiteID)
		}
		site := parsedSite{
			RFSS:     int(st.RFSS),
			SiteID:   int(st.SiteID),
			SiteName: name,
			Cty:      st.County,
			Include:  true,
		}
		for _, ch := range st.ControlChannels {
			site.Frequencies = append(site.Frequencies, parsedFreq{
				Hz:             ch.FrequencyHz,
				ControlChannel: ch.IsControl,
			})
		}
		for _, hz := range st.Secondary {
			site.Frequencies = append(site.Frequencies, parsedFreq{Hz: hz, ControlChannel: true})
		}
		ps.Sites = append(ps.Sites, site)
	}

	for _, tg := range sys.Talkgroups {
		ps.Talkgroups = append(ps.Talkgroups, parsedTalkgroup{
			Dec:       tg.Dec,
			Hex:       tg.Hex,
			Mode:      "D",
			Encrypted: tg.Encrypted,
			Scan:      true,
		})
	}
	return ps
}
