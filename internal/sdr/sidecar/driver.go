// Package sidecar implements an sdr.Driver over a generic external IQ
// producer — a "sidecar" process that owns the radio and streams raw samples
// to GopherTrunk over a pipe or socket, steered by a small datagram control
// channel.
//
// The point is to keep hardware and DSP that would drag in CGO out of this
// process while still presenting a real tuner. A UHD/RFNoC program, a GNU
// Radio flowgraph, or a vendor tool with no SoapySDR support can all be a
// sidecar; whatever it does internally — including spatial combining across
// several antennas — GopherTrunk sees one IQ stream.
//
// # Wire format
//
// Downstream (sidecar → GopherTrunk) is a bare stream of interleaved complex
// samples in the configured format. No framing, no header, no sequence
// numbers: it is the same thing `gophertrunk replay -format cs16` reads.
//
//   - cs16:      int16 I, int16 Q, little-endian, 4 bytes/sample
//   - complex64: float32 I, float32 Q, little-endian, 8 bytes/sample
//
// Upstream (GopherTrunk → sidecar) is a UDP datagram of exactly 5 bytes:
//
//	[1-byte opcode][4-byte big-endian uint32 parameter]
//
// The opcodes are rtl_tcp's, not invented here, so a tool already speaking
// rtl_tcp's command protocol works as a sidecar unmodified:
//
//	0x01 SET_CENTER_FREQ   parameter = Hz
//	0x02 SET_SAMPLE_RATE   parameter = Hz
//	0x03 SET_GAIN_MODE     parameter = 1 for manual gain, 0 for the sidecar's AGC
//	0x04 SET_GAIN          parameter = gain in tenths of a dB
//
// A negative gain is sdr.Device's AGC sentinel and is sent as SET_GAIN_MODE 0
// rather than as a wrapped-around SET_GAIN, so there is no signed-encoding
// ambiguity on the wire.
//
// Commands are fire-and-forget: UDP, no reply, no acknowledgement. A sidecar
// that cares about reliability should be on loopback or a LAN, which is where
// this belongs anyway.
package sidecar

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// DriverName is the sdr.Driver name registered with the pool.
const DriverName = "sidecar"

// Control-channel opcodes. These are rtl_tcp's values
// (internal/sdr/rtltcp/driver.go), deliberately: reusing a protocol a tool may
// already speak beats minting a third dialect, and the wire carries no schema
// to catch a mismatch.
const (
	cmdSetCenterFreq uint8 = 0x01
	cmdSetSampleRate uint8 = 0x02
	cmdSetGainMode   uint8 = 0x03
	cmdSetGain       uint8 = 0x04
)

const (
	// DefaultConnectTimeout bounds the TCP dial and the FIFO open.
	DefaultConnectTimeout = 3 * time.Second

	// streamReadDeadline bounds one blocking read so a cancelled context is
	// observed promptly. A sidecar is local IPC or a LAN peer, so this can be
	// far tighter than the 30 s the internet-facing drivers use — it is what
	// keeps a quiet sidecar from adding a teardown tail to every daemon stop.
	streamReadDeadline = time.Second

	// chunkSamples is how many samples one delivered chunk carries.
	chunkSamples = 8192

	// outChanDepth is the bounded hand-off to the DSP consumer.
	outChanDepth = 8

	// maxDatagram caps one UDP read. Jumbo frames and then some.
	maxDatagram = 1 << 16
)

var errClosed = errors.New("sidecar: device closed")

// Spec names one sidecar endpoint to expose as a virtual tuner.
type Spec struct {
	// Transport is "unix_pipe", "tcp" or "udp". Empty means "tcp".
	Transport string
	// DataAddr is a FIFO path (unix_pipe) or host:port (tcp/udp). Required.
	DataAddr string
	// ControlAddr is the sidecar's UDP command socket. Empty disables the
	// control channel: the setters become no-ops.
	ControlAddr string
	// Format is "cs16" (default) or "complex64".
	Format string
	// SampleRateHz is what the sidecar delivers; reported by ActualSampleRate
	// so the decode chain sizes its filters correctly.
	SampleRateHz uint32
	// FreqMinHz / FreqMaxHz declare the tuning range. Both zero = unknown.
	FreqMinHz, FreqMaxHz uint32
	// Serial is the virtual device serial. Empty derives one from DataAddr.
	Serial string
	// Role hints the pool's role assignment.
	Role string
	// ConnectTimeout overrides DefaultConnectTimeout when non-zero.
	ConnectTimeout time.Duration
}

