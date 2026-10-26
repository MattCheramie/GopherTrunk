package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/narrowband"
)

// TestSingleChannelFrontEndAtRTLRate pins the #1316 wiring numbers for the
// common RTL-SDR rate: a 50:1 decimation to 48 kHz, the clip-safe LO
// offset the conventional scanner also uses, and the LO programmed that
// far BELOW the channel.
func TestSingleChannelFrontEndAtRTLRate(t *testing.T) {
	fe := newSingleChannelFrontEnd(2_400_000)
	if fe.OutRate() != 48_000 || fe.Decimation() != 50 {
		t.Fatalf("2.4 MS/s front end: out=%d decim=%d, want 48000 / 50", fe.OutRate(), fe.Decimation())
	}
	if fe.OffsetHz() != pickClipSafeLOOffsetHz(2_400_000) || fe.OffsetHz() == 0 {
		t.Fatalf("offset = %v, want the clip-safe pick %v", fe.OffsetHz(), pickClipSafeLOOffsetHz(2_400_000))
	}
	const aprs = 144_390_000
	lo := singleChannelLOHz(aprs, fe)
	if lo != aprs-uint32(fe.OffsetHz()) {
		t.Errorf("LO = %d, want %d (channel − offset)", lo, aprs-uint32(fe.OffsetHz()))
	}
	// A rate with no room for an offset tunes on-channel, as before.
	small := newSingleChannelFrontEnd(250_000)
	if small.OffsetHz() != 0 || singleChannelLOHz(aprs, small) != aprs {
		t.Errorf("250 kS/s: offset %v, LO %d; want on-channel", small.OffsetHz(), singleChannelLOHz(aprs, small))
	}
	if singleChannelLOHz(aprs, nil) != aprs {
		t.Errorf("nil front end must tune on-channel")
	}
	var nilFE *narrowband.FrontEnd
	_ = nilFE
}

// TestDaemonBuildsSingleChannelDecodersBehindFrontEnd: every narrowband
// single-channel decoder the daemon constructs gets a channel front end
// and is built for ITS output rate, never the SDR's. Before #1316 each
// receiver took cfg.SDR.SampleRate and demodulated the whole 2.4 MHz.
func TestDaemonBuildsSingleChannelDecodersBehindFrontEnd(t *testing.T) {
	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.SDR.Devices = nil
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.RTLTCP = []config.RTLTCPConfig{{Addr: "127.0.0.1:1", Serial: "rig"}}
	cfg.APRS.Channels = []config.APRSChannelConfig{{Serial: "rig", FrequencyHz: 144_390_000}}
	cfg.AIS.Channels = []config.AISChannelConfig{{Serial: "rig", FrequencyHz: 161_975_000}}
	cfg.DSC.Channels = []config.DSCChannelConfig{{Serial: "rig", FrequencyHz: 156_525_000}}
	cfg.MDC1200.Channels = []config.MDC1200ChannelConfig{{Serial: "rig", FrequencyHz: 447_100_000}}
	cfg.FleetSync.Channels = []config.FleetSyncChannelConfig{{Serial: "rig", FrequencyHz: 462_562_500}}
	cfg.Paging.POCSAG = []config.PagingPOCSAGConfig{{Serial: "rig", FrequencyHz: 466_075_000}}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "single-channel-test", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	check := func(name string, fe *narrowband.FrontEnd) {
		if fe == nil {
			t.Errorf("%s: no channel front end", name)
			return
		}
		if fe.OutRate() != 48_000 {
			t.Errorf("%s: front end rate %d, want 48000", name, fe.OutRate())
		}
	}
	if len(d.aprsSpecs) != 1 || d.aprsReceivers[0] == nil {
		t.Fatalf("aprs receiver not built: %+v", d.aprsSpecs)
	}
	check("aprs", d.aprsSpecs[0].fe)
	check("ais", d.aisSpecs[0].fe)
	check("dsc", d.dscSpecs[0].fe)
	check("mdc1200", d.mdc1200Specs[0].fe)
	check("fleetsync", d.fleetsyncSpecs[0].fe)
	check("pocsag", d.pocsagSpecs[0].fe)
}
