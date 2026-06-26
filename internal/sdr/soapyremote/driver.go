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
//     stream.go), plus a second "status" socket the server requires alongside
//     it. This driver implements the TCP stream transport, which is in-order
//     and needs no UDP flow-control; the operator selects it with
//     `stream_protocol: tcp` (the default). UDP streaming is a future
//     addition (issue #536 phase 2).
//
// The wire format was reverse-engineered from SoapyRemote@master and the RPC,
// datagram framing, and TCP stream setup choreography are byte-matched to the
// source (client/Streaming.cpp + server/ClientHandler.cpp). It is exercised by
// a fake server in the tests; validate against live hardware before release.
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
	"math"
	"net"
	"strconv"
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

// streamSetupTimeout bounds the per-frame reads during TCP stream setup. It is
// deliberately longer than the per-call RPC timeout because a cold high-end
// device (e.g. a USRP X310) spends several seconds compiling its RFNoC graph
// inside the server's setupStream before it replies (issue #542).
const streamSetupTimeout = 30 * time.Second

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
	// StreamMTU sets the stream endpoint MTU in bytes, sent to the server as
	// the "remote:mtu" setupStream arg and used to size the client's
	// flow-control window. Zero (or <=0) uses SoapyRemote's default (1500).
	StreamMTU int
	// StreamWindow sets the stream flow-control window in bytes, sent to the
	// server as the "remote:window" setupStream arg and used as the client's
	// in-flight credit ceiling (advertised as window/StreamMTU sequences).
	// Zero (or <=0) uses the client default (streamWindowBytes, 8 MiB).
	StreamWindow int
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

	// Resolve the effective stream MTU and derive the flow-control window the
	// client advertises to the server. Defaulting to streamMTU keeps the
	// wire frame and advertised window byte-identical when no MTU is set.
	mtu := spec.StreamMTU
	if mtu <= 0 {
		mtu = streamMTU
	}
	window := spec.StreamWindow
	if window <= 0 {
		window = streamWindowBytes
	}
	windowSeqs := uint32(window / mtu)

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("soapyremote: dial %s: %w", addr, err)
	}
	dev := &device{
		addr:       addr,
		format:     format,
		proto:      proto,
		timeout:    timeout,
		conn:       conn,
		log:        d.log,
		mtu:        mtu,
		window:     window,
		windowSeqs: windowSeqs,
		ackTrigger: windowSeqs / streamNumBuffs,
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

	// mtu is the effective stream endpoint MTU (defaults to streamMTU when
	// unset). window is the effective flow-control window in bytes (defaults
	// to streamWindowBytes). windowSeqs is the in-flight credit advertised to
	// the server in each flow-control ACK (window/mtu); ackTrigger is how many
	// received datagrams elapse between gratuitous ACKs (windowSeqs/numBuffs).
	mtu        int
	window     int
	windowSeqs uint32
	ackTrigger uint32

	mu         sync.Mutex
	conn       net.Conn // RPC control socket
	dataConn   net.Conn // stream data socket (set in StreamIQ)
	statusConn net.Conn // stream status socket (the server requires it; we drain it)
	streamID   int32
	closed     bool
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

// ActualSampleRate reports the rate the remote device is actually delivering,
// which may differ from the value passed to SetSampleRate. A USRP (and other
// SoapySDR radios) can only run at integer decimations of its master clock, so
// UHD coerces a non-divisor request to the nearest achievable rate — e.g. a
// B210 left at its default master clock can't hit 6.144 MS/s exactly. The
// daemon's effectiveStreamRate() probes this optional extension and builds every
// per-channel DDC / symbol clock from the delivered rate, so the symbol clock
// stays aligned on coerced rates instead of drifting (issue #402, #550). It
// queries getSampleRate over the RPC control socket and rounds to whole Hz; a
// zero/error return makes effectiveStreamRate fall back to the requested rate.
func (d *device) ActualSampleRate() (uint32, error) {
	u, err := d.rpc(func(p *packer) {
		p.call(callGetSampleRate)
		p.char(dirRX)
		p.i32(0)
	})
	if err != nil {
		return 0, err
	}
	if err := u.checkException(); err != nil {
		return 0, err
	}
	rate, err := u.f64()
	if err != nil {
		return 0, err
	}
	if rate <= 0 {
		return 0, nil
	}
	return uint32(math.Round(rate)), nil
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
	// Manual gain: best-effort disable AGC, then set the overall gain in dB.
	// Disabling AGC maps to setGainMode(false); on front-ends with no AGC at
	// all (e.g. a USRP TwinRX) the server rejects it with "set_rx_agc() is not
	// supported on this radio". That must not abort the manual setGain that
	// follows — setGain is the call that actually applies the configured gain
	// (issue #542).
	_ = d.rpcBestEffort("disable agc", func(p *packer) {
		p.call(callSetGainMode)
		p.char(dirRX)
		p.i32(0)
		p.boolean(false)
	})
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
	streamID, dataConn, statusConn, err := d.setupStreamTCP()
	if err != nil {
		return nil, err
	}

	// Prime the server's sender with an initial flow-control ACK. The server
	// blocks in waitSend() until it receives one and would otherwise never
	// stream a sample (see encodeStreamACK / issue #542 follow-up). Sent before
	// ACTIVATE, mirroring the upstream receiver constructor.
	if err := d.sendStreamACK(dataConn, 0); err != nil {
		d.clearStreamConns()
		return nil, fmt.Errorf("soapyremote: initial stream ack: %w", err)
	}

	// ACTIVATE_STREAM (streamId, flags=0, timeNs=0, numElems=0).
	if err := d.rpcVoid(func(p *packer) {
		p.call(callActivateStream)
		p.i32(streamID)
		p.i32(0)
		p.i64(0)
		p.i32(0)
	}); err != nil {
		d.clearStreamConns()
		return nil, err
	}

	go d.drainStatus(statusConn)
	out := make(chan []complex64, 8)
	go d.streamLoop(ctx, dataConn, streamID, out)
	return out, nil
}

// setupStreamTCP performs SoapyRemote's two-phase TCP stream setup. The wire
// choreography is byte-matched to upstream (client/Streaming.cpp +
// server/ClientHandler.cpp):
//
//  1. send SETUP_STREAM;
//  2. read reply #1 — the server's bound data port (a single string);
//  3. dial TWO sockets to that port, the data socket then the status socket:
//     the server does listen(2) and blocks accepting both, in that order;
//  4. read reply #2 — the int stream id (plus a repeated port string we
//     discard).
//
// The whole exchange holds d.mu so no other RPC interleaves on the control
// socket between the two reply frames. On success the data/status sockets and
// stream id are stored on the device for teardown.
func (d *device) setupStreamTCP() (streamID int32, dataConn, statusConn net.Conn, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.conn == nil {
		return 0, nil, nil, errClosed
	}

	// (1) SETUP_STREAM. clientBindPort/statusBindPort are unused in TCP mode
	// (the client dials the server), so "0" is sent for both.
	p := newPacker()
	p.call(callSetupStream)
	p.char(dirRX)
	p.str(d.format.soapyName())
	p.sizeList([]int{0})
	// Stream args. remote:mtu / remote:window are only sent when configured to
	// a non-default value, keeping the default setup frame byte-identical to
	// before.
	streamArgs := map[string]string{"remote:prot": "tcp"}
	if d.mtu != streamMTU {
		streamArgs["remote:mtu"] = strconv.Itoa(d.mtu)
	}
	if d.window != streamWindowBytes {
		streamArgs["remote:window"] = strconv.Itoa(d.window)
	}
	p.kwargs(streamArgs)
	p.str("0")
	p.str("0")
	if err := p.writeTo(d.conn, streamSetupTimeout); err != nil {
		return 0, nil, nil, err
	}

	// (2) Reply #1: the server's bound data port.
	u, err := readResponse(d.conn, streamSetupTimeout)
	if err != nil {
		return 0, nil, nil, err
	}
	if err := u.checkException(); err != nil {
		return 0, nil, nil, err
	}
	serverPort, err := u.str()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("soapyremote: setup stream port: %w", err)
	}

	host, _, err := net.SplitHostPort(d.addr)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("soapyremote: split addr %q: %w", d.addr, err)
	}
	dataAddr := net.JoinHostPort(host, serverPort)

	// (3) Dial the data socket then the status socket. The server's two accepts
	// are ordered: first connection is the stream, second is the status channel.
	dataConn, err = net.DialTimeout("tcp", dataAddr, d.timeout)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("soapyremote: dial stream %s: %w", dataAddr, err)
	}
	statusConn, err = net.DialTimeout("tcp", dataAddr, d.timeout)
	if err != nil {
		dataConn.Close()
		return 0, nil, nil, fmt.Errorf("soapyremote: dial status %s: %w", dataAddr, err)
	}

	// (4) Reply #2: the int stream id (and a repeated port string we ignore).
	u2, err := readResponse(d.conn, streamSetupTimeout)
	if err != nil {
		dataConn.Close()
		statusConn.Close()
		return 0, nil, nil, err
	}
	if err := u2.checkException(); err != nil {
		dataConn.Close()
		statusConn.Close()
		return 0, nil, nil, err
	}
	streamID, err = u2.i32()
	if err != nil {
		dataConn.Close()
		statusConn.Close()
		return 0, nil, nil, fmt.Errorf("soapyremote: setup stream id: %w", err)
	}

	if d.closed {
		dataConn.Close()
		statusConn.Close()
		return 0, nil, nil, errClosed
	}
	d.dataConn = dataConn
	d.statusConn = statusConn
	d.streamID = streamID
	return streamID, dataConn, statusConn, nil
}

