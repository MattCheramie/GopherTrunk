package main

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/broadcast"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// buildBroadcastManager constructs the outbound call-streaming Manager
// from config. It returns (nil, nil) when no feed is enabled, so the
// daemon simply skips the subsystem. sampleRate is the recorder PCM
// rate, used to synthesise inter-call silence for live Icecast feeds.
func buildBroadcastManager(cfg config.BroadcastConfig, sampleRate int, bus *events.Bus, log *slog.Logger) (*broadcast.Manager, error) {
	var backends []broadcast.Backend

	for _, f := range cfg.Broadcastify {
		if !f.Enabled {
			continue
		}
		b, err := broadcast.NewBroadcastify(broadcast.BroadcastifyConfig{
			Name:     f.Name,
			APIKey:   f.APIKey,
			SystemID: f.SystemID,
			Systems:  f.Systems,
		}, nil)
		if err != nil {
			return nil, err
		}
		backends = append(backends, b)
	}
	for _, f := range cfg.RdioScanner {
		if !f.Enabled {
			continue
		}
		b, err := broadcast.NewRdioScanner(broadcast.RdioScannerConfig{
			Name:     f.Name,
			URL:      f.URL,
			APIKey:   f.APIKey,
			SystemID: f.SystemID,
			Systems:  f.Systems,
		}, nil)
		if err != nil {
			return nil, err
		}
		backends = append(backends, b)
	}
	for _, f := range cfg.OpenMHz {
		if !f.Enabled {
			continue
		}
		b, err := broadcast.NewOpenMHz(broadcast.OpenMHzConfig{
			Name:      f.Name,
			APIKey:    f.APIKey,
			ShortName: f.ShortName,
			Systems:   f.Systems,
		}, nil)
		if err != nil {
			return nil, err
		}
		backends = append(backends, b)
	}
	for _, f := range cfg.Icecast {
		if !f.Enabled {
			continue
		}
		b, err := broadcast.NewIcecast(broadcast.IcecastConfig{
			Name:       f.Name,
			Host:       f.Host,
			Port:       f.Port,
			Mount:      f.Mount,
			Username:   f.Username,
			Password:   f.Password,
			StreamName: f.StreamName,
			SampleRate: sampleRate,
			Systems:    f.Systems,
		}, log)
		if err != nil {
			return nil, err
		}
		backends = append(backends, b)
	}

	if len(backends) == 0 {
		return nil, nil
	}
	return broadcast.NewManager(broadcast.Options{
		Bus:         bus,
		Log:         log,
		Backends:    backends,
		MinDuration: time.Duration(cfg.MinDurationMs) * time.Millisecond,
		Workers:     cfg.Workers,
	})
}
