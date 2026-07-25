//go:build integration

package phase2_test

// TestP25P2GroundTruthReplay is the ground-truth offline reproduction for
// issue #915: on real Victorian MMR (WACN 0xBEE00, SysID 0x164, NAC 0x161)
// Phase 2 air the completed-call source RID never lands because the
// traffic-channel MAC PDUs never satisfy the outer RS(24,16,9) — the demod
// does not recover clean enough symbols for the FEC to close.
//
// It is the successor to TestP25P2FindingBReplay (PR #979). That harness drove
// the reporter's first capture — which had NO simultaneous SDRTrunk decode — and
// could only measure demod *health* (superframes locked, ISCH-unknown %,
// max RS-valid), ruling the descrambler OUT and pinning the blocker to symbol/
// carrier recovery. What it could not do was verify a demod change actually
// recovers a *correct* source RID, because there was no known-good answer.
//
// The reporter then captured a second file WITH ground truth: 120 s of wideband
// IQ over the MMR voice channels, plus a JSON sidecar of 16 SDRTrunk-decoded
// source RIDs (the clear MAC_PTT / GROUP_VOICE_CHANNEL_USER field), each tagged
// with its offset into the capture, TGID, voice frequency and NAC. That turns
// #915 into a closed-loop check: for each grant, tune the voice channel, run it
// through GopherTrunk's *own* Phase 2 receiver + superframe decoder + MAC FEC,
// and ask whether GT recovers the RID SDRTrunk already read off the same air.
//
// This is the "known-good RID target ... failing-first check against your bytes"
// the maintainer said was the one missing measurement to iterate the demod
// against. It is skip-gated on the capture (CI has none), matching the
// finding-B harness and internal/voice/composer's real-air tests:
//
//	GT_915_GT_CAPTURE=/path/p2_gt_..._120s.cfile \
//	GT_915_GT_JSON=/path/p2_gt_..._ground_truth.json \
//	  go test -tags integration ./internal/radio/p25/phase2/ \
//	    -run GroundTruthReplay -v -timeout 30m
//
// Capture format: complex float32 (cf32) interleaved I/Q at 2.048 MS/s,
// centred on capture_center_hz. Each grant's voice channel sits at
// (freq_hz − capture_center_hz) plus a small residual (the Airspy's tuner
// offset), so the harness searches a ±kHz DDC window around the nominal
// offset and keeps the strongest superframe lock — the receiver's own
// carrier recovery then pulls in the fine residual.
//
// Current baseline (2.048 MS/s ground-truth capture, 12 in-file grants): the
// receiver locks a handful of superframes on the stronger channels but recovers
// 0 RS-valid MAC PDUs and 0 source RIDs — ~46–62 % of sub-frames fail even the
// ISCH Golay where they lock at all, and the weaker channels do not lock. The
// demod symbol/carrier recovery is the wall, exactly as PR #979 pinned. This
// harness makes that a per-grant, one-number metric (rids_recovered / grants)
// that a demod fix has to move off zero.

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

const (
	gtTargetHz  = 48_000.0
	gtMacDibits = 146 // trellis-coded MAC channel window
	gtSeqBits   = 4320
	// MMR identity confirmed by the reporter from GT's NET_STS_BCST
	// (issue #915): WACN 0xBEE00 / SysID 0x164 / NAC 0x161.
	gtWACN  = 0xBEE00
	gtSysID = 0x164
	gtNAC   = 0x161
)

// gtGroundTruth mirrors the reporter's ground_truth.json sidecar.
type gtGroundTruth struct {
	CaptureCenterHz int       `json:"capture_center_hz"`
	SampleRateHz    float64   `json:"sample_rate_hz"`
	Format          string    `json:"format"`
	Grants          []gtGrant `json:"grants_in_band"`
}

type gtGrant struct {
	OffsetS   float64 `json:"offset_s"`
	TGID      int     `json:"tgid"`
	SourceRID uint32  `json:"source_rid"`
	FreqHz    int     `json:"freq_hz"`
	NAC       int     `json:"nac"`
}

