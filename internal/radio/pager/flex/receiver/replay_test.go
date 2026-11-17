package receiver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"

	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// TestFLEXAudioReplay decodes a recording of FLEX discriminator audio (WAV or
// FLAC, any rate) and prints every page — the instrument for a field capture:
//
//	GT_FLEX_AUDIO=<file> go test ./internal/radio/pager/flex/receiver -run TestFLEXAudioReplay -v
func TestFLEXAudioReplay(t *testing.T) {
	path := os.Getenv("GT_FLEX_AUDIO")
	if path == "" {
		t.Skip("set GT_FLEX_AUDIO to a discriminator-audio recording")
	}
	pcm, rate, err := voice.ReadAudioSamples(path)
	if err != nil {
		t.Fatal(err)
	}
	dm, err := NewAudioDemod(rate)
	if err != nil {
		t.Fatal(err)
	}
	audio := make([]float32, len(pcm))
	for i, s := range pcm {
		audio[i] = float32(s) / 32768
	}
	// GT_FLEX_DUMP=<file> writes each frame's post-sync symbols, one frame a
	// line ("<mode code> <inverted> <symbols>"), for building fixtures.
	var dump []string
	if os.Getenv("GT_FLEX_DUMP") != "" {
		dm.OnFrame = func(m flex.Mode, inv bool) {
			dump = append(dump, fmt.Sprintf("%04x %v ", m.Code, inv))
		}
		dm.OnSymbol = func(sym byte) { dump[len(dump)-1] += string('0' + sym) }
		defer func() {
			_ = os.WriteFile(os.Getenv("GT_FLEX_DUMP"), []byte(strings.Join(dump, "\n")+"\n"), 0o644)
		}()
	}
	t.Logf("%s: %.1f s at %d Hz", path, float64(len(pcm))/float64(rate), rate)
	for _, m := range dm.ProcessAudio(audio) {
		t.Logf("%s (corrected %d)", m, m.Corrected)
	}
	st := dm.Decoder().Stats()
	t.Logf("frames=%d (1600:%d 3200:%d 6400:%d) fiw_errors=%d pages=%d words_failed=%d",
		st.FramesSynced, st.Frames1600, st.Frames3200, st.Frames6400, st.FIWErrors, st.PagesEmitted, st.WordsFailed)
}
