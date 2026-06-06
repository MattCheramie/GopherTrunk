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

	// samples streamed per datagram once activated.
	streamSamples []complex64
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
	activate := make(chan string, 1) // streamID -> start streaming
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
		// activateSID is signalled to the data goroutine only AFTER the
		// call is recorded below; otherwise the client can receive IQ and
		// the test can check sawCall(ACTIVATE) before this loop records it.
		var activateSID string
		var doActivate bool
		switch id {
		case callSetFrequency:
			_, _ = u.char()
			_, _ = u.i32()
			rec.freqHz, _ = u.f64()
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
		case callSetSampleRate:
			_, _ = u.char()
			_, _ = u.i32()
			rec.freqHz, _ = u.f64()
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
		case callSetGain:
			_, _ = u.char()
			_, _ = u.i32()
			rec.gainDB, _ = u.f64()
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
		case callSetGainMode:
			_, _ = u.char()
			_, _ = u.i32()
			rec.gainAuto, _ = u.boolean()
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
		case callGetNativeStreamFormat:
			s.respond(conn, func(p *packer) {
				p.str("CS16")
				p.f64(1.0)
			})
		case callSetupStream:
			dataPort := s.startDataServer(activate)
			s.respond(conn, func(p *packer) {
				p.str("0") // streamId
				p.str(dataPort)
			})
		case callActivateStream:
			sid, _ := u.str()
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
			activateSID = sid
			doActivate = true
		default:
			// MAKE, DEACTIVATE, CLOSE, WRITE_SETTING, FREQ_CORRECTION, ...
			s.respond(conn, func(p *packer) { p.raw8(tVoid) })
		}
		s.mu.Lock()
		s.calls = append(s.calls, rec)
		s.mu.Unlock()
		if doActivate {
			select {
			case activate <- activateSID:
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

// startDataServer binds a TCP data listener and, once activated, streams
// repeating CS16 datagrams of s.streamSamples. Returns the bound port.
func (s *fakeSoapyServer) startDataServer(activate <-chan string) string {
	dataLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.t.Fatalf("data listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(dataLn.Addr().String())
	go func() {
		defer dataLn.Close()
		conn, err := dataLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		<-activate // wait until ACTIVATE_STREAM
		seq := uint32(0)
		for {
			payload := encodeCS16(s.streamSamples)
			hdr := encodeStreamHeader(streamHeader{
				bytes:    uint32(streamHeaderSize + len(payload)),
				sequence: seq,
				elems:    int32(len(s.streamSamples)),
			})
			if _, err := conn.Write(append(hdr, payload...)); err != nil {
				return
			}
			seq++
			time.Sleep(time.Millisecond)
		}
	}()
	return port
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
