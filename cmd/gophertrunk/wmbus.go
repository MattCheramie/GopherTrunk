package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runWMBus is `gophertrunk wmbus`: a Wireless M-Bus (EN 13757-4, T1 and
// C1 modes) telegram decoder for utility meters at 868.95 MHz, reading a
// capture file or a live local / rtl_tcp SDR (issue #1256).
func runWMBus(args []string) {
	fs := flag.NewFlagSet("wmbus", flag.ExitOnError)
	in := fs.String("in", "", "IQ capture to decode instead of a live SDR")
	format := fs.String("format", "u8", "capture sample format for headerless files: u8 (rtl_sdr .cu8) | cs16 | f32; wav and flac are detected from the file")
	rate := fs.Uint("rate", 1_600_000, "sample rate in S/s (a capture's rate; for a live SDR the rate to run at)")
	offset := fs.Float64("offset", 0, "where 868.95 MHz sits relative to the capture/tuner centre, in Hz (e.g. 325000 for a capture centred on 868.625 MHz)")
	freq := fs.Uint("freq", wmbus.ChannelHz, "live SDR centre frequency in Hz")
	gain := fs.String("gain", "auto", `live tuner gain: "auto", or tenths of a dB ("400" = 40 dB)`)
	ppm := fs.Int("ppm", 0, "live frequency correction in ppm")
	serial := fs.String("serial", "", "local SDR serial (required when more than one is attached)")
	rtlTCP := fs.String("rtltcp", "", "host:port of an rtl_tcp server to listen through instead of a local SDR")
	seconds := fs.Float64("seconds", 0, "stop a live run after this many seconds (0 = until Ctrl-C)")
	asJSON := fs.Bool("json", false, "print one JSON object per telegram")
	all := fs.Bool("all", false, "also print frames whose CRC failed")
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

  # Decode an rtl_sdr capture (rtl_sdr -f 868.95M -s 1600000 m.cu8)
  gophertrunk wmbus -in m.cu8 -rate 1600000

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

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	// stamp labels a frame: wall-clock time live, capture time for a file.
	var stamp func(sample int64) string
	newRx := func(rateHz float64) *wmbus.Receiver {
		rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{
			SampleRateHz: rateHz,
			OffsetHz:     *offset,
			OnFrame: func(f wmbus.Frame, info wmbus.FrameInfo) {
				if !f.CRCOK && !*all {
					return
				}
				tg, err := wmbus.ParseTelegram(f)
				writeWMBusTelegram(out, stamp(info.Sample), tg, err, f, info, *asJSON)
				_ = out.Flush()
			},
		})
		if err != nil {
			rep.Fatal(2, err)
		}
		return rx
	}

	if *in != "" {
		rateHz := float64(*rate)
		stamp = func(s int64) string { return fmt.Sprintf("t=%8.3fs", float64(s)/rateHz) }
		rx, err := decodeWMBusFile(*in, *format, &rateHz, newRx)
		if err != nil {
			rep.Fatal(1, err)
		}
		st := rx.Stats()
		fmt.Fprintf(os.Stderr, "wmbus: %.1f s of IQ at %.0f S/s, %d frames (%d CRC-valid), %d access codes\n",
			float64(st.Samples)/rateHz, rateHz, st.Framer.Frames, st.Framer.CRCOK, st.Framer.Syncs)
		return
	}

	gainTenthDB, ok := parseGain(*gain)
	if !ok {
		rep.Fatalf(2, `-gain: want "auto" or tenths of a dB, got %q`, *gain)
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	dev, name, err := openPowerDevice(*rtlTCP, *serial, log)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()
	if err := dev.SetSampleRate(uint32(*rate)); err != nil {
		rep.Fatal(1, fmt.Errorf("set sample rate: %w", err))
	}
	if hw, note := captureEffectiveRate(dev, uint32(*rate)); note != "" {
		fmt.Fprintln(os.Stderr, strings.Replace(note, "capture:", "wmbus:", 1))
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
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
	rx := newRx(float64(*rate))
	fmt.Fprintf(os.Stderr, "wmbus: listening on %s at %.3f MHz, %d S/s (Ctrl-C to stop)\n", name, float64(*freq)/1e6, *rate)
	for {
		select {
		case <-ctx.Done():
			st := rx.Stats()
			fmt.Fprintf(os.Stderr, "wmbus: %d frames (%d CRC-valid), %d access codes\n", st.Framer.Frames, st.Framer.CRCOK, st.Framer.Syncs)
			return
		case iq, ok := <-stream:
			if !ok {
				return
			}
			rx.Process(iq)
		}
	}
}

// decodeWMBusFile replays a capture through a receiver. wav/flac containers
// are detected from the content and carry their own rate (it overrides
// *rateHz); headerless files are read in the named format, in chunks.
func decodeWMBusFile(path, format string, rateHz *float64, newRx func(float64) *wmbus.Receiver) (*wmbus.Receiver, error) {
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
		rx.Process(iq)
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
	buf := make([]byte, size*65536)
	iq := make([]complex64, 65536)
	br := bufio.NewReaderSize(f, len(buf))
	for {
		n, err := io.ReadFull(br, buf)
		if n >= size {
			k := n / size
			dec(buf[:k*size], iq[:k])
			rx.Process(iq[:k])
		}
		if err != nil {
			break
		}
	}
	rx.Flush()
	return rx, nil
}

// wmbusJSON is the -json line for one telegram.
type wmbusJSON struct {
	Time         string        `json:"time,omitempty"`
	Mode         string        `json:"mode"`
	Format       string        `json:"frame_format"`
	CRCOK        bool          `json:"crc_ok"`
	LevelDBFS    float64       `json:"level_dbfs"`
	Manufacturer string        `json:"manufacturer"`
	ID           string        `json:"id"`
	Version      int           `json:"version"`
	DeviceType   int           `json:"device_type"`
	Device       string        `json:"device"`
	CI           int           `json:"ci"`
	AccessNumber *int          `json:"access_number,omitempty"`
	Status       *int          `json:"status,omitempty"`
	SecurityMode *int          `json:"security_mode,omitempty"`
	Encrypted    bool          `json:"encrypted"`
	Records      []wmbusRecord `json:"records,omitempty"`
	Error        string        `json:"error,omitempty"`
	Frame        string        `json:"frame"`
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

// writeWMBusTelegram prints one telegram: a summary line plus one indented
// line per data record, or a JSON object.
func writeWMBusTelegram(w io.Writer, ts string, tg wmbus.Telegram, perr error, f wmbus.Frame, info wmbus.FrameInfo, asJSON bool) {
	if asJSON {
		j := wmbusJSON{
			Time: ts, Mode: tg.Mode.String(), Format: tg.Format.String(), CRCOK: tg.CRCOK,
			LevelDBFS: info.LevelDBFS, Manufacturer: tg.Manufacturer, ID: tg.ID,
			Version: int(tg.Version), DeviceType: int(tg.DeviceType), Device: tg.DeviceTypeName(),
			CI: int(tg.AppCI), Encrypted: tg.Encrypted, Frame: hex.EncodeToString(f.Data),
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
		b.WriteString("  CRC FAILED")
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
	fmt.Fprintln(w, b.String())
	for _, r := range tg.Records {
		fmt.Fprintf(w, "    %s\n", r.String())
	}
	if tg.RecordsErr != "" {
		fmt.Fprintf(w, "    (records stop: %s)\n", tg.RecordsErr)
	}
}
