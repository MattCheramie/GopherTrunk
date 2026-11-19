//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// pagingAirDriver enumerates one pagingAirFake.
type pagingAirDriver struct{ dev *pagingAirFake }

func (p *pagingAirDriver) Name() string { return "fake-paging-air" }
func (p *pagingAirDriver) Enumerate() ([]sdr.Info, error) {
	return []sdr.Info{{Driver: p.Name(), Index: 0, Serial: p.dev.serial}}, nil
}
func (p *pagingAirDriver) Open(int) (sdr.Device, error) { return p.dev, nil }

// TestDaemonDecodesWidebandPagingWithNoLiveView runs the whole daemon on the
// Discord reporter's config — a single SDR with role: dedicated, no trunking
// system, and one paging.wideband group of 929 MHz FLEX channels — and
// expects the page on the bus with nothing but the daemon running: no
// Spectrum view, no other consumer of the dongle. Before the fix the group
// was spawned, tuned the SDR and then waited on a broker subscription that
// nothing fed, so the dongle's stream was never opened.
func TestDaemonDecodesWidebandPagingWithNoLiveView(t *testing.T) {
	const serial = "pager-air-sdr"
	dev := &pagingAirFake{serial: serial, rateHz: pagingTestRateHz,
		txHz: pagingTestTxHz, syms: pagingTestFrame(t)}
	sdr.Register(&pagingAirDriver{dev: dev})

	cfg := config.Default()
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Trunking.Systems = nil
	cfg.SDR.SampleRate = pagingTestRateHz
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: serial, Role: "dedicated"}}
	group := pagingTestGroup()
	group.Serial = serial
	cfg.Paging.Wideband = []config.PagingWidebandConfig{group}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config: %v", err)
	}

	d, err := NewDaemon(cfg, "integration-paging-wideband", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	pages := d.Bus().Subscribe()
	defer pages.Close()

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-runErrCh:
		case <-time.After(5 * time.Second):
			t.Error("daemon did not stop within 5s")
		}
	})
	waitReachable(t, "http://"+cfg.API.HTTPAddr+"/api/v1/health", 3*time.Second)

	awaitTestPage(t, pages, 15*time.Second,
		"the daemon is running a paging.wideband group but its SDR was never streamed")
	if got := dev.streams.Load(); got != 1 {
		t.Fatalf("StreamIQ sessions = %d, want 1", got)
	}
	if e := d.pool.FindBySerial(serial); e == nil || e.Role != sdr.RoleDedicated {
		t.Fatalf("paging SDR entry = %+v, want role dedicated", e)
	}
}
