package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// getScannerSystems serves the cockpit through the real API server and
// returns GET /api/v1/scanner's systems array as raw JSON objects, so the
// test reads the wire format the web console reads.
func getScannerSystems(t *testing.T, d *Daemon, ck scannerCockpit) []map[string]any {
	t.Helper()
	srv, err := api.NewServer(api.ServerOptions{
		Addr:    "127.0.0.1:0",
		Bus:     d.bus,
		Scanner: ck,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); _ = srv.Close() })
	go func() { _ = srv.Run(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for srv.BoundAddr() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if srv.BoundAddr() == "" {
		t.Fatal("API server never bound")
	}
	resp, err := http.Get("http://" + srv.BoundAddr() + "/api/v1/scanner")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Systems []map[string]any `json:"systems"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Systems
}

// TestScannerEndpointListsWidebandSystems pins the "decode: —" report (PR
// #1384's notes): on a rig whose only SDR is a role: wideband dongle there is
// no hunter, so GET /api/v1/scanner returned "systems": [] and every decode
// chip read "decode: —" beside a control channel decoding cleanly. Each
// wideband system must be listed, named by its dongle.
func TestScannerEndpointListsWidebandSystems(t *testing.T) {
	d := buildWidebandDaemon(t,
		[]config.SystemConfig{{Name: "metro", Protocol: "p25", ControlChannels: []uint32{467_912_500}}},
		[]config.DeviceChannelConfig{{FrequencyHz: 467_912_500, System: "metro"}})
	if d.cchuntSup != nil {
		t.Fatal("a wideband-only rig built a control-channel hunter; this test needs none")
	}
	systems := getScannerSystems(t, d, d.newScannerCockpit())
	if len(systems) != 1 {
		t.Fatalf("GET /api/v1/scanner listed %d systems on a wideband-only rig, want 1: %v", len(systems), systems)
	}
	s := systems[0]
	if s["name"] != "metro" || s["protocol"] != "p25" || s["wideband_serial"] != "mock-00" {
		t.Errorf("system = %v", s)
	}
	// No IQ has flowed: listed, not decoding.
	if s["state"] != "listening" || s["attempted_freq_hz"] != float64(467_912_500) {
		t.Errorf("state = %v at %v, want listening on 467912500", s["state"], s["attempted_freq_hz"])
	}
}

// fakeChannelHealth stands in for a running wideband engine.
type fakeChannelHealth []widebandt2.ChannelHealth

func (f fakeChannelHealth) ChannelHealth() []widebandt2.ChannelHealth { return f }

// TestScannerEndpointCarriesWidebandDecodeHealth: a decoding wideband control
// channel reads like a hunter's lock on the wire, so the decode chip, the
// symbol chip (locked_freq_hz) and the level bar all have what they read.
func TestScannerEndpointCarriesWidebandDecodeHealth(t *testing.T) {
	d := buildWidebandDaemon(t,
		[]config.SystemConfig{{Name: "metro", Protocol: "p25", ControlChannels: []uint32{467_912_500}}},
		[]config.DeviceChannelConfig{{FrequencyHz: 467_912_500, System: "metro"}})
	since := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	ck := scannerCockpit{wideband: []channelHealthSource{fakeChannelHealth{{
		System: "metro", Protocol: "p25", FreqHz: 467_912_500, Serial: "mock-00",
		Decoding: true, DecodingSince: since, RecentDecodes: 1200,
		DecodeQuality: "clean", HasDecodeHealth: true,
		SignalDbFS: -38.5, HasSignal: true,
	}}}}
	systems := getScannerSystems(t, d, ck)
	if len(systems) != 1 {
		t.Fatalf("systems = %v", systems)
	}
	s := systems[0]
	want := map[string]any{
		"state":             "locked",
		"locked_freq_hz":    float64(467_912_500),
		"locked_at":         "2026-11-19T10:00:00Z",
		"decode_quality":    "clean",
		"has_decode_health": true,
		"signal_dbfs":       -38.5,
		"has_signal":        true,
		"wideband_serial":   "mock-00",
	}
	for k, v := range want {
		if s[k] != v {
			t.Errorf("%s = %v, want %v", k, s[k], v)
		}
	}
}

func TestMergeWidebandSystems(t *testing.T) {
	decoding := func(sys string, hz uint32, recent int64) widebandt2.ChannelHealth {
		return widebandt2.ChannelHealth{System: sys, Protocol: "p25", FreqHz: hz, Serial: "wb",
			Decoding: true, RecentDecodes: recent, DecodeQuality: "clean", HasDecodeHealth: true}
	}
	idle := func(sys string, hz uint32) widebandt2.ChannelHealth {
		return widebandt2.ChannelHealth{System: sys, Protocol: "p25", FreqHz: hz, Serial: "wb"}
	}

	t.Run("a system's busiest decoding channel represents it", func(t *testing.T) {
		rows := mergeWidebandSystems(nil, []widebandt2.ChannelHealth{
			idle("metro", 1), decoding("metro", 2, 50), decoding("metro", 3, 900),
		})
		if len(rows) != 1 || rows[0].LockedFreqHz != 3 || rows[0].State != "locked" {
			t.Fatalf("rows = %+v, want one locked row on channel 3", rows)
		}
	})
	t.Run("a hunter's locked row is kept", func(t *testing.T) {
		hunter := []api.SystemHuntStatusDTO{{Name: "metro", State: "locked", LockedFreqHz: 9}}
		rows := mergeWidebandSystems(hunter, []widebandt2.ChannelHealth{decoding("metro", 2, 50)})
		if len(rows) != 1 || rows[0].LockedFreqHz != 9 || rows[0].WidebandSerial != "" {
			t.Fatalf("rows = %+v, want the hunter's row unchanged", rows)
		}
	})
	t.Run("a hunting system the wideband side decodes reads locked", func(t *testing.T) {
		hunter := []api.SystemHuntStatusDTO{{Name: "metro", State: "hunting", AttemptedFreqHz: 9}}
		rows := mergeWidebandSystems(hunter, []widebandt2.ChannelHealth{decoding("metro", 2, 50)})
		if len(rows) != 1 || rows[0].State != "locked" || rows[0].LockedFreqHz != 2 {
			t.Fatalf("rows = %+v, want the wideband lock", rows)
		}
	})
	t.Run("a hunting system the wideband side does not decode keeps the hunter's row", func(t *testing.T) {
		hunter := []api.SystemHuntStatusDTO{{Name: "metro", State: "hunting", AttemptedFreqHz: 9}}
		rows := mergeWidebandSystems(hunter, []widebandt2.ChannelHealth{idle("metro", 2)})
		if len(rows) != 1 || rows[0].State != "hunting" || rows[0].AttemptedFreqHz != 9 {
			t.Fatalf("rows = %+v, want the hunter's row", rows)
		}
	})
	t.Run("an idle conventional system is camped", func(t *testing.T) {
		ch := idle("fire", 5)
		ch.Protocol, ch.CampsWhenIdle = "dmr-tier2", true
		rows := mergeWidebandSystems(nil, []widebandt2.ChannelHealth{ch})
		if len(rows) != 1 || rows[0].State != "camped" || rows[0].AttemptedFreqHz != 5 {
			t.Fatalf("rows = %+v, want camped on 5", rows)
		}
	})
	t.Run("systems keep their configured order", func(t *testing.T) {
		rows := mergeWidebandSystems(
			[]api.SystemHuntStatusDTO{{Name: "hunted", State: "locked"}},
			[]widebandt2.ChannelHealth{idle("b", 1), idle("a", 2), idle("b", 3)})
		if len(rows) != 3 || rows[0].Name != "hunted" || rows[1].Name != "b" || rows[2].Name != "a" {
			t.Fatalf("order = %+v", rows)
		}
	})
}
