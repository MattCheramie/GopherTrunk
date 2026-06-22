package ka9qradio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// DriverName is the sdr.Driver name registered with the pool.
const DriverName = "ka9qradio"

// DefaultConnectTimeout caps mDNS resolution and the status poll. radiod
// usually lives on the same LAN; a fast failure beats a hang on a misconfigured
// address.
const DefaultConnectTimeout = 3 * time.Second

var errClosed = errors.New("ka9qradio: device closed")

// Spec names one radiod channel to expose as a virtual tuner.
type Spec struct {
	// Addr is the status+command multicast group: an mDNS name like
	// "hf.local" / "hf.local:5006", or a literal "239.1.2.3:5006". A missing
	// port defaults to 5006. Required.
	Addr string
	// SSRC selects the channel within the radiod instance. Required.
	SSRC uint32
	// Serial is the virtual device serial the pool reports. Empty generates
	// one from Addr+SSRC so multi-channel configs stay unique.
	Serial string
	// Role hints the pool's role assignment: "control" | "voice" | "auto".
	Role string
	// DataAddr optionally pins the IQ multicast group ("239.4.5.6:5004"),
	// skipping discovery of OUTPUT_DATA_DEST_SOCKET from the status poll. A
	// missing port defaults to 5004.
	DataAddr string
	// SampleRate optionally pins the channel's IQ rate (Hz), skipping
	// OUTPUT_SAMPRATE discovery. Required if Encoding is also pinned and no
	// poll is wanted.
	SampleRate uint32
	// Encoding optionally pins the wire sample encoding ("s16be"|"s16le"|
	// "f32le"|"f32be"), skipping OUTPUT_ENCODING discovery.
	Encoding string
	// Channels optionally pins the output channel count (2 for raw IQ).
	Channels int
	// ConnectTimeout overrides DefaultConnectTimeout when non-zero.
	ConnectTimeout time.Duration
}

// Driver implements sdr.Driver over a set of radiod channels.
type Driver struct {
	specs []Spec
	log   *slog.Logger
}

// New builds a Driver over the given channels.
func New(specs []Spec, log *slog.Logger) *Driver {
	if log == nil {
		log = slog.Default()
	}
	return &Driver{specs: specs, log: log}
}

// Name implements sdr.Driver.
func (d *Driver) Name() string { return DriverName }

// Enumerate returns one Info per configured channel without probing — a radiod
// that's momentarily down stays in the pool and surfaces its error at Open,
// matching the rtltcp / soapyremote drivers.
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
			Product:   "ka9q-radio",
			TunerName: "ka9q-radio",
		})
	}
	return out, nil
}

