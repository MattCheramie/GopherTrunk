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
		})
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
	writeWMBusTelegram(&out, "2026-10-10 12:00:00.000", tg, nil, fr, wmbus.FrameInfo{LevelDBFS: -30}, true)
	var j wmbusJSON
	if err := json.Unmarshal(out.Bytes(), &j); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	if j.Manufacturer != "QDS" || j.ID != "78563412" || j.Device != "heat cost allocator" || j.Encrypted ||
		j.SecurityMode == nil || *j.SecurityMode != 0 || len(j.Records) != 7 ||
		j.Records[0].Value == nil || *j.Records[0].Value != 127 {
		t.Errorf("json %s", out.String())
	}
}
