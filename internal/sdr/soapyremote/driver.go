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
//   - Receive only. One RX channel (channel 0) by default; `diversity: mrc`
//     opens channels 0 and 1 and combines them into one stream (see mrc.go).
//     Every per-channel setting (frequency, rate, gain, frequency correction)
//     is programmed on both — an unconfigured second receiver is a branch that
//     contributes nothing (issue #1062).
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
	"slices"
	"strconv"
	"strings"
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

// streamReadIdleTimeout is the backstop for a server that goes quiet without
// closing the socket. It is NOT the shutdown path — a watcher in streamLoop
// expires the deadline on ctx cancel — so it can stay generous.
const streamReadIdleTimeout = 30 * time.Second

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
	// Diversity selects a spatial-diversity combiner over a multi-channel RX
	// stream. "" / "none" (default) is the ordinary single-channel stream.
	// "mrc" opens RX channels 0 and 1 and phase-coherently combines them into
	// one stream (shared-LO front-ends only, e.g. USRP B210 / AD9361). See
	// mrc.go. EXPERIMENTAL — issue #1062.
	Diversity string
	// Antennas selects the RX antenna port per channel (SoapySDR setAntenna),
	// applied in channel order after MAKE: Antennas[0] to RX channel 0,
	// Antennas[1] to channel 1 (the diversity second receiver). Empty leaves the
	// device default. A comma-separated multi-antenna value cannot be expressed
	// in the flat DeviceArgs kwargs string, so per-channel antenna routing (e.g.
	// an X310's RX1/RX2 under MRC) goes here instead of args.
	Antennas []string
	// DiversityCapture is a path prefix under which the driver dumps the
	// PRE-COMBINE per-branch IQ once per stream (see branchcapture.go). Empty
	// disables it. Only meaningful with Diversity set.
	DiversityCapture string
	// DiversityCaptureSeconds bounds that dump; <=0 selects
	// defaultDiversityCaptureSeconds.
	DiversityCaptureSeconds int
	// DiversityCaptureFormat selects the branch container: "" / "cs16"
	// (headerless int16 pairs) or "flac" (lossless compressed twin; falls back
	// to cs16 above flacCaptureMaxRateHz).
	DiversityCaptureFormat string
	// VerboseDebug logs every control-channel RPC request and response —
	// decoded arguments plus a hex dump of the frame — at DEBUG. Off by
	// default; the trace is per endpoint so a multi-radio config can follow
	// one server. See rpcdebug.go.
	VerboseDebug bool
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
	divMode, err := parseDiversity(spec.Diversity)
	if err != nil {
		return nil, err
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
		diversity:  divMode,
		info: sdr.Info{
			Driver:    DriverName,
			Index:     idx,
			Serial:    serialFor(spec, idx),
			Product:   "SoapyRemote",
			TunerName: deviceArgKey(spec.DeviceArgs),
			Gains:     genericGainLadder(),
		},
	}
	if divMode.enabled() {
		// The calibration window is sized from stream time, so the combiner needs
		// the rate. It is not programmed yet at Open; StreamIQ re-sizes the window
		// once the delivered rate is known.
		dev.mrc = newMRCCombiner(format, divMode, 0)
		dev.mrc.log = d.log
		dev.capturePrefix = spec.DiversityCapture
		dev.captureSeconds = spec.DiversityCaptureSeconds
		dev.captureFormat = spec.DiversityCaptureFormat
	}
	dev.deviceArgs = formatDeviceArgs(spec.DeviceArgs)
	dev.antennas = append([]string(nil), spec.Antennas...)
	if spec.VerboseDebug {
		dev.tracer = newRPCTracer(d.log, addr)
		d.log.Info("soapyremote: verbose RPC debug enabled — every control-channel "+
			"frame is logged at DEBUG (needs log.level: debug)", "addr", addr)
	}
	// Create the remote device.
	if err := dev.rpcVoid(func(p *packer) {
		p.call(callMake)
		p.kwargs(spec.DeviceArgs)
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("soapyremote: make device: %w", err)
	}
	// Apply per-channel RX antennas (X310 RX1/RX2 under MRC, etc.) now that the
	// device exists and its RX channels are known. An explicitly configured
	// antenna that the remote rejects is a real misconfiguration, so this is
	// rpcVoid (loud), not best-effort.
	if err := dev.applyAntennas(spec.Antennas); err != nil {
		conn.Close()
		return nil, fmt.Errorf("soapyremote: set antenna: %w", err)
	}
	// Best-effort: learn the native format for diagnostics.
	if native, ok := dev.nativeStreamFormat(); ok {
		dev.info.TunerName = native
	}
	d.log.Info("soapyremote: connected",
		"addr", addr,
		"format", format.soapyName(),
		"proto", proto,
		"diversity", divMode.String())
	if divMode.enabled() {
		d.log.Info("soapyremote: MRC diversity enabled — RX0+RX1 phase-coherent combine (experimental)",
			"addr", addr, "mode", divMode.String())
	}
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

	// diversity selects phase-coherent MRC over RX0+RX1; mrc is then the
	// combiner that turns the 2-channel stream into one. Both are set once at
	// Open and read-only thereafter. mrc is nil in the ordinary single-channel
	// case. Issue #1062.
	diversity diversityMode
	mrc       *mrcCombiner

	// capturePrefix/captureSeconds/captureFormat configure the one-shot
	// pre-combine branch dump; empty prefix disables it. Set once at Open.
	capturePrefix  string
	captureSeconds int
	captureFormat  string

	// tracer logs every RPC frame when Spec.VerboseDebug is set; nil otherwise.
	// Set once at Open, read-only thereafter, and nil-safe at every call site.
	tracer *rpcTracer

	// deviceArgs / antennas are kept only so a diversity capture's sidecar can
	// record what the branches were: replaying one without knowing which
	// antenna ports and device it came from is guesswork.
	deviceArgs string
	antennas   []string

	mu         sync.Mutex
	centerHz   uint32   // last programmed centre frequency, for the capture sidecar
	gainTenth  int      // last programmed gain, likewise
	conn       net.Conn // RPC control socket
	dataConn   net.Conn // stream data socket (set in StreamIQ)
	statusConn net.Conn // stream status socket (the server requires it; we drain it)
	streamID   int32
	closed     bool
}

