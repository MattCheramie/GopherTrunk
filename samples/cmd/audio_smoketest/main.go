// Audio-to-bits smoke-test harness for the MPT 1327 control channel.
//
// MPT 1327 carries 1200-baud CCIR FFSK (mark = 1200 Hz, space =
// 1800 Hz) on top of NBFM. The sigidwiki samples in
// samples/mpt1327/ are MP3 recordings of the FM-demodulated audio
// — already at the FFSK helper's input. This harness:
//
//   1. shells out to ffmpeg to convert MP3 → 8 kHz mono float32 PCM,
//   2. feeds the PCM through the same FFSK + Mueller-Müller chain
//      the real receiver uses,
//   3. routes the resulting bit stream into mpt1327.ControlChannel,
//   4. prints every cc.locked and grant event the state machine
//      emits.
//
// Build / run:
//
//	go run ./samples/cmd/audio_smoketest -file samples/mpt1327/MPT1327_Sound.mp3
//
// The harness lives outside the main package tree to keep the
// production build clean — it's invoked manually when a new
// capture lands.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

func main() {
	var (
		path       = flag.String("file", "", "audio file path (MP3/WAV)")
		protocol   = flag.String("protocol", "auto", "protocol: mpt1327 / nxdn / tetra / ysf / auto (by folder)")
		sampleRate = flag.Float64("rate", 0, "PCM resample rate in Hz (default per-protocol)")
		clockGain  = flag.Float64("gain", 0.05, "Mueller-Müller loop gain")
		swapIQ     = flag.Bool("swap-iq", false, "swap I and Q channels (some SDR recordings invert)")
		conjIQ     = flag.Bool("conj-iq", false, "conjugate IQ (negate Q) — alternative spectrum-inversion fix")
	)
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "usage: audio_smoketest -file <path> [-protocol mpt1327|nxdn|auto]")
		os.Exit(2)
	}

	proto := *protocol
	if proto == "auto" {
		switch {
		case strings.Contains(*path, "/mpt1327/"):
			proto = "mpt1327"
		case strings.Contains(*path, "/nxdn/"):
			proto = "nxdn"
		case strings.Contains(*path, "/ysf/"):
			proto = "ysf"
		case strings.Contains(*path, "/tetra/"):
			proto = "tetra"
		default:
			fmt.Fprintln(os.Stderr, "cannot auto-detect protocol from path; pass -protocol")
			os.Exit(2)
		}
	}

	// Files named "*IQ*" are stereo 16-bit WAVs with I in the left
	// channel and Q in the right — the SDR# / SDRSharp recording
	// convention. Route them through the IQ-input code path that
	// uses the real receiver pipelines instead of the FM-bypass
	// audio harness.
	isIQ := strings.Contains(strings.ToLower(filepath.Base(*path)), "iq")

	if isIQ {
		iq, rate, err := readIQWav(*path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read IQ %s: %v\n", *path, err)
			os.Exit(1)
		}
		if *swapIQ {
			for i, c := range iq {
				iq[i] = complex(imag(c), real(c))
			}
		}
		if *conjIQ {
			for i, c := range iq {
				iq[i] = complex(real(c), -imag(c))
			}
		}
		fmt.Printf("file=%s  protocol=%s  iq=%d  rate=%.0f Hz  dur=%.2f s  swap=%v conj=%v\n",
			filepath.Base(*path), proto, len(iq), rate, float64(len(iq))/rate, *swapIQ, *conjIQ)
		switch proto {
		case "nxdn":
			runNXDNFromIQ(iq, rate, *clockGain)
		case "tetra":
			runTETRAFromIQ(iq, rate)
		default:
			fmt.Fprintf(os.Stderr, "IQ input unsupported for protocol %q\n", proto)
			os.Exit(2)
		}
		return
	}

	rate := *sampleRate
	if rate == 0 {
		switch proto {
		case "mpt1327":
			rate = 8000
		case "nxdn", "ysf":
			rate = 48000
		default:
			rate = 8000
		}
	}

	pcm, err := decodeToPCM(*path, rate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode %s: %v\n", *path, err)
		os.Exit(1)
	}
	fmt.Printf("file=%s  protocol=%s  samples=%d  rate=%.0f Hz  dur=%.2f s\n",
		filepath.Base(*path), proto, len(pcm), rate, float64(len(pcm))/rate)

	switch proto {
	case "mpt1327":
		runMPT1327(pcm, rate, *clockGain)
	case "nxdn":
		runNXDN(pcm, rate, *clockGain)
	case "ysf":
		runYSF(pcm, rate, *clockGain)
	default:
		fmt.Fprintf(os.Stderr, "unsupported protocol %q\n", proto)
		os.Exit(2)
	}
}

