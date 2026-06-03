package main

import (
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runReplayViaSiglab handles `replay` for every protocol beyond the three
// native deep-diagnostic paths, driving the shared siglab engine and
// rendering the same human-readable summary `analyze` uses. It is what lets
// `gophertrunk replay -protocol tetra|nxdn|edacs|…` work without duplicating
// the engine in the replay command.
func runReplayViaSiglab(protocolName, in, format string, sampleRateHz float64, freqHz uint32, tuneHz float64, autoTune, conjugate, iqCorrect, collectDiag bool, rep *diag.Reporter) {
	proto, err := siglab.ParseProtocolCLI(protocolName)
	if err != nil {
		rep.Fatal(2, err)
	}
	sampleFormat, err := siglab.ParseSampleFormat(format)
	if err != nil {
		rep.Fatal(2, err)
	}
	if in == "" {
		rep.Fatalf(2, "-in is required")
	}

	res, err := siglab.Run(in, siglab.Config{
		Protocol:      proto,
		SystemName:    "replay",
		FrequencyHz:   freqHz,
		SampleRateHz:  sampleRateHz,
		Format:        sampleFormat,
		TuneHz:        tuneHz,
		AutoTune:      autoTune,
		Conjugate:     conjugate,
		IQCorrect:     iqCorrect,
		CollectIQDiag: collectDiag,
	})
	if err != nil {
		rep.Fatal(1, err)
	}
	renderResultText(os.Stdout, res)
}
