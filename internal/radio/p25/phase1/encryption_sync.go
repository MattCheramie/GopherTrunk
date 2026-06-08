package phase1

import (
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 LDU2 Encryption Sync word.
//
// An LDU2 carries an Encryption Sync (ES) word in the same 6 × 40-bit
// LC/ES slots an LDU1 uses for Link Control — so it is protected by the
// identical 24-codeword shortened Hamming(10,6,3) inner FEC, decoded
// here by lcInnerDecode (link_control.go). The ES identifies the
// privacy parameters of an encrypted voice call:
//
//   - MessageIndicator: the 72-bit per-call cryptographic sync vector
//   - AlgorithmID: the encryption algorithm (0x80 = unencrypted/clear,
//     0x81 = DES-OFB, 0x84 = AES-256, … per the TIA algorithm registry)
//   - KeyID: which key in the radio's keyset the call uses
//
// Like SDRtrunk, GopherTrunk identifies encryption — it surfaces the
// algorithm and key — but does not decrypt.
//
// The 96-bit ES content layout is the project's working model (the
// 144-bit Hamming-recovered field's first 12 octets). As for Link
// Control, the inner Hamming layer is followed by an outer RS code —
// here RS(24,16,9) (framing.DecodeRS24_16, t=4) — which ParseEncryptionSync
// runs to correct the residual symbol errors that otherwise produce
// garbage Algorithm IDs under marginal SNR.
//
//	octets 0-8 : Message Indicator (72 bits)
//	octet 9    : Algorithm ID
//	octets 10-11: Key ID
type EncryptionSync struct {
	MessageIndicator [9]byte
	AlgorithmID      uint8
	KeyID            uint16
}

// esContentOctets is the ES content size in octets (96 bits).
const esContentOctets = 12

// AlgorithmClear is the Algorithm ID a clear (unencrypted) call
// advertises in its Encryption Sync.
const AlgorithmClear uint8 = 0x80

// Encrypted reports whether the ES describes an encrypted call (any
// Algorithm ID other than the clear-voice value).
func (e EncryptionSync) Encrypted() bool { return e.AlgorithmID != AlgorithmClear }

// ErrEncryptionSyncUncorrectable is returned when the outer RS(24,16,9)
// layer cannot correct the Encryption Sync word — more than t=4 symbol
// errors survived the inner Hamming layer. The decoded algorithm/key are
// then low-confidence and should not be surfaced as a real encryption
// change.
var ErrEncryptionSyncUncorrectable = errors.New("p25/phase1: Encryption Sync RS-uncorrectable")

// ParseEncryptionSync decodes the 6 LC/ES blocks of an LDU2 into a
// structured EncryptionSync. The inner Hamming(10,6,3) layer is decoded
// first, then the outer RS(24,16,9) layer corrects the residual symbol
// errors that otherwise produce garbage Algorithm IDs under marginal
// SNR. It returns the total corrected-error count and
// ErrEncryptionSyncUncorrectable when the RS layer cannot recover the word.
func ParseEncryptionSync(blocks [LDULCESBlockCount][]byte) (EncryptionSync, int, error) {
	data, innerErrs, err := lcInnerDecode(blocks)
	if err != nil {
		return EncryptionSync{}, 0, err
	}
	cw := bitsToRSSymbols(data)
	info, rsErrs, rerr := framing.DecodeRS24_16(cw[:])
	if rerr != nil {
		return EncryptionSync{}, innerErrs, ErrEncryptionSyncUncorrectable
	}
	oct := bitsToOctets(rsSymbolsToBits(info[:]))
	var es EncryptionSync
	copy(es.MessageIndicator[:], oct[0:9])
	es.AlgorithmID = oct[9]
	es.KeyID = uint16(oct[10])<<8 | uint16(oct[11])
	return es, innerErrs + rsErrs, nil
}

// AssembleEncryptionSync is the inverse of ParseEncryptionSync; it
// builds the 6 on-wire ES blocks, computing the outer RS(24,16,9) parity
// so the word round-trips through the RS-correcting parser.
func AssembleEncryptionSync(es EncryptionSync) [LDULCESBlockCount][]byte {
	oct := make([]byte, esContentOctets)
	copy(oct[0:9], es.MessageIndicator[:])
	oct[9] = es.AlgorithmID
	oct[10], oct[11] = byte(es.KeyID>>8), byte(es.KeyID)

	return lcInnerEncode(rsEncodeContentBits(octetsToBits(oct), 16))
}
