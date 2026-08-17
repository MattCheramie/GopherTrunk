package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// A config whose only SDR source is a network one registered no driver at all:
// the gate that guards pool construction named sdr.devices, baseband.replay
// and sdr.rtl_tcp but not sdr.soapy_remote or sdr.ka9q_radio, and the drivers
// are registered INSIDE that block. The daemon came up quietly with no radio.
//
// Registration is what this asserts, not the pool handle: the pool is set to
// nil when it fails to open, which it always does here because nothing is
// listening at these addresses.
func TestRemoteOnlySDRConfigsRegisterTheirDriver(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		apply  func(*config.Config)
	}{
		{"soapy_remote only", "soapyremote", func(c *config.Config) {
			c.SDR.SoapyRemote = []config.SoapyRemoteConfig{{
				Addr: "127.0.0.1:1", Serial: "gate-soapy",
			}}
		}},
		{"ka9q_radio only", "ka9qradio", func(c *config.Config) {
			c.SDR.Ka9qRadio = []config.Ka9qRadioConfig{{
				Addr: "239.1.2.3:5006", SSRC: 162550, Serial: "gate-ka9q",
			}}
		}},
		{"rtl_tcp only", "rtltcp", func(c *config.Config) {
			c.SDR.RTLTCP = []config.RTLTCPConfig{{Addr: "127.0.0.1:1", Serial: "gate-rtltcp"}}
		}},
		{"sidecar only", "sidecar", func(c *config.Config) {
			c.SDR.Sidecar = []config.SidecarConfig{{
				Transport: "tcp", DataAddr: "127.0.0.1:1",
				SampleRateHz: 2_400_000, Serial: "gate-sidecar",
			}}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.API.HTTPAddr = ""
			cfg.SDR.Devices = nil
			tc.apply(&cfg)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			d, err := NewDaemon(cfg, "pool-gate-test", logger)
			if err != nil {
				t.Fatalf("NewDaemon: %v", err)
			}
			t.Cleanup(d.Close)

			var found bool
			for _, drv := range sdr.Drivers() {
				if drv.Name() == tc.driver {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("driver %q was never registered — a config whose only SDR "+
					"source is %s has no radio at all", tc.driver, tc.name)
			}
		})
	}
}
