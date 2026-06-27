package tools

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/voice"
)

type descrambleTool struct{}

func (descrambleTool) Name() string            { return "descramble" }
func (descrambleTool) Synopsis() string        { return "analog voice descramblers (spectral inversion)" }
func (descrambleTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{descrambleInvert{}} }

type descrambleInvert struct{}

func (descrambleInvert) Name() string { return "invert" }
func (descrambleInvert) Synopsis() string {
	return "full-band spectral inversion of 16-bit LE mono PCM (self-inverse)"
}

func (descrambleInvert) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("descramble invert")
	in := fs.String("in", "", "raw 16-bit signed little-endian mono PCM — required")
	out := fs.String("out", "", "output PCM path — required")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *out == "" {
		return nil, fmt.Errorf("-out is required")
	}
	raw, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	if len(raw)%2 != 0 {
		return nil, fmt.Errorf("descramble: input length %d is not a whole number of 16-bit samples", len(raw))
	}
	samples := make([]int16, len(raw)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(raw[2*i:]))
	}
	inv := voice.SpectralInvertInt16(samples)
	obuf := make([]byte, len(raw))
	for i, s := range inv {
		binary.LittleEndian.PutUint16(obuf[2*i:], uint16(s))
	}
	if err := os.WriteFile(*out, obuf, 0o644); err != nil {
		return nil, fmt.Errorf("descramble: write %s: %w", *out, err)
	}
	env.Logf("inverted", "samples", len(samples), "out", *out)

	res := &cryptolab.Result{Tool: "descramble", Mode: "invert",
		Summary: fmt.Sprintf("spectrally inverted %d samples -> %s", len(samples), *out)}
	res.AddField("samples", len(samples))
	res.AddField("output", *out)
	res.Note("spectral inversion is its own inverse: run again to undo. For split-band/rolling inverters this is only a first pass.")
	return res, nil
}

func init() { cryptolab.Register(descrambleTool{}) }
