package trunking

import "testing"

// TestDiscoveredTalkgroupModeFollowsProtocol pins the #1143 follow-up: a
// SmartNet/SmartZone system carries analog FM voice, but every auto-discovered
// talkgroup was stamped Mode "D" (digital) regardless of protocol, so an
// analog SmartZone talkgroup showed as digital in the Talkgroups UI and CSV
// export. Discovered mode must follow the grant's protocol: analog trunking
// protocols (motorola, ltr, mpt1327, non-ProVoice edacs) are "A", digital
// protocols stay "D", and an EDACS ProVoice grant is "D".
func TestDiscoveredTalkgroupModeFollowsProtocol(t *testing.T) {
	cases := []struct {
		name  string
		grant Grant
		want  string
	}{
		{"motorola is analog", Grant{System: "msp", Protocol: "motorola", GroupID: 46256, SourceID: 700123, FrequencyHz: 857_237_500}, "A"},
		{"ltr is analog", Grant{System: "L", Protocol: "ltr", GroupID: 42, SourceID: 7, FrequencyHz: 451_500_000}, "A"},
		{"mpt1327 is analog", Grant{System: "M", Protocol: "mpt1327", GroupID: 101, SourceID: 8, FrequencyHz: 174_500_000}, "A"},
		{"edacs is analog", Grant{System: "E", Protocol: "edacs", GroupID: 203, SourceID: 9, FrequencyHz: 856_500_000}, "A"},
		{"edacs provoice is digital", Grant{System: "E", Protocol: "edacs", GroupID: 204, SourceID: 9, FrequencyHz: 856_512_500, ProVoice: true}, "D"},
		{"p25 stays digital", Grant{System: "P", Protocol: "p25", GroupID: 305, SourceID: 10, FrequencyHz: 853_450_000}, "D"},
		{"dmr stays digital", Grant{System: "D", Protocol: "dmr-tier3", GroupID: 406, SourceID: 11, FrequencyHz: 442_387_500, Timeslot: 1}, "D"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, _, bus, _ := mkEngine(t, 2)
			defer bus.Close()

			e.HandleGrant(tc.grant)
			tg := e.talkgroups.Lookup(tc.grant.GroupID)
			if tg == nil {
				t.Fatalf("talkgroup %d was not discovered", tc.grant.GroupID)
			}
			if tg.Tag != discoveredTag {
				t.Fatalf("talkgroup %d tag = %q, want %q", tc.grant.GroupID, tg.Tag, discoveredTag)
			}
			if tg.Mode != tc.want {
				t.Errorf("discovered talkgroup %d (protocol %s) Mode = %q, want %q",
					tc.grant.GroupID, tc.grant.Protocol, tg.Mode, tc.want)
			}
		})
	}
}
