package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// wmbus -save-frames: a short IQ snippet per decoded telegram (#1256).
//
// A meter sends a telegram a few times a minute and each one lasts tens of
// milliseconds, so a whole capture is almost entirely empty air: 30 s at
// 1.6 MS/s is ~96 MB even as 8-bit IQ, too big to share from a phone. Cutting the IQ around
// each telegram keeps exactly what an offline check needs (a few hundred KB
// per meter) and works on the live path too, including rtl_tcp, where
// `gophertrunk capture` has no input at all. Each snippet is an IQ WAV (it
// carries its own rate, so `wmbus -in snippet.wav` replays it) plus a
// siglab sidecar naming the telegram it holds.

const (
	// wmbusSnippetHistorySec is how much IQ the saver keeps: longer than the
	// longest frame (255 bytes + CRCs, 3-of-6 coded: ~40 ms) plus margins.
	wmbusSnippetHistorySec = 0.5
	// wmbusSnippetPreSec / PostSec pad the snippet around the frame. The
	// pre-roll also covers the preamble, which varies by meter.
	wmbusSnippetPreSec  = 0.010
	wmbusSnippetPostSec = 0.005
	// wmbusChipsPerByte bounds the on-air length of a byte: 3-of-6 coding
	// (T1) spends 12 chips on it, NRZ (C1) 8.
	wmbusChipsPerByte = 12
	// wmbusSyncChips covers preamble + sync word ahead of the L-field.
	wmbusSyncChips = 100
)

type wmbusSnippet struct {
	start, end int64 // input-sample indices, end exclusive
	base       string
	meta       *siglab.Metadata
}

// wmbusSnippetSaver keeps a sliding window of recent IQ and writes the span
// around each telegram once the samples after it have arrived.
//
// The window is a fixed ring. It used to be a slice trimmed by copying the
// whole 0.5 s window down on every chunk: ~6 MB of memmove per 8192-sample
// RTL chunk at 1.6 MS/s, 37% of real time on a desktop core and far past
// real time on the reporter's 32-bit phone, so a live `-save-frames` run fell
// behind, the SDR dropped chunks and no telegram decoded — the empty meters/
// folder of #1256. A ring costs one copy of the chunk itself.
type wmbusSnippetSaver struct {
	dir      string
	rateHz   float64
	centerHz uint32
	offsetHz float64

	ring    []complex64 // len == capacity; sample i lives at ring[i%len]
	histEnd int64       // absolute index one past the newest sample held

	pending []wmbusSnippet
	saved   int
	// onSaved, when set, is told each snippet's path as it is written, so a
	// live run shows telegrams being kept instead of only a count at exit.
	onSaved func(path string)
	used    map[string]int
}

func newWMBusSnippetSaver(dir string, rateHz float64, centerHz uint32, offsetHz float64) (*wmbusSnippetSaver, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &wmbusSnippetSaver{
		dir: dir, rateHz: rateHz, centerHz: centerHz, offsetHz: offsetHz,
		ring: make([]complex64, max(1, int(rateHz*wmbusSnippetHistorySec))),
		used: map[string]int{},
	}, nil
}

// feed appends a chunk to the history. Call it before the receiver
// processes the chunk, so a frame that ends inside it is already held.
func (s *wmbusSnippetSaver) feed(iq []complex64) {
	n := len(s.ring)
	if len(iq) > n {
		// Only the newest window's worth can be held.
		s.histEnd += int64(len(iq) - n)
		iq = iq[len(iq)-n:]
	}
	for len(iq) > 0 {
		at := int(s.histEnd % int64(n))
		k := copy(s.ring[at:], iq)
		iq = iq[k:]
		s.histEnd += int64(k)
	}
}

// histStart is the absolute index of the oldest sample still held.
func (s *wmbusSnippetSaver) histStart() int64 {
	return max(0, s.histEnd-int64(len(s.ring)))
}

