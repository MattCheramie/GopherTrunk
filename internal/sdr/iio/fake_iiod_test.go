package iio

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeIIOD is a minimal iiod speaking the v0.25 ASCII protocol. It parses
// commands generically (verb + tokens) and records each one byte-for-byte —
// a WRITE as its header line plus its value bytes — so tests can assert
// LITERAL wire strings rather than strings built by the driver's own
// formatters (the SoapyRemote opcode trap in CLAUDE.md).
type fakeIIOD struct {
	t   *testing.T
	ln  net.Listener
	xml []byte

	mu       sync.Mutex
	cmds     []string
	attrs    map[string]string // "dev chn attr" → value for READ
	failWith map[string]int    // command prefix → negative errno to answer
	// readbuf serves one READBUF: it must write the whole reply itself.
	readbuf func(w io.Writer, dev string, n int, call int)
	rbCalls int
	conns   []net.Conn
}

func newFakeIIOD(t *testing.T) *fakeIIOD {
	t.Helper()
	xml, err := os.ReadFile("testdata/pluto_context_reconstructed.xml")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIIOD{
		t: t, ln: ln, xml: xml,
		attrs: map[string]string{
			"iio:device1 altvoltage0 frequency_available": "[70000000 1 6000000000]",
			"iio:device4 voltage0 sampling_frequency":     "2400000",
		},
		failWith: map[string]int{},
	}
	go f.serve()
	t.Cleanup(func() {
		ln.Close()
		f.mu.Lock()
		for _, c := range f.conns {
			c.Close()
		}
		f.mu.Unlock()
	})
	return f
}

func (f *fakeIIOD) addr() string { return f.ln.Addr().String() }

func (f *fakeIIOD) commands() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.cmds...)
}

func (f *fakeIIOD) serve() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		f.mu.Lock()
		f.conns = append(f.conns, c)
		f.mu.Unlock()
		go f.handle(c)
	}
}

func (f *fakeIIOD) record(s string) {
	f.mu.Lock()
	f.cmds = append(f.cmds, s)
	f.mu.Unlock()
}

func (f *fakeIIOD) errnoFor(cmd string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	for p, e := range f.failWith {
		if strings.HasPrefix(cmd, p) {
			return e
		}
	}
	return 0
}

func (f *fakeIIOD) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		rec := line
		if tok[0] == "WRITE" {
			n, _ := strconv.Atoi(tok[len(tok)-1])
			payload := make([]byte, n)
			if _, err := io.ReadFull(r, payload); err != nil {
				return
			}
			rec += string(payload)
		}
		f.record(rec)
		if e := f.errnoFor(line); e != 0 {
			fmt.Fprintf(c, "%d\n", e)
			continue
		}
		switch tok[0] {
		case "VERSION":
			io.WriteString(c, "0.25.b6028fd\n")
		case "PRINT":
			fmt.Fprintf(c, "%d\n", len(f.xml))
			c.Write(f.xml)
			io.WriteString(c, "\n")
		case "READ":
			// READ <dev> INPUT|OUTPUT <chn> <attr>
			key := tok[1] + " " + tok[3] + " " + tok[4]
			f.mu.Lock()
			v, ok := f.attrs[key]
			f.mu.Unlock()
			if !ok {
				io.WriteString(c, "-2\n")
				continue
			}
			// iiod's read_chn_attr: the value's length, the value, '\n'.
			fmt.Fprintf(c, "%d\n%s\n", len(v), v)
		case "WRITE":
			n, _ := strconv.Atoi(tok[len(tok)-1])
			fmt.Fprintf(c, "%d\n", n)
		case "OPEN", "CLOSE", "TIMEOUT":
			io.WriteString(c, "0\n")
		case "READBUF":
			n, _ := strconv.Atoi(tok[2])
			f.mu.Lock()
			fn := f.readbuf
			call := f.rbCalls
			f.rbCalls++
			f.mu.Unlock()
			if fn == nil {
				// Block like a board that never fills a buffer.
				time.Sleep(time.Hour)
				return
			}
			fn(c, tok[1], n, call)
		case "EXIT":
			return
		default:
			io.WriteString(c, "-22\n")
		}
	}
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func openFake(t *testing.T, f *fakeIIOD, spec Spec) *device {
	t.Helper()
	spec.Addr = f.addr()
	if spec.ConnectTimeout == 0 {
		spec.ConnectTimeout = 2 * time.Second
	}
	dev, err := New([]Spec{spec}, quietLogger()).Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { dev.Close() })
	return dev.(*device)
}
