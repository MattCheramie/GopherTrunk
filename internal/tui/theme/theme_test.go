package theme

import (
	"sync"
	"testing"
)

func TestDarkPalette_NonEmpty(t *testing.T) {
	p := DarkPalette()
	if p.Accent == "" || p.Border == "" || p.Danger == "" {
		t.Fatalf("dark palette has empty roles: %+v", p)
	}
}

func TestMonochromePalette_AllEmpty(t *testing.T) {
	p := MonochromePalette()
	// Every colour role must collapse to default so lipgloss emits
	// no ANSI on --no-color.
	if p.Accent != "" || p.Border != "" || p.Danger != "" || p.Fg != "" {
		t.Fatalf("monochrome palette has non-empty roles: %+v", p)
	}
}

func TestSet_GoroutineSafe(t *testing.T) {
	// Race-detector smoke: hammer Set and Theme from N goroutines.
	// The atomic.Pointer swap should keep this clean under -race.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if i%2 == 0 {
					Set(DarkPalette())
				} else {
					Set(MonochromePalette())
				}
				_ = Theme()
			}
		}(i)
	}
	wg.Wait()
	// Restore dark for downstream tests.
	Set(DarkPalette())
}

func TestFrame_FocusedHasDifferentBorder(t *testing.T) {
	p := DarkPalette()
	// Compare style attributes directly — lipgloss strips ANSI when
	// the test process has no TTY, so Render output is identical
	// regardless of colour, but the underlying style fields differ.
	if p.Frame(false).GetBorderTopForeground() == p.Frame(true).GetBorderTopForeground() {
		t.Fatal("Frame(true) and Frame(false) share the same border colour")
	}
}
