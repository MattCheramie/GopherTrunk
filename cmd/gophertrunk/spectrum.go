package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"

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
	format := fs.String("format", "f32", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_000_000, "IQ sample rate in Hz")
	freq := fs.Uint64("freq", 0, "capture centre frequency in Hz (0 ⇒ report carrier offsets from DC only)")
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

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
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
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("read %s: %w", *in, err))
	}
	dec, bytesPerSample := sampleFormat.Decoder()
	pairs := len(raw) / bytesPerSample
	if pairs == 0 {
		rep.Fatalf(1, "%s: no samples", *in)
	}
	iq := make([]complex64, pairs)
	dec(raw[:pairs*bytesPerSample], iq)

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
