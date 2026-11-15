package dstar

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// On-air encoders for D-STAR, for fixtures and the synthetic signal
// generators. They build the format MMDVM transmits (DStarTX.cpp),
// and are pinned to that firmware's own output rather than to this
// package's decoders: framing's header vectors, and the null AMBE frame
// (voice_null_test.go). Bits are in time order and MMDVM's polarity; a
// receiver may see them complemented (Polarity).

// SealHeader returns h with its CRC set over the other fields.
func SealHeader(h Header) Header {
	h.CRC = ComputeCRC(AssembleHeader(h)[:39])
	return h
}

// EncodeHeaderBurst returns the frame sync pattern (FrameSyncBitsSlice,
// which opens with the end of the bit-sync preamble) followed by the
// 660-bit header FEC section, for h as given (call SealHeader first for
// a valid CRC). A transmission is BitSyncBits, this burst, DV frames,
// then EndSyncBitsSlice.
func EncodeHeaderBurst(h Header) []uint8 {
	out := append([]uint8(nil), FrameSyncBitsSlice()...)
	return append(out, framing.EncodeDStarHeaderFEC(AssembleHeader(h))...)
}

// EncodeDVFrame returns one 96-bit DV frame: 72 on-air voice bits (as
// EncodeDVVoiceBits returns them) and the slow-data field of frame n of
// the transmission (n counts from 0, the first frame after the header).
// Frame n % 21 == 0 carries the slow-data sync; the others carry slow
// (3 bytes), scrambled with the first 24 bits of D-STAR's PN sequence,
// as MMDVMHost's DStarSlowData does.
func EncodeDVFrame(voice []uint8, n int, slow [3]byte) []uint8 {
	out := make([]uint8, 0, DVFrameBits)
	out = append(out, voice[:DVVoiceBits]...)
	if n%DVFramesPerSyncCycle == 0 {
		return append(out, SlowDataSyncBitsSlice()...)
	}
	return append(out, scrambleSlowData(BytesToBitsLSB(slow[:]))...)
}

// slowDataPN scrambles a DV frame's 24 slow-data bits.
var slowDataPN = framing.DStarPN(DVDataBits)

// scrambleSlowData XORs 24 slow-data bits with slowDataPN (its own
// inverse).
func scrambleSlowData(b []uint8) []uint8 {
	out := make([]uint8, len(b))
	for i := range b {
		out[i] = b[i] ^ slowDataPN[i]
	}
	return out
}

// NullAMBEFrame returns the on-air bits of the AMBE frame MMDVMHost and
// the MMDVM firmware send for silence (DSTAR_NULL_AMBE_DATA_BYTES).
func NullAMBEFrame() []uint8 {
	return BytesToBitsLSB([]byte{0x9E, 0x8D, 0x32, 0x88, 0x26, 0x1A, 0x3F, 0x61, 0xE8})
}
