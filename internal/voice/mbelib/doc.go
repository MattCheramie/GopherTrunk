// Package mbelib registers IMBE and AMBE+2 decoders backed by
// libmbe (https://github.com/szechyjs/mbelib). The wrapper is
// guarded behind the `mbelib` build tag so default builds stay
// clean; operators with libmbe installed opt in by building with:
//
//	make build TAGS=mbelib
//
// Why a build tag, not a runtime check:
//
//   - libmbe implements algorithms that are subject to active
//     patents in some jurisdictions. The wrapper makes the
//     dependency explicit and operator-controlled rather than
//     accidentally enabled.
//   - libmbe isn't packaged in standard distros; it must be built
//     from source. Linking it unconditionally would break every
//     out-of-the-box build.
//   - The CGO requirement (linking against libmbe.so) raises the
//     toolchain bar; gating it keeps `go build` working on systems
//     without a C toolchain or shared library.
//
// What this package exposes:
//
//   - Default build (no tag): no vocoders are registered; the
//     package is essentially empty. The `null` vocoder from
//     internal/voice covers any caller that expects a Vocoder
//     factory by name.
//   - With `-tags mbelib && cgo`: registers two factories on
//     voice.DefaultRegistry — `imbe` (IMBE 4400 bps for P25
//     Phase 1) and `ambe2` (AMBE+2 2400 bps for P25 Phase 2,
//     DMR, and NXDN) — each wrapping the corresponding mbelib
//     entry point. Decoded audio is 8 kHz / 20 ms / 160 samples
//     per frame, returned as 16-bit signed PCM.
//
// See docs/vocoders.md for the full licensing situation.
package mbelib
