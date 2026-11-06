package iio

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// ProbeReport is what Probe learned about a board. It is filled as far as
// the probe got: a board that is not Pluto-shaped still reports its version,
// context XML and device list, with BindErr saying why the driver would
// refuse it.
type ProbeReport struct {
	Addr    string
	Version string
	XML     []byte
	Context *Context
	// BindErr is the error Driver.Open would fail with on this board's
	// context (missing ad9361-phy / cf-ad9361-lpc, unexpected sample
	// format). Nil when the board is usable.
	BindErr error

	RXDevice     string // e.g. "iio:device4"
	SampleFormat string // e.g. "le:S12/16>>0"
	FullScale    float32

	// Read back from the board. Empty when the attribute is absent or
	// the read failed (the error is in ReadErrs).
	LOFrequency   string
	LORange       string
	PHYRate       string
	RXRate        string
	RFBandwidth   string
	GainMode      string
	HardwareGain  string
	RFPort        string
	ReadErrs      []string
	StreamErr     error
	StreamSamples int
	RMSdBFS       float64
	PeakdBFS      float64
	ClipFraction  float64 // share of I/Q values at the ADC rail
}

// Probe connects to iiod at addr and reports what the driver would see,
// without writing any attribute: the board's settings are left as they are.
// With streamSamples > 0 it also opens the RX buffer and reads that many
// samples once, to check the sample path and measure the level.
func Probe(addr string, timeout time.Duration, streamSamples int) (*ProbeReport, error) {
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	addr = NormalizeAddr(addr)
	rep := &ProbeReport{Addr: addr}
	c, err := dial(addr, timeout)
	if err != nil {
		return rep, explainDialError(addr, err)
	}
	defer c.Close()
	if rep.Version, err = c.Version(); err != nil {
		return rep, err
	}
	if rep.XML, err = c.PrintXML(); err != nil {
		return rep, err
	}
	if rep.Context, err = ParseContext(rep.XML); err != nil {
		return rep, err
	}
	dv, err := bindDevices(rep.Context)
	if err != nil {
		rep.BindErr = err
		return rep, nil
	}
	rep.RXDevice = dv.rx.ID
	rep.SampleFormat = dv.iChan.Scan.Raw
	rep.FullScale = dv.fullScale

	read := func(dst *string, dev string, ch *Channel, output bool, attr string) {
		if !ch.HasAttr(attr) {
			return
		}
		v, err := c.ReadChnAttr(dev, output, ch.ID, attr)
		if err != nil {
			rep.ReadErrs = append(rep.ReadErrs, fmt.Sprintf("%s %s %s: %v", dev, ch.ID, attr, err))
			return
		}
		*dst = v
	}
	read(&rep.LOFrequency, dv.phy.ID, dv.loChan, true, attrFreq)
	read(&rep.LORange, dv.phy.ID, dv.loChan, true, attrFreqAv)
	read(&rep.PHYRate, dv.phy.ID, dv.phyChan, false, attrRate)
	read(&rep.RXRate, dv.rx.ID, dv.iChan, false, attrRate)
	read(&rep.RFBandwidth, dv.phy.ID, dv.phyChan, false, attrBW)
	read(&rep.GainMode, dv.phy.ID, dv.phyChan, false, attrGainMd)
	read(&rep.HardwareGain, dv.phy.ID, dv.phyChan, false, attrGain)
	read(&rep.RFPort, dv.phy.ID, dv.phyChan, false, attrPort)

	if streamSamples > 0 {
		rep.StreamErr = probeStream(addr, timeout, dv, streamSamples, rep)
	}
	return rep, nil
}

