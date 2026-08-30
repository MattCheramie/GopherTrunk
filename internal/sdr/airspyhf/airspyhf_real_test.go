package airspyhf

import (
	"context"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

const (
	airspyhfRealEnv       = "GOPHERTRUNK_AIRSPYHF_REAL"
	airspyhfRealSerialEnv = "GOPHERTRUNK_AIRSPYHF_REAL_SERIAL"
	airspyhfRealHzEnv     = "GOPHERTRUNK_AIRSPYHF_REAL_CENTER_HZ"
	airspyhfRealRateEnv   = "GOPHERTRUNK_AIRSPYHF_REAL_RATE_HZ"
	airspyhfRealGainEnv   = "GOPHERTRUNK_AIRSPYHF_REAL_GAIN_TENTH_DB"
	airspyhfRealBiasEnv   = "GOPHERTRUNK_AIRSPYHF_REAL_BIAS_TEE"
	airspyhfRealDiagEnv   = "GOPHERTRUNK_AIRSPYHF_REAL_DIAG"
)

func TestRealHardware_OpenConfigureStream(t *testing.T) {
	requireRealAirspyHF(t)

	// Default center sits in the VHF window (NOAA WX, 162.4 MHz). The HF+
	// covers 9 kHz–31 MHz plus 60–260 MHz with a coverage gap between —
	// the default must NOT land in that 31–60 MHz gap, where the
	// synthesizer won't lock and the stream carries nothing meaningful.
	centerHz := mustEnvUint32(t, airspyhfRealHzEnv, 162_400_000)
	// 768 kSa/s is the one rate every HF+ variant advertises (Discovery
	// adds 192k/256k/384k below it; Dual Port typically lists only 768k).
	rateHz := mustEnvUint32(t, airspyhfRealRateEnv, 768_000)
	gainTenthDB := mustEnvInt(t, airspyhfRealGainEnv, -1) // default: negative = HF AGC on
	serialHint := strings.TrimSpace(os.Getenv(airspyhfRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", airspyhfRealEnv)
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
			t.Fatalf("no enumerated Airspy HF+ serial matched %q; found: %v", serialHint, collectSerials(infos))
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

	// 10 s, not the Airspy R2's 5 s: at 192–768 kSa/s the HF+ takes
	// materially longer to fill the first bulk transfer than the R2/Mini
	// does at 2.5–10 MSa/s.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	requireRealAirspyHF(t)
	if !envBool(airspyhfRealBiasEnv) {
		t.Skipf("set %s=1 to run real Airspy HF+ bias-tee toggle validation", airspyhfRealBiasEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyhfRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", airspyhfRealEnv)
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
			t.Fatalf("no enumerated Airspy HF+ serial matched %q; found: %v", serialHint, collectSerials(infos))
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

	// Note: on the HF+ Dual Port only the HF (SMA-1) port carries the
	// bias voltage; the VHF SMA-2 port is unaffected by this toggle.
	if err := dev.SetBiasTee(true); err != nil {
		t.Fatalf("SetBiasTee(true): %v", err)
	}
	if err := dev.SetBiasTee(false); err != nil {
		t.Fatalf("SetBiasTee(false): %v", err)
	}
}

func TestRealHardware_USBControlTransferProbe(t *testing.T) {
	requireRealAirspyHF(t)
	if !envBool(airspyhfRealDiagEnv) {
		t.Skipf("set %s=1 to run raw USB control-transfer probe", airspyhfRealDiagEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(airspyhfRealSerialEnv))
	enum := usb.DefaultEnumerator()
	descs, err := enum.List(vidAirspyHF, pidAirspyHF)
	if err != nil {
		t.Fatalf("usb enumerate: %v", err)
	}
	if len(descs) == 0 {
		t.Fatalf("usb enumerate returned no Airspy HF+ descriptors")
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
			t.Fatalf("no Airspy HF+ descriptor matched %q; found serials: %v", serialHint, collectUSBSerials(descs))
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

	okCount := 0
	var lastErr error

	// Firmware version string (opcode 9). The driver reads this at
	// Open-time too, but here we log the raw reply so a NAK-ing older
	// firmware is visible.
	if buf, verErr := tr.ControlIn(reqGetVersionString, 0, 0, 255, controlTimeoutMs); verErr == nil {
		okCount++
		t.Logf("version string probe ok: req=0x%02x version=%q", reqGetVersionString, cleanVersionString(buf))
	} else {
		lastErr = verErr
		t.Logf("version string probe failed: req=0x%02x err=%v", reqGetVersionString, verErr)
	}

	// Serial + board ID (opcode 7). The driver never issues this call,
	// which makes it a good diag-only probe: it exercises a request path
	// no driver bug can have masked. libairspyhf reads 4 u32 words
	// (board ID + 3 serial words).
	if buf, snErr := tr.ControlIn(reqGetSerialBoardID, 0, 0, 16, controlTimeoutMs); snErr == nil {
		okCount++
		t.Logf("serial/board-id probe ok: req=0x%02x gotLen=%d raw=% x", reqGetSerialBoardID, len(buf), buf)
	} else {
		lastErr = snErr
		t.Logf("serial/board-id probe failed: req=0x%02x err=%v", reqGetSerialBoardID, snErr)
	}

	// Supported-samplerate table (opcode 3): wIndex=0 returns a u32
	// count; wIndex=count returns count×u32 rates. Logging the real
	// table is the point — SetSampleRate sends the *index* of the
	// closest advertised rate, so what the firmware actually advertises
	// decides what an operator's `sample_rate` maps to.
	if cntBytes, cntErr := tr.ControlIn(reqGetSamplerates, 0, 0, 4, controlTimeoutMs); cntErr == nil {
		okCount++
		count := binary.LittleEndian.Uint32(cntBytes)
		t.Logf("samplerate count probe ok: req=0x%02x count=%d", reqGetSamplerates, count)
		if count > 0 && count <= 32 {
			if listBytes, listErr := tr.ControlIn(reqGetSamplerates, 0, uint16(count), int(count*4), controlTimeoutMs); listErr == nil && len(listBytes) >= int(count*4) {
				rates := make([]uint32, count)
				for i := range rates {
					rates[i] = binary.LittleEndian.Uint32(listBytes[4*i:])
				}
				t.Logf("samplerate list probe ok: rates=%v Hz", rates)
			} else {
				t.Logf("samplerate list probe failed: req=0x%02x wIndex=%d err=%v gotLen=%d", reqGetSamplerates, count, listErr, len(listBytes))
			}
		} else {
			t.Logf("samplerate count %d implausible; skipping list probe", count)
		}
	} else {
		lastErr = cntErr
		t.Logf("samplerate count probe failed: req=0x%02x err=%v", reqGetSamplerates, cntErr)
	}

	if okCount == 0 {
		t.Fatalf("all raw control probes failed (last err: %v)", lastErr)
	}
}

func requireRealAirspyHF(t *testing.T) {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(airspyhfRealEnv))
	if v == "" || v == "0" || strings.EqualFold(v, "false") {
		t.Skipf("set %s=1 to run real Airspy HF+ hardware validation", airspyhfRealEnv)
	}
	if testing.Short() {
		t.Skip("skipping real Airspy HF+ hardware validation in -short mode")
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
