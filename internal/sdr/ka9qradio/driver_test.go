package ka9qradio

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestEnumerate(t *testing.T) {
	d := New([]Spec{
		{Addr: "hf.local", SSRC: 100},
		{Addr: "239.1.2.3:5006", SSRC: 200, Serial: "named"},
		{Addr: "", SSRC: 300}, // skipped: no addr
	}, testLogger())
	infos, err := d.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("want 2 infos, got %d: %+v", len(infos), infos)
	}
	if infos[0].Driver != DriverName || infos[0].Serial == "" {
		t.Fatalf("info0: %+v", infos[0])
	}
	if infos[1].Serial != "named" {
		t.Fatalf("info1 serial: %+v", infos[1])
	}
}

// newLoopbackDevice builds a device whose data socket is a local unicast UDP
// listener, so StreamIQ can be exercised without real multicast. It returns the
// device and the address a test sender should write RTP packets to.
func newLoopbackDevice(t *testing.T, enc encoding, ssrc uint32) (*device, *net.UDPAddr) {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	dev := &device{
		info:       sdr.Info{Driver: DriverName, Serial: "test"},
		ssrc:       ssrc,
		sampleRate: 24000,
		enc:        enc,
		dataConn:   conn,
		log:        testLogger(),
	}
	t.Cleanup(func() { _ = dev.Close() })
	return dev, conn.LocalAddr().(*net.UDPAddr)
}

func TestStreamIQLoopback(t *testing.T) {
	const ssrc = 0xCAFE
	dev, addr := newLoopbackDevice(t, encS16LE, ssrc)

	if got, _ := dev.ActualSampleRate(); got != 24000 {
		t.Fatalf("ActualSampleRate %d", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	sender, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer sender.Close()

	// One IQ sample: I=16384(->0.5), Q=-16384(->-0.5).
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint16(payload[0:2], u16(16384))
	binary.LittleEndian.PutUint16(payload[2:4], u16(-16384))

	// A foreign SSRC that must be dropped, then our packet.
	_, _ = sender.Write(buildRTP(96, 1, ssrc+1, payload))
	_, _ = sender.Write(buildRTP(96, 2, ssrc, payload))

	select {
	case chunk := <-ch:
		if len(chunk) != 1 || !closeC(chunk[0], complex(0.5, -0.5), 1e-4) {
			t.Fatalf("got chunk %v", chunk)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for IQ chunk")
	}
}

func TestSetCenterFreqSendsCommand(t *testing.T) {
	// Listener stands in for the radiod status group.
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	cmdConn, err := net.DialUDP("udp4", nil, listener.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	dev := &device{ssrc: 777, cmdConn: cmdConn, dataConn: nil, log: testLogger()}

	if err := dev.SetCenterFreq(151000000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	buf := make([]byte, 1500)
	_ = listener.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := listener.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	if buf[0] != pktCommand {
		t.Fatalf("not a command packet: % x", buf[:n])
	}
	st, err := parseStatus(buf[:n])
	if err != nil {
		t.Fatalf("parse command: %v", err)
	}
	if st.radioFreqHz != 151000000 {
		t.Fatalf("retune freq %v", st.radioFreqHz)
	}
	if !st.hasSSRC || st.ssrc != 777 {
		t.Fatalf("retune ssrc %+v", st)
	}
}

func TestNoopSettersAndClose(t *testing.T) {
	dev := &device{log: testLogger()}
	if err := dev.SetSampleRate(2_400_000); err != nil {
		t.Fatalf("SetSampleRate must not fail: %v", err)
	}
	for _, err := range []error{dev.SetGain(100), dev.SetPPM(5), dev.SetBiasTee(true)} {
		if err != nil {
			t.Fatalf("noop setter returned %v", err)
		}
	}
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Idempotent.
	if err := dev.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestResolveAddrLiteral(t *testing.T) {
	ap, err := resolveAddr("239.5.6.7:5006", defaultStatusPort, time.Second)
	if err != nil {
		t.Fatalf("resolveAddr: %v", err)
	}
	if ap.String() != "239.5.6.7:5006" {
		t.Fatalf("got %s", ap)
	}
	// Default port applied when omitted.
	ap, err = resolveAddr("239.5.6.7", defaultDataPort, time.Second)
	if err != nil {
		t.Fatalf("resolveAddr no port: %v", err)
	}
	if ap.Port() != defaultDataPort {
		t.Fatalf("default port not applied: %s", ap)
	}
}
