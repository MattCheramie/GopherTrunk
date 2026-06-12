package tier3

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestControlChannelPublishesBTVGrant proves a Broadcast TalkGroup Voice
// grant (CSBKO 0x32) is followed as a voice call — it keys up voice, unlike
// the data grants below.
func TestControlChannelPublishesBTVGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Resolver:    TableBandPlan{8: 866_175_000},
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	// LPCN 8 (p[1]=0x88 → TS2), group 0x123456, source 0xABCDEF — same
	// channel-grant layout as TV_GRANT.
	csbk := CSBK{LB: true, Opcode: OpBTVGrant, Payload: [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF}}
	cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 3, DataType: dmr.DTCSBK})

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindGrant {
				continue
			}
			g := ev.Payload.(trunking.Grant)
			if g.GroupID != 0x123456 || g.SourceID != 0xABCDEF {
				t.Errorf("group/source = %X/%X", g.GroupID, g.SourceID)
			}
			if g.FrequencyHz != 866_175_000 || g.Timeslot != 2 {
				t.Errorf("freq/ts = %d/%d, want 866_175_000/2", g.FrequencyHz, g.Timeslot)
			}
			return
		case <-deadline:
			t.Fatal("no voice grant published for BTV_GRANT")
		}
	}
}

// TestControlChannelDataGrantObservedNotFollowed is load-bearing: a data
// channel grant (PD_GRANT / TD_GRANT) must publish KindDMRGrantObserved (so
// the LCN learner still sees the carrier) but must NOT publish KindGrant —
// the engine would otherwise retune a voice SDR onto a packet channel.
func TestControlChannelDataGrantObservedNotFollowed(t *testing.T) {
	cases := []struct {
		name   string
		opcode CSBKOpcode
	}{
		{"td_grant", OpTDGrant},
		{"pd_grant", OpPDGrant},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(16)
			defer bus.Close()
			sub := bus.Subscribe()
			defer sub.Close()

			cc := New(Options{
				Bus:         bus,
				SystemName:  "DataSys",
				FrequencyHz: 851_012_500,
				// A resolver IS set: this proves the data grant is withheld
				// by policy, not merely dropped for want of a band plan.
				Resolver: TableBandPlan{8: 866_175_000},
				Now:      func() time.Time { return time.Unix(1_700_000_002, 0).UTC() },
			})
			// LPCN 8 (p[1]=0x88 → TS2), target 0x123456, source 0xABCDEF.
			csbk := CSBK{LB: true, Opcode: tc.opcode, Payload: [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF}}
			cc.IngestBurst(burstWithCSBK(csbk), dmr.SlotType{ColorCode: 4, DataType: dmr.DTCSBK})

			obs, sawObs, sawGrant, _ := collectGrantObserved(t, sub)
			if !sawObs {
				t.Fatal("no KindDMRGrantObserved for data grant")
			}
			if sawGrant {
				t.Error("KindGrant published for a data grant — would retune a voice device")
			}
			if obs.LCN != 8 || obs.Timeslot != 1 || obs.GroupID != 0x123456 || obs.SourceID != 0xABCDEF {
				t.Errorf("obs = %+v", obs)
			}
		})
	}
}

func TestParseDataGrant(t *testing.T) {
	// LPCN 8, TS2, target 0x123456, source 0xABCDEF.
	p := [8]byte{0x00, 0x88, 0x12, 0x34, 0x56, 0xAB, 0xCD, 0xEF}
	td := ParseDataGrant(p, false)
	if td.LCN != 8 || td.Timeslot != 1 || td.TargetID != 0x123456 || td.SourceID != 0xABCDEF || td.Private {
		t.Errorf("TD = %+v", td)
	}
	pd := ParseDataGrant(p, true)
	if !pd.Private {
		t.Error("Private flag not set for PD_GRANT")
	}
}
