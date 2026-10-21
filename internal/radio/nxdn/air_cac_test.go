package nxdn_test

import (
	"fmt"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// realControlChannelFrames runs the #1261 control-channel slice through the
// voice-chain receiver configuration and returns each framed, descrambled
// post-FSW frame.
func realControlChannelFrames(t *testing.T) [][]uint8 {
	t.Helper()
	iq := readCS16(t, "testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16")
	var dib []uint8
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz:          48000,
		SymbolRateHz:          nxdnrx.SymbolRate,
		DeviationHz:           1800,
		EnableAFC:             true,
		EnableCarrierTracking: true,
		EnableChannelFilter:   true,
		DibitSink:             func(d []uint8, _ int) { dib = append(dib, d...) },
	})
	for i := 0; i < len(iq); i += 4096 {
		rx.Process(iq[i:min(i+4096, len(iq))])
	}
	var frames [][]uint8
	for k := 0; k+nxdn.AirFrameDibits <= len(dib); k++ {
		errs := 0
		for j := range nxdn.AirFSW {
			if dib[k+j] != nxdn.AirFSW[j] {
				errs++
			}
		}
		if errs > 1 {
			continue
		}
		post := append([]uint8(nil), dib[k+nxdn.AirFSWDibits:k+nxdn.AirFrameDibits]...)
		nxdn.DescrambleAir(post)
		frames = append(frames, post)
	}
	return frames
}

// TestDecodeAirCACRealControlChannel pins the real-air CAC decode on the
// reporter's control channel: every frame passes the CAC CRC, the SR names
// RAN 1, and the channel broadcasts SRV_INFO and SITE_INFO. The two literal
// messages below are what the capture carries (the reference decoder's field
// positions read them as location 0x400AA1, and SITE_INFO names control
// channel 1 and two adjacent sites, as the reporter describes the system).
// Two independent fields agree:
// the location is category Local, system 85, site 1, and the RAN every CAC's
// SR carries is that site code mod 64. The old CAC model (invented FSW, LICH
// coding and 88-bit message) frames none of these.
func TestDecodeAirCACRealControlChannel(t *testing.T) {
	frames := realControlChannelFrames(t)
	if len(frames) < 18 {
		t.Fatalf("%d frames framed, want ≥ 18", len(frames))
	}
	want := map[string]bool{
		// SRV_INFO: location 400AA1, service BBC0, restriction 000000.
		"19400aa1bbc000000000000000000000000000": true,
		// SITE_INFO: location 400AA1, channel structure 4925, service BBC0.
		"18400aa14925bbc00000000000000120040000": true,
	}
	types := map[uint8]int{}
	for i, post := range frames {
		cac, ok := nxdn.DecodeAirCAC(post)
		if !ok {
			t.Errorf("frame %d: CAC CRC failed", i)
			continue
		}
		if cac.RAN != 1 {
			t.Errorf("frame %d: RAN %d, want 1", i, cac.RAN)
		}
		if m := fmt.Sprintf("%x", cac.Message); !want[m] {
			t.Errorf("frame %d: message %s is neither of the capture's two", i, m)
		}
		types[cac.Type]++
		if cac.Type == nxdn.AirMsgSiteInfo {
			s := nxdn.ParseAirSiteInfo(cac.Message[:])
			cat, sys, site := s.Location.Split()
			if s.Location != 0x400AA1 || cat != 1 || sys != 85 || site != 1 {
				t.Errorf("SITE_INFO location %06X = category %d system %d site %d, want 400AA1 = local 85 / 1", uint32(s.Location), cat, sys, site)
			}
			// The reporter's system: two microwave-linked sites, channel 1
			// one of the control channels.
			if s.ControlChannel1 != 1 || s.AdjacentSites != 2 {
				t.Errorf("SITE_INFO control channel 1 = %d, adjacent sites = %d; want 1 and 2", s.ControlChannel1, s.AdjacentSites)
			}
			if uint8(site%64) != cac.RAN {
				t.Errorf("site code %d mod 64 != RAN %d", site, cac.RAN)
			}
		}
	}
	if types[nxdn.AirMsgSrvInfo] == 0 || types[nxdn.AirMsgSiteInfo] == 0 {
		t.Errorf("message types %v, want both SRV_INFO (0x19) and SITE_INFO (0x18)", types)
	}
	t.Logf("frames=%d types=%v", len(frames), types)
}

// TestDecodeAirCACRejectsCorruption: a CAC with its channel bits damaged
// beyond the code's reach fails the CRC instead of decoding garbage.
func TestDecodeAirCACRejectsCorruption(t *testing.T) {
	frames := realControlChannelFrames(t)
	post := append([]uint8(nil), frames[len(frames)/2]...)
	for i := nxdn.AirLICHDibits; i < nxdn.AirLICHDibits+40; i++ {
		post[i] ^= 3
	}
	if _, ok := nxdn.DecodeAirCAC(post); ok {
		t.Fatal("a CAC with 40 inverted dibits decoded")
	}
}
