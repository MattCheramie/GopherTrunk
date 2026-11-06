package iio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// DriverName is the sdr.Driver name registered with the pool.
const DriverName = "iio"

// DefaultConnectTimeout bounds the TCP dial and every control command.
const DefaultConnectTimeout = 5 * time.Second

// Device and channel names on Pluto firmware (SoapyPlutoSDR
// PlutoSDR_Settings.cpp / PlutoSDR_Streaming.cpp).
const (
	phyName    = "ad9361-phy"    // the transceiver: LO, rate, gain, bandwidth
	rxName     = "cf-ad9361-lpc" // the RX DMA core the samples come from
	rxLOChan   = "altvoltage0"   // output channel: RX LO
	rxChan     = "voltage0"      // input channel on both devices
	rxIChan    = "voltage0"      // cf-ad9361-lpc scan element: I
	rxQChan    = "voltage1"      // cf-ad9361-lpc scan element: Q
	attrFreq   = "frequency"
	attrRate   = "sampling_frequency"
	attrBW     = "rf_bandwidth"
	attrGainMd = "gain_control_mode"
	attrGain   = "hardwaregain"
	attrPort   = "rf_port_select"
	attrPDown  = "powerdown"
	attrFreqAv = "frequency_available"
)

// AD9361 baseband rate limits without a FIR loaded (SoapyPlutoSDR
// setSampleRate, built without libad9361): below 25 MHz/12 the PHY runs at
// 8x the rate and the HDL decimates by 8, and 8x the rate must still clear
// 25 MHz/12, so the floor is 25 MHz/96. The ceiling is the AD9361's
// 61.44 MS/s.
const (
	minPHYRateHz   = 25_000_000 / 12 // 2 083 333
	minSampleRate  = 25_000_000 / 96 // 260 416
	maxSampleRate  = 61_440_000
	decimateFactor = 8
	minRFBWHz      = 200_000
	maxRFBWHz      = 56_000_000
)

// streamBufferLatency sizes the output channel by time, not chunk count
// (the #1242 / #1321 depth-8 lesson every driver now follows).
const streamBufferLatency = 400 * time.Millisecond

// Spec names one iiod endpoint.
type Spec struct {
	// Addr is the board's address: "192.168.2.1", "pluto.local:30431", or a
	// libiio URI "ip:192.168.2.1". The port defaults to 30431.
	Addr string
	// Serial is the pool serial. Empty derives one from Addr.
	Serial string
	// Role is the pool role hint (control|voice|auto|dedicated|wideband).
	Role string
	// RFBandwidthHz pins the AD9361 RX analog bandwidth. 0 follows the
	// sample rate (clamped to the chip's 200 kHz..56 MHz).
	RFBandwidthHz uint32
	// RFPort selects the RX input (rf_port_select), e.g. "A_BALANCED".
	// Empty leaves the board's setting.
	RFPort string
	// ConnectTimeout overrides DefaultConnectTimeout when non-zero.
	ConnectTimeout time.Duration
}

// Driver implements sdr.Driver.
type Driver struct {
	specs []Spec
	log   *slog.Logger
}

// New builds a Driver over the given endpoints.
func New(specs []Spec, log *slog.Logger) *Driver {
	if log == nil {
		log = slog.Default()
	}
	return &Driver{specs: specs, log: log}
}

// Name implements sdr.Driver.
func (d *Driver) Name() string { return DriverName }

// Enumerate lists every configured endpoint without dialling it, so a board
// that is briefly unreachable still has a pool slot; Open reports errors.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	out := make([]sdr.Info, 0, len(d.specs))
	for i, s := range d.specs {
		if s.Addr == "" {
			continue
		}
		out = append(out, sdr.Info{
			Driver:    DriverName,
			Index:     i,
			Serial:    serialFor(s, i),
			Product:   "AD936x over iiod",
			TunerName: "AD9361",
			Gains:     gainLadder(),
		})
	}
	return out, nil
}

// NormalizeAddr turns an operator-written address into host:port: a libiio
// "ip:" prefix is dropped and a missing port becomes 30431.
func NormalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "ip:")
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), strconv.Itoa(DefaultPort))
}

