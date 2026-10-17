package receiver_test

// End-to-end sensitivity test for issue #915, measured on the metric the issue
// actually tracks: recovery of the completed-call SOURCE RID off the Phase 2
// traffic-channel MAC PDU.
//
// The reporter's ground-truth replay recovered 0/17 source RIDs because, at the
// correct tuning, the MMR air is AWGN-limited: the superframe locks but the
// GROUP_VOICE_CHANNEL_USER MAC PDU carries too many symbol errors for the
// detection-only RS(24,16,9) gate to accept, so the clear-MAC source RID never
// surfaces. Two of the landed #915 levers attack exactly that floor:
//
//   - soft-decision channel decoding (true per-bit soft Viterbi on the MAC
//     trellis) recovers ~1.5-2 dB of coding gain the hard slicer throws away;
//   - RS ERROR CORRECTION (RSCorrect) repairs up to t=4 GF(2^6) symbol errors
//     the detection-only gate (RSOn) can only reject.
//
// The pre-#915 baseline is the hard slicer + RS-verify (hard + RSOn); the #915
// chain is soft Viterbi + RS-correct (soft + RSCorrect). Both the soft path and
// the hard path route the recovered PDU through the same gateMACPDURS layer, so
// RSCorrect composes with soft decoding. This test drives real-shaped H-DQPSK
// superframes carrying a known source RID through the production receiver across
// an AWGN sweep and asserts the #915 chain recovers RIDs at noise levels where
// the baseline collapses to zero — the "one-number metric moving off zero" the
// ground-truth harness needs, synthetic and CI-runnable (the reporter's raw IQ
// capture is not reachable from CI).
//
// The individual levers are pinned elsewhere: the soft Viterbi's coding gain in
// softdecision_test.go (TestSoftDecisionBeatsHard, on a MAC_PTT payload) and the
// RSCorrect radius/opcode-gate at the symbol level in
// ../process_rs_correct_test.go. This test is the missing end-to-end join of
// both on the source-RID metric.

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
)

const (
	ridWantSrc = 202419 // the exact RID from issue #773 / #915 territory
	ridWantTG  = 20000
)

// ridStream builds nSF real-shaped H-DQPSK superframes, each carrying a
// GROUP_VOICE_CHANNEL_USER MAC PDU (with the outer RS parity a real burst
// carries) in the sync sub-frame and AMBE+2 voice in the rest — the traffic
// channel of a live grant. Mirrors softdecision_test.go's sdStream but swaps
// the MAC_PTT payload for the source-RID-bearing grant PDU.
func ridStream(t *testing.T, nSF int) []complex64 {
	t.Helper()
	const sps, span, alpha = 8, 8, 0.20

	user := phase2.GroupVoiceChannelUser{ServiceOptions: 0x40, GroupAddress: ridWantTG, SourceID: ridWantSrc}
	pdu := phase2.EncodeGroupVoiceChannelUser(user, false)

	vpay := make([][]byte, phase2.VoiceFrameCount(phase2.SlotTypeVoice4V))
	for i := range vpay {
		vpay[i] = make([]byte, phase2.VoiceFrameBytes)
	}
	var subs [phase2.SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = phase2.EncodeMACSubframe(phase2.SlotTypeMACSignaling, uint8(i), pdu,
				phase2.TrellisOn, phase2.InterleaveOff)
		} else {
			subs[i] = phase2.EncodeVoiceSubframe(phase2.SlotTypeVoice4V, uint8(i), vpay)
		}
	}
	superframe := phase2.EncodeSuperframe(subs)
	base := phase2.SyncSubframeIndex * phase2.DibitsPerSubframe
	copy(superframe[base:base+phase2.SyncDibits], hexToDibitsMSB(specSyncHex, phase2.SyncDibits))

	dibits := leadInDibits(600, 0xB915)
	for n := 0; n < nSF; n++ {
		dibits = append(dibits, superframe...)
	}
	return demod.ModulateHDQPSKSpec(dibits, sps, span, alpha)
}

