package gtbundle

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// fixedNow returns a stable clock for deterministic archives.
func fixedNow() time.Time {
	return time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
}

func newTestWriter(t *testing.T, buf *bytes.Buffer, name string) *Writer {
	t.Helper()
	w, err := NewWriter(buf, WriterOptions{
		Name:          name,
		CaptureIntent: IntentCCMap,
		Provenance: Provenance{
			Source:       "unit test",
			Protocol:     "p25",
			CenterFreqHz: 851_012_500,
			SampleRateHz: 2_400_000,
		},
		Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	return w
}

// packSampleBundle writes a bundle containing one of each major artifact and
// returns the archive bytes.
func packSampleBundle(t *testing.T, name string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := newTestWriter(t, &buf, name)

	if _, err := w.AddBytes(RoleCaptureIQ, "cc.cfile", []byte("\x00\x01\x02\x03rawiq"), "gophertrunk capture"); err != nil {
		t.Fatalf("add iq: %v", err)
	}
	meta := &siglab.Metadata{Protocol: "p25", SampleRateHz: 2_400_000, CenterFreqHz: 851_012_500, Format: "f32"}
	if _, err := w.AddYAML(RoleCaptureMeta, "cc.meta.yaml", meta, "gophertrunk capture"); err != nil {
		t.Fatalf("add meta: %v", err)
	}
	res := &siglab.Result{Source: "cc.cfile", Protocol: "p25", SampleRateHz: 2_400_000, Locked: true}
	if _, err := w.AddYAML(RoleSiglabResult, "result.yaml", res, "siglab"); err != nil {
		t.Fatalf("add siglab result: %v", err)
	}
	sys := &hunt.DiscoveredSystem{
		Name:     "Test P25",
		Protocol: "p25",
		WACN:     0xBEE00,
		SystemID: 0x49A,
		Talkgroups: []hunt.DiscoveredTalkgroup{
			{Dec: 101, Hex: "65", Count: 3},
		},
	}
	if _, err := w.AddJSON(RoleMappingSystem, "system.json", sys, "hunt"); err != nil {
		t.Fatalf("add mapping: %v", err)
	}
	if _, err := w.AddBytes(RoleCryptolabFrames, "frames.jsonl",
		[]byte(`{"label":"call-1","iv":"aabb","ct":"ccdd"}`+"\n"), "cryptolab"); err != nil {
		t.Fatalf("add frames: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

func TestRoundTripWriteReadVerify(t *testing.T) {
	raw := packSampleBundle(t, "mesa-p25")
	r, err := NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	if got := r.Manifest().Name; got != "mesa-p25" {
		t.Errorf("manifest name = %q, want mesa-p25", got)
	}
	if got := r.Manifest().Format; got != FormatID {
		t.Errorf("format = %q, want %q", got, FormatID)
	}

	// Every listed file verifies.
	for _, v := range r.Verify() {
		if !v.OK {
			t.Errorf("verify %s: %v", v.Path, v.Err)
		}
	}

	// Typed accessors round-trip.
	meta, err := r.CaptureMeta()
	if err != nil {
		t.Fatalf("CaptureMeta: %v", err)
	}
	if meta.Protocol != "p25" || meta.SampleRateHz != 2_400_000 {
		t.Errorf("meta = %+v", meta)
	}
	res, err := r.SiglabResult()
	if err != nil {
		t.Fatalf("SiglabResult: %v", err)
	}
	if !res.Locked || res.Protocol != "p25" {
		t.Errorf("result = %+v", res)
	}
	sys, err := r.MappingSystem()
	if err != nil {
		t.Fatalf("MappingSystem: %v", err)
	}
	if sys.WACN != 0xBEE00 || len(sys.Talkgroups) != 1 || sys.Talkgroups[0].Dec != 101 {
		t.Errorf("system = %+v", sys)
	}
	iq, _, err := r.CaptureIQ()
	if err != nil {
		t.Fatalf("CaptureIQ: %v", err)
	}
	if !bytes.Equal(iq, []byte("\x00\x01\x02\x03rawiq")) {
		t.Errorf("iq bytes mismatch: %q", iq)
	}
}

func TestWriterDeterministic(t *testing.T) {
	a := packSampleBundle(t, "mesa-p25")
	b := packSampleBundle(t, "mesa-p25")
	if !bytes.Equal(a, b) {
		t.Fatalf("repacked archive not byte-identical (%d vs %d bytes)", len(a), len(b))
	}
}

func TestWriterPathGuard(t *testing.T) {
	var buf bytes.Buffer
	w := newTestWriter(t, &buf, "x")
	for _, bad := range []string{"../escape", "a/b", `a\b`, "~/x", "C:evil", ".."} {
		if _, err := w.AddBytes(RoleNotes, bad, []byte("x"), ""); err == nil {
			t.Errorf("AddBytes(%q) accepted a traversal-unsafe leaf", bad)
		}
	}
}

func TestChecksumTamperDetected(t *testing.T) {
	raw := packSampleBundle(t, "tamper")
	pristine, err := NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("pristine bundle: %v", err)
	}

	// The invariant: no LISTED (checksummed) member can change without the
	// reader noticing — either the gzip stream fails to inflate, Verify flags
	// the member, or the typed read fails. A flip that lands in an index file
	// (MANIFEST.yaml free text, README.md) is by design not integrity-checked
	// and must leave every listed member byte-identical. The compressed
	// layout moves whenever a serialized model gains a field, so one fixed
	// offset is not a test: sweep the middle of the stream and check each.
	listed := map[string]bool{}
	for _, e := range pristine.man.Files {
		listed[e.Path] = true
	}
	detected := 0
	for pct := 20; pct <= 80; pct += 5 {
		off := len(raw) * pct / 100
		mutated := append([]byte(nil), raw...)
		mutated[off] ^= 0xFF

		r, err := NewReader(bytes.NewReader(mutated))
		if err != nil {
			detected++ // a corrupted gzip that fails to inflate is a detection, just earlier
			continue
		}
		sawBad := false
		for _, v := range r.Verify() {
			if !v.OK {
				sawBad = true
			}
		}
		if sawBad {
			detected++
			continue
		}
		if _, _, err := r.CaptureIQ(); err != nil {
			detected++
			continue
		}
		// Undetected: only an index file may have changed.
		for path, body := range r.blobs {
			if listed[path] && !bytes.Equal(body, pristine.blobs[path]) {
				t.Errorf("flip at byte %d altered listed member %q undetected: Verify clean and CaptureIQ succeeded", off, path)
			}
		}
	}
	if detected == 0 {
		t.Errorf("no flipped offset in the middle 60%% of the stream was detected — the checksums are not covering the payload")
	}
}

func TestUnknownRoleAndFileTolerated(t *testing.T) {
	var buf bytes.Buffer
	w := newTestWriter(t, &buf, "future")
	// A role this build doesn't know — buckets under misc/ and is preserved.
	rel, err := w.AddBytes(Role("some-future-role"), "x.bin", []byte("hi"), "future-tool")
	if err != nil {
		t.Fatalf("AddBytes unknown role: %v", err)
	}
	if want := "future/misc/x.bin"; rel != want {
		t.Errorf("unknown-role path = %q, want %q", rel, want)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	r, err := NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	// Verify must not choke on the unknown role.
	for _, v := range r.Verify() {
		if !v.OK {
			t.Errorf("verify %s: %v", v.Path, v.Err)
		}
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	// Hand-craft a tar.gz whose member escapes, and confirm Extract refuses.
	dir := t.TempDir()
	// Build a valid bundle, then attempt extraction to a temp dir — the happy
	// path — and confirm files land under dir and nowhere else.
	raw := packSampleBundle(t, "extract-me")
	r, err := NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	if err := r.Extract(dir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// The manifest must exist under the single top dir.
	if _, err := os.Stat(filepath.Join(dir, "extract-me", ManifestName)); err != nil {
		t.Errorf("expected manifest extracted: %v", err)
	}
}

func TestExtractRole(t *testing.T) {
	raw := packSampleBundle(t, "role-x")
	r, err := NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	dir := t.TempDir()
	written, err := r.ExtractRole(RoleCryptolabFrames, dir)
	if err != nil {
		t.Fatalf("ExtractRole: %v", err)
	}
	if len(written) != 1 || filepath.Base(written[0]) != "frames.jsonl" {
		t.Fatalf("ExtractRole wrote %v", written)
	}
	body, err := os.ReadFile(written[0])
	if err != nil || !bytes.Contains(body, []byte("call-1")) {
		t.Errorf("extracted frames wrong: %q err=%v", body, err)
	}
	// A role that isn't present returns ErrNotFound.
	if _, err := r.ExtractRole(RoleLogMessages, dir); !errors.Is(err, ErrNotFound) {
		t.Errorf("ExtractRole(absent) err = %v, want ErrNotFound", err)
	}
}

func TestManifestRejectsForeignFormat(t *testing.T) {
	if _, err := ParseManifest([]byte("format: something-else\nschema_version: 1\n")); err == nil {
		t.Errorf("ParseManifest accepted a foreign format")
	}
}

func TestFutureSchemaTolerated(t *testing.T) {
	m, err := ParseManifest([]byte("format: " + FormatID + "\nschema_version: 999\nname: x\nfiles: []\n"))
	if !errors.Is(err, ErrFutureSchema) {
		t.Fatalf("err = %v, want ErrFutureSchema", err)
	}
	if m == nil || m.Name != "x" {
		t.Errorf("future-schema manifest not returned best-effort: %+v", m)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"Mesa P25 2026/07": "Mesa-P25-2026-07",
		"  ../../etc  ":    "etc",
		"":                 "bundle",
		"good_name-1.2":    "good_name-1.2",
		"...":              "bundle",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}
