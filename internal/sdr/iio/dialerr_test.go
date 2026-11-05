package iio

import (
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func opErr(err error) error {
	return &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", err)}
}

func TestClassifyDialError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want dialFailure
	}{
		// #1329: the PlutoSky did not answer ping, and libiio said exactly this.
		{"posix no route", opErr(syscall.EHOSTUNREACH), dialUnreachable},
		{"posix net unreachable", opErr(syscall.ENETUNREACH), dialUnreachable},
		{"posix refused", opErr(syscall.ECONNREFUSED), dialRefused},
		{"timeout", &net.OpError{Op: "dial", Net: "tcp", Err: timeoutErr{}}, dialTimeout},
		{"windows unreachable", errors.New("dial tcp 192.168.1.31:30431: connectex: A socket operation was attempted to an unreachable host."), dialUnreachable},
		{"windows refused", errors.New("dial tcp 192.168.1.31:30431: connectex: No connection could be made because the target machine actively refused it."), dialRefused},
		{"other", errors.New("dial tcp: lookup pluto.local: no such host"), dialOther},
	}
	for _, c := range cases {
		if got := classifyDialError(c.err); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestExplainDialErrorNamesTheLayer(t *testing.T) {
	err := explainDialError("192.168.1.31:30431", opErr(syscall.EHOSTUNREACH))
	msg := err.Error()
	for _, want := range []string{"ping 192.168.1.31", "before iiod", "192.168.2.1"} {
		if !strings.Contains(msg, want) {
			t.Errorf("unreachable hint missing %q: %s", want, msg)
		}
	}
	if !errors.Is(err, syscall.EHOSTUNREACH) {
		t.Error("wrapped error lost the underlying errno")
	}
}

// TestProbeOnClosedPortSaysIIODIsNotListening dials a real closed port: the
// host answers with a refusal, so the hint must point at iiod, not at the
// network.
func TestProbeOnClosedPortSaysIIODIsNotListening(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	_, err = Probe(addr, 2*time.Second, 0)
	if err == nil {
		t.Fatal("probe of a closed port succeeded")
	}
	if !strings.Contains(err.Error(), "nothing is listening on the iiod port") {
		t.Errorf("closed-port probe error does not point at iiod: %v", err)
	}
}
