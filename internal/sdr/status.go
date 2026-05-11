package sdr

// SDRStatus is the per-device snapshot the pool publishes on the events
// bus when a device is opened or closed, and the same payload returned
// by GET /api/v1/devices. Fields that are unknown at snapshot time
// (e.g. the daemon never programmed a gain because the YAML left it
// blank) are zero-valued; consumers should treat that as "default" /
// "unset" rather than "explicitly zero".
//
// The shape mirrors the `gophertrunk.v1.SDRStatus` proto message but
// keeps the JSON layer self-contained so the api package doesn't have
// to import the pb generated types just to render the response.
type SDRStatus struct {
	Driver       string `json:"driver"`
	Serial       string `json:"serial"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	TunerName    string `json:"tuner_name,omitempty"`
	Role         string `json:"role"`
	Attached     bool   `json:"attached"`

	// Configured hint values applied at open time. PPM is in
	// parts-per-million; GainTenthDB follows the SetGain convention
	// (negative = AGC). BiasTee reflects whether the YAML asked the
	// pool to enable the 5 V output.
	GainTenthDB  int  `json:"gain_tenth_db"`
	GainAuto     bool `json:"gain_auto"`
	PPM          int  `json:"ppm"`
	BiasTee      bool `json:"bias_tee"`

	// Gains is the tuner's quantized gain ladder (tenths of dB),
	// useful for UIs that want to render valid choices.
	Gains []int `json:"gains,omitempty"`
}
