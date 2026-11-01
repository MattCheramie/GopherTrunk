package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtltcp"
)

// TestWMBusLogKeepsTheStopReason is the #1256 regression: the reporter's S5
// run stopped on its own in the field, and "there was the question to give
// details, but they could not be saved and are lost". The reason a live run
// ended (IQ stream ended + the source's own cause), the SDR warnings and the
// stop hints went to the screen only; -log ended mid-stream with no reason.
// Here an rtl_tcp server streams briefly and hangs up — a lost source — and
// the -log file must carry the stop reason, in plain and in -json form.
//
// The command exits through os.Exit, so it runs in a child process: this
// test binary re-executed with GT_WMBUS_LOG_HELPER set.
func TestWMBusLogKeepsTheStopReason(t *testing.T) {
	if os.Getenv("GT_WMBUS_LOG_HELPER") != "" {
		runWMBus(strings.Fields(os.Getenv("GT_WMBUS_LOG_ARGS")))
		return
	}
	for _, asJSON := range []bool{false, true} {
		name := "text"
		if asJSON {
			name = "json"
		}
		t.Run(name, func(t *testing.T) {
			addr := hangUpRTLTCP(t, 64_000) // a little IQ, then the server hangs up
			logPath := filepath.Join(t.TempDir(), "wm.log")
			args := "-rtltcp " + addr + " -rate 1600000 -status 0 -reconnect=false -log " + logPath
			if asJSON {
				args += " -json"
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestWMBusLogKeepsTheStopReason$")
			cmd.Env = append(os.Environ(), "GT_WMBUS_LOG_HELPER=1", "GT_WMBUS_LOG_ARGS="+args)
			cmd.Stdin = nil // /dev/null: no stop-on-"q" watcher, no prompt
			var screen bytes.Buffer
			cmd.Stdout, cmd.Stderr = &screen, &screen
			done := make(chan error, 1)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if err == nil {
					t.Fatalf("wmbus exited 0 after its source hung up; want a failure exit\n%s", screen.String())
				}
			case <-time.After(30 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("wmbus did not exit after its source hung up\n%s", screen.String())
			}
			if !strings.Contains(screen.String(), "wmbus: IQ stream ended") {
				t.Fatalf("screen lacks the stop reason:\n%s", screen.String())
			}
			b, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			log := string(b)
			t.Logf("-log:\n%s", log)
			for _, want := range []string{"wmbus: listening on rtl_tcp", "wmbus: stopped after", "wmbus: IQ stream ended"} {
				if !strings.Contains(log, want) {
					t.Errorf("-log lacks %q", want)
				}
			}
			if asJSON {
				for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
					var v map[string]any
					if err := json.Unmarshal([]byte(line), &v); err != nil {
						t.Errorf("-json -log line does not parse: %q", line)
					}
				}
			}
		})
	}
}

// The echo writer must turn piecewise writes into whole log lines, wrap each
// for -json, and drop blank lines.
func TestWMBusLogEchoSplitsLines(t *testing.T) {
	var buf bytes.Buffer
	e := newWMBusLogEcho(&buf, false)
	_, _ = io.WriteString(e, "wmbus: one")
	_, _ = io.WriteString(e, " line\n\nwmbus: two\r\nwmbus: partial")
	if got, want := buf.String(), "wmbus: one line\nwmbus: two\n"; got != want {
		t.Errorf("text echo = %q, want %q", got, want)
	}

	buf.Reset()
	e = newWMBusLogEcho(&buf, true)
	_, _ = io.WriteString(e, "time=x level=WARN msg=\"usb stall\"\n")
	var v struct{ Status string }
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil || v.Status != `time=x level=WARN msg="usb stall"` {
		t.Errorf("json echo = %q (err %v)", buf.String(), err)
	}
}

// flakyRTLTCP is an rtl_tcp server whose first session streams iqBytes and
// hangs up — a dongle dropping off a phone's USB bus — and whose later
// sessions stream until the client leaves: the dongle back on the bus.
func flakyRTLTCP(t *testing.T, iqBytes int) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	hdr := make([]byte, 12)
	copy(hdr[:4], rtltcp.Magic[:])
	binary.BigEndian.PutUint32(hdr[4:8], 5)
	binary.BigEndian.PutUint32(hdr[8:12], 29)
	chunk := make([]byte, 32_000) // 10 ms of u8 IQ at 1.6 MS/s
	for i := range chunk {
		chunk[i] = 127
	}
	go func() {
		for session := 0; ; session++ {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn, first bool) {
				defer c.Close()
				if _, err := c.Write(hdr); err != nil {
					return
				}
				if first {
					_, _ = c.Write(make([]byte, iqBytes))
					time.Sleep(100 * time.Millisecond)
					return
				}
				for {
					if _, err := c.Write(chunk); err != nil {
						return
					}
					time.Sleep(10 * time.Millisecond)
				}
			}(c, session == 0)
		}
	}()
	return ln.Addr().String()
}

// TestWMBusReconnectsAfterLosingTheSDR is the #1256 walk of 1 Nov 2026: the
// phone's RTL-SDR dropped off the USB bus twice ("usbdevfs: SUBMITURB
// resubmit: usb: device disconnected"), each time ending the run, and only a
// restart by hand — minutes later, when the operator noticed — brought it
// back. A live run now waits for the SDR and carries on: the log says when
// it was lost and when it came back, and the run ends normally (exit 0, with
// its summary) at -seconds.
func TestWMBusReconnectsAfterLosingTheSDR(t *testing.T) {
	if os.Getenv("GT_WMBUS_RECONNECT_HELPER") != "" {
		runWMBus(strings.Fields(os.Getenv("GT_WMBUS_RECONNECT_ARGS")))
		return
	}
	addr := flakyRTLTCP(t, 64_000)
	logPath := filepath.Join(t.TempDir(), "wm.log")
	cmd := exec.Command(os.Args[0], "-test.run=^TestWMBusReconnectsAfterLosingTheSDR$")
	cmd.Env = append(os.Environ(), "GT_WMBUS_RECONNECT_HELPER=1",
		"GT_WMBUS_RECONNECT_ARGS=-rtltcp "+addr+" -rate 1600000 -status 0 -seconds 3 -log "+logPath)
	cmd.Stdin = nil
	var screen bytes.Buffer
	cmd.Stdout, cmd.Stderr = &screen, &screen
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wmbus exited with %v after its SDR came back; want a normal end at -seconds\n%s", err, screen.String())
		}
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("wmbus did not end at -seconds\n%s", screen.String())
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(b)
	t.Logf("-log:\n%s", log)
	for _, want := range []string{"wmbus: lost the SDR at", "rtl_tcp server closed the connection", "wmbus: SDR back after", "wmbus: stopped after"} {
		if !strings.Contains(log, want) {
			t.Errorf("-log lacks %q", want)
		}
	}
	if strings.Count(log, "wmbus: stopped after") != 1 {
		t.Error("want exactly one summary, at the real end of the run")
	}
}
