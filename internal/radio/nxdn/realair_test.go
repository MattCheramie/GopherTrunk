package nxdn_test

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// Real-air NXDN voice (issue #1261). The fixtures are 48 kS/s cs16 slices
// of the reporter's scanner voice_iq_debug captures on 447.1 MHz (Kenwood
// radios, conventional/direct RDCH), taken after the composer's channel
// front end: a 12.5 kHz NXDN96 call (15 Oct) and a 6.25 kHz NXDN48 call
// (16 Oct). Both sit ~400 Hz off frequency, as captured.
//
// Before the real air format (airframe.go), the voice path searched for an
// invented 16-bit FSW and decoded a placeholder AMBE layout, so these
// slices produced either nothing or random vocoder parameters. Speech is
// judged by pitch continuity (|Δb0| ≤ 10 between consecutive frames:
// speech ≳ 0.5, random ≈ 0.16), never by output level.
func TestTrafficChannelDecodesRealAirNXDN(t *testing.T) {
	for _, tc := range []struct {
		file       string
		symbolRate float64
		minFrames  int
	}{
		{"testdata/nxdn96_rdch_voice_447100_15oct_48k.cs16", nxdnrx.SymbolRate, 24},
		{"testdata/nxdn48_rdch_voice_447100_16oct_48k.cs16", nxdnrx.SymbolRateNXDN48, 32},
	} {
		t.Run(tc.file, func(t *testing.T) {
			iq := readCS16(t, tc.file)
			var frames []nxdn.VCHFrame
			trf := nxdn.NewTrafficChannel(func(f []nxdn.VCHFrame) { frames = append(frames, f...) })
			// The composer's voice-chain receiver configuration.
			rx := nxdnrx.New(nxdnrx.Options{
				SampleRateHz: 48000,
				SymbolRateHz: tc.symbolRate,
				DeviationHz:  1800,
				EnableAFC:    true,
				// The composer runs the pre-discriminator carrier tracker too
				// (#1261 drift); it must cost nothing on a near-centred capture.
				EnableCarrierTracking: true,
				// The channel filter must cost nothing on a clean capture
				// (#1261): same frames, Golay errors within the bounds below.
				EnableChannelFilter: true,
				DibitSink:           func(d []uint8, base int) { trf.Process(d, base) },
			})
			for i := 0; i < len(iq); i += 4096 {
				rx.Process(iq[i:min(i+4096, len(iq))])
			}
			errs, cont := 0, 0
			for i, f := range frames {
				errs += f.Errors
				if i > 0 {
					d := b0(f.Payload) - b0(frames[i-1].Payload)
					if d >= -10 && d <= 10 {
						cont++
					}
				}
			}
			n := len(frames)
			t.Logf("frames=%d lich_ok=%d voice_frames=%d golay_errs/frame=%.2f b0_continuity=%.2f",
				trf.Frames, trf.LICHOK, n, float64(errs)/float64(max(n, 1)), float64(cont)/float64(max(n-1, 1)))
			if n < tc.minFrames {
				t.Fatalf("%d voice frames, want ≥ %d", n, tc.minFrames)
			}
			if c := float64(cont) / float64(n-1); c < 0.5 {
				t.Errorf("pitch continuity %.2f, want ≥ 0.5 (speech)", c)
			}
			if e := float64(errs) / float64(n); e > 1.0 {
				t.Errorf("%.2f Golay corrections per frame, want ≤ 1", e)
			}
		})
	}
}

// TestAirFramerFramesRealNXDNControlChannel pins the real air format on an
// NXDN control channel. The fixture is one second of the #1261 reporter's
// 19 Oct capture of a trunked system's control channel (453.775 MHz,
// channel 1 of 10; a voice_iq_debug recording at 2.4 MS/s, cut after the
// composer's channel front end to 48 kS/s). Every frame must frame at the
// real FSW, 192 dibits apart, with a LICH that passes parity and names a
// control channel (RFCT 0). The control-channel (CAC) decoder still runs on
// the package's invented layout; this is the ground its rebuild stands on.
//
// The capture was drifting while it recorded: its carrier walks from -645
// to -679 Hz over 8.7 s. The receiver's carrier tracker must find it.
func TestAirFramerFramesRealNXDNControlChannel(t *testing.T) {
	iq := readCS16(t, "testdata/nxdn96_rcch_cc_453775_19oct_48k.cs16")
	var dib []uint8
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz:          48000,
		SymbolRateHz:          nxdnrx.SymbolRate,
		DeviationHz:           1800,
		EnableAFC:             true,
		EnableCarrierTracking: true,
		EnableChannelFilter:   true,
		DibitSink:             func(d []uint8, _ int) { dib = append(dib, d...) },
	})
	for i := 0; i < len(iq); i += 4096 {
		rx.Process(iq[i:min(i+4096, len(iq))])
	}
	var starts []int
	for k := 0; k+nxdn.AirFrameDibits <= len(dib); k++ {
		errs := 0
		for j := range nxdn.AirFSW {
			if dib[k+j] != nxdn.AirFSW[j] {
				errs++
			}
		}
		if errs > 1 {
			continue
		}
		post := append([]uint8(nil), dib[k+nxdn.AirFSWDibits:k+nxdn.AirFrameDibits]...)
		nxdn.DescrambleAir(post)
		l := nxdn.ParseAirLICH(post)
		if !l.ParityOK || l.RFCT != nxdn.AirRFCtRCCH {
			t.Errorf("frame at dibit %d: LICH %08b (parity ok %v, RFCT %d), want a control-channel LICH", k, l.Raw, l.ParityOK, l.RFCT)
		}
		if len(starts) > 0 && k-starts[len(starts)-1] != nxdn.AirFrameDibits {
			t.Errorf("frame at dibit %d is %d dibits after the previous one, want %d", k, k-starts[len(starts)-1], nxdn.AirFrameDibits)
		}
		starts = append(starts, k)
	}
	// 4800 sym/s / 192 = 25 frames/s; allow the receiver's pull-in and the
	// slice's last, partial frame (20 decode).
	t.Logf("frames=%d carrier=%.0f Hz", len(starts), rx.CarrierOffsetHz())
	if len(starts) < 18 {
		t.Fatalf("%d control-channel frames in 1 s, want ≥ 18", len(starts))
	}
	if off := rx.CarrierOffsetHz(); off > -550 || off < -750 {
		t.Errorf("carrier tracker at %.0f Hz, want the capture's ≈ -650 Hz", off)
	}
}

// b0 is AMBE+2's 7-bit pitch index: payload bits 0..3 and 37..39.
func b0(p []byte) int {
	return int(p[0])<<6 | int(p[1])<<5 | int(p[2])<<4 | int(p[3])<<3 | int(p[37])<<2 | int(p[38])<<1 | int(p[39])
}

func readCS16(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}