// sendStreamACK writes a flow-control ACK for seq to the stream/data socket.
// SoapyRemote requires these or the server never streams (see encodeStreamACK).
func (d *device) sendStreamACK(conn net.Conn, seq uint32) error {
	_ = conn.SetWriteDeadline(time.Now().Add(d.timeout))
	_, err := conn.Write(encodeStreamACK(seq, d.windowSeqs))
	return err
}

// drainStatus reads and discards the stream's status socket. SoapyRemote's
// server status thread may emit messages over it; leaving it unread can
// back-pressure the server. It returns when the socket is closed (on teardown).
func (d *device) drainStatus(statusConn net.Conn) {
	buf := make([]byte, 256)
	for {
		if _, err := statusConn.Read(buf); err != nil {
			return
		}
	}
}

// clearStreamConns closes and forgets the data/status sockets. Used when
// activation fails after setup; teardownStream handles the normal path.
func (d *device) clearStreamConns() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dataConn != nil {
		d.dataConn.Close()
		d.dataConn = nil
	}
	if d.statusConn != nil {
		d.statusConn.Close()
		d.statusConn = nil
	}
}

// soapyStatusOverflow is SoapySDR's SOAPY_SDR_OVERFLOW code, carried in a stream
// datagram's elems field when the device dropped samples because the host wasn't
// draining the stream fast enough. The other negative codes (timeout, stream
// error, corruption…) are not the operator-actionable "host too slow" signal, so
// they stay at DEBUG.
const soapyStatusOverflow = -4

