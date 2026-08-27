package nxdn

import (
	"encoding/binary"
	"log/slog"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildSpecCACInfo returns a 155-bit §4.5.1.1 info block whose L3 prefix
// carries a SITEINFO CAC (the layout packCACBlockFromInfo repacks): 8 SR
// bits, then the 8-bit RCCH message type, then the 64-bit payload, then
// filler L3 + 3 null bits.
func buildSpecCACInfo() []byte {
	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA) // LocationID
	binary.BigEndian.PutUint16(payload[2:4], 0x1234) // SiteID
	binary.BigEndian.PutUint16(payload[4:6], 0x5678) // SystemID

	info := make([]byte, CACInfoBits)
	typ := byte(RCCHSITEINFO)
	for i := 0; i < 8; i++ {
		info[8+i] = (typ >> uint(7-i)) & 1
	}
	for b := 0; b < 8; b++ {
		for i := 0; i < 8; i++ {
			info[16+8*b+i] = (payload[b] >> uint(7-i)) & 1
		}
	}
	return info
}

// buildSpecStream assembles the ViterbiSpec on-air dibit stream: padding +
// FSW + LICH + spec-encoded 150-dibit CAC, plus the parallel per-bit LLR
// track (2 LLRs per dibit, MSB then LSB) with AWGN of the given sigma. The
// hard dibits are sliced from the same noisy LLRs, so the hard and soft
// paths see one consistent channel.
func buildSpecStream(t *testing.T, sigma float64, rng *rand.Rand) (dibits []uint8, llrs []float32) {
	t.Helper()
	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)

	cacChannel := EncodeCACChannel(buildSpecCACInfo())
	if cacChannel == nil {
		t.Fatal("EncodeCACChannel returned nil")
	}

	// Clean bit stream: padding + FSW + LICH + CAC.
	bits := make([]byte, 0, 2*(30+8+8+150))
	appendDibits := func(ds []uint8) {
		for _, d := range ds {
			bits = append(bits, (d>>1)&1, d&1)
		}
	}
	appendDibits(make([]uint8, 30))
	appendDibits(FSWDibitsOutbound)
	appendDibits(lichDibits)
	bits = append(bits, cacChannel...)

	// Noise ONLY on the CAC region: the FSW/LICH must still detect (their
	// robustness is not under test) while the CAC channel bits carry AWGN.
	cacStart := len(bits) - len(cacChannel)
	llrs = make([]float32, len(bits))
	dibits = make([]uint8, len(bits)/2)
	for i, b := range bits {
		v := 1.0
		if b == 1 {
			v = -1.0
		}
		if i >= cacStart {
			v += rng.NormFloat64() * sigma
		}
		llrs[i] = float32(v)
	}
	for i := range dibits {
		var msb, lsb uint8
		if llrs[2*i] < 0 {
			msb = 1
		}
		if llrs[2*i+1] < 0 {
			lsb = 1
		}
		dibits[i] = msb<<1 | lsb
	}
	return dibits, llrs
}

// TestProcessSoftLocksWhereHardFails is the failing-first CC-level pin for
// nxdn_soft_decision: at an AWGN level where the hard Process path fails
// the CAC CRC on every burst (no lock ever — KindCCLocked is edge-triggered,
// so lock count is the decoded-anything signal), ProcessSoft's per-bit soft
// Viterbi recovers a SITEINFO and locks. The channel is fully seeded, so
// the outcome is deterministic — no flake window.
func TestProcessSoftLocksWhereHardFails(t *testing.T) {
	const sigma = 0.9
	const bursts = 30

	countLocks := func(soft bool) int {
		bus := events.NewBus(64)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()
		cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
		cc.SetViterbiMode(ViterbiSpec)

		rng := rand.New(rand.NewSource(42)) // same channel for both paths
		base := 0
		for burst := 0; burst < bursts; burst++ {
			dibits, llrs := buildSpecStream(t, sigma, rng)
			if soft {
				base = cc.ProcessSoft(dibits, llrs, base)
			} else {
				base = cc.Process(dibits, base)
			}
		}
		locks := 0
		for {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLocked {
					locks++
				}
			default:
				return locks
			}
		}
	}

	hardLocks := countLocks(false)
	softLocks := countLocks(true)
	t.Logf("sigma=%.2f over %d bursts: hard locks=%d soft locks=%d", sigma, bursts, hardLocks, softLocks)
	if softLocks == 0 {
		t.Error("ProcessSoft never locked on the noisy stream (soft CAC decode not wired)")
	}
	// The failing-first shape: at this SNR the hard path decodes nothing at
	// all on this (seeded, deterministic) channel, so before the soft path
	// existed there was no lock — exactly what the assertion above catches.
	if hardLocks != 0 {
		t.Errorf("hard path locked (%d) at a noise level chosen to defeat it — fixture no longer discriminates", hardLocks)
	}
}

// TestProcessSoftCleanMatchesHard: on a noiseless stream the soft path must
// ingest exactly what the hard path does — same lock, no regression on
// clean signals.
func TestProcessSoftCleanMatchesHard(t *testing.T) {
	for _, soft := range []bool{false, true} {
		bus := events.NewBus(8)
		sub := bus.Subscribe()
		cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)
		cc.SetViterbiMode(ViterbiSpec)

		rng := rand.New(rand.NewSource(1))
		dibits, llrs := buildSpecStream(t, 0, rng)
		if soft {
			cc.ProcessSoft(dibits, llrs, 0)
		} else {
			cc.Process(dibits, 0)
		}
		locked := false
		for {
			done := false
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLocked {
					locked = true
				}
			default:
				done = true
			}
			if done {
				break
			}
		}
		if !locked {
			t.Errorf("soft=%v: clean spec stream did not lock", soft)
		}
		sub.Close()
		bus.Close()
	}
}
