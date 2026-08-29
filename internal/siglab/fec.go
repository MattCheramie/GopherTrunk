package siglab

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/motorola"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	p25phase2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// rotateDibits returns dibits with each symbol cyclically rotated by rot
// (mod 4) — used to present a frame to a decoder under a candidate polarity.
func rotateDibits(dibits []uint8, rot uint8) []uint8 {
	out := make([]uint8, len(dibits))
	for i, d := range dibits {
		out[i] = (d + rot) & 3
	}
	return out
}

// dibitsToBitsRot de-rotates dibits by rot and expands to bits, MSB-first per
// dibit (matching framing.BitsToDibits' (bits[2i]<<1 | bits[2i+1]) packing).
func dibitsToBitsRot(dibits []uint8, rot uint8) []byte {
	out := make([]byte, len(dibits)*2)
	for i, d := range dibits {
		dd := (d + rot) & 3
		out[2*i] = (dd >> 1) & 1
		out[2*i+1] = dd & 1
	}
	return out
}

// This file holds the per-protocol FEC-outcome tallies — the symbol-domain
// analog of P25's NID-BCH probe. Each re-runs the protocol's exported FEC
// decoder on the frame found at every clean sync hit and tallies
// clean/corrected/uncorrectable (or CRC pass/fail). To stay robust against the
// sync-landscape's rotation/polarity ambiguity (e.g. DMR BS-Voice rot2 ≡
// BS-Data rot0), each frame is decoded under every cyclic rotation and the
// best outcome is kept — so a clean capture tallies as clean regardless of
// which equivalent rotation the landscape picked as winner.

// tallyErrs folds one decoder error count (−1 = uncorrectable, 0 = clean,
// >0 = corrected) into st.
func tallyErrs(st *FECStat, errs int) {
	st.Frames++
	switch {
	case errs < 0:
		st.Uncorrectable++
	case errs == 0:
		st.Clean++
	default:
		st.Corrected++
	}
}

// bestErrs returns the smallest non-negative error count from attempts, or −1
// when every attempt was uncorrectable.
func bestErrs(attempts []int) int {
	best := -1
	for _, e := range attempts {
		if e < 0 {
			continue
		}
		if best < 0 || e < best {
			best = e
		}
	}
	return best
}

// dmrSlotTypeFEC tallies the DMR slot-type Hamming(20,8) decode at each clean
// burst-sync hit. The 10 slot-type dibits flank the 24-dibit sync (5 before,
// 5 after — burst layout HalfPayload(49) + slot(5) + sync(24) + slot(5) +
// HalfPayload(49)).
func dmrSlotTypeFEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	st := FECStat{Stage: "slot-type Hamming(20,8)"}
	const half = dmr.SlotTypeDibits // 5
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		if pos < half || pos+syncLen+half > len(stream) {
			continue
		}
		slot := make([]uint8, 0, 2*half)
		slot = append(slot, stream[pos-half:pos]...)
		slot = append(slot, stream[pos+syncLen:pos+syncLen+half]...)

		var attempts []int
		for rot := uint8(0); rot < 4; rot++ {
			var cw uint32
			for i, d := range slot {
				dd := (d + rot) & 3
				cw |= uint32((dd>>1)&1) << uint(19-2*i)
				cw |= uint32(dd&1) << uint(19-2*i-1)
			}
			_, errs := framing.HammingDecode20_8(cw)
			attempts = append(attempts, errs)
		}
		tallyErrs(&st, bestErrs(attempts))
	}
	return []FECStat{st}
}

