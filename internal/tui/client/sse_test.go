package client

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseSSE_BasicEvents(t *testing.T) {
	body := strings.Join([]string{
		`event: cc.locked`,
		`data: {"kind":"cc.locked","timestamp":"2024-01-02T03:04:05Z","payload":{"FrequencyHz":851012500}}`,
		``,
		`event: grant`,
		`data: {"kind":"grant","timestamp":"2024-01-02T03:04:06Z","payload":{"system":"Demo","group_id":42,"frequency_hz":852012500}}`,
		``,
	}, "\n")
	ch := make(chan Event, 8)
	if err := parseSSE(strings.NewReader(body), ch); err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	close(ch)
	var got []Event
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 events, got %d", len(got))
	}
	if got[0].Kind != "cc.locked" {
		t.Errorf("ev0 kind = %q", got[0].Kind)
	}
	want0 := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if !got[0].Time.Equal(want0) {
		t.Errorf("ev0 time = %v, want %v", got[0].Time, want0)
	}
	if got[1].Kind != "grant" {
		t.Errorf("ev1 kind = %q", got[1].Kind)
	}
}

func TestParseSSE_IgnoresComments(t *testing.T) {
	body := strings.Join([]string{
		`: keep-alive comment`,
		`event: ping`,
		`data: {"kind":"ping","timestamp":"2024-01-02T03:04:05Z","payload":null}`,
		``,
	}, "\n")
	ch := make(chan Event, 4)
	if err := parseSSE(strings.NewReader(body), ch); err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	close(ch)
	count := 0
	for range ch {
		count++
	}
	if count != 1 {
		t.Errorf("comment not ignored: got %d events", count)
	}
}

func TestParseSSE_FlushesTrailingEventWithoutBlankLine(t *testing.T) {
	// Server may close mid-event; we should still flush what we have.
	body := strings.Join([]string{
		`event: tone.alert`,
		`data: {"kind":"tone.alert","timestamp":"2024-01-02T03:04:05Z","payload":{"profile":"two-tone"}}`,
	}, "\n")
	ch := make(chan Event, 4)
	if err := parseSSE(strings.NewReader(body), ch); err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	close(ch)
	var got []Event
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 1 {
		t.Fatalf("trailing event lost: got %d", len(got))
	}
	var tone Tone
	if err := json.Unmarshal(got[0].Raw, &tone); err != nil {
		t.Fatalf("decode tone: %v", err)
	}
	if tone.Profile != "two-tone" {
		t.Errorf("tone.Profile = %q", tone.Profile)
	}
}

func TestParseSSE_MultiLineData(t *testing.T) {
	body := "event: x\ndata: line1\ndata: line2\n\n"
	ch := make(chan Event, 4)
	if err := parseSSE(bytes.NewReader([]byte(body)), ch); err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	close(ch)
	ev := <-ch
	// Multi-line data is joined with "\n"; the daemon doesn't emit
	// this shape so we just need to not crash and not drop the event.
	if !strings.Contains(string(ev.Raw), "line1") || !strings.Contains(string(ev.Raw), "line2") {
		t.Errorf("multi-line data lost: %s", ev.Raw)
	}
}
