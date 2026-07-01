package hunt

import (
	"context"
	"math/rand"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// TestRunLiveSurvey_NamesRowsFromAllocation verifies the survey fills the
// consolidated inventory naming end-to-end: a carrier at an allocated frequency
// comes back with its Service/Purpose set from the allocation table, even in the
// fast classify-only pass.
func TestRunLiveSurvey_NamesRowsFromAllocation(t *testing.T) {
	const rate = 96_000
	const frsHz = 462_562_500 // FRS/GMRS channel 1
	carrier := fmNoise(rate*2, float64(rate), 3000, 11)

	src := NewMappedIQSource(map[uint32][]complex64{frsHz: carrier}, rate)
	sv, _, err := RunLiveSurvey(context.Background(), LiveHuntOptions{
		Source:       src,
		Candidates:   []uint32{frsHz},
		ClassifyOnly: true,
		DwellSeconds: 1,
	})
	if err != nil {
		t.Fatalf("RunLiveSurvey: %v", err)
	}
	if len(sv.Signals) != 1 {
		t.Fatalf("len(Signals) = %d, want 1", len(sv.Signals))
	}
	s := sv.Signals[0]
	if s.Service != "FRS / GMRS" {
		t.Errorf("Service = %q, want FRS / GMRS", s.Service)
	}
	if s.Name == "" {
		t.Error("Name is empty; want at least the class fallback")
	}
}

// TestRunLiveSurvey_WidebandRowNamed drives the wideband path end-to-end: a
// synthetic signal far wider than the tune is stitched, routed straight to a
// named-but-not-decoded row, and labeled cellular from its allocation.
func TestRunLiveSurvey_WidebandRowNamed(t *testing.T) {
	const rate = 2_048_000
	src := &bandNoiseSource{
		rate:  rate,
		sigLo: 700_000_000,
		sigHi: 706_000_000, // ~6 MHz, in the 700 MHz cellular band
		rng:   rand.New(rand.NewSource(0x9e3779b9)),
	}
	sv, _, err := RunLiveSurvey(context.Background(), LiveHuntOptions{
		Source:         src,
		Bands:          []Band{{LowHz: 699_000_000, HighHz: 708_000_000}},
		FFTSize:        1024,
		DetectWideband: true,
		ClassifyOnly:   true,
		DwellSeconds:   0.1, // bound any spurious narrowband dwell over synthetic noise
	})
	if err != nil {
		t.Fatalf("RunLiveSurvey: %v", err)
	}
	var wb *DetectedSignal
	for i := range sv.Signals {
		if sv.Signals[i].Wideband {
			wb = &sv.Signals[i]
			break
		}
	}
	if wb == nil {
		t.Fatalf("no wideband row in survey: %+v", sv.Signals)
	}
	if wb.Class != survey.ClassWideband {
		t.Errorf("class = %q, want wideband", wb.Class)
	}
	if !strings.Contains(wb.Name, "LTE") {
		t.Errorf("Name = %q, want an LTE/5G best guess", wb.Name)
	}
	if !strings.Contains(wb.Service, "Cellular 700") {
		t.Errorf("Service = %q, want the 700 MHz cellular allocation", wb.Service)
	}
	if wb.OccupiedBwHz < 2_000_000 {
		t.Errorf("OccupiedBwHz = %d, want a multi-MHz stitched width", wb.OccupiedBwHz)
	}
	if wb.Trunking != nil {
		t.Error("a wideband row must not carry a trunking decode")
	}
}
