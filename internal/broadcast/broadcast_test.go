package broadcast

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// writeWAV emits a 16-bit PCM mono WAV of seconds of 440 Hz tone at
// 8 kHz and returns its path. Used as a Call.AudioPath fixture.
func writeWAV(t *testing.T, seconds float64) string {
	t.Helper()
	const rate = 8000
	n := int(float64(rate) * seconds)
	samples := make([]int16, n)
	for i := range samples {
		samples[i] = int16(8000 * math.Sin(2*math.Pi*440*float64(i)/float64(rate)))
	}
	path := filepath.Join(t.TempDir(), "call.wav")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dataLen := uint32(n * 2)
	hdr := make([]byte, 44)
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:8], 36+dataLen)
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], 16)
	binary.LittleEndian.PutUint16(hdr[20:22], 1)
	binary.LittleEndian.PutUint16(hdr[22:24], 1)
	binary.LittleEndian.PutUint32(hdr[24:28], rate)
	binary.LittleEndian.PutUint32(hdr[28:32], rate*2)
	binary.LittleEndian.PutUint16(hdr[32:34], 2)
	binary.LittleEndian.PutUint16(hdr[34:36], 16)
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:44], dataLen)
	if _, err := f.Write(hdr); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, dataLen)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(s))
	}
	if _, err := f.Write(buf); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCallMP3CachesEncode(t *testing.T) {
	c := &Call{AudioPath: writeWAV(t, 0.5), SampleRate: 8000}
	first, err := c.MP3()
	if err != nil {
		t.Fatalf("first MP3: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("first MP3 empty")
	}
	second, err := c.MP3()
	if err != nil {
		t.Fatalf("second MP3: %v", err)
	}
	// The cache returns the same backing array, not a re-encode.
	if &first[0] != &second[0] {
		t.Fatal("MP3() re-encoded instead of caching")
	}
}

func TestCallMP3NormalizesDistributed(t *testing.T) {
	path := writeWAV(t, 1.0) // quiet tone (~-26 dBFS)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := (&Call{AudioPath: path, SampleRate: 8000}).MP3()
	if err != nil {
		t.Fatalf("plain MP3: %v", err)
	}
	norm, err := (&Call{
		AudioPath: path, SampleRate: 8000,
		normalize: NormalizeConfig{Enabled: true, Params: loudness.NormalizeParams{}.WithDefaults()},
	}).MP3()
	if err != nil {
		t.Fatalf("normalized MP3: %v", err)
	}

	// The on-disk WAV must be left pristine — normalization is in memory.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("distributed normalization mutated the on-disk WAV")
	}
	// A quiet tone gets boosted, so the normalized MP3 must differ.
	if bytes.Equal(plain, norm) {
		t.Fatal("normalized MP3 is identical to the un-normalized encode (no gain applied)")
	}
}

func TestCallMP3ReportsEncodeError(t *testing.T) {
	c := &Call{AudioPath: "/nonexistent/call.wav"}
	if _, err := c.MP3(); err == nil {
		t.Fatal("MP3() on a missing file should error")
	}
}

func TestSystemFilter(t *testing.T) {
	open := newSystemFilter(nil)
	if !open.Accepts("anything") {
		t.Fatal("empty filter must accept every system")
	}
	scoped := newSystemFilter([]string{"Alpha", "Bravo"})
	if !scoped.Accepts("Alpha") {
		t.Fatal("scoped filter must accept a listed system")
	}
	if scoped.Accepts("Charlie") {
		t.Fatal("scoped filter must reject an unlisted system")
	}
}

func TestCallFromEvent(t *testing.T) {
	start := time.Now().Add(-3 * time.Second)
	end := time.Now()
	cc := trunking.CallComplete{
		Grant: trunking.Grant{
			System:      "Metro",
			Protocol:    "p25",
			GroupID:     1234,
			SourceID:    5678,
			FrequencyHz: 851_012_500,
			Emergency:   true,
		},
		Talkgroup:  &trunking.TalkGroup{ID: 1234, AlphaTag: "Dispatch"},
		StartedAt:  start,
		EndedAt:    end,
		AudioPath:  "/tmp/call.wav",
		SampleRate: 8000,
	}
	c := callFromEvent(cc)
	if c.System != "Metro" || c.Talkgroup != 1234 || c.Source != 5678 {
		t.Fatalf("metadata not copied: %+v", c)
	}
	if c.TalkgroupLabel != "Dispatch" {
		t.Fatalf("talkgroup label = %q, want Dispatch", c.TalkgroupLabel)
	}
	if !c.Emergency {
		t.Fatal("emergency flag not copied")
	}
	if c.Duration() < 2*time.Second {
		t.Fatalf("duration = %v, want ~3s", c.Duration())
	}
}

// TestCallFromEventGrantLabel covers #1105: a label carried on the grant
// itself (e.g. the conventional scanner's per-channel label) becomes the
// upload's talkgroupLabel even when no talkgroup-roster entry exists, and
// takes precedence over a roster entry when both are present.
func TestCallFromEventGrantLabel(t *testing.T) {
	// No roster entry: the synthetic conventional GID resolves to no
	// TalkGroup, so before #1105 the label was empty. It must now come
	// from the grant.
	noRoster := callFromEvent(trunking.CallComplete{
		Grant: trunking.Grant{
			System:     "Scanner",
			Protocol:   "fm-conv",
			GroupID:    0x80000000,
			GroupLabel: "KP4-DH",
		},
	})
	if noRoster.TalkgroupLabel != "KP4-DH" {
		t.Fatalf("grant label (no roster): got %q, want KP4-DH", noRoster.TalkgroupLabel)
	}

	// Roster entry present too: the per-grant label wins.
	withRoster := callFromEvent(trunking.CallComplete{
		Grant: trunking.Grant{
			System:     "Scanner",
			Protocol:   "fm-conv",
			GroupID:    0x80000001,
			GroupLabel: "V2M Simplex",
		},
		Talkgroup: &trunking.TalkGroup{ID: 0x80000001, AlphaTag: "Roster Name"},
	})
	if withRoster.TalkgroupLabel != "V2M Simplex" {
		t.Fatalf("grant label (with roster): got %q, want V2M Simplex", withRoster.TalkgroupLabel)
	}
}
