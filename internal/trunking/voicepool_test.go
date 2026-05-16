package trunking

import (
	"sync"
	"testing"
	"time"
)

type fakeVoiceTuner struct {
	mu     sync.Mutex
	freqs  []uint32
	failOn uint32
}

func (f *fakeVoiceTuner) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.freqs = append(f.freqs, hz)
	if f.failOn != 0 && hz == f.failOn {
		return tuneError("forced failure")
	}
	return nil
}

// tuned returns a copy of the recorded center-frequency calls. Tests must
// use this rather than reading the field directly to stay race-free.
func (f *fakeVoiceTuner) tuned() []uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uint32(nil), f.freqs...)
}

type tuneError string

func (e tuneError) Error() string { return string(e) }

func mkPool(n int) (*VoicePool, []*fakeVoiceTuner) {
	tuners := make([]*fakeVoiceTuner, n)
	devs := make([]*VoiceDevice, n)
	for i := 0; i < n; i++ {
		t := &fakeVoiceTuner{}
		tuners[i] = t
		devs[i] = &VoiceDevice{Tuner: t, Serial: serial(i)}
	}
	return NewVoicePool(devs), tuners
}

func serial(i int) string {
	return string(rune('A'+i)) + "-voice"
}

func TestPoolFindFreeAndBind(t *testing.T) {
	p, _ := mkPool(2)
	now := time.Now()
	if free := p.FindFree(); free == nil || free.Serial != "A-voice" {
		t.Fatalf("FindFree = %+v", free)
	}
	d := p.FindFree()
	ac, err := p.Bind(d, Grant{FrequencyHz: 851_000_000}, nil, now)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if ac.Device != d {
		t.Errorf("ac.Device != d")
	}
	if free := p.FindFree(); free == nil || free.Serial != "B-voice" {
		t.Fatalf("after binding A, FindFree should return B; got %+v", free)
	}
}

func TestPoolBindFailsWhenBusy(t *testing.T) {
	p, _ := mkPool(1)
	d := p.FindFree()
	if _, err := p.Bind(d, Grant{FrequencyHz: 1}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Bind(d, Grant{FrequencyHz: 2}, nil, time.Now()); err == nil {
		t.Error("expected busy error on second Bind")
	}
}

func TestPoolBindRetunesDevice(t *testing.T) {
	p, tuners := mkPool(1)
	d := p.FindFree()
	const freq = 853_000_000
	if _, err := p.Bind(d, Grant{FrequencyHz: freq}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := tuners[0].tuned(); len(got) != 1 || got[0] != freq {
		t.Errorf("tune sequence = %v, want [%d]", got, freq)
	}
}

func TestPoolReleaseReturnsActiveCall(t *testing.T) {
	p, _ := mkPool(1)
	d := p.FindFree()
	p.Bind(d, Grant{FrequencyHz: 1}, nil, time.Now())
	if ac := p.Release(d.Serial); ac == nil {
		t.Error("Release should return the freed ActiveCall")
	}
	if ac := p.Release(d.Serial); ac != nil {
		t.Error("second Release should return nil")
	}
	if free := p.FindFree(); free == nil {
		t.Error("after Release, device should be free again")
	}
}

func TestLowestPriorityActiveSelectsLeastImportant(t *testing.T) {
	p, _ := mkPool(3)
	for i, prio := range []int{2, 7, 5} {
		d := p.Devices()[i]
		_, _ = p.Bind(d, Grant{FrequencyHz: uint32(100 + i)}, &TalkGroup{Priority: prio}, time.Now())
	}
	got := p.LowestPriorityActive()
	if got == nil || got.Talkgroup.Priority != 7 {
		t.Errorf("LowestPriorityActive = %+v, want talkgroup priority=7", got)
	}
}

func TestTouchUpdatesLastHeard(t *testing.T) {
	p, _ := mkPool(1)
	d := p.FindFree()
	start := time.Unix(1000, 0)
	p.Bind(d, Grant{FrequencyHz: 1}, nil, start)
	later := start.Add(5 * time.Second)
	p.Touch(d.Serial, later)
	for _, ac := range p.Active() {
		if !ac.LastHeardAt.Equal(later) {
			t.Errorf("LastHeardAt = %v, want %v", ac.LastHeardAt, later)
		}
	}
}

func TestBindFailsWhenTuneFails(t *testing.T) {
	p, tuners := mkPool(1)
	tuners[0].failOn = 999_999_999
	d := p.FindFree()
	_, err := p.Bind(d, Grant{FrequencyHz: 999_999_999}, nil, time.Now())
	if err == nil {
		t.Fatal("expected tune failure to propagate")
	}
	if free := p.FindFree(); free == nil {
		t.Error("device should remain free after tune failure")
	}
}
