package motorola

import "github.com/MattCheramie/GopherTrunk/internal/cryptolab"

// aliasTool is the flagship cryptolab tool: the Motorola P25 talker-alias
// obfuscation, with its four recovery modes.
type aliasTool struct{}

func (aliasTool) Name() string     { return "alias" }
func (aliasTool) Synopsis() string { return "Motorola P25 talker-alias obfuscation recovery" }
func (aliasTool) Modes() []cryptolab.Mode {
	return []cryptolab.Mode{
		gaugeMode{},
		structureMode{},
		cellsMode{},
		fromseedMode{},
	}
}

func init() { cryptolab.Register(aliasTool{}) }
