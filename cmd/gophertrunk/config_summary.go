package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25phase2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// logConfigSummary emits a one-time, human-readable snapshot of the effective
// decode configuration at startup: a global line (SDR/recording levers) plus
// one line per trunking system listing every decode "enhancement" relevant to
// that system's protocol and whether it is actually on.
//
// The point is answerability: most per-system enhancement toggles are string
// tri-states where "" means "use the per-protocol default", so an operator
// reading config.yaml cannot tell whether, say, the TETRA traffic LMS
// equalizer is running. This summary resolves each toggle through the SAME
// parser the decode pipeline uses (tetra.ParseTrafficLMS,
// p25phase2rx.ParseEqualizer, …) rather than testing the raw string for
// emptiness, so the reported state matches what the receiver was actually
// built with — including defaults.
func (d *Daemon) logConfigSummary() {
	d.log.Info("daemon: config summary", summarizeGlobal(d.cfg)...)
	for _, s := range d.systems {
		d.log.Info("daemon: system config",
			"name", s.Name,
			"protocol", s.Protocol.String(),
			"control_channels", len(s.ControlChannels),
			"enhancements", systemEnhancements(s),
		)
	}
}

// summarizeGlobal builds the slog key/value attrs for the daemon-wide decode
// and recording levers that aren't per-system.
func summarizeGlobal(cfg config.Config) []any {
	attrs := []any{
		"systems", len(cfg.Trunking.Systems),
		"sample_rate_hz", cfg.SDR.SampleRate,
	}

	// input_sample_rate is a pre-decimation stage: the front end runs at the
	// native rate and the daemon integer-decimates to sample_rate. Report the
	// factor when it is actually engaged, "off" otherwise.
	if cfg.SDR.SampleRate != 0 &&
		cfg.SDR.InputSampleRate != 0 &&
		cfg.SDR.InputSampleRate != cfg.SDR.SampleRate {
		attrs = append(attrs,
			"input_sample_rate_hz", cfg.SDR.InputSampleRate,
			"predecimation_factor", cfg.SDR.InputSampleRate/cfg.SDR.SampleRate,
		)
	} else {
		attrs = append(attrs, "input_predecimation", "off")
	}

	attrs = append(attrs, "autotune", cfg.SDR.Autotune)

	// Spatial-diversity (MRC) combiner is per soapy_remote endpoint. List the
	// endpoints that enable it so a multi-radio config is unambiguous.
	var diversity []string
	for _, sr := range cfg.SDR.SoapyRemote {
		mode := strings.ToLower(strings.TrimSpace(sr.Diversity))
		if mode == "" || mode == "none" {
			continue
		}
		label := strings.TrimSpace(sr.Serial)
		if label == "" {
			label = strings.TrimSpace(sr.Addr)
		}
		diversity = append(diversity, label+":"+mode)
	}
	if len(diversity) > 0 {
		sort.Strings(diversity)
		attrs = append(attrs, "diversity", strings.Join(diversity, ","))
	} else {
		attrs = append(attrs, "diversity", "off")
	}

	// Recording / voice audio-shaping enhancements. spec_amplitude_enhance is a
	// tri-state pointer that defaults ON when unset.
	attrs = append(attrs,
		"rec_enhance", cfg.Recordings.Enhance.Enabled,
		"rec_normalize", cfg.Recordings.Normalize.Enabled,
		"rec_fm_equalizer", cfg.Recordings.Equalizer.Enabled,
		"rec_warm_dmr_audio", cfg.Recordings.WarmDMRAudio,
		"rec_spec_amplitude_enhance",
		cfg.Recordings.SpecAmplitudeEnhance == nil || *cfg.Recordings.SpecAmplitudeEnhance,
	)

	return attrs
}

// systemEnhancements renders the space-separated "key=value" list of decode
// enhancements relevant to a system's protocol, with each tri-state resolved
// through its real parser so the reported value is the effective one. Returns
// "(none)" for protocols with no operator-tunable decode enhancements.
func systemEnhancements(s trunking.System) string {
	var parts []string
	add := func(k, v string) { parts = append(parts, k+"="+v) }

	switch s.Protocol {
	case trunking.ProtocolP25:
		demod, _ := p25phase1rx.ParseDemodMode(s.P25Phase1DemodMode)
		add("demod", demod.String())
		soft, _ := p25phase1rx.ParseSoftDecision(s.P25Phase1SoftDecision)
		add("soft_decision", onOff(soft))

	case trunking.ProtocolP25Phase2:
		soft, _ := p25phase2rx.ParseSoftDecision(s.P25Phase2SoftDecision)
		add("soft_decision", onOff(soft))
		eq, _ := p25phase2rx.ParseEqualizer(s.P25Phase2Equalizer)
		add("cma_equalizer", onOff(eq))
		dc, _ := p25phase2rx.ParseDCBlock(s.P25Phase2DCBlock)
		add("dc_block", onOff(dc))
		clk, _ := p25phase2rx.ParseClockMode(s.P25Phase2ClockMode)
		add("clock", p25Phase2ClockLabel(clk))

	case trunking.ProtocolTETRA, trunking.ProtocolTETRADMO:
		cc, _ := tetra.ParseChannelCoding(s.TETRAChannelCoding)
		add("channel_coding", onOff(cc == tetra.ChannelCodingOn))
		clk, _ := tetrarx.ParseClockMode(s.TETRAClockMode)
		add("clock", tetraClockLabel(clk))
		// The receiver's blind CMA equalizer is always wired on the TETRA
		// pipeline (it roughly doubles CRC-clean yield on ISI-limited
		// captures); the trained LMS equalizer is the additional opt-in lever.
		add("cma_equalizer", "on")
		lms, _ := tetra.ParseTrafficLMS(s.TETRATrafficLMS)
		add("traffic_lms", onOff(lms))

	case trunking.ProtocolDMR, trunking.ProtocolDMRTier2, trunking.ProtocolDMRTier1:
		add("interleaved_voice", onOff(s.DMRInterleavedVoice))
		if s.DMRColorCode != nil {
			add("color_code", strconv.Itoa(int(*s.DMRColorCode)))
		}

	case trunking.ProtocolNXDN:
		soft, _ := nxdnrx.ParseSoftDecision(s.NXDNSoftDecision)
		add("soft_decision", onOff(soft))
		afc, _ := nxdnrx.ParseAFC(s.NXDNAFC)
		add("afc", onOff(afc))
	}

	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " ")
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func p25Phase2ClockLabel(m p25phase2rx.ClockMode) string {
	if m == p25phase2rx.ClockGardner {
		return "gardner"
	}
	return "naive"
}

func tetraClockLabel(m tetrarx.ClockMode) string {
	if m == tetrarx.ClockGardner {
		return "gardner"
	}
	return "naive"
}
