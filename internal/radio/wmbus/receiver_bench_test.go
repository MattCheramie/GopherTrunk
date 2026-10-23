package wmbus

import (
	"fmt"
	"math/rand"
	"testing"
)

// BenchmarkReceiver measures the live decode cost per input rate, as the
// fraction of one core a live run needs (cpu_share; above 1 the SDR drops
// IQ). The S5 in issue #1256 dropped IQ at 1.6 MS/s: halving the
// processing rate to 4 samples per chip took this from 0.13 to 0.07 on an
// x86 dev box. Noise in, so the framers see access-code-free input, as a
// live run does most of the time.
func BenchmarkReceiver(b *testing.B) {
	for _, rate := range []float64{1_024_000, 1_200_000, 1_600_000, 2_400_000} {
		b.Run(fmt.Sprintf("%.3fMSps", rate/1e6), func(b *testing.B) {
			rng := rand.New(rand.NewSource(1))
			iq := make([]complex64, 16384)
			for i := range iq {
				iq[i] = complex(float32(rng.NormFloat64()*0.05), float32(rng.NormFloat64()*0.05))
			}
			rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate})
			if err != nil {
				b.Fatal(err)
			}
			rx.afc(-50_000)                // the #1256 dongle's offset: the carrier NCO runs too
			b.SetBytes(int64(len(iq)) * 2) // u8 IQ bytes per chunk
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rx.Process(iq)
			}
			perSample := b.Elapsed().Seconds() / float64(b.N*len(iq))
			b.ReportMetric(perSample*rate, "cpu_share")
		})
	}
}