// runYSF feeds FM-demodulated audio through a C4FM matched filter +
// Mueller-Müller + 4-level slicer and reports raw symbol-bin stats.
// YSF lives in the ysf package which exposes a different state-
// machine surface — the smoketest just confirms the post-FM-demod
// audio produces a usable 4-level constellation. A real ysf
// ControlChannel-driven test gets written once we know the audio
// path is viable.
func runYSF(pcm []float32, rate, clockGain float64) {
	sps := rate / 4800
	if sps < 2 {
		fmt.Fprintf(os.Stderr, "YSF: sps=%.2f too low; need rate >= 9600 Hz\n", sps)
		os.Exit(1)
	}
	const (
		span  = 8
		alpha = 0.2
	)
	mf := demod.NewC4FM(int(sps+0.5), span, alpha, 1.0)
	clock := sync.NewMuellerMuller(sps, clockGain)

	var (
		matched []float32
		symbols []float32
		sliced  []int8
		bins    [4]int
		total   int
	)
	chunk := 4096
	for off := 0; off < len(pcm); off += chunk {
		end := off + chunk
		if end > len(pcm) {
			end = len(pcm)
		}
		matched = mf.MatchedFilter(matched[:0], pcm[off:end])
		symbols = clock.Process(symbols[:0], matched)
		if len(symbols) == 0 {
			continue
		}
		sliced = mf.SliceMany(sliced[:0], symbols)
		for _, s := range sliced {
			switch s {
			case 3:
				bins[0]++
			case 1:
				bins[1]++
			case -1:
				bins[2]++
			case -3:
				bins[3]++
			}
			total++
		}
	}
	fmt.Printf("  total symbols: %d\n", total)
	fmt.Printf("  symbol bins (+3 / +1 / -1 / -3): %d / %d / %d / %d\n",
		bins[0], bins[1], bins[2], bins[3])
	fmt.Printf("  bin balance: %.1f%% / %.1f%% / %.1f%% / %.1f%%\n",
		pct(bins[0], total), pct(bins[1], total), pct(bins[2], total), pct(bins[3], total))
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(total)
}

// runMPT1327 feeds FM-demodulated audio through the FFSK +
// Mueller-Müller + state-machine chain. Mirrors the
// internal/radio/mpt1327/receiver pipeline starting at step 2 (the
// receiver does step 1 — IQ → FM demod — internally).
func runMPT1327(pcm []float32, rate, clockGain float64) {
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cc := mpt1327.New(mpt1327.Options{
		Bus:         bus,
		Log:         log,
		SystemName:  "smoketest",
		FrequencyHz: 0,
	})
	// Turn on BCH(64,48,2) — the alignment search keeps only
	// windows that pass the BCH parity, dropping noise that
	// happens to parse as a valid opcode.
	cc.SetBCHMode(mpt1327.BCHOn)

	ffsk := demod.NewFFSK(rate, 1200, 1800)
	sps := rate / 1200
	clock := sync.NewMuellerMuller(sps, clockGain)

	var (
		tone    []float32
		symbols []float32
		bits    []byte
		baseIdx int
	)
	chunk := 4096
	for off := 0; off < len(pcm); off += chunk {
		end := off + chunk
		if end > len(pcm) {
			end = len(pcm)
		}
		tone = ffsk.Discriminate(tone[:0], pcm[off:end])
		symbols = clock.Process(symbols[:0], tone)
		if len(symbols) == 0 {
			continue
		}
		if cap(bits) < len(symbols) {
			bits = make([]byte, len(symbols))
		} else {
			bits = bits[:len(symbols)]
		}
		for i, s := range symbols {
			bits[i] = byte(ffsk.Slice(s))
		}
		cc.Process(bits, baseIdx)
		baseIdx += len(bits)
	}

	summarise(sub.C)
}

