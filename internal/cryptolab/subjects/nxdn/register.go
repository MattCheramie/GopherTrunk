// Package nxdn is a cryptolab subject: it descrambles NXDN's optional 15-bit
// scrambler and brute-forces the scrambling key. The key space is only 2^15, so
// a guess can be confirmed by descrambling and checking an NXDN frame CRC (or a
// known-plaintext crib). The scrambler primitive itself lives in
// internal/radio/nxdn so the live decoder can reuse it; this package wires it
// into the cryptolab Tool/Mode interface.
package nxdn

import "github.com/MattCheramie/GopherTrunk/internal/cryptolab"

type nxdnTool struct{}

func (nxdnTool) Name() string { return "nxdn" }
func (nxdnTool) Synopsis() string {
	return "NXDN 15-bit scrambler: descramble and brute-force the scrambling key"
}
func (nxdnTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{descrambleMode{}, bruteMode{}}
}

func init() { cryptolab.Register(nxdnTool{}) }
