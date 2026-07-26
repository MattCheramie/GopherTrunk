package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// This file is the failing-first regression suite for the RSCorrect weak-frame
// recovery path (issue #915): a Phase 2 MAC PDU that framed and descrambled at
// the right phase but carries a handful of GF(2^6) symbol errors from
// marginal-SNR demod. The detection-only gate (RSOn) drops it — which is why
// the ground-truth replay recovered 0 source RIDs — while RSCorrect repairs up
// to t=4 symbol errors and recovers the correct RID. The safety tests pin the
// t=4 radius and the corrected-PDU known-opcode gate so correction cannot inject
// a bogus PDU.

// gvcuCodeword builds a valid RS(24,16,9) MAC-PDU codeword (18 bytes) for a
// GROUP_VOICE_CHANNEL_USER-abbreviated PDU carrying the given talkgroup and
// source RID. The 12 information bytes hold opcode + svc-opts + TG + RID; the
// 16 RS information symbols are computed from those bytes and RS-encoded to a
// full 24-symbol codeword, then packed MSB-first into 18 bytes exactly as the
// verify/correct helpers regroup them.
func gvcuCodeword(opcode byte, tg uint16, rid uint32) [18]byte {
	var infoBytes [12]byte
	infoBytes[0] = opcode
	infoBytes[1] = 0x00 // service options
	infoBytes[2] = byte(tg >> 8)
	infoBytes[3] = byte(tg)
	infoBytes[4] = byte(rid >> 16)
	infoBytes[5] = byte(rid >> 8)
	infoBytes[6] = byte(rid)
	// bytes 7..11 padding (0)

	// 96 information bits (MSB-first) → 16 six-bit GF(2^6) symbols.
	var info [16]byte
	for i := 0; i < 16; i++ {
		var s byte
		for j := 0; j < 6; j++ {
			bitIdx := i*6 + j
			bit := (infoBytes[bitIdx>>3] >> uint(7-(bitIdx&7))) & 1
			s = (s << 1) | bit
		}
		info[i] = s
	}
	cw := framing.EncodeRS24_16(info)
	return packSymbols18(cw)
}

// packSymbols18 packs 24 six-bit symbols MSB-first into 18 bytes.
func packSymbols18(cw [24]byte) [18]byte {
	var pdu [18]byte
	bitIdx := 0
	for _, sym := range cw {
		for b := 5; b >= 0; b-- {
			if (sym>>uint(b))&1 == 1 {
				pdu[bitIdx>>3] |= 1 << uint(7-(bitIdx&7))
			}
			bitIdx++
		}
	}
	return pdu
}

// symbolsOf regroups an 18-byte PDU into its 24 six-bit symbols (the inverse of
// packSymbols18 / the grouping verifyMACPDURS and correctMACPDURS use).
func symbolsOf(pdu [18]byte) [24]byte {
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	var syms [24]byte
	for i := 0; i < 24; i++ {
		var s byte
		for j := 0; j < 6; j++ {
			s = (s << 1) | bits[i*6+j]
		}
		syms[i] = s
	}
	return syms
}

// corruptSymbols flips `count` distinct symbols of pdu (spaced across the
// codeword) to non-zero-difference values and returns the repacked 18 bytes.
// The flips land inside the information region so they perturb decodable fields,
// mirroring the marginal-SNR symbol errors seen on air.
func corruptSymbols(pdu [18]byte, count int) [18]byte {
	syms := symbolsOf(pdu)
	// Positions 1,3,5,7 sit inside the 16 information symbols (RID lives in
	// symbols ~5-9), so a corruption here damages the recovered RID unless RS
	// repairs it.
	positions := []int{1, 3, 5, 7, 9, 11}
	for i := 0; i < count && i < len(positions); i++ {
		p := positions[i]
		syms[p] ^= 0o37 // flip low 5 bits → guaranteed non-zero symbol delta
	}
	return packSymbols18(syms)
}

