package tools

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/recipe"
)

// recipeTool is a CyberChef-style operation pipeline: chain byte transforms
// (XOR, cipher decrypt, bit reversal, spectral inversion, hex/base64) and
// analyses (entropy, randomness) into one reusable recipe, piping the bytes
// from each step into the next. It encodes the de-obfuscation sequence an RF
// analyst repeats by hand.
type recipeTool struct{}

func (recipeTool) Name() string { return "recipe" }
func (recipeTool) Synopsis() string {
	return "chain transform/analysis operations into a reusable pipeline"
}
func (recipeTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{recipeRun{}}
}

type recipeRun struct{}

func (recipeRun) Name() string     { return "run" }
func (recipeRun) Synopsis() string { return "run a recipe (JSON/YAML steps) over an input file" }

func (recipeRun) Run(_ context.Context, args []string, _ cryptolab.Env) (*cryptolab.Result, error) {
	fs := newFlagSet("recipe run")
	in := fs.String("in", "", "input file — required")
	recipePath := fs.String("recipe", "", "recipe file (JSON or YAML) — required")
	out := fs.String("out", "", "optional file to write the final transformed bytes")
	list := fs.Bool("list", false, "list the available operations instead of running")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *list {
		res := &cryptolab.Result{Tool: "recipe", Mode: "run", Summary: "available recipe operations"}
		for _, op := range recipe.Ops() {
			kind := "analysis"
			if op.Transform {
				kind = "transform"
			}
			res.AddFinding(op.Name, 0, map[string]any{"kind": kind, "synopsis": op.Synopsis})
		}
		return res, nil
	}
	data, err := readInput(*in)
	if err != nil {
		return nil, err
	}
	if *recipePath == "" {
		return nil, fmt.Errorf("-recipe is required")
	}
	steps, err := loadRecipe(*recipePath)
	if err != nil {
		return nil, err
	}

	rep, runErr := recipe.Run(data, steps)
	// rep is non-nil even on error (partial); surface what ran before failing.

	res := &cryptolab.Result{Tool: "recipe", Mode: "run"}
	res.AddField("input_bytes", rep.InputBytes)
	res.AddField("steps", len(rep.Steps))
	for i, s := range rep.Steps {
		detail := map[string]any{"transform": s.Transform, "bytes_in": s.BytesIn, "bytes_out": s.BytesOut}
		for k, v := range s.Info {
			detail[k] = v
		}
		if s.Note != "" {
			detail["note"] = s.Note
		}
		res.AddFinding(fmt.Sprintf("%d. %s", i+1, s.Op), float64(i+1), detail)
	}
	if runErr != nil {
		return res, runErr
	}
	res.Summary = fmt.Sprintf("ran %d step(s): %d → %d bytes", len(rep.Steps), rep.InputBytes, len(rep.FinalBytes))
	res.AddField("final_bytes", len(rep.FinalBytes))
	res.AddField("final_hex", rep.FinalHex)
	res.AddField("final_ascii", rep.FinalASCII)
	if *out != "" {
		if err := os.WriteFile(*out, rep.FinalBytes, 0o644); err != nil {
			return res, fmt.Errorf("write %s: %w", *out, err)
		}
		res.AddField("written", *out)
	}
	res.Note("transform steps rewrite the working buffer; analysis steps (stats, randomness) observe it without changing it.")
	return res, nil
}

// loadRecipe parses a recipe file into ordered steps. The file is JSON or YAML
// (JSON is valid YAML, so one parser handles both): either a top-level list of
// step objects, or an object with a `steps:` list. Each step object has an
// `op` key plus that op's parameters.
func loadRecipe(path string) ([]recipe.Step, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe %s: %w", path, err)
	}
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse recipe %s: %w", path, err)
	}
	var list []any
	switch d := doc.(type) {
	case []any:
		list = d
	case map[string]any:
		s, ok := d["steps"].([]any)
		if !ok {
			return nil, fmt.Errorf("recipe %s: object form needs a `steps:` list", path)
		}
		list = s
	default:
		return nil, fmt.Errorf("recipe %s: expected a list of steps or an object with `steps:`", path)
	}
	var steps []recipe.Step
	for i, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("recipe step %d: expected an object {op: ...}", i+1)
		}
		op, _ := m["op"].(string)
		if op == "" {
			return nil, fmt.Errorf("recipe step %d: missing `op`", i+1)
		}
		params := make(map[string]any, len(m))
		for k, v := range m {
			if k != "op" {
				params[k] = v
			}
		}
		steps = append(steps, recipe.Step{Op: op, Params: params})
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("recipe %s: no steps", path)
	}
	return steps, nil
}

func init() { cryptolab.Register(recipeTool{}) }