// Open connects to the board, reads its context, checks it is a Pluto-
// style AD936x, and powers the RX LO up.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("iio: index %d out of range", idx)
	}
	spec := d.specs[idx]
	if spec.Addr == "" {
		return nil, errors.New("iio: spec missing addr")
	}
	timeout := spec.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	addr := NormalizeAddr(spec.Addr)
	c, err := dial(addr, timeout)
	if err != nil {
		return nil, explainDialError(addr, err)
	}
	dev, err := openOn(c, addr, spec, idx, d.log)
	if err != nil {
		c.Close()
		return nil, err
	}
	return dev, nil
}

func openOn(c *client, addr string, spec Spec, idx int, log *slog.Logger) (*device, error) {
	version, err := c.Version()
	if err != nil {
		return nil, fmt.Errorf("iio: %s: %w", addr, err)
	}
	raw, err := c.PrintXML()
	if err != nil {
		return nil, fmt.Errorf("iio: %s: %w", addr, err)
	}
	ctx, err := ParseContext(raw)
	if err != nil {
		return nil, fmt.Errorf("iio: %s: %w", addr, err)
	}
	dv, err := bindDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("iio: %s: %w", addr, err)
	}
	dv.addr = addr
	dv.ctl = c
	dv.timeout = c.timeout
	dv.spec = spec
	dv.log = log
	dv.version = version
	dv.minHz, dv.maxHz = 70_000_000, math.MaxUint32
	if dv.loChan.HasAttr(attrFreqAv) {
		if v, err := c.ReadChnAttr(dv.phy.ID, true, rxLOChan, attrFreqAv); err == nil {
			if lo, hi, ok := parseRange(v); ok {
				dv.minHz, dv.maxHz = lo, hi
			}
		}
	}
	if dv.loChan.HasAttr(attrPDown) {
		// SoapyPlutoSDR rx_streamer: "Turn ON RX LO" (write_bool false).
		if err := c.WriteChnAttr(dv.phy.ID, true, rxLOChan, attrPDown, "0"); err != nil {
			return nil, fmt.Errorf("iio: %s: power up RX LO: %w", addr, err)
		}
	}
	if spec.RFPort != "" {
		if err := c.WriteChnAttr(dv.phy.ID, false, rxChan, attrPort, spec.RFPort); err != nil {
			return nil, fmt.Errorf("iio: %s: rf_port_select %q: %w", addr, spec.RFPort, err)
		}
	}
	model := ctx.Attrs["hw_model"]
	if model == "" {
		model = ctx.Description
	}
	dv.info = sdr.Info{
		Driver:       DriverName,
		Index:        idx,
		Serial:       serialFor(spec, idx),
		Manufacturer: "Analog Devices",
		Product:      strings.TrimSpace("AD936x " + model),
		TunerName:    "AD9361",
		Gains:        gainLadder(),
	}
	log.Info("iio: connected",
		"addr", addr,
		"iiod_version", version,
		"hw_model", ctx.Attrs["hw_model"],
		"fw_version", ctx.Attrs["fw_version"],
		"rx_device", dv.rx.ID,
		"sample_format", dv.iChan.Scan.Raw,
		"full_scale", dv.fullScale,
		"freq_range_hz", fmt.Sprintf("%d-%d", dv.minHz, dv.maxHz))
	return dv, nil
}

