package wmbus

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// loadIQWav reads a 16-bit stereo IQ WAV (what `wmbus -save-frames`
// writes) and returns its samples and rate.
func loadIQWav(t *testing.T, path string) ([]complex64, float64) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		t.Fatalf("%s: not a WAV file", path)
	}
	var rate float64
	for p := 12; p+8 <= len(raw); {
		id, n := string(raw[p:p+4]), int(binary.LittleEndian.Uint32(raw[p+4:]))
		body := raw[p+8 : min(p+8+n, len(raw))]
		switch id {
		case "fmt ":
			if binary.LittleEndian.Uint16(body[0:]) != 1 || binary.LittleEndian.Uint16(body[2:]) != 2 || binary.LittleEndian.Uint16(body[14:]) != 16 {
				t.Fatalf("%s: want 16-bit stereo PCM", path)
			}
			rate = float64(binary.LittleEndian.Uint32(body[4:]))
		case "data":
			iq := make([]complex64, len(body)/4)
			for i := range iq {
				re := int16(binary.LittleEndian.Uint16(body[4*i:]))
				im := int16(binary.LittleEndian.Uint16(body[4*i+2:]))
				iq[i] = complex(float32(re)/32768, float32(im)/32768)
			}
			return iq, rate
		}
		p += 8 + n + n&1
	}
	t.Fatalf("%s: no data chunk", path)
	return nil, 0
}

// TestWMBusRealAirSnippets replays a directory of `wmbus -save-frames`
// snippets (each an IQ .wav around one telegram) and counts the snippets
// that yield a CRC-valid frame: the yield metric for receiver changes.
// Skip-guarded:
//
//	GT_WMBUS_SNIPPETS=<dir> [GT_WMBUS_OFFSET=<Hz>] [GT_WMBUS_MIN=<n>]
//	go test ./internal/radio/wmbus -run RealAirSnippets -v
//
// It counts two ways. "cold" gives every snippet a fresh receiver, so
// nothing is learned. "live" feeds them in file-name (time) order through
// one receiver, as the run that saved them heard them, so the carrier
// estimate one telegram leaves behind serves the next; GT_WMBUS_MIN
// checks the live count.
//
// Issue #1256: the reporter's 156 snippets from a Samsung S5 + RTL-SDR
// walking past Qundis heat cost allocators and Diehl water meters (20 Oct,
// saved with -all; 82 failed their CRC on v1.3.1). That tuner hears the
// meters about 50 kHz low.
func TestWMBusRealAirSnippets(t *testing.T) {
	dir := os.Getenv("GT_WMBUS_SNIPPETS")
	if dir == "" {
		t.Skip("GT_WMBUS_SNIPPETS not set")
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatalf("no .wav snippets in %s", dir)
	}
	offHz, _ := strconv.ParseFloat(os.Getenv("GT_WMBUS_OFFSET"), 64)
	snippets := make([][]complex64, len(paths))
	var rate float64
	for i, p := range paths {
		var r float64
		snippets[i], r = loadIQWav(t, p)
		if rate != 0 && r != rate {
			t.Fatalf("%s: rate %.0f, the others %.0f", p, r, rate)
		}
		rate = r
	}

	// run feeds the given snippets through one receiver and returns which
	// of them a CRC-valid frame ended in.
	run := func(idx []int) map[int]bool {
		var ends []int64 // cumulative sample index where each snippet ends
		var total int64
		for _, i := range idx {
			total += int64(len(snippets[i]))
			ends = append(ends, total)
		}
		ok := map[int]bool{}
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OffsetHz: offHz, OnFrame: func(f Frame, info FrameInfo) {
			if !f.CRCOK {
				return
			}
			k := sort.Search(len(ends), func(k int) bool { return ends[k] > info.Sample })
			ok[idx[min(k, len(idx)-1)]] = true
		}})
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range idx {
			iq := snippets[i]
			for j := 0; j < len(iq); j += 16384 {
				rx.Process(iq[j:min(j+16384, len(iq))])
			}
		}
		rx.Flush()
		if st := rx.Stats(); st.CarrierLocked {
			t.Logf("carrier estimate %.0f Hz", st.CarrierOffsetHz)
		}
		return ok
	}

	cold := 0
	for i := range snippets {
		if run([]int{i})[i] {
			cold++
		}
	}
	all := make([]int, len(snippets))
	for i := range all {
		all[i] = i
	}
	liveOK := run(all)
	for i, p := range paths {
		if !liveOK[i] && testing.Verbose() {
			t.Logf("no CRC-valid frame: %s", filepath.Base(p))
		}
	}
	t.Logf("CRC-valid: cold %d of %d snippets, live %d", cold, len(paths), len(liveOK))
	if v := os.Getenv("GT_WMBUS_MIN"); v != "" {
		if want, _ := strconv.Atoi(v); len(liveOK) < want {
			t.Errorf("live: %d snippets decode, want at least %d", len(liveOK), want)
		}
	}
}
