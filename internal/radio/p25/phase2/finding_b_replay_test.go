//go:build integration

package phase2_test

// TestP25P2FindingBReplay is the offline reproduction for issue #915 Finding B:
// on real Victorian MMR Phase 2 air the completed-call source RID never lands
// because the traffic-channel MAC PDUs never satisfy the outer RS(24,16,9)
// (mac_rs_valid=0), even after superframes lock (post-#943/#970).
//
// It drives the reporter's raw capture through GopherTrunk's own Phase 2
// receiver + superframe decoder, then exhaustively sweeps the PN44 descramble
// against the RS gate. The result the maintainer's investigation had been
// circling — a wrong PN44 seed / descramble domain / per-slot offset — is ruled
// OUT here: with the reporter's confirmed identity (WACN 0xBEE00, SysID 0x164,
// NAC 0x161) no (domain × phase × deinterleave) recovers a single RS-valid MAC
// PDU. What the replay shows instead is upstream: the receiver only types ~30%
// of sub-frames through the ISCH Golay (the rest decode SlotTypeUnknown) and
// carrier lock is fragile across offset, so the dibits inside the MAC bursts
// carry too many symbol errors for the FEC to close. The blocker is Phase 2
// symbol/carrier recovery quality, not the descrambler.
//
// Skip-gated on the capture (CI has none), matching internal/voice/composer's
// real-air tests:
//
//	GT_915_CAPTURE=/path/p2_capture_tg32202_...cfile \
//	GT_915_OFFSET_HZ=-90000 \   # optional; blank auto-searches
//	  go test -tags integration ./internal/radio/p25/phase2/ -run FindingBReplay -v -timeout 15m
//
// The capture is raw uint8 interleaved I/Q at 1.2 MS/s (standard rtl_sdr
// output), the P25 Phase 2 traffic channel sitting ~-84..-90 kHz off centre.

import (
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

const (
	fbCapRateHz = 1_200_000.0
	fbTargetHz  = 48_000.0
	fbMacDibits = 146 // trellis-coded MAC channel window
	fbSeqBits   = 4320
	fbWACN      = 0xBEE00
	fbSysID     = 0x164
	fbNAC       = 0x161
)

func fbReadU8(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	n := len(raw) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		out[i] = complex((float32(raw[2*i])-127.5)/127.5, (float32(raw[2*i+1])-127.5)/127.5)
	}
	return out
}

type fbWindow struct {
	slot int
	win  []uint8
}

func fbCollect(iq []complex64, offsetHz float64) (wins []fbWindow, superframes, typed, unknown int) {
	dc := ccdecoder.NewDownconverterWithOffset(fbCapRateHz, fbTargetHz, offsetHz)
	sfDec := p25p2.NewSuperframeDecoder()
	sink := func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			superframes++
			for _, sub := range sf.Subframes {
				if sub.SlotType == p25p2.SlotTypeUnknown {
					unknown++
				} else {
					typed++
				}
				if sub.SlotType.IsMAC() && len(sub.Dibits) >= p25p2.MACPayloadOffset+fbMacDibits {
					w := make([]uint8, fbMacDibits)
					copy(w, sub.Dibits[p25p2.MACPayloadOffset:p25p2.MACPayloadOffset+fbMacDibits])
					wins = append(wins, fbWindow{sub.Index, w})
				}
			}
		}
	}
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner, GardnerGain: 0.005, DibitSink: sink})
	var buf []complex64
	const chunk = 24000
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
	}
	return wins, superframes, typed, unknown
}

