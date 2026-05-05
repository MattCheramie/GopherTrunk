// Package equalizer implements adaptive channel equalizers used to
// fight simulcast distortion — the inter-symbol interference produced
// when multiple transmitters cover the same frequency at slightly
// different arrival delays at the receiver. Premium hardware scanners
// market this capability as "True I/Q"; with an SDR we always have
// I/Q, so the win is what we do with it.
//
// Two complementary algorithms ship here:
//
//   lms.go   Least-Mean-Squares adaptive FIR equalizer. Trained
//            with reference (or decision-directed) symbols; fast to
//            converge but needs a known training sequence (or a
//            slicer it can trust).
//   cma.go   Constant Modulus Algorithm — blind equalizer for
//            constant-envelope modulations (PSK family). Drives the
//            output toward a constant magnitude without ever needing
//            a reference; useful when the upstream demod has no
//            preamble to lock to.
//
// The package operates on complex64 IQ samples / symbols (matching
// the rest of the DSP stack). Equalizers slot between the channelizer
// and the symbol-time-recovery / demodulator stages of a per-call
// chain — the demod-pipeline composer is the natural integration
// point once a protocol decoder needs them on a real signal.
package equalizer
