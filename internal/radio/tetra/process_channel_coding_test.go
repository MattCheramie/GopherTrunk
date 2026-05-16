package tetra

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// pduToType1Bits converts a PDU (header byte + payload bytes)
// into a bit slice of exactly k1 bits, MSB-first per byte,
// zero-padded if the PDU is shorter than k1. Returns nil if
// the PDU is too long to fit in k1 bits.
func pduToType1Bits(pdu PDU, k1 int) []byte {
	bytes := AssemblePDU(pdu)
	if len(bytes)*8 > k1 {
		return nil
	}
	out := make([]byte, k1)
	for i, b := range bytes {
		for j := 0; j < 8; j++ {
			out[i*8+j] = (b >> uint(7-j)) & 1
		}
	}
	return out
}

// TestProcessChannelCodingOnSCHHDRoundTrip: encode a known PDU
// through EncodeSCHHD, build a wire-bit stream (padding + sync +
// 108 dibits of channel-coded SCH/HD), push it through
// Process(ChannelCodingOn / SCHHD), confirm the state machine
// publishes the expected event.
func TestProcessChannelCodingOnSCHHDRoundTrip(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 412_062_500,
	})
	cc.SetChannelCoding(ChannelCodingOn)
	cc.SetExpectedChannel(ChannelSCHHD)
	cc.SetColourCode(0x12345)

	// MLE SYSINFO — drives a cc.locked event.
	pdu := PDU{
		Disc: DiscMLE,
		Type: uint8(MLESystemInfo),
		Payload: []byte{
			// MCC=10 bits, MNC=14 bits, LA=14 bits — 38 bits across 5 bytes
			// (rest zero — minimum valid).
			0x00, 0x40, 0x00, 0x00, 0x00,
		},
	}
	info := pduToType1Bits(pdu, 124)
	if info == nil {
		t.Fatalf("PDU too large for SCH/HD 124 type-1 bits")
	}
	type5 := EncodeSCHHD(info, 0x12345)
	if len(type5) != 216 {
		t.Fatalf("EncodeSCHHD produced %d bits, want 216", len(type5))
	}
	dibits := framing.BitsToDibits(type5)
	if len(dibits) != 108 {
		t.Fatalf("type-5 → dibits = %d, want 108", len(dibits))
	}

	// Stream: 30 padding dibits + 38-dibit normal training sequence
	// + 108 channel-coded SCH/HD dibits.
	stream := make([]uint8, 30)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, dibits...)

	cc.Process(stream, 0)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("ChannelCodingOn Process did not publish KindCCLocked for a MLE SYSINFO encoded via SCH/HD")
			}
			return
		}
	}
}

// TestProcessChannelCodingOnSCHFRoundTrip: same idea but with a
// full-slot signaling channel (432 type-5 bits / 216 dibits) and
// a CMCE D-CONNECT (voice grant) PDU.
func TestProcessChannelCodingOnSCHFRoundTrip(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 412_062_500,
	})
	cc.SetChannelCoding(ChannelCodingOn)
	cc.SetExpectedChannel(ChannelSCHF)
	cc.SetColourCode(0xAAAA)

	// CMCE D-CONNECT carrying a VoiceGrant for talkgroup 0xCAFE,
	// source 0xAAAAAA, dest 0xBBBBBB, carrier 7, group flag set.
	payload := make([]byte, 11)
	payload[0], payload[1] = 0x00, 0x04 // CID = 1
	payload[2], payload[3], payload[4] = 0xAA, 0xAA, 0xAA
	payload[5], payload[6], payload[7] = 0xBB, 0xBB, 0xBB
	payload[8] = 0x80                    // group flag
	payload[9], payload[10] = 0x00, 0x70 // carrier 7
	pdu := PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect), Payload: payload}
	info := pduToType1Bits(pdu, 268)
	if info == nil {
		t.Fatalf("PDU too large for SCH/F 268 type-1 bits")
	}
	type5 := EncodeSCHF(info, 0xAAAA)
	dibits := framing.BitsToDibits(type5)
	if len(dibits) != 216 {
		t.Fatalf("type-5 → dibits = %d, want 216", len(dibits))
	}

	stream := make([]uint8, 30)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, dibits...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("ChannelCodingOn Process did not publish KindGrant for a CMCE D-CONNECT encoded via SCH/F")
			}
			return
		}
	}
}

// TestProcessChannelCodingOnRejectsCRCFailure: corrupt the
// channel-coded dibits enough to overwhelm the inner Viterbi
// (30 adjacent bit flips), confirm Process drops the frame
// (no Grant / cc.locked).
func TestProcessChannelCodingOnRejectsCRCFailure(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:        bus,
		Log:        slog.Default(),
		SystemName: "Sys",
	})
	cc.SetChannelCoding(ChannelCodingOn)
	cc.SetExpectedChannel(ChannelSCHHD)
	cc.SetColourCode(0x12345)

	pdu := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: []byte{0x00, 0x40, 0x00, 0x00, 0x00},
	}
	info := pduToType1Bits(pdu, 124)
	type5 := EncodeSCHHD(info, 0x12345)
	// Heavy corruption: 30 adjacent bit flips
	for i := 50; i < 80; i++ {
		type5[i] ^= 1
	}
	dibits := framing.BitsToDibits(type5)

	stream := make([]uint8, 30)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, dibits...)

	cc.Process(stream, 0)

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked || ev.Kind == events.KindGrant {
				t.Errorf("ChannelCodingOn accepted a heavily-corrupted frame: %v", ev.Kind)
			}
		default:
			return
		}
	}
}

