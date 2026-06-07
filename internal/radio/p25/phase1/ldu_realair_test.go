package phase1_test

import (
	"encoding/hex"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
)

// Independent real-air LDU fixture for the full
// dibits→StripStatusSymbols→voice-offset→IMBE-channel-decode path.
//
// Unlike ldu_e2e_test.go / TestExtractVoiceFramesRoundTrip — which
// build the LDU from the package's own lduVoiceOffsets and so only
// prove self-consistency — this fixture was generated entirely from
// the canonical mbelib/DSD reference, OUTSIDE the GopherTrunk code:
//
//   - 9 random valid IMBE frames were channel-encoded (Golay(23,12)
//     with golayGenerator, Hamming(15,11) with hammingGenerator, the
//     §7.4 PRBS scramble, the §7.5 interleave);
//   - placed at the canonical P25 LDU1 payload bit positions
//     112,256,440,624,808,992,1176,1360,1536 (u_0 and u_1 adjacent,
//     an LC block after each of u_1..u_6, LSD after u_7) — hard-coded
//     here, NOT read from the package;
//   - the rest of the 1680-bit payload filled with a deterministic
//     pattern;
//   - status symbols interleaved as "2 bits after every 70 bits".
//
// ExtractVoiceFrames must recover the 9 original frames bit-for-bit.
// A wrong lduVoiceOffsets table (the issue-#489 follow-up bug, where
// an LC block was placed between u_0 and u_1) makes u_1..u_7 decode
// to garbage and fails this test.
const realAirLDUHex = "aaaaaaaaaaaaaaaaa9aaaaaaaaaa8656a025264b9bfb8ce0b1e3e962183c03cfc827cc79b40c262e42cc33f341f6ad1cbb15e4eaaaa9aaaaa22cadd412fd79c6d3a89d2e4709e3d9bd6d78aaaaaaaaaa79fa77f63d42be2a4d9521653262a83bf4bf99aaaaaaaaaad9cdf0d12869d485f9f8a4203dba7b978001aaaaaaa9aaa37d439d49bcf48915662f83100f7cc8a19cffaaaaaaaaaac27de4492602dbf9eed3ad585389375f83250aa9aaaaaaa8a9f2144e85a58bc7027fc1539cddda519d9deaaaaaaaad2af9a1b6162f056319675f3499db4b97d1ed"

var realAirExpectedFrames = []string{
	"69e70d5a196a2c26836f17",
	"5a0c44e99fcc97df9e6df9",
	"37c5bfee3ddadc491a0cfe",
	"78b608f1096bc68d6b6b5a",
	"92916f692485d5b4f1a779",
	"5a1c0a0d38daa2f140cc13",
	"b2569b992c5ae387e7f7a6",
	"24c8fb9eae031e0c8dffaf",
	"b49a2a9a051001acfc76ff",
}

func TestExtractVoiceFramesRealAirFixture(t *testing.T) {
	packed, err := hex.DecodeString(realAirLDUHex)
	if err != nil {
		t.Fatalf("bad LDU hex: %v", err)
	}
	// Expand 216 packed bytes → 1728 one-bit-per-byte LDU bits (MSB-first).
	ldu := make([]byte, phase1.LDUTotalBits)
	for b := 0; b < phase1.LDUTotalBits; b++ {
		ldu[b] = (packed[b/8] >> (7 - uint(b)%8)) & 1
	}

	frames, errs, err := phase1.ExtractVoiceFrames(ldu)
	if err != nil {
		t.Fatalf("ExtractVoiceFrames: %v", err)
	}
	if errs != 0 {
		t.Errorf("clean fixture decoded with %d corrected errors, want 0", errs)
	}
	for i, wantHex := range realAirExpectedFrames {
		want, _ := hex.DecodeString(wantHex)
		if got := frames[i]; hex.EncodeToString(got) != wantHex {
			t.Errorf("voice subframe u_%d mismatch\n got %x\nwant %x", i, got, want)
		}
	}
}