// overrunWarnInterval throttles the operator-facing overrun WARN so a sustained
// overrun storm logs a periodic summary with counts instead of flooding one line
// per dropped datagram. Matches the 5 s cadence the wideband diagnostics use.
const overrunWarnInterval = 5 * time.Second

// overrunThrottle surfaces SDR sample loss to the operator at WARN, rate-limited.
// It folds two sources into one summary: device-side SOAPY_SDR_OVERFLOW datagrams
// (the radio dropped samples) and host-side local drops (streamLoop shed a chunk
// because the DSP consumer fell behind, see sendOrDrop). Both mean the same thing
// operationally — the host can't keep up with the configured rate — and both used
// to be invisible (overflow logged only at DEBUG; local back-pressure stalled the
// reader instead). It is not safe for concurrent use; streamLoop owns one.
type overrunThrottle struct {
	log      *slog.Logger
	addr     string
	interval time.Duration
	now      func() time.Time

	lastWarn     time.Time
	devOverflows uint64 // SOAPY_SDR_OVERFLOW datagrams since the last summary
	hostDrops    uint64 // chunks dropped locally since the last summary
}

func newOverrunThrottle(log *slog.Logger, addr string) *overrunThrottle {
	return &overrunThrottle{log: log, addr: addr, interval: overrunWarnInterval, now: time.Now}
}

func (t *overrunThrottle) deviceOverflow() { t.devOverflows++; t.maybeWarn(false) }
func (t *overrunThrottle) hostDrop()       { t.hostDrops++; t.maybeWarn(false) }

