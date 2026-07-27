package tetra

import "testing"

// cmceBitWriter emits an MSB-first one-bit-per-byte stream — the same convention
// bitReader consumes — so tests can hand-assemble bit-packed CMCE PDUs. A single
// fixture is round-tripped through the reader in TestCMCEBitWriterRoundTrip to
// guarantee the two conventions match (a mismatch would make every fixture
// self-consistently wrong).
type cmceBitWriter struct{ bits []byte }

func (w *cmceBitWriter) u(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, byte((v>>uint(i))&1))
	}
}

// opt appends a type-2 element: a presence bit, then n content bits if present.
func (w *cmceBitWriter) opt(present bool, v uint64, n int) {
	if present {
		w.u(1, 1)
		w.u(v, n)
	} else {
		w.u(0, 1)
	}
}

// pbit appends a bare presence bit (for elements whose content width varies).
func (w *cmceBitWriter) pbit(present bool) {
	if present {
		w.u(1, 1)
	} else {
		w.u(0, 1)
	}
}

// cmceHeader writes the 3-bit CMCE MLE discriminator + 5-bit PDU type.
func (w *cmceBitWriter) header(t CMCEType) {
	w.u(uint64(cmceMLEPD), 3)
	w.u(uint64(t), 5)
}

func TestCMCEBitWriterRoundTrip(t *testing.T) {
	// A D-RELEASE with known fields must read back exactly what was written —
	// proves the writer and bitReader share the MSB-first convention.
	w := &cmceBitWriter{}
	w.header(CMCETypeDRelease)
	w.u(0x1ABC&0x3FFF, 14) // call id
	w.u(0x0A, 5)           // disconnect cause
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ParseCMCE returned ok=false on a valid D-RELEASE")
	}
	if msg.CallIdentifier != (0x1ABC & 0x3FFF) {
		t.Errorf("CallIdentifier = %#x, want %#x", msg.CallIdentifier, 0x1ABC&0x3FFF)
	}
	if msg.DisconnectCause != 0x0A {
		t.Errorf("DisconnectCause = %#x, want 0x0A", msg.DisconnectCause)
	}
}

func TestParseCMCE_DSetupEmergency(t *testing.T) {
	const (
		callID = 0x0123
		gssi   = 0x0F5670 // 24-bit GSSI
		src    = 0x0123AB // 24-bit calling party SSI
	)
	w := &cmceBitWriter{}
	w.header(CMCETypeDSetup)
	w.u(callID, 14)                       // call identifier
	w.u(0, 4)                             // call time-out
	w.u(0, 1)                             // hook
	w.u(0, 1)                             // simplex/duplex
	w.u(0, 8)                             // basic service info
	w.u(0, 2)                             // transmission grant
	w.u(0, 1)                             // transmission request permission
	w.u(uint64(callPriorityEmergency), 4) // call priority = emergency
	w.u(1, 1)                             // O-bit: optional elements present
	w.opt(false, 0, 6)                    // notification indicator absent
	w.opt(true, gssi, 24)                 // temporary address (GSSI) present
	w.pbit(true)                          // calling party type identifier present
	w.u(1, 2)                             // CPTI = 1 (SSI present)
	w.u(src, 24)                          // calling party SSI

	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ParseCMCE ok=false on a valid D-SETUP")
	}
	if msg.Type != CMCETypeDSetup {
		t.Errorf("Type = %#x, want D-SETUP", msg.Type)
	}
	if !msg.Emergency {
		t.Error("Emergency = false, want true (call priority 1111)")
	}
	if msg.CallIdentifier != callID {
		t.Errorf("CallIdentifier = %#x, want %#x", msg.CallIdentifier, callID)
	}
	if msg.GroupSSI != gssi {
		t.Errorf("GroupSSI = %#x, want %#x", msg.GroupSSI, gssi)
	}
	if msg.PartySSI != src {
		t.Errorf("PartySSI = %#x, want %#x", msg.PartySSI, src)
	}
}

func TestParseCMCE_DSetupNonEmergency(t *testing.T) {
	w := &cmceBitWriter{}
	w.header(CMCETypeDSetup)
	w.u(1, 14)          // call id
	w.u(0, 4+1+1+8+2+1) // timeout..txreqperm
	w.u(0x1, 4)         // call priority = 1 (lowest), not emergency
	w.u(0, 1)           // O-bit clear
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ok=false")
	}
	if msg.Emergency {
		t.Error("Emergency = true, want false (priority 0001)")
	}
	if !msg.HasPriority || msg.Priority != 1 {
		t.Errorf("Priority = %d (has=%v), want 1", msg.Priority, msg.HasPriority)
	}
	if msg.GroupSSI != 0 || msg.PartySSI != 0 {
		t.Errorf("expected no optional SSIs, got group=%#x party=%#x", msg.GroupSSI, msg.PartySSI)
	}
}

