package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestServiceOptionsBits(t *testing.T) {
	for _, c := range []struct {
		so   ServiceOptions
		emer bool
		enc  bool
		prio uint8
	}{
		{0x00, false, false, 0},
		{0x80, true, false, 0},
		{0x40, false, true, 0},
		{0xC5, true, true, 5},
	} {
		if c.so.Emergency() != c.emer || c.so.Encrypted() != c.enc || c.so.Priority() != c.prio {
			t.Errorf("ServiceOptions(%#x) = emer %v / enc %v / prio %d", uint8(c.so),
				c.so.Emergency(), c.so.Encrypted(), c.so.Priority())
		}
	}
}

func TestParseUnitToUnitVoiceChannelGrant(t *testing.T) {
	in := UnitToUnitVoiceChannelGrant{ChannelID: 2, ChannelNumber: 0x123, TargetID: 0x00BEEF, SourceID: 0x00ABCD}
	if got := ParseUnitToUnitVoiceChannelGrant(AssembleUnitToUnitVoiceChannelGrant(in)); got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestParseTelephoneInterconnectGrant(t *testing.T) {
	in := TelephoneInterconnectGrant{ServiceOptions: 0x80, ChannelID: 1, ChannelNumber: 0x05, CallTimer: 600, TargetID: 0x00ABCD}
	if got := ParseTelephoneInterconnectGrant(AssembleTelephoneInterconnectGrant(in)); got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestParseSNDCPDataChannelGrant(t *testing.T) {
	in := SNDCPDataChannelGrant{ServiceOptions: 0x10, ChannelID: 3, ChannelNumber: 0x044, TargetID: 0x00BEEF}
	if got := ParseSNDCPDataChannelGrant(AssembleSNDCPDataChannelGrant(in)); got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

// drainGrants collects every KindGrant published within a short window.
func drainGrants(t *testing.T, sub *events.Subscription) []trunking.Grant {
	t.Helper()
	var out []trunking.Grant
	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				out = append(out, ev.Payload.(trunking.Grant))
			}
		case <-deadline:
			return out
		default:
			return out
		}
	}
}

// TestControlChannelDispatchesNewGrantOpcodes feeds an IdentifierUpdate
// followed by each newly-wired grant opcode and asserts a resolved
// trunking.Grant lands on the bus.
func TestControlChannelDispatchesNewGrantOpcodes(t *testing.T) {
	const nac = 0x293
	ident := TSBK{Opcode: OpIdentifierUpdate}
	ident.Payload = AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
	})
	// channel ID 1, number 16 → 851_000_000 + 16*12_500 = 851_200_000.
	const wantFreq = 851_200_000

	cases := []struct {
		name     string
		tsbk     TSBK
		group    uint32
		source   uint32
		dataCall bool
	}{
		{
			name: "GroupVoiceChannelUpdate",
			tsbk: TSBK{Opcode: OpGroupVoiceChannelUpdate, Payload: AssembleGroupVoiceChannelUpdate(
				GroupVoiceChannelUpdate{ChannelAID: 1, ChannelANumber: 16, GroupAddressA: 0x1234})},
			group: 0x1234,
		},
		{
			name: "UnitToUnitVoiceChannelGrant",
			tsbk: TSBK{Opcode: OpUnitToUnitVoiceChannelGrant, Payload: AssembleUnitToUnitVoiceChannelGrant(
				UnitToUnitVoiceChannelGrant{ChannelID: 1, ChannelNumber: 16, TargetID: 0x00BEEF, SourceID: 0x00ABCD})},
			group: 0x00BEEF, source: 0x00ABCD,
		},
		{
			name: "TelephoneInterconnectGrant",
			tsbk: TSBK{Opcode: OpTelephoneInterconnectGrant, Payload: AssembleTelephoneInterconnectGrant(
				TelephoneInterconnectGrant{ChannelID: 1, ChannelNumber: 16, TargetID: 0x00ABCD})},
			group: 0x00ABCD,
		},
		{
			name: "SNDCPDataChannelGrant",
			tsbk: TSBK{Opcode: OpSNDCPDataChannelGrant, Payload: AssembleSNDCPDataChannelGrant(
				SNDCPDataChannelGrant{ChannelID: 1, ChannelNumber: 16, TargetID: 0x00BEEF})},
			group: 0x00BEEF, dataCall: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(16)
			defer bus.Close()
			sub := bus.Subscribe()
			defer sub.Close()

			cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 851_000_000})
			cc.Process(buildLockedStreamWithTSBK(10, nac, DUIDTrunkingSignaling, ident), 0)
			cc.Process(buildLockedStreamWithTSBK(0, nac, DUIDTrunkingSignaling, tc.tsbk), 1<<20)

			grants := drainGrants(t, sub)
			if len(grants) != 1 {
				t.Fatalf("got %d grants, want 1", len(grants))
			}
			g := grants[0]
			if g.GroupID != tc.group || g.SourceID != tc.source {
				t.Errorf("group/source = %d/%d, want %d/%d", g.GroupID, g.SourceID, tc.group, tc.source)
			}
			if g.FrequencyHz != wantFreq {
				t.Errorf("freq = %d, want %d", g.FrequencyHz, wantFreq)
			}
			if g.DataCall != tc.dataCall {
				t.Errorf("DataCall = %v, want %v", g.DataCall, tc.dataCall)
			}
		})
	}
}