// maybeWarn emits the summary WARN when the throttle interval has elapsed (or
// immediately on the first event), then clears the running counts. force ignores
// the interval — used on teardown to flush a trailing batch that would otherwise
// never be reported.
func (t *overrunThrottle) maybeWarn(force bool) {
	if t.devOverflows == 0 && t.hostDrops == 0 {
		return
	}
	now := t.now()
	if !force && !t.lastWarn.IsZero() && now.Sub(t.lastWarn) < t.interval {
		return
	}
	t.log.Warn("soapyremote: SDR overruns — the host can't keep up with the configured sample rate, so samples are being dropped and decoded audio will glitch. Lower sdr.sample_rate or reduce the channel/tap count.",
		"addr", t.addr,
		"device_overflows", t.devOverflows,
		"host_drops", t.hostDrops,
	)
	t.lastWarn = now
	t.devOverflows = 0
	t.hostDrops = 0
}

// sendOrDrop hands samples to the DSP consumer without ever blocking the read
// loop. A blocked send stalls socket reads and flow-control ACKs, which makes the
// radio overflow and drop samples in a way that shreds every channel's framing at
// once. When the bounded out-channel is full instead, we drop the oldest queued
// chunk (keeping latency low and the data freshest) and count it — one clean local
// glitch rather than a device-side overrun storm. Returns false only on ctx cancel.
func (d *device) sendOrDrop(ctx context.Context, out chan []complex64, samples []complex64, t *overrunThrottle) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case out <- samples:
			return true
		default:
		}
		// Full: shed the oldest buffered chunk to make room. streamLoop is the
		// sole producer, so one drop frees a slot and the next send succeeds.
		select {
		case <-out:
			t.hostDrop()
		default:
		}
	}
}

func (d *device) streamLoop(ctx context.Context, dataConn net.Conn, streamID int32, out chan []complex64) {
	defer close(out)
	defer d.teardownStream(streamID)

	throttle := newOverrunThrottle(d.log, d.addr)
	defer throttle.maybeWarn(true) // flush any trailing counts on teardown

	hdr := make([]byte, streamHeaderSize)
	// Flow-control state: lastRecv tracks the next sequence we expect; lastAck
	// is the sequence carried by our most recent ACK. The initial ACK (seq 0)
	// was already sent in StreamIQ. uint32 wrap arithmetic matches upstream.
	var lastRecv, lastAck uint32
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
		// Flow control: advance the acked sequence and send a gratuitous ACK
		// every d.ackTrigger datagrams so the server keeps streaming. Done
		// for every datagram (including status codes), matching acquireRecv.
		lastRecv = h.sequence + 1
		if lastRecv-lastAck >= d.ackTrigger {
			if err := d.sendStreamACK(dataConn, lastRecv); err != nil {
				d.log.Debug("soapyremote: stream ack", "addr", d.addr, "err", err)
				return
			}
			lastAck = lastRecv
		}
		if h.elems < 0 {
			// Negative elems is a SoapySDR status/error code, not samples.
			// OVERFLOW means the device dropped samples because we read too
			// slowly — surface it to the operator (rate-limited); other codes
			// stay at DEBUG.
			if h.elems == soapyStatusOverflow {
				throttle.deviceOverflow()
			} else {
				d.log.Debug("soapyremote: stream status code", "addr", d.addr, "code", h.elems)
			}
			continue
		}
		if payloadLen == 0 {
			continue
		}
		samples := d.format.convert(payload)
		if len(samples) == 0 {
			continue
		}
		if !d.sendOrDrop(ctx, out, samples, throttle) {
			return
		}
	}
}

// teardownStream best-effort deactivates and closes the remote stream and the
// local data/status sockets. Errors are ignored — the connection may already be
// gone.
func (d *device) teardownStream(streamID int32) {
	_ = d.rpcVoid(func(p *packer) {
		p.call(callDeactivateStream)
		p.i32(streamID)
		p.i32(0)
		p.i64(0)
	})
	_ = d.rpcVoid(func(p *packer) {
		p.call(callCloseStream)
		p.i32(streamID)
	})
	d.clearStreamConns()
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
	if d.statusConn != nil {
		d.statusConn.Close()
		d.statusConn = nil
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
