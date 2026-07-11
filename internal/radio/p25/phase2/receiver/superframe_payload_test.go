package receiver_test

// Regression test for issue #813: on real P25 Phase 2 traffic the ALGID/KID
// (algorithm_id/key_id) never populated even when superframes locked, because
// the receiver emitted dibits in demod.DQPSK.Decode's convention (the two
// negative-phase symbols 2↔3 transposed relative to TIA-102.BBAC). Sync could
// be made to lock by matching that transposed convention, but the MAC PDU
// FEC (ISCH Golay + trellis) is written against canonical TIA-102 dibits, so
// the payload never decoded. The fix canonicalises the receiver's dibit output
// (canonicalDibitRemap, mirroring the verified Phase 1 CQPSK lsmDibitRemap).
//
// This test is deliberately INDEPENDENT of GopherTrunk's own Phase 2 modulator:
// it drives a real MAC_PTT-borne Encryption Sync through the production receiver
// via demod.ModulateHDQPSKSpec — the standard π/4-DQPSK-family phase map a real
// transmitter uses — and asserts the ALGID/KID come back out. It fails without
// the remap
// (superframes either never lock against the authoritative OutboundSyncHex, or,
// if the transposed sync constant is restored, the MAC trellis never decodes)
// and passes with it.

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
)

// leadInDibits returns n deterministic varied dibits so the receiver's coarse
// carrier seed and Gardner timing loop warm up before the first superframe sync.
func leadInDibits(n int, seed uint32) []uint8 {
	out := make([]uint8, n)
	x := seed | 1
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = uint8(x & 0x3)
	}
	return out
}

func TestReceiverRecoversEncryptionSyncPayload(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.20
	)

	// A MAC_PTT message carrying an Encryption Sync with a distinctive
	// ALGID/KID (the operationally useful fields of issue #813).
	const wantAlg, wantKey = 0x84, 0x1234
	es := phase2.EncryptionSync{
		AlgorithmID:      wantAlg,
		KeyID:            wantKey,
		MessageIndicator: [9]byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	ptt := phase2.EncodePushToTalk(es)
	if b := phase2.AssembleMACPDU(ptt); len(b) < 18 {
		ptt.Payload = append(ptt.Payload, make([]byte, 18-len(b))...) // pad to 144-bit MAC PDU
	}

	// One superframe: MAC_PTT-slot PDU at sub-frame 0 (the sync-bearing
	// sub-frame; sync occupies dibits [0,20), the ISCH+MAC follow), voice
	// elsewhere. EncodeSuperframe injects the outbound sync.
	vpay := make([][]byte, phase2.VoiceFrameCount(phase2.SlotTypeVoice4V))
	for i := range vpay {
		vpay[i] = make([]byte, phase2.VoiceFrameBytes)
	}
	var subs [phase2.SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = phase2.EncodeMACSubframe(phase2.SlotTypeMACPTT, uint8(i), ptt,
				phase2.TrellisOn, phase2.InterleaveOff)
		} else {
			subs[i] = phase2.EncodeVoiceSubframe(phase2.SlotTypeVoice4V, uint8(i), vpay)
		}
	}
	superframe := phase2.EncodeSuperframe(subs)
	// Overwrite the sync region with the AUTHORITATIVE standard outbound sync
	// (0x575D57F7FF), independent of OutboundSyncHex, so the stimulus is exactly
	// what a real base station transmits. This isolates the payload bug: with
	// the pre-fix transposed OutboundSyncHex and no remap, sync still LOCKS
	// (the transposed constant matches the un-remapped decoder) yet the MAC
	// trellis never decodes — so this test fails specifically on ALGID/KID, not
	// on sync. hexToDibitsMSB / specSyncHex are shared with the specsync test.
	base := phase2.SyncSubframeIndex * phase2.DibitsPerSubframe
	copy(superframe[base:base+phase2.SyncDibits], hexToDibitsMSB(specSyncHex, phase2.SyncDibits))

	// Lead-in (past the coarse-seed budget) + several superframes so at least
	// one arrives after warmup.
	dibits := leadInDibits(800, 0xB813)
	for n := 0; n < 6; n++ {
		dibits = append(dibits, superframe...)
	}

	iq := demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)

	var got []uint8
	r := rx.New(rx.Options{
		SampleRateHz: sps * rx.SymbolRate,
		ClockMode:    rx.ClockGardner,
		GardnerGain:  0.005, // the value the voice/CC paths use
		DibitSink:    func(d []uint8, _ int) { got = append(got, d...) },
	})
	for off := 0; off < len(iq); off += 192 { // production-sized chunks
		end := off + 192
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[off:end])
	}

	sfs := phase2.NewSuperframeDecoder().Process(got, 0)
	if len(sfs) == 0 {
		t.Fatalf("no superframes locked from standard-modulated air "+
			"(OutboundSyncHex=0x%X); receiver dibit convention does not match real air (issue #813)",
			phase2.OutboundSyncHex)
	}

	cfg := phase2.MACDecodeConfig{Trellis: phase2.TrellisOn}
	var gotAlg, gotKey int = -1, -1
	for _, sf := range sfs {
		for _, tagged := range phase2.DecodeSuperframeMACPDUsWithSlot(sf, cfg) {
			if tagged.SlotType != phase2.SlotTypeMACPTT {
				continue
			}
			if es2, ok := tagged.PDU.AsPushToTalk(); ok {
				gotAlg, gotKey = int(es2.AlgorithmID), int(es2.KeyID)
			}
		}
	}
	if gotAlg != wantAlg || gotKey != wantKey {
		t.Fatalf("ALGID/KID not recovered from standard-modulated Phase 2 air: "+
			"got alg=%#x key=%#x, want alg=%#x key=%#x (issue #813: MAC payload decoded 2↔3 transposed)",
			gotAlg, gotKey, wantAlg, wantKey)
	}
}