// span returns the held samples [start, end) in order, which may wrap the
// ring; the caller has clamped both ends to what is held.
func (s *wmbusSnippetSaver) span(start, end int64) []complex64 {
	n := int64(len(s.ring))
	a, b := int(start%n), int(end%n)
	if a < b || end-start == 0 {
		return s.ring[a : a+int(end-start)]
	}
	out := make([]complex64, 0, end-start)
	return append(append(out, s.ring[a:]...), s.ring[:b]...)
}

// onFrame queues the snippet for a frame that ended at info.Sample. label
// is the frame's timestamp as printed (wall clock live, capture time for a
// file).
func (s *wmbusSnippetSaver) onFrame(label string, f wmbus.Frame, tg wmbus.Telegram, info wmbus.FrameInfo) {
	frameSec := float64(len(f.Raw)*wmbusChipsPerByte+wmbusSyncChips) / wmbus.ChipRateHz
	start := info.Sample - int64((frameSec+wmbusSnippetPreSec)*s.rateHz)
	end := info.Sample + int64(wmbusSnippetPostSec*s.rateHz)
	mfr, id := tg.Manufacturer, tg.ID
	if mfr == "" {
		mfr = "unknown"
	}
	if id == "" {
		id = "unknown"
	}
	base := wmbusFileToken(fmt.Sprintf("%s_%s-%s_%s_%s", label, tg.Mode, tg.Format, mfr, id))
	if n := s.used[base]; n > 0 {
		s.used[base] = n + 1
		base = fmt.Sprintf("%s_%d", base, n+1)
	} else {
		s.used[base] = 1
	}
	meta := &siglab.Metadata{
		Protocol:     "wmbus",
		Source:       "gophertrunk wmbus -save-frames (IQ around one telegram)",
		SampleRateHz: s.rateHz,
		CenterFreqHz: s.centerHz,
		Format:       siglab.FormatWAV.String(),
		System: map[string]string{
			"wmbus_offset_hz":   strconv.FormatFloat(s.offsetHz, 'f', -1, 64),
			"mode":              fmt.Sprintf("%s-%s", tg.Mode, tg.Format),
			"manufacturer":      tg.Manufacturer,
			"id":                tg.ID,
			"device":            tg.DeviceTypeName(),
			"crc_ok":            strconv.FormatBool(f.CRCOK),
			"level_dbfs":        strconv.FormatFloat(info.LevelDBFS, 'f', 1, 64),
			"carrier_offset_hz": strconv.FormatFloat(math.Round(info.CarrierOffsetHz), 'f', 0, 64),
			"frame_hex":         hex.EncodeToString(f.Data),
		},
	}
	s.pending = append(s.pending, wmbusSnippet{start: start, end: end, base: base, meta: meta})
}

// flush writes every queued snippet whose trailing samples have arrived;
// final writes the rest with whatever has arrived.
func (s *wmbusSnippetSaver) flush(final bool) error {
	keep := s.pending[:0]
	var firstErr error
	for _, sn := range s.pending {
		if !final && sn.end > s.histEnd {
			keep = append(keep, sn)
			continue
		}
		if err := s.write(sn); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.pending = keep
	return firstErr
}

func (s *wmbusSnippetSaver) write(sn wmbusSnippet) error {
	start, end := max(sn.start, s.histStart()), min(sn.end, s.histEnd)
	if end <= start {
		return fmt.Errorf("wmbus: snippet %s fell out of the IQ history", sn.base)
	}
	path := filepath.Join(s.dir, sn.base+".wav")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	c, err := siglab.NewIQContainer(f, siglab.FormatWAV, int(s.rateHz+0.5))
	if err == nil {
		err = c.Write(s.span(start, end))
	}
	if err == nil {
		err = c.Finalize()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("wmbus: write %s: %w", path, err)
	}
	if err := siglab.WriteMetadata(strings.TrimSuffix(path, ".wav")+".metadata.json", sn.meta); err != nil {
		return fmt.Errorf("wmbus: write sidecar for %s: %w", path, err)
	}
	s.saved++
	if s.onSaved != nil {
		s.onSaved(path)
	}
	return nil
}

// wmbusFileToken keeps a snippet name filesystem-safe on every platform.
func wmbusFileToken(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}, s)
}
