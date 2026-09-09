package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestLogConfigSummaryEmitsPerSystemLine proves the wiring: logConfigSummary
// emits the global line plus one "system config" line carrying the effective
// enhancements string per system.
func TestLogConfigSummaryEmitsPerSystemLine(t *testing.T) {
	var buf bytes.Buffer
	d := &Daemon{
		log: slog.New(slog.NewTextHandler(&buf, nil)),
		systems: []trunking.System{
			{
				Name: "MetroTETRA", Protocol: trunking.ProtocolTETRA,
				ControlChannels: []uint32{450_000_000},
				TETRATrafficLMS: "on",
			},
		},
	}
	d.cfg.SDR.SampleRate = 2_400_000

	d.logConfigSummary()

	out := buf.String()
	for _, want := range []string{
		"daemon: config summary",
		"daemon: system config",
		"MetroTETRA",
		"traffic_lms=on",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q\n---\n%s", want, out)
		}
	}
}

// TestWarnRecordingDestination pins the "autorecord not working" diagnosability
// fix: when recordings.dir is empty the daemon records no .wav audio, and that
// must be surfaced — loudly (WARN) when a raw-IQ capture subsystem is enabled
// (the exact footgun: voice_iq_debug/auto_record write IQ under their own dirs
// while no audio lands and nothing said why), and plainly (INFO) for a
// deliberate decode-only setup. A configured recordings.dir stays quiet.
func TestWarnRecordingDestination(t *testing.T) {
	run := func(cfg config.Config) string {
		var buf bytes.Buffer
		d := &Daemon{log: slog.New(slog.NewTextHandler(&buf, nil)), cfg: cfg}
		d.warnRecordingDestination()
		return buf.String()
	}

	t.Run("dir set: no warning", func(t *testing.T) {
		cfg := config.Config{}
		cfg.Recordings.Dir = "/rec"
		cfg.Baseband.VoiceIQDebug.Enabled = true
		if out := run(cfg); out != "" {
			t.Errorf("expected no log when recordings.dir is set, got:\n%s", out)
		}
	})

	t.Run("dir empty + voice_iq_debug on: WARN names the fix", func(t *testing.T) {
		cfg := config.Config{}
		cfg.Baseband.VoiceIQDebug.Enabled = true
		out := run(cfg)
		if !strings.Contains(out, "level=WARN") {
			t.Errorf("expected WARN, got:\n%s", out)
		}
		for _, want := range []string{"recordings.dir", "voice_iq_debug", "not"} {
			if !strings.Contains(out, want) {
				t.Errorf("WARN missing %q\n---\n%s", want, out)
			}
		}
	})

	t.Run("dir empty + auto_record on: WARN", func(t *testing.T) {
		cfg := config.Config{}
		cfg.Baseband.AutoRecord.Enabled = true
		if out := run(cfg); !strings.Contains(out, "level=WARN") {
			t.Errorf("expected WARN when auto_record enabled + no recordings.dir, got:\n%s", out)
		}
	})

	t.Run("dir empty, no capture: plain INFO decode-only", func(t *testing.T) {
		out := run(config.Config{})
		if strings.Contains(out, "level=WARN") {
			t.Errorf("deliberate decode-only should not WARN, got:\n%s", out)
		}
		if !strings.Contains(out, "decode-only") {
			t.Errorf("expected decode-only INFO, got:\n%s", out)
		}
	})
}

