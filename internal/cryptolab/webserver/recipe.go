package webserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/recipe"
)

// maxRecipeOutputB64 caps how many output bytes the recipe endpoint returns
// inline (base64) so the builder can offer a download without a second
// round-trip; larger outputs return the previews only.
const maxRecipeOutputB64 = 16 << 20

// handleRecipeOps returns the recipe operation catalogue (name, synopsis,
// transform flag, and per-op parameters) the web builder renders its palette
// and step forms from. Read-only, so it is not gated.
func (s *Server) handleRecipeOps(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, recipe.Specs())
}

type recipeStepDTO struct {
	Op     string         `json:"op"`
	Params map[string]any `json:"params"`
}

type recipeRequest struct {
	InputID string          `json:"input_id"`
	Steps   []recipeStepDTO `json:"steps"`
}

// handleRecipe runs a structured recipe (built in the web UI) over a
// previously-uploaded input file and returns the per-step report plus the
// final bytes. The pipeline is pure and in-memory, so it runs synchronously —
// no job is created. Gated like the other mutations.
func (s *Server) handleRecipe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	var req recipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "recipe: "+err.Error())
		return
	}
	if req.InputID == "" {
		writeErr(w, http.StatusBadRequest, "recipe: input_id is required")
		return
	}
	if len(req.Steps) == 0 {
		writeErr(w, http.StatusBadRequest, "recipe: at least one step is required")
		return
	}

	s.mu.Lock()
	u, ok := s.files[req.InputID]
	s.mu.Unlock()
	if !ok {
		writeErr(w, http.StatusBadRequest, "recipe: unknown input_id (upload the input first)")
		return
	}
	data, err := os.ReadFile(u.Path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "recipe: read input: "+err.Error())
		return
	}

	steps := make([]recipe.Step, len(req.Steps))
	for i, st := range req.Steps {
		// Refuse ops that shell out to a host program — those are CLI-only, so
		// a browser request can never run an arbitrary command on the host.
		if recipe.External(st.Op) {
			writeErr(w, http.StatusForbidden, "recipe: the "+st.Op+" operation runs a host program and is not available over the web; use the CLI")
			return
		}
		steps[i] = recipe.Step{Op: st.Op, Params: st.Params}
	}

	rep, runErr := recipe.Run(data, steps)
	resp := map[string]any{
		"input_bytes": rep.InputBytes,
		"steps":       rep.Steps,
		"final_hex":   rep.FinalHex,
		"final_ascii": rep.FinalASCII,
		"final_bytes": len(rep.FinalBytes),
	}
	if len(rep.FinalBytes) <= maxRecipeOutputB64 {
		resp["output_b64"] = base64.StdEncoding.EncodeToString(rep.FinalBytes)
	}
	if runErr != nil {
		// The pipeline aborts at the failing step but the partial report is
		// still useful, so return 200 with the error surfaced in the body.
		resp["error"] = runErr.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}
