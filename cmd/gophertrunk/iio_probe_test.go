package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// serveMiniIIOD answers just enough of iiod's protocol for a read-only probe:
// VERSION, PRINT, an unknown-attribute READ (-ENOENT), OPEN, one READBUF and
// CLOSE. It fails the test on any WRITE, since a probe must not change the
// board.
func serveMiniIIOD(t *testing.T, xml []byte) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				r := bufio.NewReader(c)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					tok := strings.Fields(line)
					switch tok[0] {
					case "VERSION":
						io.WriteString(c, "0.25.b6028fd\n")
					case "PRINT":
						fmt.Fprintf(c, "%d\n", len(xml))
						c.Write(xml)
						io.WriteString(c, "\n")
					case "READ":
						io.WriteString(c, "-2\n")
					case "OPEN", "CLOSE":
						io.WriteString(c, "0\n")
					case "READBUF":
						n, _ := strconv.Atoi(tok[2])
						buf := make([]byte, n)
						for i := 0; i+4 <= n; i += 4 {
							binary.LittleEndian.PutUint16(buf[i:], uint16(int16(512)))
						}
						fmt.Fprintf(c, "%d\n00000003\n", n)
						c.Write(buf)
					case "WRITE":
						t.Errorf("iio probe sent a WRITE: %q", line)
						return
					default:
						io.WriteString(c, "-22\n")
					}
				}
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestIIOProbeCommandSavesContextXML(t *testing.T) {
	xml, err := os.ReadFile("../../internal/sdr/iio/testdata/plutosky_rev_c_context.xml")
	if err != nil {
		t.Fatal(err)
	}
	addr := serveMiniIIOD(t, xml)
	out := filepath.Join(t.TempDir(), "board.xml")
	if code := runIIOProbe([]string{"-addr", addr, "-out", out, "-samples", "4096"}); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, xml) {
		t.Errorf("saved XML differs from the board's (%d vs %d bytes)", len(got), len(xml))
	}
}

func TestIIOProbeCommandFailsOnNonPlutoBoard(t *testing.T) {
	addr := serveMiniIIOD(t, []byte(`<?xml version="1.0"?><context name="x"><device id="iio:device0" name="adm1177"/></context>`))
	if code := runIIOProbe([]string{"-addr", addr, "-samples", "0"}); code != 1 {
		t.Fatalf("exit code %d, want 1 for a board sdr.iio would refuse", code)
	}
}
