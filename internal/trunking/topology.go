package trunking

// TopologySnapshot is the protocol-neutral system topology a control-channel
// decoder accumulates over a run: the system/site identity, the neighbor
// (adjacent) sites it advertised, and its band plan. The signal-lab engine
// attaches it to a decode Result, and the hunt accumulator folds it into a
// discovered system.
//
// It lives in package trunking (not siglab) so the per-protocol control-channel
// decoders — which siglab imports — can implement TopologyProvider without an
// import cycle. siglab re-exports these as type aliases for ergonomics.
//
// Why this exists separately from per-event payloads: the identifying fields
// (WACN/SYSID/RFSS/Site for P25, and the per-protocol equivalents) are
// accumulated from periodic status broadcasts inside the decoder's network
// model — they are NOT carried on any per-event payload, so a consumer cannot
// recover them from the event stream. This snapshot is the bridge.
//
// All fields are optional; a protocol fills in only what it can observe.
// Capability varies: P25 surfaces full identity + neighbors + band plan;
// DMR Tier III / EDACS / Motorola can add identity + neighbors; NXDN / TETRA
// give single-site identity; the rest give minimal identity. Band-plan-from-air
// is P25-only today.
type TopologySnapshot struct {
	// Display metadata (populated where the decoder knows it). These name the
	// run for the human-readable network-configuration report and are not part
	// of the topology's Empty() contract.
	SystemName string `json:"system_name,omitempty" yaml:"system_name,omitempty"`
	Protocol   string `json:"protocol,omitempty" yaml:"protocol,omitempty"`

	// P25 identity.
	WACN     uint32 `json:"wacn,omitempty" yaml:"wacn,omitempty"`
	SystemID uint32 `json:"system_id,omitempty" yaml:"system_id,omitempty"`
	NAC      uint16 `json:"nac,omitempty" yaml:"nac,omitempty"` // Network Access Code (P25)
	RFSS     uint8  `json:"rfss,omitempty" yaml:"rfss,omitempty"`
	Site     uint8  `json:"site,omitempty" yaml:"site,omitempty"`
	LRA      uint8  `json:"lra,omitempty" yaml:"lra,omitempty"`

	// PrimaryCC is the camped site's primary control channel (P25), with its
	// resolved downlink frequency when the band plan was known.
	PrimaryCC *TopoChannelRef `json:"primary_cc,omitempty" yaml:"primary_cc,omitempty"`

	// Per-protocol identity (populated where applicable).
	ColorCode    uint8  `json:"color_code,omitempty" yaml:"color_code,omitempty"`       // DMR
	RAN          uint8  `json:"ran,omitempty" yaml:"ran,omitempty"`                     // NXDN
	MCC          uint16 `json:"mcc,omitempty" yaml:"mcc,omitempty"`                     // TETRA
	MNC          uint16 `json:"mnc,omitempty" yaml:"mnc,omitempty"`                     // TETRA
	LocationArea uint16 `json:"location_area,omitempty" yaml:"location_area,omitempty"` // TETRA LA / NXDN location

	// Secondary control channels advertised for the camped site.
	Secondary []TopoChannelRef `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	// Neighbors are adjacent sites the control channel advertised.
	Neighbors []TopoNeighborRef `json:"neighbors,omitempty" yaml:"neighbors,omitempty"`
	// BandPlan maps channel IDs to frequencies (P25 IDEN_UP and equivalents).
	BandPlan []TopoBandPlanSlot `json:"band_plan,omitempty" yaml:"band_plan,omitempty"`
}

// TopoChannelRef is a channel identified by its band-plan (id, number)
// coordinates and, when resolvable, an absolute frequency.
type TopoChannelRef struct {
	ChannelID     uint8  `json:"channel_id" yaml:"channel_id"`
	ChannelNumber uint16 `json:"channel_number" yaml:"channel_number"`
	FrequencyHz   uint32 `json:"frequency_hz,omitempty" yaml:"frequency_hz,omitempty"`
	// UplinkHz is the channel's uplink (MS transmit) frequency when the protocol
	// resolves it directly rather than via a band-plan transmit offset — TETRA
	// derives it from the SYSINFO duplex spacing (§21.4.4.1). Zero when unknown or
	// when the uplink is instead computed from a band plan (P25).
	UplinkHz uint32 `json:"uplink_hz,omitempty" yaml:"uplink_hz,omitempty"`
}

// TopoNeighborRef is an adjacent site advertised by the control channel.
type TopoNeighborRef struct {
	RFSS          uint8  `json:"rfss" yaml:"rfss"`
	Site          uint8  `json:"site" yaml:"site"`
	ChannelID     uint8  `json:"channel_id,omitempty" yaml:"channel_id,omitempty"`
	ChannelNumber uint16 `json:"channel_number,omitempty" yaml:"channel_number,omitempty"`
	FrequencyHz   uint32 `json:"frequency_hz,omitempty" yaml:"frequency_hz,omitempty"`
}

// TopoBandPlanSlot is one entry of a P25-style band plan (IDEN_UP): a channel
// ID mapped to a base frequency, channel spacing, and transmit offset.
type TopoBandPlanSlot struct {
	ChannelID   uint8  `json:"channel_id" yaml:"channel_id"`
	BaseHz      uint64 `json:"base_hz" yaml:"base_hz"`
	SpacingHz   uint32 `json:"spacing_hz" yaml:"spacing_hz"`
	BandwidthHz uint32 `json:"bandwidth_hz,omitempty" yaml:"bandwidth_hz,omitempty"`
	TxOffsetHz  int64  `json:"tx_offset_hz,omitempty" yaml:"tx_offset_hz,omitempty"`
	AccessTDMA  bool   `json:"access_tdma,omitempty" yaml:"access_tdma,omitempty"`
}

// TopologyProvider is the optional hook a control-channel pipeline implements
// to expose its accumulated topology after a run. The siglab engine queries it
// once at EOF and attaches the snapshot to the Result. Returning nil (or an
// empty snapshot) is fine for protocols/runs that observed no topology.
type TopologyProvider interface {
	TopologySnapshot() *TopologySnapshot
}

// Empty reports whether the snapshot carries no usable information, so callers
// can avoid attaching a hollow snapshot to a Result.
func (t *TopologySnapshot) Empty() bool {
	if t == nil {
		return true
	}
	return t.WACN == 0 && t.SystemID == 0 && t.RFSS == 0 && t.Site == 0 && t.LRA == 0 &&
		t.ColorCode == 0 && t.RAN == 0 && t.MCC == 0 && t.MNC == 0 && t.LocationArea == 0 &&
		len(t.Secondary) == 0 && len(t.Neighbors) == 0 && len(t.BandPlan) == 0
}
