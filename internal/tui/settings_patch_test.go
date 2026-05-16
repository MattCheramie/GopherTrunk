package tui

import (
	"strings"
	"testing"
)

func TestBuildSettingsPatch_StringField(t *testing.T) {
	p, err := buildSettingsPatch("log.level", "debug")
	if err != nil {
		t.Fatal(err)
	}
	if p.LogLevel == nil || *p.LogLevel != "debug" {
		t.Errorf("LogLevel = %v want debug", p.LogLevel)
	}
}

func TestBuildSettingsPatch_FloatField(t *testing.T) {
	p, err := buildSettingsPatch("audio.volume", "0.42")
	if err != nil {
		t.Fatal(err)
	}
	if p.AudioVolume == nil || *p.AudioVolume != 0.42 {
		t.Errorf("AudioVolume = %v want 0.42", p.AudioVolume)
	}
}

func TestBuildSettingsPatch_VolumeOutOfRange(t *testing.T) {
	if _, err := buildSettingsPatch("audio.volume", "2.5"); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestBuildSettingsPatch_BoolField(t *testing.T) {
	p, err := buildSettingsPatch("recordings.write_raw", "true")
	if err != nil {
		t.Fatal(err)
	}
	if p.RecordingsWriteRaw == nil || !*p.RecordingsWriteRaw {
		t.Errorf("RecordingsWriteRaw = %v want true", p.RecordingsWriteRaw)
	}

	p2, err := buildSettingsPatch("recordings.write_raw", "off")
	if err != nil {
		t.Fatal(err)
	}
	if p2.RecordingsWriteRaw == nil || *p2.RecordingsWriteRaw {
		t.Errorf("off should map to false, got %v", p2.RecordingsWriteRaw)
	}
}

func TestBuildSettingsPatch_BoolBadInput(t *testing.T) {
	_, err := buildSettingsPatch("audio.muted", "maybe")
	if err == nil {
		t.Fatal("expected error for non-bool input")
	}
	if !strings.Contains(err.Error(), "true/false") {
		t.Errorf("error should mention true/false, got %q", err.Error())
	}
}

func TestBuildSettingsPatch_IntField(t *testing.T) {
	p, err := buildSettingsPatch("retention.call_log_days", "30")
	if err != nil {
		t.Fatal(err)
	}
	if p.RetentionCallLogDays == nil || *p.RetentionCallLogDays != 30 {
		t.Errorf("RetentionCallLogDays = %v want 30", p.RetentionCallLogDays)
	}
}

func TestBuildSettingsPatch_UintField(t *testing.T) {
	p, err := buildSettingsPatch("sdr.sample_rate", "2400000")
	if err != nil {
		t.Fatal(err)
	}
	if p.SDRSampleRate == nil || *p.SDRSampleRate != 2_400_000 {
		t.Errorf("SDRSampleRate = %v want 2400000", p.SDRSampleRate)
	}
}

func TestBuildSettingsPatch_UnknownField(t *testing.T) {
	_, err := buildSettingsPatch("nonexistent.knob", "x")
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}
