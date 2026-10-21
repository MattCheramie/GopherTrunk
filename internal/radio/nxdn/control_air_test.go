package nxdn_test

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// realControlChannelDibits is the #1261 control-channel slice as the
// production control channel sees it: the receiver's raw dibit stream.
func realControlChannelDibits(t *testing.T) []uint8 {
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
	return dib
}

// TestControlChannelLocksOnRealAirCAC pins #1261's "the trunked control
// channel never locks": a real NXDN96 control channel, fed through
// ControlChannel.Process in the production (ViterbiSpec) mode, must lock on
// the site its SITE_INFO / SRV_INFO announce and count every CAC as decoded.
// The old path searched for an invented 8-dibit FSW and CAC coding that real
// air never carries, and locked on nothing.
func TestControlChannelLocksOnRealAirCAC(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := nxdn.NewControlChannel(bus, nil, 453_775_000, nxdn.Rate9600)
	cc.SetViterbiMode(nxdn.ViterbiSpec)
	dib := realControlChannelDibits(t)
	for i := 0; i < len(dib); i += 500 {
		cc.Process(dib[i:min(i+500, len(dib))], i)
	}

	if n := cc.DecodedFrames(); n < 18 {
		t.Errorf("%d CAC frames decoded, want ≥ 18", n)
	}
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("first event %s, want cc.locked", ev.Kind)
		}
		ls := ev.Payload.(nxdn.LockState)
		if ls.SystemID != 85 || ls.SiteID != 1 || ls.FrequencyHz != 453_775_000 {
			t.Errorf("lock = %+v, want system 85 site 1 at 453.775 MHz", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no cc.locked from a real control channel")
	}
	if topo := cc.Topology(); topo.SystemID != 85 || topo.SiteID != 1 || topo.LocationID != 0x400AA1 {
		t.Errorf("topology = %+v", topo)
	}
}

// encodeAirCAC builds the 150 CAC dibits for a 147-bit message with an
// encoder written from MMDVMHost's CNXDNConvolution::encode and the inverse
// of the 12x25 interleave and 12-of-14 puncture, independent of the decoder.
// The CRC bits are solved for so the decoder's check passes.
func encodeAirCAC(sr byte, msg []byte, crcOK func([]byte) bool) []uint8 {
	bits := make([]byte, 0, 175)
	push := func(v uint32, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, byte(v>>uint(i)&1))
		}
	}
	push(uint32(sr), 8)
	for i := 0; i < 147; i++ {
		bits = append(bits, msg[i/8]>>(7-uint(i%8))&1)
	}
	base := len(bits)
	push(0, 16)
	// The CRC is affine in its 16 bits: solve one bit at a time.
	for j := 0; j < 16; j++ {
		bits[base+j] = 0
	}
	for c := uint32(0); c < 1<<16; c++ {
		for j := 0; j < 16; j++ {
			bits[base+j] = byte(c >> uint(15-j) & 1)
		}
		if crcOK(bits[:171]) {
			break
		}
	}
	push(0, 4) // tail
	var coded []byte
	var d1, d2, d3, d4 byte
	for _, d := range bits {
		coded = append(coded, d^d3^d4, d^d1^d2^d4)
		d4, d3, d2, d1 = d3, d2, d1, d
	}
	keep := [14]bool{true, true, true, false, true, true, true, true, true, true, true, false, true, true}
	var punct []byte
	for i, b := range coded {
		if keep[i%14] {
			punct = append(punct, b)
		}
	}
	air := make([]byte, 300)
	for i := range air {
		air[i] = punct[(i%25)*12+i/25]
	}
	out := make([]uint8, 150)
	for i := range out {
		out[i] = air[2*i]<<1 | air[2*i+1]
	}
	return out
}

// airCRCZero is the CAC check written from the reference formula (DSD-FME
// crc16cac: a 17-bit shift register seeded 0xC3EE, x^16+x^12+x^5+1, final
// XOR 0xFFFF) over a message with its CRC appended: true when the remainder
// is zero.
func airCRCZero(bits []byte) bool {
	reg := uint32(0xC3EE)
	for _, b := range bits {
		reg = (reg<<1 | uint32(b)) & 0x1FFFF
		if reg&0x10000 != 0 {
			reg = reg&0xFFFF ^ (1<<12 | 1<<5 | 1)
		}
	}
	return reg^0xFFFF == 0
}