// Open resolves the status group (mDNS for .local names), learns the channel's
// data socket / sample rate / encoding via a status poll (unless pinned in the
// Spec), joins the IQ multicast group, and returns a streaming Device.
func (d *Driver) Open(idx int) (sdr.Device, error) {
	if idx < 0 || idx >= len(d.specs) {
		return nil, fmt.Errorf("ka9qradio: index %d out of range", idx)
	}
	spec := d.specs[idx]
	if spec.Addr == "" {
		return nil, errors.New("ka9qradio: spec missing Addr")
	}
	if spec.SSRC == 0 {
		return nil, errors.New("ka9qradio: spec missing SSRC")
	}
	timeout := spec.ConnectTimeout
	if timeout <= 0 {
		timeout = DefaultConnectTimeout
	}

	statusGroup, err := resolveAddr(spec.Addr, defaultStatusPort, timeout)
	if err != nil {
		return nil, err
	}

	// Determine the channel parameters: prefer the pinned Spec overrides, else
	// poll the status group for them.
	dataAddr, hasData, err := pinnedDataAddr(spec)
	if err != nil {
		return nil, err
	}
	enc, encPinned, err := pinnedEncoding(spec)
	if err != nil {
		return nil, err
	}
	sampleRate := spec.SampleRate
	channels := spec.Channels

	if !hasData || !encPinned || sampleRate == 0 {
		st, err := pollStatus(statusGroup, spec.SSRC, timeout)
		if err != nil {
			return nil, fmt.Errorf("ka9qradio: status poll %s ssrc %d: %w", spec.Addr, spec.SSRC, err)
		}
		if !hasData {
			if !st.hasDataDest {
				return nil, fmt.Errorf("ka9qradio: status for ssrc %d carried no data socket", spec.SSRC)
			}
			dataAddr, hasData = st.dataDest, true
		}
		if !encPinned {
			enc = st.encoding
		}
		if sampleRate == 0 {
			sampleRate = st.sampleRate
		}
		if channels == 0 {
			channels = st.channels
		}
	}

	if !isIQEncoding(enc) {
		return nil, fmt.Errorf("ka9qradio: channel encoding %s is not raw IQ (configure a linear/IQ channel)", enc)
	}
	if sampleRate == 0 {
		return nil, errors.New("ka9qradio: could not determine sample rate")
	}
	if channels != 0 && channels != 2 {
		d.log.Warn("ka9qradio: channel is not 2-channel IQ; treating payload as interleaved I,Q anyway",
			"ssrc", spec.SSRC, "channels", channels)
	}

	dataConn, err := joinMulticast(dataAddr)
	if err != nil {
		return nil, fmt.Errorf("ka9qradio: join data group %s: %w", dataAddr, err)
	}
	// A connected UDP socket for sending retune commands to the status group.
	cmdConn, err := net.DialUDP(udpNetwork(statusGroup), nil, udpAddr(statusGroup))
	if err != nil {
		dataConn.Close()
		return nil, fmt.Errorf("ka9qradio: command socket: %w", err)
	}

	info := sdr.Info{
		Driver:    DriverName,
		Index:     idx,
		Serial:    serialFor(spec, idx),
		Product:   "ka9q-radio",
		TunerName: "ka9q-radio",
	}
	d.log.Info("ka9qradio: connected",
		"status", statusGroup.String(),
		"data", dataAddr.String(),
		"ssrc", spec.SSRC,
		"samprate", sampleRate,
		"encoding", enc.String())

	return &device{
		info:       info,
		addr:       spec.Addr,
		ssrc:       spec.SSRC,
		sampleRate: sampleRate,
		enc:        enc,
		dataConn:   dataConn,
		cmdConn:    cmdConn,
		log:        d.log,
	}, nil
}

// device implements sdr.Device on top of a joined IQ multicast group plus a
// command socket aimed at the status group.
type device struct {
	info       sdr.Info
	addr       string
	ssrc       uint32
	sampleRate uint32
	enc        encoding
	log        *slog.Logger

	mu       sync.Mutex
	dataConn *net.UDPConn
	cmdConn  *net.UDPConn
	closed   bool
}

func (d *device) Info() sdr.Info { return d.info }

// ActualSampleRate reports the rate radiod actually streams (learned from the
// status poll or pinned in config). The daemon's effectiveStreamRate probes
// this so every per-channel DDC and symbol clock is built from the delivered
// rate, not the pool's requested rate (issue #402).
func (d *device) ActualSampleRate() (uint32, error) { return d.sampleRate, nil }

// SetSampleRate is a no-op: the channel's rate is fixed by radiod's
// configuration. The pool calls this at open and must not fail, and the real
// rate is reported via ActualSampleRate.
func (d *device) SetSampleRate(uint32) error { return nil }

// SetGain / SetPPM / SetBiasTee are front-end concerns owned by radiod, not the
// downstream consumer, so they no-op (matching soapy_remote's best-effort
// stance on knobs the backend doesn't expose).
func (d *device) SetGain(int) error     { return nil }
func (d *device) SetPPM(int) error      { return nil }
func (d *device) SetBiasTee(bool) error { return nil }