// gtReadCF32Window reads a [startS, startS+durS) window of complex float32
// interleaved I/Q from the capture, seeking so a 120 s file need not be held
// in memory. Returns fewer samples near EOF.
func gtReadCF32Window(t *testing.T, path string, rateHz, startS, durS float64) []complex64 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open capture: %v", err)
	}
	defer f.Close()
	start := int64(startS * rateHz)
	if start < 0 {
		start = 0
	}
	n := int64(durS * rateHz)
	if _, err := f.Seek(start*8, 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	raw := make([]byte, n*8)
	got, _ := f.Read(raw)
	raw = raw[:got]
	out := make([]complex64, len(raw)/8)
	for i := range out {
		re := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
		im := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8+4:]))
		out[i] = complex(re, im)
	}
	return out
}

type gtMacWindow struct {
	slot int
	win  []uint8
}

type gtCollect struct {
	superframes int
	typed       int
	unknown     int
	windows     []gtMacWindow
}

// gtRun drives one IQ window through the DDC + Phase 2 receiver + superframe
// decoder at a fixed DDC offset, harvesting every MAC sub-frame window and the
// ISCH typing counts.
func gtRun(iq []complex64, rateHz, offsetHz float64) gtCollect {
	dc := ccdecoder.NewDownconverterWithOffset(rateHz, gtTargetHz, offsetHz)
	sfDec := p25p2.NewSuperframeDecoder()
	var res gtCollect
	sink := func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			res.superframes++
			for _, sub := range sf.Subframes {
				if sub.SlotType == p25p2.SlotTypeUnknown {
					res.unknown++
				} else {
					res.typed++
				}
				if sub.SlotType.IsMAC() && len(sub.Dibits) >= p25p2.MACPayloadOffset+gtMacDibits {
					w := make([]uint8, gtMacDibits)
					copy(w, sub.Dibits[p25p2.MACPayloadOffset:p25p2.MACPayloadOffset+gtMacDibits])
					res.windows = append(res.windows, gtMacWindow{sub.Index, w})
				}
			}
		}
	}
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner, GardnerGain: 0.005, DibitSink: sink})
	var buf []complex64
	const chunk = 8192
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
	}
	return res
}

// gtDecodeSource runs the production MAC recovery (channel-bit PN44 descramble
// at a per-slot phase → deinterleave → trellis → RS gate → parse) over a MAC
// window and returns the GROUP_VOICE_CHANNEL_USER source RID if one RS-validates.
func gtDecodeSource(win []uint8, seed uint64, phase int) (rid uint32, rsValid, isSource bool) {
	ch := make([]uint8, len(win))
	copy(ch, win)
	b := framing.DibitsToBits(ch)
	s := framing.NewPN44Scrambler(seed)
	off := ((phase % gtSeqBits) + gtSeqBits) % gtSeqBits
	if off > 0 {
		s.Advance(off)
	}
	s.Apply(b)
	ch = framing.BitsToDibits(b)
	ch = framing.DeinterleaveMACBurst(ch)
	info, _ := framing.DecodeP25Trellis(ch)
	if len(info) != 72 {
		return 0, false, false
	}
	packed := framing.PackBitsMSB(framing.DibitsToBits(info))
	if len(packed) < 18 || !gtVerifyRS(packed[:18]) {
		return 0, false, false
	}
	pdu, err := p25p2.ParseMACPDU(packed[:18])
	if err != nil {
		return 0, true, false
	}
	if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
		return u.SourceID, true, true
	}
	return 0, true, false
}

func gtVerifyRS(pdu []byte) bool {
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
	return framing.VerifyRS24_16(syms[:])
}

