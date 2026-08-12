package configbuilder

import (
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/radioreference"
)

// RRToSystemConfig projects a RadioReference FullSystem into a config-ready
// SystemConfig (control channels collapsed + de-duplicated across sites) plus
// the talkgroup rows for its CSV sidecar. Shared by the web builder's
// /rr/system handler and the TUI's RadioReference import.
func RRToSystemConfig(full radioreference.FullSystem) (config.SystemConfig, []TalkgroupCSVRow) {
	var ccs []uint32
	seen := make(map[uint32]struct{})
	for _, site := range full.Sites {
		for _, hz := range site.ControlChannels {
			if _, dup := seen[hz]; dup {
				continue
			}
			seen[hz] = struct{}{}
			ccs = append(ccs, hz)
		}
	}
	rows := make([]TalkgroupCSVRow, 0, len(full.Talkgroups))
	for _, tg := range full.Talkgroups {
		rows = append(rows, TalkgroupCSVRow{
			Decimal:     tg.Dec,
			AlphaTag:    tg.AlphaTag,
			Description: tg.Description,
			Tag:         tg.Tag,
			Group:       tg.Group,
			Mode:        tg.Mode,
		})
	}
	// Carry each site's name + position into SiteConfig so the imported system
	// reads sites as named places and the console can map the ones RR
	// geolocated. A site needs a usable RFSS/Site and a name (its RR
	// description) to be worth an entry; positionless sites still get a name.
	var sites []config.SiteConfig
	for _, site := range full.Sites {
		if site.Description == "" {
			continue
		}
		sites = append(sites, config.SiteConfig{
			RFSS:      uint8(site.RFSS),
			Site:      uint8(site.SiteNumber),
			Name:      site.Description,
			Latitude:  site.Latitude,
			Longitude: site.Longitude,
		})
	}
	return config.SystemConfig{
		Name:            full.Name,
		Protocol:        full.Protocol,
		ControlChannels: ccs,
		Sites:           sites,
	}, rows
}