// TestControlChannelGrantsFromAirVCallAssign: a VCALL_ASSGN on the real air
// format (built into a frame of the real capture, at the reference decoder's
// field positions) publishes a grant at the band plan's frequency. The
// capture has no call on it, so these positions are reference-pinned, not
// yet seen on air.
func TestControlChannelGrantsFromAirVCallAssign(t *testing.T) {
	template := realControlChannelFrames(t)[3]

	// VCALL_ASSGN: CC option 0, call type 1 (group), voice option 0,
	// source 0x1092, destination (talkgroup) 0x0064, call timer 0, channel 7.
	msg := make([]byte, 19)
	put := func(at, n int, v uint32) {
		for i := 0; i < n; i++ {
			if v>>uint(n-1-i)&1 != 0 {
				b := at + i
				msg[b/8] |= 0x80 >> uint(b%8)
			}
		}
	}
	put(2, 6, nxdn.AirMsgVCallAssign)
	put(16, 3, 1)
	put(24, 16, 0x1092)
	put(40, 16, 0x0064)
	put(62, 10, 7)
	cacDibits := encodeAirCAC(0x01, msg, airCRCZero)
	post := append([]uint8(nil), template...)
	copy(post[nxdn.AirLICHDibits:], cacDibits)
	got, ok := nxdn.DecodeAirCAC(post)
	if !ok {
		t.Fatal("the encoded VCALL_ASSGN failed its own CAC CRC")
	}
	a := nxdn.ParseAirVCallAssign(got.Message[:])
	if got.Type != nxdn.AirMsgVCallAssign || a.CallType != 1 || a.Source != 0x1092 || a.Destination != 0x0064 || a.Channel != 7 {
		t.Fatalf("decoded %+v type %#x", a, got.Type)
	}

	// The same frame, scrambled and framed, through the control channel.
	nxdn.DescrambleAir(post)
	var stream []uint8
	for i := 0; i < 3; i++ {
		stream = append(stream, nxdn.AirFSW[:]...)
		stream = append(stream, post...)
	}
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := nxdn.NewControlChannel(bus, nil, 453_775_000, nxdn.Rate9600)
	cc.SetViterbiMode(nxdn.ViterbiSpec)
	cc.SetSystemName("Metro")
	cc.SetBandPlan(nxdn.LinearBandPlan{BaseHz: 453_700_000, SpacingHz: 12_500, Offset: 1})
	cc.Process(stream, 0)
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindGrant {
			t.Fatalf("event %s, want grant", ev.Kind)
		}
		g := ev.Payload.(trunking.Grant)
		if g.GroupID != 0x64 || g.SourceID != 0x1092 || g.ChannelNum != 7 || g.FrequencyHz != 453_775_000 {
			t.Errorf("grant = group %d src %#x channel %d freq %d", g.GroupID, g.SourceID, g.ChannelNum, g.FrequencyHz)
		}
	case <-time.After(time.Second):
		t.Fatal("no grant from a real-air VCALL_ASSGN")
	}
}

// TestEncodeAirControlFrameMatchesIndependentEncoder: the production frame
// encoder (used by the Signal Lab synthesiser) writes the same CAC dibits as
// the test's MMDVMHost-derived encoder, and decodes back to its message.
func TestEncodeAirControlFrameMatchesIndependentEncoder(t *testing.T) {
	msg := make([]byte, 19)
	copy(msg, []byte{0x18, 0x40, 0x0a, 0xa1, 0x49, 0x25, 0xbb, 0xc0, 0, 0, 0, 0, 0, 0, 0x01, 0x20, 0x04})
	frame := nxdn.EncodeAirControlFrame(1, msg)
	if len(frame) != nxdn.AirFrameDibits {
		t.Fatalf("frame is %d dibits, want %d", len(frame), nxdn.AirFrameDibits)
	}
	post := append([]uint8(nil), frame[nxdn.AirFSWDibits:]...)
	nxdn.DescrambleAir(post)
	want := encodeAirCAC(0x01, msg, airCRCZero)
	for i, d := range want {
		if post[nxdn.AirLICHDibits+i] != d {
			t.Fatalf("CAC dibit %d = %d, independent encoder %d", i, post[nxdn.AirLICHDibits+i], d)
		}
	}
	if l := nxdn.ParseAirLICH(post); !l.ParityOK || l.RFCT != nxdn.AirRFCtRCCH || l.Direction != 1 {
		t.Errorf("LICH %+v, want an outbound RCCH LICH with good parity", l)
	}
	cac, ok := nxdn.DecodeAirCAC(post)
	if !ok || cac.RAN != 1 || cac.Message != [19]byte(msg) {
		t.Errorf("decoded ok=%v ran=%d message %x", ok, cac.RAN, cac.Message)
	}
}
