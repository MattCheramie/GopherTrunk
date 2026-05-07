package tui

import "testing"

func TestRingBuf_PushOverwritesOldest(t *testing.T) {
	r := NewRingBuf[int](3)
	for i := 1; i <= 5; i++ {
		r.Push(i)
	}
	if r.Len() != 3 {
		t.Fatalf("Len = %d", r.Len())
	}
	got := r.Snapshot()
	want := []int{3, 4, 5}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("Snapshot[%d] = %d, want %d", i, got[i], v)
		}
	}
	latest := r.Latest(2)
	if latest[0] != 5 || latest[1] != 4 {
		t.Errorf("Latest(2) = %v, want [5 4]", latest)
	}
}

func TestRingBuf_ClearResets(t *testing.T) {
	r := NewRingBuf[int](2)
	r.Push(1)
	r.Push(2)
	r.Clear()
	if r.Len() != 0 {
		t.Errorf("Len after Clear = %d", r.Len())
	}
	r.Push(7)
	if r.Latest(1)[0] != 7 {
		t.Errorf("Latest after Clear+Push = %v", r.Latest(1))
	}
}
