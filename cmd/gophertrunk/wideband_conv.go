package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional/tonecode"
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
	// widebandSerial is the role: wideband dongle the channel's tap is on.
	widebandSerial string
	// tone is the channel's CTCSS / DCS gate, which is what tells logical
	// channels sharing one frequency apart.
	tone tonecode.Gate
	// nac names a p25 channel's NAC filter ("293", or "F7E" for receive
	// any), "" when unset; on a shared frequency it is what tells the
	// channels apart.
	nac string
}

// wbConvSerial is the virtual device serial a wideband conventional channel's
// calls run under: one per channel, so the engine's synthetic calls (keyed by
// serial) and the composer's voice chains never collide on the shared dongle.
func wbConvSerial(widebandSerial string, freqHz uint32) string {
	return fmt.Sprintf("wb:%s:conv:%d", widebandSerial, freqHz)
}

// wbConvSharedSerial is the serial of one of several logical channels on a
// shared frequency (issue #1288 §5): the frequency's serial plus the tone
// that tells this channel apart (validation guarantees each has a distinct
// one), e.g. "wb:WB1:conv:460100000:ctcss100.0". It depends only on the
// channel's own config, so it is stable across restarts and does not change
// when another channel joins or leaves the frequency's list. A channel alone
// on its frequency keeps wbConvSerial.
func wbConvSharedSerial(widebandSerial string, freqHz uint32, tone tonecode.Gate) string {
	return wbConvSerial(widebandSerial, freqHz) + ":" + tone.ID()
}

// wbConvSharedP25Serial is wbConvSharedSerial for a p25 channel on a shared
// frequency, told apart by its NAC: "wb:WB1:conv:460100000:nac293".
func wbConvSharedP25Serial(widebandSerial string, freqHz uint32, nac uint16) string {
	return fmt.Sprintf("%s:nac%03x", wbConvSerial(widebandSerial, freqHz), nac)
}

// convChannelFromConfig is the conventional.Channel for a configured channel
// under talkgroup ID tgid, without the per-SDR gain only the scan-list
// scanner applies. A p25 channel's nac and demod come through for both the
// scan list and a wideband tap (issue #1288); config validation already
// rejected a malformed nac.
func convChannelFromConfig(ch config.ConvChannelConfig, tgid uint32) conventional.Channel {
	nac, nacErr := conventional.ParseNAC(ch.NAC)
	return conventional.Channel{
		NAC:                nac,
		NACSet:             strings.TrimSpace(ch.NAC) != "" && nacErr == nil,
		P25Phase1DemodMode: ch.P25Phase1DemodMode,
		AudioGainDB:        ch.AudioGainDB,
		Label:              ch.Label,
		FrequencyHz:        ch.FrequencyHz,
		Mode:               ch.Mode,
		SquelchDbFS:        ch.SquelchDbFS,
		SquelchCNDb:        ch.SquelchCNDb,
		Hangtime:           msToDuration(ch.HangtimeMs, 1500*time.Millisecond),
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
		perFreq := make(map[uint32]int, len(devCfg.Conventional))
		for _, ch := range devCfg.Conventional {
			perFreq[ch.FrequencyHz]++
		}
		for _, ch := range devCfg.Conventional {
			tgid := ch.TalkgroupID
			if tgid == 0 {
				tgid = config.WidebandConvPositionalBase | uint32(k)
			}
			k++
			tone := convToneGate(ch.Tone)
			conv := convChannelFromConfig(ch, tgid)
			vserial := wbConvSerial(serial, ch.FrequencyHz)
			if perFreq[ch.FrequencyHz] > 1 {
				// One engine tap carries the frequency (see
				// widebandConvTapChannels); each logical channel gets
				// its own source on it, and its own tone gate (a p25
				// channel's NAC gate) decides which transmissions it
				// records.
				if conv.Mode == conventional.ModeP25 {
					vserial = wbConvSharedP25Serial(serial, ch.FrequencyHz, conv.NAC)
				} else {
					vserial = wbConvSharedSerial(serial, ch.FrequencyHz, tone)
				}
			}
			nacName := ""
			if conv.NACSet {
				nacName = fmt.Sprintf("%03X", conv.NAC)
			}
			d.wbConvChannels = append(d.wbConvChannels, wbConvChannel{
				source:         widebandt2.NewChannelVoiceSource(get, ch.FrequencyHz, vserial),
				channel:        conv,
				widebandSerial: serial,
				tone:           tone,
				nac:            nacName,
			})
		}
		if n := len(devCfg.Conventional); n > 0 {
			log.Info("daemon: wideband: continuous conventional channels registered — each records its own calls in parallel",
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

// wbConvStatusSources pairs each wideband conventional scanner with its
// channel's dongle and talkgroup for the scanner cockpit. buildWidebandConvScanners
// builds wbConvScans from wbConvChannels in order, so index i is channel i.
func (d *Daemon) wbConvStatusSources() []wbConvStatusSource {
	out := make([]wbConvStatusSource, 0, len(d.wbConvScans))
	for i, cs := range d.wbConvScans {
		if i >= len(d.wbConvChannels) {
			break
		}
		out = append(out, wbConvStatusSource{
			scanner:        cs,
			source:         d.wbConvChannels[i].source,
			widebandSerial: d.wbConvChannels[i].widebandSerial,
			talkgroupID:    d.wbConvChannels[i].channel.TalkgroupID,
			tone:           d.wbConvChannels[i].tone.String(),
			nac:            d.wbConvChannels[i].nac,
		})
	}
	return out
}

// convToneGate is a configured tone squelch in tonecode's terms.
func convToneGate(t config.ConvToneConfig) tonecode.Gate {
	return tonecode.Gate{Mode: t.Mode, CTCSSHz: t.CTCSSHz, DCSCode: t.DCSCode, DCSPolarity: t.DCSPolarity}
}

// widebandConvTag is the engine's protocol tag for a conventional tap of mode
// m: "" for the analog modes (the engine names them fm-conv / am-conv), else
// "<mode>-conv" ("p25-conv", "dmr-conv", …), which the startup log and the
// channel-power events carry.
func widebandConvTag(m string) string {
	if conventional.IsDigitalMode(m) {
		return m + "-conv"
	}
	return ""
}

// widebandConvTapChannels is the engine channel list for a device's
// conventional entries: one tap per frequency, however many logical channels
// share it (issue #1288 §5), so the carrier is extracted once. Logical
// channels on one frequency share a mode (validation guarantees it).
func widebandConvTapChannels(chans []config.ConvChannelConfig) []widebandt2.ChannelConfig {
	out := make([]widebandt2.ChannelConfig, 0, len(chans))
	seen := make(map[uint32]bool, len(chans))
	for _, ch := range chans {
		if seen[ch.FrequencyHz] {
			continue
		}
		seen[ch.FrequencyHz] = true
		out = append(out, widebandt2.ChannelConfig{
			FrequencyHz: ch.FrequencyHz, Conventional: true, ConventionalAM: ch.Mode == "am",
			ConventionalTag: widebandConvTag(ch.Mode),
		})
	}
	return out
}
