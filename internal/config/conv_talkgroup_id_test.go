package config

import (
	"strings"
	"testing"
)

// TestConvChannelTalkgroupIDCollisionRejected pins the #1105 validation rule:
// two conventional channels resolving to the same effective talkgroup ID —
// whether two identical explicit talkgroup_id values, or an explicit value
// colliding with another channel's positional 0x80000000|index default — are
// a config error, because the engine folds synthetic calls keyed on
// (system, GroupID). Fails against a config package without the check (the
// collision validated clean and the two channels silently shared one call
// identity).
func TestConvChannelTalkgroupIDCollisionRejected(t *testing.T) {
	errText := func(c Config) string {
		var b strings.Builder
		for _, err := range c.validateScanner() {
			b.WriteString(err.Error())
			b.WriteString("\n")
		}
		return b.String()
	}

	// Two identical explicit IDs collide.
	c := Config{}
	c.Scanner.Conventional = []ConvChannelConfig{
		{Label: "A", FrequencyHz: 100_000_000, TalkgroupID: 42},
		{Label: "B", FrequencyHz: 200_000_000, TalkgroupID: 42},
	}
	if got := errText(c); !strings.Contains(got, "collides") {
		t.Errorf("identical explicit IDs: errors = %q, want a collision error", got)
	}

	// An explicit ID equal to another channel's positional default collides
	// too: channel index 1's default is 0x80000000|1.
	c = Config{}
	c.Scanner.Conventional = []ConvChannelConfig{
		{Label: "A", FrequencyHz: 100_000_000, TalkgroupID: 0x80000001},
		{Label: "B", FrequencyHz: 200_000_000}, // positional default 0x80000001
	}
	if got := errText(c); !strings.Contains(got, "collides") {
		t.Errorf("explicit vs positional default: errors = %q, want a collision error", got)
	}

	// Distinct effective IDs validate clean.
	c = Config{}
	c.Scanner.Conventional = []ConvChannelConfig{
		{Label: "A", FrequencyHz: 100_000_000, TalkgroupID: 42},
		{Label: "B", FrequencyHz: 200_000_000},
		{Label: "C", FrequencyHz: 300_000_000, TalkgroupID: 43},
	}
	if got := errText(c); got != "" {
		t.Errorf("distinct IDs: unexpected errors = %q", got)
	}
}
