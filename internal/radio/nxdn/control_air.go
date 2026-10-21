package nxdn

import "time"

// ingestAirFrame handles one real-air frame for the control channel: a
// descrambled RCCH frame whose CAC passes its CRC refreshes the decode
// heartbeat, locks on the site's identity and publishes voice grants. It
// reports whether the frame's LICH held (the framer's lock flywheel).
func (c *ControlChannel) ingestAirFrame(post []uint8) bool {
	DescrambleAir(post)
	lich := ParseAirLICH(post)
	if !lich.ParityOK {
		return false
	}
	if lich.RFCT != AirRFCtRCCH && lich.RFCT != AirRFCtRTCHC {
		return true
	}
	cac, ok := DecodeAirCAC(post)
	if !ok {
		return true
	}
	now := c.now
	if now == nil {
		now = time.Now
	}
	c.lastActivityNano.Store(now().UnixNano())
	c.framesDecoded.Add(1)
	if c.ran != cac.RAN {
		c.ran = cac.RAN
		c.log.Debug("nxdn: control channel RAN", "system", c.systemName, "ran", cac.RAN)
	}
	switch cac.Type {
	case AirMsgSiteInfo:
		s := ParseAirSiteInfo(cac.Message[:])
		c.lockOnLocation(s.Location)
		if c.siteInfo != s {
			c.siteInfo = s
			cat, sys, site := s.Location.Split()
			c.log.Info("nxdn: site info", "system", c.systemName, "ran", cac.RAN,
				"location_id", uint32(s.Location), "category", locationCategoryName(cat),
				"system_code", sys, "site_code", site,
				"control_channel_1", s.ControlChannel1, "control_channel_2", s.ControlChannel2,
				"adjacent_sites", s.AdjacentSites)
		}
	case AirMsgSrvInfo:
		c.lockOnLocation(ParseAirSrvInfo(cac.Message[:]).Location)
	case AirMsgVCallAssign, AirMsgVCallAssignDup:
		a := ParseAirVCallAssign(cac.Message[:])
		c.publishGrant(VCallAssignPayload{
			ServiceOptions: a.CCOption,
			GroupAddress:   a.Destination,
			SourceID:       a.Source,
			Channel:        a.Channel,
		})
	}
	return true
}

// lockOnLocation records the site identity a SITE_INFO / SRV_INFO carries
// and locks the control channel on it.
func (c *ControlChannel) lockOnLocation(l LocationID) {
	_, sys, site := l.Split()
	c.topo.applySiteInfo(sys, site, uint32(l))
	c.maybeLock(LockState{FrequencyHz: c.freqHz, BaudRate: c.rate, SiteID: site, SystemID: sys})
}

func locationCategoryName(cat uint8) string {
	switch cat {
	case 0:
		return "global"
	case 1:
		return "local"
	case 2:
		return "regional"
	}
	return "reserved"
}
