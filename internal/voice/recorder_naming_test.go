package voice

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// recForNaming builds a file-writing recorder with the given templates (no Run
// loop — basenameFor/directoryFor are pure).
func recForNaming(t *testing.T, filenameTmpl, pathTmpl string, loc *time.Location) *Recorder {
	t.Helper()
	bus := events.NewBus(1)
	t.Cleanup(func() { bus.Close() })
	r, err := NewRecorder(RecorderOptions{
		Bus:              bus,
		OutDir:           "/rec",
		SampleRate:       8000,
		DisplayLoc:       loc,
		FilenameTemplate: filenameTmpl,
		PathTemplate:     pathTmpl,
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRecorderNamingDefaults(t *testing.T) {
	r := recForNaming(t, "", "", nil)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Metro", Protocol: "tetra", GroupID: 1020545, SourceID: 1005738,
			FrequencyHz: 467_912_500, Timeslot: 2, CallID: 42,
		},
		Talkgroup: &trunking.TalkGroup{ID: 1020545, AlphaTag: "OPS-1"},
		StartedAt: time.Date(2026, 8, 10, 23, 0, 41, 0, time.UTC),
	}
	// Default basename: date_time_tg — no tz offset, no src, no ts.
	if got, want := r.basenameFor(cs), "20260810_230041_1020545"; got != want {
		t.Errorf("default basename = %q, want %q", got, want)
	}
	// Default dir: <system>/<alpha>.
	if got, want := r.directoryFor(cs), filepath.Join("/rec", "Metro", "OPS-1"); got != want {
		t.Errorf("default dir = %q, want %q", got, want)
	}
	// Default dir for an individual call: <system>/individual/<dest>.
	ics := cs
	ics.Grant.Individual = true
	ics.Talkgroup = nil
	if got, want := r.directoryFor(ics), filepath.Join("/rec", "Metro", "individual", "1020545"); got != want {
		t.Errorf("individual dir = %q, want %q", got, want)
	}
}

func TestRecorderNamingCustomTemplates(t *testing.T) {
	// A P25-friendly filename + a trunk-recorder-style date tree.
	r := recForNaming(t, "{date}_{time}_{tg}_{freq}", "{system}/{year}/{month}/{day}", nil)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Metro", Protocol: "p25", GroupID: 50, SourceID: 999,
			FrequencyHz: 851_000_000, CallID: 7,
		},
		StartedAt: time.Date(2026, 8, 10, 23, 0, 41, 0, time.UTC),
	}
	if got, want := r.basenameFor(cs), "20260810_230041_50_851000000"; got != want {
		t.Errorf("templated basename = %q, want %q", got, want)
	}
	if got, want := r.directoryFor(cs), filepath.Join("/rec", "Metro", "2026", "08", "10"); got != want {
		t.Errorf("date-tree dir = %q, want %q", got, want)
	}
	// The date tree applies to individual calls too (template author's choice).
	ics := cs
	ics.Grant.Individual = true
	if got, want := r.directoryFor(ics), filepath.Join("/rec", "Metro", "2026", "08", "10"); got != want {
		t.Errorf("date-tree individual dir = %q, want %q", got, want)
	}
}

func TestRecorderNamingRespectsDisplayTimezone(t *testing.T) {
	r := recForNaming(t, "", "", time.FixedZone("UTC+3", 3*3600))
	cs := trunking.CallStart{
		Grant:     trunking.Grant{System: "S", GroupID: 7},
		StartedAt: time.Date(2026, 8, 10, 23, 30, 0, 0, time.UTC), // → 02:30 next day local
	}
	if got, want := r.basenameFor(cs), "20260811_023000_7"; got != want {
		t.Errorf("tz basename = %q, want %q", got, want)
	}
}