func TestParseCMCE_DConnectOptionalPriorityAndGSSI(t *testing.T) {
	const gssi = 0x0ABCDE
	w := &cmceBitWriter{}
	w.header(CMCETypeDConnect)
	w.u(0x2A, 14)                                 // call id
	w.u(0, 4+1+1+2+1+1)                           // timeout..call ownership
	w.u(1, 1)                                     // O-bit
	w.opt(true, uint64(callPriorityEmergency), 4) // call priority = emergency
	w.opt(false, 0, 8)                            // basic service absent
	w.opt(true, gssi, 24)                         // temporary address present
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ok=false")
	}
	if !msg.Emergency {
		t.Error("Emergency = false, want true")
	}
	if msg.GroupSSI != gssi {
		t.Errorf("GroupSSI = %#x, want %#x", msg.GroupSSI, gssi)
	}
	if msg.CallIdentifier != 0x2A {
		t.Errorf("CallIdentifier = %#x, want 0x2A", msg.CallIdentifier)
	}
}

func TestParseCMCE_DTxGrantedTalker(t *testing.T) {
	const talker = 0x00CAFE
	w := &cmceBitWriter{}
	w.header(CMCETypeDTxGranted)
	w.u(0x3F, 14)      // call id
	w.u(0, 2+1+1+1)    // tx grant..reserved
	w.u(1, 1)          // O-bit
	w.opt(false, 0, 6) // notification indicator absent
	w.pbit(true)       // transmitting party type identifier present
	w.u(1, 2)          // TPTI = 1
	w.u(talker, 24)    // transmitting party SSI
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ok=false")
	}
	if msg.Type != CMCETypeDTxGranted {
		t.Errorf("Type = %#x, want D-TX-GRANTED", msg.Type)
	}
	if msg.PartySSI != talker {
		t.Errorf("PartySSI = %#x, want %#x", msg.PartySSI, talker)
	}
}

func TestParseCMCE_DTxCeased(t *testing.T) {
	w := &cmceBitWriter{}
	w.header(CMCETypeDTxCeased)
	w.u(0x1BCD&0x3FFF, 14)
	w.u(0, 1)
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ok=false")
	}
	if msg.Type != CMCETypeDTxCeased || msg.CallIdentifier != (0x1BCD&0x3FFF) {
		t.Errorf("got Type=%#x CallId=%#x", msg.Type, msg.CallIdentifier)
	}
}

func TestParseCMCE_NoOptionalChain(t *testing.T) {
	// O-bit clear: mandatory fields still parse; optionals stay zero.
	w := &cmceBitWriter{}
	w.header(CMCETypeDSetup)
	w.u(0x2B, 14)
	w.u(0, 4+1+1+8+2+1)
	w.u(0x3, 4) // priority 3
	w.u(0, 1)   // O-bit clear
	msg, ok := ParseCMCE(w.bits)
	if !ok {
		t.Fatal("ok=false")
	}
	if msg.GroupSSI != 0 || msg.PartySSI != 0 {
		t.Error("optional SSIs should be absent with O-bit clear")
	}
	if msg.CallIdentifier != 0x2B || msg.Priority != 3 {
		t.Errorf("mandatory fields wrong: callid=%#x prio=%d", msg.CallIdentifier, msg.Priority)
	}
}

func TestParseCMCE_RejectsNonCMCE(t *testing.T) {
	w := &cmceBitWriter{}
	w.u(0x1, 3) // MLE PD = 001 (MM), not CMCE
	w.u(uint64(CMCETypeDSetup), 5)
	w.u(0, 40)
	if _, ok := ParseCMCE(w.bits); ok {
		t.Error("ParseCMCE accepted a non-CMCE MLE discriminator")
	}
}

func TestParseCMCE_RejectsUnmodelledType(t *testing.T) {
	w := &cmceBitWriter{}
	w.u(uint64(cmceMLEPD), 3)
	w.u(0x00, 5) // D-ALERT — not modelled
	w.u(0, 40)
	if _, ok := ParseCMCE(w.bits); ok {
		t.Error("ParseCMCE accepted an unmodelled PDU type")
	}
}

func TestParseCMCE_Truncated(t *testing.T) {
	// Header says D-SETUP but the slice ends inside the mandatory block.
	w := &cmceBitWriter{}
	w.header(CMCETypeDSetup)
	w.u(0x2B, 14)
	w.u(0, 5) // far short of the mandatory field count
	if _, ok := ParseCMCE(w.bits); ok {
		t.Error("ParseCMCE accepted a truncated mandatory block")
	}
}
