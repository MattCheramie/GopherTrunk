package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runWMBus is `gophertrunk wmbus`: a Wireless M-Bus (EN 13757-4, T1 and
// C1 modes) telegram decoder for utility meters at 868.95 MHz, reading a
// capture file or a live local / rtl_tcp SDR (issue #1256).
func runWMBus(args []string) {
	fs := flag.NewFlagSet("wmbus", flag.ExitOnError)
	in := fs.String("in", "", "IQ capture to decode instead of a live SDR")
	format := fs.String("format", "u8", "capture sample format for headerless files: u8 (rtl_sdr .cu8) | cs16 | f32; wav and flac are detected from the file")
	rate := fs.Uint("rate", 1_600_000, "sample rate in S/s (a capture's rate, read from its .metadata.json when present; for a live SDR the rate to run at)")
	sampleRate := fs.Uint("sample-rate", 0, "same as -rate (the name `gophertrunk capture` uses)")
	offset := fs.Float64("offset", 0, "where 868.95 MHz sits relative to the capture/tuner centre, in Hz (e.g. 325000 for a capture centred on 868.625 MHz)")
	freq := fs.Uint("freq", wmbus.ChannelHz, "live SDR centre frequency in Hz")
	gain := fs.String("gain", "auto", `live tuner gain: "auto", or tenths of a dB ("400" = 40 dB)`)
	ppm := fs.Int("ppm", 0, "live frequency correction in ppm")
	serial := fs.String("serial", "", "local SDR serial (required when more than one is attached)")
	rtlTCP := fs.String("rtltcp", "", "host:port of an rtl_tcp server to listen through instead of a local SDR")
	seconds := fs.Float64("seconds", 0, "stop a live run after this many seconds (0 = until Ctrl-C)")
	asJSON := fs.Bool("json", false, "print one JSON object per telegram")
	all := fs.Bool("all", false, "also print frames whose CRC failed")
	saveFrames := fs.String("save-frames", "", "write the IQ around each telegram to this directory (an IQ .wav plus a .metadata.json per telegram; a few hundred KB each, small enough to share), live or from -in; CRC-valid telegrams only unless -all is set")
	status := fs.Duration("status", 30*time.Second, "live: print a progress line (frames, CRC-valid / CRC-failed, access codes, dropped chunks) this often; 0 = only at the end")
	logPath := fs.String("log", "", "also append every telegram, progress line, warning and the reason a live run stopped to this file, written as it happens (survives Ctrl-C, a closed terminal or a killed `| tee`)")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk wmbus — Wireless M-Bus (EN 13757-4) meter telegram decoder, modes T1 and C1 at 868.95 MHz.

USAGE:
  gophertrunk wmbus -in capture [-format u8|cs16|f32] [-rate S/s] [-offset Hz]
  gophertrunk wmbus [-serial S | -rtltcp host:port] [-gain G] [-ppm N] [-seconds N]

Prints the sender of every telegram (manufacturer, ID, device type), and the
readings of unencrypted meters. Most OMS meters encrypt their readings (AES-128,
security mode 5 or 7, key held by the metering company): those show as
"encrypted" with their header only.

EXAMPLES:
  # Listen live on a local RTL-SDR
  gophertrunk wmbus

  # Through an rtl_tcp server (e.g. a phone running rtl_tcp)
  gophertrunk wmbus -rtltcp 127.0.0.1:1234

  # Decode a capture recorded with (flac carries at most ~1.05 MS/s):
  #   gophertrunk capture -freq 868950000 -sample-rate 1024000 -seconds 60 -format flac -out m.flac
  gophertrunk wmbus -in m.flac

  # Decode a headerless capture; its rate, format and centre come from the
  # .metadata.json that "gophertrunk capture" writes beside it
  gophertrunk wmbus -in m.cu8

  # Decode an rtl_sdr capture (rtl_sdr -f 868.95M -s 1600000 m.cu8), which
  # has no sidecar, so give the rate
  gophertrunk wmbus -in m.cu8 -rate 1600000

  # Keep the IQ around each telegram (works over rtl_tcp too), then replay one
  gophertrunk wmbus -rtltcp 127.0.0.1:1234 -save-frames meters/
  gophertrunk wmbus -in meters/<file>.wav

  # A long unattended listen: a fixed low gain (tenths of a dB; 0 = minimum),
  # failed-CRC telegrams printed and saved too, everything appended to a log
  # that survives Ctrl-C, and a progress line every minute
  gophertrunk wmbus -gain 0 -all -save-frames meters/ -log wm.log -status 1m

A live run prints a progress line every -status (default 30s): frames heard,
how many passed / failed their CRC, access codes (sync words; many with no
frames means noise or overload), and IQ chunks the SDR dropped (any at all
means the decoder is not keeping up). Meters send every 2 minutes to a few
hours, so expect long quiet stretches.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("wmbus")
	if fs.NArg() > 0 {
		fs.Usage()
		rep.Fatalf(2, "unexpected arguments %q", fs.Args())
	}
	explicit := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	if explicit["sample-rate"] {
		if explicit["rate"] && *rate != *sampleRate {
			rep.Fatalf(2, "-rate %d and -sample-rate %d disagree; give one", *rate, *sampleRate)
		}
		*rate = *sampleRate
		explicit["rate"] = true
	}

	var logFile *os.File
	if *logPath != "" {
		var err error
		if logFile, err = os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err != nil {
			rep.Fatal(1, err)
		}
		defer logFile.Close()
	}
	// stderr is the screen, plus the -log file once one is open: every
	// diagnostic wmbus prints (the reason a live run stopped, SDR / USB
	// warnings, the stop hints, a fatal error) reaches the log too. They
	// used to go to the screen only, so a run that stopped on its own in
	// the field left a log that ended mid-stream with no reason (#1256).
	var stderr io.Writer = os.Stderr
	if logFile != nil {
		echo := newWMBusLogEcho(logFile, *asJSON)
		stderr = io.MultiWriter(os.Stderr, echo)
		rep.Out = stderr
	}
	// statusf prints a progress / summary line to stderr and the -log file.
	// With -json the log gets it as a {"status": …} object, so every line
	// of the log parses as JSON for a script reading it.
	statusf := func(line string) { fmt.Fprintln(stderr, line) }
	var stdout io.Writer = os.Stdout
	if logFile != nil {
		// The log first: io.MultiWriter stops at the first failed write,
		// and stdout is the one that fails (a `| tee` killed by Ctrl-C).
		stdout = io.MultiWriter(logFile, os.Stdout)
	}
	out := bufio.NewWriter(stdout)
	defer out.Flush()
	// stamp labels a frame: wall-clock time live, capture time for a file.
	var stamp func(sample int64) string
	// saver, when -save-frames is set, keeps the IQ around each telegram.
	var saver *wmbusSnippetSaver
	newSaver := func(rateHz float64, centerHz uint32) {
		if *saveFrames == "" {
			return
		}
		var err error
		if saver, err = newWMBusSnippetSaver(*saveFrames, rateHz, centerHz, *offset); err != nil {
			rep.Fatal(1, err)
		}
		saver.onSaved = func(path string) { statusf("wmbus: saved " + path) }
	}
	reportSaved := func() {
		if saver == nil {
			return
		}
		if err := saver.flush(true); err != nil {
			fmt.Fprintln(stderr, err)
		}
		statusf(fmt.Sprintf("wmbus: saved %d telegram snippet(s) to %s", saver.saved, *saveFrames))
	}
	newRx := func(rateHz float64) *wmbus.Receiver {
		rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{
			SampleRateHz: rateHz,
			OffsetHz:     *offset,
			OnFrame: func(f wmbus.Frame, info wmbus.FrameInfo) {
				if !f.CRCOK && !*all {
					return
				}
				tg, err := wmbus.ParseTelegram(f)
				ts := stamp(info.Sample)
				writeWMBusTelegram(out, ts, tg, err, f, info, *asJSON)
				_ = out.Flush()
				if saver != nil {
					saver.onFrame(ts, f, tg, info)
				}
			},
		})
		if err != nil {
			rep.Fatal(2, err)
		}
		return rx
	}

	if *in != "" {
		notes, err := applyWMBusSidecar(*in, explicit, rate, format, offset)
		if err != nil {
			rep.Fatal(1, err)
		}
		for _, n := range notes {
			fmt.Fprintln(stderr, n)
		}
		rateHz := float64(*rate)
		stamp = func(s int64) string { return fmt.Sprintf("t=%8.3fs", float64(s)/rateHz) }
		rx, err := decodeWMBusFile(*in, *format, &rateHz, func(r float64) *wmbus.Receiver {
			newSaver(r, 0)
			return newRx(r)
		}, func() *wmbusSnippetSaver { return saver })
		if err != nil {
			rep.Fatal(1, err)
		}
		reportSaved()
		st := rx.Stats()
		statusf(fmt.Sprintf("wmbus: %.1f s of IQ at %.0f S/s, %d frames (%d CRC-valid), %d access codes",
			float64(st.Samples)/rateHz, rateHz, st.Framer.Frames, st.Framer.CRCOK, st.Framer.Syncs) + wmbusCarrierNote(st))
		return
	}

	gainTenthDB, ok := parseGain(*gain)
	if !ok {
		rep.Fatalf(2, `-gain: want "auto" or tenths of a dB, got %q`, *gain)
	}
	log := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	// The SDR drivers report USB recovery and stream trouble through the
	// default logger; route it through the same writer so it reaches -log.
	slog.SetDefault(log)
	dev, name, err := openPowerDevice(*rtlTCP, *serial, log)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer closeSDRBounded(dev, sdrCloseTimeout, stderr, "wmbus")
	if err := dev.SetSampleRate(uint32(*rate)); err != nil {
		rep.Fatal(1, fmt.Errorf("set sample rate: %w", err))
	}
	if hw, note := captureEffectiveRate(dev, uint32(*rate)); note != "" {
		fmt.Fprintln(stderr, strings.Replace(note, "capture:", "wmbus:", 1))
		*rate = uint(hw)
	}
	if *ppm != 0 {
		if err := dev.SetPPM(*ppm); err != nil {
			rep.Fatal(1, fmt.Errorf("set ppm: %w", err))
		}
	}
	if err := dev.SetGain(gainTenthDB); err != nil {
		rep.Fatal(1, fmt.Errorf("set gain: %w", err))
	}
	if err := dev.SetCenterFreq(uint32(*freq)); err != nil {
		rep.Fatal(1, fmt.Errorf("tune %d Hz: %w", *freq, err))
	}
	ctx, stop := sdrStopContext("wmbus", stderr)
	defer stop()
	if *seconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*seconds*float64(time.Second)))
		defer cancel()
	}
	stream, err := dev.StreamIQ(ctx)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
	}
	stamp = func(int64) string { return time.Now().Format("2006-01-02 15:04:05.000") }
	newSaver(float64(*rate), uint32(*freq))
	rx := newRx(float64(*rate))
	drops := &captureDropCounter{serial: dev.Info().Serial}
	sdr.SetIQDropObserver(drops.observe)
	defer sdr.SetIQDropObserver(nil)
	fmt.Fprintf(stderr, "wmbus: listening on %s at %.3f MHz, %d S/s (Ctrl-C to stop)\n", name, float64(*freq)/1e6, *rate)
	if saver != nil && !*all {
		fmt.Fprintln(stderr, "wmbus: -save-frames keeps CRC-valid telegrams only; add -all to keep the CRC-failed ones too")
	}
	ended := listenWMBus(ctx, stream, rx, saver, wmbusListenOptions{
		RateHz: float64(*rate), Status: *status, Drops: drops.count,
		Statusf: statusf,
	})
	reportSaved()
	if ended {
		// Only a cancel or -seconds ends a live listen normally; a stream
		// that closes on its own is a lost source, which used to exit with
		// no message at all (#1256).
		rep.Fatal(1, iqStreamEnded(dev, errors.New("IQ stream ended")))
	}
}