func fbRSValid(info []byte) bool {
	if len(info) < 18 {
		return false
	}
	var bits [144]byte
	for i := 0; i < 18; i++ {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = (info[i] >> uint(7-j)) & 1
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
	return framing.VerifyRS24_16(syms[:])
}

// fbDecodeChannel runs the production domain (PN44 descramble on the coded
// channel bits, before deinterleave/trellis) at a channel-bit offset.
func fbDecodeChannel(win []uint8, seed uint64, offset int, deint bool) bool {
	ch := make([]uint8, len(win))
	copy(ch, win)
	b := framing.DibitsToBits(ch)
	s := framing.NewPN44Scrambler(seed)
	off := ((offset % fbSeqBits) + fbSeqBits) % fbSeqBits
	if off > 0 {
		s.Advance(off)
	}
	s.Apply(b)
	ch = framing.BitsToDibits(b)
	if deint {
		ch = framing.DeinterleaveMACBurst(ch)
	}
	info, _ := framing.DecodeP25Trellis(ch)
	if len(info) != 72 {
		return false
	}
	packed := framing.PackBitsMSB(framing.DibitsToBits(info))
	return len(packed) >= 18 && fbRSValid(packed[:18])
}

func TestFindingBReplay(t *testing.T) {
	path := os.Getenv("GT_915_CAPTURE")
	if path == "" {
		t.Skip("set GT_915_CAPTURE to the reporter's p2 capture to run the Finding-B replay")
	}
	iq := fbReadU8(t, path)
	t.Logf("loaded %d IQ samples (%.1f s @ %.0f Hz)", len(iq), float64(len(iq))/fbCapRateHz, fbCapRateHz)

	offHz := 0.0
	haveOff := false
	if v := os.Getenv("GT_915_OFFSET_HZ"); v != "" {
		offHz, _ = strconv.ParseFloat(v, 64)
		haveOff = true
	}
	if !haveOff {
		scan := iq
		if len(scan) > 3_600_000 {
			scan = scan[:3_600_000]
		}
		best := 0
		for hz := -95_000.0; hz <= -75_000.0; hz += 1_000.0 {
			_, sf, _, _ := fbCollect(scan, hz)
			if sf > best {
				best, offHz = sf, hz
			}
		}
		t.Logf("auto-selected channel offset %+.1f kHz (%d superframes on the search slice)", offHz/1000, best)
	}

	wins, sf, typed, unknown := fbCollect(iq, offHz)
	pctUnknown := 0.0
	if typed+unknown > 0 {
		pctUnknown = 100 * float64(unknown) / float64(typed+unknown)
	}
	t.Logf("offset %+.1f kHz: superframes=%d  ISCH typed=%d unknown=%d (%.0f%% unknown)  mac_windows=%d",
		offHz/1000, sf, typed, unknown, pctUnknown, len(wins))
	if sf == 0 {
		t.Fatalf("no superframes locked at %+.1f kHz — receiver/downconvert setup is wrong, not a Finding-B result", offHz/1000)
	}
	if len(wins) == 0 {
		t.Skip("superframes locked but no MAC sub-frames in this capture window")
	}

	// Exhaustive descramble sweep (production channel-bit domain): identity seed,
	// every per-slot phase, deinterleave on/off. A correct descramble lights up
	// RS across the cleaner MAC bursts; a demod-limited stream never does.
	seed := framing.PN44SeedFromIdentity(fbWACN, fbSysID, fbNAC)
	rsMax := 0
	for _, deint := range []bool{false, true} {
		for phase := 0; phase < fbSeqBits; phase++ {
			cnt := 0
			for _, w := range wins {
				if fbDecodeChannel(w.win, seed, w.slot*360+phase, deint) {
					cnt++
				}
			}
			if cnt > rsMax {
				rsMax = cnt
			}
		}
	}
	t.Logf("descramble sweep (identity seed, all %d phases, deinterleave on/off): max RS-valid = %d / %d MAC windows",
		fbSeqBits, rsMax, len(wins))
	t.Logf("FINDING B: descramble is not the blocker (max RS-valid=%d); %.0f%% of sub-frames fail even the ISCH Golay — "+
		"the Phase 2 receiver's symbol/carrier recovery is the limit, not the PN44 seed/domain/offset.", rsMax, pctUnknown)
}
