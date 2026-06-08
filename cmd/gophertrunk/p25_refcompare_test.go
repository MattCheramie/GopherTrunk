//go:build integration

package main

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/refcompare"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestP25ReferenceCrossCheck runs an external reference decoder (OP25 /
// DSD-FME) on the same samples/p25 capture GopherTrunk decodes and diffs the
// two — the harness's GopherTrunk-vs-reference leg (see docs/demod-calibration.md).
// It is doubly gated: it skips unless BOTH a samples/p25/*.cfile is present AND
// P25_REFERENCE_CMD names how to invoke the reference. It never runs in normal
// CI (external binary), so it is an operator/contributor instrument, not a gate.
//
// Configuration (environment):
//
//	P25_REFERENCE_CMD          shell command to decode the capture; the
//	                           substring {} is replaced with the .cfile path.
//	                           e.g. "dsd-fme -i {} -fp -o /dev/null"
//	P25_REFERENCE_MIN_AGREE    NAC-set agreement required to pass (default 1.0)
//	P25_REFERENCE_DEMOD        "c4fm" (default) or "cqpsk" for GopherTrunk's path
func TestP25ReferenceCrossCheck(t *testing.T) {
	cfilePath, metaPath := findRealairCapture(t, "p25")
	if cfilePath == "" {
		t.Skip("samples/p25/: no *.cfile present (see samples/p25/README.md)")
	}
	refCmd := os.Getenv("P25_REFERENCE_CMD")
	if refCmd == "" {
		t.Skip("P25_REFERENCE_CMD unset — set it to the reference decoder invocation (see docs/demod-calibration.md)")
	}
	meta := loadRealairMetadata(t, metaPath)

	minAgree := 1.0
	if s := os.Getenv("P25_REFERENCE_MIN_AGREE"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			minAgree = v
		}
	}
	demodMode := p25phase1rx.DemodC4FM
	rotations := p25phase1.RotationsC4FM
	if os.Getenv("P25_REFERENCE_DEMOD") == "cqpsk" {
		demodMode = p25phase1rx.DemodCQPSK
		rotations = p25phase1.RotationsAll
	}

	// GopherTrunk decode → NAC set + TSBK frame yield.
	gtNACs, gtFrames := gopherTrunkDecode(t, cfilePath, float64(meta.SampleRateHz), demodMode, rotations)

	// Reference decode → scrape NACs from its output.
	out := runReference(t, refCmd, cfilePath)
	refNACs := refcompare.ParseNACs(out, nil)

	rep := refcompare.Compare(gtNACs, refNACs, gtFrames, 0, minAgree, 0.8)
	t.Logf("P25 GT-vs-reference cross-check (%s):", cfilePath)
	t.Logf("  GopherTrunk NACs: %v (TSBKs=%d)", rep.GTNACs, rep.GTFrames)
	t.Logf("  reference NACs:   %v", rep.RefNACs)
	t.Logf("  %s", rep.Notes)
	if !rep.Pass {
		t.Errorf("cross-check failed: %s", rep.Notes)
	}
}

// gopherTrunkDecode runs the production receiver + control channel on the
// capture and returns the distinct locked NACs and the decoded-TSBK count.
func gopherTrunkDecode(t *testing.T, cfilePath string, sampleRateHz float64, mode p25phase1rx.DemodMode, rot p25phase1.RotationSet) ([]uint16, int) {
	t.Helper()
	raw, err := os.ReadFile(cfilePath)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	pairs := len(raw) / 8
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)

	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	nacSet := map[uint16]bool{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.C {
			if ev.Kind != events.KindCCLocked {
				continue
			}
			if ls, ok := ev.Payload.(p25phase1.LockState); ok {
				nacSet[ls.NAC] = true
			}
		}
	}()
	cc := p25phase1.New(p25phase1.Options{
		Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "p25-refcompare", Rotations: rot,
	})
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: sampleRateHz, DeviationHz: 1800.0, DemodMode: mode,
		DibitSink: func(d []uint8, baseIdx int) { cc.Process(d, baseIdx) },
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[i:end])
	}
	sub.Close()
	<-done

	nacs := make([]uint16, 0, len(nacSet))
	for n := range nacSet {
		nacs = append(nacs, n)
	}
	return nacs, int(cc.Stats().TSBKDecoded)
}

// runReference invokes the reference decoder, substituting {} with the capture
// path, and returns its combined stdout+stderr.
func runReference(t *testing.T, cmdTemplate, cfilePath string) string {
	t.Helper()
	cmdline := strings.ReplaceAll(cmdTemplate, "{}", cfilePath)
	cmd := exec.Command("sh", "-c", cmdline)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Many reference tools exit non-zero at EOF; the output is still
		// usable, so log rather than fail.
		t.Logf("reference command exited with %v (output still parsed)", err)
	}
	return string(out)
}
