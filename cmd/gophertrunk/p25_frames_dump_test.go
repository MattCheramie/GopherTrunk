//go:build integration

package main

// Per-frame dump of GopherTrunk's P25 decode for the three-decoder comparison
// (sdrtrunk / OP25 / GopherTrunk, adjudicated by an independent oracle).
// Prints one CSV row per frame, prefixed FRAME so it can be grepped out of the
// go test chatter:
//
//	FRAME,gophertrunk,<file>,<phase>,<t_sec>,<duid>,<nac>,<valid>,<extra>
//
// Phase 1 runs the production C4FM chain exactly as internal/siglab builds it
// (receiver → ControlChannel), tapping every accepted NID through
// Options.NIDSink and every TSBK through Options.PDUSink. Phase 2 runs the
// production Phase 2 receiver + SuperframeDecoder + ACCH decode.
//
//	GT_FRAMES_P1="a.wav b.wav" GT_FRAMES_P2="c.wav" go test -tags integration \
//	  ./cmd/gophertrunk/ -run TestP25FramesDump -v | grep '^FRAME,'

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	p25p2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

const framesReceiverRate = 48000.0

func framesDuidName(d p25phase1.DUID) string {
	switch d {
	case p25phase1.DUIDTrunkingSignaling:
		return "TSBK"
	default:
		return d.String()
	}
}

func dumpP1Frames(t *testing.T, path string) {
	iq, rate := readWavIQ16(t, path)
	base := filepath.Base(path)
	dc := ccdecoder.NewDownconverterWithOffset(rate, framesReceiverRate, 0)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()

	// Mirror internal/siglab/engine_p25.go's production construction.
	cc := p25phase1.New(p25phase1.Options{
		Bus:           bus,
		Log:           log,
		SystemName:    "compare",
		FrequencyHz:   0,
		Rotations:     p25phase1.RotationsC4FM,
		NIDSearchSpan: p25phase1.NIDSearchSpan,
		NIDSink: func(r p25phase1.NIDReport) {
			valid := 1
			if r.Corroborated {
				valid = 0 // NID alone exceeded the accept bound
			}
			fmt.Printf("FRAME,gophertrunk,%s,1,%.4f,%s,%03X,%d,nid_errs=%d;rot=%d\n",
				base, float64(r.FSWStart)/p25phase1rx.SymbolRate, framesDuidName(r.DUID),
				r.NAC, valid, r.Errs, r.Rotation)
		},
		PDUSink: func(b p25phase1.SignalingBlock) {
			v := 0
			if b.CRCOK {
				v = 1
			}
			fmt.Printf("FRAME,gophertrunk,%s,1,%.4f,TSBKBLK,%03X,%d,%s;op=%02X;metric=%d\n",
				base, float64(b.DibitStart)/p25phase1rx.SymbolRate, b.NAC, v,
				strings.ToUpper(fmt.Sprintf("%x", b.RawPayload)), b.Opcode, b.FECMetric)
		},
	})
	rxOpts := p25phase1rx.Options{
		SampleRateHz: dc.OutRateHz(),
		DeviationHz:  1800, // siglab p25DeviationHz
		DemodMode:    p25phase1rx.DemodC4FM,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	}
	// Experiment knobs for the decoder comparison. GT_FRAMES_SOFT=1 engages
	// the soft-decision TSBK trellis exactly as widebandt2 does
	// (BitLLRSink → ControlChannel.StashSoft); GT_FRAMES_CLOCKGAIN overrides
	// the Mueller-Müller loop gain (default 0.05).
	if os.Getenv("GT_FRAMES_SOFT") == "1" {
		rxOpts.BitLLRSink = cc.StashSoft
	}
	rxOpts.EnableAdaptiveC4FMSlicer = os.Getenv("GT_FRAMES_ADAPTIVE") == "1"
	rxOpts.EnableDecisionDirectedAFC = os.Getenv("GT_FRAMES_DDA") == "1"
	if g := os.Getenv("GT_FRAMES_CLOCKGAIN"); g != "" {
		if v, err := strconv.ParseFloat(g, 64); err == nil {
			rxOpts.ClockGain = v
		}
	}
	rx := p25phase1rx.New(rxOpts)
	var buf []complex64
	const chunk = 8192
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		rx.Process(buf)
	}
}

func dumpP2Frames(t *testing.T, path string) {
	iq, rate := readWavIQ16(t, path)
	base := filepath.Base(path)
	// Scramble identity of the Phase 2 corpus (its own control channel's
	// NET_STATUS_BCAST).
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(0xBEE00, 0x1FC, 0x1F0))
	dc := ccdecoder.NewDownconverterWithOffset(rate, framesReceiverRate, 0)
	sfDec := p25p2.NewSuperframeDecoder()
	sink := func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			for _, sub := range sf.Subframes {
				if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsACCH() {
					continue
				}
				start := sf.StartDibit + sub.Index*p25p2.DibitsPerSubframe
				for phase := 0; phase < 12; phase++ {
					if res, ok := p25p2.DecodeACCHBurst(sub.Dibits, phase, seq); ok {
						hex := ""
						for _, v := range framing.PackBitsMSB(res.Message) {
							hex += fmt.Sprintf("%02X", v)
						}
						rs := 0
						if res.RSValid {
							rs = 1
						}
						fmt.Printf("FRAME,gophertrunk,%s,2,%.4f,MAC,1F0,1,%s;rs=%d\n",
							base, float64(start)/p25p2rx.SymbolRate, hex, rs)
						break
					}
				}
			}
		}
	}
	r := p25p2rx.New(p25p2rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: p25p2rx.ClockGardner,
		GardnerGain: 0.03, DibitSink: sink})
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
}

func TestP25FramesDump(t *testing.T) {
	p1 := strings.Fields(os.Getenv("GT_FRAMES_P1"))
	p2 := strings.Fields(os.Getenv("GT_FRAMES_P2"))
	if len(p1)+len(p2) == 0 {
		t.Skip("set GT_FRAMES_P1 and/or GT_FRAMES_P2 to wav paths")
	}
	for _, f := range p1 {
		dumpP1Frames(t, f)
	}
	for _, f := range p2 {
		dumpP2Frames(t, f)
	}
}

// readWavIQ16 reads a 2-channel 16-bit PCM baseband WAV (L=I, R=Q).
func readWavIQ16(t *testing.T, path string) ([]complex64, float64) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		t.Fatalf("%s: not RIFF/WAVE", path)
	}
	var rate float64
	var data []byte
	for off := 12; off+8 <= len(raw); {
		id := string(raw[off : off+4])
		sz := int(uint32(raw[off+4]) | uint32(raw[off+5])<<8 | uint32(raw[off+6])<<16 | uint32(raw[off+7])<<24)
		body := raw[off+8:]
		if sz > len(body) {
			sz = len(body)
		}
		body = body[:sz]
		switch id {
		case "fmt ":
			rate = float64(uint32(body[4]) | uint32(body[5])<<8 | uint32(body[6])<<16 | uint32(body[7])<<24)
		case "data":
			data = body
		}
		off += 8 + sz + (sz & 1)
	}
	n := len(data) / 4
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		re := int16(uint16(data[4*i]) | uint16(data[4*i+1])<<8)
		im := int16(uint16(data[4*i+2]) | uint16(data[4*i+3])<<8)
		out[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return out, rate
}
