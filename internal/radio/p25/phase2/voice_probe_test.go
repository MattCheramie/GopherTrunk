//go:build integration

package phase2_test

// Real-air check on the voice-burst geometry. A voice codeword's first two
// fields are Golay-protected, so a correct descramble and offset show up as
// near-zero corrected bits, while a wrong one leaves the decoder correcting
// at its radius on nearly every frame. The old geometry is run alongside as
// the control.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run VoiceGeometry -v

import (
	"os"
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
)

func TestVoiceGeometry(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))
	sfDec := p25p2.NewSuperframeDecoder()
	wd := p25p2.NewCarrierWatchdog(0)
	sfCount = 0

	var voiceBursts, frames, uncorrectable int
	var errHist []int
	// Control: the pre-fix geometry — an even 36-dibit grid from dibit 32,
	// with no descramble at all.
	var ctlFrames, ctlUncorrectable int
	var ctlErrHist []int

	probeDibitsWD(iq, rate, wd, sfDec, func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			sfCount++
			// Resolve the superframe's slot offset by vote over its ACCH
			// bursts, the same way the production path does.
			best, bestScore := 0, -1
			for _, cand := range []int{2, 3, 6, 7, 10, 11} {
				n := 0
				for _, sub := range sf.Subframes {
					if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsACCH() {
						continue
					}
					if _, ok := p25p2.DecodeACCHBurst(sub.Dibits, (sub.Index+cand)%12, seq); ok {
						n++
					}
				}
				if n > bestScore {
					best, bestScore = cand, n
				}
			}
			if bestScore <= 0 {
				continue
			}
			for _, sub := range sf.Subframes {
				if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsVoice() {
					continue
				}
				voiceBursts++
				slot := (sub.Index + best) % 12
				fr, errs, unc, err := p25p2.ExtractBurstVoiceFrames(sub.Dibits, slot, seq)
				if err != nil {
					continue
				}
				frames += len(fr)
				uncorrectable += unc
				errHist = append(errHist, errs)

				// Control: old offsets, no descramble.
				for i := 0; i < len(fr); i++ {
					off := 32 + i*p25p2.VoiceCodewordDibits
					if off+p25p2.VoiceCodewordDibits > p25p2.BurstDibits {
						break
					}
					_, e, cerr := p25p2.DecodeVoiceCodeword(sub.Dibits[off : off+p25p2.VoiceCodewordDibits])
					ctlFrames++
					if cerr != nil {
						ctlUncorrectable++
					} else {
						ctlErrHist = append(ctlErrHist, e)
					}
				}
			}
		}
	})

	median := func(v []int) int {
		if len(v) == 0 {
			return -1
		}
		sort.Ints(v)
		return v[len(v)/2]
	}
	t.Logf("voice bursts=%d frames=%d", voiceBursts, frames)
	t.Logf("  FIXED   : median corrected bits/burst=%d over %d frames", median(errHist), frames)
	t.Logf("  CONTROL : uncorrectable=%d/%d (%.0f%%)  median corrected bits/frame=%d",
		ctlUncorrectable, ctlFrames, 100*float64(ctlUncorrectable)/float64(max(ctlFrames, 1)), median(ctlErrHist))
}
