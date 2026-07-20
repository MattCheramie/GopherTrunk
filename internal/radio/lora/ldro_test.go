package lora

import "testing"

// TestAutoLDRO pins the Semtech Ts >= 16 ms rule across the standard
// SF x BW grid: LDRO on for SF11/SF12 at 125 kHz and SF12 at 250 kHz,
// off everywhere else.
func TestAutoLDRO(t *testing.T) {
	cases := []struct {
		sf   int
		bw   Bandwidth
		want bool
	}{
		{7, BW125, false}, {10, BW125, false}, {11, BW125, true}, {12, BW125, true},
		{11, BW250, false}, {12, BW250, true},
		{12, BW500, false},
	}
	for _, c := range cases {
		if got := AutoLDRO(c.sf, c.bw); got != c.want {
			t.Errorf("AutoLDRO(SF%d, %d) = %v, want %v", c.sf, int(c.bw), got, c.want)
		}
	}
}

// TestParseLDROMode covers the config-string -> LDROMode mapping.
func TestParseLDROMode(t *testing.T) {
	cases := []struct {
		in   string
		want LDROMode
		ok   bool
	}{
		{"", LDROAuto, true}, {"auto", LDROAuto, true}, {"AUTO", LDROAuto, true},
		{"on", LDROOn, true}, {"true", LDROOn, true}, {"1", LDROOn, true},
		{"off", LDROOff, true}, {"false", LDROOff, true}, {"0", LDROOff, true},
		{" on ", LDROOn, true},
		{"maybe", LDROAuto, false},
	}
	for _, c := range cases {
		got, ok := ParseLDROMode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseLDROMode(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestPayloadSymbolCountLDRO confirms a reduced-rate payload takes MORE
// symbols than a full-rate one — each block carries SF-2 nibbles instead
// of SF, so more blocks are needed.
func TestPayloadSymbolCountLDRO(t *testing.T) {
	const sf, cr, payloadLen = 12, 4, 32
	full := PayloadSymbolCount(sf, cr, payloadLen, true, false)
	reduced := PayloadSymbolCount(sf, cr, payloadLen, true, true)
	if reduced <= full {
		t.Errorf("LDRO payload symbols = %d, want > full-rate %d", reduced, full)
	}
}

// TestModDemodLDROAuto drives the full modulate -> demodulate path at the
// (SF, BW) pairs where LDRO auto-engages and confirms both the round trip
// and that the decoded frame reports LDRO correctly. SF12/SF11 at 125 kHz
// engage LDRO; SF9 does not.
func TestModDemodLDROAuto(t *testing.T) {
	osf := DefaultOversample
	pl := []byte("ldro payload")
	cases := []struct {
		sf       int
		wantLDRO bool
	}{
		{9, false}, {11, true}, {12, true},
	}
	for _, c := range cases {
		iq := LoRaModulate(pl, c.sf, 4, osf, BW125, true, 0x12)
		iq = pad(iq, 800, 400)
		m := NewDemodulator(c.sf, osf, BW125)
		if m.LDRO() != c.wantLDRO {
			t.Errorf("SF%d: demod LDRO = %v, want %v", c.sf, m.LDRO(), c.wantLDRO)
		}
		f, ok := m.DecodeFirst(iq)
		if !ok || !f.CRCOK || string(f.Payload) != string(pl) {
			t.Fatalf("SF%d: ok=%v crc=%v payload=%x", c.sf, ok, f.CRCOK, f.Payload)
		}
		if f.LDRO != c.wantLDRO {
			t.Errorf("SF%d: frame LDRO = %v, want %v", c.sf, f.LDRO, c.wantLDRO)
		}
	}
}

// TestModDemodLDROForced round-trips with LDRO forced on at an SF where the
// auto rule would leave it off (SF7), exercising the operator-override path.
func TestModDemodLDROForced(t *testing.T) {
	osf := DefaultOversample
	pl := []byte("forced ldro")
	iq := LoRaModulateMode(pl, 7, 4, osf, BW125, true, 0x12, LDROOn)
	iq = pad(iq, 500, 500)
	m := NewDemodulatorMode(7, osf, BW125, LDROOn)
	if !m.LDRO() {
		t.Fatal("forced LDROOn demod reports LDRO=false")
	}
	f, ok := m.DecodeFirst(iq)
	if !ok || !f.CRCOK || string(f.Payload) != string(pl) {
		t.Fatalf("forced-LDRO round trip: ok=%v crc=%v payload=%x", ok, f.CRCOK, f.Payload)
	}
	if !f.LDRO {
		t.Error("forced-LDRO frame reports LDRO=false")
	}
}

// TestLDROMismatchFailsToDecode proves LDRO actually changes the payload
// wire format: a frame modulated with LDRO on does not decode to the same
// payload when the demodulator has LDRO off (and vice versa). The header is
// LDRO-independent, so it still parses — but the reduced-rate payload
// geometry differs, so the payload does not survive the mismatch.
func TestLDROMismatchFailsToDecode(t *testing.T) {
	osf := DefaultOversample
	pl := []byte("mismatch check")

	// TX LDRO on, RX LDRO off.
	iqOn := pad(LoRaModulateMode(pl, 7, 4, osf, BW125, true, 0x12, LDROOn), 500, 500)
	if f, ok := NewDemodulatorMode(7, osf, BW125, LDROOff).DecodeFirst(iqOn); ok && f.CRCOK && string(f.Payload) == string(pl) {
		t.Error("LDRO-on frame wrongly decoded by an LDRO-off demod")
	}

	// TX LDRO off, RX LDRO on.
	iqOff := pad(LoRaModulateMode(pl, 7, 4, osf, BW125, true, 0x12, LDROOff), 500, 500)
	if f, ok := NewDemodulatorMode(7, osf, BW125, LDROOn).DecodeFirst(iqOff); ok && f.CRCOK && string(f.Payload) == string(pl) {
		t.Error("LDRO-off frame wrongly decoded by an LDRO-on demod")
	}
}