// edacsBCHFEC tallies the EDACS BCH(40,28,2) decode on the 40-bit CCW that
// follows each clean outbound-sync hit. The on-wire bit i maps to codeword bit
// (39−i) (see the fixture encoder).
func edacsBCHFEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	st := FECStat{Stage: "BCH(40,28,2)"}
	const cwLen = 40
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		start := pos + syncLen
		if start+cwLen > len(stream) {
			continue
		}
		var attempts []int
		for rot := uint8(0); rot < 2; rot++ {
			var cw uint64
			for i := 0; i < cwLen; i++ {
				if (stream[start+i]+rot)&1 != 0 {
					cw |= uint64(1) << uint(39-i)
				}
			}
			_, errs := framing.BCHDecodeEDACS(cw)
			attempts = append(attempts, errs)
		}
		tallyErrs(&st, bestErrs(attempts))
	}
	return []FECStat{st}
}

// motorolaOSWFEC tallies the SmartNet OSW decode (deinterleave →
// convolutional-parity ECC → CRC-10, motorola.DecodeOSWPayloadDetail)
// on the 76-bit payload that follows each outbound-sync hit. The sync
// is only 8 bits, so payload data produces chance hits; exactly like
// the live framer (rx_smartnet's sync bracket), a window only counts
// when the NEXT frame's sync sits 76 bits later — otherwise it is a
// misalignment, not a frame.
func motorolaOSWFEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	st := FECStat{Stage: "SmartNet ECC + CRC-10"}
	syncLen := land.SyncLen
	syncBits := motorola.OutboundSyncBits()
	for _, pos := range land.positions {
		start := pos + syncLen
		if start+motorola.PayloadBits+syncLen > len(stream) {
			continue
		}
		bracketed := true
		for i, b := range syncBits {
			if stream[start+motorola.PayloadBits+i]&1 != b {
				bracketed = false
				break
			}
		}
		if !bracketed {
			continue
		}
		_, flips, ok := motorola.DecodeOSWPayloadDetail(stream[start : start+motorola.PayloadBits])
		if ok {
			st.CRCPass++
			tallyErrs(&st, flips)
		} else {
			st.CRCFail++
			tallyErrs(&st, -1)
		}
	}
	return []FECStat{st}
}

// dstarCRCFEC tallies the D-STAR header CRC-16 over the 41-byte (FEC-off)
// header that follows each clean frame-sync hit. Bytes are packed MSB-first;
// the CRC sits in bytes 39..40 (big-endian) over bytes 0..38.
func dstarCRCFEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	st := FECStat{Stage: "header CRC-16"}
	const hdrBits = 328
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		start := pos + syncLen
		if start+hdrBits > len(stream) {
			continue
		}
		pass := false
		for rot := uint8(0); rot < 2; rot++ {
			var b [41]byte
			for i := 0; i < 41; i++ {
				var v byte
				for j := 0; j < 8; j++ {
					v = (v << 1) | ((stream[start+i*8+j] + rot) & 1)
				}
				b[i] = v
			}
			got := uint16(b[39])<<8 | uint16(b[40])
			if dstar.ComputeCRC(b[:39]) == got {
				pass = true
				break
			}
		}
		st.Frames++
		if pass {
			st.CRCPass++
		} else {
			st.CRCFail++
		}
	}
	return []FECStat{st}
}

// nxdnFEC tallies the NXDN LICH Hamming-doubling decode and the CAC Viterbi +
// CRC-16 decode. The frame after each clean FSW hit is LICH (8 wire dibits) +
// CAC (CACChannelBits/2 = 150 dibits); the fixture omits the SACCH so the CAC
// follows the LICH directly.
func nxdnFEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	lich := FECStat{Stage: "LICH Hamming-doubling"}
	cac := FECStat{Stage: "CAC Viterbi(K=5)+CRC-16"}
	const lichDibits = 8
	cacDibits := nxdn.CACChannelBits / 2
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		lichStart := pos + syncLen
		cacStart := lichStart + lichDibits
		if cacStart+cacDibits > len(stream) {
			continue
		}
		bestDis := -1
		for rot := uint8(0); rot < 4; rot++ {
			_, dis := nxdn.DecodeLICHWire(dibitsToBitsRot(stream[lichStart:lichStart+lichDibits], rot))
			if bestDis < 0 || dis < bestDis {
				bestDis = dis
			}
		}
		tallyErrs(&lich, bestDis)

		pass := false
		for rot := uint8(0); rot < 4; rot++ {
			if _, ok := nxdn.DecodeCACChannel(dibitsToBitsRot(stream[cacStart:cacStart+cacDibits], rot)); ok {
				pass = true
				break
			}
		}
		cac.Frames++
		if pass {
			cac.CRCPass++
		} else {
			cac.CRCFail++
		}
	}
	return []FECStat{lich, cac}
}