// runNXDN feeds FM-demodulated audio through the C4FM matched-
// filter + Mueller-Müller + 4-level slicer + state machine. Mirrors
// the internal/radio/nxdn/receiver pipeline starting AFTER FM demod
// (the audio captures are already at the FM discriminator output).
//
// NXDN 4-FSK operates at 4800 sym/s with 9600-baud bit rate. The
// MM clock recovery needs sps = rate / 4800; 48 kHz audio gives
// sps = 10 which is what the production receiver uses.
func runNXDN(pcm []float32, rate, clockGain float64) {
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cc := nxdn.NewControlChannel(bus, log, 0, nxdn.Rate9600)
	// ViterbiSpec exercises the full §4.5.1.1 outbound CAC chain.
	cc.SetViterbiMode(nxdn.ViterbiSpec)

	sps := rate / 4800
	if sps < 2 {
		fmt.Fprintf(os.Stderr, "NXDN: sps=%.2f too low; need rate >= 9600 Hz\n", sps)
		os.Exit(1)
	}
	const (
		span  = 8 // pulse span symbols — matches receiver default
		alpha = 0.2
	)
	mf := demod.NewC4FM(int(sps+0.5), span, alpha, 1.0)
	clock := sync.NewMuellerMuller(sps, clockGain)

	var (
		matched []float32
		symbols []float32
		sliced  []int8
		dibits  []uint8
		baseIdx int
	)
	chunk := 4096
	for off := 0; off < len(pcm); off += chunk {
		end := off + chunk
		if end > len(pcm) {
			end = len(pcm)
		}
		matched = mf.MatchedFilter(matched[:0], pcm[off:end])
		symbols = clock.Process(symbols[:0], matched)
		if len(symbols) == 0 {
			continue
		}
		sliced = mf.SliceMany(sliced[:0], symbols)
		if cap(dibits) < len(sliced) {
			dibits = make([]uint8, len(sliced))
		} else {
			dibits = dibits[:len(sliced)]
		}
		for i, sym := range sliced {
			dibits[i] = nxdnrx.SymbolToDibit(sym)
		}
		cc.Process(dibits, baseIdx)
		baseIdx += len(dibits)
	}

	summarise(sub.C)
}

// summarise drains every event currently in the bus channel and
// prints a compact summary: total locks / grants, plus a sample of
// each.
func summarise(ch <-chan events.Event) {
	var locks, grants int
	var firstLock, firstGrant *events.Event
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case events.KindCCLocked:
				locks++
				if firstLock == nil {
					ev := ev
					firstLock = &ev
				}
			case events.KindGrant:
				grants++
				if firstGrant == nil {
					ev := ev
					firstGrant = &ev
				}
			}
		default:
			fmt.Printf("  cc.locked events: %d\n", locks)
			fmt.Printf("  grant events:    %d\n", grants)
			if firstLock != nil {
				fmt.Printf("  first lock:  %#v\n", firstLock.Payload)
			}
			if firstGrant != nil {
				fmt.Printf("  first grant: %#v\n", firstGrant.Payload)
			}
			return
		}
	}
}

