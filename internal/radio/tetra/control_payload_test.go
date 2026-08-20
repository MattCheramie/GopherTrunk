package tetra

import (
	"io"
	"log/slog"
	"testing"
)

// buildBSCHOnlySBStream is buildCleanSBStream with an idle (all-zero) BNCH:
// the BSCH decodes and stamps the ordinary activity heartbeat, but no SCH
// block decodes — the "locked but deaf" ingredient (on air, a wrong AFC alias
// latch produces exactly this: BSCH survives, every SCH fails CRC).
func buildBSCHOnlySBStream(t *testing.T, cc6 uint8, mcc, mnc uint16) []uint8 {
	t.Helper()
	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, uint32(cc6))
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	var s []uint8
	s = append(s, make([]uint8, 50)...)
	s = append(s, bschDibits...)
	s = append(s, SyncTrainingDibits()...)
	s = append(s, make([]uint8, sbBroadcastDibits)...)
	s = append(s, make([]uint8, 108)...) // BNCH idle: must not decode
	s = append(s, make([]uint8, 300)...)
	return s
}

// TestPayloadHeartbeatStampsOnSCHDecode pins the payload/activity heartbeat
// split the pipeline's payload-drought check depends on: a BSCH-only sync
// burst stamps the ordinary activity heartbeat but NOT the payload one, while
// a full sync burst whose BNCH (SCH/HD-coded) decodes stamps both.
func TestPayloadHeartbeatStampsOnSCHDecode(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()

	cc.Process(buildBSCHOnlySBStream(t, 0x2D, 3, 5), 0)
	if cc.LastActivityNano() == 0 {
		t.Fatal("BSCH-only burst did not stamp the activity heartbeat")
	}
	if cc.LastPayloadNano() != 0 {
		t.Fatal("BSCH-only burst stamped the payload heartbeat — the drought check could never fire in the field failure state")
	}

	cc2, bus2 := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus2.Close()
	cc2.Process(buildCleanSBStream(t, 0x2D, 3, 5, cleanSysinfoPDU()), 0)
	if cc2.LastPayloadNano() == 0 {
		t.Fatal("a CRC-clean BNCH (SCH/HD) decode did not stamp the payload heartbeat")
	}
	if cc2.LastActivityNano() == 0 {
		t.Fatal("payload decode did not also stamp the activity heartbeat")
	}
}
