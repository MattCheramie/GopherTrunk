package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// Continuous conventional FM and AM on a wideband dongle (issue #1288).
//
// A scan-list scanner owns a whole SDR and dwells on one channel at a time,
// so a call on one channel hides every other channel until it ends. A
// role: wideband dongle already receives its whole IQ band at once, so each
// sdr.devices[].conventional entry gets its own channelizer tap (a tap-only
// widebandt2 channel) and its own single-channel conventional.Scanner reading
// that tap through a widebandt2.ChannelVoiceSource. Every channel therefore
// has its own squelch, tone gate, hangtime and synthetic call under its own
// virtual serial, and overlapping transmissions record as separate calls.
// The composer follows each call on the same source, so the FM chain hears
// exactly the IQ the squelch measured.

// wbConvTapRateHz is the rate a conventional channel's tap delivers: the
// widebandt2 per-channel rate (48 kHz) the scanner's squelch, tone gates and
// data decoders run at.
const wbConvTapRateHz = 48_000

// wbConvScanWindow is a wideband conventional scanner's dwell window. With
// one channel the window only decides when the scanner re-opens its IQ
// subscription, and every re-open drops the chunks in between, so the
// scanner keeps one subscription open until squelch breaks.
const wbConvScanWindow = 24 * time.Hour

// wbConvChannel is one sdr.devices[].conventional entry: its source (the
// tap's IQ under a virtual serial) and its scan channel.
type wbConvChannel struct {
	source  *widebandt2.ChannelVoiceSource
	channel conventional.Channel
}

// wbConvSerial is the virtual device serial a wideband conventional channel's
// calls run under: one per channel, so the engine's synthetic calls (keyed by
// serial) and the composer's voice chains never collide on the shared dongle.
func wbConvSerial(widebandSerial string, freqHz uint32) string {
	return fmt.Sprintf("wb:%s:conv:%d", widebandSerial, freqHz)
}

// convChannelFromConfig is the conventional.Channel for a configured channel
// under talkgroup ID tgid, without the per-SDR settings (gain, NAC, demod)
// only the scan-list scanner applies.
func convChannelFromConfig(ch config.ConvChannelConfig, tgid uint32) conventional.Channel {
	return conventional.Channel{
		AudioGainDB: ch.AudioGainDB,
		Label:       ch.Label,
		FrequencyHz: ch.FrequencyHz,
		Mode:        ch.Mode,
		SquelchDbFS: ch.SquelchDbFS,
		SquelchCNDb: ch.SquelchCNDb,
		Hangtime:    msToDuration(ch.HangtimeMs, 1500*time.Millisecond),
		// 0 => let the scanner apply its own defaults
		// (50 ms debounce / 3 dB hysteresis) in one place.
		ActivityDebounce:    msToDuration(ch.ActivityDebounceMs, 0),
		SquelchHysteresisDb: ch.SquelchHysteresisDb,
		Priority:            ch.Priority,
		TalkgroupID:         tgid,
		Tone: conventional.ToneConfig{
			Mode:        ch.Tone.Mode,
			CTCSSHz:     ch.Tone.CTCSSHz,
			DCSCode:     ch.Tone.DCSCode,
			DCSPolarity: ch.Tone.DCSPolarity,
		},
		Decoders: ch.Decoders,
	}
}

// buildWidebandConvSources registers one source per sdr.devices[].conventional
// entry on every role: wideband dongle in the pool. Called before the
// composer is built, so its device map resolves the virtual serials. The
// sources stay out of the trunking voice pool: they serve their channel's own
// calls, never a grant.
func (d *Daemon) buildWidebandConvSources(cfg config.Config, log *slog.Logger) {
	if d.pool == nil {
		return
	}
	k := 0 // positional talkgroup index across every wideband device
	for _, devCfg := range cfg.SDR.Devices {
		if devCfg.Role != "wideband" {
			continue
		}
		entry := d.pool.FindBySerial(devCfg.Serial)
		if entry == nil {
			k += len(devCfg.Conventional) // keep the IDs config validation checked
			continue
		}
		serial := entry.Info.Serial
		get := func() *widebandt2.Engine { return d.widebandEngineForSerial(serial) }
		for _, ch := range devCfg.Conventional {
			tgid := ch.TalkgroupID
			if tgid == 0 {
				tgid = config.WidebandConvPositionalBase | uint32(k)
			}
			k++
			d.wbConvChannels = append(d.wbConvChannels, wbConvChannel{
				source:  widebandt2.NewChannelVoiceSource(get, ch.FrequencyHz, wbConvSerial(serial, ch.FrequencyHz)),
				channel: convChannelFromConfig(ch, tgid),
			})
		}
		if n := len(devCfg.Conventional); n > 0 {
			log.Info("daemon: wideband: continuous conventional FM/AM channels registered — each records its own calls in parallel",
				"wideband_serial", serial, "channels", n)
		}
	}
}

// newWidebandConvScanner is the single-channel scanner for one wideband
// conventional channel. src is the channel's tap: its SetCenterFreq is a
// no-op, so the scanner never retunes the dongle.
func newWidebandConvScanner(src *widebandt2.ChannelVoiceSource, ch conventional.Channel,
	eng conventional.Engine, rec conventional.Recorder, decoders conventional.DataDecoderFactory,
	log *slog.Logger) (*conventional.Scanner, error) {
	return conventional.New(conventional.Options{
		Log:                log,
		Tuner:              src,
		IQ:                 src,
		Engine:             eng,
		Recorder:           rec,
		DeviceSerial:       src.Serial(),
		SystemName:         "scanner",
		Channels:           []conventional.Channel{ch},
		SampleRateHz:       wbConvTapRateHz,
		MinDwellPerChannel: wbConvScanWindow,
		DataDecoders:       decoders,
	})
}

// buildWidebandConvScanners builds the scanners for the sources
// buildWidebandConvSources registered.
func (d *Daemon) buildWidebandConvScanners(rec conventional.Recorder, log *slog.Logger) error {
	for _, c := range d.wbConvChannels {
		cs, err := newWidebandConvScanner(c.source, c.channel, d.engine, rec,
			convDataDecoderFactory(d.bus, c.source.Serial(), log), log)
		if err != nil {
			return fmt.Errorf("daemon: wideband conventional channel %d Hz: %w", c.channel.FrequencyHz, err)
		}
		d.wbConvScans = append(d.wbConvScans, cs)
	}
	return nil
}
