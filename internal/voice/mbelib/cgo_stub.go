//go:build !mbelib || !cgo

package mbelib

// Empty stub — the build tag wasn't supplied (or CGO is disabled).
// No vocoder factories are registered; calling code that asks the
// voice.Registry for "imbe" / "ambe2" will get an "unknown vocoder"
// error and can fall back to the null vocoder or to the raw-frame
// sidecar output the recorder already writes.
//
// To enable the real wrapper, install libmbe-dev (build it from
// source — https://github.com/szechyjs/mbelib) and run:
//
//	make build TAGS=mbelib
//
// CI does NOT exercise the wrapper; the project ships the stub by
// default and the wrapper code is verified at build time only when
// an operator opts in.
