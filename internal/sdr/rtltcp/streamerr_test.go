package rtltcp

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// drain reads ch until it closes (or fails the test after 2 s).
func drain(t *testing.T, ch <-chan []complex64) int {
	t.Helper()
	n := 0
	deadline := time.After(2 * time.Second)
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				return n
			}
			n += len(c)
		case <-deadline:
			t.Fatal("stream channel did not close")
		}
	}
}

// TestStreamErrReportsServerHangUp is the #1230 / #1256 regression: the
// reporter's rtl_tcp driver app dropped the session, and the stream channel
// just closed. `power` turned that into a bare "unexpected EOF" and `wmbus`
// exited with no message at all. The device must now say the server hung up,
// after how long and how much IQ.
func TestStreamErrReportsServerHangUp(t *testing.T) {
	srv := newFakeServer(t)
	dev, err := newDriver(t, srv.Addr()).Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()
	ch, err := dev.StreamIQ(context.Background())
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	if err := sdr.StreamErr(dev); err != nil {
		t.Fatalf("StreamErr while streaming = %v, want nil", err)
	}
	srv.Feed(make([]byte, 1000))
	close(srv.feed) // the fake server closes the socket once its feed is closed

	if got := drain(t, ch); got != 500 {
		t.Fatalf("received %d samples before the hang-up, want 500", got)
	}
	err = sdr.StreamErr(dev)
	if !errors.Is(err, ErrServerClosed) {
		t.Fatalf("StreamErr after server hang-up = %v, want ErrServerClosed", err)
	}
	if !strings.Contains(err.Error(), "1000 bytes") {
		t.Errorf("StreamErr %q does not say how much IQ arrived", err)
	}
}

// A stream the caller cancels ended normally: no reason to report.
func TestStreamErrIsNilAfterCancel(t *testing.T) {
	srv := newFakeServer(t)
	dev, err := newDriver(t, srv.Addr()).Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	cancel()
	drain(t, ch)
	if err := sdr.StreamErr(dev); err != nil {
		t.Fatalf("StreamErr after cancel = %v, want nil", err)
	}
}

// TestOpenHeaderTimeoutExplainsBusyServer: on #1256 the phone's rtl_tcp app
// accepted the TCP connection and then sent nothing, and the error was a bare
// "read header: i/o timeout". rtl_tcp serves one client at a time, so the
// message now says what that silence usually means.
func TestOpenHeaderTimeoutExplainsBusyServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		time.Sleep(time.Second) // accept, then say nothing
	}()
	d := New([]Spec{{Addr: ln.Addr().String(), ConnectTimeout: 150 * time.Millisecond}}, nil)
	_, err = d.Open(0)
	if err == nil {
		t.Fatal("Open succeeded against a server that sent no header")
	}
	if !strings.Contains(err.Error(), "one client at a time") {
		t.Fatalf("header-timeout error %q does not explain the busy-server case", err)
	}
}
