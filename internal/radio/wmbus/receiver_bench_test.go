package wmbus

import (
	"fmt"
	"math/rand"
	"testing"
)

// BenchmarkReceiver measures the live decode cost per input rate, as the
// fraction of one core a live run needs (cpu_share; above 1 the SDR drops
// IQ), with the burst gate and without it. Noise in, as a live run hears
// most of the time: 4 s of it, so the gate opens on noise at its natural
// rate (demod_share is how much of the IQ the demodulator ran on).
//
// The S5 in issue #1256 dropped IQ at 1.6 MS/s whenever the phone slowed
// the process down. Halving the processing rate to 4 samples per chip took
// the ungated cost from 0.13 to 0.06 on an x86 dev box; the gate takes it to
// about 0.003.
func BenchmarkReceiver(b *testing.B) {
	for _, rate := range []float64{1_024_000, 1_200_000, 1_600_000, 2_400_000} {
		for _, noGate := range []bool{false, true} {
			name := fmt.Sprintf("%.3fMSps", rate/1e6)
			if noGate {
				name += "/ungated"
			}
			b.Run(name, func(b *testing.B) {
				rng := rand.New(rand.NewSource(1))
				noise := make([]complex64, int(4*rate))
				for i := range noise {
					noise[i] = complex(float32(rng.NormFloat64()*0.05), float32(rng.NormFloat64()*0.05))
				}
				rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, NoBurstGate: noGate})
				if err != nil {
					b.Fatal(err)
				}
				rx.afc(-50_000) // the #1256 dongle's offset: the carrier NCO runs too
				const chunk = 16384
				b.SetBytes(chunk * 2) // u8 IQ bytes per chunk
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					k := (i * chunk) % (len(noise) - chunk)
					rx.Process(noise[k : k+chunk])
				}
				perSample := b.Elapsed().Seconds() / float64(b.N*chunk)
				b.ReportMetric(perSample*rate, "cpu_share")
				st := rx.Stats()
				b.ReportMetric(float64(st.Demodulated)/float64(st.Samples), "demod_share")
			})
		}
	}
}
