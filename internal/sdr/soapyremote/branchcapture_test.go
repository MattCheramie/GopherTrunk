package soapyremote

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// readCS16 decodes a headerless cs16 branch file back to complex64.
func readCS16(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(raw)%4 != 0 {
		t.Fatalf("%s: %d bytes is not a whole number of cs16 samples", path, len(raw))
	}
	out := make([]complex64, len(raw)/4)
	for i := range out {
		iv := int16(binary.LittleEndian.Uint16(raw[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}

func readBranchMeta(t *testing.T, prefix string) branchCaptureMeta {
	t.Helper()
	raw, err := os.ReadFile(prefix + ".diversity.json")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var m branchCaptureMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse sidecar: %v", err)
	}
	return m
}

// TestBranchRecorderWritesBothBranches pins the basic contract: the capture is
// the pre-combine per-branch IQ, sample-for-sample, one file per receiver.
func TestBranchRecorderWritesBothBranches(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	rng := rand.New(rand.NewSource(1))
	ch0 := mrcSignal(rng, 512, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	rec, err := newBranchRecorder(prefix, "cs16", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	m.rec = rec

	const rounds = 4
	for i := 0; i < rounds; i++ {
		m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	}
	rec.finish()

	got0 := readCS16(t, prefix+".br0.cs16")
	got1 := readCS16(t, prefix+".br1.cs16")
	if len(got0) != rounds*len(ch0) || len(got1) != rounds*len(ch1) {
		t.Fatalf("branch lengths = %d/%d, want %d each", len(got0), len(got1), rounds*len(ch0))
	}
	if !approxEqualC(got0[:len(ch0)], ch0, 2e-4) {
		t.Error("branch 0 file does not carry the branch-0 samples")
	}
	if !approxEqualC(got1[:len(ch1)], ch1, 2e-4) {
		t.Error("branch 1 file does not carry the branch-1 samples")
	}

	meta := readBranchMeta(t, prefix)
	if meta.SamplesPerBranch != int64(rounds*len(ch0)) {
		t.Errorf("sidecar samples_per_branch = %d, want %d", meta.SamplesPerBranch, rounds*len(ch0))
	}
	if meta.Datagrams != rounds || meta.DroppedDatagrams != 0 {
		t.Errorf("sidecar datagrams = %d, dropped = %d, want %d/0", meta.Datagrams, meta.DroppedDatagrams, rounds)
	}
	if meta.Branches != 2 || meta.Format != "cs16" {
		t.Errorf("sidecar = %+v, want 2 cs16 branches", meta)
	}
}

// TestBranchRecorderFLACIsBitExactTwin pins the diversity_capture_format: flac
// contract: the .flac branches decode to EXACTLY the int16 samples the cs16
// recorder writes for the same input (same clamp, same scale), the alignment
// invariant holds (equal lengths, degenerate datagrams dropped from both), and
// the sidecar says flac so the offline harness picks the right loader.
func TestBranchRecorderFLACIsBitExactTwin(t *testing.T) {
	dir := t.TempDir()
	rng := rand.New(rand.NewSource(7))
	ch0 := mrcSignal(rng, 512, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))

	recCS, err := newBranchRecorder(filepath.Join(dir, "cs"), "cs16", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder cs16: %v", err)
	}
	recFL, err := newBranchRecorder(filepath.Join(dir, "fl"), "flac", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder flac: %v", err)
	}

	const rounds = 3
	for i := 0; i < rounds; i++ {
		recCS.write([][]complex64{ch0, ch1})
		recFL.write([][]complex64{ch0, ch1})
		// A degenerate (single-branch) datagram must be dropped from BOTH files
		// on both containers — a short write would desynchronise the branches.
		recCS.write([][]complex64{ch0})
		recFL.write([][]complex64{ch0})
	}
	recCS.finish()
	recFL.finish()

	metaFL := readBranchMeta(t, filepath.Join(dir, "fl"))
	if metaFL.Format != "flac" {
		t.Fatalf("flac sidecar format = %q, want flac", metaFL.Format)
	}
	if metaFL.SamplesPerBranch != int64(rounds*len(ch0)) || metaFL.DroppedDatagrams != rounds {
		t.Fatalf("flac sidecar samples=%d dropped=%d, want %d/%d",
			metaFL.SamplesPerBranch, metaFL.DroppedDatagrams, rounds*len(ch0), rounds)
	}
	for br := 0; br < 2; br++ {
		flacPath := filepath.Join(dir, "fl.br"+strconv.Itoa(br)+".flac")
		magic := make([]byte, 4)
		f, err := os.Open(flacPath)
		if err != nil {
			t.Fatalf("open %s: %v", flacPath, err)
		}
		if _, err := f.Read(magic); err != nil || string(magic) != "fLaC" {
			t.Errorf("%s does not start with the fLaC marker (got %q, err %v)", flacPath, magic, err)
		}
		f.Close()
		got, rate, err := baseband.ReadIQFLACSamples(flacPath)
		if err != nil {
			t.Fatalf("decode %s: %v", flacPath, err)
		}
		if rate != 200000 {
			t.Errorf("flac branch %d sample rate = %d, want 200000", br, rate)
		}
		want := readCS16(t, filepath.Join(dir, "cs.br"+strconv.Itoa(br)+".cs16"))
		if len(got) != len(want) {
			t.Fatalf("flac branch %d decodes %d samples, cs16 twin has %d", br, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("flac branch %d sample %d = %v, cs16 twin = %v — containers are not bit-exact", br, i, got[i], want[i])
			}
		}
	}
}

// TestBranchRecorderFLACRejectsUnencodableRate: FLAC's STREAMINFO cannot carry a
// multi-MS/s rate; the constructor refuses so startBranchCapture's cs16 fallback
// (rather than a broken stream) handles it.
func TestBranchRecorderFLACRejectsUnencodableRate(t *testing.T) {
	if _, err := newBranchRecorder(filepath.Join(t.TempDir(), "x"), "flac", 2, 0, 6_250_000, testLogger()); err == nil {
		t.Fatal("flac branch recorder accepted a 6.25 MS/s rate; STREAMINFO cannot carry it")
	}
}

// TestBranchRecorderInterruptedCaptureStillWritesSidecar: a capture cut short
// of its budget (daemon stopped mid-dump — device.Close calls finish) must
// still flush and write the sidecar with the partial sample count. Without
// this, an operator's interrupted dump left branch files with no sidecar,
// which the offline harness refuses to load — a real 64 s / 120 s capture
// arrived unusable exactly this way.
func TestBranchRecorderInterruptedCaptureStillWritesSidecar(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	// Budget far larger than what is written: the capture is "in flight".
	rec, err := newBranchRecorder(prefix, "cs16", 2, 1_000_000, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	rng := rand.New(rand.NewSource(3))
	ch0 := mrcSignal(rng, 256, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))
	rec.write([][]complex64{ch0, ch1})
	rec.finish() // what device.Close does on teardown

	meta := readBranchMeta(t, prefix) // Fatal here = no sidecar (pre-fix behaviour)
	if meta.SamplesPerBranch != int64(len(ch0)) {
		t.Errorf("sidecar samples_per_branch = %d, want the partial %d", meta.SamplesPerBranch, len(ch0))
	}
	if got := readCS16(t, prefix+".br0.cs16"); len(got) != len(ch0) {
		t.Errorf("branch 0 holds %d samples, want the flushed %d", len(got), len(ch0))
	}
}

// TestBranchCaptureSidecarWireNames pins the JSON key names the sidecar emits.
// The offline harness (cmd/gophertrunk/diversity_replay_test.go) declares its
// own struct with these same names, so a rename here would silently produce
// zero-valued fields there — a capture that loads but reports nothing. Both
// sides assert against these literals rather than against each other.
func TestBranchCaptureSidecarWireNames(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	rec, err := newBranchRecorder(prefix, "cs16", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	rec.meta.Addr = "10.0.0.1:23313"
	rec.meta.SampleRateHz = 1e6
	rec.meta.DiversityMode = "tracking"
	rec.write([][]complex64{{complex(0.1, 0.1)}, {complex(0.2, 0.2)}})
	rec.finish()

	raw, err := os.ReadFile(prefix + ".diversity.json")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parse sidecar: %v", err)
	}
	for _, key := range []string{
		"addr", "sample_rate_hz", "format", "diversity_mode",
		"branches", "branch_files", "samples_per_branch",
		"datagrams", "dropped_datagrams",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("sidecar is missing key %q", key)
		}
	}
	if files, ok := got["branch_files"].([]any); !ok || len(files) != 2 {
		t.Errorf("branch_files = %v, want two entries", got["branch_files"])
	}
}

// TestBranchRecorderKeepsBranchesAligned is the invariant that makes the
// capture usable at all: sample i of one file must be simultaneous with sample
// i of the other. A datagram that did not carry every branch is dropped from
// BOTH files rather than written short — writing it would silently desynchronise
// them, and every conclusion drawn from the capture afterwards would be wrong
// with nothing to show for it.
func TestBranchRecorderKeepsBranchesAligned(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	rng := rand.New(rand.NewSource(2))
	ch0 := mrcSignal(rng, 256, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))
	single := mrcSignal(rng, 256, 0.4)

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	rec, err := newBranchRecorder(prefix, "cs16", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	m.rec = rec

	m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	// A payload the server delivered only one channel of: deinterleave reports
	// a single branch, and it must not reach either file.
	m.combine(encodeCS16(single), len(single))
	m.combine(twoBranchPayload(ch0, ch1), len(ch0))
	rec.finish()

	got0 := readCS16(t, prefix+".br0.cs16")
	got1 := readCS16(t, prefix+".br1.cs16")
	if len(got0) != len(got1) {
		t.Fatalf("branch files desynchronised: %d vs %d samples", len(got0), len(got1))
	}
	if len(got0) != 2*len(ch0) {
		t.Errorf("wrote %d samples/branch, want %d (the degenerate datagram must be dropped)", len(got0), 2*len(ch0))
	}
	if meta := readBranchMeta(t, prefix); meta.DroppedDatagrams != 1 {
		t.Errorf("sidecar dropped_datagrams = %d, want 1", meta.DroppedDatagrams)
	}
}

// TestBranchRecorderStopsAtBudget pins that the dump is one-shot and bounded,
// and — more importantly — that the stream keeps flowing unchanged once it
// stops. A diagnostic tap that perturbs the signal path is worse than none.
func TestBranchRecorderStopsAtBudget(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	rng := rand.New(rand.NewSource(3))
	ch0 := mrcSignal(rng, 100, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	rec, err := newBranchRecorder(prefix, "cs16", 2, 250, 200000, testLogger()) // 2.5 datagrams
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	m.rec = rec

	for i := 0; i < 20; i++ {
		out := m.combine(twoBranchPayload(ch0, ch1), len(ch0))
		if len(out) != len(ch0) {
			t.Fatalf("round %d: combined chunk len = %d, want %d — the capture is disturbing the stream",
				i, len(out), len(ch0))
		}
	}

	got0 := readCS16(t, prefix+".br0.cs16")
	got1 := readCS16(t, prefix+".br1.cs16")
	if len(got0) != 250 || len(got1) != 250 {
		t.Errorf("branch lengths = %d/%d, want 250 each (budget)", len(got0), len(got1))
	}
	if meta := readBranchMeta(t, prefix); meta.SamplesPerBranch != 250 {
		t.Errorf("sidecar samples_per_branch = %d, want 250", meta.SamplesPerBranch)
	}
}

// TestBranchRecorderNilIsNoOp pins that the ordinary (uncaptured) path costs
// nothing and cannot panic.
func TestBranchRecorderNilIsNoOp(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	ch0 := mrcSignal(rng, 64, 0.4)
	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)
	if m.rec != nil {
		t.Fatal("a combiner without a configured capture must have no recorder")
	}
	m.combine(twoBranchPayload(ch0, ch0), len(ch0)) // must not panic
}

// TestEncodeCS16IntoRoundTrips pins the encoder against the driver's own
// decoder, including the clamp — a combined sample can exceed full scale and
// must saturate rather than wrap to the opposite rail.
func TestEncodeCS16IntoRoundTrips(t *testing.T) {
	in := []complex64{
		complex(0.5, -0.5), complex(0, 0), complex(-0.25, 0.75),
		complex(2.0, -2.0), // beyond full scale: must clamp, not wrap
	}
	buf := make([]byte, len(in)*4)
	encodeCS16Into(buf, in)
	got := convertCS16(buf)
	for i := 0; i < 3; i++ {
		if math.Abs(float64(real(got[i]-in[i]))) > 2e-4 || math.Abs(float64(imag(got[i]-in[i]))) > 2e-4 {
			t.Errorf("sample %d = %v, want %v", i, got[i], in[i])
		}
	}
	if real(got[3]) < 0.99 || imag(got[3]) > -0.99 {
		t.Errorf("over-scale sample = %v, want saturation near (+1,-1)", got[3])
	}
}

// TestBranchCaptureArmsOncePerDevice pins that a re-opened stream does not
// truncate an existing capture. A control-channel hunt re-opens the stream every
// few seconds, so re-arming would leave the operator with whatever the last
// cycle happened to catch instead of the transmission they were recording.
func TestBranchCaptureArmsOncePerDevice(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "cap")
	rng := rand.New(rand.NewSource(5))
	ch0 := mrcSignal(rng, 128, 0.4)
	ch1 := scaleC(ch0, complex(0, 1))

	d := &device{
		log:           testLogger(),
		capturePrefix: prefix,
		diversity:     diversityMRCTracking,
		mrc:           newMRCCombiner(formatCS16, diversityMRCTracking, 0),
	}
	d.startBranchCapture(48000)
	first := d.mrc.rec
	if first == nil {
		t.Fatal("capture did not arm")
	}
	d.mrc.combine(twoBranchPayload(ch0, ch1), len(ch0))

	// A second stream on the same device must not replace the recorder.
	d.startBranchCapture(48000)
	if d.mrc.rec != first {
		t.Error("re-opening the stream re-armed the capture, truncating the files")
	}
	d.mrc.rec.finish()
	if got := readCS16(t, prefix+".br0.cs16"); len(got) != len(ch0) {
		t.Errorf("captured %d samples, want %d", len(got), len(ch0))
	}
}

// TestBranchRecorderCreatesMissingDirectory pins the auto-create contract: the
// config validator no longer hard-fails on a missing diversity_capture
// directory, so the recorder must create it like every other output path.
// Fails against the old code with "no such file or directory".
func TestBranchRecorderCreatesMissingDirectory(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "does", "not", "exist", "cap")
	rec, err := newBranchRecorder(prefix, "cs16", 2, 0, 200000, testLogger())
	if err != nil {
		t.Fatalf("newBranchRecorder: %v", err)
	}
	rec.finish()
	if _, err := os.Stat(prefix + ".br0.cs16"); err != nil {
		t.Errorf("branch file not created under auto-created dir: %v", err)
	}
}

// TestBranchRecorderDirectoryPrefixIsNotHidden pins the operator-visible
// naming: a diversity_capture value that names a DIRECTORY (trailing slash, or
// an existing dir) must produce a timestamped basename inside it — not
// prefix+".br0.cs16", which yields hidden dot-files. An operator shipped a
// capture as ".br0.cs16"/".br1.cs16"/".diversity.json" that plain ls could not
// see; this fails against the old code.
func TestBranchRecorderDirectoryPrefixIsNotHidden(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix func(dir string) string
	}{
		{"trailing separator", func(dir string) string { return dir + string(os.PathSeparator) }},
		{"existing directory", func(dir string) string { return dir }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			rec, err := newBranchRecorder(tc.prefix(dir), "cs16", 2, 0, 200000, testLogger())
			if err != nil {
				t.Fatalf("newBranchRecorder: %v", err)
			}
			rec.finish()
			ents, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(ents) == 0 {
				t.Fatal("no capture files written into the directory")
			}
			for _, e := range ents {
				if strings.HasPrefix(e.Name(), ".") {
					t.Errorf("hidden capture file %q — directory prefix must get a generated basename", e.Name())
				}
				if !strings.HasPrefix(e.Name(), "diversity_") {
					t.Errorf("capture file %q does not carry the generated diversity_<timestamp> basename", e.Name())
				}
			}
		})
	}
}

// TestResolveCapturePrefixPassthrough: a plain filename prefix is untouched, so
// existing configs keep their exact output names.
func TestResolveCapturePrefixPassthrough(t *testing.T) {
	now := time.Date(2026, 8, 17, 20, 23, 23, 0, time.UTC)
	if got := resolveCapturePrefix("captures/cap1", now); got != "captures/cap1" {
		t.Errorf("plain prefix rewritten to %q", got)
	}
	want := filepath.Join("captures", "diversity_20260817T202323Z")
	if got := resolveCapturePrefix("captures"+string(os.PathSeparator), now); got != want {
		t.Errorf("directory prefix resolved to %q, want %q", got, want)
	}
}