// Driver implements sdr.Driver over a set of sidecar endpoints.
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

// Enumerate returns one Info per configured endpoint. Like the other network
// drivers it does NOT probe: a sidecar that has not started yet would
// otherwise vanish from the pool until the daemon restarts. Open is where
// errors surface.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	out := make([]sdr.Info, 0, len(d.specs))
	for i, spec := range d.specs {
		if spec.DataAddr == "" {
			continue
		}
		out = append(out, sdr.Info{
			Driver:    DriverName,
			Index:     i,
			Serial:    serialFor(spec, i),
			Product:   "sidecar",
			TunerName: transportOf(spec),
			Gains:     genericGainLadder(),
		})
	}
	return out, nil
}

// Open connects the data stream and the control socket for spec[idx].
func (d *Driver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("sidecar: index %d out of range", idx)
	}
	spec := d.specs[idx]
	if spec.DataAddr == "" {
		return nil, errors.New("sidecar: data_addr is required")
	}
	format, err := parseFormat(spec.Format)
	if err != nil {
		return nil, err
	}
	timeout := spec.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}

	data, udp, err := openData(spec, timeout)
	if err != nil {
		return nil, err
	}

	var control net.Conn
	if spec.ControlAddr != "" {
		// UDP "dial" only fixes the destination; it cannot fail on an absent
		// peer, so this is not a liveness check for the sidecar.
		control, err = net.DialTimeout("udp", spec.ControlAddr, timeout)
		if err != nil {
			data.Close()
			return nil, fmt.Errorf("sidecar: dial control %s: %w", spec.ControlAddr, err)
		}
	} else {
		d.log.Warn("sidecar: no control_addr — the sidecar owns tuning, so "+
			"SetCenterFreq / SetSampleRate / SetGain are no-ops here. A trunked "+
			"system that needs to follow voice grants will not work on this device.",
			"data_addr", spec.DataAddr)
	}

	dev := &device{
		data:    data,
		udp:     udp,
		control: control,
		format:  format,
		rateHz:  spec.SampleRateHz,
		minHz:   spec.FreqMinHz,
		maxHz:   spec.FreqMaxHz,
		log:     d.log,
		addr:    spec.DataAddr,
		info: sdr.Info{
			Driver:    DriverName,
			Index:     idx,
			Serial:    serialFor(spec, idx),
			Product:   "sidecar",
			TunerName: transportOf(spec),
			Gains:     genericGainLadder(),
		},
	}
	d.log.Info("sidecar: connected", "data_addr", spec.DataAddr,
		"transport", transportOf(spec), "format", format.name(),
		"control_addr", spec.ControlAddr, "sample_rate_hz", spec.SampleRateHz)
	return dev, nil
}

// deadlineReader is the capability the read loop needs to honour a cancelled
// context: without a read deadline a blocking read cannot be interrupted, and
// the loop outlives the daemon by however long the sidecar stays quiet. Probed
// at Open so a transport that cannot do it fails loudly instead of wedging
// shutdown later.
type deadlineReader interface {
	io.ReadCloser
	SetReadDeadline(t time.Time) error
}

// openData connects the IQ stream. The returned *net.UDPConn is non-nil only
// for the udp transport, which needs datagram-aware reads.
func openData(spec Spec, timeout time.Duration) (deadlineReader, *net.UDPConn, error) {
	switch transportOf(spec) {
	case "unix_pipe":
		f, err := openFIFO(spec.DataAddr, timeout)
		if err != nil {
			return nil, nil, err
		}
		return f, nil, nil
	case "tcp":
		conn, err := net.DialTimeout("tcp", spec.DataAddr, timeout)
		if err != nil {
			return nil, nil, fmt.Errorf("sidecar: dial %s: %w", spec.DataAddr, err)
		}
		return conn.(*net.TCPConn), nil, nil
	case "udp":
		addr, err := net.ResolveUDPAddr("udp", spec.DataAddr)
		if err != nil {
			return nil, nil, fmt.Errorf("sidecar: resolve %s: %w", spec.DataAddr, err)
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, nil, fmt.Errorf("sidecar: bind %s: %w", spec.DataAddr, err)
		}
		return conn, conn, nil
	default:
		return nil, nil, fmt.Errorf("sidecar: unsupported transport %q (want unix_pipe, tcp or udp)",
			spec.Transport)
	}
}

