package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// countLowPowerWarns runs `windows` diagnostics flushes on ec — each spaced
// well past lowPowerWarnInterval so the 5 s throttle never suppresses — and
// returns how many "iq power very low" WARNs fired. chanIQ/wbIQ are reloaded
// before each flush so the per-window power split is stable.
func countLowPowerWarns(t *testing.T, ec *engineChannel, windows int, chanIQ, wbIQ []complex64) int {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	e := &Engine{log: slog.New(handler), channels: []*engineChannel{ec}, now: time.Now}
	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base) // seeds lastDiagAt (no flush)
	for i := 1; i <= windows; i++ {
		ec.pwr.Add(chanIQ)
		e.widebandPwr.Add(wbIQ)
		// 6 s spacing > lowPowerWarnInterval (5 s) so the throttle is never
		// the reason a window stays quiet — only the conventional-idle gate is.
		e.maybeLogDiagnostics(base.Add(time.Duration(i) * 6 * time.Second))
	}
	mu.Lock()
	defer mu.Unlock()
	n := 0
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			n++
		}
	}
	return n
}

// validVoiceLCHeaderBurst builds a FEC-valid Voice LC Header burst so a
// ConventionalChannel counts a real FECPass when it ingests it (mirrors the
// tier2 package's own test helper, kept local to this package).
func validVoiceLCHeaderBurst(t *testing.T) *dmr.Burst {
	t.Helper()
	flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x42, SrcAddr: 0x100}
	flcBytes := dmr.AssembleFLC(flc)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedVoiceLCHeader[i]
	}
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (cw[i>>3] >> uint(7-(i&7))) & 1
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

// TestLowPowerConventionalIdleWarnsOnce pins the DMR IPSC log-quieting fix: a
// conventional-DMR carrier that has never decoded (a parked/unused listed
// frequency) must warn about low power exactly ONCE, not every diagnostics
// window forever. This is the field-reported flood (a dead listed frequency
// producing 150+ identical WARNs). Before the fix this returns `windows`.
func TestLowPowerConventionalIdleWarnsOnce(t *testing.T) {
	const n = 4096
	bus := events.NewBus(1)
	defer bus.Close()
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "IPSC", FrequencyHz: 443_237_500})
	ec := &engineChannel{freqHz: 443_237_500, sysName: "IPSC", protoTag: "dmr-tier2", tier2Cnt: cc}

	if got := countLowPowerWarns(t, ec, 4, quietIQ(n), loudIQ(n)); got != 1 {
		t.Fatalf("low-power WARNs = %d, want 1 (a never-decoded conventional carrier warns once, not every window)", got)
	}
}

// TestLowPowerConventionalDecodedThenIdleIsSilent pins the other half: a
// conventional-DMR carrier that HAS decoded a call and then goes quiet between
// transmissions must not warn at all — an idle repeater is normal, not a
// mistune. This is the common healthy case (the user's 443.2375 carrier).
func TestLowPowerConventionalDecodedThenIdleIsSilent(t *testing.T) {
	const n = 4096
	bus := events.NewBus(8)
	defer bus.Close()
	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "IPSC", FrequencyHz: 443_237_500})
	// One FEC-valid Voice LC Header → FECPass > 0 (the carrier has proven it
	// is a real repeater), so subsequent idle windows must stay silent.
	cc.IngestBurst(validVoiceLCHeaderBurst(t), dmr.SlotType{ColorCode: 12, DataType: dmr.DTVoiceLCHeader})
	if cc.Counters().FECPass == 0 {
		t.Fatal("test setup: expected FECPass > 0 after a valid header")
	}
	ec := &engineChannel{freqHz: 443_237_500, sysName: "IPSC", protoTag: "dmr-tier2", tier2Cnt: cc}

	if got := countLowPowerWarns(t, ec, 4, quietIQ(n), loudIQ(n)); got != 0 {
		t.Fatalf("low-power WARNs = %d, want 0 (a repeater that has decoded must not warn when merely idle)", got)
	}
}

// TestLowPowerControlChannelStillRepeats pins that the quieting is scoped to
// conventional DMR: a trunked control channel (tier2Cnt == nil) that sits at
// the floor is a genuine ongoing fault and must keep warning every window.
func TestLowPowerControlChannelStillRepeats(t *testing.T) {
	const n = 4096
	ec := &engineChannel{freqHz: 450_125_000, sysName: "Main", protoTag: "p25-phase1"} // tier2Cnt nil

	if got := countLowPowerWarns(t, ec, 4, quietIQ(n), loudIQ(n)); got != 4 {
		t.Fatalf("low-power WARNs = %d, want 4 (a control channel keeps warning at the floor)", got)
	}
}
