package ltr

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// statusFCSMessageBits builds the 24-bit message vector that the
// CRC-7 covers, per sdrtrunk's CRCLTR.java layout:
//
//	index 0      sdrtrunk Area    — 1 bit  — mapped from
//	                                Status.Group (the F-bit)
//	indices 1..5 sdrtrunk Channel — 5 bits — Status.Channel
//	                                MSB-first
//	indices 6..10 sdrtrunk Home    — 5 bits — Status.Home
//	                                MSB-first
//	indices 11..18 sdrtrunk Group  — 8 bits — Status.GroupID
//	                                MSB-first
//	indices 19..23 sdrtrunk Free   — 5 bits — Status.Free
//	                                MSB-first
//
// The gophertrunk Status struct's 5-bit Area field doesn't map
// into the 24-bit FCS-protected message — it's used elsewhere in
// the package (multi-system filter) and isn't part of the CRC.
func statusFCSMessageBits(s Status) []byte {
	msg := make([]byte, 24)
	if s.Group {
		msg[0] = 1
	}
	// Channel: 5 bits MSB-first.
	for i := 0; i < 5; i++ {
		msg[1+i] = (s.Channel >> uint(4-i)) & 1
	}
	// Home: 5 bits MSB-first.
	for i := 0; i < 5; i++ {
		msg[6+i] = (s.Home >> uint(4-i)) & 1
	}
	// GroupID: 8 bits MSB-first.
	for i := 0; i < 8; i++ {
		msg[11+i] = byte((s.GroupID >> uint(7-i)) & 1)
	}
	// Free: 5 bits MSB-first.
	for i := 0; i < 5; i++ {
		msg[19+i] = (s.Free >> uint(4-i)) & 1
	}
	return msg
}

// ComputeStatusFCS returns the 7-bit LTR Standard CRC computed
// over the message bits derived from Status. Use this to populate
// the Status.FCS field of a synthesized frame so the
// FCSOn-equipped Ingest path accepts it.
func ComputeStatusFCS(s Status) uint8 {
	return framing.CRC7LTR(statusFCSMessageBits(s))
}

// verifyStatusFCS reports whether the low 7 bits of Status.FCS
// match the CRC-7 of the message bits derived from Status.
func verifyStatusFCS(s Status) bool {
	want := framing.CRC7LTR(statusFCSMessageBits(s))
	return uint8(s.FCS&0x7F) == want
}
