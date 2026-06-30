//go:build !cryptolab

package rfscope

import (
	"context"
	"errors"

	"github.com/MattCheramie/GopherTrunk/internal/cryptolab"
)

// CryptolabLinked reports whether the cryptolab toolkit is compiled into this
// binary. The default build links the stub, so it is false; rebuild with
// -tags cryptolab to enable the registry-driven tool hand-off.
const CryptolabLinked = false

// ErrCryptolabNotLinked is returned by RunCryptolabTool in the default build.
var ErrCryptolabNotLinked = errors.New("rfscope: cryptolab toolkit not linked — rebuild with -tags cryptolab")

// RunCryptolabTool is the stub for the default (untagged) build. The in-binary
// entropy triage (entropy.go) and the ks frames emission (bridge.go) still work;
// only the registry-driven tool invocation needs the cryptolab build tag.
func RunCryptolabTool(_ context.Context, _, _ string, _ []string) (*cryptolab.Result, error) {
	return nil, ErrCryptolabNotLinked
}
