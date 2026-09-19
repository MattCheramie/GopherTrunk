package widebandt2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestDecideTier2Health pins the DMR Tier II decode-health verdict rule the
// scanner-status signal meter reads. The DMR IPSC case (the reported bug) is
// the beacon-liveness path: a locked repeater whose only signalling this
// window is a CRC-valid idle beacon must read "clean", and must NOT flap to
// "marginal" during the ~5–9 s gaps between beacon trains.
func TestDecideTier2Health(t *testing.T) {
	t0 := time.Unix(1_700_000_000, 0).UTC()

	tests := []struct {
		name        string
		start       healthDecision
		now         time.Time
		locked      bool
		fecPassD    uint64
		fecFailD    uint64
		beaconD     uint64
		wantVerdict string
	}{
		{
			name:        "locked repeater, idle beacon this window → clean",
			now:         t0,
			locked:      true,
			beaconD:     1,
			wantVerdict: "clean",
		},
		{
			name:        "clean voice-header FEC → clean",
			now:         t0,
			locked:      true,
			fecPassD:    20,
			wantVerdict: "clean",
		},
		{
			name:        "a few FEC failures → marginal",
			now:         t0,
			locked:      true,
			fecPassD:    97,
			fecFailD:    3, // 3% error rate
			wantVerdict: "marginal",
		},
		{
			name:        "mostly failing FEC → poor",
			now:         t0,
			locked:      true,
			fecPassD:    1,
			fecFailD:    19, // 95% error rate
			wantVerdict: "poor",
		},
		{
			name:        "locked, idle gap within grace keeps the last verdict",
			start:       healthDecision{verdict: "clean", lastGoodAt: t0},
			now:         t0.Add(6 * time.Second), // < wbHealthDecodeGrace
			locked:      true,
			wantVerdict: "clean",
		},
		{
			name:        "locked, idle gap beyond grace decays to marginal",
			start:       healthDecision{verdict: "clean", lastGoodAt: t0},
			now:         t0.Add(wbHealthDecodeGrace + time.Second),
			locked:      true,
			wantVerdict: "marginal",
		},
		{
			name:        "not locked → no verdict",
			now:         t0,
			locked:      false,
			wantVerdict: "",
		},
		{
			// A handful of FEC checks (below wbHealthMinAttempts) is too small a
			// sample for an error-rate bucket, but a valid header still proves a
			// clean decode this window.
			name:        "sub-threshold FEC passes still read clean",
			now:         t0,
			locked:      true,
			fecPassD:    2,
			wantVerdict: "clean",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decideTier2Health(tc.start, tc.now, tc.locked, tc.fecPassD, tc.fecFailD, tc.beaconD)
			if got.verdict != tc.wantVerdict {
				t.Fatalf("decideTier2Health verdict = %q, want %q", got.verdict, tc.wantVerdict)
			}
		})
	}
}

// TestEngineChannelFoldHealthDMRTier2 drives a real DMR Tier II conventional
// channel to a lock via an idle beacon (the IPSC keep-alive), then folds its
// health for one window — the end-to-end read the wideband engine performs for
// GET /api/v1/scanner. The reported symptom was that a wideband DMR system
// produced NO scanner-status entry at all (decode: —, no dBFS); here it must
// surface as locked + clean + a signal level.
func TestEngineChannelFoldHealthDMRTier2(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	now := time.Unix(1_700_000_000, 0).UTC()

	cc := tier2.New(tier2.Options{
		Bus:         bus,
		SystemName:  "Fire2",
		FrequencyHz: 442_387_500,
		Now:         func() time.Time { return now },
	})
	// An ETSI Idle burst at the repeater's colour code both locks the channel
	// (site alive while idle) and counts as a CRC-valid beacon.
	idle := burstWithInfo96(tier2.IdleInfoPattern[:])
	cc.IngestBurst(idle, dmr.SlotType{ColorCode: 12, DataType: dmr.DTIdle})
	if !cc.Locked() {
		t.Fatal("precondition: channel should be locked after an idle beacon")
	}

	ec := &engineChannel{
		freqHz:   442_387_500,
		sysName:  "Fire2",
		protoTag: "dmr-tier2",
		tier2Cnt: cc,
	}
	dst := make(map[string]SystemHealth)
	ec.foldHealth(dst, now, -48.3)

	h, ok := dst["Fire2"]
	if !ok {
		t.Fatal("foldHealth produced no health entry for the system")
	}
	if !h.Locked {
		t.Error("Locked = false, want true (idle beacon locked the repeater)")
	}
	if !h.HasDecodeHealth || h.DecodeQuality != "clean" {
		t.Errorf("decode health = %q (has=%v), want clean/true (CRC-valid beacon)", h.DecodeQuality, h.HasDecodeHealth)
	}
	if !h.HasSignal || h.SignalDbFS != -48.3 {
		t.Errorf("signal = %v dBFS (has=%v), want -48.3/true", h.SignalDbFS, h.HasSignal)
	}
	if h.Protocol != "dmr-tier2" || h.FreqHz != 442_387_500 {
		t.Errorf("protocol/freq = %q/%d, want dmr-tier2/442387500", h.Protocol, h.FreqHz)
	}
	if h.LockedAt != now {
		t.Errorf("LockedAt = %v, want %v", h.LockedAt, now)
	}
}

// burstWithInfo96 builds a DMR burst whose 196-bit BPTC channel payload
// carries the supplied 12-byte (96-bit) information block. Mirrors the helper
// in the tier2 package's own tests so this package can inject a beacon burst.
func burstWithInfo96(info []byte) *dmr.Burst {
	if len(info) != 12 {
		panic("burstWithInfo96 requires a 12-byte info block")
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)

	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] =
			(channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}
