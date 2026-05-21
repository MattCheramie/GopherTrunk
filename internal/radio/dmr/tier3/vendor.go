package tier3

// DMR feature-set IDs (FID) — the second octet of a CSBK info block.
// ETSI TS 102 361-4 reserves FID 0x00 for standard trunking; the major
// MotoTRBO and Hytera trunking products tag their proprietary CSBKs
// with a vendor FID. Dispatching on FID before opcode matters: a
// vendor CSBK whose 6-bit opcode happens to collide with a standard
// opcode (e.g. 0x30) would otherwise be misdecoded as a standard
// TalkGroup Voice Channel Grant and produce a bogus trunking.Grant.
const (
	FIDStandard    uint8 = 0x00 // ETSI TS 102 361-4
	FIDMotorola    uint8 = 0x10 // MotoTRBO: Capacity Plus / Capacity Max
	FIDConnectPlus uint8 = 0x06 // Motorola Connect Plus
	FIDHytera      uint8 = 0x08 // Hytera XPT
	FIDHyteraAlt   uint8 = 0x68 // Hytera (alternate feature-set tag)
)

// Vendor identifies the trunking feature set behind a CSBK's FID.
type Vendor uint8

const (
	VendorStandard Vendor = iota
	VendorMotorola
	VendorConnectPlus
	VendorHytera
)

func (v Vendor) String() string {
	switch v {
	case VendorMotorola:
		return "Motorola Capacity Plus/Max"
	case VendorConnectPlus:
		return "Motorola Connect Plus"
	case VendorHytera:
		return "Hytera XPT"
	default:
		return "ETSI standard"
	}
}

// VendorFromFID maps a CSBK FID octet to its trunking vendor.
func VendorFromFID(fid uint8) Vendor {
	switch fid {
	case FIDMotorola:
		return VendorMotorola
	case FIDConnectPlus:
		return VendorConnectPlus
	case FIDHytera, FIDHyteraAlt:
		return VendorHytera
	default:
		return VendorStandard
	}
}

// handleVendorCSBK dispatches a CSBK that carries a recognised vendor
// FID. Capacity Plus / Capacity Max (Motorola FID 0x10) carry voice
// grants in the ETSI-shaped 8-octet payload, so those decode through
// the standard TVGrant / PVGrant parsers; the Capacity Plus rest
// channel is tracked from its system-info CSBK. Connect Plus and
// Hytera XPT use materially different proprietary signaling — their
// CSBKs are recognised and logged here but NOT force-parsed as
// standard grants, which would emit garbage. On-air capture
// validation of the Connect Plus / XPT payload layouts is the
// remaining follow-up.
func (c *ControlChannel) handleVendorCSBK(vendor Vendor, cc uint8, csbk CSBK) {
	switch vendor {
	case VendorMotorola:
		switch csbk.Opcode {
		case OpTVGrant:
			c.publishVendorTVGrant(vendor, cc, ParseTVGrant(csbk.Payload))
		case OpPVGrant:
			c.publishVendorPVGrant(vendor, cc, ParsePVGrant(csbk.Payload))
		case OpSysInfo, OpAloha:
			// Capacity Plus advertises its current rest channel in
			// the system-info CSBK; the LCN field doubles as the
			// rest-channel pointer.
			rest := ParseSystemInfoBroadcast(csbk.Payload)
			c.restChannel = rest.SiteID // LCN of the rest channel
			c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: cc, SystemID: rest.SystemID})
		default:
			c.log.Debug("dmr/tier3: motorola vendor csbk",
				"opcode", csbk.Opcode, "cc", cc)
		}
	case VendorConnectPlus, VendorHytera:
		c.log.Debug("dmr/tier3: vendor csbk recognised (payload decode pending capture validation)",
			"vendor", vendor.String(), "opcode", csbk.Opcode, "fid", csbk.FID, "cc", cc)
	}
}

// publishVendorTVGrant emits a voice grant decoded from a vendor CSBK.
// The Protocol stays "dmr-tier3" — vendor trunking changes the control
// layer, not the voice codec, so the engine / recorder / vocoder paths
// are unaffected.
func (c *ControlChannel) publishVendorTVGrant(vendor Vendor, cc uint8, g TVGrant) {
	freq, ok := c.publishGrant(cc, g.LCN, g.GroupAddress, g.SourceID, g.ServiceOptions)
	if !ok {
		return
	}
	c.log.Debug("dmr/tier3: vendor tv-grant",
		"vendor", vendor.String(), "system", c.systemName, "cc", cc,
		"tg", g.GroupAddress, "src", g.SourceID, "lcn", g.LCN, "freq_hz", freq)
}

// publishVendorPVGrant emits a private voice grant from a vendor CSBK.
func (c *ControlChannel) publishVendorPVGrant(vendor Vendor, cc uint8, g PVGrant) {
	freq, ok := c.publishGrant(cc, g.LCN, g.DestinationID, g.SourceID, g.ServiceOptions)
	if !ok {
		return
	}
	c.log.Debug("dmr/tier3: vendor pv-grant",
		"vendor", vendor.String(), "system", c.systemName, "cc", cc,
		"dst", g.DestinationID, "src", g.SourceID, "lcn", g.LCN, "freq_hz", freq)
}
