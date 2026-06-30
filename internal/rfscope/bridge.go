package rfscope

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab/engine/keystream"
)

// frameRec is one line of the cryptolab `ks` frames file. The field names and
// hex encoding match exactly what cryptolab/tools/ks.go's loadFrames parses —
// the decoder→cryptolab bridge format #832 anticipates.
type frameRec struct {
	Label string `json:"label"`
	IV    string `json:"iv"`
	CT    string `json:"ct"`
}

// EmitFrames writes the entropy analyzer's recovered payloads as a cryptolab `ks`
// frames file: one JSON object per line, {label, iv, ct} with iv/ct hex-encoded.
// The emitted file feeds `cryptolab ks reuse|mtp|extract`, `classify auto`, and
// `brute` directly. Returns the number of frames written. A frame with no
// recovered IV carries an empty iv (cryptolab ignores it for reuse but still
// accepts it for the other tools).
func EmitFrames(sc *Scene, w io.Writer) (int, error) {
	enc := json.NewEncoder(w)
	n := 0
	for _, r := range entropyResults(sc) {
		if len(r.payload) == 0 {
			continue
		}
		rec := frameRec{
			Label: fmt.Sprintf("emitter-%d@%.4fMHz", r.EmitterID, float64(r.FreqHz)/1e6),
			IV:    hex.EncodeToString(r.iv),
			CT:    hex.EncodeToString(r.payload),
		}
		if err := enc.Encode(rec); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// DetectKeystreamReuse builds keystream.Frames from the entropy payloads and
// reports reuse groups (frames sharing an IV/MI) via keystream.FindReuse — the
// in-binary half of the keystream-reuse workflow. Generic unknown payloads carry
// no IV, so this fires only once IV/MI recovery is wired for a partial protocol;
// when it does, the same frames feed `cryptolab ks reuse|mtp`.
func DetectKeystreamReuse(sc *Scene) []keystream.ReuseGroup {
	results := entropyResults(sc)
	frames := make([]keystream.Frame, 0, len(results))
	for _, r := range results {
		if len(r.iv) == 0 {
			continue
		}
		frames = append(frames, keystream.Frame{
			Label: fmt.Sprintf("emitter-%d", r.EmitterID),
			IV:    r.iv,
			CT:    r.payload,
		})
	}
	return keystream.FindReuse(frames)
}
