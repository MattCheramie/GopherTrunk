package hunt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunOfflineSurvey classifies two capture files — a synthesized P25 control
// channel and an analog FM voice channel — and asserts the offline survey
// routes each like the live one: the P25 folds into the system, the analog is
// reported as active NBFM. It proves the survey works without an SDR.
func TestRunOfflineSurvey(t *testing.T) {
	p25, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := meta.SampleRateHz

	dir := t.TempDir()
	p25Path := filepath.Join(dir, "p25.cfile")
	if err := siglab.WriteCapture(p25Path, p25, siglab.FormatF32); err != nil {
		t.Fatalf("write p25: %v", err)
	}
	fmPath := filepath.Join(dir, "fm.cfile")
	analog := fmNoise(len(p25), rate, 3000, 7)
	if err := siglab.WriteCapture(fmPath, analog, siglab.FormatF32); err != nil {
		t.Fatalf("write fm: %v", err)
	}

	captures := []CaptureInput{
		{Path: p25Path, Format: siglab.FormatF32, SampleRateHz: rate, FrequencyHz: 851_000_000},
		{Path: fmPath, Format: siglab.FormatF32, SampleRateHz: rate, FrequencyHz: 154_000_000},
	}
	sv, reports, err := RunOfflineSurvey(captures, LiveHuntOptions{MinConfidence: 0.3, Name: "Offline Survey"})
	if err != nil {
		t.Fatalf("RunOfflineSurvey: %v", err)
	}
	if len(sv.Signals) != 2 {
		t.Fatalf("len(Signals) = %d, want 2", len(sv.Signals))
	}

	byFreq := map[uint32]DetectedSignal{}
	for _, s := range sv.Signals {
		byFreq[s.FreqHz] = s
	}
	if c := byFreq[851_000_000].Class; c != survey.ClassTrunkControl {
		t.Errorf("p25 capture class = %q, want trunk-control", c)
	}
	if sv.System == nil || len(sv.System.Sites) == 0 {
		t.Errorf("expected a discovered system from the offline survey")
	}
	if fm := byFreq[154_000_000]; fm.Class != survey.ClassNBFM || fm.Analog == nil || !fm.Analog.Active {
		t.Errorf("fm capture: class=%q analog=%+v, want active nbfm", fm.Class, fm.Analog)
	}
	if len(reports) == 0 {
		t.Errorf("expected a trunking report from the offline survey")
	}

	// Audio capture also works offline when a directory is supplied.
	audioDir := t.TempDir()
	if _, _, err := RunOfflineSurvey(captures, LiveHuntOptions{MinConfidence: 0.3, SurveyAudioDir: audioDir}); err != nil {
		t.Fatalf("RunOfflineSurvey with audio: %v", err)
	}
	clips, _ := filepath.Glob(filepath.Join(audioDir, "*.wav"))
	if len(clips) == 0 {
		t.Errorf("expected a WAV clip for the active analog carrier")
	}
	// Sanity: the clip is a non-empty file.
	if len(clips) > 0 {
		if fi, err := os.Stat(clips[0]); err != nil || fi.Size() < 44 {
			t.Errorf("WAV clip %s missing or too small", clips[0])
		}
	}
}
