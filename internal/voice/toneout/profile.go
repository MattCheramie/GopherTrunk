package toneout

import (
	"errors"
	"fmt"
	"time"
)

// Tone is one element of a paging-tone sequence.
type Tone struct {
	// FrequencyHz is the target tone frequency in Hz.
	FrequencyHz float64
	// MinDuration is how long the tone must persist before it counts.
	// Typical Quick Call II A-tones are ≥ 250 ms.
	MinDuration time.Duration
	// MaxDuration caps the on-time. Tones held longer than this reset
	// the match. 0 disables the upper bound.
	MaxDuration time.Duration
}

// Profile describes one tone-out alarm: a list of tones to match in
// sequence, plus optional filters and policy.
//
// For two-tone sequential paging (the most common US fire/EMS usage),
// supply two Tone entries: an A-tone and a B-tone. For single-tone
// supervision pages, supply one. DTMF and concurrent multi-tone
// patterns are out of scope for v1 — the matcher walks tones strictly
// in order.
type Profile struct {
	// Name is a stable identifier for logs + the events.KindToneAlert
	// payload. Required, must be unique per detector.
	Name string
	// AlphaTag is a human-readable label for UI / webhook display.
	AlphaTag string
	// Tones is the ordered tone list to match. At least one tone.
	Tones []Tone
	// ToleranceHz allows the matched tone to drift this far from the
	// configured FrequencyHz. Default 15 Hz (sufficient for the
	// 10 Hz Goertzel resolution at the default block size and
	// typical agency frequency stability).
	ToleranceHz float64
	// MagnitudeThreshold is the squared-magnitude floor for a tone
	// to count as detected. Range roughly 0..1 for a unit-amplitude
	// sine; default 0.05 (i.e. ~10 dB above quiet noise).
	MagnitudeThreshold float64
	// MaxGap is the longest silence allowed between tones in a
	// sequence. Quick Call II inter-tone gaps are typically < 50 ms;
	// default 200 ms tolerates real-world drift.
	MaxGap time.Duration
	// Cooldown suppresses re-fires of the same profile within this
	// window. Default 30 s.
	Cooldown time.Duration
	// System optionally restricts the profile to the named trunked
	// system. Empty matches all.
	System string
	// GroupID optionally restricts the profile to a single talkgroup.
	// 0 matches all.
	GroupID uint32
}

// Validate fills in defaults and returns a non-nil error if the
// profile is unusable.
func (p *Profile) Validate() error {
	if p.Name == "" {
		return errors.New("toneout: profile name is required")
	}
	if len(p.Tones) == 0 {
		return errors.New("toneout: profile needs at least one tone")
	}
	for i, t := range p.Tones {
		if t.FrequencyHz <= 0 {
			return fmt.Errorf("toneout: profile %q tone[%d]: frequency_hz must be > 0", p.Name, i)
		}
		if t.MinDuration <= 0 {
			return fmt.Errorf("toneout: profile %q tone[%d]: min_duration must be > 0", p.Name, i)
		}
		if t.MaxDuration > 0 && t.MaxDuration < t.MinDuration {
			return fmt.Errorf("toneout: profile %q tone[%d]: max_duration < min_duration", p.Name, i)
		}
	}
	if p.ToleranceHz == 0 {
		p.ToleranceHz = 15
	}
	if p.MagnitudeThreshold == 0 {
		p.MagnitudeThreshold = 0.05
	}
	if p.MaxGap == 0 {
		p.MaxGap = 200 * time.Millisecond
	}
	if p.Cooldown == 0 {
		p.Cooldown = 30 * time.Second
	}
	return nil
}

// Alert is the payload of an events.KindToneAlert event.
type Alert struct {
	Profile      string    `json:"profile"`
	AlphaTag     string    `json:"alpha_tag,omitempty"`
	System       string    `json:"system,omitempty"`
	DeviceSerial string    `json:"device_serial"`
	MatchedAt    time.Time `json:"matched_at"`
	// FrequenciesHz is the list of tone frequencies in match order
	// — the actual matched bins, useful for diagnostics if a profile
	// matched a slightly off-frequency reception.
	FrequenciesHz []float64 `json:"frequencies_hz"`
}
