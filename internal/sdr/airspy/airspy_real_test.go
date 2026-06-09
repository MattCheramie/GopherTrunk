package airspy

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

const (
	airspyRealEnv         = "GOPHERTRUNK_AIRSPY_REAL"
	airspyRealSerialEnv   = "GOPHERTRUNK_AIRSPY_REAL_SERIAL"
	airspyRealHzEnv       = "GOPHERTRUNK_AIRSPY_REAL_CENTER_HZ"
	airspyRealRateEnv     = "GOPHERTRUNK_AIRSPY_REAL_RATE_HZ"
	airspyRealGainEnv     = "GOPHERTRUNK_AIRSPY_REAL_GAIN_TENTH_DB"
	airspyRealBiasEnv     = "GOPHERTRUNK_AIRSPY_REAL_BIAS_TEE"
	airspyRealDiagEnv     = "GOPHERTRUNK_AIRSPY_REAL_DIAG"
	airspyRealDiagRateEnv = "GOPHERTRUNK_AIRSPY_REAL_DIAG_SAMPLERATE"
)

func TestRealHardware_OpenConfigureStream(t *testing.T) {
	requireRealAirspy(t)

	centerHz := mustEnvUint32(t, airspyRealHzEnv, 144_390_000)
	rateHz := mustEnvUint32(t, airspyRealRateEnv, 2_500_000)
	gainTenthDB := mustEnvInt(t, airspyRealGainEnv, -1) // default auto AGC
	serialHint := strings.TrimSpace(os.Getenv(airspyRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", airspyRealEnv)
	}

	idx := 0
	if serialHint != "" {
		match := -1
		for i := range infos {
			if infos[i].Serial == serialHint || strings.Contains(infos[i].Serial, serialHint) {
				match = i
				break
			}
		}
		if match < 0 {
			t.Fatalf("no enumerated Airspy serial matched %q; found: %v", serialHint, collectSerials(infos))
		}
		idx = match
	}

	dev, err := drv.Open(idx)
	if err != nil {
		t.Fatalf("Open(%d): %v", idx, err)
	}
	t.Cleanup(func() { _ = dev.Close() })

	if err := dev.SetCenterFreq(centerHz); err != nil {
		t.Fatalf("SetCenterFreq(%d): %v", centerHz, err)
	}
	if err := dev.SetSampleRate(rateHz); err != nil {
		t.Fatalf("SetSampleRate(%d): %v", rateHz, err)
	}
	if err := dev.SetGain(gainTenthDB); err != nil {
		t.Fatalf("SetGain(%d): %v", gainTenthDB, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	iq, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	select {
	case chunk, ok := <-iq:
		if !ok {
			t.Fatal("StreamIQ channel closed before first packet")
		}
		if len(chunk) == 0 {
			t.Fatal("StreamIQ produced an empty packet")
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for IQ packet: %v", ctx.Err())
	}
}

func TestRealHardware_BiasTeeToggle(t *testing.T) {
	requireRealAirspy(t)
	if !envBool(airspyRealBiasEnv) {
		t.Skipf("set %s=1 to run real Airspy bias-tee toggle validation", airspyRealBiasEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", airspyRealEnv)
	}

	idx := 0
	if serialHint != "" {
		match := -1
		for i := range infos {
			if infos[i].Serial == serialHint || strings.Contains(infos[i].Serial, serialHint) {
				match = i
				break
			}
		}
		if match < 0 {
			t.Fatalf("no enumerated Airspy serial matched %q; found: %v", serialHint, collectSerials(infos))
		}
		idx = match
	}

	dev, err := drv.Open(idx)
	if err != nil {
		t.Fatalf("Open(%d): %v", idx, err)
	}
	t.Cleanup(func() {
		// Best-effort safety reset: leave bias-tee off when the test exits.
		_ = dev.SetBiasTee(false)
		_ = dev.Close()
	})

	if err := dev.SetBiasTee(true); err != nil {
		t.Fatalf("SetBiasTee(true): %v", err)
	}
	if err := dev.SetBiasTee(false); err != nil {
		t.Fatalf("SetBiasTee(false): %v", err)
	}
}

func TestRealHardware_USBControlTransferProbe(t *testing.T) {
	requireRealAirspy(t)
	if !envBool(airspyRealDiagEnv) {
		t.Skipf("set %s=1 to run raw USB control-transfer probe", airspyRealDiagEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyRealSerialEnv))
	enum := usb.DefaultEnumerator()
	descs, err := enum.List(vidAirspy, pidAirspy)
	if err != nil {
		t.Fatalf("usb enumerate: %v", err)
	}
	if len(descs) == 0 {
		t.Fatalf("usb enumerate returned no Airspy descriptors")
	}

	desc := descs[0]
	if serialHint != "" {
		matched := false
		for _, d := range descs {
			if d.Serial == serialHint || strings.Contains(d.Serial, serialHint) {
				desc = d
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("no Airspy descriptor matched %q; found serials: %v", serialHint, collectUSBSerials(descs))
		}
	}

	t.Logf("probing backend=%s path=%q serial=%q", enum.Name(), desc.Path, desc.Serial)
	tr, err := enum.Open(desc)
	if err != nil {
		t.Fatalf("usb open: %v", err)
	}
	defer tr.Close()
	if err := tr.ClaimInterface(0); err != nil {
		t.Fatalf("usb claim interface 0: %v", err)
	}
	defer tr.ReleaseInterface(0)

	candidates := []uint16{0, 1, 2}
	var (
		inOK, outModeOK           bool
		inLastErr, outModeLastErr error
	)
	for _, idx := range candidates {
		_, inErr := tr.ControlIn(reqGetSamplerates, 0, idx, 4, controlTimeoutMs)
		modeErr := tr.ControlOut(reqReceiverMode, receiverModeOff, idx, nil, controlTimeoutMs)

		if inErr == nil {
			inOK = true
			t.Logf("raw control in samplerate-count succeeded with wIndex=%d", idx)
		} else {
			inLastErr = inErr
			t.Logf("raw control in samplerate-count failed: req=0x%02x wIndex=%d err=%v", reqGetSamplerates, idx, inErr)
		}

		if modeErr == nil {
			outModeOK = true
			t.Logf("raw control out receiver OFF succeeded with wIndex=%d", idx)
		} else {
			outModeLastErr = modeErr
			t.Logf("raw control out receiver OFF failed: req=0x%02x wValue=%d wIndex=%d err=%v", reqReceiverMode, receiverModeOff, idx, modeErr)
		}
	}

	if !inOK && !outModeOK {
		t.Fatalf("raw control probe failed for all wIndex candidates %v (IN=%v receiverMode=%v)", candidates, inLastErr, outModeLastErr)
	}
}

func TestRealHardware_USBSampleRateProbe(t *testing.T) {
	requireRealAirspy(t)
	if !envBool(airspyRealDiagRateEnv) {
		t.Skipf("set %s=1 to run raw samplerate control probe", airspyRealDiagRateEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyRealSerialEnv))
	enum := usb.DefaultEnumerator()
	descs, err := enum.List(vidAirspy, pidAirspy)
	if err != nil {
		t.Fatalf("usb enumerate: %v", err)
	}
	if len(descs) == 0 {
		t.Fatalf("usb enumerate returned no Airspy descriptors")
	}

	desc := descs[0]
	if serialHint != "" {
		matched := false
		for _, d := range descs {
			if d.Serial == serialHint || strings.Contains(d.Serial, serialHint) {
				desc = d
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("no Airspy descriptor matched %q; found serials: %v", serialHint, collectUSBSerials(descs))
		}
	}

	t.Logf("samplerate probe backend=%s path=%q serial=%q", enum.Name(), desc.Path, desc.Serial)
	tr, err := enum.Open(desc)
	if err != nil {
		t.Fatalf("usb open: %v", err)
	}
	defer tr.Close()
	if err := tr.ClaimInterface(0); err != nil {
		t.Fatalf("usb claim interface 0: %v", err)
	}
	defer tr.ReleaseInterface(0)

	inCases := []struct {
		name           string
		wValue, wIndex uint16
		n              int
	}{
		{name: "count-v0-i0", wValue: 0, wIndex: 0, n: 4},
		{name: "count-v0-i1", wValue: 0, wIndex: 1, n: 4},
		{name: "count-v0-i2", wValue: 0, wIndex: 2, n: 4},
		{name: "count-v1-i0", wValue: 1, wIndex: 0, n: 4},
		{name: "list-v0-i4", wValue: 0, wIndex: 4, n: 16},
		{name: "list-v0-i16", wValue: 0, wIndex: 16, n: 64},
	}
	outCases := []struct {
		name           string
		wValue, wIndex uint16
	}{
		{name: "set-v0-i0", wValue: 0, wIndex: 0},
		{name: "set-v1-i0", wValue: 1, wIndex: 0},
		{name: "set-v2-i0", wValue: 2, wIndex: 0},
		{name: "set-v0-i1", wValue: 0, wIndex: 1},
		{name: "set-v1-i1", wValue: 1, wIndex: 1},
		{name: "set-v2-i1", wValue: 2, wIndex: 1},
	}

	inOK, outOK := 0, 0
	var inLastErr, outLastErr error

	for _, tc := range inCases {
		buf, inErr := tr.ControlIn(reqGetSamplerates, tc.wValue, tc.wIndex, tc.n, controlTimeoutMs)
		if inErr == nil {
			inOK++
			t.Logf("samplerate IN %s ok: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d gotLen=%d", tc.name, reqGetSamplerates, tc.wValue, tc.wIndex, tc.n, len(buf))
		} else {
			inLastErr = inErr
			t.Logf("samplerate IN %s failed: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=%d err=%v", tc.name, reqGetSamplerates, tc.wValue, tc.wIndex, tc.n, inErr)
		}
	}

	for _, tc := range outCases {
		outErr := tr.ControlOut(reqSetSamplerate, tc.wValue, tc.wIndex, nil, controlTimeoutMs)
		if outErr == nil {
			outOK++
			t.Logf("samplerate OUT %s ok: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=0", tc.name, reqSetSamplerate, tc.wValue, tc.wIndex)
		} else {
			outLastErr = outErr
			t.Logf("samplerate OUT %s failed: req=0x%02x wValue=0x%04x wIndex=0x%04x wLength=0 err=%v", tc.name, reqSetSamplerate, tc.wValue, tc.wIndex, outErr)
		}
	}

	if inOK == 0 && outOK == 0 {
		t.Fatalf("samplerate probe failed for all variants (last IN=%v last OUT=%v)", inLastErr, outLastErr)
	}
}

func requireRealAirspy(t *testing.T) {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(airspyRealEnv))
	if v == "" || v == "0" || strings.EqualFold(v, "false") {
		t.Skipf("set %s=1 to run real Airspy hardware validation", airspyRealEnv)
	}
	if testing.Short() {
		t.Skip("skipping real Airspy hardware validation in -short mode")
	}
}

func mustEnvUint32(t *testing.T, key string, fallback uint32) uint32 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		t.Fatalf("%s=%q is not a valid unsigned integer: %v", key, raw, err)
	}
	return uint32(v)
}

func mustEnvInt(t *testing.T, key string, fallback int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s=%q is not a valid integer: %v", key, raw, err)
	}
	return v
}

func envBool(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return false
	}
	if v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return true
	}
	return false
}

func collectSerials(infos []sdr.Info) []string {
	out := make([]string, 0, len(infos))
	for _, i := range infos {
		out = append(out, i.Serial)
	}
	return out
}

func collectUSBSerials(descs []usb.Descriptor) []string {
	out := make([]string, 0, len(descs))
	for _, d := range descs {
		out = append(out, d.Serial)
	}
	return out
}
