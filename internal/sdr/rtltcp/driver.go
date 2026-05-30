// Package rtltcp implements an sdr.Driver that talks to a remote
// rtl_tcp server. rtl_tcp ships with librtlsdr and exposes a single
// RTL-SDR dongle over a TCP socket so a host with USB access can
// share its radio with hosts on the network. SDR++, Gqrx, OpenWebRX,
// and SDRangel all consume the same protocol — adding it here lets
// GopherTrunk demodulate trunked systems from an SDR plugged into a
// remote Raspberry Pi / NUC / VM host.
//
// Protocol summary (well-known, unencrypted, single-channel):
//
//   - Client opens a TCP connection.
//   - Server sends a 12-byte header: 4-byte magic "RTL0", 4 bytes
//     tuner-type (big-endian uint32), 4 bytes tuner gain-count
//     (big-endian uint32).
//   - Server then streams unsigned 8-bit IQ samples (interleaved
//     I, Q, I, Q, ...). Each sample maps to complex64 via
//     (b - 127.5) / 127.5, matching the local USB rtlsdr driver.
//   - Client sends commands as 5-byte packets: 1-byte opcode +
//     4-byte big-endian uint32 parameter. Setters send a command
//     and immediately return; the server applies the change to its
//     local dongle.
//
// The driver supports one remote per Spec; multiple Specs in one
// config (each pointing at a different rtl_tcp host) become separate
// pool entries that the engine roles via the usual control/voice
// hinting.
//
// Limitations:
//   - rtl_tcp is single-tuner. A single endpoint can host one client.
//   - rtl_tcp is plaintext. Use it on trusted networks only or over
//     an SSH/wireguard tunnel.
//   - Bias-tee + advanced rtlsdr-only knobs (direct sampling, offset
//     tuning, IF gain) are wired through the command channel but
//     not exposed on sdr.Device for now; SetBiasTee operates if the
//     remote runs librtlsdr >= 0.7.
package rtltcp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// DriverName is the sdr.Driver name registered with the pool.
const DriverName = "rtltcp"

// Magic is the four-byte "RTL0" tag that opens the server-sent
// header.
var Magic = [4]byte{'R', 'T', 'L', '0'}

// Wire-level commands. Values match the librtlsdr rtl_tcp server.
const (
	cmdSetFreq         uint8 = 0x01
	cmdSetSampleRate   uint8 = 0x02
	cmdSetGainMode     uint8 = 0x03
	cmdSetGain         uint8 = 0x04
	cmdSetFreqCorrPPM  uint8 = 0x05
	cmdSetIFGain       uint8 = 0x06
	cmdSetTestMode     uint8 = 0x07
	cmdSetAGCMode      uint8 = 0x08
	cmdSetDirectSamp   uint8 = 0x09
	cmdSetOffsetTuning uint8 = 0x0a
	cmdSetRTLXtal      uint8 = 0x0b
	cmdSetTunerXtal    uint8 = 0x0c
	cmdSetGainByIndex  uint8 = 0x0d
	cmdSetBiasTee      uint8 = 0x0e
)

// DefaultConnectTimeout caps the TCP dial used in Open and Enumerate.
// rtl_tcp servers typically live on the same LAN; if the dial blocks
// past this bound, the operator probably has the wrong host:port
// configured and a fast failure is more useful than a hang.
const DefaultConnectTimeout = 3 * time.Second

// Spec names one rtl_tcp endpoint to expose as a virtual tuner.
type Spec struct {
	// Addr is the host:port pair the rtl_tcp server is listening on,
	// e.g. "192.168.1.50:1234". Required.
	Addr string
	// Serial is the virtual device serial the pool reports. Empty
	// generates one from Addr so multi-endpoint configs stay unique.
	Serial string
	// Role hints the pool's role assignment: "control" | "voice" |
	// "auto". Honoured by the pool's hint matcher.
	Role string
	// ConnectTimeout overrides DefaultConnectTimeout when non-zero.
	ConnectTimeout time.Duration
}

// Driver implements sdr.Driver. Construct it from config (see the
// daemon's wiring) and register with sdr.Register before Pool.Open.
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