// readIQWav parses a stereo 16-bit PCM WAV file (the SDR# IQ
// recording format) into complex64 samples where left channel = I,
// right channel = Q. Returns the samples and the sample rate from
// the WAV header.
func readIQWav(path string) ([]complex64, float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	// Validate RIFF/WAVE header.
	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, 0, fmt.Errorf("read RIFF header: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	var (
		sampleRate    uint32
		channels      uint16
		bitsPerSample uint16
		dataOffset    int64
		dataSize      uint32
	)
	for {
		var hdr [8]byte
		_, err := io.ReadFull(f, hdr[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read chunk header: %w", err)
		}
		id := string(hdr[0:4])
		size := binary.LittleEndian.Uint32(hdr[4:8])
		switch id {
		case "fmt ":
			fmtBuf := make([]byte, size)
			if _, err := io.ReadFull(f, fmtBuf); err != nil {
				return nil, 0, fmt.Errorf("read fmt chunk: %w", err)
			}
			channels = binary.LittleEndian.Uint16(fmtBuf[2:4])
			sampleRate = binary.LittleEndian.Uint32(fmtBuf[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(fmtBuf[14:16])
		case "data":
			pos, _ := f.Seek(0, io.SeekCurrent)
			dataOffset = pos
			dataSize = size
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("skip data chunk: %w", err)
			}
		default:
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("skip chunk %q: %w", id, err)
			}
		}
		// Skip pad byte for odd-sized chunks.
		if size%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return nil, 0, fmt.Errorf("skip pad byte: %w", err)
			}
		}
	}
	if channels != 2 || bitsPerSample != 16 {
		return nil, 0, fmt.Errorf("expected stereo 16-bit IQ WAV; got %d channels, %d bits/sample",
			channels, bitsPerSample)
	}
	if _, err := f.Seek(dataOffset, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("seek to data: %w", err)
	}
	frameBytes := 4 // 2 channels × 2 bytes
	count := int(dataSize) / frameBytes
	iq := make([]complex64, count)
	buf := make([]byte, dataSize)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, 0, fmt.Errorf("read data: %w", err)
	}
	for i := 0; i < count; i++ {
		off := i * 4
		l := int16(binary.LittleEndian.Uint16(buf[off : off+2]))
		r := int16(binary.LittleEndian.Uint16(buf[off+2 : off+4]))
		iq[i] = complex(float32(l)/32768.0, float32(r)/32768.0)
	}
	return iq, float64(sampleRate), nil
}

// nxdnIQStats prints diagnostics about the captured IQ: DC bias on
// the FM discriminator output (indicates a frequency offset), and
// the symbol-bin distribution after the matched filter (indicates
// whether the 4-level constellation is recoverable).
func nxdnIQStats(iq []complex64, rate float64) {
	fm := demod.NewFM()
	disc := fm.Process(nil, iq)
	if len(disc) == 0 {
		return
	}
	var sum, sumSq float64
	for _, v := range disc {
		sum += float64(v)
		sumSq += float64(v) * float64(v)
	}
	mean := sum / float64(len(disc))
	variance := sumSq/float64(len(disc)) - mean*mean
	fmt.Printf("  FM discriminator: mean=%+.5f stddev=%.5f samples=%d\n",
		mean, sqrt(variance), len(disc))

	sps := rate / 4800
	const (
		span  = 8
		alpha = 0.2
	)
	mf := demod.NewC4FM(int(sps+0.5), span, alpha, 1.0)
	clock := sync.NewMuellerMuller(sps, 0.05)
	matched := mf.MatchedFilter(nil, disc)
	symbols := clock.Process(nil, matched)
	if len(symbols) == 0 {
		fmt.Println("  no symbols recovered")
		return
	}
	// Adaptive symbol-distribution: bin by quartile of |value| so
	// scale-agnostic stats land sensibly even when the captured
	// signal level is far from the slicer's ±1, ±3 reference.
	var absMax float32
	for _, s := range symbols {
		if s > absMax {
			absMax = s
		}
		if -s > absMax {
			absMax = -s
		}
	}
	fmt.Printf("  symbols: count=%d  abs_max=%.4f\n", len(symbols), absMax)
	if absMax == 0 {
		return
	}
	bins := [4]int{}
	for _, s := range symbols {
		// Map to ±3 / ±1 based on whether |s| is above or below
		// half of absMax.
		threshold := absMax / 2
		var bin int
		switch {
		case s >= threshold:
			bin = 0 // +3
		case s >= 0:
			bin = 1 // +1
		case s >= -threshold:
			bin = 2 // -1
		default:
			bin = 3 // -3
		}
		bins[bin]++
	}
	fmt.Printf("  symbol bins (+3 / +1 / -1 / -3): %d / %d / %d / %d\n",
		bins[0], bins[1], bins[2], bins[3])
	fmt.Printf("  bin balance: %.1f%% / %.1f%% / %.1f%% / %.1f%%\n",
		pct(bins[0], len(symbols)), pct(bins[1], len(symbols)),
		pct(bins[2], len(symbols)), pct(bins[3], len(symbols)))
}

