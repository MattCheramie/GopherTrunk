package siglab

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

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
func dmrSlotTypeFEC(stream []uint8, land *SyncLandscape) []FECStat {
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
func edacsBCHFEC(stream []uint8, land *SyncLandscape) []FECStat {
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

// motorolaBCHFEC tallies the Motorola BCH(64,16,11) decode on the two 64-bit
// OSW halves (128 bits) that follow each clean outbound-sync hit. On-wire bit
// i maps to codeword bit (63−i).
func motorolaBCHFEC(stream []uint8, land *SyncLandscape) []FECStat {
	st := FECStat{Stage: "BCH(64,16,11)"}
	syncLen := land.SyncLen
	for _, pos := range land.positions {
		start := pos + syncLen
		if start+128 > len(stream) {
			continue
		}
		for half := 0; half < 2; half++ {
			base := start + half*64
			var attempts []int
			for rot := uint8(0); rot < 2; rot++ {
				var cw uint64
				for i := 0; i < 64; i++ {
					if (stream[base+i]+rot)&1 != 0 {
						cw |= uint64(1) << uint(63-i)
					}
				}
				_, errs := framing.BCHDecode64_16(cw)
				attempts = append(attempts, errs)
			}
			tallyErrs(&st, bestErrs(attempts))
		}
	}
	return []FECStat{st}
}

// dstarCRCFEC tallies the D-STAR header CRC-16 over the 41-byte (FEC-off)
// header that follows each clean frame-sync hit. Bytes are packed MSB-first;
// the CRC sits in bytes 39..40 (big-endian) over bytes 0..38.
func dstarCRCFEC(stream []uint8, land *SyncLandscape) []FECStat {
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
