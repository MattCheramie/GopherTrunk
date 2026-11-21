package config

import "github.com/MattCheramie/GopherTrunk/internal/trunking"

// EffectiveControlChannels returns the control channels the daemon builds
// this system with. A non-empty control_channels list is used verbatim.
// Conventional DMR (Tier II / Tier I) has no real control channel, so an
// empty list there is filled from the role: wideband devices' channels
// assigned to the system (WidebandChannelsForSystem): the operator lists
// each repeater or simplex carrier once, as a wideband channel. Every
// other protocol keeps an empty list, which the daemon refuses.
//
// The daemon and config validation both call this, so `config validate`
// accepts exactly the systems the daemon can start.
func (s SystemConfig) EffectiveControlChannels(devices []DeviceConfig) []uint32 {
	if len(s.ControlChannels) > 0 {
		return s.ControlChannels
	}
	proto, err := trunking.ParseProtocol(s.Protocol)
	if err != nil {
		return nil
	}
	if proto != trunking.ProtocolDMRTier2 && proto != trunking.ProtocolDMRTier1 {
		return nil
	}
	return WidebandChannelsForSystem(devices, s.Name)
}

// WidebandChannelsForSystem returns the de-duplicated, config-order union
// of the frequencies on every role: wideband device's channels list that
// name the given system.
func WidebandChannelsForSystem(devices []DeviceConfig, system string) []uint32 {
	var out []uint32
	seen := make(map[uint32]bool)
	for _, dev := range devices {
		if dev.Role != "wideband" {
			continue
		}
		for _, ch := range dev.Channels {
			if ch.System != system || seen[ch.FrequencyHz] {
				continue
			}
			seen[ch.FrequencyHz] = true
			out = append(out, ch.FrequencyHz)
		}
	}
	return out
}
