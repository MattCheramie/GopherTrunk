package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestReplayVoiceSourceFanoutCopiesAndSurvivesCancel pins the replay voice
// source's contract: a pushed chunk is delivered as an independent copy to each
// subscriber, and after a subscriber's ctx is cancelled a further push neither
// blocks nor panics (no send on a closed channel).
func TestReplayVoiceSourceFanoutCopiesAndSurvivesCancel(t *testing.T) {
	s := newReplayVoiceSource(48000)
	if s.SampleRateHz() != 48000 {
		t.Fatalf("SampleRateHz = %d, want 48000", s.SampleRateHz())
	}
	if !s.CanTune(123456789) {
		t.Fatal("CanTune should accept any grant for a single-carrier replay")
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	src := []complex64{1 + 2i, 3 + 4i}
	go s.push(src, 48000)
	got := <-ch
	if len(got) != 2 || got[0] != 1+2i || got[1] != 3+4i {
		t.Fatalf("delivered chunk = %v, want [1+2i 3+4i]", got)
	}
	got[0] = 0 // mutating the delivered copy must not affect the source slice
	if src[0] != 1+2i {
		t.Fatal("push did not copy: mutating the delivered chunk changed the source")
	}

	cancel()
	// Let the unsubscribe goroutine run, then confirm a further push returns
	// promptly (the sub is gone / its done channel is selected) rather than
	// blocking or panicking.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			s.push([]complex64{9}, 48000)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("push blocked after subscriber cancel")
	}
}

// TestReplayRecordVoiceEndToEnd is the skip-guarded end-to-end reproduction for
// #1036: it runs a real DMR IPSC capture through the production voice path wired
// by replay -record-voice (siglab decode → grant bus → engine → composer →
// recorder) and asserts a non-empty .wav is produced — i.e. conventional DMR
// voice records from a capture. Set GT_DMR_IQ (cs16) [+ GT_DMR_IQ_RATE].
func TestReplayRecordVoiceEndToEnd(t *testing.T) {
	path := os.Getenv("GT_DMR_IQ")
	if path == "" {
		t.Skip("set GT_DMR_IQ (cs16 IQ) [+ GT_DMR_IQ_RATE] to run the replay -record-voice end-to-end test")
	}
	rate := 50000.0
	if v := os.Getenv("GT_DMR_IQ_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_DMR_IQ_RATE: %v", err)
		}
		rate = f
	}
	outDir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	proto := trunking.ProtocolDMRTier2
	ddcRate := rate
	if target := ccdecoder.DDCTargetForProtocol(proto); rate != target {
		ddcRate = target
	}
	rig, err := setupReplayVoice(outDir, ddcRate, 3500*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	cfg := siglab.Config{
		Protocol:     proto,
		SystemName:   "replay-voice-test",
		FrequencyHz:  443237500, // non-zero so grants bind (freq 0 is dropped)
		SampleRateHz: rate,
		Format:       siglab.FormatS16,
		Log:          log,
		Bus:          rig.bus,
		OnChannelIQ:  rig.onChannelIQ,
	}
	if _, err := siglab.Run(path, cfg); err != nil {
		t.Fatalf("siglab.Run: %v", err)
	}
	rig.finalize()

	var wavs []string
	_ = filepath.Walk(outDir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Ext(p) == ".wav" && info.Size() > 1024 {
			wavs = append(wavs, p)
		}
		return nil
	})
	if len(wavs) == 0 {
		t.Fatal("replay -record-voice produced no non-empty .wav — conventional DMR voice did not record (#1036)")
	}
	t.Logf("recorded %d wav(s), e.g. %s", len(wavs), filepath.Base(wavs[0]))
}
