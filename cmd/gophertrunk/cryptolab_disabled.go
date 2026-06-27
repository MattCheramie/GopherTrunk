//go:build !cryptolab

package main

import (
	"fmt"
	"os"
)

// runCryptolab is the default-build stub for the optional cryptolab
// research toolkit. The toolkit is excluded from the standard install to
// keep the operator binary lean; rebuild with -tags cryptolab to link it in.
func runCryptolab(_ []string) {
	fmt.Fprintln(os.Stderr, "gophertrunk: cryptolab not built in — rebuild with `make build TAGS=cryptolab` (or `go build -tags cryptolab ./cmd/gophertrunk`)")
	os.Exit(2)
}
