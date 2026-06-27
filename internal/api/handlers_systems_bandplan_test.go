package api

import (
	"encoding/json"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestGetSystemBandPlan checks that the live topology's decoded P25 IDEN_UP
// frequency-band table is overlaid onto SystemDTO.FrequencyBands for both the
// detail and list endpoints, sorted by channel ID and carrying the TDMA flag
// and transmit offset (issue #814).
func TestGetSystemBandPlan(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()

	snap := &trunking.TopologySnapshot{
		WACN:     0xBEE00,
		SystemID: 0x2C2,
		RFSS:     1,
		Site:     1,
		BandPlan: []trunking.TopoBandPlanSlot{
			// Band 1 first in the slice; expect it sorted after band 0.
			{ChannelID: 1, BaseHz: 421_262_500, SpacingHz: 6_250, BandwidthHz: 12_500, TxOffsetHz: 5_500_000, AccessTDMA: true},
			{ChannelID: 0, BaseHz: 418_100_000, SpacingHz: 6_250, BandwidthHz: 12_500, TxOffsetHz: -9_450_000, AccessTDMA: true},
		},
	}
	prov := fakeTopoProvider{system: "MMR", snap: snap}
	systems := []trunking.System{{Name: "MMR"}}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Systems: systems, Sites: prov})
	defer teardown()

	// Detail endpoint.
	resp := mustGet(t, base+"/api/v1/systems/MMR")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dto SystemDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(dto.FrequencyBands) != 2 {
		t.Fatalf("FrequencyBands = %d, want 2: %+v", len(dto.FrequencyBands), dto.FrequencyBands)
	}
	// Sorted by channel ID: band 0 before band 1.
	if dto.FrequencyBands[0].ChannelID != 0 || dto.FrequencyBands[1].ChannelID != 1 {
		t.Errorf("bands not sorted by channel id: %+v", dto.FrequencyBands)
	}
	b := dto.FrequencyBands[0]
	if b.BaseHz != 418_100_000 {
		t.Errorf("base = %d, want 418100000", b.BaseHz)
	}
	if b.SpacingHz != 6_250 {
		t.Errorf("spacing = %d, want 6250", b.SpacingHz)
	}
	if b.BandwidthHz != 12_500 {
		t.Errorf("bandwidth = %d, want 12500", b.BandwidthHz)
	}
	if b.TxOffsetHz != -9_450_000 {
		t.Errorf("tx offset = %d, want -9450000", b.TxOffsetHz)
	}
	if !b.AccessTDMA {
		t.Errorf("access_tdma = false, want true")
	}

	// List endpoint surfaces the same band plan.
	listResp := mustGet(t, base+"/api/v1/systems")
	defer listResp.Body.Close()
	var list struct {
		Systems []SystemDTO `json:"systems"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Systems) != 1 || len(list.Systems[0].FrequencyBands) != 2 {
		t.Fatalf("list frequency bands missing: %+v", list.Systems)
	}
}