// macDibitsFromPDU turns an 18-byte MAC PDU into the 72 raw information dibits
// the TrellisOff decode path consumes: decodeMACChannelAndParse runs
// DibitsToBits then PackBitsMSB, so BitsToDibits(bits(pdu)) round-trips to the
// same 18 bytes.
func macDibitsFromPDU(pdu [18]byte) []uint8 {
	bits := make([]byte, 144)
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (pdu[i] >> uint(7-j)) & 1
		}
	}
	return framing.BitsToDibits(bits)
}

const (
	rsTestTG  = 0x4E20 // 20000
	rsTestRID = 202419 // the exact RID from issue #773 / #915 territory
)

// TestRSCorrectRecoversRIDThatRSOnDrops is the failing-first check: a MAC PDU
// with a few symbol errors is DROPPED by RSOn (detection-only) but RECOVERED
// with the correct source RID by RSCorrect. This is the weak-frame case that
// left issue #915's ground-truth replay at 0 recovered RIDs.
func TestRSCorrectRecoversRIDThatRSOnDrops(t *testing.T) {
	clean := gvcuCodeword(byte(OpGroupVoiceChannelUserAbbreviated), rsTestTG, rsTestRID)

	for _, nErr := range []int{1, 2, 3, 4} {
		corrupt := corruptSymbols(clean, nErr)
		dibits := macDibitsFromPDU(corrupt)

		// RSOn: detection-only. A codeword with any residual symbol error must
		// be rejected — the pre-fix behaviour that yields rids_recovered = 0.
		if _, ok := decodeMACChannelAndParse(dibits, TrellisOff, InterleaveOff, RSOn); ok {
			t.Fatalf("nErr=%d: RSOn accepted a corrupted codeword; expected rejection", nErr)
		}

		// RSCorrect: repairs up to t=4 and recovers the PDU + correct RID.
		pdu, ok := decodeMACChannelAndParse(dibits, TrellisOff, InterleaveOff, RSCorrect)
		if !ok {
			t.Fatalf("nErr=%d: RSCorrect failed to recover a within-radius codeword", nErr)
		}
		u, isUser := pdu.AsGroupVoiceChannelUser()
		if !isUser {
			t.Fatalf("nErr=%d: recovered PDU is not a GROUP_VOICE_CHANNEL_USER (opcode %s)", nErr, pdu.Opcode)
		}
		if u.SourceID != rsTestRID {
			t.Fatalf("nErr=%d: recovered SourceID = %d, want %d", nErr, u.SourceID, rsTestRID)
		}
		if u.GroupAddress != rsTestTG {
			t.Fatalf("nErr=%d: recovered GroupAddress = %d, want %d", nErr, u.GroupAddress, rsTestTG)
		}
	}
}

// TestRSCorrectIsSupersetOfRSOnOnCleanCodeword confirms a clean codeword decodes
// identically under RSOn and RSCorrect (RSCorrect strictly supersets RSOn: the
// 0-correction path is not gated on the opcode set, so behaviour on valid
// traffic is unchanged).
func TestRSCorrectIsSupersetOfRSOnOnCleanCodeword(t *testing.T) {
	clean := gvcuCodeword(byte(OpGroupVoiceChannelUserAbbreviated), rsTestTG, rsTestRID)
	dibits := macDibitsFromPDU(clean)

	on, okOn := decodeMACChannelAndParse(dibits, TrellisOff, InterleaveOff, RSOn)
	corr, okCorr := decodeMACChannelAndParse(dibits, TrellisOff, InterleaveOff, RSCorrect)
	if !okOn || !okCorr {
		t.Fatalf("clean codeword: RSOn ok=%v, RSCorrect ok=%v; want both true", okOn, okCorr)
	}
	uOn, _ := on.AsGroupVoiceChannelUser()
	uCorr, _ := corr.AsGroupVoiceChannelUser()
	if uOn != uCorr {
		t.Fatalf("clean codeword decoded differently: RSOn %+v vs RSCorrect %+v", uOn, uCorr)
	}
	if uCorr.SourceID != rsTestRID {
		t.Fatalf("clean codeword SourceID = %d, want %d", uCorr.SourceID, rsTestRID)
	}
}