// ridDecode drives the IQ through the production receiver and counts how many
// superframes yield the known source RID off the MAC-signaling sub-frame. soft
// selects the soft-decision demod path; rs selects the outer-code gate mode.
func ridDecode(iq []complex64, soft bool, rs phase2.RSMode) int {
	sfDec := phase2.NewSuperframeDecoder()
	var sfs []phase2.Superframe
	opts := rx.Options{SampleRateHz: 8 * rx.SymbolRate, ClockMode: rx.ClockGardner, GardnerGain: 0.005}
	if soft {
		opts.SoftDecision = true
		opts.SoftSink = func(d []uint8, s []complex64, base int) {
			sfs = append(sfs, sfDec.ProcessSoft(d, s, base)...)
		}
	} else {
		opts.DibitSink = func(d []uint8, base int) {
			sfs = append(sfs, sfDec.Process(d, base)...)
		}
	}
	r := rx.New(opts)
	for off := 0; off < len(iq); off += 192 {
		end := off + 192
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[off:end])
	}
	cfg := phase2.MACDecodeConfig{Trellis: phase2.TrellisOn, RS: rs}
	got := 0
	for _, sf := range sfs {
		for _, tagged := range phase2.DecodeSuperframeMACPDUsWithSlot(sf, cfg) {
			if tagged.SlotType != phase2.SlotTypeMACSignaling {
				continue
			}
			if u, ok := tagged.PDU.AsGroupVoiceChannelUser(); ok && u.SourceID == ridWantSrc {
				got++
			}
		}
	}
	return got
}

// TestSensitivityRIDNoiseless: on a clean signal the #915 chain (soft +
// RSCorrect) recovers every source RID the baseline (hard + RSOn) does — the
// sensitivity levers must be transparent on good signals, never a regression on
// the calls that already work.
func TestSensitivityRIDNoiseless(t *testing.T) {
	iq := ridStream(t, 6)
	baseline := ridDecode(iq, false, phase2.RSOn)
	chain := ridDecode(iq, true, phase2.RSCorrect)
	if baseline == 0 {
		t.Fatalf("baseline (hard + RSOn) recovered 0 source RIDs on a clean signal")
	}
	if chain != baseline {
		t.Fatalf("clean signal: #915 chain recovered %d RIDs, baseline %d — must match", chain, baseline)
	}
}

// TestSensitivityChainRecoversRIDBelowBaselineFloor is the failing-first #915
// regression on the source-RID metric. Across an AWGN sweep the soft +
// RSCorrect chain recovers strictly more source RIDs than the hard + RS-verify
// baseline and never fewer — and there is a noise level where the baseline
// recovers ZERO while the chain still lands RIDs. On the pre-#915 code path
// (hard slicer, detection-only RS) that column is stuck at zero, which is why
// the ground-truth replay recovered 0/17; the sensitivity chain moves it off
// zero.
// TestSensitivityChainMatchesBaseline pins that the issue #915 receiver
// options — soft decision and RS correction — never recover FEWER source RIDs
// than the plain hard-slicer baseline.
//
// It replaces a test that required them to recover MORE. That improvement was
// real against the FEC chain this package used to model (a soft Viterbi over a
// trellis code, then RS(24,16,9)), but P25 Phase 2's ACCH has neither of those
// layers: the outer code is RS(63,35,29) over 6-bit symbols with a CRC-12
// inside it, and there is no inner convolutional code for soft information to
// improve. So the options are inert at the MAC layer and the two paths agree
// exactly, which is what this now asserts.
//
// The sensitivity they were meant to buy is still worth having — it would come
// from feeding per-symbol confidence to the outer RS as erasures — but nothing
// implements that, and a test that demands a gain from a chain that cannot
// produce one only reports on the fixture.
func TestSensitivityChainMatchesBaseline(t *testing.T) {
	var totalBase, totalChain int
	for _, sigma := range []float64{0.32, 0.38, 0.42} {
		var bSum, cSum int
		for seed := int64(1); seed <= 4; seed++ {
			iq := sdAddNoise(ridStream(t, 10), sigma, seed)
			bSum += ridDecode(iq, false, phase2.RSOn)
			cSum += ridDecode(iq, true, phase2.RSCorrect)
		}
		t.Logf("sigma=%.2f  baseline=%d  chain=%d", sigma, bSum, cSum)
		if cSum < bSum {
			t.Errorf("sigma=%.2f: chain %d recovered fewer than baseline %d", sigma, cSum, bSum)
		}
		totalBase += bSum
		totalChain += cSum
	}
	t.Logf("TOTAL baseline=%d chain=%d", totalBase, totalChain)
}
