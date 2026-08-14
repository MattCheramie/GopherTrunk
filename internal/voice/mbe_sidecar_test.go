package voice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// mkMBERecorder builds a recorder with the dsd-fme MBE sidecar enabled.
func mkMBERecorder(t *testing.T) (*Recorder, *events.Bus, string) {
	t.Helper()
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:        bus,
		OutDir:     dir,
		SampleRate: 8000,
		WriteMBE:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, bus, dir
}

func runMBECall(t *testing.T, r *Recorder, bus *events.Bus, grant trunking.Grant, frames [][]byte) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant:        grant,
		DeviceSerial: "VOICE-1",
		StartedAt:    time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && !r.HasSession("VOICE-1") {
		time.Sleep(5 * time.Millisecond)
	}
	for _, f := range frames {
		if err := r.WriteRawFrame("VOICE-1", f); err != nil {
			t.Fatal(err)
		}
	}
	bus.Publish(events.Event{
		Kind: events.KindCallEnd,
		Payload: trunking.CallEnd{
			Grant: cs.Grant, DeviceSerial: "VOICE-1",
			StartedAt: cs.StartedAt, EndedAt: cs.StartedAt.Add(time.Second),
			Reason: trunking.EndReasonNormal,
		},
	})
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && r.HasSession("VOICE-1") {
		time.Sleep(5 * time.Millisecond)
	}
}

// readAmbeSidecar mirrors dsd-fme's openMbeInFile + readAmbe2450Data
// (src/dsd_file.c): a 4-byte cookie, then per frame 1 errs byte, 6 bytes of
// bits 0..47 and a final byte whose LSB is bit 48. It reconstructs GT's
// 7-byte .raw packing (bit 48 in the MSB of byte 6) so the round-trip can be
// compared bit-exactly against the frames the recorder was fed.
func readAmbeSidecar(t *testing.T, path string) [][]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if len(data) < 4 || string(data[:4]) != ".amb" {
		t.Fatalf("cookie = %q, want %q", data[:min(4, len(data))], ".amb")
	}
	body := data[4:]
	if len(body)%8 != 0 {
		t.Fatalf("record stream length %d not a multiple of 8", len(body))
	}
	var frames [][]byte
	for off := 0; off < len(body); off += 8 {
		rec := body[off : off+8]
		if rec[0] != 0 {
			t.Errorf("record %d errs byte = %d, want 0", off/8, rec[0])
		}
		frame := make([]byte, 7)
		copy(frame, rec[1:7])
		frame[6] = (rec[7] & 1) << 7
		frames = append(frames, frame)
	}
	return frames
}

// TestRecorderMBESidecarAMB drives a DMR call through the recorder with
// recordings.mbe_files enabled and verifies the .amb sidecar is dsd-fme's
// container, bit-exactly recoverable to the AMBE+2 frames that were written.
func TestRecorderMBESidecarAMB(t *testing.T) {
	r, bus, dir := mkMBERecorder(t)
	defer r.Close()
	defer bus.Close()

	// A real on-air AMBE+2 silence frame (49 bits; bit 48 set) plus a
	// bit-pattern frame exercising both byte edges.
	frames := [][]byte{
		{0xF8, 0x05, 0xC1, 0x8F, 0xA4, 0x54, 0x80},
		{0x01, 0xFE, 0x00, 0xFF, 0xAA, 0x55, 0x00},
	}
	runMBECall(t, r, bus, trunking.Grant{System: "S", Protocol: "dmr-tier2", GroupID: 11, SourceID: 7}, frames)

	got := readAmbeSidecar(t, filepath.Join(dir, "S", "11", "20260505_000000_11.amb"))
	if len(got) != len(frames) {
		t.Fatalf("decoded %d frames, want %d", len(got), len(frames))
	}
	for i := range frames {
		if string(got[i]) != string(frames[i]) {
			t.Errorf("frame %d round-trip = % X, want % X", i, got[i], frames[i])
		}
	}
}