// TestSystemEnhancementsResolvesEffectiveState is the failing-first regression
// for the startup config summary: the effective on/off it reports must come
// from the receiver parsers (so "" means the per-protocol DEFAULT), not from
// testing the raw config string for emptiness. The two TETRA systems below
// differ only in tetra_traffic_lms — the exact "is LMS on?" question the
// summary exists to answer — and an empty string must read "off" while the
// DMR default (interleaved_voice unset ⇒ on for Tier III) must read "on".
func TestSystemEnhancementsResolvesEffectiveState(t *testing.T) {
	cc := []uint32{450_000_000}

	tests := []struct {
		name string
		sys  trunking.System
		want []string // substrings that must appear
		deny []string // substrings that must NOT appear
	}{
		{
			name: "tetra default: lms off, channel_coding + cma on",
			sys:  trunking.System{Name: "T", Protocol: trunking.ProtocolTETRA, ControlChannels: cc},
			want: []string{"traffic_lms=off", "channel_coding=on", "cma_equalizer=on", "clock=gardner"},
		},
		{
			name: "tetra with lms on",
			sys: trunking.System{
				Name: "T", Protocol: trunking.ProtocolTETRA, ControlChannels: cc,
				TETRATrafficLMS: "on",
			},
			want: []string{"traffic_lms=on"},
			deny: []string{"traffic_lms=off"},
		},
		{
			name: "tetra-dmo also reports lms",
			sys: trunking.System{
				Name: "D", Protocol: trunking.ProtocolTETRADMO, ControlChannels: cc,
				TETRATrafficLMS: "true",
			},
			want: []string{"traffic_lms=on"},
		},
		{
			name: "p25 phase1 default: c4fm, soft off",
			sys:  trunking.System{Name: "P", Protocol: trunking.ProtocolP25, ControlChannels: cc},
			want: []string{"demod=c4fm", "soft_decision=off"},
		},
		{
			name: "p25 phase1 cqpsk + soft",
			sys: trunking.System{
				Name: "P", Protocol: trunking.ProtocolP25, ControlChannels: cc,
				P25Phase1DemodMode: "lsm", P25Phase1SoftDecision: "1",
			},
			want: []string{"demod=cqpsk", "soft_decision=on"},
		},
		{
			name: "p25 phase2 equalizer + dc block on",
			sys: trunking.System{
				Name: "P2", Protocol: trunking.ProtocolP25Phase2, ControlChannels: cc,
				P25Phase2Equalizer: "cma", P25Phase2DCBlock: "on",
			},
			want: []string{"cma_equalizer=on", "dc_block=on", "soft_decision=off", "clock=gardner"},
		},
		{
			name: "dmr tier3 default: interleaved on",
			sys:  trunking.System{Name: "M", Protocol: trunking.ProtocolDMR, ControlChannels: cc, DMRInterleavedVoice: true},
			want: []string{"interleaved_voice=on"},
		},
		{
			name: "dmr tier2 with color code pinned",
			sys: trunking.System{
				Name: "M2", Protocol: trunking.ProtocolDMRTier2, ControlChannels: cc,
				DMRInterleavedVoice: false, DMRColorCode: func() *uint8 { v := uint8(7); return &v }(),
			},
			want: []string{"interleaved_voice=off", "color_code=7"},
		},
		{
			name: "nxdn afc on, soft default off",
			sys: trunking.System{
				Name: "N", Protocol: trunking.ProtocolNXDN, ControlChannels: cc,
				NXDNAFC: "on",
			},
			want: []string{"afc=on", "soft_decision=off"},
		},
		{
			name: "edacs has no tunable decode enhancements",
			sys:  trunking.System{Name: "E", Protocol: trunking.ProtocolEDACS, ControlChannels: cc},
			want: []string{"(none)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := systemEnhancements(tt.sys)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("systemEnhancements() = %q, want substring %q", got, w)
				}
			}
			for _, d := range tt.deny {
				if strings.Contains(got, d) {
					t.Errorf("systemEnhancements() = %q, must not contain %q", got, d)
				}
			}
		})
	}
}

// TestSummarizeGlobalReportsLevers pins the daemon-wide summary attrs: the
// pre-decimation factor, autotune, MRC diversity per endpoint, and the
// recording audio enhancements (including the spec_amplitude_enhance tri-state
// pointer that defaults ON when unset).
func TestSummarizeGlobalReportsLevers(t *testing.T) {
	kv := func(attrs []any) map[string]any {
		m := map[string]any{}
		for i := 0; i+1 < len(attrs); i += 2 {
			k, _ := attrs[i].(string)
			m[k] = attrs[i+1]
		}
		return m
	}

	t.Run("predecimation engaged + mrc + spec_amp default on", func(t *testing.T) {
		cfg := config.Config{}
		cfg.SDR.SampleRate = 2_500_000
		cfg.SDR.InputSampleRate = 10_000_000
		cfg.SDR.Autotune = true
		cfg.SDR.SoapyRemote = []config.SoapyRemoteConfig{
			{Serial: "x310", Diversity: "mrc"},
			{Addr: "1.2.3.4:5", Diversity: "none"}, // must be skipped
		}
		m := kv(summarizeGlobal(cfg))

		if got := m["predecimation_factor"]; got != uint32(4) {
			t.Errorf("predecimation_factor = %v, want 4", got)
		}
		if got := m["autotune"]; got != true {
			t.Errorf("autotune = %v, want true", got)
		}
		if got, _ := m["diversity"].(string); got != "x310:mrc" {
			t.Errorf("diversity = %q, want %q", got, "x310:mrc")
		}
		// SpecAmplitudeEnhance nil ⇒ defaults ON.
		if got := m["rec_spec_amplitude_enhance"]; got != true {
			t.Errorf("rec_spec_amplitude_enhance = %v, want true (default)", got)
		}
	})

	t.Run("no predecimation, no diversity, spec_amp forced off", func(t *testing.T) {
		off := false
		cfg := config.Config{}
		cfg.SDR.SampleRate = 2_400_000
		cfg.Recordings.SpecAmplitudeEnhance = &off
		m := kv(summarizeGlobal(cfg))

		if got, _ := m["input_predecimation"].(string); got != "off" {
			t.Errorf("input_predecimation = %q, want %q", got, "off")
		}
		if _, ok := m["predecimation_factor"]; ok {
			t.Errorf("predecimation_factor should be absent when disabled")
		}
		if got, _ := m["diversity"].(string); got != "off" {
			t.Errorf("diversity = %q, want %q", got, "off")
		}
		if got := m["rec_spec_amplitude_enhance"]; got != false {
			t.Errorf("rec_spec_amplitude_enhance = %v, want false", got)
		}
	})
}
