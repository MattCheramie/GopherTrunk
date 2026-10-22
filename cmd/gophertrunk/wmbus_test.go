package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// An unencrypted heat cost allocator telegram from wmbusmeters' simulation
// set (qcaloric), whose readings wmbusmeters decodes as consumption 127 and
// device time 2019-02-20 11:32.
const wmbusQcaloric = "314493441234567835087a740000200b6e2701004b6e450100426c5f2ccb086e790000c2086c7f21326cffff046d200b7422"

// `gophertrunk wmbus -in` decodes a telegram from an rtl_sdr .cu8 file at
// 1.6 MS/s and from a flac IQ container at 1.024 MS/s (as `gophertrunk
// capture -format flac` writes; FLAC cannot carry rates over 1048575 Hz),
// and prints its sender and readings.
func TestWMBusDecodesCaptureFiles(t *testing.T) {
	data, _ := hex.DecodeString(wmbusQcaloric)
	const rate, flacRate = 1_600_000, 1_024_000
	synth := func(r float64) []complex64 {
		return wmbus.Synthesize(data, wmbus.SynthOptions{Mode: wmbus.ModeT1, SampleRateHz: r, OffsetHz: 25_000, NoiseRMS: 0.03})
	}
	dir := t.TempDir()

	cu8 := filepath.Join(dir, "m.cu8")
	if err := siglab.WriteCapture(cu8, synth(rate), siglab.FormatU8); err != nil {
		t.Fatal(err)
	}
	flacPath := filepath.Join(dir, "m.flac")
	f, err := os.Create(flacPath)
	if err != nil {
		t.Fatal(err)
	}
	c, err := siglab.NewIQContainer(f, siglab.FormatFLAC, flacRate)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Write(synth(flacRate)); err != nil {
		t.Fatal(err)
	}
	if err := c.Finalize(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	for _, tc := range []struct{ path, format string }{{cu8, "u8"}, {flacPath, "u8"}} {
		var out bytes.Buffer
		var tgs []wmbus.Telegram
		rateHz := 1.0 // a container must supply its own rate
		if tc.path == cu8 {
			rateHz = rate
		}
		_, err := decodeWMBusFile(tc.path, tc.format, &rateHz, func(r float64) *wmbus.Receiver {
			rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: r, OffsetHz: 25_000, OnFrame: func(fr wmbus.Frame, info wmbus.FrameInfo) {
				tg, perr := wmbus.ParseTelegram(fr)
				tgs = append(tgs, tg)
				writeWMBusTelegram(&out, "t", tg, perr, fr, info, false)
			}})
			if err != nil {
				t.Fatal(err)
			}
			return rx
		}, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if len(tgs) != 1 || !tgs[0].CRCOK || tgs[0].ID != "78563412" {
			t.Fatalf("%s: %d telegrams: %+v", tc.path, len(tgs), tgs)
		}
		for _, want := range []string{"QDS 78563412", "heat cost allocator", "heat cost allocation 127", "date time 2019-02-20 11:32"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: output lacks %q:\n%s", tc.path, want, out.String())
			}
		}
	}
}

func TestWMBusJSONLine(t *testing.T) {
	data, _ := hex.DecodeString(wmbusQcaloric)
	fr := wmbus.Frame{Mode: wmbus.ModeC1, Format: wmbus.FormatA, Data: data, CRCOK: true}
	tg, err := wmbus.ParseTelegram(fr)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writeWMBusTelegram(&out, "2026-10-10 12:00:00.000", tg, nil, fr, wmbus.FrameInfo{LevelDBFS: -30, CarrierOffsetHz: -50144.6}, true)
	var j wmbusJSON
	if err := json.Unmarshal(out.Bytes(), &j); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	if j.Manufacturer != "QDS" || j.ID != "78563412" || j.Device != "heat cost allocator" || j.Encrypted ||
		j.SecurityMode == nil || *j.SecurityMode != 0 || len(j.Records) != 7 ||
		j.Records[0].Value == nil || *j.Records[0].Value != 127 || j.CarrierOffsetHz != -50145 {
		t.Errorf("json %s", out.String())
	}
}

