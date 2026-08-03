package voice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRecorderWritesCallJSON pins the trunk-recorder .json sidecar: a finished
// recording gets a companion <basename>.json carrying the call's talkgroup,
// source, frequency and timing, plus a srcList that grows on talker changes.
// Fail-first: without the sidecar writer the file simply does not exist.
func TestRecorderWritesCallJSON(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus:           bus,
		OutDir:        dir,
		SampleRate:    8000,
		WriteCallJSON: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Ostankino", Protocol: "tetra", GroupID: 1020539, SourceID: 132622,
			FrequencyHz: 451_925_000, Timeslot: 2,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1020539, AlphaTag: "OPS-1", Priority: 4, Record: true},
		DeviceSerial: "V1",
		StartedAt:    time.Date(2026, 8, 1, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "V1", true)

	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	// A talker change mid-call — must add a second srcList entry.
	bus.Publish(events.Event{Kind: events.KindCallSourceUpdate, Payload: trunking.CallSourceUpdate{
		DeviceSerial: "V1", SourceID: 145000,
	}})
	// Let the source update land before the next write / end.
	time.Sleep(50 * time.Millisecond)
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "V1", StartedAt: cs.StartedAt,
		EndedAt: cs.StartedAt.Add(3 * time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "V1", false)

	// Locate the WAV and its .json sibling.
	tgDir := filepath.Join(dir, "Ostankino", "OPS-1")
	entries, err := os.ReadDir(tgDir)
	if err != nil {
		t.Fatalf("read tg dir: %v", err)
	}
	var wav string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wav") {
			wav = filepath.Join(tgDir, e.Name())
		}
	}
	if wav == "" {
		t.Fatalf("no wav recorded in %s (entries: %v)", tgDir, entries)
	}
	jsonPath := strings.TrimSuffix(wav, ".wav") + ".json"
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("expected .json sidecar next to %s: %v", wav, err)
	}

	var m callMeta
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("sidecar is not valid JSON: %v", err)
	}
	if m.Talkgroup != 1020539 {
		t.Errorf("talkgroup = %d, want 1020539", m.Talkgroup)
	}
	if m.TalkgroupTag != "OPS-1" {
		t.Errorf("talkgroup_tag = %q, want OPS-1", m.TalkgroupTag)
	}
	if m.Freq != 451_925_000 {
		t.Errorf("freq = %d, want 451925000", m.Freq)
	}
	if m.ShortName != "Ostankino" {
		t.Errorf("short_name = %q, want Ostankino", m.ShortName)
	}
	if len(m.SrcList) != 2 {
		t.Fatalf("srcList = %d entries, want 2 (initial + talker change): %+v", len(m.SrcList), m.SrcList)
	}
	if m.SrcList[0].Src != 132622 || m.SrcList[1].Src != 145000 {
		t.Errorf("srcList sources = %d,%d, want 132622,145000", m.SrcList[0].Src, m.SrcList[1].Src)
	}
	if len(m.FreqList) != 1 || m.FreqList[0].Freq != 451_925_000 {
		t.Errorf("freqList = %+v, want one entry at 451925000", m.FreqList)
	}
	// The schema keys trunk-recorder parsers expect must all be present.
	for _, key := range []string{
		"call_num", "freq", "signal", "noise", "start_time", "stop_time",
		"start_time_ms", "stop_time_ms", "call_length", "talkgroup",
		"talkgroup_tag", "color_code", "audio_type", "short_name",
		"freqList", "srcList",
	} {
		if !strings.Contains(string(b), "\""+key+"\"") {
			t.Errorf("sidecar missing required key %q", key)
		}
	}
}

// TestRecorderCallJSONSrcListSpansTransmissions pins that the .json srcList is
// CALL-complete: in transmission grouping a talker change rolls the WAV to a new
// file (KindCallSegment), yet each transmission's sidecar must still list every
// talker of the whole call — so the operator's "second srcID is missed" cannot
// happen. Fail-first: before the call-level accumulator, the second file's srcs
// were reseeded from the current talker on the roll, so its sidecar carried only
// the second srcID and the two-entry assertion goes red.
func TestRecorderCallJSONSrcListSpansTransmissions(t *testing.T) {
	bus := events.NewBus(8)
	dir := t.TempDir()
	r, err := NewRecorder(RecorderOptions{
		Bus: bus, OutDir: dir, SampleRate: 8000, WriteCallJSON: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	const talkerA, talkerB = uint32(132622), uint32(145000)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "Ostankino", Protocol: "tetra", GroupID: 1020539, SourceID: talkerA,
			FrequencyHz: 451_925_000, Timeslot: 2,
		},
		Talkgroup:    &trunking.TalkGroup{ID: 1020539, AlphaTag: "OPS-1", Record: true},
		DeviceSerial: "V1",
		StartedAt:    time.Date(2026, 8, 1, 1, 2, 3, 0, time.UTC),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, "V1", true)

	// First over (talker A), then a per-transmission roll — the engine emits the
	// segment BEFORE the source update, exactly as handleCallTalker does.
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{Kind: events.KindCallSegment, Payload: trunking.CallSegment{
		DeviceSerial: "V1", At: cs.StartedAt.Add(1 * time.Second),
	}})
	time.Sleep(50 * time.Millisecond)
	// Talker change to B, then the second over's audio → second file.
	bus.Publish(events.Event{Kind: events.KindCallSourceUpdate, Payload: trunking.CallSourceUpdate{
		DeviceSerial: "V1", SourceID: talkerB,
	}})
	time.Sleep(50 * time.Millisecond)
	if err := r.WritePCM("V1", make([]int16, 800)); err != nil {
		t.Fatal(err)
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: cs.Grant, DeviceSerial: "V1", StartedAt: cs.StartedAt,
		EndedAt: cs.StartedAt.Add(3 * time.Second), Reason: trunking.EndReasonNormal,
	}})
	waitSession(t, r, "V1", false)

	// The second transmission's file is named from talker B; its sidecar must
	// carry BOTH talkers in order.
	tgDir := filepath.Join(dir, "Ostankino", "OPS-1")
	entries, err := os.ReadDir(tgDir)
	if err != nil {
		t.Fatalf("read tg dir: %v", err)
	}
	var secondJSON string
	for _, e := range entries {
		if strings.Contains(e.Name(), "src145000") && strings.HasSuffix(e.Name(), ".json") {
			secondJSON = filepath.Join(tgDir, e.Name())
		}
	}
	if secondJSON == "" {
		t.Fatalf("no second-transmission .json (src145000) found in %v", entries)
	}
	b, err := os.ReadFile(secondJSON)
	if err != nil {
		t.Fatal(err)
	}
	var m callMeta
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("second sidecar is not valid JSON: %v", err)
	}
	if len(m.SrcList) != 2 || m.SrcList[0].Src != talkerA || m.SrcList[1].Src != talkerB {
		t.Fatalf("second-transmission srcList = %+v, want the call-complete [%d, %d]", m.SrcList, talkerA, talkerB)
	}
	// pos is call-relative and non-negative even for the earlier talker.
	for i, e := range m.SrcList {
		if e.Pos < 0 {
			t.Errorf("srcList[%d].pos = %v, want >= 0 (call-relative)", i, e.Pos)
		}
	}
}