// openFIFO opens a named pipe for reading. Opening a FIFO BLOCKS until a writer
// attaches, so doing it inline would hang daemon startup whenever the sidecar
// is not running yet; the open runs on its own goroutine and is raced against
// the connect timeout.
func openFIFO(path string, timeout time.Duration) (*os.File, error) {
	type result struct {
		f   *os.File
		err error
	}
	done := make(chan result, 1)
	go func() {
		f, err := os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
		done <- result{f, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("sidecar: open fifo %s: %w", path, r.err)
		}
		return r.f, nil
	case <-time.After(timeout):
		// The open may still land later; close it then so the writer is not
		// left with a reader nobody reads.
		go func() {
			if r := <-done; r.err == nil {
				r.f.Close()
			}
		}()
		return nil, fmt.Errorf("sidecar: open fifo %s: no writer attached within %v "+
			"(is the sidecar running?)", path, timeout)
	}
}

type device struct {
	data    deadlineReader
	udp     *net.UDPConn // non-nil only for the udp transport
	control net.Conn     // nil when no control_addr is configured
	format  sampleFormat
	info    sdr.Info
	log     *slog.Logger
	addr    string

	minHz, maxHz uint32

	mu     sync.Mutex
	rateHz uint32
	closed bool
}

func (d *device) Info() sdr.Info { return d.info }

// SetCenterFreq programs the sidecar's centre frequency.
func (d *device) SetCenterFreq(hz uint32) error { return d.send(cmdSetCenterFreq, hz) }

// SetSampleRate programs the sidecar's sample rate and records it, since
// nothing else can report what the stream is actually carrying.
func (d *device) SetSampleRate(hz uint32) error {
	if err := d.send(cmdSetSampleRate, hz); err != nil {
		return err
	}
	d.mu.Lock()
	d.rateHz = hz
	d.mu.Unlock()
	return nil
}

// SetGain programs the gain in tenths of a dB. A negative value is
// sdr.Device's AGC sentinel and is sent as a gain-mode change rather than as a
// wrapped-around unsigned parameter.
func (d *device) SetGain(tenthDB int) error {
	if tenthDB < 0 {
		return d.send(cmdSetGainMode, 0) // 0 = sidecar's own AGC
	}
	if err := d.send(cmdSetGainMode, 1); err != nil {
		return err
	}
	if tenthDB > math.MaxInt32 {
		tenthDB = math.MaxInt32
	}
	return d.send(cmdSetGain, uint32(tenthDB))
}

// SetPPM is a no-op: the sidecar owns the front end, so its own clock
// correction is its business and there is no opcode for it.
func (d *device) SetPPM(int) error { return nil }

// SetBiasTee is a no-op for the same reason.
func (d *device) SetBiasTee(bool) error { return nil }

// FreqRange implements sdr.FreqRanger from the configured bounds. Reporting a
// made-up range would send the whole-device hunt sweep somewhere the hardware
// cannot go, so an unconfigured range is reported as unknown (0, 0).
func (d *device) FreqRange() (minHz, maxHz uint32) { return d.minHz, d.maxHz }

// ActualSampleRate reports the configured rate. There is no probe — the
// sidecar's stream carries no metadata — so this is what the operator declared,
// updated by SetSampleRate.
func (d *device) ActualSampleRate() (uint32, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.rateHz == 0 {
		return 0, errors.New("sidecar: sample_rate_hz is not configured")
	}
	return d.rateHz, nil
}

// send writes one 5-byte command. A device with no control socket accepts and
// discards commands: the operator was warned once at Open, and failing every
// tune would take the device out of the pool entirely.
func (d *device) send(op uint8, param uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return errClosed
	}
	if d.control == nil {
		return nil
	}
	var pkt [5]byte
	pkt[0] = op
	binary.BigEndian.PutUint32(pkt[1:], param)
	_ = d.control.SetWriteDeadline(time.Now().Add(DefaultConnectTimeout))
	if _, err := d.control.Write(pkt[:]); err != nil {
		return fmt.Errorf("sidecar: control write (op %#02x): %w", op, err)
	}
	return nil
}