// tetraFEC tallies the TETRA SCH/HD channel decode (descramble + deinterleave +
// depuncture + RCPC Viterbi + CRC-16) over the 108-dibit type-5 burst that
// follows each clean normal-sync hit. The colour code seeds the descrambler.
func tetraFEC(stream []uint8, land *SyncLandscape, sys trunking.System) []FECStat {
	st := FECStat{Stage: "SCH/HD RCPC+Viterbi+CRC-16"}
	const schDibits = 108 // 216 type-5 bits
	syncLen := land.SyncLen
	cc := sys.TETRAColourCode
	// Blind identify has no configured colour code, but SCH/HD is scrambled
	// with the cell's colour code — descrambling with 0 fails on every real
	// cell, forfeiting TETRA's whole FEC score (issue #648). The BSCH (colour 0)
	// carries the code, so recover it from a synchronisation burst in the same
	// stream and descramble SCH/HD with it.
	if cc == 0 {
		if recovered, ok := tetra.RecoverColourCode(stream); ok {
			cc = recovered
		}
	}
	for _, pos := range land.positions {
		start := pos + syncLen
		if start+schDibits > len(stream) {
			continue
		}
		pass := false
		for rot := uint8(0); rot < 4; rot++ {
			// TETRA uses the π/4-DQPSK Gray bit↔dibit convention; the
			// rotation covers the differential decoder's constant-phase
			// ambiguity.
			bits := tetra.TetraDibitsToBits(rotateDibits(stream[start:start+schDibits], rot))
			if _, ok := tetra.DecodeSCHHD(bits, cc); ok {
				pass = true
				break
			}
		}
		st.Frames++
		if pass {
			st.CRCPass++
		} else {
			st.CRCFail++
		}
	}
	return []FECStat{st}
}

// p25p2FEC tallies the P25 Phase 2 ISCH Golay(24,12,8) decode and the MAC-PDU
// ½-rate trellis decode. After each outbound-sync hit (only sub-frame 0 of a
// superframe carries the sync) the ISCH is 12 dibits at offset SyncDibits and
// the trellis-coded MAC payload is 146 dibits after that.
func p25p2FEC(stream []uint8, land *SyncLandscape, _ trunking.System) []FECStat {
	isch := FECStat{Stage: "ISCH Golay(24,12,8)"}
	mac := FECStat{Stage: "MAC trellis(1/2)"}
	const macDibits = 146
	ischDibits := p25phase2.ISCHDibits
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		ischStart := pos + syncLen
		if ischStart+ischDibits > len(stream) {
			continue
		}
		best := -1
		for rot := uint8(0); rot < 4; rot++ {
			if _, errs, err := p25phase2.DecodeISCH(rotateDibits(stream[ischStart:ischStart+ischDibits], rot)); err == nil {
				if best < 0 || errs < best {
					best = errs
				}
			}
		}
		tallyErrs(&isch, best)

		macStart := ischStart + ischDibits
		if macStart+macDibits <= len(stream) {
			bestM := -1
			for rot := uint8(0); rot < 4; rot++ {
				if _, metric := framing.DecodeP25Trellis(rotateDibits(stream[macStart:macStart+macDibits], rot)); bestM < 0 || metric < bestM {
					bestM = metric
				}
			}
			tallyErrs(&mac, bestM)
		}
	}
	return []FECStat{isch, mac}
}
