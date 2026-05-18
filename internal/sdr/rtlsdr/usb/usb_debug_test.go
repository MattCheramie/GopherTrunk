package usb

import (
	"bytes"
	"errors"
	"strings"
	"syscall"
	"testing"
)

func TestMaybeWrapDebug_OffByDefault(t *testing.T) {
	// Without RTLSDR_DEBUG_USB the wrapper is bypassed and the inner
	// transport is returned verbatim — no allocation, no logging,
	// no behavior change.
	t.Setenv(debugUSBEnv, "")
	inner := NewMockTransport()
	got := MaybeWrapDebug(inner, Descriptor{Bus: 1, Address: 4})
	if got != inner {
		t.Errorf("MaybeWrapDebug with env off returned wrapped transport; want inner returned verbatim")
	}
}

func TestMaybeWrapDebug_OnLogsControlTransfers(t *testing.T) {
	// With RTLSDR_DEBUG_USB=1 the wrapper logs each ControlOut +
	// ControlIn (request type, value/index, payload hex, outcome).
	// One ControlOut + one ControlIn are issued; the captured log
	// must contain both headers and the payload hex.
	t.Setenv(debugUSBEnv, "1")
	var buf bytes.Buffer
	prev := SetDebugSink(&buf)
	t.Cleanup(func() { SetDebugSink(prev) })

	inner := NewMockTransport()
	inner.Script = []CtrlExchange{
		{In: false, BRequest: 0, WValue: 0x2000, WIndex: 0x0110, Data: []byte{0x09}},
		{In: true, BRequest: 0, WValue: 0x0034, WIndex: 0x0600, N: 1, Reply: []byte{0x96}},
	}
	desc := Descriptor{Bus: 1, Address: 7, Serial: "abc"}
	t1 := MaybeWrapDebug(inner, desc)

	if err := t1.ControlOut(0, 0x2000, 0x0110, []byte{0x09}, 300); err != nil {
		t.Fatalf("ControlOut: %v", err)
	}
	out, err := t1.ControlIn(0, 0x0034, 0x0600, 1, 300)
	if err != nil {
		t.Fatalf("ControlIn: %v", err)
	}
	if len(out) != 1 || out[0] != 0x96 {
		t.Errorf("ControlIn returned %#x, want [0x96] (passthrough must not corrupt response)", out)
	}

	logged := buf.String()
	for _, want := range []string{
		"bus=1 addr=7 serial=abc", // descriptor label
		"ControlOut bmReqType=0x40",
		"wValue=0x2000",
		"wIndex=0x0110",
		"wLength=1",
		"data=09",
		"ControlIn  bmReqType=0xc0",
		"data=96",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("log missing %q\nlog:\n%s", want, logged)
		}
	}
}

func TestMaybeWrapDebug_LogsErrorOutcome(t *testing.T) {
	// EPIPE returned by the inner transport must propagate AND show up
	// in the log line so users diffing against rtl_test traces see
	// where the stall happened.
	t.Setenv(debugUSBEnv, "1")
	var buf bytes.Buffer
	prev := SetDebugSink(&buf)
	t.Cleanup(func() { SetDebugSink(prev) })

	inner := NewMockTransport()
	inner.Script = []CtrlExchange{
		{In: false, BRequest: 0, WValue: 0x0034, WIndex: 0x0610, Data: []byte{0xDE, 0xAD}, Err: syscall.EPIPE},
	}
	t1 := MaybeWrapDebug(inner, Descriptor{Bus: 2, Address: 3})

	err := t1.ControlOut(0, 0x0034, 0x0610, []byte{0xDE, 0xAD}, 300)
	if !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("ControlOut err = %v, want EPIPE", err)
	}
	if !strings.Contains(buf.String(), "err=broken pipe") {
		t.Errorf("log missing err=broken pipe; got:\n%s", buf.String())
	}
}

func TestHexBytes_TruncatesAtCap(t *testing.T) {
	// Long payloads are truncated to debugMaxDataBytes (64) with a
	// "... (N more)" suffix so the log line stays usable for the
	// 27-byte R820T init burst and bounds the cost for callers that
	// fan out larger blobs.
	long := make([]byte, debugMaxDataBytes+5)
	for i := range long {
		long[i] = byte(i)
	}
	got := hexBytes(long, debugMaxDataBytes)
	if !strings.HasSuffix(got, "... (5 more)") {
		t.Errorf("hexBytes truncation suffix missing; got tail %q", got[len(got)-20:])
	}
}