// wmbusListenOptions configures listenWMBus.
type wmbusListenOptions struct {
	RateHz float64
	// Status is the stream time between progress lines; 0 prints only the
	// final one.
	Status time.Duration
	// Drops reports the IQ chunks the SDR has dropped so far (nil: none).
	Drops func() uint64
	// Statusf receives each progress line and the final summary.
	Statusf func(string)
}

// listenWMBus decodes a live IQ stream until ctx ends (false) or the
// stream closes on its own (true). Every Status of stream time it reports
// how much it has heard — frames, CRC-valid frames, access codes and SDR
// drops — so an operator waiting minutes for a meter's next telegram can
// see the decoder is alive and whether anything is arriving (#1256). Stream
// time, not wall time, paces it: live they agree, and a test can drive it.
func listenWMBus(ctx context.Context, stream <-chan []complex64, rx *wmbus.Receiver, saver *wmbusSnippetSaver, o wmbusListenOptions) (streamEnded bool) {
	every := int64(o.Status.Seconds() * o.RateHz)
	next := every
	var warnedDrops uint64
	line := func(final bool) {
		st := rx.Stats()
		var d uint64
		if o.Drops != nil {
			d = o.Drops()
		}
		o.Statusf(wmbusStatusLine(float64(st.Samples)/o.RateHz, st, d, final))
		if d > warnedDrops {
			warnedDrops = d
			o.Statusf(fmt.Sprintf("wmbus: WARNING — the SDR has dropped %d IQ chunk(s): the decoder is not keeping up, and a telegram that spans a gap cannot decode. Try -rate 1200000 (the cheapest rate to decode; 1024000 costs twice as much), close other programs, or check the USB link.", d))
		}
	}
	for {
		select {
		case <-ctx.Done():
			line(true)
			return false
		case iq, ok := <-stream:
			if !ok {
				line(true)
				return true
			}
			if saver != nil {
				saver.feed(iq)
			}
			rx.Process(iq)
			if saver != nil {
				if err := saver.flush(false); err != nil {
					o.Statusf(err.Error())
				}
			}
			if every > 0 && rx.Stats().Samples >= next {
				next += every
				line(false)
			}
		}
	}
}

