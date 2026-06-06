// Package soapyremote implements an sdr.Driver that talks to a remote
// SoapySDRServer (from pothosware/SoapyRemote) in pure Go, with no CGO and no
// SoapySDR C libraries. SoapySDRServer exposes any SoapySDR-supported radio —
// USRP, LimeSDR, bladeRF, HackRF, Airspy, RTL-SDR, SDRplay and more — over the
// network with a real control plane, so GopherTrunk can demodulate trunked
// systems from high-dynamic-range hardware that rtl_tcp's hardcoded 8-bit
// stream cannot carry (issue #536).
//
// Two channels are used, mirroring SoapyRemote itself:
//
//   - A TCP RPC control socket (default port 55132) carries device creation,
//     tuning, gain and stream setup as length-framed, type-tagged packets
//     (see rpc.go).
//   - A separate stream socket carries 24-byte-framed IQ datagrams (see
//     stream.go). This driver implements the TCP stream transport, which is
//     in-order and needs no UDP flow-control; the operator selects it with
//     `stream_protocol: tcp` (the default). UDP streaming is a future
//     addition (issue #536 phase 2).
//
// The wire format was reverse-engineered from SoapyRemote@master. The RPC and
// datagram framing are byte-verified against the source; the TCP stream
// connection choreography should be validated against a live SoapySDRServer
// before relying on it in production.
//
// Limitations:
//   - Single channel (channel 0), receive only.
//   - SetPPM / SetBiasTee are best-effort: SoapySDR has no universal call for
//     either, so they map to setFrequencyCorrection / writeSetting and silently
//     no-op when the underlying driver doesn't support them.
//   - Plaintext, like rtl_tcp. Use on trusted networks or through a tunnel.
package soapyremote

import (
	"context"
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
const DriverName = "soapyremote"

// DefaultServicePort is SoapyRemote's default RPC port (SOAPY_REMOTE_DEFAULT_SERVICE).
const DefaultServicePort = "55132"

// DefaultConnectTimeout caps RPC dials and per-call round-trips.
const DefaultConnectTimeout = 3 * time.Second

// maxTransfer bounds a single stream transfer so a corrupt length field can't
// trigger a huge allocation.
const maxTransfer = 1 << 22 // 4 MiB

var errClosed = errors.New("soapyremote: device closed")

// Spec names one SoapySDRServer endpoint to expose as a virtual tuner.
type Spec struct {
	// Addr is the server host:port, e.g. "192.168.1.60:55132". A bare host
	// gets DefaultServicePort appended. Required.
	Addr string
	// Serial is the virtual device serial the pool reports. Empty generates
	// one from Addr so multi-endpoint configs stay unique.
	Serial string
	// Role hints the pool's role assignment: "control" | "voice" | "auto".
	Role string
	// DeviceArgs are the SoapySDR device-selection kwargs passed to MAKE,
	// e.g. {"driver":"lime"} or {"serial":"..."}. Empty selects the server's
	// first/only device.
	DeviceArgs map[string]string
	// Format is the requested wire sample format: "CS16" (default) or "CF32".
	Format string
	// StreamProtocol selects the stream transport: "tcp" (default/only).
	StreamProtocol string
	// ConnectTimeout overrides DefaultConnectTimeout when non-zero.
	ConnectTimeout time.Duration
}

// Driver implements sdr.Driver over a set of SoapySDRServer endpoints.
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

// Enumerate returns one Info per configured endpoint without probing — a
// remote that's momentarily down stays in the pool and surfaces its error at
// Open, matching the rtltcp driver's behaviour.
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
			Product:   "SoapyRemote",
			TunerName: deviceArgKey(spec.DeviceArgs),
			Gains:     genericGainLadder(),
		})
	}
	return out, nil
}

// Open dials the SoapySDRServer at spec[idx], makes the device, and returns a
// Device ready for setters and StreamIQ.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("soapyremote: index %d out of range", idx)
	}
	spec := d.specs[idx]
	if spec.Addr == "" {
		return nil, errors.New("soapyremote: spec missing Addr")
	}
	format, err := parseFormat(spec.Format)
	if err != nil {
		return nil, err
	}
	proto := spec.StreamProtocol
	if proto == "" {
		proto = "tcp"
	}
	if proto != "tcp" {
		return nil, fmt.Errorf("soapyremote: stream_protocol %q not supported (only \"tcp\")", proto)
	}
	timeout := spec.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}
	addr := withDefaultPort(spec.Addr)

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("soapyremote: dial %s: %w", addr, err)
	}
	dev := &device{
		addr:    addr,
		format:  format,
		proto:   proto,
		timeout: timeout,
		conn:    conn,
		log:     d.log,
		info: sdr.Info{
			Driver:    DriverName,
			Index:     idx,
			Serial:    serialFor(spec, idx),
			Product:   "SoapyRemote",
			TunerName: deviceArgKey(spec.DeviceArgs),
			Gains:     genericGainLadder(),
		},
	}
	// Create the remote device.
	if err := dev.rpcVoid(func(p *packer) {
		p.call(callMake)
		p.kwargs(spec.DeviceArgs)
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("soapyremote: make device: %w", err)
	}
	// Best-effort: learn the native format for diagnostics.
	if native, ok := dev.nativeStreamFormat(); ok {
		dev.info.TunerName = native
	}
	d.log.Info("soapyremote: connected",
		"addr", addr,
		"format", format.soapyName(),
		"proto", proto)
	return dev, nil
}

