package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestIMBEUnvoicedGainSweep is a calibration instrument, skipped unless
// GT_IMBE_RAW_DIR is set: it decodes every .raw IMBE sidecar in that directory
// once per unvoiced band gain in GT_IMBE_GAINS (comma list, e.g. "1,2,5.49")
// into GT_IMBE_OUT/<name>.g<gain>.wav with the recorder's synthesis defaults
// (§6.2 enhancement on), so the same frames can be scored per gain against an
// external reference decode of the same transmission (OP25 / trunk-recorder,
// dsd-neo) by long-term spectral distance. Used for the 10 Sep P25 A/B
// (see CLAUDE.md "P25 IMBE vs trunk-recorder").
func TestIMBEUnvoicedGainSweep(t *testing.T) {
	dir := os.Getenv("GT_IMBE_RAW_DIR")
	if dir == "" {
		t.Skip("set GT_IMBE_RAW_DIR")
	}
	out := os.Getenv("GT_IMBE_OUT")
	gains := strings.Split(os.Getenv("GT_IMBE_GAINS"), ",")
	raws, _ := filepath.Glob(filepath.Join(dir, "*.raw"))
	for _, gs := range gains {
		g, err := strconv.ParseFloat(gs, 64)
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range raws {
			v, err := voice.DefaultRegistry.New("imbe")
			if err != nil {
				t.Fatal(err)
			}
			v.(voice.SpecAmplitudeConfigurable).SetSpecAmplitudeEnhance(true)
			v.(voice.UnvoicedGainConfigurable).SetUnvoicedGain(g)
			in, err := os.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(filepath.Base(raw), ".raw")
			of, err := os.Create(filepath.Join(out, base+".g"+gs+".wav"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := voice.DecodeStreamWithVocoder(in, v, of); err != nil {
				t.Fatal(err)
			}
			of.Close()
			in.Close()
			v.Close()
		}
	}
}
