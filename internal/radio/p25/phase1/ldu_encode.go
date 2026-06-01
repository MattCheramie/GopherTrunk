package phase1

import "fmt"

// AssembleLDU builds a complete 1728-bit on-air LDU bit stream from its
// constituent fields — the inverse of ExtractVoiceFrames /
// ExtractLCESBlocks / ExtractLSDBlocks. It writes the 48-bit frame sync
// word and the 64-bit NID for (nac, duid), places the 9 voice channel
// subframes, the 6 LC/ES blocks, and the 2 LSD blocks into the 1680-bit
// payload at their TIA-102.BAAA offsets, then interleaves zero-valued
// status symbols via InjectStatusSymbols.
//
// Each voice subframe must be LDUVoiceSubframeBits on-air channel bits
// — the fully-encoded IMBE wire form (per-vector FEC + §7.4 scramble +
// §7.5 interleave), e.g. from imbe.EncodeFrameToChannel. This is the
// transmission order ExtractVoiceFrames + imbe.DecodeChannelToFrame
// consume, so the round-trip exercises the real deinterleave. A nil
// LC/ES or LSD block is written as zeros. This is synthesized-fixture
// scaffolding (GopherTrunk is receive-only); it is the encoder the
// round-trip tests pair with the extractors.
func AssembleLDU(nac uint16, duid DUID, voice [LDUVoiceSubframeCount][]byte, lces [LDULCESBlockCount][]byte, lsd [LDULSDBlockCount][]byte) ([]byte, error) {
	payload := make([]byte, LDUPayloadBits)

	copy(payload[lduFSOffset:lduFSOffset+LDUFrameSyncBits], FrameSyncBits())
	copy(payload[lduNIDOffset:lduNIDOffset+LDUNIDBits], EncodeNIDBits(nac, duid))

	for i, off := range lduVoiceOffsets {
		v := voice[i]
		if len(v) != LDUVoiceSubframeBits {
			return nil, fmt.Errorf("p25/phase1: AssembleLDU voice subframe %d must be %d bits, got %d",
				i, LDUVoiceSubframeBits, len(v))
		}
		copy(payload[off:off+LDUVoiceSubframeBits], v)
	}
	for j, off := range lduLCESBlockOffsets {
		b := lces[j]
		if b == nil {
			continue
		}
		if len(b) != LDULCESBlockBits {
			return nil, fmt.Errorf("p25/phase1: AssembleLDU LC/ES block %d must be %d bits, got %d",
				j, LDULCESBlockBits, len(b))
		}
		copy(payload[off:off+LDULCESBlockBits], b)
	}
	for k, off := range lduLSDBlockOffsets {
		b := lsd[k]
		if b == nil {
			continue
		}
		if len(b) != LDULSDBlockBits {
			return nil, fmt.Errorf("p25/phase1: AssembleLDU LSD block %d must be %d bits, got %d",
				k, LDULSDBlockBits, len(b))
		}
		copy(payload[off:off+LDULSDBlockBits], b)
	}

	var status [LDUStatusSymbolCount]uint8
	return InjectStatusSymbols(payload, status)
}
