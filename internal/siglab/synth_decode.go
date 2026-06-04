package siglab

import (
	"bytes"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// SynthesizeAndAnalyze synthesizes an idealized (optionally impaired) signal
// for synth.Protocol and immediately analyzes it through the production
// pipeline, all in memory — no capture file is written to disk. It is the
// one-call path behind the web interface's "compare against a generated
// idealized signal" feature: the returned Result (and its IQTaps, when
// decode.CaptureIQ is set) can be diffed against an uploaded capture's Result.
//
// The decode template carries the analysis knobs the caller wants
// (CollectIQDiag, CaptureIQ, CaptureIQMaxPoints, Conjugate, IQCorrect,
// DemodMode and the P25 deep knobs); the protocol, sample rate, format and
// per-protocol system knobs are taken from the synthesis fixture so the
// decode always matches the signal that was generated. onEvent (when
// non-nil) receives each EventRecord live, exactly like RunReaderStream.
func SynthesizeAndAnalyze(synth SynthOptions, decode Config, onEvent func(EventRecord)) (*Result, *Metadata, error) {
	iq, meta, err := Synthesize(synth)
	if err != nil {
		return nil, nil, err
	}

	proto, err := ParseProtocolCLI(meta.Protocol)
	if err != nil {
		return nil, nil, fmt.Errorf("siglab: synthesized metadata protocol %q: %w", meta.Protocol, err)
	}

	// Build the decode system from the fixture's per-protocol knobs so the
	// analysis path matches the synthesized waveform.
	sys := trunking.System{Name: "siglab", Protocol: proto}
	applySystemKnobs(&sys, meta.System)

	cfg := decode
	cfg.Protocol = proto
	cfg.System = sys
	cfg.SystemName = "siglab"
	cfg.FrequencyHz = meta.CenterFreqHz
	cfg.SampleRateHz = meta.SampleRateHz
	cfg.Format = synth.Format
	// A synthesized signal is centred at 0 Hz; never auto-tune it.
	cfg.AutoTune = false
	cfg.TuneHz = 0

	data := EncodeCapture(iq, synth.Format)
	res, err := RunReaderStream(bytes.NewReader(data), "synthesized", cfg, onEvent)
	if err != nil {
		return nil, nil, err
	}
	res.Source = "synthesized"
	return res, meta, nil
}