// Enumerate returns one Info per configured endpoint. The driver does
// NOT probe each server at enumeration time — a remote that's
// temporarily down would otherwise vanish from the pool until the
// next restart. Open is the place errors surface.
func (d *Driver) Enumerate() ([]sdr.Info, error) {
	out := make([]sdr.Info, 0, len(d.specs))
	for i, spec := range d.specs {
		if spec.Addr == "" {
			continue
		}
		out = append(out, sdr.Info{
			Driver:    DriverName,
			Index:     i,
			Serial:    serialFor(spec, i),
			Product:   "rtl_tcp remote",
			TunerName: "rtl_tcp",
			Gains:     standardGainLadder(),
		})
	}
	return out, nil
}

// Open dials the rtl_tcp server at spec[idx] and returns a Device
// ready for setters and StreamIQ. The 12-byte header is consumed
// here; subsequent reads from the connection yield IQ bytes.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("rtltcp: index %d out of range", idx)
	}
	spec := d.specs[idx]
	if spec.Addr == "" {
		return nil, errors.New("rtltcp: spec missing Addr")
	}
	timeout := spec.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}

	conn, err := net.DialTimeout("tcp", spec.Addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("rtltcp: dial %s: %w", spec.Addr, err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("rtltcp: set read deadline: %w", err)
	}
	var hdr [12]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		conn.Close()
		return nil, fmt.Errorf("rtltcp: read header: %w", err)
	}
	// Validate magic — guards against connecting to a non-rtl_tcp
	// service on the same port (HTTP server, SSH banner, etc.).
	if hdr[0] != Magic[0] || hdr[1] != Magic[1] || hdr[2] != Magic[2] || hdr[3] != Magic[3] {
		conn.Close()
		return nil, fmt.Errorf("rtltcp: %s: server header magic = %q, want %q", spec.Addr, hdr[:4], Magic[:])
	}
	tunerType := binary.BigEndian.Uint32(hdr[4:8])
	gainCount := binary.BigEndian.Uint32(hdr[8:12])
	// Clear the read deadline so StreamIQ blocks normally.
	_ = conn.SetReadDeadline(time.Time{})

	info := sdr.Info{
		Driver:    DriverName,
		Index:     idx,
		Serial:    serialFor(spec, idx),
		Product:   "rtl_tcp remote",
		TunerName: tunerName(tunerType),
		Gains:     standardGainLadder(),
	}
	d.log.Info("rtltcp: connected",
		"addr", spec.Addr,
		"tuner", info.TunerName,
		"gain_count", gainCount)
	return &device{
		addr: spec.Addr,
		conn: conn,
		info: info,
		log:  d.log,
	}, nil
}

// device implements sdr.Device on top of an open rtl_tcp connection.
type device struct {
	addr string
	info sdr.Info
	log  *slog.Logger

	mu     sync.Mutex
	conn   net.Conn
	closed bool
	// sampleRate is the last value passed to SetSampleRate, exposed
	// via [sdr.SampleRateGetter]. The librtlsdr rtl_tcp wire
	// protocol has no reply field for SET_SAMPLE_RATE, so the rate
	// the remote chip actually landed on isn't observable from this
	// side — returning the requested value still lets the daemon
	// honour a per-tuner rate override.
	sampleRate uint32
}

func (d *device) Info() sdr.Info { return d.info }

func (d *device) SetCenterFreq(hz uint32) error { return d.sendCmd(cmdSetFreq, hz) }
func (d *device) SetSampleRate(hz uint32) error {
	if err := d.sendCmd(cmdSetSampleRate, hz); err != nil {
		return err
	}
	d.mu.Lock()
	d.sampleRate = hz
	d.mu.Unlock()
	return nil
}

// SampleRate returns the rate last passed to SetSampleRate.
// Implements [sdr.SampleRateGetter].
func (d *device) SampleRate() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sampleRate
}

func (d *device) SetGain(tenthDB int) error {
	if tenthDB < 0 {
		// Negative = AGC: enable gain mode auto (mode 0).
		return d.sendCmd(cmdSetGainMode, 0)
	}
	// Manual gain mode then the absolute gain value.
	if err := d.sendCmd(cmdSetGainMode, 1); err != nil {
		return err
	}
	return d.sendCmd(cmdSetGain, uint32(tenthDB))
}

func (d *device) SetPPM(ppm int) error {
	return d.sendCmd(cmdSetFreqCorrPPM, uint32(int32(ppm)))
}

