package state

import "testing"

func TestPanelKindStringIsStableForEachValue(t *testing.T) {
	cases := []struct {
		kind PanelKind
		want string
	}{
		{PanelDashboard, "Dashboard"},
		{PanelSystems, "Systems"},
		{PanelTalkgroups, "Talkgroups"},
		{PanelActive, "Active"},
		{PanelHistory, "History"},
		{PanelEvents, "Events"},
		{PanelTones, "Tones"},
		{PanelMetrics, "Metrics"},
		{PanelDevices, "Devices"},
		{PanelScanner, "Scanner"},
		{PanelSettings, "Settings"},
		{PanelImport, "Import"},
	}
	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("PanelKind(%d).String() = %q, want %q", int(tc.kind), got, tc.want)
		}
	}
}

func TestPanelKindStringFallbackForUnknown(t *testing.T) {
	if got := PanelKind(99).String(); got != "?" {
		t.Errorf("unknown PanelKind String() = %q, want %q", got, "?")
	}
	if got := PanelKind(-1).String(); got != "?" {
		t.Errorf("negative PanelKind String() = %q, want %q", got, "?")
	}
	if got := PanelCount.String(); got != "?" {
		t.Errorf("PanelCount sentinel String() = %q, want %q", got, "?")
	}
}

func TestPanelCountMatchesEnumeratedKinds(t *testing.T) {
	// Catches a contributor adding a new panel kind without
	// extending String() or shifting PanelCount.
	if PanelCount != PanelImport+1 {
		t.Errorf("PanelCount = %d, want %d (PanelImport+1)", PanelCount, PanelImport+1)
	}

	// Every kind below PanelCount must have a non-fallback string.
	for k := PanelDashboard; k < PanelCount; k++ {
		if got := k.String(); got == "?" || got == "" {
			t.Errorf("PanelKind(%d).String() = %q — extend the stringer when adding panels", int(k), got)
		}
	}
}

func TestPanelKindStringsAreUnique(t *testing.T) {
	seen := map[string]PanelKind{}
	for k := PanelDashboard; k < PanelCount; k++ {
		s := k.String()
		if prev, ok := seen[s]; ok {
			t.Errorf("PanelKind(%d) and PanelKind(%d) both stringify to %q", int(prev), int(k), s)
		}
		seen[s] = k
	}
}

func TestWriteKindZeroValueIsUnknown(t *testing.T) {
	// Locks in that a default-constructed WriteRequest has a
	// discriminator the dispatcher recognises as "unset".
	var w WriteRequest
	if w.Kind != WriteKindUnknown {
		t.Errorf("zero WriteRequest.Kind = %d, want WriteKindUnknown (%d)", w.Kind, WriteKindUnknown)
	}
	if WriteKindUnknown != 0 {
		t.Errorf("WriteKindUnknown = %d, want 0", WriteKindUnknown)
	}
}

func TestWriteKindEnumIsDense(t *testing.T) {
	// Reject duplicate iota values — every WriteKind constant must
	// be distinct so the dispatcher's type switch is exhaustive.
	kinds := []WriteKind{
		WriteKindUnknown,
		WriteKindEndCall,
		WriteKindUpdateTalkgroup,
		WriteKindSweepRetention,
		WriteKindResetTone,
		WriteKindScannerMode,
		WriteKindScannerHuntHold,
		WriteKindScannerHuntResume,
		WriteKindScannerHuntRetune,
		WriteKindScannerConvHold,
		WriteKindScannerConvResume,
		WriteKindScannerConvDwell,
		WriteKindScannerConvLockout,
		WriteKindScannerConvUnlockout,
		WriteKindAudio,
		WriteKindScannerManualTune,
		WriteKindSettings,
	}
	seen := map[WriteKind]bool{}
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("WriteKind value %d appears twice in the constant list", int(k))
		}
		seen[k] = true
	}
}
