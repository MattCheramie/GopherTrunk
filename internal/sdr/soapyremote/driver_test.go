package soapyremote

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeSoapyServer mimics a SoapySDRServer well enough to exercise the driver:
// it speaks the RPC framing, records every call (with decoded numeric args for
// the tuning calls), and on SETUP/ACTIVATE serves synthetic CS16 stream
// datagrams over a TCP data socket.
type fakeSoapyServer struct {
	t  *testing.T
	ln net.Listener

	mu    sync.Mutex
	calls []recordedCall
	// acks records the flow-control ACK datagrams received from the client on
	// the stream socket (issue #542 follow-up). The first is the gratuitous
	// initial ACK the server's waitSend() blocks on before streaming.
	acks []streamHeader
	// setupKwargs records the stream args the client sent in SETUP_STREAM
	// (e.g. "remote:prot", "remote:mtu"), captured on the last setup call.
	setupKwargs map[string]string
	// setupChannels records the RX channel list the client requested in
	// SETUP_STREAM ([0] normally, [0,1] under MRC diversity).
	setupChannels []int

	// samples streamed per datagram once activated.
	streamSamples []complex64

	// failGainMode makes SET_GAIN_MODE reply with a remote exception, mimicking
	// a UHD front-end with no AGC ("set_rx_agc() is not supported"). Issue #542.
	failGainMode bool

	// lastSetRateHz is the most recent rate the client programmed via
	// SET_SAMPLE_RATE. getSampleRateHz, when non-zero, overrides what
	// GET_SAMPLE_RATE reports — modelling a USRP that coerces a non-divisor
	// request to a different achievable rate (issue #550). When zero,
	// GET_SAMPLE_RATE echoes lastSetRateHz (an exact-divisor device).
	lastSetRateHz   float64
	getSampleRateHz float64
}

type recordedCall struct {
	id       int32
	freqHz   float64
	gainDB   float64
	gainAuto bool
}

func newFakeSoapyServer(t *testing.T) *fakeSoapyServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSoapyServer{t: t, ln: ln}
	go s.acceptLoop()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeSoapyServer) Addr() string { return s.ln.Addr().String() }

func (s *fakeSoapyServer) recorded() []recordedCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedCall, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *fakeSoapyServer) recordACK(h streamHeader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acks = append(s.acks, h)
}

func (s *fakeSoapyServer) recordedACKs() []streamHeader {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]streamHeader, len(s.acks))
	copy(out, s.acks)
	return out
}

func (s *fakeSoapyServer) recordedSetupKwargs() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.setupKwargs))
	for k, v := range s.setupKwargs {
		out[k] = v
	}
	return out
}

func (s *fakeSoapyServer) recordedSetupChannels() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int, len(s.setupChannels))
	copy(out, s.setupChannels)
	return out
}

// readSetupStreamKwargs parses a SETUP_STREAM request body (positioned just
// past the call id) and returns the requested RX channel list plus stream args.
// The layout mirrors the client's setupStreamTCP packer: char(dir), str(format),
// sizeList(channels), kwargs(streamArgs), str, str.
func readSetupStreamKwargs(u *unpacker) ([]int, map[string]string, error) {
	if _, err := u.char(); err != nil { // direction
		return nil, nil, err
	}
	if _, err := u.str(); err != nil { // format
		return nil, nil, err
	}
	if err := u.expect(tSizeList); err != nil { // channel list
		return nil, nil, err
	}
	n, err := u.i32()
	if err != nil {
		return nil, nil, err
	}
	channels := make([]int, 0, n)
	for i := int32(0); i < n; i++ {
		ch, err := u.i32()
		if err != nil {
			return nil, nil, err
		}
		channels = append(channels, int(ch))
	}
	if err := u.expect(tKwargs); err != nil { // stream args
		return nil, nil, err
	}
	kn, err := u.i32()
	if err != nil {
		return nil, nil, err
	}
	m := make(map[string]string, kn)
	for i := int32(0); i < kn; i++ {
		k, err := u.str()
		if err != nil {
			return nil, nil, err
		}
		v, err := u.str()
		if err != nil {
			return nil, nil, err
		}
		m[k] = v
	}
	return channels, m, nil
}

func (s *fakeSoapyServer) sawCall(id int32) bool {
	for _, c := range s.recorded() {
		if c.id == id {
			return true
		}
	}
	return false
}

func (s *fakeSoapyServer) acceptLoop() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	go s.handleRPC(conn)
}

