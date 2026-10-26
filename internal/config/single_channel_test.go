package config

import (
	"strings"
	"testing"
)

func TestSingleChannelDecoderSerials(t *testing.T) {
	c := Default()
	c.APRS.Channels = []APRSChannelConfig{{Serial: "aprs-1", FrequencyHz: 144_390_000}, {Serial: " ", FrequencyHz: 1}}
	c.ADSB.Channels = []ADSBChannelConfig{{Serial: "adsb-1"}}
	c.Paging.Wideband = []PagingWidebandConfig{{Serial: "pager-wb"}}
	c.FleetSync.Channels = []FleetSyncChannelConfig{{Serial: "fs"}}
	c.LoRa.Channels = []LoRaChannelConfig{{Serial: "lora"}}
	got := c.SingleChannelDecoderSerials()
	for _, want := range []string{"aprs-1", "adsb-1", "pager-wb", "fs", "lora"} {
		if !got[want] {
			t.Errorf("serial %q missing from %v", want, got)
		}
	}
	if len(got) != 5 {
		t.Errorf("got %d serials, want 5 (blank skipped): %v", len(got), got)
	}
}

// TestDeviceSampleRateAllowedOnDecoderDongle (#1316): a dongle pinned to a
// single-channel decoder may run at its own sample_rate whatever its role.
// Before, only a role: voice SDR named by a scan-list channel could.
func TestDeviceSampleRateAllowedOnDecoderDongle(t *testing.T) {
	base := func() Config {
		c := Default()
		c.SDR.Devices = []DeviceConfig{{Serial: "54300989", Role: "auto", SampleRate: 240_000}}
		return c
	}
	c := base()
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "sample_rate") {
		t.Fatalf("un-pinned auto SDR with sample_rate must still be rejected; got %v", err)
	}
	c = base()
	c.APRS.Channels = []APRSChannelConfig{{Serial: "54300989", FrequencyHz: 144_390_000}}
	if err := c.Validate(); err != nil {
		t.Fatalf("APRS-pinned SDR with its own sample_rate rejected: %v", err)
	}
	c = base()
	c.SDR.Devices[0].Role = "control"
	c.ADSB.Channels = []ADSBChannelConfig{{Serial: "54300989"}}
	c.SDR.Devices[0].SampleRate = 2_000_000
	if err := c.Validate(); err != nil {
		t.Fatalf("ADS-B-pinned control SDR at 2 MS/s rejected: %v", err)
	}
}

// TestDedicatedRoleValidation: `role: dedicated` is accepted on a pinned
// serial and rejected on one no decoder names.
func TestDedicatedRoleValidation(t *testing.T) {
	c := Default()
	c.SDR.Devices = []DeviceConfig{{Serial: "54300989", Role: "dedicated"}}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "role: dedicated") {
		t.Fatalf("unpinned dedicated SDR accepted: %v", err)
	}
	c.APRS.Channels = []APRSChannelConfig{{Serial: "54300989", FrequencyHz: 144_390_000}}
	if err := c.Validate(); err != nil {
		t.Fatalf("pinned dedicated SDR rejected: %v", err)
	}
	c.SDR.RTLTCP = []RTLTCPConfig{{Addr: "127.0.0.1:1234", Serial: "remote-1", Role: "dedicated"}}
	c.ADSB.Channels = []ADSBChannelConfig{{Serial: "remote-1"}}
	if err := c.Validate(); err != nil {
		t.Fatalf("rtl_tcp role: dedicated rejected: %v", err)
	}
}
