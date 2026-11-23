package dpmr

import (
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// beaconTx builds a beacon downlink frameset by frameset.
type beaconTx struct {
	r      *rand.Rand
	dibits []uint8
	sys    uint16
}

func newBeaconTx(seed int64, sys uint16) *beaconTx {
	tx := &beaconTx{r: rand.New(rand.NewSource(seed)), sys: sys}
	tx.noise(400)
	tx.dibits = append(tx.dibits, AirFS1Dibits()...)
	return tx
}

func (tx *beaconTx) noise(n int) {
	for i := 0; i < n; i++ {
		tx.dibits = append(tx.dibits, uint8(tx.r.Intn(4)))
	}
}

// idle sends a frameset with a message and SYScast1 carrying the system
// code; the message is not an Aloha unless aloha is set.
func (tx *beaconTx) frameset(m BeaconMessage, sy2, sy3 uint32) {
	sy1 := EncodeSYScast1(SYScast1{SystemCode: tx.sys})
	tx.dibits = append(tx.dibits, EncodeBeaconFrameset(m, BeaconChannelCode0, [3]uint32{sy1, sy2, sy3})...)
}

func (tx *beaconTx) aloha() {
	tx.frameset(EncodeAloha(Aloha{MSID: 0xFFFFFF, SystemCode: tx.sys}), EncodeSYScast2(SYTypeFrameCounter, 1), EncodeSYScast2(SYTypeFrameCounter, 1))
}

// other sends a message that is neither an Aloha nor a Goto Channel.
func (tx *beaconTx) other() {
	var m BeaconMessage
	m.put(0, 4, MTAhoy)
	tx.frameset(m, EncodeSYScast2(SYTypeFrameCounter, 2), EncodeSYScast2(SYTypeFrameCounter, 2))
}

func (tx *beaconTx) gotoChannel(g GotoChannel, calling uint32) {
	a, b := EncodeCallingParty(calling)
	tx.frameset(EncodeGotoChannel(g), a, b)
}

// runCC feeds dibits through a ControlChannel and collects its events.
func runCC(t *testing.T, d []uint8) (*ControlChannel, []events.Event) {
	t.Helper()
	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	defer sub.Close()
	clock := time.Unix(1_700_000_000, 0)
	cc := New(Options{Bus: bus, SystemName: "DPMR", FrequencyHz: 446_100_000, Now: func() time.Time {
		clock = clock.Add(10 * time.Millisecond)
		return clock
	}})
	base := 0
	for i := 0; i < len(d); i += 480 {
		base = cc.Process(d[i:min(i+480, len(d))], base)
	}
	var evs []events.Event
	for {
		select {
		case ev := <-sub.C:
			evs = append(evs, ev)
		default:
			return cc, evs
		}
	}
}

func kinds(evs []events.Event, k events.Kind) []events.Event {
	var out []events.Event
	for _, e := range evs {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

// An Aloha identifies the beacon at once; the lock carries the System
// Identity Code.
func TestControlChannelLocksOnAloha(t *testing.T) {
	tx := newBeaconTx(1, 0x1234)
	tx.aloha()
	tx.other()
	_, evs := runCC(t, tx.dibits)
	locks := kinds(evs, events.KindCCLocked)
	if len(locks) != 1 {
		t.Fatalf("%d locks, want 1", len(locks))
	}
	if ls := locks[0].Payload.(LockState); ls.SystemID != 0x1234 || ls.FrequencyHz != 446_100_000 {
		t.Fatalf("lock %+v", ls)
	}
}

// Without an Aloha, the same SYScast1 system code twice in a row locks;
// once does not.
func TestControlChannelLocksOnRepeatedSYScast1(t *testing.T) {
	tx := newBeaconTx(2, 0x0777)
	tx.other()
	tx.noise(1000)
	if _, evs := runCC(t, tx.dibits); len(kinds(evs, events.KindCCLocked)) != 0 {
		t.Fatal("one SYScast1 locked the channel")
	}
	tx = newBeaconTx(2, 0x0777)
	tx.other()
	tx.other()
	_, evs := runCC(t, tx.dibits)
	locks := kinds(evs, events.KindCCLocked)
	if len(locks) != 1 || locks[0].Payload.(LockState).SystemID != 0x0777 {
		t.Fatalf("locks %v", locks)
	}
}

// A Goto Channel for a voice call becomes one grant: the called party, the
// calling party from the SYScasts after it, the downlink frequency and the
// emergency flag. The beacon's repeats of it are not new calls; a
// different call is.
func TestControlChannelPublishesGotoChannelGrant(t *testing.T) {
	band, sep, rx, _ := FrequencyFields(446_006_250)
	_, _, txf, _ := FrequencyFields(446_006_250)
	g := GotoChannel{Called: 0x02C00A, RxField: rx, TxField: txf, Band: band, Sep5k: sep, Mode: AirModeVoice, Emergency: true}
	tx := newBeaconTx(3, 0x0100)
	tx.aloha()
	for i := 0; i < 3; i++ {
		tx.gotoChannel(g, 0x1B91FD)
	}
	tx.aloha()
	g2 := g
	g2.Called, g2.Emergency = 0x68BF08, false
	tx.gotoChannel(g2, 0x02C00B)
	_, evs := runCC(t, tx.dibits)
	grants := kinds(evs, events.KindGrant)
	if len(grants) != 2 {
		t.Fatalf("%d grants, want 2 (the first call once, then the second)", len(grants))
	}
	want := []trunking.Grant{
		{System: "DPMR", Protocol: "dpmr", GroupID: 0x02C00A, SourceID: 0x1B91FD, FrequencyHz: 446_006_250, ChannelNum: rx, Emergency: true},
		{System: "DPMR", Protocol: "dpmr", GroupID: 0x68BF08, SourceID: 0x02C00B, FrequencyHz: 446_006_250, ChannelNum: rx},
	}
	for i, ev := range grants {
		got := ev.Payload.(trunking.Grant)
		w := want[i]
		if got.System != w.System || got.Protocol != w.Protocol || got.GroupID != w.GroupID ||
			got.SourceID != w.SourceID || got.FrequencyHz != w.FrequencyHz ||
			got.ChannelNum != w.ChannelNum || got.Emergency != w.Emergency {
			t.Errorf("grant %d = %+v\nwant      %+v", i, got, want[i])
		}
	}
}

// A Goto Channel for a data call, or one in a band the standard leaves to
// the operator, is not followed.
func TestControlChannelSkipsDataAndCustomBandGotos(t *testing.T) {
	tx := newBeaconTx(4, 0x0100)
	tx.aloha()
	tx.gotoChannel(GotoChannel{Called: 5, RxField: 100, Band: 5, Mode: AirModeData3}, 6)
	tx.gotoChannel(GotoChannel{Called: 7, RxField: 100, Band: BandCustom, Mode: AirModeVoice}, 8)
	_, evs := runCC(t, tx.dibits)
	if g := kinds(evs, events.KindGrant); len(g) != 0 {
		t.Fatalf("%d grants: %+v", len(g), g[0].Payload)
	}
}

// A dPMR traffic channel (a voice call, which opens with FS1 too) and
// noise never lock a control channel or grant.
func TestControlChannelIgnoresTrafficChannelAndNoise(t *testing.T) {
	tx := newAirTx(t, 5)
	tx.header()
	for i := 0; i < 10; i++ {
		tx.superframe()
	}
	tx.end()
	r := rand.New(rand.NewSource(6))
	noise := make([]uint8, 2400*60)
	for i := range noise {
		noise[i] = uint8(r.Intn(4))
	}
	for name, d := range map[string][]uint8{"traffic channel": tx.dibits, "noise": noise} {
		_, evs := runCC(t, d)
		if len(kinds(evs, events.KindCCLocked)) != 0 || len(kinds(evs, events.KindGrant)) != 0 {
			t.Errorf("%s: %d locks, %d grants", name, len(kinds(evs, events.KindCCLocked)), len(kinds(evs, events.KindGrant)))
		}
	}
}

// MarkLost publishes cc.lost once.
func TestControlChannelMarkLost(t *testing.T) {
	tx := newBeaconTx(7, 9)
	tx.aloha()
	cc, _ := runCC(t, tx.dibits)
	if !cc.Locked() {
		t.Fatal("not locked")
	}
	bus := cc.bus
	sub := bus.Subscribe()
	defer sub.Close()
	cc.MarkLost()
	cc.MarkLost()
	n := 0
	for len(sub.C) > 0 {
		if (<-sub.C).Kind == events.KindCCLost {
			n++
		}
	}
	if n != 1 || cc.Locked() {
		t.Fatalf("%d cc.lost, locked %v", n, cc.Locked())
	}
}
