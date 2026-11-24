package siglab

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The Phase 2 FEC tally takes the descramble seed from the capture's own
// clear-LCCH Network Status Broadcast, as the control channel does (issue
// #1364). It used to build the seed from the system's Site, which is not the
// 12-bit Color Code and which siglab never sets, so every scrambled burst of
// a real capture was counted uncorrectable.
func TestP25P2FECTakesSeedFromCapturesNetworkStatus(t *testing.T) {
	const wacn, sysID, nac = 0xABCDE, 0x123, 0x293
	masks := p25phase2.NewAirMasks(framing.PN44SeedFromIdentity(wacn, sysID, nac))
	nsb := []byte{0xFB, 0x07, 0xAB, 0xCD, 0xE1, 0x23, 0x10, 0x05, 0x10, 0x06, 0x70, 0x02, 0x93}
	grant := []byte{0x40, 0x00, 0x10, 0x05, 0x4E, 0xEA, 0x04, 0xCF, 0x43}
	lcch, err := p25phase2.AirMACPDUBytes(p25phase2.AirMACSignal, 0, 21, nsb)
	if err != nil {
		t.Fatal(err)
	}
	sacch, err := p25phase2.AirMACPDUBytes(p25phase2.AirMACSignal, 0, 21, grant)
	if err != nil {
		t.Fatal(err)
	}
	stream := make([]uint8, 600)
	for i := range stream {
		stream[i] = uint8(i % 4)
	}
	const superframes = 6
	for s := 0; s <= superframes; s++ {
		var sf [p25phase2.AirSlotsPerSuperframe]p25phase2.AirSlotContent
		for slot := range sf {
			sf[slot] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDSACCHScrambled, MAC: sacch}
		}
		sf[0] = p25phase2.AirSlotContent{DUID: p25phase2.DUIDLCCHClear, MAC: lcch}
		d, err := p25phase2.EncodeAirSuperframe(sf, masks, uint8(s&3))
		if err != nil {
			t.Fatal(err)
		}
		stream = append(stream, d...)
	}
	stats := p25p2FEC(stream, nil, trunking.System{Name: "siglab", Protocol: trunking.ProtocolP25Phase2})
	var mac FECStat
	for _, st := range stats {
		if st.Stage == "MAC RS(63,35)+CRC" {
			mac = st
		}
	}
	if mac.CRCPass+mac.CRCFail == 0 {
		t.Fatal("no signalling bursts tallied")
	}
	if mac.CRCFail != 0 {
		t.Fatalf("MAC tally %d pass / %d fail: the scrambled bursts did not descramble with the capture's own seed", mac.CRCPass, mac.CRCFail)
	}
}
