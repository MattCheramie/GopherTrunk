package voice

import (
	"encoding/hex"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// PIHeaderDetector finds Privacy Indicator header bursts in a followed
// traffic-channel dibit stream — the data burst an encrypted DMR
// transmission sends after its Voice LC Header to announce the algorithm,
// key ID and Message Indicator the voice that follows is scrambled with
// (issue #1187). The voice Decoder locks on voice sync and never sees it;
// this detector runs alongside, exactly like TerminatorDetector, reusing
// the same trusted decode chain: data-sync detection, 132-dibit burst
// slicing at both discriminator polarities, slot-type Hamming(20,8),
// BPTC(196,96), then dmr.ParsePIHeader's CRC-CCITT. A header is reported
// only when every layer clears.
//
// A BPTC-clean burst whose slot type says PI header but whose CRC fails is
// counted and its raw octets kept (Rejects), the instrument for a vendor
// layout or CRC convention this parser does not know — the first on-air
// capture decides, not a guess.
type PIHeaderDetector struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int
	pending  []dmr.Match

	rejects    int
	lastReject string
}

// NewPIHeaderDetector returns a ready detector.
func NewPIHeaderDetector() *PIHeaderDetector {
	return &PIHeaderDetector{det: dmr.NewSyncDetector(terminatorSyncs, 2)}
}

// Process consumes a window of dibits and returns every valid PI header
// whose burst completed within it. baseIdx is the absolute dibit index of
// dibits[0]; it must be monotonically non-decreasing across calls.
func (d *PIHeaderDetector) Process(dibits []uint8, baseIdx int) []dmr.PIHeader {
	if len(d.buf) == 0 {
		d.bufStart = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	matches, _ := d.det.Process(nil, dibits, baseIdx)
	d.pending = append(d.pending, matches...)

	var out []dmr.PIHeader
	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, m := range d.pending {
		burstStart := m.Index - termBurstLookback
		burstEnd := m.Index + termBurstLookahead + 1
		if burstEnd > bufEnd {
			keep = append(keep, m)
			continue
		}
		if burstStart < d.bufStart {
			continue
		}
		offset := burstStart - d.bufStart
		if h, ok := d.decodePIHeader(d.buf[offset : offset+dmr.BurstDibits]); ok {
			out = append(out, h)
		}
	}
	d.pending = keep

	if len(d.buf) > termBufKeep {
		drop := len(d.buf) - termBufKeep
		copy(d.buf, d.buf[drop:])
		d.buf = d.buf[:termBufKeep]
		d.bufStart += drop
	}
	return out
}

// Rejects reports how many BPTC-clean PI-header bursts failed the CRC, and
// the hex of the last one's 12 recovered octets.
func (d *PIHeaderDetector) Rejects() (int, string) { return d.rejects, d.lastReject }

func (d *PIHeaderDetector) decodePIHeader(burstDibits []uint8) (dmr.PIHeader, bool) {
	for _, k := range dmr.CandidatePolarities {
		var b dmr.Burst
		copy(b.Dibits[:], burstDibits)
		dmr.RotateBurstDibits(&b, k)

		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil || slot.DataType != dmr.DTPIHeader {
			continue
		}
		bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
		if errs < 0 {
			continue
		}
		info := infoBitsToBytes96(bits)
		h, err := dmr.ParsePIHeader(info)
		if err != nil {
			d.rejects++
			d.lastReject = hex.EncodeToString(info)
			continue
		}
		return h, true
	}
	return dmr.PIHeader{}, false
}