// device implements sdr.Device over an open SoapySDRServer RPC connection.
type device struct {
	addr    string
	format  sampleFormat
	proto   string
	timeout time.Duration
	log     *slog.Logger
	info    sdr.Info

	mu       sync.Mutex
	conn     net.Conn // RPC control socket
	dataConn net.Conn // stream data socket (set in StreamIQ)
	streamID string
	closed   bool
}

func (d *device) Info() sdr.Info { return d.info }

// rpc serializes one RPC round-trip on the control socket.
func (d *device) rpc(build func(*packer)) (*unpacker, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.conn == nil {
		return nil, errClosed
	}
	p := newPacker()
	build(p)
	if err := p.writeTo(d.conn, d.timeout); err != nil {
		return nil, err
	}
	return readResponse(d.conn, d.timeout)
}

// rpcVoid issues a call whose only meaningful response is success/exception.
func (d *device) rpcVoid(build func(*packer)) error {
	u, err := d.rpc(build)
	if err != nil {
		return err
	}
	return u.checkException()
}

// rpcBestEffort issues a call and downgrades a remote exception to a debug log,
// returning nil — used for knobs not every SoapySDR driver implements.
func (d *device) rpcBestEffort(what string, build func(*packer)) error {
	if err := d.rpcVoid(build); err != nil {
		if errors.Is(err, errClosed) {
			return err
		}
		d.log.Debug("soapyremote: "+what+" not applied", "addr", d.addr, "err", err)
	}
	return nil
}

func (d *device) SetCenterFreq(hz uint32) error {
	return d.rpcVoid(func(p *packer) {
		p.call(callSetFrequency)
		p.char(dirRX)
		p.i32(0)
		p.f64(float64(hz))
		p.kwargs(nil)
	})
}

func (d *device) SetSampleRate(hz uint32) error {
	return d.rpcVoid(func(p *packer) {
		p.call(callSetSampleRate)
		p.char(dirRX)
		p.i32(0)
		p.f64(float64(hz))
	})
}

func (d *device) SetGain(tenthDB int) error {
	if tenthDB < 0 {
		// Automatic gain control.
		return d.rpcVoid(func(p *packer) {
			p.call(callSetGainMode)
			p.char(dirRX)
			p.i32(0)
			p.boolean(true)
		})
	}
	// Manual gain: disable AGC, then set the overall gain in dB.
	if err := d.rpcVoid(func(p *packer) {
		p.call(callSetGainMode)
		p.char(dirRX)
		p.i32(0)
		p.boolean(false)
	}); err != nil {
		return err
	}
	return d.rpcVoid(func(p *packer) {
		p.call(callSetGain)
		p.char(dirRX)
		p.i32(0)
		p.f64(float64(tenthDB) / 10.0) // GopherTrunk tenths-dB → SoapySDR dB
	})
}

func (d *device) SetPPM(ppm int) error {
	return d.rpcBestEffort("ppm", func(p *packer) {
		p.call(callSetFrequencyCorrection)
		p.char(dirRX)
		p.i32(0)
		p.f64(float64(ppm))
	})
}

func (d *device) SetBiasTee(enable bool) error {
	val := "false"
	if enable {
		val = "true"
	}
	return d.rpcBestEffort("bias_tee", func(p *packer) {
		p.call(callWriteSetting)
		p.str("biastee")
		p.str(val)
	})
}

// nativeStreamFormat asks the server for the device's native RX format. Used
// only for diagnostics; returns ok=false if the call fails.
func (d *device) nativeStreamFormat() (string, bool) {
	u, err := d.rpc(func(p *packer) {
		p.call(callGetNativeStreamFormat)
		p.char(dirRX)
		p.i32(0)
	})
	if err != nil || u.checkException() != nil {
		return "", false
	}
	name, err := u.str()
	if err != nil {
		return "", false
	}
	return name, true
}

