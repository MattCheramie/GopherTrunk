package trunking

import "testing"

func TestEffectivePriority(t *testing.T) {
	cases := []struct {
		name string
		g    Grant
		tg   *TalkGroup
		want int
	}{
		{"emergency overrides", Grant{Emergency: true}, &TalkGroup{Priority: 5}, 0},
		{"set priority", Grant{}, &TalkGroup{Priority: 3}, 3},
		{"nil talkgroup", Grant{}, nil, priorityUnset},
		{"zero talkgroup priority", Grant{}, &TalkGroup{Priority: 0}, priorityUnset},
		{"emergency with nil tg", Grant{Emergency: true}, nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectivePriority(tc.g, tc.tg); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCanPreempt(t *testing.T) {
	high := &TalkGroup{Priority: 1}
	mid := &TalkGroup{Priority: 5}
	low := &TalkGroup{Priority: 9}
	lockedOut := &TalkGroup{Priority: 1, Lockout: true}

	cases := []struct {
		name     string
		active   *TalkGroup
		incoming *TalkGroup
		emer     bool
		want     bool
	}{
		{"higher preempts lower", low, high, false, true},
		{"lower does NOT preempt higher", high, low, false, false},
		{"equal does NOT preempt", mid, mid, false, false},
		{"emergency preempts everything", mid, mid, true, true},
		{"locked out cannot preempt", high, lockedOut, false, false},
		{"unset incoming does not preempt set", mid, nil, false, false},
		{"set incoming preempts unset", nil, mid, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activeG := Grant{}
			incomingG := Grant{Emergency: tc.emer}
			got := CanPreempt(activeG, tc.active, incomingG, tc.incoming)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