// wmbusStatusLine is one live progress line (or, final, the summary).
func wmbusStatusLine(seconds float64, st wmbus.ReceiverStats, drops uint64, final bool) string {
	what := "listening"
	if final {
		what = "stopped after"
	}
	repaired := ""
	if st.Framer.Repaired > 0 {
		repaired = fmt.Sprintf(" of which %d repaired", st.Framer.Repaired)
	}
	s := fmt.Sprintf("wmbus: %s %s: %d frames (%d CRC-valid%s, %d CRC-failed), %d access codes, %d dropped chunks",
		what, time.Duration(seconds*float64(time.Second)).Round(time.Second).String(), st.Framer.Frames, st.Framer.CRCOK,
		repaired, st.Framer.Frames-st.Framer.CRCOK, st.Framer.Syncs, drops)
	return s + wmbusCarrierNote(st)
}

// wmbusCarrierOffsetNoteHz is the carrier offset from which the progress
// line suggests a -ppm: 20 ppm at 868.95 MHz.
const wmbusCarrierOffsetNoteHz = 17_400

// wmbusCarrierNote reports where the receiver found the meters' carrier,
// once a CRC-valid frame has located it. Every meter heard through one
// tuner sits at that tuner's frequency error, which the receiver now
// follows; past about 20 ppm it also says how to centre the tuner, as the
// S5 + RTL-SDR in issue #1256 heard its meters 50 kHz (58 ppm) low. The
// sign convention of -ppm is driver-specific (see
// formatCarrierOffsetWarning), so it quotes the magnitude.
func wmbusCarrierNote(st wmbus.ReceiverStats) string {
	if !st.CarrierLocked {
		return ""
	}
	off := st.CarrierOffsetHz
	ppm := offsetPPM(off, wmbus.ChannelHz)
	s := fmt.Sprintf(", carrier %+.1f kHz (%+.0f ppm)", off/1000, ppm)
	if math.Abs(off) >= wmbusCarrierOffsetNoteHz {
		s += fmt.Sprintf(" — corrected for, but -ppm ±%.0f would centre the tuner (keep the sign that shrinks it)", math.Abs(ppm))
	}
	return s
}