// SetCenterFreq retunes the radiod channel by sending a RADIO_FREQUENCY command
// to the status group. radiod ignores it if the channel is locked.
func (d *device) SetCenterFreq(hz uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.cmdConn == nil {
		return errClosed
	}
	pkt := buildSetFreq(d.ssrc, rand.Uint32(), float64(hz))
	_ = d.cmdConn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := d.cmdConn.Write(pkt); err != nil {
		return fmt.Errorf("ka9qradio: send retune: %w", err)
	}
	_ = d.cmdConn.SetWriteDeadline(time.Time{})
	return nil
}

func (d *device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	var err error
	if d.dataConn != nil {
		err = d.dataConn.Close()
	}
	if d.cmdConn != nil {
		_ = d.cmdConn.Close()
	}
	return err
}

// StreamIQ reads RTP datagrams off the IQ multicast group, decodes each
// payload to complex64, and emits one chunk per packet. The channel closes when
// the context cancels or the socket closes.
func (d *device) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	d.mu.Lock()
	if d.closed || d.dataConn == nil {
		d.mu.Unlock()
		return nil, errClosed
	}
	conn := d.dataConn
	enc := d.enc
	ssrc := d.ssrc
	d.mu.Unlock()

	out := make(chan []complex64, 16)
	go func() {
		defer close(out)
		// Guard the reader so a decode panic surfaces as a logged stream
		// close (→ ccdecoder retry) instead of crashing the daemon (issue #492).
		defer gtlog.Recover(d.log, "ka9qradio-stream", nil)
		buf := make([]byte, 65536)
		var rtpState struct {
			have bool
			seq  uint16
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					// No data in the window; loop to re-check ctx.
					continue
				}
				d.log.Debug("ka9qradio: stream read", "addr", d.addr, "err", err)
				return
			}
			hdr, off, err := parseRTP(buf[:n])
			if err != nil {
				d.log.Debug("ka9qradio: bad RTP packet", "err", err)
				continue
			}
			// radiod sends one SSRC per data group, but guard against a
			// crossed/misconfigured group by dropping foreign SSRCs.
			if hdr.ssrc != ssrc {
				continue
			}
			if rtpState.have {
				if gap := int16(hdr.seq - rtpState.seq - 1); gap != 0 {
					d.log.Debug("ka9qradio: RTP sequence gap", "expected", rtpState.seq+1, "got", hdr.seq)
				}
			}
			rtpState.have = true
			rtpState.seq = hdr.seq

			samples, err := decodeIQ(buf[off:n], enc)
			if err != nil {
				d.log.Debug("ka9qradio: decode", "err", err)
				continue
			}
			if len(samples) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case out <- samples:
			}
		}
	}()
	return out, nil
}

// --- helpers ---------------------------------------------------------------

func serialFor(spec Spec, idx int) string {
	if spec.Serial != "" {
		return spec.Serial
	}
	return fmt.Sprintf("ka9q-%s-%d", sanitizeAddr(spec.Addr), spec.SSRC)
}

