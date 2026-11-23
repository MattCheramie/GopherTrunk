package phase2

import (
	"math/rand"
	"os"
	"testing"
)

// The soft retry erases a burst's least reliable symbols and asks the
// Reed-Solomon decoder again, up to nine more times; each attempt is a chance
// to land on a wrong codeword that the CRC must then catch. On random bursts
// with random reliabilities none of 20 000 per channel decodes, hard or soft
// (200 000 each also decoded none while airMACSoftErasureMargin was swept from
// 6 down to 0).
func TestAirMACSoftRetryFalseAcceptRate(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	const n = 20000
	for _, d := range []DUID{DUIDFACCHScrambled, DUIDSACCHScrambled, DUIDLCCHClear} {
		accepted := 0
		for i := 0; i < n; i++ {
			pkt := make([]uint8, AirPacketDibits)
			rel := make([]float32, AirPacketDibits)
			for j := range pkt {
				pkt[j] = uint8(r.Intn(4))
				rel[j] = r.Float32()
			}
			if _, ok := DecodeAirMAC(AirBurst{DUID: d, Packet: pkt, Rel: rel}, nil); ok {
				accepted++
			}
		}
		if accepted != 0 {
			t.Errorf("%v: %d of %d random bursts decoded with the soft retry", d, accepted, n)
		}
	}
}

// A framer fed by ProcessSoft hands each burst the reliabilities of its own
// dibits; one fed by Process hands none.
func TestAirFramerCarriesReliabilities(t *testing.T) {
	bursts := frameMMR(t, mmrDibitsClean)
	if len(bursts) == 0 || bursts[0].Rel != nil {
		t.Fatalf("hard framer: %d bursts, first carries reliabilities %v", len(bursts), bursts[0].Rel != nil)
	}
	d := mustReadMMR(t)
	rel := make([]float32, len(d))
	for i := range rel {
		rel[i] = float32(i) // a dibit's reliability is its index
	}
	f := NewAirFramer()
	var soft []AirBurst
	for i := 0; i < len(d); i += 82 {
		end := min(i+82, len(d))
		soft = append(soft, f.ProcessSoft(d[i:end], rel[i:end], i)...)
	}
	if len(soft) != len(bursts) {
		t.Fatalf("soft framer cut %d bursts, hard %d", len(soft), len(bursts))
	}
	for _, b := range soft {
		if len(b.Rel) != AirPacketDibits {
			t.Fatalf("burst at %d carries %d reliabilities", b.StartDibit, len(b.Rel))
		}
		for i, v := range b.Rel {
			if int(v) != b.StartDibit+i {
				t.Fatalf("burst at %d: reliability %d belongs to dibit %d", b.StartDibit, i, int(v))
			}
		}
	}
}

func mustReadMMR(t *testing.T) []uint8 {
	t.Helper()
	d, err := os.ReadFile(mmrDibitsClean)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