// sqrt is a minimal float64 square root used by nxdnIQStats so the
// file avoids importing math just for one call.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton-Raphson, 20 iterations gives plenty of precision for
	// stats output.
	g := x
	for i := 0; i < 20; i++ {
		g = 0.5 * (g + x/g)
	}
	return g
}

// runNXDNFromIQ feeds complex IQ samples through the real NXDN
// receiver pipeline (IQ → FM demod → C4FM matched filter → MM
// clock → 4-level slicer → dibits → ControlChannel.Process).
func runNXDNFromIQ(iq []complex64, rate, clockGain float64) {
	nxdnIQStats(iq, rate)
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cc := nxdn.NewControlChannel(bus, log, 0, nxdn.Rate9600)
	cc.SetViterbiMode(nxdn.ViterbiSpec)

	var (
		totalDibits int
		dibitBins   [4]int
	)
	rx := nxdnrx.New(nxdnrx.Options{
		SampleRateHz: rate,
		// Match the production connector — NXDN peak deviation per
		// the Common Air Interface calibrates the 4-level slicer
		// against the actual FM-discriminator output level. Without
		// this the slicer assumes a normalised ±1 input and every
		// real symbol clusters into the inner ±1 bins.
		DeviationHz: 1800.0,
		DibitSink: func(dibits []uint8, baseIdx int) {
			for _, d := range dibits {
				if d < 4 {
					dibitBins[d]++
				}
				totalDibits++
			}
			cc.Process(dibits, baseIdx)
		},
		ClockGain: clockGain,
	})
	chunk := 4096
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[off:end])
	}
	fmt.Printf("  dibits to state machine: %d  bins (0/1/2/3): %d / %d / %d / %d (%.0f%% / %.0f%% / %.0f%% / %.0f%%)\n",
		totalDibits, dibitBins[0], dibitBins[1], dibitBins[2], dibitBins[3],
		pct(dibitBins[0], totalDibits), pct(dibitBins[1], totalDibits),
		pct(dibitBins[2], totalDibits), pct(dibitBins[3], totalDibits))
	summarise(sub.C)
}

// runTETRAFromIQ feeds complex IQ samples through the real TETRA
// receiver pipeline (IQ → π/4-DQPSK matched filter → Gardner clock
// → dibit slicer → ControlChannel.Process).
func runTETRAFromIQ(iq []complex64, rate float64) {
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cc := tetra.New(tetra.Options{
		Bus:         bus,
		Log:         log,
		SystemName:  "smoketest",
		FrequencyHz: 0,
	})

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: rate,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
		ClockMode:   tetrarx.ClockGardner,
		GardnerGain: 0.005, // matches the production connector tuning
	})
	chunk := 4096
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[off:end])
	}
	summarise(sub.C)
}

// decodeToPCM shells out to ffmpeg to convert any input audio file
// to mono float32 PCM at the requested sample rate, then returns
// the samples normalised to [-1, 1].
func decodeToPCM(path string, rate float64) ([]float32, error) {
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", path,
		"-ac", "1", // mono
		"-ar", fmt.Sprintf("%.0f", rate),
		"-f", "s16le", // 16-bit signed little-endian
		"-",
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg produced no output for %s", path)
	}
	if buf.Len()%2 != 0 {
		return nil, fmt.Errorf("ffmpeg output size %d not aligned to int16", buf.Len())
	}
	count := buf.Len() / 2
	pcm := make([]float32, count)
	br := bytes.NewReader(buf.Bytes())
	for i := 0; i < count; i++ {
		var v int16
		if err := binary.Read(br, binary.LittleEndian, &v); err != nil {
			return nil, fmt.Errorf("read sample %d: %w", i, err)
		}
		pcm[i] = float32(v) / 32768.0
	}
	return pcm, nil
}
