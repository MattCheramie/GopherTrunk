package siglab

import "github.com/MattCheramie/GopherTrunk/internal/trunking"

// The protocol-neutral topology types live in package trunking so the
// per-protocol control-channel decoders (which siglab imports) can implement
// TopologyProvider without an import cycle. siglab re-exports them as aliases
// so existing siglab/hunt call sites keep using siglab.TopologySnapshot etc.
type (
	// TopologySnapshot is the accumulated system topology attached to a
	// Result.Topology. See trunking.TopologySnapshot.
	TopologySnapshot = trunking.TopologySnapshot
	// ChannelRef is a band-plan channel coordinate. See trunking.TopoChannelRef.
	ChannelRef = trunking.TopoChannelRef
	// NeighborRef is an adjacent site. See trunking.TopoNeighborRef.
	NeighborRef = trunking.TopoNeighborRef
	// BandPlanSlot is one band-plan entry. See trunking.TopoBandPlanSlot.
	BandPlanSlot = trunking.TopoBandPlanSlot
	// TopologyProvider is the optional pipeline hook. See trunking.TopologyProvider.
	TopologyProvider = trunking.TopologyProvider
)