// bindDevices finds the PHY and RX devices and checks the sample format.
func bindDevices(ctx *Context) (*device, error) {
	phy := ctx.FindDevice(phyName)
	rx := ctx.FindDevice(rxName)
	if phy == nil || rx == nil {
		var names []string
		for _, d := range ctx.Devices {
			names = append(names, d.Name)
		}
		return nil, fmt.Errorf("not a Pluto-style AD936x board: need devices %s and %s, board has [%s]",
			phyName, rxName, strings.Join(names, ", "))
	}
	lo := phy.FindChannel(rxLOChan, true)
	pch := phy.FindChannel(rxChan, false)
	ich := rx.FindChannel(rxIChan, false)
	qch := rx.FindChannel(rxQChan, false)
	switch {
	case lo == nil:
		return nil, fmt.Errorf("%s has no output channel %s (RX LO)", phyName, rxLOChan)
	case pch == nil:
		return nil, fmt.Errorf("%s has no input channel %s", phyName, rxChan)
	case ich == nil || qch == nil || ich.Scan == nil || qch.Scan == nil:
		return nil, fmt.Errorf("%s has no %s/%s scan elements (I/Q)", rxName, rxIChan, rxQChan)
	}
	for _, ch := range []*Channel{ich, qch} {
		f := ch.Scan
		// Pluto RX is "le:S12/16>>0": 12-bit two's complement, LSB-aligned
		// in 16 bits, full scale 2048 (SoapyPlutoSDR getNativeStreamFormat,
		// the #1331 level fix). Refuse anything else rather than guess.
		if f.BigEndian || !f.Signed || f.Length != 16 || f.Repeat != 1 || f.Bits < 2 || f.Bits > 16 || f.Bits+f.Shift > 16 {
			return nil, fmt.Errorf("%s %s sample format %q is not the 16-bit little-endian signed form this driver reads", rxName, ch.ID, f.Raw)
		}
	}
	if ich.Scan.Bits != qch.Scan.Bits || ich.Scan.Shift != qch.Scan.Shift {
		return nil, fmt.Errorf("%s I/Q formats differ (%q vs %q)", rxName, ich.Scan.Raw, qch.Scan.Raw)
	}
	first, second := ich, qch
	if qch.Number < ich.Number {
		first, second = qch, ich
	}
	return &device{
		phy:       phy,
		rx:        rx,
		loChan:    lo,
		phyChan:   pch,
		iChan:     ich,
		qChan:     qch,
		iFirst:    first == ich,
		mask:      rx.Mask(first, second),
		maskWords: rx.MaskWords(),
		bits:      ich.Scan.Bits,
		shift:     ich.Scan.Shift,
		fullScale: float32(int(1) << uint(ich.Scan.Bits-1)),
	}, nil
}

// parseRange reads an IIO "[min step max]" range.
func parseRange(s string) (lo, hi uint32, ok bool) {
	f := strings.Fields(strings.Trim(strings.TrimSpace(s), "[]"))
	if len(f) != 3 {
		return 0, 0, false
	}
	a, err1 := strconv.ParseUint(f[0], 10, 64)
	b, err2 := strconv.ParseUint(f[2], 10, 64)
	if err1 != nil || err2 != nil || a >= b {
		return 0, 0, false
	}
	if b > math.MaxUint32 {
		b = math.MaxUint32
	}
	if a > math.MaxUint32 {
		return 0, 0, false
	}
	return uint32(a), uint32(b), true
}

type device struct {
	addr    string
	spec    Spec
	info    sdr.Info
	log     *slog.Logger
	version string
	timeout time.Duration

	phy, rx        *Device
	loChan         *Channel
	phyChan        *Channel
	iChan, qChan   *Channel
	iFirst         bool
	mask           string
	maskWords      int
	bits, shift    int
	fullScale      float32
	minHz, maxHz   uint32
	ppmWarned      bool
	streamDialFunc func() (*client, error) // tests override

	mu        sync.Mutex // guards ctl, closed, rate, streamErr
	ctl       *client
	closed    bool
	rate      uint32
	streamErr error
}

func (d *device) Info() sdr.Info { return d.info }

// FreqRange implements sdr.FreqRanger from the LO's frequency_available.
func (d *device) FreqRange() (minHz, maxHz uint32) { return d.minHz, d.maxHz }

// StreamErr implements sdr.StreamEnder.
func (d *device) StreamErr() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streamErr
}

func (d *device) control(fn func(c *client) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.ctl == nil {
		return errors.New("iio: device closed")
	}
	return fn(d.ctl)
}

// SetCenterFreq writes the RX LO frequency.
func (d *device) SetCenterFreq(hz uint32) error {
	if hz < d.minHz || hz > d.maxHz {
		return fmt.Errorf("iio: %d Hz is outside this board's RX LO range %d-%d Hz", hz, d.minHz, d.maxHz)
	}
	return d.control(func(c *client) error {
		return c.WriteChnAttr(d.phy.ID, true, rxLOChan, attrFreq, strconv.FormatUint(uint64(hz), 10))
	})
}

// SampleRatePlan is how a rate maps onto the AD9361: the PHY rate and
// whether the RX core's x8 decimator runs.
func SampleRatePlan(hz uint32) (phyHz uint32, decimate bool, err error) {
	if hz < minSampleRate || hz > maxSampleRate {
		return 0, false, fmt.Errorf("iio: sample rate %d Hz is outside the AD9361's %d-%d Hz without a FIR filter loaded", hz, minSampleRate, maxSampleRate)
	}
	if hz < minPHYRateHz {
		return hz * decimateFactor, true, nil
	}
	return hz, false, nil
}