func TestGroundTruthReplay(t *testing.T) {
	capPath := os.Getenv("GT_915_GT_CAPTURE")
	jsonPath := os.Getenv("GT_915_GT_JSON")
	if capPath == "" || jsonPath == "" {
		t.Skip("set GT_915_GT_CAPTURE and GT_915_GT_JSON to run the #915 ground-truth replay")
	}

	jb, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read ground truth: %v", err)
	}
	var gt gtGroundTruth
	if err := json.Unmarshal(jb, &gt); err != nil {
		t.Fatalf("parse ground truth: %v", err)
	}
	rateHz := gt.SampleRateHz
	if rateHz <= 0 {
		rateHz = 2_048_000.0
	}
	if gt.CaptureCenterHz == 0 {
		t.Fatalf("ground truth missing capture_center_hz")
	}
	t.Logf("ground truth: center=%.4f MHz rate=%.0f Hz format=%s grants=%d",
		float64(gt.CaptureCenterHz)/1e6, rateHz, gt.Format, len(gt.Grants))

	seed := framing.PN44SeedFromIdentity(gtWACN, gtSysID, gtNAC)

	// A grant on a channel at (freq−center) with an unknown small tuner
	// residual. Search a ±kHz DDC window around the nominal offset and keep
	// the strongest superframe lock; the receiver's carrier loop pulls in the
	// fine residual from there.
	const (
		searchLoHz   = -18_000.0
		searchHiHz   = +6_000.0
		searchStepHz = 2_000.0
		windowPreS   = 1.0 // seconds of IQ to load before the grant offset
		windowDurS   = 6.0 // total window length
	)

	var (
		grants       int
		anyLock      int
		ridsRecov    int
		rsValidTotal int
	)
	for _, g := range gt.Grants {
		iq := gtReadCF32Window(t, capPath, rateHz, g.OffsetS-windowPreS, windowDurS)
		if len(iq) < int(rateHz*0.5) {
			continue // grant past the end of a shorter-than-advertised capture
		}
		grants++
		nominal := float64(g.FreqHz - gt.CaptureCenterHz)

		bestSF, bestUnkPct := 0, 100.0
		grantRS, grantRID := 0, 0
		for d := searchLoHz; d <= searchHiHz; d += searchStepHz {
			res := gtRun(iq, rateHz, nominal+d)
			if res.superframes == 0 {
				continue
			}
			rs, hit := 0, 0
			for _, w := range res.windows {
				for slot := 0; slot < p25p2.SubframesPerSuperframe; slot++ {
					rid, ok, isSrc := gtDecodeSource(w.win, seed, slot*360)
					if ok {
						rs++
						if isSrc && rid == g.SourceRID {
							hit++
						}
						break
					}
				}
			}
			grantRS += rs
			grantRID += hit
			if res.superframes > bestSF {
				bestSF = res.superframes
				bestUnkPct = 100.0 * float64(res.unknown) / float64(res.typed+res.unknown+1)
			}
		}
		if bestSF > 0 {
			anyLock++
		}
		rsValidTotal += grantRS
		recov := grantRID > 0
		if recov {
			ridsRecov++
		}
		t.Logf("grant t=%6.2fs tg=%-6d rid=%-8d f=%.4fMHz nom=%+.0fkHz | bestSF=%d unk=%.0f%% rs_valid=%d rid_recovered=%v",
			g.OffsetS, g.TGID, g.SourceRID, float64(g.FreqHz)/1e6, nominal/1000, bestSF, bestUnkPct, grantRS, recov)
	}

	if grants == 0 {
		t.Fatalf("no in-file grants — capture shorter than expected or JSON offsets wrong")
	}
	if anyLock == 0 {
		t.Fatalf("no superframes locked on ANY grant — DDC/receiver setup is wrong, not a #915 result")
	}
	t.Logf("=== #915 ground-truth demod baseline ===")
	t.Logf("grants replayed:      %d", grants)
	t.Logf("grants with SF lock:  %d", anyLock)
	t.Logf("RS-valid MAC PDUs:    %d", rsValidTotal)
	t.Logf("SOURCE RIDs RECOVERED: %d / %d", ridsRecov, grants)
	t.Logf("The demod fix has to move rids_recovered off %d. This harness is the", ridsRecov)
	t.Logf("failing-first check against the reporter's real bytes (issue #915).")
}
