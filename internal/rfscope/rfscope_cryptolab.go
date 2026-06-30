//go:build cryptolab

package rfscope

import (
	"context"
	"io"
	"log/slog"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
	// Blank import registers the toolkit's tools (classify, ks, …) via init(),
	// the same way cmd/gophertrunk/cryptolab_enabled.go links them.
	_ "github.com/MattCheramie/GopherTrunk/internal/cryptolab/tools"
)

// CryptolabLinked reports whether the cryptolab toolkit is compiled into this
// binary (true only under -tags cryptolab).
const CryptolabLinked = true

// RunCryptolabTool runs a cryptolab tool/mode (e.g. "ks"/"reuse" or
// "classify"/"auto") over args and returns its structured Result. It is the
// registry-driven half of the bridge — the in-binary triage in entropy.go works
// without the tag, this adds the full tool result when cryptolab is linked.
func RunCryptolabTool(ctx context.Context, tool, mode string, args []string) (*cryptolab.Result, error) {
	_, m, err := cryptolab.LookupMode(tool, mode)
	if err != nil {
		return nil, err
	}
	env := cryptolab.Env{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Stdout: io.Discard,
	}
	return m.Run(ctx, args, env)
}