// SetSampleRate writes the PHY rate, the RX core's rate (which engages its
// x8 decimator below 25 MHz/12, as SoapyPlutoSDR does), and the RX analog
// bandwidth.
func (d *device) SetSampleRate(hz uint32) error {
	phyHz, _, err := SampleRatePlan(hz)
	if err != nil {
		return err
	}
	bw := d.spec.RFBandwidthHz
	if bw == 0 {
		bw = hz
	}
	if bw < minRFBWHz {
		bw = minRFBWHz
	}
	if bw > maxRFBWHz {
		bw = maxRFBWHz
	}
	return d.control(func(c *client) error {
		if err := c.WriteChnAttr(d.phy.ID, false, rxChan, attrRate, strconv.FormatUint(uint64(phyHz), 10)); err != nil {
			return err
		}
		if d.iChan.HasAttr(attrRate) {
			if err := c.WriteChnAttr(d.rx.ID, false, rxChan, attrRate, strconv.FormatUint(uint64(hz), 10)); err != nil {
				return err
			}
		} else if phyHz != hz {
			return fmt.Errorf("iio: %d Hz needs the RX core's x8 decimator, but %s %s has no %s attribute", hz, rxName, rxChan, attrRate)
		}
		if d.phyChan.HasAttr(attrBW) {
			if err := c.WriteChnAttr(d.phy.ID, false, rxChan, attrBW, strconv.FormatUint(uint64(bw), 10)); err != nil {
				return err
			}
		}
		d.rate = hz
		return nil
	})
}

// ActualSampleRate reads the rate the RX core delivers back from the board
// (the AD9361 driver rounds), so the pool can warn on a mismatch.
func (d *device) ActualSampleRate() (uint32, error) {
	var out uint32
	err := d.control(func(c *client) error {
		dev, ch := d.rx.ID, rxChan
		if !d.iChan.HasAttr(attrRate) {
			dev = d.phy.ID
		}
		v, err := c.ReadChnAttr(dev, false, ch, attrRate)
		if err != nil {
			return err
		}
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return fmt.Errorf("iio: %s: unparseable rate %q", attrRate, v)
		}
		out = uint32(n)
		return nil
	})
	return out, err
}

// SetGain selects the AD9361 AGC (slow_attack, the SoapyPlutoSDR choice)
// for -1, otherwise manual gain in whole dB (tenths rounded).
func (d *device) SetGain(tenthDB int) error {
	return d.control(func(c *client) error {
		if tenthDB < 0 {
			return c.WriteChnAttr(d.phy.ID, false, rxChan, attrGainMd, "slow_attack")
		}
		if err := c.WriteChnAttr(d.phy.ID, false, rxChan, attrGainMd, "manual"); err != nil {
			return err
		}
		db := (tenthDB + 5) / 10
		return c.WriteChnAttr(d.phy.ID, false, rxChan, attrGain, strconv.Itoa(db))
	})
}

// SetPPM is a no-op: the AD9361 exposes no ppm trim through the attributes
// this driver drives (xo_correction is board-specific). The carrier-offset
// estimator reports the error as on any SDR.
func (d *device) SetPPM(ppm int) error {
	if ppm != 0 && !d.ppmWarned {
		d.ppmWarned = true
		d.log.Warn("iio: ppm is not applied on AD936x boards; use the decoder's measured carrier offset instead",
			"serial", d.info.Serial, "ppm", ppm)
	}
	return nil
}

// SetBiasTee is a no-op: Pluto-style boards have no bias-tee.
func (d *device) SetBiasTee(bool) error { return nil }

func (d *device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.ctl != nil {
		return d.ctl.Close()
	}
	return nil
}

// bufferSamples picks the iiod buffer size: about 1/60 s rounded up to a
// power of two, as SoapyPlutoSDR's set_buffer_size_by_samplerate does.
func bufferSamples(rate uint32) int {
	target := int(math.Round(float64(rate) / 60))
	n := 1
	for n < target {
		n <<= 1
	}
	if n < 4096 {
		n = 4096
	}
	if n > 1<<20 {
		n = 1 << 20
	}
	return n
}

