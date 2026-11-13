package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoSDRHintNamesTheIIOFlag pins #1230: the reporter put a PlutoSky in
// config.yaml's sdr.iio, ran `gophertrunk power`, and got only "no SDRs found".
// The one-shot tools read no config, so the error now names the flag that
// reaches each network radio the discovered config lists. The config is the
// reporter's own (sdr.iio at 192.168.1.179, no systems: the daemon's full
// validation would reject it, the hint must not).
func TestNoSDRHintNamesTheIIOFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `sdr:
  # sample_rate: 2_400_000
  sample_rate: 4_800_000
  iio:
    - addr: "192.168.1.179"     # the PlutoSky's address
      serial: "plutosky"
      role: control
      gain: "auto"
  rtl_tcp:
    - addr: "10.0.0.5:1234"
  soapy_remote:
    - addr: "10.0.0.9:55132"
`)
	noSDR := errors.New("capture: no SDRs found")
	got := noSDRWithConfigHintAt("power", path, noSDR).Error()
	for _, want := range []string{
		"capture: no SDRs found", // stays the prefix: wmbus matches on it
		"`power` reads no config file",
		"gophertrunk power -iio 192.168.1.179",
		"gophertrunk power -rtltcp 10.0.0.5:1234",
		"10.0.0.9:55132",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hint missing %q:\n%s", want, got)
		}
	}
	if !strings.HasPrefix(got, "capture: no SDRs found") {
		t.Errorf("original message must stay the prefix:\n%s", got)
	}
	if !errors.Is(noSDRWithConfigHintAt("power", path, noSDR), noSDR) {
		t.Error("hinted error must still wrap the original")
	}
}

// TestNoSDRHintStaysQuietWithoutNetworkRadios: no config, a config with only
// local devices, an unreadable file, and errors other than "no SDRs found" all
// leave the error untouched.
func TestNoSDRHintStaysQuietWithoutNetworkRadios(t *testing.T) {
	noSDR := errors.New("capture: no SDRs found")
	dir := t.TempDir()
	local := filepath.Join(dir, "local.yaml")
	writeFile(t, local, "sdr:\n  sample_rate: 2_400_000\n  devices:\n    - serial: \"00000001\"\n")
	bad := filepath.Join(dir, "bad.yaml")
	writeFile(t, bad, "sdr: [unbalanced\n")
	for _, path := range []string{"", local, bad, filepath.Join(dir, "missing.yaml")} {
		if got := noSDRWithConfigHintAt("power", path, noSDR); got != noSDR {
			t.Errorf("path %q: error changed to %q", path, got)
		}
	}
	withIIO := filepath.Join(dir, "iio.yaml")
	writeFile(t, withIIO, "sdr:\n  iio:\n    - addr: 192.168.2.1\n")
	other := errors.New("set gain: boom")
	if got := noSDRWithConfigHintAt("power", withIIO, other); got != other {
		t.Errorf("unrelated error changed to %q", got)
	}
	if got := noSDRWithConfigHintAt("power", withIIO, nil); got != nil {
		t.Errorf("nil error became %v", got)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
