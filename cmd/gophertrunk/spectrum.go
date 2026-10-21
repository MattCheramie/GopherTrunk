package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/carriers"
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runSpectrum is the entry point for `gophertrunk spectrum`. It is the offline,
// no-decode, no-server analog of the numpy.fft + matplotlib workflow: load a
// capture, compute its Welch-averaged power spectrum, detect the carriers in
// it, and report the band's RMS power — all from the terminal. It reuses the
// production DSP (spectrum.AverageDB, carriers.DetectPeaks, dsp.RMS) so the
// numbers match what the receiver and the web survey see; #679 exposed the same
// survey over REST, this is the CLI surface.
func runSpectrum(args []string) {
	fs := flag.NewFlagSet("spectrum", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ input file (required)")
	format := fs.String("format", "f32", "sample format: u8 | f32 | cs16 (default: the capture's .metadata.json, else f32; wav/flac are detected)")
	sampleRate := fs.Float64("sample-rate", 2_000_000, "IQ sample rate in Hz (default: the capture's .metadata.json, else 2000000)")
	freq := fs.Uint64("freq", 0, "capture centre frequency in Hz (default: the capture's .metadata.json; 0 ⇒ report carrier offsets from DC only)")
	fftSize := fs.Int("fft", 4096, "FFT size for the averaged spectrum (power of two)")
	thresholdDb := fs.Float64("peak-threshold-db", 10, "minimum dB above the noise floor for a carrier to count")
	minSpacing := fs.Uint("min-spacing", 12500, "minimum Hz between detected carriers")
	outFormat := fs.String("out-format", "text", "output format: text | json | csv")
	out := fs.String("out", "", "write output to this file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk spectrum — power spectrum + carrier list + RMS of a capture (no decode).

USAGE:
  gophertrunk spectrum -in <path> [-format u8|f32] [-sample-rate Hz] [-freq Hz]
                      [-fft N] [-peak-threshold-db dB] [-min-spacing Hz]
                      [-out-format text|json|csv] [-out <path>]

EXAMPLES:
  # Find the carriers in a wideband cfile and print the band RMS (offsets from DC)
  gophertrunk spectrum -in wide.cfile -format f32 -sample-rate 2000000

  # Absolute carrier frequencies (give the capture centre), full PSD as JSON
  gophertrunk spectrum -in wide.cfile -format f32 -sample-rate 2000000 \
                      -freq 450500000 -out-format json -out spec.json

Format, rate and centre come from the .metadata.json that 'gophertrunk capture'
writes beside a capture; flags you give win. wav/flac files carry their own rate.

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	explicit := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("spectrum")

	if *in == "" {
		fs.Usage()
		rep.Fatalf(2, "-in is required")
	}
	if *sampleRate <= 0 {
		rep.Fatalf(2, "-sample-rate must be > 0")
	}
	if *fftSize < 8 || *fftSize&(*fftSize-1) != 0 {
		rep.Fatalf(2, "-fft must be a power of two >= 8")
	}
	iq, notes, err := loadSpectrumInput(*in, explicit, format, sampleRate, freq)
	for _, n := range notes {
		fmt.Fprintln(os.Stderr, n)
	}
	if err != nil {
		rep.Fatal(1, err)
	}

	report := buildSpectrumReport(iq, *sampleRate, uint32(*freq), *fftSize, float32(*thresholdDb), uint32(*minSpacing))

	w := os.Stdout
	if *out != "" {
		f, cerr := os.Create(*out)
		if cerr != nil {
			rep.Fatal(1, fmt.Errorf("create %s: %w", *out, cerr))
		}
		defer f.Close()
		w = f
	}
	switch *outFormat {
	case "text":
		writeSpectrumText(w, report)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			rep.Fatal(1, err)
		}
	case "csv":
		if err := writeSpectrumCSV(w, report); err != nil {
			rep.Fatal(1, err)
		}
	default:
		rep.Fatalf(2, "-out-format must be text|json|csv")
	}
}

// loadSpectrumInput reads a capture for `spectrum`. Settings the user did not
// give come from the capture's .metadata.json: a headerless file carries no
// format or rate, and the f32 default read the #1256 reporter's u8 capture as
// floats — every bin NaN, 153 NaN "carriers", rate 2 MS/s instead of 1.6.
// wav/flac containers are detected from their content and carry their rate.
// A file whose samples are not all finite numbers is refused: that is never
// real IQ, only the wrong -format.
func loadSpectrumInput(in string, explicit map[string]bool, format *string, rate *float64, freq *uint64) ([]complex64, []string, error) {
	var notes []string
	if path := siglab.DiscoverMetadata(in); path != "" {
		m, err := siglab.ReadMetadata(path)
		if err != nil {
			return nil, nil, err
		}
		var used []string
		if f := strings.ToLower(m.Format); !explicit["format"] && (f == "u8" || f == "cs16" || f == "f32") {
			*format = f
			used = append(used, f)
		}
		if !explicit["sample-rate"] && m.SampleRateHz > 0 {
			*rate = m.SampleRateHz
			used = append(used, fmt.Sprintf("%.0f S/s", m.SampleRateHz))
		}
		if !explicit["freq"] && m.CenterFreqHz != 0 {
			*freq = uint64(m.CenterFreqHz)
			used = append(used, fmt.Sprintf("centre %.4f MHz", float64(m.CenterFreqHz)/1e6))
		}
		if len(used) > 0 {
			notes = append(notes, fmt.Sprintf("spectrum: %s from %s", strings.Join(used, ", "), path))
		}
	}

	f, err := os.Open(in)
	if err != nil {
		return nil, notes, err
	}
	head := make([]byte, 64)
	n, _ := io.ReadFull(f, head)
	f.Close()
	var iq []complex64
	if _, isContainer := siglab.SniffContainer(head[:n]); isContainer {
		var r uint32
		if iq, r, err = siglab.DecodeContainerFile(in); err != nil {
			return nil, notes, err
		}
		if r > 0 {
			*rate = float64(r)
		}
	} else {
		sf, err := siglab.ParseSampleFormat(*format)
		if err != nil {
			return nil, notes, err
		}
		raw, err := os.ReadFile(in)
		if err != nil {
			return nil, notes, fmt.Errorf("read %s: %w", in, err)
		}
		dec, bytesPerSample := sf.Decoder()
		pairs := len(raw) / bytesPerSample
		iq = make([]complex64, pairs)
		dec(raw[:pairs*bytesPerSample], iq)
	}
	if len(iq) == 0 {
		return nil, notes, fmt.Errorf("%s: no samples", in)
	}
	bad := 0
	for _, v := range iq {
		re, im := float64(real(v)), float64(imag(v))
		if math.IsNaN(re) || math.IsNaN(im) || math.IsInf(re, 0) || math.IsInf(im, 0) {
			bad++
		}
	}
	if bad > 0 {
		return nil, notes, fmt.Errorf("%s: %d of %d samples are not finite numbers when read as %s, so that is not this file's format; "+
			"give the right one (-format u8 for an RTL-SDR .cu8/.raw from `gophertrunk capture -format u8`)", in, bad, len(iq), *format)
	}
	return iq, notes, nil
}

// spectrumCarrier is one detected carrier in the report.
type spectrumCarrier struct {
	OffsetHz  float64 `json:"offset_hz"`         // from the capture centre
	FreqHz    uint64  `json:"freq_hz,omitempty"` // absolute, when -freq was given
	PowerDBFS float64 `json:"power_dbfs"`        // peak bin power
	SNRDb     float64 `json:"snr_db"`            // above the noise floor
}

// spectrumReport is the full spectrum/carrier/RMS report (json/csv schema).
type spectrumReport struct {
	Source       string            `json:"source"`
	SampleRateHz float64           `json:"sample_rate_hz"`
	CenterHz     uint64            `json:"center_hz,omitempty"`
	Samples      int               `json:"samples"`
	DurationSec  float64           `json:"duration_sec"`
	RMSPowerDBFS float64           `json:"rms_power_dbfs"`
	FFTSize      int               `json:"fft_size"`
	Carriers     []spectrumCarrier `json:"carriers"`
	// SpectrumDB is the averaged dBFS spectrum, FFT-shifted (bin 0 = centre -
	// rate/2). Emitted in JSON only.
	SpectrumDB []float32 `json:"spectrum_dbfs,omitempty"`
}

func buildSpectrumReport(iq []complex64, rate float64, freqHz uint32, fftSize int, thresholdDb float32, minSpacing uint32) spectrumReport {
	hasFreq := freqHz != 0
	// DetectPeaks maps bins to absolute Hz as centre+offset and returns uint32,
	// which underflows for a carrier below a zero centre. Use a synthetic centre
	// of one sample rate so every carrier in ±rate/2 maps positive, then recover
	// the signed offset.
	center := freqHz
	if !hasFreq {
		center = uint32(math.Round(rate))
	}
	frame := spectrum.AverageDB(iq, center, rate, fftSize)
	peaks := carriers.DetectPeaks(frame, carriers.PeakOptions{ThresholdDb: thresholdDb, MinSpacingHz: minSpacing})

	cs := make([]spectrumCarrier, 0, len(peaks))
	for _, p := range peaks {
		offset := float64(int64(p.FreqHz) - int64(center))
		c := spectrumCarrier{OffsetHz: offset, PowerDBFS: float64(p.PowerDbFS), SNRDb: float64(p.SNRDb)}
		if hasFreq {
			c.FreqHz = uint64(int64(freqHz) + int64(offset))
		}
		cs = append(cs, c)
	}

	rep := spectrumReport{
		SampleRateHz: rate,
		Samples:      len(iq),
		DurationSec:  float64(len(iq)) / rate,
		RMSPowerDBFS: dsp.Mag2dB(dsp.RMS(iq)),
		FFTSize:      fftSize,
		Carriers:     cs,
		SpectrumDB:   frame.Bins,
	}
	if hasFreq {
		rep.CenterHz = uint64(freqHz)
	}
	return rep
}

func writeSpectrumText(w *os.File, r spectrumReport) {
	fmt.Fprintf(w, "samples=%d  duration=%.3fs  rate=%.0f Hz  rms=%.1f dBFS  fft=%d  carriers=%d\n",
		r.Samples, r.DurationSec, r.SampleRateHz, r.RMSPowerDBFS, r.FFTSize, len(r.Carriers))
	if len(r.Carriers) == 0 {
		fmt.Fprintln(w, "  (no carriers above threshold)")
		return
	}
	for i, c := range r.Carriers {
		if c.FreqHz != 0 {
			fmt.Fprintf(w, "  [%2d] %+9.1f kHz  (%.4f MHz)  %6.1f dBFS  SNR %4.1f dB\n",
				i, c.OffsetHz/1e3, float64(c.FreqHz)/1e6, c.PowerDBFS, c.SNRDb)
		} else {
			fmt.Fprintf(w, "  [%2d] %+9.1f kHz  %6.1f dBFS  SNR %4.1f dB\n",
				i, c.OffsetHz/1e3, c.PowerDBFS, c.SNRDb)
		}
	}
}

func writeSpectrumCSV(w *os.File, r spectrumReport) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"offset_hz", "freq_hz", "power_dbfs", "snr_db"}); err != nil {
		return err
	}
	for _, c := range r.Carriers {
		freq := ""
		if c.FreqHz != 0 {
			freq = strconv.FormatUint(c.FreqHz, 10)
		}
		if err := cw.Write([]string{
			strconv.FormatFloat(c.OffsetHz, 'f', 1, 64),
			freq,
			strconv.FormatFloat(c.PowerDBFS, 'f', 2, 64),
			strconv.FormatFloat(c.SNRDb, 'f', 2, 64),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
