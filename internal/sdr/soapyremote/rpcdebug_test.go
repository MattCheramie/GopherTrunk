package soapyremote

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// bufLogger returns a DEBUG-level logger writing to a buffer, plus a snapshot
// func. The handler is wrapped in a mutex because the driver logs from the
// stream goroutine as well as the caller's.
func bufLogger() (*slog.Logger, func() string) {
	var mu sync.Mutex
	var buf bytes.Buffer
	h := slog.NewTextHandler(&lockedWriter{mu: &mu, w: &buf}, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// The whole point of the trace is that it renders what actually went on the
// wire, not what the caller believed it packed — so this decodes a real
// SETUP_STREAM frame built by the production packer.
func TestDescribePayloadRendersASetupStreamFrame(t *testing.T) {
	p := newPacker()
	p.call(callSetupStream)
	p.char(dirRX)
	p.str("CS16")
	p.sizeList([]int{0, 1})
	p.kwargs(map[string]string{"remote:prot": "tcp", "remote:mtu": "8000"})
	p.str("0")
	p.str("0")
	// writeTo appends the trailer and stamps the header; do the same by hand so
	// the payload boundaries match a real frame.
	p.rawU32(rpcTrailerWord)

	got := describePayload(framePayload(p.buf))
	for _, want := range []string{
		"call SETUP_STREAM(300)",
		// SOAPY_SDR_RX is the byte 1, not the letter 'R', so hex is the honest
		// rendering — the trace must not dress the wire up as something else.
		"char 0x01",
		`str "CS16"`,
		"sizeList[0 1]",
		`remote:mtu="8000"`,
		`remote:prot="tcp"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("describePayload missing %q\ngot: %s", want, got)
		}
	}
}

func TestDescribeCallNamesTheOpcode(t *testing.T) {
	tests := []struct {
		id   int32
		want string
	}{
		{callSetAntenna, "SET_ANTENNA(501)"},
		{callActivateStream, "ACTIVATE_STREAM(302)"},
		{callSetGain, "SET_GAIN(703)"},
		// An id this client never sends still has to render — an unknown
		// opcode in a trace IS the finding, so it must not be swallowed.
		{600, "UNKNOWN(600)"},
	}
	for _, tc := range tests {
		if got := callName(tc.id); got != tc.want {
			t.Errorf("callName(%d) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

// Every opcode the client can send must have a name, or a trace of the very
// call that is misbehaving reads as UNKNOWN.
func TestEveryIssuedOpcodeHasAName(t *testing.T) {
	for _, id := range []int32{
		callFind, callMake, callUnmake, callHangup, callGetServerID,
		callGetHardwareKey, callSetupStream, callCloseStream, callActivateStream,
		callDeactivateStream, callGetNativeStreamFormat, callListAntennas,
		callSetAntenna, callGetAntenna, callSetFrequencyCorrection,
		callSetGainMode, callSetGain, callSetFrequency, callSetSampleRate,
		callGetSampleRate, callGetHardwareTime, callWriteSetting,
	} {
		if _, ok := callNames[id]; !ok {
			t.Errorf("opcode %d has no entry in callNames", id)
		}
	}
}

func TestDescribePayloadReportsAnUndecodableTail(t *testing.T) {
	p := newPacker()
	p.call(callSetGain)
	p.raw8(tArgInfo) // a tag this client never decodes
	p.rawU32(0xdeadbeef)
	p.rawU32(rpcTrailerWord)

	got := describePayload(framePayload(p.buf))
	if !strings.Contains(got, "call SET_GAIN(703)") {
		t.Errorf("prefix not decoded: %s", got)
	}
	if !strings.Contains(got, "undecodable") || !strings.Contains(got, "bytes left") {
		t.Errorf("undecodable tail not reported: %s", got)
	}
}

func TestHexBytesTruncates(t *testing.T) {
	if got := hexBytes([]byte{0x00, 0x0f, 0xff}, 8); got != "00 0f ff" {
		t.Errorf("hexBytes = %q", got)
	}
	got := hexBytes(bytes.Repeat([]byte{0xab}, 10), 3)
	if got != "ab ab ab ... (7 more)" {
		t.Errorf("truncated hexBytes = %q", got)
	}
}

// End-to-end through the real driver against the fake server: opening a device
// with VerboseDebug must trace the MAKE round-trip with its decoded arguments.
func TestOpenWithVerboseDebugTracesRPCFrames(t *testing.T) {
	srv := newFakeSoapyServer(t)
	log, dump := bufLogger()
	drv := New([]Spec{{
		Addr:         srv.Addr(),
		Format:       "CS16",
		DeviceArgs:   map[string]string{"driver": "uhd"},
		VerboseDebug: true,
	}}, log)
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	out := dump()
	if !strings.Contains(out, "soapyremote: rpc >") {
		t.Fatalf("no request trace emitted\n%s", out)
	}
	if !strings.Contains(out, "soapyremote: rpc <") {
		t.Fatalf("no response trace emitted\n%s", out)
	}
	if !strings.Contains(out, "MAKE(1)") {
		t.Errorf("MAKE not named in the trace\n%s", out)
	}
	if !strings.Contains(out, `driver=\"uhd\"`) {
		t.Errorf("MAKE kwargs not decoded in the trace\n%s", out)
	}
}

// Off is the default and must stay silent: no trace lines at all, so an
// operator who leaves log.level at debug for other reasons is unaffected.
func TestOpenWithoutVerboseDebugTracesNothing(t *testing.T) {
	srv := newFakeSoapyServer(t)
	log, dump := bufLogger()
	drv := New([]Spec{{Addr: srv.Addr(), Format: "CS16"}}, log)
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if out := dump(); strings.Contains(out, "soapyremote: rpc") {
		t.Errorf("trace emitted with VerboseDebug off\n%s", out)
	}
}