// TestRSCorrectRejectsBeyondRadius pins the correction radius: a codeword with
// more than t=4 symbol errors is uncorrectable and must be rejected, not
// miscorrected into a plausible PDU.
func TestRSCorrectRejectsBeyondRadius(t *testing.T) {
	clean := gvcuCodeword(byte(OpGroupVoiceChannelUserAbbreviated), rsTestTG, rsTestRID)
	corrupt := corruptSymbols(clean, 6) // 6 > t=4
	dibits := macDibitsFromPDU(corrupt)

	if _, ok := decodeMACChannelAndParse(dibits, TrellisOff, InterleaveOff, RSCorrect); ok {
		t.Fatalf("RSCorrect accepted a codeword with 6 symbol errors (beyond t=4)")
	}
}

// TestRSCorrectKnownOpcodeGate confirms the safety gate: a codeword that
// corrects cleanly (within t=4) but resolves to an UNRECOGNISED opcode is
// rejected, so a wrong-phase / weak-frame window cannot be miscorrected into a
// PDU that injects a bogus source RID (issue #915 / #924). The same corrupted
// codeword with a known opcode is accepted.
func TestRSCorrectKnownOpcodeGate(t *testing.T) {
	const unknownOpcode = 0x5A // not in Opcode.IsKnown()
	if Opcode(unknownOpcode).IsKnown() {
		t.Fatalf("test precondition: opcode 0x%02X unexpectedly recognised", unknownOpcode)
	}

	// A valid codeword whose opcode is unknown, corrupted within the radius:
	// RS corrects it back to the unknown opcode, and the gate must reject it.
	unknownCW := gvcuCodeword(unknownOpcode, rsTestTG, rsTestRID)
	unknownDibits := macDibitsFromPDU(corruptSymbols(unknownCW, 2))
	if _, ok := decodeMACChannelAndParse(unknownDibits, TrellisOff, InterleaveOff, RSCorrect); ok {
		t.Fatalf("RSCorrect accepted a corrected PDU with an unrecognised opcode 0x%02X", unknownOpcode)
	}

	// Same corruption count, known opcode → accepted.
	knownDibits := macDibitsFromPDU(corruptSymbols(
		gvcuCodeword(byte(OpGroupVoiceChannelUserAbbreviated), rsTestTG, rsTestRID), 2))
	if _, ok := decodeMACChannelAndParse(knownDibits, TrellisOff, InterleaveOff, RSCorrect); !ok {
		t.Fatalf("RSCorrect rejected a corrected PDU with a recognised opcode")
	}
}

// TestCorrectMACPDURSReencodesCleanCodeword is a unit check on the helper: the
// corrected output is always a valid RS codeword (verifyMACPDURS true), and a
// within-radius corruption is repaired back to the original bytes.
func TestCorrectMACPDURSReencodesCleanCodeword(t *testing.T) {
	clean := gvcuCodeword(byte(OpGroupVoiceChannelUserAbbreviated), rsTestTG, rsTestRID)
	corrupt := corruptSymbols(clean, 3)

	out, nErr, ok := correctMACPDURS(corrupt[:])
	if !ok {
		t.Fatalf("correctMACPDURS failed on a 3-error (within t=4) codeword")
	}
	if nErr != 3 {
		t.Errorf("correctMACPDURS reported %d corrections, want 3", nErr)
	}
	if !verifyMACPDURS(out) {
		t.Errorf("correctMACPDURS output does not satisfy the RS syndromes")
	}
	for i := range clean {
		if out[i] != clean[i] {
			t.Errorf("corrected byte %d = 0x%02X, want 0x%02X (original)", i, out[i], clean[i])
		}
	}
}
