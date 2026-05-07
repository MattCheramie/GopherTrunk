package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, h http.Handler) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := New(srv.URL, 2*time.Second, false)
	return c, srv.Close
}

func TestHealth_OK(t *testing.T) {
	cli, done := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","now":"2024-01-02T03:04:05Z"}`))
	}))
	defer done()
	h, err := cli.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("Status = %q", h.Status)
	}
}

func TestHealth_HTTPError(t *testing.T) {
	cli, done := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`unavailable`))
	}))
	defer done()
	_, err := cli.Health(context.Background())
	if err == nil {
		t.Fatal("want error")
	}
	var herr *HTTPError
	if !errors.As(err, &herr) {
		t.Fatalf("want *HTTPError, got %T", err)
	}
	if herr.Status != 503 {
		t.Errorf("Status = %d", herr.Status)
	}
}

func TestSystems_Talkgroups_Active(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/systems", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"systems":[{"name":"Demo","protocol":"P25","control_channels":[851012500]}]}`))
	})
	mux.HandleFunc("/api/v1/talkgroups", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"talkgroups":[{"id":42,"alpha_tag":"Dispatch"}]}`))
	})
	mux.HandleFunc("/api/v1/calls/active", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"calls":[]}`))
	})
	cli, done := newTestServer(t, mux)
	defer done()

	sys, err := cli.Systems(context.Background())
	if err != nil || len(sys) != 1 || sys[0].Name != "Demo" {
		t.Fatalf("Systems: %v %+v", err, sys)
	}
	tgs, err := cli.Talkgroups(context.Background())
	if err != nil || len(tgs) != 1 || tgs[0].AlphaTag != "Dispatch" {
		t.Fatalf("Talkgroups: %v %+v", err, tgs)
	}
	calls, err := cli.ActiveCalls(context.Background())
	if err != nil {
		t.Fatalf("ActiveCalls: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("calls = %d", len(calls))
	}
}

func TestHistory_QueryParams(t *testing.T) {
	var got string
	cli, done := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"calls":[]}`))
	}))
	defer done()
	since := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err := cli.History(context.Background(), HistoryFilter{
		System:  "Demo",
		GroupID: 42,
		Since:   since,
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	for _, want := range []string{"system=Demo", "group_id=42", "limit=50", "since=2024-01-02T03"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestMetrics_ParsePrometheusText(t *testing.T) {
	body := strings.Join([]string{
		`# HELP gophertrunk_calls_active Currently active calls.`,
		`# TYPE gophertrunk_calls_active gauge`,
		`gophertrunk_calls_active 3`,
		`gophertrunk_grants_total{system="Demo"} 17`,
		`gophertrunk_grants_total{system="Other"} 5`,
		`gophertrunk_cc_locked{repeater="r1"} 1`,
	}, "\n")
	cli, done := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer done()
	m, err := cli.Metrics(context.Background())
	if err != nil {
		t.Fatalf("Metrics: %v", err)
	}
	if m["gophertrunk_calls_active"] != 3 {
		t.Errorf("calls_active = %v", m["gophertrunk_calls_active"])
	}
	// labelled series should sum.
	if m["gophertrunk_grants_total"] != 22 {
		t.Errorf("grants_total = %v", m["gophertrunk_grants_total"])
	}
	if m["gophertrunk_cc_locked"] != 1 {
		t.Errorf("cc_locked = %v", m["gophertrunk_cc_locked"])
	}
}

func TestFormatFreqMHz(t *testing.T) {
	cases := map[uint32]string{
		0:           "—",
		851_012_500: "851.012500 MHz",
	}
	for in, want := range cases {
		got := FormatFreqMHz(in)
		if got != want {
			t.Errorf("FormatFreqMHz(%d) = %q, want %q", in, got, want)
		}
	}
}
