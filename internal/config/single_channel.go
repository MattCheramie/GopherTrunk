package config

import "strings"

// SingleChannelDecoderSerials is the set of SDR serials the single-channel
// decoder sections pin a dongle to: paging (pocsag / flex / wideband), aprs,
// ais, dsc, adsb (native channels), m17, mdc1200, fleetsync and lora. Such a
// dongle is tuned to its decoder's frequency for the daemon's lifetime, so
// no other consumer may retune it (#1316): the trunking voice pool, the
// conventional scanner's spare-SDR heuristics, a live hunt's auto-selection
// and rigctld all exclude these serials. Blank serials are skipped.
func (c Config) SingleChannelDecoderSerials() map[string]bool {
	out := map[string]bool{}
	add := func(serial string) {
		if s := strings.TrimSpace(serial); s != "" {
			out[s] = true
		}
	}
	for _, ch := range c.Paging.POCSAG {
		add(ch.Serial)
	}
	for _, ch := range c.Paging.FLEX {
		add(ch.Serial)
	}
	for _, g := range c.Paging.Wideband {
		add(g.Serial)
	}
	for _, ch := range c.APRS.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.AIS.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.DSC.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.ADSB.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.M17.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.MDC1200.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.FleetSync.Channels {
		add(ch.Serial)
	}
	for _, ch := range c.LoRa.Channels {
		add(ch.Serial)
	}
	return out
}