// wmbusLogStatusLine is how a progress line goes into the -log file: as
// is, or with -json as a {"status": …} object, so a script reading the log
// can parse every line and tell telegrams (which have "frame") from
// progress.
// wmbusLogEcho copies what wmbus prints to the screen into the -log file,
// one log line per printed line, wrapped by wmbusLogStatusLine so a -json
// log still parses line by line. It buffers a partial line until its
// newline (slog and fmt.Fprintf may write a line in pieces) and is safe for
// the driver goroutines that log through slog while the main loop prints.
type wmbusLogEcho struct {
	mu      sync.Mutex
	w       io.Writer
	asJSON  bool
	pending []byte
}

func newWMBusLogEcho(w io.Writer, asJSON bool) *wmbusLogEcho {
	return &wmbusLogEcho{w: w, asJSON: asJSON}
}

// Write implements io.Writer. It always reports len(p): a log write that
// fails must not stop io.MultiWriter from reaching the screen after it.
func (e *wmbusLogEcho) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pending = append(e.pending, p...)
	for {
		i := bytes.IndexByte(e.pending, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(e.pending[:i]), "\r")
		e.pending = e.pending[i+1:]
		if line == "" {
			continue
		}
		// One Write per line, so a line never interleaves with the
		// telegram lines the main loop writes to the same file.
		_, _ = io.WriteString(e.w, wmbusLogStatusLine(line, e.asJSON)+"\n")
	}
	return len(p), nil
}

