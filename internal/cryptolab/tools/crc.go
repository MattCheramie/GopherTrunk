package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/crc"
)

type crcTool struct{}

func (crcTool) Name() string            { return "crc" }
func (crcTool) Synopsis() string        { return "recover CRC parameters from sample frames" }
func (crcTool) Modes() []cryptolab.Mode { return []cryptolab.Mode{crcRecover{}, crcCompute{}} }

type crcRecover struct{}

func (crcRecover) Name() string { return "recover" }
func (crcRecover) Synopsis() string {
	return "recover poly/init/refin/refout/xorout from `datahex,crchex` sample lines"
}

func (crcRecover) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("crc recover")
	in := fs.String("in", "", "samples file, one `datahex,crchex` per line — required")
	widthsArg := fs.String("widths", "16", "comma-separated CRC widths to try")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	raw, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	samples, err := parseSamples(string(raw))
	if err != nil {
		return nil, err
	}
	widths, err := parseWidths(*widthsArg)
	if err != nil {
		return nil, err
	}
	env.Logf("recovering CRC", "samples", len(samples), "widths", widths)
	found, err := crc.Recover(samples, widths)
	if err != nil {
		return nil, err
	}

	res := &cryptolab.Result{Tool: "crc", Mode: "recover",
		Summary: fmt.Sprintf("%d sample frames -> %d candidate parameter set(s)", len(samples), len(found))}
	res.AddField("samples", len(samples))
	res.AddField("candidates", len(found))
	for _, p := range found {
		res.AddFinding(
			fmt.Sprintf("w%d poly=%#x init=%#x xor=%#x refin=%v refout=%v", p.Width, p.Poly, p.Init, p.XorOut, p.RefIn, p.RefOut),
			1, map[string]any{"width": p.Width, "poly": p.Poly, "init": p.Init, "xorout": p.XorOut, "refin": p.RefIn, "refout": p.RefOut})
	}
	if len(found) == 0 {
		res.Note("no parameters fit: add more sample frames of varied length, or widen -widths.")
	} else if len(found) > 1 {
		res.Note("multiple parameter sets fit; add frames of differing length to disambiguate init/xorout.")
	}
	return res, nil
}

type crcCompute struct{}

func (crcCompute) Name() string     { return "compute" }
func (crcCompute) Synopsis() string { return "compute a CRC over a file with explicit parameters" }

func (crcCompute) Run(_ context.Context, args []string, env cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("crc compute")
	in := fs.String("in", "", "input file — required")
	width := fs.Int("width", 16, "CRC width in bits")
	poly := fs.Uint64("poly", 0x1021, "generator polynomial")
	initV := fs.Uint64("init", 0, "initial register value")
	xorOut := fs.Uint64("xorout", 0xFFFF, "final XOR")
	refIn := fs.Bool("refin", false, "reflect input bytes")
	refOut := fs.Bool("refout", false, "reflect output")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	data, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	p := crc.Params{Width: *width, Poly: *poly, Init: *initV, XorOut: *xorOut, RefIn: *refIn, RefOut: *refOut}
	sum := p.Compute(data)

	res := &cryptolab.Result{Tool: "crc", Mode: "compute",
		Summary: fmt.Sprintf("CRC-%d = %#x over %d bytes", *width, sum, len(data))}
	res.AddField("crc", fmt.Sprintf("%#x", sum))
	res.AddField("width", *width)
	return res, nil
}

func parseSamples(text string) ([]crc.Sample, error) {
	var out []crc.Sample
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: want `datahex,crchex`", i+1)
		}
		data, err := hex.DecodeString(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("line %d: data: %w", i+1, err)
		}
		cv, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(parts[1]), "0x"), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: crc: %w", i+1, err)
		}
		out = append(out, crc.Sample{Data: data, CRC: cv})
	}
	return out, nil
}

func parseWidths(s string) ([]int, error) {
	var out []int
	for _, p := range strings.Split(s, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("bad width %q: %w", p, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func init() { cryptolab.Register(crcTool{}) }