func streamDepth(rate uint32, samples int) int {
	if rate == 0 || samples <= 0 {
		return 8
	}
	n := int(math.Ceil(streamBufferLatency.Seconds() * float64(rate) / float64(samples)))
	if n < 4 {
		n = 4
	}
	return n
}

// StreamIQ opens the RX core on a second connection (libiio's network
// backend also streams on a socket of its own, so control writes such as a
// retune never queue behind sample data) and loops READBUF.
func (d *device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, errors.New("iio: device closed")
	}
	rate := d.rate
	d.streamErr = nil
	d.mu.Unlock()
	if rate == 0 {
		rate = 2_400_000
	}

	dialStream := d.streamDialFunc
	if dialStream == nil {
		dialStream = func() (*client, error) { return dial(d.addr, d.timeout) }
	}
	sc, err := dialStream()
	if err != nil {
		return nil, fmt.Errorf("iio: stream dial %s: %w", d.addr, err)
	}
	samples := bufferSamples(rate)
	if err := sc.Open(d.rx.ID, samples, d.mask); err != nil {
		sc.Close()
		return nil, fmt.Errorf("iio: %s: open RX buffer: %w", d.addr, err)
	}
	// Streaming has no per-command deadline: READBUF blocks until the board
	// fills a buffer, and cancellation closes the socket instead.
	sc.timeout = 0
	_ = sc.conn.SetDeadline(time.Time{})

	out := make(chan []complex64, streamDepth(rate, samples))
	go func() {
		defer close(out)
		defer gtlog.Recover(d.log, "iio-stream", nil)

		// Closing the socket on cancel unblocks a READBUF parked on a quiet
		// board at once (the shutdown-stall lesson: never wait out a long
		// read deadline on ctx cancel).
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			select {
			case <-ctx.Done():
			case <-stop:
			}
			sc.Close()
		}()

		started := time.Now()
		var total int64
		emit := func(b []byte) error {
			total += int64(len(b))
			iq := d.convert(b)
			select {
			case out <- iq:
			case <-ctx.Done():
				return ctx.Err()
			default:
				sdr.NotifyIQDrop(d.info)
			}
			return nil
		}
		for ctx.Err() == nil {
			err := sc.ReadBuf(d.rx.ID, samples*4, d.maskWords, emit)
			if err == nil || errors.Is(err, errReadbufShort) {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			why := fmt.Errorf("iio %s: stream ended after %.1f s and %d bytes of IQ: %w",
				d.addr, time.Since(started).Seconds(), total, err)
			d.mu.Lock()
			d.streamErr = why
			d.mu.Unlock()
			d.log.Warn("iio: IQ stream ended", "addr", d.addr, "err", why)
			return
		}
	}()
	return out, nil
}

// convert maps interleaved 16-bit little-endian sample words to complex64
// in [-1, 1): shift, keep the significant bits, sign-extend (what libiio's
// iio_channel_convert does for an "S12/16" channel), divide by full scale.
func (d *device) convert(b []byte) []complex64 {
	n := len(b) / 4
	out := make([]complex64, n)
	sh := uint(d.shift)
	ext := uint(32 - d.bits)
	scale := 1 / d.fullScale
	for i := 0; i < n; i++ {
		w0 := uint32(uint16(b[4*i]) | uint16(b[4*i+1])<<8)
		w1 := uint32(uint16(b[4*i+2]) | uint16(b[4*i+3])<<8)
		a := float32(int32((w0>>sh)<<ext)>>ext) * scale
		c := float32(int32((w1>>sh)<<ext)>>ext) * scale
		if d.iFirst {
			out[i] = complex(a, c)
		} else {
			out[i] = complex(c, a)
		}
	}
	return out
}

func gainLadder() []int {
	g := make([]int, 0, 74)
	for db := 0; db <= 73; db++ {
		g = append(g, db*10)
	}
	return g
}

func serialFor(s Spec, idx int) string {
	if s.Serial != "" {
		return s.Serial
	}
	addr := NormalizeAddr(s.Addr)
	b := make([]byte, 0, len(addr))
	for i := 0; i < len(addr); i++ {
		c := addr[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return fmt.Sprintf("iio-%s-%02d", b, idx)
}
