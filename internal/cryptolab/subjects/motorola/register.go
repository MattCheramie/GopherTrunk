package motorola

import "github.com/MattCheramie/GopherTrunk/internal/cryptolab"

// aliasTool is a cryptolab subject: a length-seeded, keyless, byte-oriented
// alias obfuscator, with five recovery modes.
type aliasTool struct{}

func (aliasTool) Name() string     { return "alias" }
func (aliasTool) Synopsis() string { return "length-seeded byte-obfuscator alias recovery" }
func (aliasTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{
		gaugeMode{},
		structureMode{},
		cellsMode{},
		fromseedMode{},
		propagateMode{},
	}
}

func init() { cryptolab.Register(aliasTool{}) }
