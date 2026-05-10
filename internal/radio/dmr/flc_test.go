package dmr

import "testing"

func TestFLCRoundTripGroupVoiceUser(t *testing.T) {
	in := FLC{
		PF:             false,
		FLCO:           FLCOGroupVoiceUser,
		FID:            0x00,
		ServiceOptions: 0xC0, // emergency + encrypted
		DstAddr:        0x00ABCD, // talkgroup
		SrcAddr:        0x123456, // subscriber
	}
	out, err := ParseFLC(AssembleFLC(in))
	if err != nil {
		t.Fatalf("ParseFLC: %v", err)
	}
	if out != in {
		t.Errorf("round-trip:\n got %+v\nwant %+v", out, in)
	}
}

func TestFLCRoundTripUnitToUnit(t *testing.T) {
	in := FLC{
		FLCO:    FLCOUnitToUnitVoice,
		FID:     0x10,
		DstAddr: 0xFEDCBA,
		SrcAddr: 0x00DEAD,
	}
	out, err := ParseFLC(AssembleFLC(in))
	if err != nil {
		t.Fatalf("ParseFLC: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: %+v != %+v", out, in)
	}
}

func TestFLCRejectsShortBuffer(t *testing.T) {
	if _, err := ParseFLC(make([]byte, 8)); err == nil {
		t.Error("expected error for 8-byte buffer")
	}
}

func TestFLCAsGroupVoiceUser(t *testing.T) {
	f := FLC{
		FLCO:           FLCOGroupVoiceUser,
		ServiceOptions: 0x80, // emergency only
		DstAddr:        0x000064,
		SrcAddr:        0x100200,
	}
	g, ok := f.AsGroupVoiceUser()
	if !ok {
		t.Fatal("expected ok")
	}
	if g.GroupAddress != 0x64 || g.SourceID != 0x100200 {
		t.Errorf("addresses = %x/%x", g.GroupAddress, g.SourceID)
	}
	if !g.Emergency || g.Encrypted {
		t.Errorf("flags = emer=%v enc=%v, want emer=true enc=false", g.Emergency, g.Encrypted)
	}

	// Wrong opcode → no group-voice projection.
	if _, ok := (FLC{FLCO: FLCOTerminator}).AsGroupVoiceUser(); ok {
		t.Error("Terminator FLC reported as group-voice user")
	}
}

func TestFLCOStringCoversKnownAndUnknown(t *testing.T) {
	cases := map[FLCO]string{
		FLCOGroupVoiceUser:  "GroupVoiceChannelUser",
		FLCOUnitToUnitVoice: "UnitToUnitVoiceChannelUser",
		FLCOTerminator:      "Terminator",
		FLCO(0x2A):          "FLCO(2A)",
	}
	for code, want := range cases {
		if got := code.String(); got != want {
			t.Errorf("%X.String() = %s, want %s", uint8(code), got, want)
		}
	}
}