// rxChannelCount is the number of RX channels the stream requests: 1 normally,
// diversityChannels (2) under MRC diversity.
func (d *device) rxChannelCount() int {
	if d.diversity.enabled() {
		return diversityChannels
	}
	return 1
}

// perRXChannel issues an RPC once per active RX channel — channel 0 only in the
// single-channel default, channels 0 and 1 under MRC diversity so the shared-LO
// second receiver is configured identically to the reference. Stops at the first
// error.
func (d *device) perRXChannel(build func(p *packer, ch int32)) error {
	for ch := 0; ch < d.rxChannelCount(); ch++ {
		ch := int32(ch)
		if err := d.rpcVoid(func(p *packer) { build(p, ch) }); err != nil {
			return err
		}
	}
	return nil
}

// perSecondaryRXChannel runs fn for every diversity branch past the reference
// (nothing at all in the single-channel default). A failure on a secondary
// branch is reported but not fatal: the reference receiver still works, and
// downgrading one branch beats refusing to tune at all. It is logged at WARN
// rather than swallowed — an unconfigured second receiver is precisely the
// silent failure behind issue #1062.
func (d *device) perSecondaryRXChannel(what string, fn func(ch int32) error) {
	for ch := 1; ch < d.rxChannelCount(); ch++ {
		if err := fn(int32(ch)); err != nil {
			if errors.Is(err, errClosed) {
				return
			}
			d.log.Warn("soapyremote: diversity branch not fully configured — that receiver will contribute little or nothing to the combine",
				"addr", d.addr, "channel", ch, "setting", what, "err", err)
		}
	}
}