func (s *fakeSoapyServer) handleRPC(conn net.Conn) {
	defer conn.Close()
	activate := make(chan struct{}, 1) // signal: start streaming
	for {
		u, err := readResponse(conn, 0)
		if err != nil {
			return
		}
		id, err := u.call()
		if err != nil {
			return
		}
		rec := recordedCall{id: id}
		var doActivate bool
		// Parse the request args and pick the reply, but DON'T send it yet.
		// The call is recorded BEFORE the response is written so a client call
		// that has already returned is always visible via recorded() — sending
		// first and recording afterwards raced the test's assertions.
		reply := func(p *packer) { p.raw8(tVoid) }
		twoPhaseSetup := false
		switch id {
		case callSetFrequency:
			_, _ = u.char()
			_, _ = u.i32()
			rec.freqHz, _ = u.f64()
		case callSetSampleRate:
			_, _ = u.char()
			_, _ = u.i32()
			rec.freqHz, _ = u.f64()
			s.mu.Lock()
			s.lastSetRateHz = rec.freqHz
			s.mu.Unlock()
		case callGetSampleRate:
			_, _ = u.char()
			_, _ = u.i32()
			s.mu.Lock()
			rate := s.getSampleRateHz
			if rate == 0 {
				rate = s.lastSetRateHz
			}
			s.mu.Unlock()
			reply = func(p *packer) { p.f64(rate) }
		case callSetGain:
			_, _ = u.char()
			_, _ = u.i32()
			rec.gainDB, _ = u.f64()
		case callSetGainMode:
			_, _ = u.char()
			_, _ = u.i32()
			rec.gainAuto, _ = u.boolean()
			if s.failGainMode {
				reply = func(p *packer) {
					p.raw8(tException)
					p.str("RuntimeError: NotImplementedError: set_rx_agc() is not supported on this radio!")
				}
			}
		case callGetNativeStreamFormat:
			reply = func(p *packer) {
				p.str("CS16")
				p.f64(1.0)
			}
		case callSetupStream:
			twoPhaseSetup = true
			if chans, kw, err := readSetupStreamKwargs(u); err == nil {
				s.mu.Lock()
				s.setupKwargs = kw
				s.setupChannels = chans
				s.mu.Unlock()
			}
		case callActivateStream:
			_, _ = u.i32() // streamId (int)
			doActivate = true
		default:
			// MAKE, DEACTIVATE, CLOSE, WRITE_SETTING, FREQ_CORRECTION, ...
		}

		// Record before responding (and before signalling activate) so the
		// client never observes a returned call that the server hasn't logged.
		s.mu.Lock()
		s.calls = append(s.calls, rec)
		s.mu.Unlock()

		if twoPhaseSetup {
			// Real SoapyRemote TCP setup is two-phase: reply #1 is the bound
			// data port, the server then accepts two client sockets (stream +
			// status), and reply #2 carries the int stream id. Issue #542.
			dataPort, bothConnected := s.startDataServer(activate)
			s.respond(conn, func(p *packer) { p.str(dataPort) }) // reply #1
			<-bothConnected
			s.respond(conn, func(p *packer) { // reply #2
				p.i32(0) // streamId (int)
				p.str(dataPort)
			})
		} else {
			s.respond(conn, reply)
		}
		if doActivate {
			select {
			case activate <- struct{}{}:
			default:
			}
		}
	}
}

func (s *fakeSoapyServer) respond(conn net.Conn, build func(*packer)) {
	p := newPacker()
	build(p)
	if err := p.writeTo(conn, 0); err != nil {
		s.t.Logf("fake respond: %v", err)
	}
}