// TestRecorderMBESidecarIMB does the same for a P25 Phase 1 call: the .imb
// container is cookie + [errs][11 bytes] records, payload verbatim.
func TestRecorderMBESidecarIMB(t *testing.T) {
	r, bus, dir := mkMBERecorder(t)
	defer r.Close()
	defer bus.Close()

	frame := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD}
	runMBECall(t, r, bus, trunking.Grant{System: "S", Protocol: "p25", GroupID: 42, SourceID: 7}, [][]byte{frame})

	data, err := os.ReadFile(filepath.Join(dir, "S", "42", "20260505_000000_42.imb"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data[:4]) != ".imb" {
		t.Fatalf("cookie = %q, want .imb", data[:4])
	}
	body := data[4:]
	if len(body) != 12 {
		t.Fatalf("body length = %d, want 12 (1 errs + 11 payload)", len(body))
	}
	if body[0] != 0 {
		t.Errorf("errs byte = %d, want 0", body[0])
	}
	if string(body[1:]) != string(frame) {
		t.Errorf("payload = % X, want % X", body[1:], frame)
	}
}

// TestRecorderMBESidecarSkipsUnsupportedProtocol pins that protocols dsd-fme
// cannot play (TETRA ACELP here) produce no MBE sidecar even with the option
// on — and that a malformed frame size cannot desync a stream that IS open.
func TestRecorderMBESidecarSkipsUnsupportedProtocol(t *testing.T) {
	r, bus, dir := mkMBERecorder(t)
	defer r.Close()
	defer bus.Close()

	// 18-byte TETRA ACELP speech frame shape.
	frame := make([]byte, 18)
	runMBECall(t, r, bus, trunking.Grant{System: "S", Protocol: "tetra", GroupID: 9, SourceID: 7}, [][]byte{frame})

	entries, err := os.ReadDir(filepath.Join(dir, "S", "9"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); ext == ".amb" || ext == ".imb" {
			t.Errorf("unexpected MBE sidecar %s for TETRA call", e.Name())
		}
	}
}

// TestRecorderMBESidecarSkipsMalformedFrame: a wrong-size frame is dropped
// from the MBE stream (not written) so later records stay readable.
func TestRecorderMBESidecarSkipsMalformedFrame(t *testing.T) {
	r, bus, dir := mkMBERecorder(t)
	defer r.Close()
	defer bus.Close()

	good := []byte{0xF8, 0x05, 0xC1, 0x8F, 0xA4, 0x54, 0x80}
	frames := [][]byte{good, {0x01, 0x02}, good} // middle frame malformed
	runMBECall(t, r, bus, trunking.Grant{System: "S", Protocol: "dmr-tier2", GroupID: 12, SourceID: 7}, frames)

	got := readAmbeSidecar(t, filepath.Join(dir, "S", "12", "20260505_000000_12.amb"))
	if len(got) != 2 {
		t.Fatalf("decoded %d frames, want 2 (malformed skipped)", len(got))
	}
}

// TestRemoveSessionFilesIncludesMBE pins that discarding a recording (dead-key
// / encrypted abort paths both call removeSessionFiles) deletes the MBE
// sidecar along with the .wav/.raw.
func TestRemoveSessionFilesIncludesMBE(t *testing.T) {
	r, _, _ := mkMBERecorder(t)
	defer r.Close()
	dir := t.TempDir()
	s := &recordingSession{
		wavPath: filepath.Join(dir, "a.wav"),
		rawPath: filepath.Join(dir, "a.raw"),
		mbePath: filepath.Join(dir, "a.amb"),
	}
	for _, p := range []string{s.wavPath, s.rawPath, s.mbePath} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r.removeSessionFiles(s)
	for _, p := range []string{s.wavPath, s.rawPath, s.mbePath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists after removeSessionFiles", filepath.Base(p))
		}
	}
}
