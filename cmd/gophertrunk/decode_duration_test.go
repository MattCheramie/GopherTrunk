package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestDecodeReportsAudioLengthPerVocoder pins #1321: `gophertrunk decode`
// reported every vocoder's output at 20 ms per frame, so 66 tetra-acelp frames
// read "1320 ms" for a 1.98 s WAV (TETRA ACELP frames are 30 ms). The summary
// now counts the samples actually written.
func TestDecodeReportsAudioLengthPerVocoder(t *testing.T) {
	for _, tc := range []struct {
		vocoder   string
		frames    int
		wantMilli int
	}{
		{"tetra-acelp", 66, 1980},
		{"imbe", 50, 1000},
		{"ambe2", 50, 1000},
	} {
		t.Run(tc.vocoder, func(t *testing.T) {
			v, err := voice.DefaultRegistry.New(tc.vocoder)
			if err != nil {
				t.Fatalf("vocoder %q not registered: %v", tc.vocoder, err)
			}
			defer v.Close()
			in := bytes.NewReader(make([]byte, v.FrameSize()*tc.frames))
			out, err := os.Create(filepath.Join(t.TempDir(), "out.wav"))
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			counted := &sampleCountingVocoder{Vocoder: v}
			n, err := voice.DecodeStreamWithVocoder(in, counted, out)
			if err != nil {
				t.Fatal(err)
			}
			if n != tc.frames {
				t.Fatalf("decoded %d frames, want %d", n, tc.frames)
			}
			if got := counted.audioMillis(); got != tc.wantMilli {
				t.Fatalf("audioMillis = %d, want %d (%d frames of %s)", got, tc.wantMilli, n, tc.vocoder)
			}
			st, err := out.Stat()
			if err != nil {
				t.Fatal(err)
			}
			if wavMilli := int((st.Size() - 44) / 2 * 1000 / 8000); wavMilli != tc.wantMilli {
				t.Fatalf("WAV holds %d ms, reported %d ms", wavMilli, tc.wantMilli)
			}
		})
	}
}