// startDataServer binds a TCP data listener that accepts the stream socket then
// the status socket (matching the server's listen(2) / two-accept choreography),
// and once activated streams repeating CS16 datagrams of s.streamSamples on the
// stream socket while draining the status socket. Returns the bound port and a
// channel closed once both sockets have connected. Issue #542.
func (s *fakeSoapyServer) startDataServer(activate <-chan struct{}) (string, <-chan struct{}) {
	dataLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.t.Fatalf("data listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(dataLn.Addr().String())
	bothConnected := make(chan struct{})
	go func() {
		defer dataLn.Close()
		streamConn, err := dataLn.Accept()
		if err != nil {
			return
		}
		defer streamConn.Close()
		statusConn, err := dataLn.Accept()
		if err != nil {
			return
		}
		defer statusConn.Close()
		close(bothConnected)
		<-activate // wait until ACTIVATE_STREAM

		// Real SoapyRemote blocks in waitSend() until the receiver sends an
		// initial flow-control ACK; model that so the test fails if the client
		// never ACKs (issue #542 follow-up). Subsequent ACKs are drained in the
		// background so the client's writes never block.
		ackHdr := make([]byte, streamHeaderSize)
		_ = streamConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := io.ReadFull(streamConn, ackHdr); err != nil {
			return
		}
		_ = streamConn.SetReadDeadline(time.Time{})
		s.recordACK(decodeStreamHeader(ackHdr))
		go func() {
			buf := make([]byte, streamHeaderSize)
			for {
				if _, err := io.ReadFull(streamConn, buf); err != nil {
					return
				}
				s.recordACK(decodeStreamHeader(buf))
			}
		}()

		seq := uint32(0)
		for {
			payload := encodeCS16(s.streamSamples)
			hdr := encodeStreamHeader(streamHeader{
				bytes:    uint32(streamHeaderSize + len(payload)),
				sequence: seq,
				elems:    int32(len(s.streamSamples)),
			})
			if _, err := streamConn.Write(append(hdr, payload...)); err != nil {
				return
			}
			seq++
			time.Sleep(time.Millisecond)
		}
	}()
	return port, bothConnected
}

func encodeCS16(samples []complex64) []byte {
	buf := make([]byte, 0, len(samples)*4)
	for _, c := range samples {
		iv := int16(real(c) * 32768)
		qv := int16(imag(c) * 32768)
		var b [4]byte
		binary.LittleEndian.PutUint16(b[0:], uint16(iv))
		binary.LittleEndian.PutUint16(b[2:], uint16(qv))
		buf = append(buf, b[:]...)
	}
	return buf
}

func TestOpenAndSetters(t *testing.T) {
	srv := newFakeSoapyServer(t)
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, testLogger())

	infos, err := drv.Enumerate()
	if err != nil || len(infos) != 1 {
		t.Fatalf("Enumerate: infos=%d err=%v", len(infos), err)
	}

	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if !srv.sawCall(callMake) {
		t.Error("server did not see MAKE")
	}

	if err := dev.SetCenterFreq(851000000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if err := dev.SetSampleRate(2400000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if err := dev.SetGain(300); err != nil { // 30.0 dB
		t.Fatalf("SetGain: %v", err)
	}
	if err := dev.SetGain(-1); err != nil { // AGC
		t.Fatalf("SetGain(AGC): %v", err)
	}

	var sawFreq, sawRate, sawGainManual, sawGainAuto bool
	for _, c := range srv.recorded() {
		switch c.id {
		case callSetFrequency:
			if c.freqHz == 851000000 {
				sawFreq = true
			}
		case callSetSampleRate:
			if c.freqHz == 2400000 {
				sawRate = true
			}
		case callSetGain:
			if c.gainDB == 30.0 {
				sawGainManual = true
			}
		case callSetGainMode:
			if c.gainAuto {
				sawGainAuto = true
			}
		}
	}
	if !sawFreq {
		t.Error("SET_FREQUENCY with 851 MHz not recorded")
	}
	if !sawRate {
		t.Error("SET_SAMPLE_RATE with 2.4 MS/s not recorded")
	}
	if !sawGainManual {
		t.Error("SET_GAIN with 30.0 dB not recorded")
	}
	if !sawGainAuto {
		t.Error("SET_GAIN_MODE(auto=true) not recorded")
	}
}

// TestSetGainManualSurvivesAGCException covers issue #542: on a front-end with
// no AGC (e.g. a USRP TwinRX), disabling gain mode throws "set_rx_agc() is not
// supported". The manual setGain that follows must still run and succeed so the
// configured gain is actually applied — the AGC-disable is best-effort.
func TestSetGainManualSurvivesAGCException(t *testing.T) {
	srv := newFakeSoapyServer(t)
	srv.failGainMode = true
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, testLogger())

	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if err := dev.SetGain(750); err != nil { // 75.0 dB, like the issue's gain=750
		t.Fatalf("SetGain must not fail when AGC-disable is unsupported: %v", err)
	}

	var sawGainManual bool
	for _, c := range srv.recorded() {
		if c.id == callSetGain && c.gainDB == 75.0 {
			sawGainManual = true
		}
	}
	if !sawGainManual {
		t.Error("SET_GAIN with 75.0 dB not applied after SET_GAIN_MODE exception")
	}
}

// TestActualSampleRateReportsCoercedRate covers issue #550: a USRP can only
// deliver integer decimations of its master clock, so UHD coerces a non-divisor
// request to the nearest achievable rate. ActualSampleRate must read back that
// delivered rate (via GET_SAMPLE_RATE) so the daemon's effectiveStreamRate()
// builds the symbol clock from the truth, not the requested value.
func TestActualSampleRateReportsCoercedRate(t *testing.T) {
	srv := newFakeSoapyServer(t)
	srv.getSampleRateHz = 6_250_000 // device coerces the request to ÷32 of 200 MHz
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, testLogger())

	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if err := dev.SetSampleRate(6_144_000); err != nil { // a B210-style request
		t.Fatalf("SetSampleRate: %v", err)
	}

	ar, ok := dev.(interface{ ActualSampleRate() (uint32, error) })
	if !ok {
		t.Fatal("soapyremote device does not implement ActualSampleRate")
	}
	got, err := ar.ActualSampleRate()
	if err != nil {
		t.Fatalf("ActualSampleRate: %v", err)
	}
	if got != 6_250_000 {
		t.Errorf("ActualSampleRate = %d, want 6_250_000 (the coerced rate)", got)
	}
}

