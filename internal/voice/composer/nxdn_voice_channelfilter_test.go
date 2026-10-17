package composer

import (
	"encoding/binary"
	"math"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
)

// The NXDN voice chain must reject the neighbouring channel (#1261). The
// front end ahead of the receiver barely band-limits at a 2.4 MS/s SDR rate,
// so the FM discriminator saw the whole ±24 kHz intermediate stream. Here the
// reporter's two real-air captures (447.1 MHz Kenwood, NXDN96 and NXDN48)
// get a same-power copy of a different stretch of the same signal one
// channel away — 12.5 kHz for NXDN96, 6.25 kHz for NXDN48. Without the
// channel filter the chain's receiver decoded 0 and 12 voice frames; with it,
// the whole call (36 and 56, as on the clean captures).
func TestNXDNVoiceReceiverRejectsAdjacentChannel(t *testing.T) {
	for _, tc := range []struct {
		file       string
		symbolRate float64
		adjHz      float64
		minFrames  int
	}{
		{"../../radio/nxdn/testdata/nxdn96_rdch_voice_447100_15oct_48k.cs16", nxdnrx.SymbolRate, 12500, 32},
		{"../../radio/nxdn/testdata/nxdn48_rdch_voice_447100_16oct_48k.cs16", nxdnrx.SymbolRateNXDN48, 6250, 48},
	} {
		t.Run(tc.file[len("../../radio/nxdn/testdata/"):], func(t *testing.T) {
			raw, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			iq := make([]complex64, len(raw)/4)
			for i := range iq {
				iq[i] = complex(float32(int16(binary.LittleEndian.Uint16(raw[4*i:])))/32768,
					float32(int16(binary.LittleEndian.Uint16(raw[4*i+2:])))/32768)
			}
			in := make([]complex64, len(iq))
			for i := range iq {
				ph := 2 * math.Pi * tc.adjHz * float64(i) / nxdnVoiceIntermediateHz
				in[i] = iq[i] + iq[(i+len(iq)/2)%len(iq)]*complex64(complex(math.Cos(ph), math.Sin(ph)))
			}

			var frames []nxdn.VCHFrame
			trf := nxdn.NewTrafficChannel(func(f []nxdn.VCHFrame) { frames = append(frames, f...) })
			rx := newNXDNVoiceReceiver(nxdnVoiceIntermediateHz, tc.symbolRate, func(d []uint8, base int) { trf.Process(d, base) })
			for i := 0; i < len(in); i += 4096 {
				rx.Process(in[i:min(i+4096, len(in))])
			}
			t.Logf("voice_frames=%d frames=%d lich_ok=%d", len(frames), trf.Frames, trf.LICHOK)
			if len(frames) < tc.minFrames {
				t.Fatalf("%d voice frames with a neighbour %.2f kHz away, want ≥ %d", len(frames), tc.adjHz/1000, tc.minFrames)
			}
		})
	}
}