// A telegram that passed only after a CRC repair says so on its line, in
// its JSON and in the progress counts, so a script can skip it if a reading
// must never be wrong (issue #1256). A clean one says nothing.
func TestWMBusRepairedTelegramIsMarked(t *testing.T) {
	data, _ := hex.DecodeString(wmbusQcaloric)
	for _, bits := range []int{0, 1, 2} {
		fr := wmbus.Frame{Mode: wmbus.ModeC1, Format: wmbus.FormatA, Data: data, CRCOK: true, CorrectedBits: bits}
		tg, err := wmbus.ParseTelegram(fr)
		if err != nil {
			t.Fatal(err)
		}
		var text, js bytes.Buffer
		writeWMBusTelegram(&text, "", tg, nil, fr, wmbus.FrameInfo{}, false)
		writeWMBusTelegram(&js, "", tg, nil, fr, wmbus.FrameInfo{}, true)
		var j wmbusJSON
		if err := json.Unmarshal(js.Bytes(), &j); err != nil {
			t.Fatal(err)
		}
		marked := strings.Contains(text.String(), "CRC repaired")
		if marked != (bits > 0) || j.CorrectedBits != bits || strings.Contains(js.String(), "corrected_bits") != (bits > 0) {
			t.Errorf("%d bits: text %q json %s", bits, text.String(), js.String())
		}
		if bits == 2 && !strings.Contains(text.String(), "(CRC repaired: 2 bits)") {
			t.Errorf("text %q", text.String())
		}
	}
	st := wmbus.ReceiverStats{Framer: wmbus.FramerStats{Frames: 14, CRCOK: 12, Repaired: 2}}
	if s := wmbusStatusLine(30, st, 0, false); !strings.Contains(s, "14 frames (12 CRC-valid of which 2 repaired, 2 CRC-failed)") {
		t.Errorf("status %s", s)
	}
}

// Once a CRC-valid frame has located the meters' carrier, the progress
// line says where it is, and past 20 ppm how to centre the tuner (the
// issue #1256 S5 heard its meters 50 kHz low). Before that it says
// nothing about the carrier.
func TestWMBusStatusLineReportsCarrier(t *testing.T) {
	st := wmbus.ReceiverStats{}
	if s := wmbusStatusLine(30, st, 0, false); strings.Contains(s, "carrier") {
		t.Errorf("unlocked line mentions the carrier: %s", s)
	}
	st.CarrierLocked, st.CarrierOffsetHz = true, -50_145
	s := wmbusStatusLine(30, st, 0, false)
	if !strings.Contains(s, "carrier -50.1 kHz (-58 ppm)") || !strings.Contains(s, "-ppm ±58") {
		t.Errorf("line %s", s)
	}
	st.CarrierOffsetHz = 4_000
	if s := wmbusStatusLine(30, st, 0, false); !strings.Contains(s, "carrier +4.0 kHz (+5 ppm)") || strings.Contains(s, "-ppm") {
		t.Errorf("small offset: %s", s)
	}
}

// With -json every line of the -log file is a JSON object: progress lines
// go in as {"status": …}, so a script reading the log can parse each line
// and tell telegrams from progress (issue #1256: a Perl reader).
func TestWMBusLogStatusLineJSON(t *testing.T) {
	const line = "wmbus: listening 30s: 2 frames (1 CRC-valid, 1 CRC-failed)"
	if got := wmbusLogStatusLine(line, false); got != line {
		t.Errorf("plain: %q", got)
	}
	var j struct {
		Time   string `json:"time"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(wmbusLogStatusLine(line, true)), &j); err != nil || j.Status != line || j.Time == "" {
		t.Errorf("json: %+v err %v", j, err)
	}
}

// A CRC-failed frame says which block failed, and flags the sender as
// unreliable when that block is the header (issue #1256).
func TestWMBusCRCFailedLineNamesBlock(t *testing.T) {
	data, _ := hex.DecodeString(wmbusQcaloric)
	for _, tc := range []struct {
		failed []int
		want   string
	}{
		{[]int{1}, "CRC FAILED (block 1 of 4; block 1 is the header, so the sender shown is unreliable)"},
		{[]int{3, 4}, "CRC FAILED (blocks 3,4 of 4; header OK)"},
	} {
		fr := wmbus.Frame{Mode: wmbus.ModeC1, Format: wmbus.FormatA, Data: data, Raw: []byte{0x2e}, Blocks: 4, FailedBlocks: tc.failed}
		tg, err := wmbus.ParseTelegram(fr)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		writeWMBusTelegram(&out, "", tg, nil, fr, wmbus.FrameInfo{}, false)
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("failed %v: got %q, want it to contain %q", tc.failed, out.String(), tc.want)
		}
		out.Reset()
		writeWMBusTelegram(&out, "", tg, nil, fr, wmbus.FrameInfo{}, true)
		var j wmbusJSON
		if err := json.Unmarshal(out.Bytes(), &j); err != nil {
			t.Fatal(err)
		}
		if j.CRCBlocks != 4 || len(j.CRCFailedBlocks) != len(tc.failed) || j.HeaderCRCOK != (tc.failed[0] != 1) || j.Raw != "2e" {
			t.Errorf("failed %v: json %s", tc.failed, out.String())
		}
	}
}
