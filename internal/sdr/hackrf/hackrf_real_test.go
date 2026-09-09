package hackrf

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
	hackrfRealEnv       = "GOPHERTRUNK_HACKRF_REAL"
	hackrfRealSerialEnv = "GOPHERTRUNK_HACKRF_REAL_SERIAL"
	hackrfRealHzEnv     = "GOPHERTRUNK_HACKRF_REAL_CENTER_HZ"
	hackrfRealRateEnv   = "GOPHERTRUNK_HACKRF_REAL_RATE_HZ"
	hackrfRealGainEnv   = "GOPHERTRUNK_HACKRF_REAL_GAIN_TENTH_DB"
	hackrfRealBiasEnv   = "GOPHERTRUNK_HACKRF_REAL_BIAS_TEE"
	hackrfRealAmpEnv    = "GOPHERTRUNK_HACKRF_REAL_RF_AMP"
	hackrfRealDiagEnv   = "GOPHERTRUNK_HACKRF_REAL_DIAG"
)

func TestRealHardware_OpenConfigureStream(t *testing.T) {
	requireRealHackRF(t)

	centerHz := mustEnvUint32(t, hackrfRealHzEnv, 144_390_000)
	rateHz := mustEnvUint32(t, hackrfRealRateEnv, 8_000_000)
	// The HackRF has no AGC: a negative "auto" gain maps to the driver's
	// fixed LNA 16 dB / VGA 20 dB preset (amp off), not to any
	// firmware-managed loop.
	gainTenthDB := mustEnvInt(t, hackrfRealGainEnv, -1)
	serialHint := strings.TrimSpace(os.Getenv(hackrfRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", hackrfRealEnv)
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
			t.Fatalf("no enumerated HackRF serial matched %q; found: %v", serialHint, collectSerials(infos))
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

	// 5 s is plenty at 8 MSa/s — the first bulk transfer fills quickly.
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
	requireRealHackRF(t)
	if !envBool(hackrfRealBiasEnv) {
		t.Skipf("set %s=1 to run real HackRF bias-tee toggle validation", hackrfRealBiasEnv)
	}

	dev := openRealHackRF(t)
	t.Cleanup(func() {
		// Best-effort safety reset: leave the +3.3 V antenna bias
		// (vendor request ANTENNA_ENABLE) off when the test exits.
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

func TestRealHardware_RFAmpToggle(t *testing.T) {
	requireRealHackRF(t)
	if !envBool(hackrfRealAmpEnv) {
		t.Skipf("set %s=1 to run real HackRF RF-amp toggle validation", hackrfRealAmpEnv)
	}

	dev := openRealHackRF(t)
	t.Cleanup(func() {
		// Best-effort safety reset: the front-end amp adds ~+14 dB ahead
		// of everything, so leave it off when the test exits — a HackRF
		// left amp-on near a strong transmitter risks front-end overload.
		_ = dev.SetRFAmp(false)
		_ = dev.Close()
	})

	if err := dev.SetRFAmp(true); err != nil {
		t.Fatalf("SetRFAmp(true): %v", err)
	}
	if err := dev.SetRFAmp(false); err != nil {
		t.Fatalf("SetRFAmp(false): %v", err)
	}
}

func TestRealHardware_ProFeatures(t *testing.T) {
	requireRealHackRF(t)

	dev := openRealHackRF(t)
	isPro := dev.Info().Product == "HackRF Pro"
	t.Cleanup(func() {
		if isPro {
			// Best-effort safety reset: leave the Pro-only knobs off.
			_ = dev.SetNarrowbandFilter(false)
			_ = dev.SetFPGADCBlock(false)
		}
		_ = dev.Close()
	})

	if !isPro {
		// On any non-Pro board the Pro-only calls must return an error,
		// not silently no-op — the pool relies on the error to warn the
		// operator that the configured filter doesn't exist on this unit.
		if err := dev.SetNarrowbandFilter(true); err == nil {
			t.Errorf("SetNarrowbandFilter(true) on %q: want error, got nil", dev.Info().Product)
		}
		if err := dev.SetFPGADCBlock(true); err == nil {
			t.Errorf("SetFPGADCBlock(true) on %q: want error, got nil", dev.Info().Product)
		}
		return
	}

	if err := dev.SetNarrowbandFilter(true); err != nil {
		t.Fatalf("SetNarrowbandFilter(true): %v", err)
	}
	if err := dev.SetNarrowbandFilter(false); err != nil {
		t.Fatalf("SetNarrowbandFilter(false): %v", err)
	}
	if err := dev.SetFPGADCBlock(true); err != nil {
		t.Fatalf("SetFPGADCBlock(true): %v", err)
	}
	if err := dev.SetFPGADCBlock(false); err != nil {
		t.Fatalf("SetFPGADCBlock(false): %v", err)
	}
}

func TestRealHardware_USBControlTransferProbe(t *testing.T) {
	requireRealHackRF(t)
	if !envBool(hackrfRealDiagEnv) {
		t.Skipf("set %s=1 to run raw USB control-transfer probe", hackrfRealDiagEnv)
	}

	serialHint := strings.TrimSpace(os.Getenv(hackrfRealSerialEnv))
	enum := usb.DefaultEnumerator()

	found := 0
	okCount := 0
	var lastErr error

	// The HackRF firmware family enumerates under three PIDs — One,
	// Jawbreaker, Rad1o — all sharing one wire protocol; probe whatever
	// is attached under any of them.
	for _, vp := range KnownVIDPIDs() {
		descs, err := enum.List(vp.VID, vp.PID)
		if err != nil {
			lastErr = err
			t.Logf("usb enumerate %04x:%04x (%s): %v", vp.VID, vp.PID, vp.Name, err)
			continue
		}
		for _, desc := range descs {
			if serialHint != "" && desc.Serial != serialHint && !strings.Contains(desc.Serial, serialHint) {
				continue
			}
			found++
			t.Logf("probing %s backend=%s path=%q serial=%q", vp.Name, enum.Name(), desc.Path, desc.Serial)
			tr, err := enum.Open(desc)
			if err != nil {
				lastErr = err
				t.Logf("usb open failed: %v", err)
				continue
			}
			if err := tr.ClaimInterface(0); err != nil {
				lastErr = err
				t.Logf("usb claim interface 0 failed: %v", err)
				_ = tr.Close()
				continue
			}

			if buf, bidErr := tr.ControlIn(reqBoardIDRead, 0, 0, 1, controlTimeoutMs); bidErr == nil && len(buf) > 0 {
				okCount++
				name := boardIDNames[buf[0]]
				if name == "" {
					name = "unknown"
				}
				t.Logf("board-id probe ok: req=0x%02x id=%d (%s)", reqBoardIDRead, buf[0], name)
			} else {
				if bidErr != nil {
					lastErr = bidErr
				}
				t.Logf("board-id probe failed: req=0x%02x gotLen=%d err=%v", reqBoardIDRead, len(buf), bidErr)
			}

			if buf, verErr := tr.ControlIn(reqVersionStringRead, 0, 0, 255, controlTimeoutMs); verErr == nil {
				okCount++
				t.Logf("version string probe ok: req=0x%02x version=%q", reqVersionStringRead, cleanVersionString(buf))
			} else {
				lastErr = verErr
				t.Logf("version string probe failed: req=0x%02x err=%v", reqVersionStringRead, verErr)
			}

			// Safe write probe: SET_TRANSCEIVER_MODE → off is idempotent
			// on an idle device and exercises the control-OUT path.
			if modeErr := tr.ControlOut(reqSetTransceiverMode, transceiverModeOff, 0, nil, controlTimeoutMs); modeErr == nil {
				okCount++
				t.Logf("transceiver-mode-off probe ok: req=0x%02x wValue=%d", reqSetTransceiverMode, transceiverModeOff)
			} else {
				lastErr = modeErr
				t.Logf("transceiver-mode-off probe failed: req=0x%02x wValue=%d err=%v", reqSetTransceiverMode, transceiverModeOff, modeErr)
			}

			_ = tr.ReleaseInterface(0)
			_ = tr.Close()
		}
	}

	if found == 0 {
		t.Fatalf("usb enumerate returned no HackRF descriptors (serial hint %q; last err: %v)", serialHint, lastErr)
	}
	if okCount == 0 {
		t.Fatalf("all raw control probes failed across %d device(s) (last err: %v)", found, lastErr)
	}
}

// openRealHackRF enumerates, applies the optional serial hint, and opens
// the selected device, returning the concrete *Device so tests can reach
// the HackRF-specific knobs (SetRFAmp, the Pro-only calls).
func openRealHackRF(t *testing.T) *Device {
	t.Helper()

	serialHint := strings.TrimSpace(os.Getenv(hackrfRealSerialEnv))

	drv := New(nil)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("Enumerate returned no devices (set %s only when hardware is attached)", hackrfRealEnv)
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
			t.Fatalf("no enumerated HackRF serial matched %q; found: %v", serialHint, collectSerials(infos))
		}
		idx = match
	}

	dev, err := drv.Open(idx)
	if err != nil {
		t.Fatalf("Open(%d): %v", idx, err)
	}
	hdev, ok := dev.(*Device)
	if !ok {
		_ = dev.Close()
		t.Fatalf("Open returned %T, want *hackrf.Device", dev)
	}
	return hdev
}

func requireRealHackRF(t *testing.T) {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(hackrfRealEnv))
	if v == "" || v == "0" || strings.EqualFold(v, "false") {
		t.Skipf("set %s=1 to run real HackRF hardware validation", hackrfRealEnv)
	}
	if testing.Short() {
		t.Skip("skipping real HackRF hardware validation in -short mode")
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
