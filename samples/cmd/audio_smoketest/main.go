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
)

func main() {
	var (
		path       = flag.String("file", "", "audio file path (MP3/WAV)")
		protocol   = flag.String("protocol", "auto", "protocol: mpt1327 / nxdn / auto (by folder)")
		sampleRate = flag.Float64("rate", 0, "PCM resample rate in Hz (default per-protocol)")
		clockGain  = flag.Float64("gain", 0.05, "Mueller-Müller loop gain")
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
		default:
			fmt.Fprintln(os.Stderr, "cannot auto-detect protocol from path; pass -protocol")
			os.Exit(2)
		}
	}

	rate := *sampleRate
	if rate == 0 {
		switch proto {
		case "mpt1327":
			rate = 8000 // FFSK at 1200 baud — 8k oversamples well past Nyquist
		case "nxdn", "ysf":
			rate = 48000 // 4-FSK at 4800 baud — sps = 10 for the matched filter
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
