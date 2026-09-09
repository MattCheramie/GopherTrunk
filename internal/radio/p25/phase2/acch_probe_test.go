//go:build integration

package phase2_test

// Real-air harness for the P25 Phase 2 ACCH layer (issue #915): replays a
// baseband capture of a Phase 2 traffic channel through this package's own
// receiver, superframe decoder and ACCH decode, and prints every MAC PDU it
// recovers as hex — so the output can be diffed against SDRtrunk's decode of
// the same file, which is how the layer was validated in the first place.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run ACCH -v
//
// SDRtrunk's side of the diff comes from p25-tools/sdrtrunk-probe:
//
//	PROBE_OPTS=-Dp2.machex=2000 ./run.sh -p2 /path/capture.wav
//
// The unit tests in acch_test.go pin three bursts from this harness, so an
// everyday `go test` still covers the chain without needing a capture.

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// The capture's scramble identity, read off its control channel
// (NET_STATUS_BCAST). A Phase 2 capture cannot be descrambled without it.
const (
	probeWACN  = 0xBEE00
	probeSysID = 0x1FC
	probeNAC   = 0x1F0
)

// readWavIQ reads a 2-channel 16-bit PCM baseband recording — SDRtrunk's
// BASEBAND / TRAFFIC_BASEBAND format, left = I, right = Q — into complex64.
func readWavIQ(t *testing.T, path string) ([]complex64, float64) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		t.Fatalf("not a RIFF/WAVE file")
	}
	var rate float64
	var channels, bits int
	var data []byte
	for off := 12; off+8 <= len(raw); {
		id := string(raw[off : off+4])
		sz := int(binary.LittleEndian.Uint32(raw[off+4 : off+8]))
		body := raw[off+8:]
		if sz > len(body) {
			sz = len(body)
		}
		switch id {
		case "fmt ":
			channels = int(binary.LittleEndian.Uint16(body[2:4]))
			rate = float64(binary.LittleEndian.Uint32(body[4:8]))
			bits = int(binary.LittleEndian.Uint16(body[14:16]))
		case "data":
			data = body[:sz]
		}
		off += 8 + sz + (sz & 1)
	}
	if channels != 2 || bits != 16 {
		t.Fatalf("want 2ch/16-bit, got %dch/%d-bit", channels, bits)
	}
	out := make([]complex64, len(data)/4)
	for i := range out {
		i16 := int16(binary.LittleEndian.Uint16(data[i*4:]))
		q16 := int16(binary.LittleEndian.Uint16(data[i*4+2:]))
		out[i] = complex(float32(i16)/32768, float32(q16)/32768)
	}
	return out, rate
}

// probeDibits runs an IQ capture through the Phase 2 receiver and feeds the
// dibit stream to sink.
//
// The capture is resampled to 48 kHz first, and that is not cosmetic: the
// receiver computes samples-per-symbol as int(SampleRateHz/6000 + 0.5), so a
// 50 kHz capture (8.33 sps) rounds to 8 and the whole stream comes out at 6250
// baud — 4.2 % fast, half a burst of drift per superframe. Sync still locks,
// because a 20-dibit word tolerates the drift, and everything downstream
// walks. Remove this resample only when the receiver handles a fractional sps.
func probeDibits(iq []complex64, rate float64, sink func([]uint8, int)) {
	probeDibitsWD(iq, rate, nil, nil, sink)
}

// probeDibitsWD is probeDibits with the production carrier watchdog: when wd
// and sfDec are non-nil, the receiver is reset between IQ chunks after a
// superframe's worth of dibits with no lock, exactly as the composer's voice
// chain does. Without it a capture decodes only until the one-shot carrier
// seed goes stale — about 0.6 s on the reference file.
func probeDibitsWD(iq []complex64, rate float64, wd *p25p2.CarrierWatchdog,
	sfDec *p25p2.SuperframeDecoder, sink func([]uint8, int)) {
	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	reacquire := false
	wrapped := sink
	if wd != nil {
		wrapped = func(dibits []uint8, baseIdx int) {
			before := sfCount
			sink(dibits, baseIdx)
			if wd.Observe(len(dibits), sfCount-before) {
				reacquire = true
			}
		}
	}
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
		GardnerGain: 0.005, DibitSink: wrapped})
	var buf []complex64
	const chunk = 8192
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
		if reacquire {
			r.Reset()
			sfDec.Reset()
			wd.Reset()
			reacquire = false
		}
	}
}