// TestActualSampleRateEchoesExactRate confirms the no-coercion path: when the
// requested rate is an exact divisor the server reports it back unchanged, so
// effectiveStreamRate stays quiet (requested == actual).
func TestActualSampleRateEchoesExactRate(t *testing.T) {
	srv := newFakeSoapyServer(t) // getSampleRateHz==0 → echo the last set rate
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, testLogger())

	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if err := dev.SetSampleRate(6_250_000); err != nil { // X310 ÷32, exact
		t.Fatalf("SetSampleRate: %v", err)
	}

	got, err := dev.(interface{ ActualSampleRate() (uint32, error) }).ActualSampleRate()
	if err != nil {
		t.Fatalf("ActualSampleRate: %v", err)
	}
	if got != 6_250_000 {
		t.Errorf("ActualSampleRate = %d, want 6_250_000", got)
	}
}

func TestStreamIQ(t *testing.T) {
	srv := newFakeSoapyServer(t)
	srv.streamSamples = []complex64{
		complex(0.5, -0.5),
		complex(0.25, 0.25),
		complex(-1, 0.75),
	}
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	select {
	case got, ok := <-ch:
		if !ok {
			t.Fatal("stream channel closed before any data")
		}
		if len(got) != len(srv.streamSamples) {
			t.Fatalf("chunk len = %d, want %d", len(got), len(srv.streamSamples))
		}
		for i, want := range srv.streamSamples {
			if absDiff(got[i], want) > 1e-3 {
				t.Errorf("sample %d = %v, want ~%v", i, got[i], want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for IQ")
	}

	if !srv.sawCall(callSetupStream) {
		t.Error("server did not see SETUP_STREAM")
	}
	if !srv.sawCall(callActivateStream) {
		t.Error("server did not see ACTIVATE_STREAM")
	}

	// The client must have sent the initial flow-control ACK (issue #542
	// follow-up); without it a real server streams nothing. The ACK advertises
	// the in-flight credit window in its elems field.
	acks := srv.recordedACKs()
	if len(acks) == 0 {
		t.Fatal("server did not receive any flow-control ACK from client")
	}
	if acks[0].elems != int32(maxInFlightSeqs) {
		t.Errorf("initial ACK elems = %d, want %d (advertised window)", acks[0].elems, maxInFlightSeqs)
	}

	// With no stream_mtu configured the setup frame stays byte-identical to
	// before: only remote:prot is sent, never remote:mtu.
	if kw := srv.recordedSetupKwargs(); kw["remote:prot"] != "tcp" || kw["remote:mtu"] != "" {
		t.Errorf("default setup kwargs = %v, want only remote:prot=tcp", kw)
	}

	// The default (non-diversity) path requests a single RX channel.
	if chans := srv.recordedSetupChannels(); len(chans) != 1 || chans[0] != 0 {
		t.Errorf("default SETUP_STREAM channels = %v, want [0]", chans)
	}

	// Cancelling the context must close the channel.
	cancel()
	drain := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // closed — success
			}
		case <-drain:
			t.Fatal("stream channel not closed after ctx cancel")
		}
	}
}