func wmbusLogStatusLine(line string, asJSON bool) string {
	if !asJSON {
		return line
	}
	b, _ := json.Marshal(struct {
		Time   string `json:"time"`
		Status string `json:"status"`
	}{time.Now().Format("2006-01-02 15:04:05.000"), line})
	return string(b)
}

// applyWMBusSidecar fills the -in settings the user did not give from the
// capture's .metadata.json (`gophertrunk capture` writes one beside every
// file). A headerless .cu8 carries no rate of its own, and decoding a 2.8 MS/s
// capture at the 1.6 MS/s default finds access codes but never a frame — the
// #1256 field captures read as "52.5 s of IQ, 0 frames". Flags the user set
// win; a -rate that disagrees with the sidecar is reported, not overridden.
// Container files (wav/flac) still take their rate from the container.
func applyWMBusSidecar(in string, explicit map[string]bool, rate *uint, format *string, offset *float64) ([]string, error) {
	path := siglab.DiscoverMetadata(in)
	if path == "" {
		return nil, nil
	}
	m, err := siglab.ReadMetadata(path)
	if err != nil {
		return nil, err
	}
	var notes []string
	if r := uint(m.SampleRateHz + 0.5); r > 0 {
		switch {
		case !explicit["rate"]:
			*rate = r
			notes = append(notes, fmt.Sprintf("wmbus: %d S/s from %s", r, path))
		case *rate != r:
			notes = append(notes, fmt.Sprintf("wmbus: note — -rate %d differs from the %d S/s in %s; decoding at -rate", *rate, r, path))
		}
	}
	if f := strings.ToLower(m.Format); !explicit["format"] && (f == "u8" || f == "cs16" || f == "f32") {
		*format = f
	}
	if !explicit["offset"] {
		if s, ok := m.System["wmbus_offset_hz"]; ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				*offset = v
			}
		} else if m.CenterFreqHz != 0 && m.CenterFreqHz != wmbus.ChannelHz {
			*offset = float64(wmbus.ChannelHz) - float64(m.CenterFreqHz)
			notes = append(notes, fmt.Sprintf("wmbus: capture centred on %.4f MHz, so 868.95 MHz sits at %+.0f Hz (-offset)", float64(m.CenterFreqHz)/1e6, *offset))
		}
	}
	return notes, nil
}

// decodeWMBusFile replays a capture through a receiver. wav/flac containers
// are detected from the content and carry their own rate (it overrides
// *rateHz); headerless files are read in the named format, in chunks.
//
// saverFor returns the -save-frames saver (nil when off) once newRx has run;
// with one, the IQ is fed in chunks so the saver's history stays bounded.
func decodeWMBusFile(path, format string, rateHz *float64, newRx func(float64) *wmbus.Receiver, saverFor func() *wmbusSnippetSaver) (*wmbus.Receiver, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	head := make([]byte, 64)
	n, _ := io.ReadFull(f, head)
	if _, isContainer := siglab.SniffContainer(head[:n]); isContainer {
		iq, r, err := siglab.DecodeContainerFile(path)
		if err != nil {
			return nil, err
		}
		if r > 0 {
			*rateHz = float64(r)
		}
		rx := newRx(*rateHz)
		feed := wmbusFeeder(rx, saverFor)
		for len(iq) > 0 {
			n := min(len(iq), 65536)
			feed(iq[:n])
			iq = iq[n:]
		}
		rx.Flush()
		return rx, nil
	}
	sf, err := siglab.ParseSampleFormat(format)
	if err != nil {
		return nil, err
	}
	dec, size := sf.Decoder()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	rx := newRx(*rateHz)
	feed := wmbusFeeder(rx, saverFor)
	buf := make([]byte, size*65536)
	iq := make([]complex64, 65536)
	br := bufio.NewReaderSize(f, len(buf))
	for {
		n, err := io.ReadFull(br, buf)
		if n >= size {
			k := n / size
			dec(buf[:k*size], iq[:k])
			feed(iq[:k])
		}
		if err != nil {
			break
		}
	}
	rx.Flush()
	return rx, nil
}

