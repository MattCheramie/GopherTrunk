package trunking

import "time"

// Location is the payload of an events.KindLocation event: a
// geographic fix a subscriber unit reported over the air. P25 Motorola
// Unit GPS, P25 L3Harris Talker GPS, and DMR LRRP all ultimately
// resolve to one of these. Latitude is positive north, Longitude
// positive east, both in decimal degrees.
//
// The storage layer persists every Location to the location_log table
// and the API surfaces recent fixes for map display.
type Location struct {
	System     string  // trunking system name
	Protocol   string  // "p25", "dmr", ...
	RadioID    uint32  // reporting subscriber unit
	Talkgroup  uint32  // associated talkgroup; 0 when not call-associated
	Latitude   float64 // decimal degrees, positive north
	Longitude  float64 // decimal degrees, positive east
	SpeedKnots float64 // 0 when not reported
	HeadingDeg float64 // 0 when not reported
	At         time.Time
}
