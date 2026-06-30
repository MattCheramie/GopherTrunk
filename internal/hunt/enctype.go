package hunt

import (
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/dmrcrypto"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/p25crypto"
)

// encTypeName maps a protocol + encryption algorithm id to a human label
// (AES-256, ADP/RC4, DES-OFB, …) by reusing the cryptolab algorithm-name
// tables — the same labels the Crypto Lab uses, so the survey and the
// cryptanalysis toolkit agree. An algid of 0 means the algorithm was not
// surfaced (e.g. a P25 Phase 1 control channel whose ALGID lives in the voice
// LDU2), and returns "" so the caller can still flag the call encrypted.
func encTypeName(protocol string, algid uint8) string {
	if algid == 0 {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(protocol), "dmr") {
		return dmrcrypto.AlgName(algid)
	}
	// P25 (ADP/DES/AES) covers the common land-mobile algorithm-id namespace;
	// other protocols fall back to its hex label for an unrecognised id.
	return p25crypto.AlgName(algid)
}