func (d *device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.control != nil {
		_ = d.control.Close()
	}
	return d.data.Close()
}

// StreamIQ delivers converted samples until the context is cancelled or the
// sidecar hangs up.
func (d *device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, errClosed
	}
	d.mu.Unlock()

	out := make(chan []complex64, outChanDepth)
	go func() {
		defer close(out)
		// A panic in the reader must surface as a logged stream close (which
		// the ccdecoder retries) rather than taking the daemon down with no
		// log line (issue #492).
		defer gtlog.Recover(d.log, "sidecar-stream", nil)
		th := newDropThrottle(d.log, d.addr)
		defer th.flush()
		if d.udp != nil {
			d.readDatagrams(ctx, out, th)
			return
		}
		d.readStream(ctx, out, th)
	}()
	return out, nil
}

// readStream consumes a byte stream (FIFO or TCP). A short read is NOT a
// sample boundary: the leftover bytes are carried into the next read, because
// dropping a partial sample would swap I and Q for the entire rest of the
// stream.
func (d *device) readStream(ctx context.Context, out chan []complex64, th *dropThrottle) {
	bytesPerSample := d.format.bytesPerSample()
	buf := make([]byte, chunkSamples*bytesPerSample)
	carry := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = d.data.SetReadDeadline(time.Now().Add(streamReadDeadline))
		n, err := d.data.Read(buf[carry:])
		if n > 0 {
			total := carry + n
			whole := total - total%bytesPerSample
			if whole > 0 {
				samples := d.format.convert(buf[:whole])
				if !sendOrDrop(ctx, out, samples, th) {
					return
				}
			}
			carry = total - whole
			copy(buf, buf[whole:total])
		}
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				// A quiet sidecar, not a dead one: re-check ctx and read again.
				continue
			}
			if !errors.Is(err, io.EOF) {
				d.log.Debug("sidecar: stream read", "addr", d.addr, "err", err)
			}
			return
		}
	}
}

// readDatagrams consumes UDP. Each datagram is one independent unit — a UDP
// read returns exactly one and discards any excess — so they are decoded
// individually rather than concatenated. A datagram that is not a whole number
// of samples is dropped and counted: carrying a remainder across datagrams
// would splice two unrelated packets together.
func (d *device) readDatagrams(ctx context.Context, out chan []complex64, th *dropThrottle) {
	bytesPerSample := d.format.bytesPerSample()
	buf := make([]byte, maxDatagram)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = d.udp.SetReadDeadline(time.Now().Add(streamReadDeadline))
		n, _, err := d.udp.ReadFromUDP(buf)
		if n > 0 {
			if n%bytesPerSample != 0 {
				th.misaligned()
			} else {
				samples := d.format.convert(buf[:n])
				if !sendOrDrop(ctx, out, samples, th) {
					return
				}
			}
		}
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if !errors.Is(err, io.EOF) {
				d.log.Debug("sidecar: datagram read", "addr", d.addr, "err", err)
			}
			return
		}
	}
}

// sendOrDrop hands samples to the consumer without ever blocking the reader:
// a blocked read stalls the socket and makes the sidecar's own buffers
// overflow, turning one local hiccup into a stream-wide glitch. When the
// bounded channel is full the OLDEST queued chunk is shed instead — latency
// stays low and the data stays freshest. Mirrors the soapyremote driver.
// Returns false only on ctx cancel.
func sendOrDrop(ctx context.Context, out chan []complex64, samples []complex64, th *dropThrottle) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case out <- samples:
			return true
		default:
		}
		select {
		case <-out:
			th.dropped()
		default:
		}
	}
}

// serialFor derives a stable, unique virtual serial from the data address.
func serialFor(spec Spec, idx int) string {
	if spec.Serial != "" {
		return spec.Serial
	}
	return fmt.Sprintf("sidecar-%s-%02d", sanitize(spec.DataAddr), idx)
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			return r
		}
		return '_'
	}, s)
}

func transportOf(spec Spec) string {
	if spec.Transport == "" {
		return "tcp"
	}
	return spec.Transport
}

// genericGainLadder is a coarse 0..50 dB hint for the UI; what a sidecar does
// with a gain value is its own business.
func genericGainLadder() []int {
	out := make([]int, 0, 11)
	for g := 0; g <= 500; g += 50 {
		out = append(out, g)
	}
	return out
}