// probeStream reads one buffer on a second connection, as StreamIQ does.
func probeStream(addr string, timeout time.Duration, dv *device, samples int, rep *ProbeReport) error {
	sc, err := dial(addr, timeout)
	if err != nil {
		return fmt.Errorf("stream dial: %w", err)
	}
	defer sc.Close()
	if err := sc.Open(dv.rx.ID, samples, dv.mask); err != nil {
		return fmt.Errorf("open RX buffer: %w", err)
	}
	// One READBUF of a whole buffer; a board that never fills it times
	// out on the per-command deadline instead of hanging the probe.
	_ = sc.conn.SetDeadline(time.Now().Add(timeout))
	var sumSq, peak float64
	var n, clipped int
	rail := float64(dv.fullScale) - 1
	err = sc.ReadBuf(dv.rx.ID, samples*4, dv.maskWords, func(b []byte) error {
		for _, v := range dv.convert(b) {
			for _, x := range [2]float64{float64(real(v)), float64(imag(v))} {
				a := math.Abs(x)
				sumSq += x * x
				if a > peak {
					peak = a
				}
				if a*float64(dv.fullScale) >= rail {
					clipped++
				}
			}
			n++
		}
		return nil
	})
	_ = sc.conn.SetDeadline(time.Time{})
	if err != nil && !errors.Is(err, errReadbufShort) {
		return err
	}
	_ = sc.CloseDev(dv.rx.ID)
	rep.StreamSamples = n
	if n > 0 {
		rep.RMSdBFS = 10 * math.Log10(sumSq/float64(n)+1e-30)
		rep.PeakdBFS = 20 * math.Log10(peak+1e-30)
		rep.ClipFraction = float64(clipped) / float64(2*n)
	}
	if n == 0 {
		return errors.New("the board returned no samples")
	}
	return nil
}

// String renders the report for a terminal, ready to paste into an issue.
func (r *ProbeReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "iiod %s at %s\n", r.Version, r.Addr)
	if r.Context != nil {
		for _, k := range []string{"hw_model", "fw_version", "ad9361-phy,model", "hw_serial"} {
			if v, ok := r.Context.Attrs[k]; ok {
				fmt.Fprintf(&b, "  %-18s %s\n", k+":", v)
			}
		}
		b.WriteString("  devices:\n")
		for _, d := range r.Context.Devices {
			var scan []string
			for _, ch := range d.Channels {
				if ch.Scan != nil {
					scan = append(scan, ch.ID+"="+ch.Scan.Raw)
				}
			}
			line := fmt.Sprintf("    %-13s %-24s %d channels", d.ID, d.Name, len(d.Channels))
			if len(scan) > 0 {
				line += "  scan: " + strings.Join(scan, " ")
			}
			b.WriteString(line + "\n")
		}
	}
	if r.BindErr != nil {
		fmt.Fprintf(&b, "NOT USABLE by sdr.iio: %v\n", r.BindErr)
		return b.String()
	}
	fmt.Fprintf(&b, "usable by sdr.iio: RX %s, format %s, full scale %.0f\n", r.RXDevice, r.SampleFormat, r.FullScale)
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "  %-18s %s\n", k+":", v)
		}
	}
	row("RX LO frequency", r.LOFrequency)
	row("RX LO range", r.LORange)
	row("PHY sample rate", r.PHYRate)
	row("RX core rate", r.RXRate)
	row("RF bandwidth", r.RFBandwidth)
	row("gain mode", r.GainMode)
	row("hardware gain", r.HardwareGain)
	row("RF port", r.RFPort)
	for _, e := range r.ReadErrs {
		fmt.Fprintf(&b, "  read failed: %s\n", e)
	}
	switch {
	case r.StreamErr != nil:
		fmt.Fprintf(&b, "stream: FAILED: %v\n", r.StreamErr)
	case r.StreamSamples > 0:
		fmt.Fprintf(&b, "stream: %d samples, RMS %.1f dBFS, peak %.1f dBFS, %.2f%% at the ADC rail\n",
			r.StreamSamples, r.RMSdBFS, r.PeakdBFS, 100*r.ClipFraction)
	}
	return b.String()
}