// TestProcessChannelCodingOnRejectsWrongColourCode: encode with
// colour code A, configure decoder for colour code B — descramble
// produces garbage, CRC fails, no event.
func TestProcessChannelCodingOnRejectsWrongColourCode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:        bus,
		Log:        slog.Default(),
		SystemName: "Sys",
	})
	cc.SetChannelCoding(ChannelCodingOn)
	cc.SetExpectedChannel(ChannelSCHHD)
	cc.SetColourCode(0xBBBB) // decoder configured with wrong colour

	pdu := PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: []byte{0x00, 0x40, 0x00, 0x00, 0x00},
	}
	info := pduToType1Bits(pdu, 124)
	type5 := EncodeSCHHD(info, 0xAAAA) // encoded with different colour
	dibits := framing.BitsToDibits(type5)

	stream := make([]uint8, 30)
	stream = append(stream, NormalSyncDibits()...)
	stream = append(stream, dibits...)

	cc.Process(stream, 0)

	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked || ev.Kind == events.KindGrant {
				t.Errorf("ChannelCodingOn accepted a frame with wrong colour code: %v", ev.Kind)
			}
		default:
			return
		}
	}
}

// TestSetChannelCodingDefault: the zero-value ControlChannel uses
// ChannelCodingOff (the legacy 48-dibit raw path).
func TestSetChannelCodingDefault(t *testing.T) {
	cc := New(Options{Bus: events.NewBus(1)})
	if cc.channelCoding != ChannelCodingOff {
		t.Errorf("default channelCoding = %v, want ChannelCodingOff", cc.channelCoding)
	}
	cc.SetChannelCoding(ChannelCodingOn)
	if cc.channelCoding != ChannelCodingOn {
		t.Errorf("SetChannelCoding(ChannelCodingOn) did not take effect")
	}
	cc.SetExpectedChannel(ChannelSCHF)
	if cc.channelType != ChannelSCHF {
		t.Errorf("SetExpectedChannel(ChannelSCHF) did not take effect")
	}
	cc.SetColourCode(0x12345)
	if cc.colourCode != 0x12345 {
		t.Errorf("SetColourCode(0x12345) did not take effect")
	}
	// Colour code mask should clip to low 30 bits.
	cc.SetColourCode(0xFFFFFFFF)
	if cc.colourCode != 0x3FFFFFFF {
		t.Errorf("SetColourCode(0xFFFFFFFF) was not masked to 30 bits: got %#x", cc.colourCode)
	}
}

// TestParseChannelCoding covers the config-string →
// ChannelCodingMode mapping the ccdecoder connector uses to
// translate the `tetra_channel_coding` YAML field into a
// SetChannelCoding call.
func TestParseChannelCoding(t *testing.T) {
	cases := []struct {
		in   string
		want ChannelCodingMode
		ok   bool
	}{
		{"", ChannelCodingOn, true},
		{"on", ChannelCodingOn, true},
		{"ON", ChannelCodingOn, true},
		{"true", ChannelCodingOn, true},
		{"1", ChannelCodingOn, true},
		{" on ", ChannelCodingOn, true},
		{"off", ChannelCodingOff, true},
		{"OFF", ChannelCodingOff, true},
		{"false", ChannelCodingOff, true},
		{"0", ChannelCodingOff, true},
		{"nonsense", ChannelCodingOn, false},
	}
	for _, tc := range cases {
		got, ok := ParseChannelCoding(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseChannelCoding(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestParseChannelType covers the config-string → ChannelType
// mapping the ccdecoder connector uses to translate the
// `tetra_channel` YAML field into a SetExpectedChannel call.
func TestParseChannelType(t *testing.T) {
	cases := []struct {
		in   string
		want ChannelType
		ok   bool
	}{
		{"", ChannelSCHHD, true},
		{"sch/hd", ChannelSCHHD, true},
		{"SCH/HD", ChannelSCHHD, true},
		{"sch_hd", ChannelSCHHD, true},
		{"schhd", ChannelSCHHD, true},
		{"bnch", ChannelSCHHD, true}, // BNCH shares the SCH/HD coding chain.
		{"stch", ChannelSCHHD, true}, // STCH too.
		{"sch/f", ChannelSCHF, true},
		{"schf", ChannelSCHF, true},
		{"sch/hu", ChannelSCHHU, true},
		{"schhu", ChannelSCHHU, true},
		{"bsch", ChannelBSCH, true},
		{"BSCH", ChannelBSCH, true},
		{"aach", ChannelAACH, true},
		{"AACH", ChannelAACH, true},
		{"nonsense", ChannelSCHHD, false},
		{"sch/q", ChannelSCHHD, false},
	}
	for _, tc := range cases {
		got, ok := ParseChannelType(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseChannelType(%q) = (%v, %v), want (%v, %v)",
				tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestChannelDibitCount sanity check that each channel maps to
// its spec-correct type-5-bit / dibit count.
func TestChannelDibitCount(t *testing.T) {
	cases := []struct {
		ch        ChannelType
		wantDibit int
	}{
		{ChannelAACH, 15},
		{ChannelBSCH, 60},
		{ChannelSCHHD, 108},
		{ChannelSCHHU, 84},
		{ChannelSCHF, 216},
	}
	for _, tc := range cases {
		got := channelDibitCount(tc.ch)
		if got != tc.wantDibit {
			t.Errorf("channelDibitCount(%d) = %d, want %d", tc.ch, got, tc.wantDibit)
		}
	}
}
