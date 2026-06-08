package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// runSiglabSweep is `gophertrunk siglab sweep`: it synthesizes P25 Phase 1 at a
// ladder of injected SNRs on both demod paths, decodes each through the
// production pipeline, and prints the measured demod-quality curve (lock, EVM,
// estimated SNR) against the theoretical symbol-error-rate reference. It is the
// ad-hoc, human-readable companion to the hard-gated sweep regression test —
// the same instrument, on demand, for "did my demod change help?"
func runSiglabSweep(args []string) {
	fs := flag.NewFlagSet("siglab sweep", flag.ExitOnError)
	snrMin := fs.Float64("snr-min", 2, "minimum injected SNR (dB)")
	snrMax := fs.Float64("snr-max", 30, "maximum injected SNR (dB)")
	snrStep := fs.Float64("snr-step", 2, "SNR step (dB)")
	seed := fs.Int64("seed", 0x5175, "AWGN seed (reproducible draws)")
	csvPath := fs.String("csv", "", "optional path to write the sweep as CSV")
	fs.Parse(args)

	if *snrStep <= 0 || *snrMax < *snrMin {
		fmt.Fprintln(os.Stderr, "siglab sweep: need snr-step > 0 and snr-max >= snr-min")
		os.Exit(2)
	}

	const sps = 48_000.0 / 4800.0 // P25 Phase 1 samples/symbol at 48 kHz
	modes := []struct {
		name       string
		synthMod   string
		demodMode  string
		ref        func(esN0dB float64) float64
		bitsPerSym float64
	}{
		{"c4fm", "c4fm", "", metrics.SER4PAM, 2},
		{"cqpsk", "lsm", "cqpsk", metrics.SERCoherentQPSK, 2},
	}

	var rows []sweepRow

	for _, m := range modes {
		fmt.Printf("\n%s (vs theoretical reference):\n", m.name)
		fmt.Printf("  %-8s %-8s %-7s %-8s %-9s\n", "SNR dB", "Es/N0", "lock", "EVM %", "SNR~ dB")
		for snr := *snrMin; snr <= *snrMax+1e-9; snr += *snrStep {
			synth := siglab.SynthOptions{
				Protocol:    trunking.ProtocolP25,
				Format:      siglab.FormatF32,
				Modulation:  m.synthMod,
				Impairments: demod.Impairments{SNRdB: snr, Seed: *seed},
			}
			cfg := siglab.Config{CollectIQDiag: true, DemodMode: m.demodMode}
			res, _, err := siglab.SynthesizeAndAnalyze(synth, cfg, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "siglab sweep: %s @ %.0f dB: %v\n", m.name, snr, err)
				os.Exit(1)
			}
			r := sweepRow{mode: m.name, snrDB: snr, esN0dB: metrics.EsN0FromSNR(snr, sps), locked: res.Locked}
			if res.Signal != nil && res.Signal.Demod != nil {
				r.evmPct = res.Signal.Demod.EVMPct
				r.snrEstDB = res.Signal.Demod.SNREstimateDB
			}
			rows = append(rows, r)
			lock := "no"
			if r.locked {
				lock = "yes"
			}
			fmt.Printf("  %-8.1f %-8.1f %-7s %-8.1f %-9.1f\n", r.snrDB, r.esN0dB, lock, r.evmPct, r.snrEstDB)
		}
	}

	if *csvPath != "" {
		if err := writeSweepCSV(*csvPath, rows); err != nil {
			fmt.Fprintf(os.Stderr, "siglab sweep: write csv: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nwrote %s\n", *csvPath)
	}
}

// sweepRow is one measured point of the demod sweep.
type sweepRow struct {
	mode             string
	snrDB, esN0dB    float64
	locked           bool
	evmPct, snrEstDB float64
}

func writeSweepCSV(path string, rows []sweepRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"mode", "snr_db", "es_n0_db", "locked", "evm_pct", "snr_est_db"}); err != nil {
		return err
	}
	for _, r := range rows {
		rec := []string{
			r.mode,
			strconv.FormatFloat(r.snrDB, 'f', 1, 64),
			strconv.FormatFloat(r.esN0dB, 'f', 1, 64),
			strconv.FormatBool(r.locked),
			strconv.FormatFloat(r.evmPct, 'f', 2, 64),
			strconv.FormatFloat(r.snrEstDB, 'f', 2, 64),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}