func (d *device) SetBiasTee(enable bool) error {
	var v uint32
	if enable {
		v = 1
	}
	return d.sendCmd(cmdSetBiasTee, v)
}

// sendCmd writes one 5-byte command packet on the control side of
// the rtl_tcp socket. The same socket carries IQ in the other
// direction; rtl_tcp's protocol relies on the server reading commands
// out-of-band while writing IQ continuously, which net.Conn supports
// over the full-duplex TCP connection.
func (d *device) sendCmd(op uint8, param uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.conn == nil {
		return errors.New("rtltcp: device closed")
	}
	var pkt [5]byte
	pkt[0] = op
	binary.BigEndian.PutUint32(pkt[1:], param)
	_ = d.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := d.conn.Write(pkt[:]); err != nil {
		return fmt.Errorf("rtltcp: send cmd 0x%02x: %w", op, err)
	}
	_ = d.conn.SetWriteDeadline(time.Time{})
	return nil
}

func (d *device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// StreamIQ reads u8 IQ bytes from the rtl_tcp socket and emits
// complex64 chunks. Chunks are sized at chunkSamples samples (i.e.
// 2*chunkSamples bytes). Returns a channel that closes when the
// context cancels or the socket closes.
func (d *device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed || d.conn == nil {
		d.mu.Unlock()
		return nil, errors.New("rtltcp: device closed")
	}
	conn := d.conn
	d.mu.Unlock()

	const chunkSamples = 8192
	out := make(chan []complex64, 8)

	go func() {
		defer close(out)
		buf := make([]byte, chunkSamples*2)
		for {
			// Allow ctx to interrupt a stalled read.
			if dl, ok := ctx.Deadline(); ok {
				_ = conn.SetReadDeadline(dl)
			}
			// Use a long deadline so a wedged server can be torn down by
			// ctx cancel without blocking forever; ctx.Done is checked
			// after every read.
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := io.ReadFull(conn, buf)
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
					d.log.Debug("rtltcp: stream read", "addr", d.addr, "err", err)
				}
				if n == 0 {
					return
				}
				// Fall through to emit the partial chunk so subscribers
				// see what arrived before the EOF.
			}
			if n > 0 {
				samples := convertU8(buf[:n])
				select {
				case <-ctx.Done():
					return
				case out <- samples:
				}
			}
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return out, nil
}

// convertU8 maps interleaved u8 I,Q,I,Q bytes to complex64 in [-1, 1].
// Mirrors the rtlsdr/purego driver's convertU8IQ so frames coming off
// a remote rtl_tcp look identical to frames from a local USB dongle.
func convertU8(buf []byte) []complex64 {
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := float32(buf[2*i]) - 127.5
		qv := float32(buf[2*i+1]) - 127.5
		out[i] = complex(iv/127.5, qv/127.5)
	}
	return out
}

func serialFor(spec Spec, idx int) string {
	if spec.Serial != "" {
		return spec.Serial
	}
	return fmt.Sprintf("rtltcp-%s-%02d", sanitizeAddr(spec.Addr), idx)
}

func sanitizeAddr(addr string) string {
	out := make([]byte, 0, len(addr))
	for i := 0; i < len(addr); i++ {
		c := addr[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		case c == '-' || c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// tunerName decodes the 4-byte tuner-type field from the rtl_tcp
// header into the librtlsdr-style label.
func tunerName(code uint32) string {
	switch code {
	case 0:
		return "unknown"
	case 1:
		return "E4000"
	case 2:
		return "FC0012"
	case 3:
		return "FC0013"
	case 4:
		return "FC2580"
	case 5:
		return "R820T"
	case 6:
		return "R828D"
	default:
		return fmt.Sprintf("tuner-%d", code)
	}
}

// standardGainLadder returns the R820T2 gain ladder librtlsdr exposes
// over rtl_tcp. Servers running older librtlsdr or non-R820T2 tuners
// silently quantize gain values to whatever their hardware supports;
// the wire protocol carries the raw tenths-of-dB value.
func standardGainLadder() []int {
	return []int{0, 9, 14, 27, 37, 77, 87, 125, 144, 157, 166, 197, 207, 229, 254, 280, 297, 328, 338, 364, 372, 386, 402, 421, 434, 439, 445, 480, 496}
}