// TestStreamIQDiversity drives the end-to-end MRC diversity path: with
// Diversity="mrc" the client must request a 2-channel stream ([0,1]) and emit a
// single combined branch. The fake server streams two contiguous per-channel
// blocks in one datagram; here ch1 == ch0, so the coherent combine recovers ch0
// exactly and the emitted chunk is one branch (half the datagram's samples).
//
// NOTE: this validates the plumbing (2-channel request → de-interleave →
// combine → single stream), not the SoapyRemote wire-format assumption — the
// fake interleaves channels the same way deinterleave() reads them. The block
// layout still needs on-air confirmation against a real dual-RX server (#1062).
func TestStreamIQDiversity(t *testing.T) {
	srv := newFakeSoapyServer(t)
	ch := []complex64{complex(0.5, -0.5), complex(0.25, 0.25), complex(-0.3, 0.4)}
	// One datagram = [ch0 block][ch1 block], ch1 == ch0.
	srv.streamSamples = append(append([]complex64{}, ch...), ch...)

	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16", Diversity: "mrc"}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	select {
	case got, ok := <-out:
		if !ok {
			t.Fatal("stream channel closed before any data")
		}
		if len(got) != len(ch) {
			t.Fatalf("combined chunk len = %d, want %d (one branch)", len(got), len(ch))
		}
		for i, want := range ch {
			if absDiff(got[i], want) > 2e-3 {
				t.Errorf("combined sample %d = %v, want ~%v (ch1==ch0 ⇒ recover ch0)", i, got[i], want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for combined IQ")
	}

	if chans := srv.recordedSetupChannels(); len(chans) != 2 || chans[0] != 0 || chans[1] != 1 {
		t.Errorf("SETUP_STREAM channels = %v, want [0 1]", chans)
	}
}

// TestStreamMTU verifies a configured stream_mtu both reaches the server as the
// remote:mtu setup arg and resizes the client's advertised flow-control window
// (streamWindowBytes/mtu), keeping both ends consistent.
func TestStreamMTU(t *testing.T) {
	const mtu = 3000
	srv := newFakeSoapyServer(t)
	srv.streamSamples = []complex64{complex(0.5, -0.5), complex(0.25, 0.25)}
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16", StreamMTU: mtu}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("stream channel closed before any data")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for IQ")
	}

	if kw := srv.recordedSetupKwargs(); kw["remote:mtu"] != "3000" {
		t.Errorf("setup kwargs remote:mtu = %q, want \"3000\" (kwargs=%v)", kw["remote:mtu"], kw)
	}

	acks := srv.recordedACKs()
	if len(acks) == 0 {
		t.Fatal("server did not receive any flow-control ACK from client")
	}
	wantWindow := int32(streamWindowBytes / mtu)
	if acks[0].elems != wantWindow {
		t.Errorf("initial ACK elems = %d, want %d (streamWindowBytes/mtu)", acks[0].elems, wantWindow)
	}
}

// TestStreamWindow verifies a configured stream_window both reaches the server
// as the remote:window setup arg and resizes the client's advertised
// flow-control credit (window/mtu) instead of the default streamWindowBytes.
func TestStreamWindow(t *testing.T) {
	const window = 2 * 1024 * 1024 // 2097152
	srv := newFakeSoapyServer(t)
	srv.streamSamples = []complex64{complex(0.5, -0.5), complex(0.25, 0.25)}
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16", StreamWindow: window}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for IQ")
	}

	if kw := srv.recordedSetupKwargs(); kw["remote:window"] != "2097152" {
		t.Errorf("setup kwargs remote:window = %q, want \"2097152\" (kwargs=%v)", kw["remote:window"], kw)
	}

	acks := srv.recordedACKs()
	if len(acks) == 0 {
		t.Fatal("server did not receive any flow-control ACK from client")
	}
	// Default MTU (1500) with the configured window: credit = window/mtu.
	wantWindow := int32(window / streamMTU)
	if acks[0].elems != wantWindow {
		t.Errorf("initial ACK elems = %d, want %d (window/mtu)", acks[0].elems, wantWindow)
	}
}

func TestOpenRejectsUDP(t *testing.T) {
	srv := newFakeSoapyServer(t)
	drv := New([]Spec{{Addr: srv.Addr(), StreamProtocol: "udp"}}, testLogger())
	if _, err := drv.Open(0); err == nil {
		t.Fatal("expected Open to reject udp stream_protocol")
	}
}

func absDiff(a, b complex64) float64 {
	dr := float64(real(a) - real(b))
	di := float64(imag(a) - imag(b))
	if dr < 0 {
		dr = -dr
	}
	if di < 0 {
		di = -di
	}
	if dr > di {
		return dr
	}
	return di
}
