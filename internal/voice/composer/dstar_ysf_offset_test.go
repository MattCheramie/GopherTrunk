package composer

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
)

// Issue #1261: the D-STAR and System Fusion voice receivers had no carrier
// correction and no channel filter. A real radio is routinely a few hundred
// hertz off frequency (both of the reporter's NXDN captures sat ~400 Hz off),
// and at a 2.4 MS/s SDR rate the voice front end hands the receiver ±24 kHz,
// so a neighbouring channel lands in the discriminator. Each case below is a
// transmission 400 Hz off frequency with a same-power neighbour one channel
// away; the old receivers decoded none of it.

// offsetPlusNeighbour shifts want by offHz and adds adj shifted by adjHz.
func offsetPlusNeighbour(want, adj []complex64, offHz, adjHz, fs float64) []complex64 {
	out := make([]complex64, len(want))
	for i := range want {
		p1 := 2 * math.Pi * offHz * float64(i) / fs
		p2 := 2 * math.Pi * adjHz * float64(i) / fs
		out[i] = want[i]*complex64(complex(math.Cos(p1), math.Sin(p1))) +
			adj[i%len(adj)]*complex64(complex(math.Cos(p2), math.Sin(p2)))
	}
	return out
}

func TestDStarVoiceReceiverToleratesOffsetAndNeighbour(t *testing.T) {
	const fs, dev = 48_000.0, 1200.0
	r := rand.New(rand.NewSource(11))
	bits := make([]uint8, 2*dstar.DVFrameBits) // warm-up: carrier settling
	for i := range bits {
		bits[i] = uint8(r.Intn(2))
	}
	var want [][]byte
	for f := 0; f < 2*dstar.DVFramesPerSyncCycle; f++ {
		info := make([]byte, 49)
		for i := range info {
			info[i] = byte(r.Intn(2))
		}
		want = append(want, info)
		voice, err := dstar.EncodeDVVoiceBits(info)
		if err != nil {
			t.Fatal(err)
		}
		bits = append(bits, voice...)
		if f%dstar.DVFramesPerSyncCycle == 0 {
			bits = append(bits, dstar.SlowDataSyncBitsSlice()...)
		} else {
			for i := 0; i < dstar.DVDataBits; i++ {
				bits = append(bits, uint8(r.Intn(2)))
			}
		}
	}
	adjBits := make([]uint8, len(bits))
	for i := range adjBits {
		adjBits[i] = uint8(r.Intn(2))
	}
	iq := offsetPlusNeighbour(
		demod.ModulateGFSK(bits, 10, 4, 0.5, fs, dev),
		demod.ModulateGFSK(adjBits, 10, 4, 0.5, fs, dev),
		400, 12_500, fs)

	match := 0
	vc := dstar.NewVoiceChannel(func(f dstar.DVFrame) {
		for _, w := range want {
			if string(f.Payload) == string(w) {
				match++
				return
			}
		}
	})
	rx := newDStarVoiceReceiver(fs, func(b []byte, base int) { vc.Process(b, base) })
	for i := 0; i < len(iq); i += 4096 {
		rx.Process(iq[i:min(i+4096, len(iq))])
	}
	if min := len(want) - 4; match < min {
		t.Fatalf("decoded %d of %d DV voice frames 400 Hz off with a neighbour 12.5 kHz away, want ≥ %d", match, len(want), min)
	}
}

func TestYSFVoiceReceiverToleratesOffsetAndNeighbour(t *testing.T) {
	const fs, dev, frames = 48_000.0, 1800.0, 8
	r := rand.New(rand.NewSource(12))
	lead := make([]uint8, 480)
	for i := range lead {
		lead[i] = uint8(r.Intn(4))
	}
	dibits := append([]uint8(nil), lead...)
	dibits = append(dibits, ysfFrameDibits(t, ysfFICHHeader, make([]uint8, ysf.PayloadDibits))...)
	var want []string
	for f := 0; f < frames; f++ {
		var infos [5][]byte
		for j := range infos {
			infos[j] = make([]byte, ysf.VD2InfoBits)
			for k := range infos[j] {
				infos[j][k] = uint8(r.Intn(2))
			}
			want = append(want, string(framing.PackBitsMSB(infos[j])))
		}
		dibits = append(dibits, ysfFrameDibits(t, ysfFICHVD2, ysf.EncodeVD2Voice(infos))...)
	}
	dibits = append(dibits, ysfFrameDibits(t, ysfFICHTerm, make([]uint8, ysf.PayloadDibits))...)
	dibits = append(dibits, lead...)
	adj := make([]uint8, len(dibits))
	for i := range adj {
		adj[i] = uint8(r.Intn(4))
	}
	iq := offsetPlusNeighbour(
		demod.ModulateC4FM(dibits, 10, 8, 0.20, fs, dev),
		demod.ModulateC4FM(adj, 10, 8, 0.20, fs, dev),
		400, 12_500, fs)

	match := 0
	framer := ysf.NewAirFramer(2)
	rx := newYSFVoiceReceiver(fs, func(d []uint8, base int) {
		for _, af := range framer.Process(d, base) {
			if !af.FICHOK || af.FICH.DT != ysf.AirDTVDMode2 || af.FICH.FI != ysf.AirFIComms {
				continue
			}
			for _, fr := range ysf.DecodeVD2Voice(af.Payload) {
				got := string(framing.PackBitsMSB(fr.Info))
				for _, w := range want {
					if got == w {
						match++
						break
					}
				}
			}
		}
	})
	for i := 0; i < len(iq); i += 4096 {
		rx.Process(iq[i:min(i+4096, len(iq))])
	}
	if min := (frames - 2) * 5; match < min {
		t.Fatalf("decoded %d of %d V/D mode 2 voice frames 400 Hz off with a neighbour 12.5 kHz away, want ≥ %d", match, len(want), min)
	}
}
