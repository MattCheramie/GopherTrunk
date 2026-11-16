package survey

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
)

// flexIQ modulates one frame as 4-level FSK (±4800 / ±1600 Hz) at rate.
func flexIQ(syms []byte, mode flex.Mode, rate float64) []complex64 {
	const head = 32 + 64 + 48 // symbols sent at 1600/s whatever the mode
	lead := int(0.1 * rate)
	iq := make([]complex64, 0, lead+int(2*rate))
	var phase float64
	emit := func(hz float64, n int) {
		for i := 0; i < n; i++ {
			phase += 2 * math.Pi * hz / rate
			iq = append(iq, complex(float32(math.Cos(phase)), float32(math.Sin(phase))))
		}
	}
	emit(0, lead)
	var t float64 // running symbol-edge time, in samples
	done := 0
	for i, s := range syms {
		baud := 1600.0
		if i >= head {
			baud = float64(mode.Baud)
		}
		t += rate / baud
		n := int(t+0.5) - done
		done += n
		emit((float64(s)*2-3)*1600, n)
	}
	emit(0, lead)
	return iq
}

// TestDecodeFLEXEveryRate: the survey's FLEX decode follows all the modes and
// reports the bit rate a page came in at.
func TestDecodeFLEXEveryRate(t *testing.T) {
	for _, mode := range flex.Modes {
		phase, ok := flex.PhaseWith(flex.Page{Capcode: 123456, Text: "SURVEY PAGE"})
		if !ok {
			t.Fatal("page does not fit")
		}
		pages := DecodeFLEX(flexIQ(flex.FrameSymbols(mode, 0, 1, phase), mode, 48000), 48000)
		if len(pages) != 1 {
			t.Fatalf("%d bps: %d pages, want 1", mode.BitRate(), len(pages))
		}
		p := pages[0]
		if p.Protocol != "flex" || p.Capcode != 123456 || p.Text != "SURVEY PAGE" ||
			p.Encoding != "alpha" || p.BaudHz != uint32(mode.BitRate()) {
			t.Errorf("%d bps: page = %+v", mode.BitRate(), p)
		}
	}
	if !IsPagingBaud(3200) {
		t.Error("3200 symbols/s (FLEX 3200 / 6400 bps) is not routed to the paging decoders")
	}
}