// wmbusFeeder returns the per-chunk step for a file replay: the receiver
// alone, or, with -save-frames, history in, receiver, then any snippet whose
// trailing samples have arrived out.
func wmbusFeeder(rx *wmbus.Receiver, saverFor func() *wmbusSnippetSaver) func([]complex64) {
	var saver *wmbusSnippetSaver
	if saverFor != nil {
		saver = saverFor()
	}
	if saver == nil {
		return rx.Process
	}
	return func(iq []complex64) {
		saver.feed(iq)
		rx.Process(iq)
		if err := saver.flush(false); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// wmbusJSON is the -json line for one telegram.
type wmbusJSON struct {
	Time      string  `json:"time,omitempty"`
	Mode      string  `json:"mode"`
	Format    string  `json:"frame_format"`
	CRCOK     bool    `json:"crc_ok"`
	LevelDBFS float64 `json:"level_dbfs"`
	// CarrierOffsetHz is where the frame's carrier sat relative to
	// 868.95 MHz (mostly the tuner's own frequency error).
	CarrierOffsetHz float64       `json:"carrier_offset_hz"`
	Manufacturer    string        `json:"manufacturer"`
	ID              string        `json:"id"`
	Version         int           `json:"version"`
	DeviceType      int           `json:"device_type"`
	Device          string        `json:"device"`
	CI              int           `json:"ci"`
	AccessNumber    *int          `json:"access_number,omitempty"`
	Status          *int          `json:"status,omitempty"`
	SecurityMode    *int          `json:"security_mode,omitempty"`
	Encrypted       bool          `json:"encrypted"`
	Records         []wmbusRecord `json:"records,omitempty"`
	Error           string        `json:"error,omitempty"`
	Frame           string        `json:"frame"`
	// CRC block detail, for a frame that failed: how many CRC blocks it
	// has, which failed (1-based; block 1 is the link header), and the
	// frame as received with its CRC bytes, so it can be checked offline.
	CRCBlocks       int    `json:"crc_blocks"`
	CRCFailedBlocks []int  `json:"crc_failed_blocks,omitempty"`
	HeaderCRCOK     bool   `json:"header_crc_ok"`
	Raw             string `json:"raw,omitempty"`
	// CorrectedBits is how many bits a soft-decision repair flipped to make
	// the frame's CRC blocks pass (omitted when none).
	CorrectedBits int `json:"corrected_bits,omitempty"`
}

type wmbusRecord struct {
	Quantity string   `json:"quantity"`
	Function string   `json:"function,omitempty"`
	Storage  int      `json:"storage,omitempty"`
	Tariff   int      `json:"tariff,omitempty"`
	Value    *float64 `json:"value,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Text     string   `json:"text,omitempty"`
	Raw      string   `json:"raw"`
}

// wmbusCRCDetail says where a frame's CRC failed. A failed block 1 is the
// link header, so the manufacturer, ID and device type printed before it are
// whatever the corrupt bytes decoded to (issue #1256: a "0x097D …
// electricity" frame whose manufacturer code was not a valid letter triple).
func wmbusCRCDetail(f wmbus.Frame) string {
	if f.Blocks == 0 {
		return ""
	}
	nums := make([]string, len(f.FailedBlocks))
	for i, n := range f.FailedBlocks {
		nums[i] = strconv.Itoa(n)
	}
	word := "block"
	if len(nums) > 1 {
		word = "blocks"
	}
	s := fmt.Sprintf("(%s %s of %d", word, strings.Join(nums, ","), f.Blocks)
	if !f.HeaderCRCOK() {
		return s + "; block 1 is the header, so the sender shown is unreliable)"
	}
	return s + "; header OK)"
}

// writeWMBusTelegram prints one telegram: a summary line plus one indented
// line per data record, or a JSON object.
func writeWMBusTelegram(w io.Writer, ts string, tg wmbus.Telegram, perr error, f wmbus.Frame, info wmbus.FrameInfo, asJSON bool) {
	if asJSON {
		j := wmbusJSON{
			Time: ts, Mode: tg.Mode.String(), Format: tg.Format.String(), CRCOK: tg.CRCOK,
			LevelDBFS: info.LevelDBFS, CarrierOffsetHz: math.Round(info.CarrierOffsetHz), Manufacturer: tg.Manufacturer, ID: tg.ID,
			Version: int(tg.Version), DeviceType: int(tg.DeviceType), Device: tg.DeviceTypeName(),
			CI: int(tg.AppCI), Encrypted: tg.Encrypted, Frame: hex.EncodeToString(f.Data),
			CRCBlocks: f.Blocks, CRCFailedBlocks: f.FailedBlocks, HeaderCRCOK: f.HeaderCRCOK(),
			CorrectedBits: f.CorrectedBits,
		}
		if !f.CRCOK {
			j.Raw = hex.EncodeToString(f.Raw)
		}
		if tg.HasTPL {
			acc, st, sm := int(tg.Access), int(tg.Status), tg.SecurityMode
			j.AccessNumber, j.Status, j.SecurityMode = &acc, &st, &sm
		}
		for _, r := range tg.Records {
			jr := wmbusRecord{Quantity: r.Quantity, Storage: r.Storage, Tariff: r.Tariff, Unit: r.Unit, Text: r.Text, Raw: hex.EncodeToString(r.Raw)}
			if jr.Quantity == "" {
				jr.Quantity = fmt.Sprintf("vif_%02x", r.VIF)
			}
			if r.Function != "instantaneous" {
				jr.Function = r.Function
			}
			if r.HasValue {
				v := r.Value
				jr.Value = &v
			}
			j.Records = append(j.Records, jr)
		}
		if perr != nil {
			j.Error = perr.Error()
		}
		b, _ := json.Marshal(j)
		fmt.Fprintf(w, "%s\n", b)
		return
	}
	var b strings.Builder
	if ts != "" {
		b.WriteString(ts + "  ")
	}
	fmt.Fprintf(&b, "%s-%s %6.1f dBFS  %s %s  %s (v%02X)", tg.Mode, tg.Format, info.LevelDBFS,
		tg.Manufacturer, tg.ID, tg.DeviceTypeName(), tg.Version)
	switch {
	case perr != nil:
		fmt.Fprintf(&b, "  error: %v", perr)
	case !tg.CRCOK:
		b.WriteString("  CRC FAILED " + wmbusCRCDetail(f))
	case tg.Encrypted && tg.HasTPL:
		fmt.Fprintf(&b, "  encrypted: %s, access %d", wmbus.SecurityModeName(tg.SecurityMode), tg.Access)
	case tg.Encrypted:
		b.WriteString("  encrypted (extended link layer)")
	case tg.AppCI >= 0xA0 && tg.AppCI <= 0xB7:
		fmt.Fprintf(&b, "  manufacturer-specific data (CI %02X)", tg.AppCI)
	case len(tg.Records) == 0:
		fmt.Fprintf(&b, "  CI %02X, no data records", tg.AppCI)
	}
	if tg.HasTPL && !tg.Encrypted && tg.Status != 0 {
		fmt.Fprintf(&b, "  status %02X", tg.Status)
	}
	if f.CRCOK && f.CorrectedBits == 1 {
		b.WriteString("  (CRC repaired: 1 bit)")
	} else if f.CRCOK && f.CorrectedBits > 1 {
		fmt.Fprintf(&b, "  (CRC repaired: %d bits)", f.CorrectedBits)
	}
	fmt.Fprintln(w, b.String())
	for _, r := range tg.Records {
		fmt.Fprintf(w, "    %s\n", r.String())
	}
	if tg.RecordsErr != "" {
		fmt.Fprintf(w, "    (records stop: %s)\n", tg.RecordsErr)
	}
}
