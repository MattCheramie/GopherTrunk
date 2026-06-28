package tools

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/voice"
)

type descrambleTool struct{}

func (descrambleTool) Name() string     { return "descramble" }
func (descrambleTool) Synopsis() string { return "analog voice descramblers (spectral inversion)" }
func (descrambleTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{descrambleInvert{}, descrambleSplitband{}, descrambleRolling{}}
}

// readPCM reads a raw 16-bit signed little-endian mono PCM file into float64
// samples in [-1,1).
func readPCM(path string) ([]float64, int, error) {
	raw, err := readInput(path)
	if err != nil {
		return nil, 0, err
	}
	if len(raw)%2 != 0 {
		return nil, 0, fmt.Errorf("descramble: input length %d is not a whole number of 16-bit samples", len(raw))
	}
	n := len(raw) / 2
	s := make([]float64, n)
	for i := 0; i < n; i++ {
		s[i] = float64(int16(binary.LittleEndian.Uint16(raw[2*i:]))) / 32768.0
	}
	return s, n, nil
}

// writePCM writes float64 samples back to 16-bit signed little-endian PCM.
func writePCM(path string, s []float64) error {
	buf := make([]byte, len(s)*2)
	for i, v := range s {
		x := v * 32768.0
		if x > 32767 {
			x = 32767
		} else if x < -32768 {
			x = -32768
		}
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(int16(x)))
	}
	return os.WriteFile(path, buf, 0o644)
}

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
	s, n, err := readPCM(*in)
	if err != nil {
		return nil, err
	}
	if err := writePCM(*out, voice.BandInvert(s, 0, 1)); err != nil {
		return nil, fmt.Errorf("descramble: write %s: %w", *out, err)
	}
	env.Logf("inverted", "samples", n, "out", *out)
	res := &cryptolab.Result{Tool: "descramble", Mode: "invert",
		Summary: fmt.Sprintf("spectrally inverted %d samples -> %s", n, *out)}
	res.AddField("samples", n)
	res.AddField("output", *out)
	res.Note("spectral inversion is its own inverse: run again to undo.")
	return res, nil
}

type descrambleSplitband struct{}

func (descrambleSplitband) Name() string { return "splitband" }
func (descrambleSplitband) Synopsis() string {
	return "split-band inversion (invert low and high sub-bands independently)"
}

func (descrambleSplitband) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("descramble splitband")
	in := fs.String("in", "", "raw 16-bit signed little-endian mono PCM — required")
	out := fs.String("out", "", "output PCM path — required")
	split := fs.Float64("split", 0.5, "split point as a fraction of Nyquist (0..1)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *out == "" {
		return nil, fmt.Errorf("-out is required")
	}
	s, n, err := readPCM(*in)
	if err != nil {
		return nil, err
	}
	if err := writePCM(*out, voice.SplitBandInvert(s, *split)); err != nil {
		return nil, fmt.Errorf("descramble: write %s: %w", *out, err)
	}
	env.Logf("split-band inverted", "samples", n, "split", *split, "out", *out)
	res := &cryptolab.Result{Tool: "descramble", Mode: "splitband",
		Summary: fmt.Sprintf("split-band inverted %d samples at split=%.3f -> %s", n, *split, *out)}
	res.AddField("samples", n)
	res.AddField("split", *split)
	res.AddField("output", *out)
	res.Note("split-band inversion is its own inverse for a given split: re-run with the same -split to undo.")
	return res, nil
}

type descrambleRolling struct{}

func (descrambleRolling) Name() string { return "rolling" }
func (descrambleRolling) Synopsis() string {
	return "rolling inversion: a per-frame split schedule, or auto-detect inverted frames"
}

func (descrambleRolling) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("descramble rolling")
	in := fs.String("in", "", "raw 16-bit signed little-endian mono PCM — required")
	out := fs.String("out", "", "output PCM path — required")
	frame := fs.Int("frame", 1024, "frame length in samples")
	schedule := fs.String("schedule", "auto", "comma-separated split fractions, or 'auto' to detect inverted frames")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *out == "" {
		return nil, fmt.Errorf("-out is required")
	}
	s, n, err := readPCM(*in)
	if err != nil {
		return nil, err
	}

	res := &cryptolab.Result{Tool: "descramble", Mode: "rolling"}
	var result []float64
	if strings.EqualFold(strings.TrimSpace(*schedule), "auto") {
		var flags []bool
		result, flags = voice.RollingAuto(s, *frame)
		inverted := 0
		for _, f := range flags {
			if f {
				inverted++
			}
		}
		res.AddField("frames", len(flags))
		res.AddField("frames_inverted", inverted)
		res.Summary = fmt.Sprintf("auto-descrambled %d frames (%d inverted) -> %s", len(flags), inverted, *out)
		res.Note("auto mode is a heuristic full-band inversion detector (energy balance); a known schedule is exact.")
	} else {
		splits, err := parseFloatList(*schedule)
		if err != nil {
			return nil, fmt.Errorf("descramble: -schedule: %w", err)
		}
		result = voice.RollingInvert(s, *frame, splits)
		res.AddField("schedule", splits)
		res.Summary = fmt.Sprintf("rolling-inverted %d samples with a %d-step schedule -> %s", n, len(splits), *out)
		res.Note("re-run with the same -frame and -schedule to undo a known rolling inverter.")
	}
	if err := writePCM(*out, result); err != nil {
		return nil, fmt.Errorf("descramble: write %s: %w", *out, err)
	}
	env.Logf("rolling descramble", "samples", n, "frame", *frame, "out", *out)
	res.AddField("samples", n)
	res.AddField("output", *out)
	return res, nil
}

func parseFloatList(s string) ([]float64, error) {
	var out []float64
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("bad split %q: %w", p, err)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty schedule")
	}
	return out, nil
}

func init() { cryptolab.Register(descrambleTool{}) }
