package tetra

import "testing"

// TestParseSysInfoExtended decodes the extended (non-frequency) SYSINFO fields
// from the same fixture block the frequency tests use (carrier 2716 — the
// operator's 467.9125 MHz network). The expected values were derived by
// walking the block with osmo-tetra-sq5bpf's macpdu_decode_sysinfo field
// offsets (tetra_mac_pdu.c) — main carrier at bit 4, D-MLE-SYSINFO at the
// fixed offset 124−42 — independently of this package's parser; tetra-kit's
// Mac::pduProcessSysinfo implements the identical walk (TM-SDU = 42 bits at
// position 82).
func TestParseSysInfoExtended(t *testing.T) {
	bits := hexToBits(t, "8a9c4c0e928eec8bd0c0041cffffd700")
	ext, ok := ParseSysInfoExtended(bits)
	if !ok {
		t.Fatal("ParseSysInfoExtended ok=false on a full SYSINFO block")
	}
	if ext.SCCHInUse != 0 || ext.MSTxPwrMaxCell != 7 || ext.RxLevAccessMin != 4 ||
		ext.AccessParameter != 9 || ext.RadioDLTimeout != 4 {
		t.Errorf("access params = %+v, want scch=0 txpwr=7 rxlev=4 access=9 timeout=4", ext)
	}
	if ext.CCKValid || ext.CounterOrCCK != 0xEEC8 {
		t.Errorf("counter = valid=%v 0x%04x, want hyperframe 0xEEC8", ext.CCKValid, ext.CounterOrCCK)
	}
	if ext.OptionalFieldFlag != 2 || ext.OptionalField != 0xF4300 {
		t.Errorf("optional field = %d/0x%05x, want 2 (access code A)/0xF4300", ext.OptionalFieldFlag, ext.OptionalField)
	}
	// D-MLE-SYSINFO (the last 42 bits): LA + subscriber class + BS service
	// details. LA 1052 sits inside the 1021..1089 LA range this network's
	// neighbour broadcast advertises — corroborating the offset.
	if ext.LocationArea != 1052 || ext.SubscriberClass != 0xFFFF || ext.BSServiceDetails != 0xD70 {
		t.Errorf("d-mle-sysinfo = la=%d ss=0x%04x bs=0x%03x, want 1052/0xFFFF/0xD70",
			ext.LocationArea, ext.SubscriberClass, ext.BSServiceDetails)
	}
	if got, want := BSServiceDetailsString(ext.BSServiceDetails), "reg,dereg,min-mode,sys-wide,voice,data"; got != want {
		t.Errorf("BSServiceDetailsString = %q, want %q", got, want)
	}
	if dbm, ok := ext.MSTxPwrMaxCellDBm(); !ok || dbm != 45 {
		t.Errorf("MSTxPwrMaxCellDBm = %d/%v, want 45/true", dbm, ok)
	}
	if got := ext.RxLevAccessMinDBm(); got != -105 {
		t.Errorf("RxLevAccessMinDBm = %d, want -105", got)
	}

	// A short block (frequency-only decode) yields no extended fields; a
	// non-SYSINFO broadcast is rejected outright.
	if _, ok := ParseSysInfoExtended(bits[:60]); ok {
		t.Error("ParseSysInfoExtended accepted a short block")
	}
	if _, ok := ParseSysInfoExtended(hexToBits(t, "20760f572c3c83d538c90a1fe305009e20760f572c3c83d538c90a1fe305009e")); ok {
		t.Error("ParseSysInfoExtended accepted a non-broadcast PDU")
	}
}

// TestSCCHRendering pins the common-SCCH count → timeslot mapping (§21.4.4.1:
// n common SCCH occupy timeslots 2..n+1 of the main carrier).
func TestSCCHRendering(t *testing.T) {
	for scch, want := range map[uint8]string{0: "", 1: "TS2", 2: "TS2-3", 3: "TS2-4"} {
		if got := (SysInfoExt{SCCHInUse: scch}).SCCHTimeslots(); got != want {
			t.Errorf("SCCHTimeslots(%d) = %q, want %q", scch, got, want)
		}
	}
}

// TestSysInfoExtChangeGating: the hyperframe counter must not count as a
// parameter change (it advances every multiframe cycle), while a real
// parameter change — and a CCK id change — must.
func TestSysInfoExtChangeGating(t *testing.T) {
	a := SysInfoExt{SCCHInUse: 1, CounterOrCCK: 100}
	b := a
	b.CounterOrCCK = 101
	if !a.sameCellParams(b) {
		t.Error("a hyperframe tick counted as a cell-parameter change")
	}
	// The BS rotates which 20-bit optional definition each broadcast carries —
	// that rotation must not count as a change either.
	b.CounterOrCCK = 100
	b.OptionalFieldFlag = 3
	b.OptionalField = 0xABCDE
	if !a.sameCellParams(b) {
		t.Error("the rotating optional field counted as a cell-parameter change")
	}
	b.SCCHInUse = 2
	if a.sameCellParams(b) {
		t.Error("an SCCH change did not count as a cell-parameter change")
	}
	c := SysInfoExt{CCKValid: true, CounterOrCCK: 100}
	d := c
	d.CounterOrCCK = 101
	if c.sameCellParams(d) {
		t.Error("a CCK id change did not count as a cell-parameter change")
	}
}