// StreamIQ sets up and activates an RX stream, then emits complex64 chunks from
// the TCP stream socket. The channel closes when the context cancels or the
// socket closes.
func (d *device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if d.proto != "tcp" {
		return nil, fmt.Errorf("soapyremote: stream_protocol %q not supported", d.proto)
	}
	// SETUP_STREAM. clientBindPort/statusBindPort are unused in TCP mode
	// (the client dials the server), so "0" is sent for both.
	u, err := d.rpc(func(p *packer) {
		p.call(callSetupStream)
		p.char(dirRX)
		p.str(d.format.soapyName())
		p.sizeList([]int{0})
		p.kwargs(map[string]string{"remote:prot": "tcp"})
		p.str("0")
		p.str("0")
	})
	if err != nil {
		return nil, err
	}
	if err := u.checkException(); err != nil {
		return nil, err
	}
	streamID, err := u.str()
	if err != nil {
		return nil, fmt.Errorf("soapyremote: setup stream id: %w", err)
	}
	serverPort, err := u.str()
	if err != nil {
		return nil, fmt.Errorf("soapyremote: setup stream port: %w", err)
	}

	host, _, err := net.SplitHostPort(d.addr)
	if err != nil {
		return nil, fmt.Errorf("soapyremote: split addr %q: %w", d.addr, err)
	}
	dataAddr := net.JoinHostPort(host, serverPort)
	dataConn, err := net.DialTimeout("tcp", dataAddr, d.timeout)
	if err != nil {
		return nil, fmt.Errorf("soapyremote: dial stream %s: %w", dataAddr, err)
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		dataConn.Close()
		return nil, errClosed
	}
	d.dataConn = dataConn
	d.streamID = streamID
	d.mu.Unlock()

	// ACTIVATE_STREAM (flags=0, timeNs=0, numElems=0).
	if err := d.rpcVoid(func(p *packer) {
		p.call(callActivateStream)
		p.str(streamID)
		p.i32(0)
		p.i64(0)
		p.i32(0)
	}); err != nil {
		dataConn.Close()
		return nil, err
	}

	out := make(chan []complex64, 8)
	go d.streamLoop(ctx, dataConn, streamID, out)
	return out, nil
}

func (d *device) streamLoop(ctx context.Context, dataConn net.Conn, streamID string, out chan<- []complex64) {
	defer close(out)
	defer d.teardownStream(streamID)

	hdr := make([]byte, streamHeaderSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Bounded deadline so a wedged server is torn down on ctx cancel.
		_ = dataConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if _, err := io.ReadFull(dataConn, hdr); err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				d.log.Debug("soapyremote: stream header read", "addr", d.addr, "err", err)
			}
			return
		}
		h := decodeStreamHeader(hdr)
		if h.bytes < streamHeaderSize || h.bytes > maxTransfer {
			d.log.Debug("soapyremote: bad transfer size", "addr", d.addr, "bytes", h.bytes)
			return
		}
		payloadLen := int(h.bytes) - streamHeaderSize
		var payload []byte
		if payloadLen > 0 {
			payload = make([]byte, payloadLen)
			if _, err := io.ReadFull(dataConn, payload); err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
					d.log.Debug("soapyremote: stream payload read", "addr", d.addr, "err", err)
				}
				return
			}
		}
		if h.elems < 0 {
			// Negative elems is a SoapySDR status/error code, not samples.
			d.log.Debug("soapyremote: stream status code", "addr", d.addr, "code", h.elems)
			continue
		}
		if payloadLen == 0 {
			continue
		}
		samples := d.format.convert(payload)
		if len(samples) == 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case out <- samples:
		}
	}
}

// teardownStream best-effort deactivates and closes the remote stream and the
// local data socket. Errors are ignored — the connection may already be gone.
func (d *device) teardownStream(streamID string) {
	_ = d.rpcVoid(func(p *packer) {
		p.call(callDeactivateStream)
		p.str(streamID)
		p.i32(0)
		p.i64(0)
	})
	_ = d.rpcVoid(func(p *packer) {
		p.call(callCloseStream)
		p.str(streamID)
	})
	d.mu.Lock()
	if d.dataConn != nil {
		d.dataConn.Close()
		d.dataConn = nil
	}
	d.mu.Unlock()
}

func (d *device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.dataConn != nil {
		d.dataConn.Close()
		d.dataConn = nil
	}
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// withDefaultPort appends DefaultServicePort to a bare host.
func withDefaultPort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, DefaultServicePort)
}

func serialFor(spec Spec, idx int) string {
	if spec.Serial != "" {
		return spec.Serial
	}
	return fmt.Sprintf("soapy-%s-%02d", sanitizeAddr(spec.Addr), idx)
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

// deviceArgKey returns the SoapySDR driver key for display, defaulting to the
// driver name when no kwargs were given.
func deviceArgKey(args map[string]string) string {
	if d, ok := args["driver"]; ok && d != "" {
		return d
	}
	return "soapy"
}

// genericGainLadder returns a coarse 0..50 dB ladder (tenths of dB). SoapySDR
// gain is continuous; this is only a hint for UI/validation.
func genericGainLadder() []int {
	out := make([]int, 0, 11)
	for g := 0; g <= 500; g += 50 {
		out = append(out, g)
	}
	return out
}
