package phase2

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// The MMR fixtures are the production receiver's dibits (receiver.New with the
// voice chain's options, timing acquisition armed) for a real P25 Phase 2 TDMA
// carrier: the MMR (Victoria) voice channel at 420.650 MHz, idle, in
// samples/mmr/mmr-s9-1.cfile and mmr-s9-3.cfile (RTL-SDR, 2.4 MS/s centred on
// 420.0125 MHz, the carrier at +637.5 kHz, through the composer's Phase 2
// voice front end). The site's Phase 1 control channel in the same captures is
// WACN BEE00 / system 164 / NAC 164, and its colour code seeds the PN44
// descrambler. mmr-s9-3 has a hole in it about 0.26 s in: the receiver's
// symbol count stays exact, but the on-air timeslot grid moves by 155 dibits,
// consistent with 25 dropped 16 KiB USB buffers (8192 samples each).
const (
	mmrDibitsClean = "testdata/mmr-s9-1-420650k-p2.dibits"
	mmrDibitsHole  = "testdata/mmr-s9-3-420650k-p2-hole.dibits"
)

func mmrSeed() uint64 { return framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x164) }

// frameMMR feeds a dibit fixture through an AirFramer in the receiver's
// ~82-dibit chunks (655 samples at 8 samples/symbol) and returns the bursts.
func frameMMR(t *testing.T, path string) []AirBurst {
	t.Helper()
	d, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := NewAirFramer()
	var out []AirBurst
	for i := 0; i < len(d); i += 82 {
		out = append(out, f.Process(d[i:min(i+82, len(d))], i)...)
	}
	return out
}

// idleDUID is what every burst of this idle carrier carries: the SACCH on
// timeslots 10 and 11, DATCH fill elsewhere.
func idleDUID(slot int) DUID {
	if slot >= 10 {
		return DUIDSACCHScrambled
	}
	return DUIDDATCHScrambled
}

// TestRealAirMMRPhase2MACDecodes is the first real-air pin of the rebuilt
// Phase 2 MAC path (PR #1290: air-format framing, RS(63,35), CRC-12, PN44):
// the idle carrier's SACCH bursts decode CRC-valid with the seed built from
// the control channel's own identity, and with no other. Their content
// cross-checks against the Phase 1 control channel in the same capture, which
// announces Motorola supergroup 3202 (0x0C82) patching talkgroup 201 (0x00C9):
// the Phase 2 SACCH carries that regroup, MFID 0x90 opcode 0x81.
func TestRealAirMMRPhase2MACDecodes(t *testing.T) {
	bursts := frameMMR(t, mmrDibitsClean)
	right := NewAirMasks(mmrSeed())
	wrong := NewAirMasks(framing.PN44SeedFromIdentity(0xBEE00, 0x164, 0x165))
	var sacch, good, wrongGood int
	regroup := false
	wantRegroup, _ := hex.DecodeString("81900b0c8200c900c900c9")
	for _, b := range bursts {
		if b.DUID != DUIDSACCHScrambled {
			continue
		}
		sacch++
		if m, ok := DecodeAirMAC(b, right); ok {
			good++
			if m.Type != AirMACIdle {
				t.Errorf("SACCH at slot %d decoded as %v, want MAC_IDLE on an idle channel", b.Slot, m.Type)
			}
			if bytes.Contains(m.Bytes, wantRegroup) {
				regroup = true
			}
		}
		if _, ok := DecodeAirMAC(b, wrong); ok {
			wrongGood++
		}
	}
	t.Logf("%d bursts, %d SACCH-S, %d CRC-valid with the site's seed, %d with colour code 0x165", len(bursts), sacch, good, wrongGood)
	if good < 4 {
		t.Errorf("only %d of %d SACCH-S bursts decoded with the site's seed", good, sacch)
	}
	if wrongGood != 0 {
		t.Errorf("%d SACCH-S bursts decoded with a wrong seed: the descramble is not what makes them valid", wrongGood)
	}
	if !regroup {
		t.Error("no SACCH carried the supergroup 3202 / talkgroup 201 regroup the control channel announces")
	}
}

// TestAirFramerReanchorsAfterIQHole is the issue #1364 regression for a hole
// in the IQ (dropped USB buffers, a lagging broker subscriber) that the dibit
// counter does not see: the on-air timeslot grid moves under the framer. It
// used to keep cutting packets on the stale grid until two superframes passed
// without a recognised ISCH (airLockLossDibits, 720 ms), handing out ~20
// garbage bursts with random DUIDs (voice among them) and losing the
// signalling. It now looks ahead for an S-ISCH off its grid once two
// consecutive packets miss theirs, and moves there.
func TestAirFramerReanchorsAfterIQHole(t *testing.T) {
	bursts := frameMMR(t, mmrDibitsHole)
	masks := NewAirMasks(mmrSeed())
	var bad, sacchOK int
	for _, b := range bursts {
		if b.DUID != idleDUID(b.Slot) {
			bad++
		}
		if b.DUID == DUIDSACCHScrambled {
			if _, ok := DecodeAirMAC(b, masks); ok {
				sacchOK++
			}
		}
	}
	t.Logf("%d bursts, %d off the idle pattern, %d SACCH-S CRC-valid", len(bursts), bad, sacchOK)
	if bad > 3 {
		t.Errorf("%d bursts carried a DUID the idle carrier never sends: the framer stayed on the stale grid after the hole", bad)
	}
	if sacchOK < 5 {
		t.Errorf("only %d SACCH-S bursts decoded", sacchOK)
	}
}