func sanitizeAddr(addr string) string {
	out := make([]byte, 0, len(addr))
	for i := 0; i < len(addr); i++ {
		c := addr[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// resolveAddr turns a Spec address (mDNS name or literal, with optional port)
// into a netip.AddrPort, applying defaultPort when none is given.
func resolveAddr(addr string, defaultPort uint16, timeout time.Duration) (netip.AddrPort, error) {
	host, port, err := splitHostPort(addr, defaultPort)
	if err != nil {
		return netip.AddrPort{}, err
	}
	if ip, perr := netip.ParseAddr(host); perr == nil {
		return netip.AddrPortFrom(ip, port), nil
	}
	if strings.HasSuffix(strings.ToLower(host), ".local") {
		ip, merr := resolveMDNS(host, timeout)
		if merr != nil {
			return netip.AddrPort{}, merr
		}
		return netip.AddrPortFrom(ip, port), nil
	}
	// Fall back to the system resolver for non-.local hostnames.
	ips, lerr := net.LookupHost(host)
	if lerr != nil || len(ips) == 0 {
		return netip.AddrPort{}, fmt.Errorf("ka9qradio: resolve %q: %w", host, lerr)
	}
	ip, perr := netip.ParseAddr(ips[0])
	if perr != nil {
		return netip.AddrPort{}, fmt.Errorf("ka9qradio: resolve %q: %w", host, perr)
	}
	return netip.AddrPortFrom(ip, port), nil
}

// splitHostPort splits "host", "host:port", or "[v6]:port" into a host and
// port, defaulting the port when absent.
func splitHostPort(addr string, defaultPort uint16) (string, uint16, error) {
	if !strings.Contains(addr, ":") || (strings.Count(addr, ":") > 1 && !strings.Contains(addr, "]")) {
		// Bare host (incl. bare IPv6 with no brackets/port).
		return addr, defaultPort, nil
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("ka9qradio: bad addr %q: %w", addr, err)
	}
	p, err := net.LookupPort("udp", portStr)
	if err != nil {
		return "", 0, fmt.Errorf("ka9qradio: bad port in %q: %w", addr, err)
	}
	return host, uint16(p), nil
}

func pinnedDataAddr(spec Spec) (netip.AddrPort, bool, error) {
	if spec.DataAddr == "" {
		return netip.AddrPort{}, false, nil
	}
	ap, err := resolveAddr(spec.DataAddr, defaultDataPort, spec.ConnectTimeout)
	if err != nil {
		return netip.AddrPort{}, false, err
	}
	return ap, true, nil
}

func pinnedEncoding(spec Spec) (encoding, bool, error) {
	if spec.Encoding == "" {
		return encNone, false, nil
	}
	enc, ok := parseEncoding(spec.Encoding)
	if !ok {
		return encNone, false, fmt.Errorf("ka9qradio: unknown encoding %q", spec.Encoding)
	}
	return enc, true, nil
}

func udpNetwork(ap netip.AddrPort) string {
	if ap.Addr().Is4() || ap.Addr().Is4In6() {
		return "udp4"
	}
	return "udp6"
}

func udpAddr(ap netip.AddrPort) *net.UDPAddr {
	return &net.UDPAddr{IP: ap.Addr().AsSlice(), Port: int(ap.Port())}
}

// joinMulticast opens a UDP socket bound to the group's port and joins the
// multicast group on the default interface.
func joinMulticast(ap netip.AddrPort) (*net.UDPConn, error) {
	conn, err := net.ListenMulticastUDP(udpNetwork(ap), nil, udpAddr(ap))
	if err != nil {
		return nil, err
	}
	// A generous read buffer absorbs scheduling jitter at high IQ rates.
	_ = conn.SetReadBuffer(1 << 21) // 2 MiB
	return conn, nil
}

// pollStatus joins the status group, sends a poll for ssrc, and returns the
// first matching status reply. Retries the poll a few times within timeout
// since the first datagram can be lost on a freshly-joined group.
func pollStatus(group netip.AddrPort, ssrc uint32, timeout time.Duration) (channelStatus, error) {
	conn, err := joinMulticast(group)
	if err != nil {
		return channelStatus{}, fmt.Errorf("join status group: %w", err)
	}
	defer conn.Close()

	dst := udpAddr(group)
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 65536)
	nextPoll := time.Now()
	for time.Now().Before(deadline) {
		if !time.Now().Before(nextPoll) {
			if _, err := conn.WriteToUDP(buildPoll(ssrc, rand.Uint32()), dst); err != nil {
				return channelStatus{}, fmt.Errorf("send poll: %w", err)
			}
			nextPoll = time.Now().Add(250 * time.Millisecond)
		}
		_ = conn.SetReadDeadline(minTime(nextPoll, deadline))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return channelStatus{}, err
		}
		if n == 0 || buf[0] != pktStatus {
			continue // commands (incl. our own echoes) aren't status replies
		}
		st, perr := parseStatus(buf[:n])
		if perr != nil {
			continue
		}
		if st.hasSSRC && st.ssrc == ssrc {
			return st, nil
		}
	}
	return channelStatus{}, errors.New("timed out waiting for status reply")
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
