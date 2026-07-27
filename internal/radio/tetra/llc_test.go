package tetra

import "testing"

// bl builds an LLC basic-link header as a one-bit-per-byte slice: the 4-bit PDU
// type followed by hdrExtra sequence bits (N(R)/N(S)), matching Tables
// 21.4/21.6/21.8/21.10.
func bl(pduType llcPDUType, hdrExtra int) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(pduType), 4)
	if hdrExtra > 0 {
		w.u(0, hdrExtra)
	}
	return w.bits
}

// TestParseLLCStripsBasicLinkHeaders checks each basic-link header width: the
// returned TL-SDU must start exactly at the byte after the header (and before any
// FCS). We wrap a known 16-bit payload and assert it comes back intact.
func TestParseLLCStripsBasicLinkHeaders(t *testing.T) {
	payload := []byte{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1} // 16 bits
	cases := []struct {
		name    string
		pduType llcPDUType
		extra   int  // sequence bits after the 4-bit type
		fcs     bool // trailing 32-bit FCS
	}{
		{"BL-ADATA", llcBLADATA, 2, false},
		{"BL-DATA", llcBLDATA, 1, false},
		{"BL-ACK", llcBLACK, 1, false},
		{"BL-UDATA", llcBLUDATA, 0, false},
		{"BL-ADATA-FCS", llcBLADATAFCS, 2, true},
		{"BL-DATA-FCS", llcBLDATAFCS, 1, true},
		{"BL-ACK-FCS", llcBLACKFCS, 1, true},
		{"BL-UDATA-FCS", llcBLUDATAFCS, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pdu := append(bl(tc.pduType, tc.extra), payload...)
			if tc.fcs {
				pdu = append(pdu, make([]byte, llcFCSBits)...) // 32-bit FCS trailer
			}
			tl, ok := ParseLLC(pdu)
			if !ok {
				t.Fatalf("ParseLLC ok=false for %s", tc.name)
			}
			if len(tl) != len(payload) {
				t.Fatalf("%s: TL-SDU len = %d, want %d (header/FCS not stripped correctly)", tc.name, len(tl), len(payload))
			}
			for i := range payload {
				if tl[i] != payload[i] {
					t.Fatalf("%s: TL-SDU bit %d = %d, want %d", tc.name, i, tl[i], payload[i])
				}
			}
		})
	}
}

// TestParseLLCRejectsAdvancedLink ensures advanced-link / supplementary /
// layer-2-signalling PDUs (types 8..15) — which do not carry CMCE call control —
// are reported as not basic-link rather than misparsed.
func TestParseLLCRejectsAdvancedLink(t *testing.T) {
	for _, pt := range []llcPDUType{0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF} {
		w := &cmceBitWriter{}
		w.u(uint64(pt), 4)
		w.u(0, 40)
		if _, ok := ParseLLC(w.bits); ok {
			t.Errorf("ParseLLC accepted advanced-link PDU type %#x", pt)
		}
	}
}

// TestLLCRealignsCMCE is the ISSI/GSSI regression. On real air the MAC TM-SDU is
// an LLC PDU; feeding it straight into ParseCMCE (skipping the LLC header) reads
// the 3-bit MLE discriminator and every downstream address at the wrong bit
// offset, corrupting the decoded ISSI/GSSI. Here a D-SETUP with a known GSSI
// (temporary address) and calling-party ISSI is wrapped in a BL-ADATA LLC PDU:
// ParseCMCE on the raw LLC PDU fails or misframes, but ParseCMCE on the
// LLC-stripped TL-SDU recovers the exact GSSI and ISSI.
func TestLLCRealignsCMCE(t *testing.T) {
	const (
		callID = 0x0123
		gssi   = 0x0F5670 // 24-bit GSSI (temporary address)
		issi   = 0x0123AB // 24-bit calling-party ISSI
	)
	// The MLE-PDU (TL-SDU): a D-SETUP with GSSI + calling ISSI.
	c := &cmceBitWriter{}
	c.header(CMCETypeDSetup)
	c.u(callID, 14)
	c.u(0, 4+1+1+8+2+1) // timeout..txReqPerm
	c.u(uint64(callPriorityEmergency), 4)
	c.u(1, 1)             // O-bit
	c.opt(false, 0, 6)    // notification indicator absent
	c.opt(true, gssi, 24) // temporary address (GSSI)
	c.pbit(true)          // calling party type identifier present
	c.u(1, 2)             // CPTI = 1 ⇒ SSI present
	c.u(issi, 24)         // calling party SSI (ISSI)

	// Wrap it in a BL-ADATA LLC PDU (4-bit type + N(R) + N(S) = 6-bit header).
	llcPDU := append(bl(llcBLADATA, 2), c.bits...)

	// Without stripping LLC, the address fields are misframed: ParseCMCE either
	// rejects the PDU (wrong MLE discriminator) or returns the wrong ISSI/GSSI.
	if bad, ok := ParseCMCE(llcPDU); ok && bad.GroupSSI == gssi && bad.PartySSI == issi {
		t.Fatal("ParseCMCE decoded correct ISSI/GSSI from an un-stripped LLC PDU — the LLC layer would be a no-op")
	}

	// Stripping LLC first realigns the L3 fields.
	tl, ok := ParseLLC(llcPDU)
	if !ok {
		t.Fatal("ParseLLC ok=false on a BL-ADATA PDU")
	}
	msg, ok := ParseCMCE(tl)
	if !ok {
		t.Fatal("ParseCMCE ok=false on the LLC-stripped TL-SDU")
	}
	if msg.Type != CMCETypeDSetup {
		t.Errorf("Type = %#x, want D-SETUP", msg.Type)
	}
	if msg.GroupSSI != gssi {
		t.Errorf("GroupSSI = %#x, want %#x (GSSI)", msg.GroupSSI, gssi)
	}
	if msg.PartySSI != issi {
		t.Errorf("PartySSI = %#x, want %#x (calling ISSI)", msg.PartySSI, issi)
	}
	if !msg.Emergency {
		t.Error("Emergency = false, want true")
	}
}
