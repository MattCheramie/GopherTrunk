package airspy

import "github.com/MattCheramie/GopherTrunk/internal/sdr"

// Register the Airspy and HydraSDR RFOne drivers under their canonical
// names so a blank import of this package from cmd/gophertrunk makes the
// devices discoverable through sdr.Pool.Open.
func init() {
	sdr.Register(New(nil))
	sdr.Register(NewHydraSDR(nil))
}