func (d *device) Info() sdr.Info { return d.info }

// rpc serializes one RPC round-trip on the control socket.
func (d *device) rpc(build func(*packer)) (*unpacker, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.conn == nil {
		return nil, errClosed
	}
	p := newTracedPacker(d.tracer)
	build(p)
	id := p.callID()
	if err := p.writeTo(d.conn, d.timeout); err != nil {
		return nil, err
	}
	return readResponseTraced(d.conn, d.timeout, d.tracer, p.seq, id)
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

// applyAntennas sets the RX antenna port for channel i to antennas[i] via
// SET_ANTENNA (SoapySDR setAntenna). Channels past len(antennas) keep the
// device default. Called once at Open, after MAKE.
//
// Each channel is checked against the device's own LIST_ANTENNAS before the set
// and read back with GET_ANTENNA after it. Port names are device-specific and
// do not transfer between radios — a B210 offers "TX/RX" and "RX2" while a
// TwinRX offers "RX1" and "RX2" — so a config moved between rigs names a port
// that does not exist, and the failure has to say which names do. The read-back
// exists because the log line used to assert what GopherTrunk had asked for
// rather than what the device did.
func (d *device) applyAntennas(antennas []string) error {
	for ch, name := range antennas {
		if name == "" {
			continue
		}
		ch := int32(ch)
		avail, err := d.listAntennas(ch)
		if err != nil {
			return fmt.Errorf("channel %d list antennas: %w", ch, err)
		}
		if len(avail) > 0 && !slices.Contains(avail, name) {
			return fmt.Errorf("channel %d antenna %q is not a port on this device (available: %s)",
				ch, name, strings.Join(avail, ", "))
		}
		if err := d.rpcVoid(func(p *packer) {
			p.call(callSetAntenna)
			p.char(dirRX)
			p.i32(ch)
			p.str(name)
		}); err != nil {
			return fmt.Errorf("channel %d antenna %q: %w", ch, name, err)
		}
		got, err := d.getAntenna(ch)
		if err != nil {
			return fmt.Errorf("channel %d antenna %q read back: %w", ch, name, err)
		}
		if got != name {
			return fmt.Errorf("channel %d antenna set to %q but device reports %q", ch, name, got)
		}
		d.log.Info("soapyremote: rx antenna set", "addr", d.addr, "channel", ch, "antenna", got)
	}
	return nil
}

// listAntennas returns the RX antenna port names the device advertises for a
// channel. A driver that reports none yields an empty list, which callers treat
// as "cannot validate" rather than "nothing is valid".
func (d *device) listAntennas(ch int32) ([]string, error) {
	u, err := d.rpc(func(p *packer) {
		p.call(callListAntennas)
		p.char(dirRX)
		p.i32(ch)
	})
	if err != nil {
		return nil, err
	}
	if err := u.checkException(); err != nil {
		return nil, err
	}
	return u.strList()
}

// getAntenna returns the RX antenna port a channel is currently on.
func (d *device) getAntenna(ch int32) (string, error) {
	u, err := d.rpc(func(p *packer) {
		p.call(callGetAntenna)
		p.char(dirRX)
		p.i32(ch)
	})
	if err != nil {
		return "", err
	}
	if err := u.checkException(); err != nil {
		return "", err
	}
	return u.str()
}

func (d *device) SetCenterFreq(hz uint32) error {
	// A retune to the frequency the device is already on needs no RPC and — more
	// importantly for diversity — no recalibration: the LO does not re-lock, so
	// the frozen RX0↔RX1 phase constant is still valid. The cc-hunt supervisor
	// re-issues the same frequency when it re-locks the same carrier; before
	// this guard each such call reset a perfectly good MRC calibration.
	d.mu.Lock()
	same := hz != 0 && d.centerHz == hz
	d.mu.Unlock()
	if same {
		return nil
	}
	if err := d.perRXChannel(func(p *packer, ch int32) {
		p.call(callSetFrequency)
		p.char(dirRX)
		p.i32(ch)
		p.f64(float64(hz))
		p.kwargs(nil)
	}); err != nil {
		return err
	}
	d.mu.Lock()
	d.centerHz = hz
	d.mu.Unlock()
	// A retune is a new LO lock, so the frozen RX0↔RX1 phase constant is stale.
	// Re-arm calibration; the next combined window re-estimates it.
	if d.mrc != nil {
		d.mrc.requestRecalibrate()
	}
	return nil
}

func (d *device) SetSampleRate(hz uint32) error {
	return d.perRXChannel(func(p *packer, ch int32) {
		p.call(callSetSampleRate)
		p.char(dirRX)
		p.i32(ch)
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

// SetGain programs the gain on every active RX channel. Under MRC diversity the
// second receiver must be gained too: a SoapySDR RX channel comes up at the
// driver's default (0 dB on a UHD device), so leaving RX1 alone left it tens of
// dB below RX0 — far enough down that the maximal-ratio weight |h|² made it
// contribute nothing, which reads on air as "RX1 is dark" (issue #1062). The
// combiner's complex-gain estimate absorbs a *small* branch imbalance; it cannot
// conjure signal out of a receiver that was never gained.
func (d *device) SetGain(tenthDB int) error {
	if err := d.applyGain(0, tenthDB); err != nil {
		return err
	}
	d.mu.Lock()
	d.gainTenth = tenthDB
	d.mu.Unlock()
	d.perSecondaryRXChannel("gain", func(ch int32) error { return d.applyGain(ch, tenthDB) })
	return nil
}

// applyGain programs one RX channel's gain: AGC when tenthDB is negative,
// otherwise manual gain in dB.
func (d *device) applyGain(ch int32, tenthDB int) error {
	if tenthDB < 0 {
		// Automatic gain control.
		return d.rpcVoid(func(p *packer) {
			p.call(callSetGainMode)
			p.char(dirRX)
			p.i32(ch)
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
		p.i32(ch)
		p.boolean(false)
	})
	return d.rpcVoid(func(p *packer) {
		p.call(callSetGain)
		p.char(dirRX)
		p.i32(ch)
		p.f64(float64(tenthDB) / 10.0) // GopherTrunk tenths-dB → SoapySDR dB
	})
}

// SetPPM applies the frequency correction to every active RX channel. Both
// branches share one LO, but SoapySDR's correction is a per-channel setting, so
// correcting only channel 0 would leave the branches tuned to different
// frequencies — a slow relative rotation the frozen phase constant cannot track.
func (d *device) SetPPM(ppm int) error {
	for ch := 0; ch < d.rxChannelCount(); ch++ {
		ch := int32(ch)
		if err := d.rpcBestEffort("ppm", func(p *packer) {
			p.call(callSetFrequencyCorrection)
			p.char(dirRX)
			p.i32(ch)
			p.f64(float64(ppm))
		}); err != nil {
			return err
		}
	}
	return nil
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

// streamBufferLatency is how much IQ the stream channel is sized to hold. The
// read loop never blocks on the DSP consumer (see sendOrDrop) — that property is
// what keeps the socket drained and flow-control ACKs flowing so the device does
// not overflow. The flip side is that this bounded channel is the *only* cushion
// for normal consumer jitter (GC pauses, goroutine scheduling, bursty network
// delivery). When it is too shallow, sub-millisecond jitter overflows it and
// sendOrDrop sheds a chunk — a silent IQ discontinuity that breaks every channel's
// symbol framing — even while the device itself keeps up (device_overflows=0). The
// pre-fix depth was a fixed 8 chunks (~a few ms at a multi-MS/s remote rate), which
// shed ~2% of chunks on ordinary jitter and shredded P25 decode. ~400 ms is
// comfortably above realistic jitter while bounding added latency and memory;
// sendOrDrop still sheds the oldest chunk once this fills, so genuine *sustained*
// over-capacity is handled and surfaced rather than stalling the reader.
const streamBufferLatency = 400 * time.Millisecond

const (
	// minStreamBufferChunks floors the computed depth so even a low-rate or
	// large-MTU stream keeps a healthy jitter cushion (well above the depth-8
	// regression).
	minStreamBufferChunks = 64
	// maxStreamBufferChunks ceilings the depth so a very high rate can't size a
	// pathologically large channel; at a few thousand samples per chunk this
	// caps the channel's backing memory in the low tens of MB.
	maxStreamBufferChunks = 2048
	// defaultStreamBufferChunks is used when the delivered rate is unknown (the
	// ActualSampleRate probe failed). Generous enough to absorb jitter at the
	// rates these remote wideband streams run.
	defaultStreamBufferChunks = 512
)

// streamBufferDepth sizes the stream channel to hold ~streamBufferLatency of IQ at
// the delivered rate, given how many complex samples arrive per datagram chunk.
// The result is clamped to [minStreamBufferChunks, maxStreamBufferChunks]; a
// zero/unknown rate or chunk size falls back to defaultStreamBufferChunks.
func streamBufferDepth(rateHz uint32, samplesPerChunk int) int {
	if rateHz == 0 || samplesPerChunk <= 0 {
		return defaultStreamBufferChunks
	}
	chunksPerSec := float64(rateHz) / float64(samplesPerChunk)
	depth := int(math.Ceil(streamBufferLatency.Seconds() * chunksPerSec))
	if depth < minStreamBufferChunks {
		return minStreamBufferChunks
	}
	if depth > maxStreamBufferChunks {
		return maxStreamBufferChunks
	}
	return depth
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

	if err := d.activateStream(streamID); err != nil {
		d.clearStreamConns()
		return nil, err
	}

	go d.drainStatus(statusConn)
	// Size the stream buffer to ride out consumer jitter without dropping IQ.
	// The read loop never blocks (sendOrDrop), so this bounded channel is the only
	// jitter cushion; a fixed depth of 8 (~a few ms at these rates) shed ~2% of
	// chunks on ordinary scheduling jitter and shredded decode even with
	// device_overflows=0. Size from the delivered rate (best-effort probe; 0 on
	// failure → default depth) and the samples-per-datagram the MTU/format imply.
	rate, _ := d.ActualSampleRate()
	// Per-channel samples: a multi-channel datagram splits the same MTU across
	// the branches, and the combiner emits one branch's worth downstream.
	samplesPerChunk := (d.mtu - streamHeaderSize) / (d.format.bytesPerSample() * d.rxChannelCount())
	out := make(chan []complex64, streamBufferDepth(rate, samplesPerChunk))
	// Re-arm the MRC calibration for this fresh stream so it re-estimates the
	// gain rather than reusing a stale one from a prior stream, and size its
	// estimation window from the rate now that one is known (the window is
	// specified in stream time, so it must not depend on the MTU).
	if d.mrc != nil {
		d.mrc.setSampleRate(float64(rate))
		d.mrc.requestRecalibrate()
		d.startBranchCapture(float64(rate))
	}
	go d.streamLoop(ctx, dataConn, streamID, out)
	return out, nil
}

// activateStream issues ACTIVATE_STREAM. A single-channel stream starts
// immediately (flags=0, timeNs=0). A multi-channel (MRC diversity) stream must
// instead start at a scheduled device timestamp: UHD rejects an immediate
// start on a multi-channel streamer ("Invalid recv stream command - stream now
// on multiple channels in a single streamer will fail to time align") because
// the two receive chains can only be sample-aligned by starting both at the
// same hardware time. So under diversity we read the remote hardware clock and
// schedule the start diversityActivateLead in the future (SOAPY_SDR_HAS_TIME).
// Remotes without a hardware clock — or that reject the timed form — fall back
// to the immediate start, preserving the single-channel behaviour.
func (d *device) activateStream(streamID int32) error {
	if d.rxChannelCount() > 1 {
		hw, err := d.getHardwareTime()
		if err == nil {
			err = d.rpcVoid(func(p *packer) {
				p.call(callActivateStream)
				p.i32(streamID)
				p.i32(soapyFlagHasTime)
				p.i64(hw + diversityActivateLead.Nanoseconds())
				p.i32(0)
			})
			if err == nil {
				return nil
			}
		}
		if errors.Is(err, errClosed) {
			return err
		}
		d.log.Debug("soapyremote: timed multi-channel stream start unavailable; falling back to immediate start",
			"addr", d.addr, "err", err)
	}
	return d.rpcVoid(func(p *packer) {
		p.call(callActivateStream)
		p.i32(streamID)
		p.i32(0)
		p.i64(0)
		p.i32(0)
	})
}

// getHardwareTime queries the remote device's hardware clock in nanoseconds
// (GET_HARDWARE_TIME with no clock qualifier). Not every SoapySDR driver keeps
// a hardware clock; callers treat an error as "no time source".
func (d *device) getHardwareTime() (int64, error) {
	u, err := d.rpc(func(p *packer) {
		p.call(callGetHardwareTime)
		p.str("")
	})
	if err != nil {
		return 0, err
	}
	if err := u.checkException(); err != nil {
		return 0, err
	}
	return u.i64()
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
	p := newTracedPacker(d.tracer)
	p.call(callSetupStream)
	p.char(dirRX)
	p.str(d.format.soapyName())
	// Channel list: [0] normally, [0,1] under MRC diversity so the server
	// streams both RX channels interleaved for phase-coherent combining.
	chans := []int{0}
	if d.diversity.enabled() {
		chans = []int{0, 1}
	}
	p.sizeList(chans)
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
	u, err := readResponseTraced(d.conn, streamSetupTimeout, d.tracer, p.seq, callSetupStream)
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
	u2, err := readResponseTraced(d.conn, streamSetupTimeout, d.tracer, p.seq, callSetupStream)
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
	health := newDiversityReporter(d.log, d.addr)

	// The ctx check below only runs BETWEEN reads, so a cancel that arrives
	// while the server is quiet would not be seen until the read deadline
	// expired. Expiring the deadline on cancel unblocks the in-flight read
	// immediately while leaving the deferred teardown to run normally —
	// closing the socket here would race that teardown.
	stopWatch := make(chan struct{})
	defer close(stopWatch)
	go func() {
		select {
		case <-ctx.Done():
			_ = dataConn.SetReadDeadline(time.Now())
		case <-stopWatch:
		}
	}()

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
		// Backstop for a server that stops sending without closing the socket;
		// ctx cancel is handled by the watcher above, not by this expiring.
		_ = dataConn.SetReadDeadline(time.Now().Add(streamReadIdleTimeout))
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
		var samples []complex64
		if d.mrc != nil {
			// MRC diversity: de-interleave RX0/RX1, phase-coherently combine
			// into one branch. Emits the reference branch until calibrated.
			// h.elems is the valid sample count per channel — the block stride
			// is derived from it (see deinterleave).
			samples = d.mrc.combine(payload, int(h.elems))
			health.observe(d.mrc)
		} else {
			samples = d.format.convert(payload)
		}
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
// formatDeviceArgs renders the MAKE kwargs back into the flat "k=v,k=v" form
// the operator wrote in config, so a capture sidecar records which device it
// came from rather than leaving the field blank.
func formatDeviceArgs(args map[string]string) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+args[k])
	}
	return strings.Join(parts, ",")
}

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
