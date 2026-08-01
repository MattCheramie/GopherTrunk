package main

import (
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/broadcast"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// buildBroadcastManager constructs the outbound call-streaming Manager
// from config. It returns (nil, nil) when no feed is enabled, so the
// daemon simply skips the subsystem. sampleRate is the recorder PCM
// rate, used to synthesise inter-call silence for live Icecast feeds.
// normCfg carries the per-call loudness-normalization settings; when it
// applies to the distributed copy, the Manager normalizes the MP3 in
// memory (leaving the on-disk WAV untouched).
func buildBroadcastManager(cfg config.BroadcastConfig, normCfg config.NormalizeConfig, sampleRate int, bus *events.Bus, loc *time.Location, log *slog.Logger) (*broadcast.Manager, error) {
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
			Loc:      loc,
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
	for _, f := range cfg.Webhook {
		if !f.Enabled {
			continue
		}
		b, err := broadcast.NewWebhook(broadcast.WebhookConfig{
			Name:         f.Name,
			URL:          f.URL,
			AuthHeader:   f.AuthHeader,
			IncludeAudio: f.IncludeAudio,
			Systems:      f.Systems,
			Loc:          loc,
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
	var normalize broadcast.NormalizeConfig
	if normCfg.AppliesToDistributed() {
		normalize = broadcast.NormalizeConfig{
			Enabled: true,
			Params: loudness.NormalizeParams{
				TargetLUFS:   normCfg.TargetLUFS,
				TruePeakDBTP: normCfg.TruePeakDBTP,
				MaxBoostDB:   normCfg.MaxBoostDB,
			}.WithDefaults(),
		}
	}
	return broadcast.NewManager(broadcast.Options{
		Bus:         bus,
		Log:         log,
		Backends:    backends,
		MinDuration: time.Duration(cfg.MinDurationMs) * time.Millisecond,
		Workers:     cfg.Workers,
		Normalize:   normalize,
	})
}

// buildGrantWebhooks constructs the push grant-webhook sinks from config —
// one per enabled feed. It returns nil when none are enabled, so the daemon
// simply skips the subsystem. Each sink subscribes to the bus at construction
// (so grants decoded before Run starts are not lost) and POSTs the GrantDTO
// schema per control-channel grant (issue #915 / #268).
func buildGrantWebhooks(cfg config.BroadcastConfig, bus *events.Bus, loc *time.Location, log *slog.Logger) ([]*broadcast.GrantWebhook, error) {
	var hooks []*broadcast.GrantWebhook
	for _, f := range cfg.GrantWebhook {
		if !f.Enabled {
			continue
		}
		h, err := broadcast.NewGrantWebhook(broadcast.GrantWebhookOptions{
			Bus: bus,
			Log: log,
			Config: broadcast.GrantWebhookConfig{
				Name:       f.Name,
				URL:        f.URL,
				AuthHeader: f.AuthHeader,
				Systems:    f.Systems,
				Loc:        loc,
			},
		})
		if err != nil {
			// Close any already-built sinks so their bus subscriptions
			// don't leak on a partial failure.
			for _, done := range hooks {
				_ = done.Close()
			}
			return nil, err
		}
		hooks = append(hooks, h)
	}
	return hooks, nil
}
