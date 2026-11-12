package iio

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// A probe must leave the board as it found it: no WRITE of any kind, not
// even the RX LO power-up Open performs.
func TestProbeIsReadOnlyAndReportsBoard(t *testing.T) {
	f := newFakeIIOD(t)
	f.mu.Lock()
	f.attrs["iio:device0 altvoltage0 frequency"] = "433920000"
	f.attrs["iio:device0 voltage0 gain_control_mode"] = "slow_attack"
	f.attrs["iio:device0 voltage0 hardwaregain"] = "71.000000 dB"
	f.readbuf = func(w io.Writer, _ string, n int, _ int) {
		// I at the positive rail, Q at half scale, for n/4 samples.
		vals := make([]int16, n/2)
		for i := range vals {
			if i%2 == 0 {
				vals[i] = 2047
			} else {
				vals[i] = -1024
			}
		}
		writeChunk(w, le16(vals...), true)
	}
	f.mu.Unlock()

	rep, err := Probe(f.addr(), 2*time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.commands() {
		if strings.HasPrefix(c, "WRITE") {
			t.Errorf("probe wrote to the board: %q", c)
		}
	}
	if rep.BindErr != nil {
		t.Fatalf("BindErr = %v", rep.BindErr)
	}
	if rep.Version != "0.25.b6028fd" || rep.RXDevice != "iio:device3" || rep.SampleFormat != "le:S12/16>>0" {
		t.Errorf("report = %+v", rep)
	}
	if rep.LOFrequency != "433920000" || rep.GainMode != "slow_attack" || rep.HardwareGain != "71.000000 dB" {
		t.Errorf("read-back = LO %q mode %q gain %q", rep.LOFrequency, rep.GainMode, rep.HardwareGain)
	}
	if rep.StreamErr != nil || rep.StreamSamples != 4096 {
		t.Fatalf("stream: %v, %d samples", rep.StreamErr, rep.StreamSamples)
	}
	// Half the values sit at the rail.
	if rep.ClipFraction < 0.49 || rep.ClipFraction > 0.51 {
		t.Errorf("clip fraction = %v, want 0.5", rep.ClipFraction)
	}
	out := rep.String()
	for _, want := range []string{"usable by sdr.iio", "PlutoSDR Rev.C", "cf-ad9361-lpc", "433920000", "at the ADC rail"} {
		if !strings.Contains(out, want) {
			t.Errorf("report text lacks %q:\n%s", want, out)
		}
	}
	var open string
	for _, c := range f.commands() {
		if strings.HasPrefix(c, "OPEN") {
			open = c
		}
	}
	if open != "OPEN iio:device3 4096 00000003\r\n" {
		t.Errorf("OPEN = %q", open)
	}
}

// A board the driver would refuse still yields its version, XML and device
// list, so the operator has something to paste.
func TestProbeReportsUnusableBoard(t *testing.T) {
	f := newFakeIIOD(t)
	f.xml = []byte(`<?xml version="1.0"?><context name="x"><context-attribute name="hw_model" value="Something Else"/><device id="iio:device0" name="adm1177"/></context>`)
	rep, err := Probe(f.addr(), time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if rep.BindErr == nil || !strings.Contains(rep.BindErr.Error(), "ad9361-phy") {
		t.Fatalf("BindErr = %v", rep.BindErr)
	}
	if len(rep.XML) == 0 {
		t.Error("XML missing from the report")
	}
	if out := rep.String(); !strings.Contains(out, "NOT USABLE") || !strings.Contains(out, "adm1177") {
		t.Errorf("report text:\n%s", out)
	}
	for _, c := range f.commands() {
		if strings.HasPrefix(c, "OPEN") || strings.HasPrefix(c, "WRITE") {
			t.Errorf("probe of an unusable board sent %q", c)
		}
	}
}

// A board that never fills the buffer must not hang the probe.
func TestProbeStreamTimesOut(t *testing.T) {
	f := newFakeIIOD(t) // readbuf nil: blocks
	start := time.Now()
	rep, err := Probe(f.addr(), 500*time.Millisecond, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if rep.StreamErr == nil {
		t.Fatal("StreamErr = nil on a board that sent nothing")
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Errorf("probe took %v", el)
	}
	if !strings.Contains(fmt.Sprint(rep), "stream: FAILED") {
		t.Errorf("report text lacks the stream failure")
	}
}
