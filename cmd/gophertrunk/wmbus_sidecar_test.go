package main

import (
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// #1256 field report: `capture -sample-rate 2800000 -format u8` on a phone,
// then `wmbus -in tui.bin` printed "52.5 s of IQ at 1600000 S/s, 0 frames"
// — the headerless file was decoded at the -rate default, and the sidecar
// `capture` wrote beside it (with "protocol": "") was ignored. The sidecar's
// rate and centre must now drive the decode.
func TestWMBusFileUsesCaptureSidecar(t *testing.T) {
	data, _ := hex.DecodeString(wmbusQcaloric)
	const captureRate = 2_800_000
	const center = wmbus.ChannelHz - 25_000 // meter channel sits at +25 kHz
	iq := wmbus.Synthesize(data, wmbus.SynthOptions{Mode: wmbus.ModeT1, SampleRateHz: captureRate, OffsetHz: 25_000, NoiseRMS: 0.03})
	dir := t.TempDir()
	in := filepath.Join(dir, "tui.bin")
	if err := siglab.WriteCapture(in, iq, siglab.FormatU8); err != nil {
		t.Fatal(err)
	}
	// Exactly what the reporter's `capture` run wrote (no -protocol).
	if err := siglab.WriteMetadata(filepath.Join(dir, "tui.metadata.json"), &siglab.Metadata{
		SampleRateHz: captureRate, CenterFreqHz: center, Format: "u8",
	}); err != nil {
		t.Fatal(err)
	}

	decode := func(rate uint, format string, offset float64) (frames, crcOK uint64) {
		rateHz := float64(rate)
		rx, err := decodeWMBusFile(in, format, &rateHz, func(r float64) *wmbus.Receiver {
			rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: r, OffsetHz: offset, OnFrame: func(wmbus.Frame, wmbus.FrameInfo) {}})
			if err != nil {
				t.Fatal(err)
			}
			return rx
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		st := rx.Stats()
		return st.Framer.Frames, st.Framer.CRCOK
	}

	// The old behaviour: the -rate default, no offset.
	if _, ok := decode(1_600_000, "u8", 0); ok != 0 {
		t.Fatalf("decoding a 2.8 MS/s capture at 1.6 MS/s found %d CRC-valid frames; the fixture no longer reproduces #1256", ok)
	}

	rate, format, offset := uint(1_600_000), "u8", 0.0
	notes, err := applyWMBusSidecar(in, map[string]bool{}, &rate, &format, &offset)
	if err != nil {
		t.Fatal(err)
	}
	if rate != captureRate || offset != 25_000 {
		t.Fatalf("sidecar gave rate=%d offset=%v, want %d / 25000 (notes %q)", rate, offset, captureRate, notes)
	}
	if _, ok := decode(rate, format, offset); ok != 1 {
		t.Fatalf("with the sidecar's settings: %d CRC-valid frames, want 1", ok)
	}

	// Flags the user gave win over the sidecar.
	rate, offset = 1_000_000, 7
	if _, err := applyWMBusSidecar(in, map[string]bool{"rate": true, "offset": true}, &rate, &format, &offset); err != nil {
		t.Fatal(err)
	}
	if rate != 1_000_000 || offset != 7 {
		t.Fatalf("explicit -rate/-offset overridden: rate=%d offset=%v", rate, offset)
	}
}

// A capture with no sidecar (rtl_sdr) keeps the flags as given.
func TestWMBusFileWithoutSidecarKeepsFlags(t *testing.T) {
	in := filepath.Join(t.TempDir(), "m.cu8")
	if err := siglab.WriteCapture(in, make([]complex64, 16), siglab.FormatU8); err != nil {
		t.Fatal(err)
	}
	rate, format, offset := uint(1_600_000), "u8", 0.0
	notes, err := applyWMBusSidecar(in, map[string]bool{}, &rate, &format, &offset)
	if err != nil || notes != nil || rate != 1_600_000 || offset != 0 {
		t.Fatalf("no sidecar: rate=%d offset=%v notes=%q err=%v", rate, offset, notes, err)
	}
}