// sfCount is how many superframes the current probe sink has drained; the
// watchdog wrapper reads it to tell a locked window from an idle one.
var sfCount int

func hexOf(bits []byte) string {
	var s string
	for _, v := range framing.PackBitsMSB(bits) {
		s += fmt.Sprintf("%02X", v)
	}
	return s
}

func TestACCHProbe(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV to a Phase 2 traffic-channel baseband capture")
	}
	iq, rate := readWavIQ(t, path)
	t.Logf("capture: %d samples @ %.0f Hz (%.2f s)", len(iq), rate, float64(len(iq))/rate)

	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))
	sfDec := p25p2.NewSuperframeDecoder()

	var superframes, acchBursts, decoded, rsValid, rsErrors int
	typeHist := map[int]int{}
	slotOffsets := map[int]int{}
	pdus := map[string]int{}
	worstErr := map[string]int{}

	wd := p25p2.NewCarrierWatchdog(0)
	sfCount = 0
	probeDibitsWD(iq, rate, wd, sfDec, func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
			superframes++
			sfCount++
			for _, sub := range sf.Subframes {
				if len(sub.Dibits) < p25p2.BurstDibits {
					continue
				}
				bt := p25p2.BurstTypeOf(sub.Dibits)
				typeHist[int(bt)]++
				if !bt.IsACCH() {
					continue
				}
				acchBursts++
				// The superframe anchor is whichever S-ISCH slot the sync
				// matched, so a sub-frame's Index is offset from its true slot
				// by an unknown constant. Search the 12 phases; the CRC-12
				// picks the right one, and the offsets it finds should land
				// only on the six S-ISCH slots {2,3,6,7,10,11}.
				for phase := 0; phase < 12; phase++ {
					res, ok := p25p2.DecodeACCHBurst(sub.Dibits, phase, seq)
					if !ok {
						continue
					}
					decoded++
					if res.RSValid {
						rsValid++
						rsErrors += res.RSErrors
					}
					h := hexOf(res.Message)
					pdus[h]++
					if res.RSErrors > worstErr[h] {
						worstErr[h] = res.RSErrors
					}
					slotOffsets[((phase-sub.Index)%12+12)%12]++
					break
				}
			}
		}
	})

	t.Logf("superframes=%d acch_bursts=%d decoded=%d (%.0f%%) rs_valid=%d rs_symbol_errors=%d",
		superframes, acchBursts, decoded, 100*float64(decoded)/float64(max(acchBursts, 1)),
		rsValid, rsErrors)
	types := make([]int, 0, len(typeHist))
	for k := range typeHist {
		types = append(types, k)
	}
	sort.Ints(types)
	for _, k := range types {
		t.Logf("  burst type %3d: %d", k, typeHist[k])
	}
	t.Logf("slot-offset histogram (expect only 2,3,6,7,10,11): %v", slotOffsets)
	hexes := make([]string, 0, len(pdus))
	for h := range pdus {
		hexes = append(hexes, h)
	}
	sort.Strings(hexes)
	for _, h := range hexes {
		fmt.Printf("GTMAC %s x%d max_rs_errors=%d\n", h, pdus[h], worstErr[h])
	}
	for off := range slotOffsets {
		switch off {
		case 2, 3, 6, 7, 10, 11:
		default:
			t.Errorf("slot offset %d is not an S-ISCH slot; the superframe "+
				"anchor or the scramble origin is wrong", off)
		}
	}
}
